// Package notifications provides webhook notification delivery and
// stale-task / idle-project detection helpers.
package notifications

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/onsomlem/cocopilot/internal/dbstore"
	"github.com/onsomlem/cocopilot/internal/models"
)

// WebhookNotifier sends event notifications to configured webhook URLs.
type WebhookNotifier struct {
	mu      sync.RWMutex
	URLs    []string
	Client  *http.Client
	Enabled bool
}

// Notifier is the package-level webhook notifier instance.
var Notifier *WebhookNotifier

// NotifiableEvents lists the event kinds that trigger webhook notifications.
var NotifiableEvents = map[string]bool{
	"task.failed":   true,
	"task.stalled":  true,
	"policy.denied": true,
	"run.failed":    true,
	"lease.expired": true,
	"project.idle":  true,
}

// InitWebhookNotifier reads COCO_WEBHOOK_URL and initialises the global Notifier.
func InitWebhookNotifier() {
	raw := strings.TrimSpace(os.Getenv("COCO_WEBHOOK_URL"))
	if raw == "" {
		Notifier = &WebhookNotifier{Enabled: false}
		return
	}

	var urls []string
	for _, u := range strings.Split(raw, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			urls = append(urls, u)
		}
	}

	Notifier = &WebhookNotifier{
		URLs:    urls,
		Enabled: len(urls) > 0,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if Notifier.Enabled {
		log.Printf("Webhook notifications enabled for %d URL(s)", len(urls))
	}
}

// WebhookPayload is the JSON body sent to webhook endpoints.
type WebhookPayload struct {
	EventID    string                 `json:"event_id"`
	EventKind  string                 `json:"event_kind"`
	ProjectID  string                 `json:"project_id"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	CreatedAt  string                 `json:"created_at"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// NotifyEvent sends a webhook notification for the given event if enabled
// and the event kind is in the notifiable set.
func NotifyEvent(event models.Event) {
	if Notifier == nil || !Notifier.Enabled {
		return
	}
	if !NotifiableEvents[event.Kind] {
		return
	}

	payload := WebhookPayload{
		EventID:    event.ID,
		EventKind:  event.Kind,
		ProjectID:  event.ProjectID,
		EntityType: event.EntityType,
		EntityID:   event.EntityID,
		CreatedAt:  event.CreatedAt,
		Payload:    event.Payload,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Webhook: failed to marshal payload for event %s: %v", event.ID, err)
		return
	}

	Notifier.mu.RLock()
	urls := Notifier.URLs
	client := Notifier.Client
	Notifier.mu.RUnlock()

	for _, u := range urls {
		go func(target string) {
			req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
			if err != nil {
				log.Printf("Webhook: failed to create request for %s: %v", target, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Cocopilot-Event", event.Kind)

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Webhook: failed to send to %s for event %s: %v", target, event.ID, err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 400 {
				log.Printf("Webhook: %s returned status %d for event %s", target, resp.StatusCode, event.ID)
			}
		}(u)
	}
}

// DetectStalledTasks finds tasks that have been CLAIMED for longer than the
// given threshold and emits task.stalled events + webhook notifications.
func DetectStalledTasks(database *sql.DB, threshold time.Duration) {
	cutoff := time.Now().UTC().Add(-threshold).Format("2006-01-02T15:04:05.000000Z")

	rows, err := database.Query(
		"SELECT t.id, t.project_id, t.title, l.agent_id, l.created_at "+
			"FROM tasks t JOIN leases l ON l.task_id = t.id "+
			"WHERE t.status_v2 = ? AND l.created_at < ?",
		models.TaskStatusClaimed, cutoff,
	)
	if err != nil {
		log.Printf("stall detection: query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var taskID int
		var projectID string
		var title sql.NullString
		var agentID, claimedAt string
		if err := rows.Scan(&taskID, &projectID, &title, &agentID, &claimedAt); err != nil {
			continue
		}
		payload := map[string]interface{}{
			"task_id":    taskID,
			"agent_id":  agentID,
			"claimed_at": claimedAt,
			"reason":    "Task has been CLAIMED without progress beyond threshold",
		}
		if title.Valid {
			payload["title"] = title.String
		}
		evt, cErr := dbstore.CreateEvent(database, projectID, "task.stalled", "task", fmt.Sprintf("%d", taskID), payload)
		if cErr != nil {
			log.Printf("stall detection: failed to emit event for task %d: %v", taskID, cErr)
			continue
		}
		NotifyEvent(*evt)
	}
}

// DetectIdleProjects finds projects with no events in the given window and
// emits project.idle events.
func DetectIdleProjects(database *sql.DB, threshold time.Duration) {
	cutoff := time.Now().UTC().Add(-threshold).Format("2006-01-02T15:04:05.000000Z")

	rows, err := database.Query(
		"SELECT p.id, p.name FROM projects p "+
			"WHERE NOT EXISTS (SELECT 1 FROM events e WHERE e.project_id = p.id AND e.created_at > ?)",
		cutoff,
	)
	if err != nil {
		log.Printf("idle detection: query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var projectID, name string
		if err := rows.Scan(&projectID, &name); err != nil {
			continue
		}
		payload := map[string]interface{}{
			"project_id": projectID,
			"name":       name,
			"reason":     "No activity detected within threshold",
		}
		evt, cErr := dbstore.CreateEvent(database, projectID, "project.idle", "project", projectID, payload)
		if cErr != nil {
			log.Printf("idle detection: failed to emit event for project %s: %v", projectID, cErr)
			continue
		}
		NotifyEvent(*evt)
	}
}
