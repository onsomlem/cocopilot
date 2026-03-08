package server

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// StuckTask describes a task detected as stuck (CLAIMED/RUNNING past threshold
// with no active lease or a mismatched/expired lease).
type StuckTask struct {
	TaskID    int          `json:"task_id"`
	ProjectID string      `json:"project_id"`
	StatusV2  TaskStatusV2 `json:"status_v2"`
	UpdatedAt string       `json:"updated_at"`
	LeaseID   *string      `json:"lease_id,omitempty"`
	Reason    string       `json:"reason"`
}

// DetectStuckTasks finds tasks in CLAIMED or RUNNING status that have been
// active longer than the given threshold with no valid lease. These tasks
// indicate an agent that died without releasing its lease or completing.
func DetectStuckTasks(database *sql.DB, threshold time.Duration) ([]StuckTask, error) {
	cutoff := time.Now().UTC().Add(-threshold).Format(leaseTimeFormat)
	now := nowISO()

	rows, err := database.Query(`
		SELECT t.id, t.project_id, t.status_v2, COALESCE(t.updated_at, t.created_at),
		       l.id
		FROM tasks t
		LEFT JOIN leases l ON t.id = l.task_id AND l.expires_at > ?
		WHERE t.status_v2 IN (?, ?)
		  AND COALESCE(t.updated_at, t.created_at) < ?
		  AND l.id IS NULL
	`, now, TaskStatusClaimed, TaskStatusRunning, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query stuck tasks: %w", err)
	}
	defer rows.Close()

	var stuck []StuckTask
	for rows.Next() {
		var st StuckTask
		var leaseID sql.NullString
		if err := rows.Scan(&st.TaskID, &st.ProjectID, &st.StatusV2, &st.UpdatedAt, &leaseID); err != nil {
			return nil, fmt.Errorf("scan stuck task: %w", err)
		}
		if leaseID.Valid {
			st.LeaseID = &leaseID.String
		}
		st.Reason = fmt.Sprintf("task in %s status since %s with no active lease", st.StatusV2, st.UpdatedAt)
		stuck = append(stuck, st)
	}

	return stuck, rows.Err()
}

// RequeueStuckTasks detects and re-queues stuck tasks past the threshold.
// Returns the number of tasks re-queued.
func RequeueStuckTasks(database *sql.DB, threshold time.Duration) (int, error) {
	stuck, err := DetectStuckTasks(database, threshold)
	if err != nil {
		return 0, err
	}

	requeued := 0
	now := nowISO()
	for _, st := range stuck {
		result, err := database.Exec(
			"UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ? AND status_v2 IN (?, ?)",
			StatusNotPicked, TaskStatusQueued, now, st.TaskID, TaskStatusClaimed, TaskStatusRunning,
		)
		if err != nil {
			log.Printf("Warning: failed to requeue stuck task %d: %v", st.TaskID, err)
			continue
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			requeued++
			if _, err := CreateEvent(database, st.ProjectID, "task.stalled", "task",
				fmt.Sprintf("%d", st.TaskID), map[string]interface{}{
					"task_id":   st.TaskID,
					"reason":    st.Reason,
					"action":    "requeued",
				}); err != nil {
				log.Printf("Warning: failed to emit task.stalled event for task %d: %v", st.TaskID, err)
			}
		}
	}

	return requeued, nil
}

// PruneOldEvents deletes events older than retentionDays.
// This is a thin wrapper around db_v2.PruneEvents for operational use.
func PruneOldEvents(database *sql.DB, retentionDays int) (int64, error) {
	return PruneEvents(database, retentionDays, 0)
}

// defaultStuckTaskThreshold is the default time threshold for detecting stuck tasks.
const defaultStuckTaskThreshold = 30 * time.Minute

// defaultEventRetentionDays is the default retention period for event pruning.
const defaultEventRetentionDays = 30

// startOperationalChecks starts periodic background tasks for stuck task
// detection and event pruning. Call this after the server starts.
func startOperationalChecks(database *sql.DB, stopCh <-chan struct{}) {
	// Stuck task detector: runs every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				count, err := RequeueStuckTasks(database, defaultStuckTaskThreshold)
				if err != nil {
					log.Printf("Operational: stuck task check error: %v", err)
				} else if count > 0 {
					log.Printf("Operational: requeued %d stuck tasks", count)
				}
			}
		}
	}()
}
