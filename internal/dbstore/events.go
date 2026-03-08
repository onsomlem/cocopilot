package dbstore

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func ParseEventTaskID(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func TaskIDFromPayload(payload map[string]interface{}) (int, bool) {
	if payload == nil {
		return 0, false
	}
	raw, ok := payload["task_id"]
	if !ok || raw == nil {
		return 0, false
	}
	switch value := raw.(type) {
	case int:
		if value > 0 {
			return value, true
		}
	case int64:
		if value > 0 {
			return int(value), true
		}
	case float64:
		if value > 0 {
			return int(value), true
		}
	case string:
		return ParseEventTaskID(value)
	}
	return 0, false
}

func resolveEventProjectID(db *sql.DB, projectID, entityType, entityID string, payload map[string]interface{}) (string, error) {
	trimmed := strings.TrimSpace(projectID)
	if trimmed != "" {
		return trimmed, nil
	}

	if db == nil {
		return DefaultProjectID, nil
	}

	var taskID int
	var ok bool

	switch entityType {
	case "task":
		taskID, ok = ParseEventTaskID(entityID)
	case "task_dependency":
		taskID, ok = TaskIDFromPayload(payload)
		if !ok {
			parts := strings.SplitN(entityID, ":", 2)
			if len(parts) > 0 {
				taskID, ok = ParseEventTaskID(parts[0])
			}
		}
	case "lease":
		taskID, ok = TaskIDFromPayload(payload)
	}

	if ok {
		return resolveTaskProjectID(db, taskID)
	}

	return DefaultProjectID, nil
}

func CreateEvent(db *sql.DB, projectID, kind, entityType, entityID string, payload map[string]interface{}) (*models.Event, error) {
	resolvedProjectID, err := resolveEventProjectID(db, projectID, entityType, entityID, payload)
	if err != nil {
		return nil, err
	}

	event := &models.Event{
		ID:         "evt_" + uuid.New().String(),
		ProjectID:  resolvedProjectID,
		Kind:       kind,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  models.NowISO(),
		Payload:    payload,
	}

	payloadJSON, err := models.MarshalJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO events (id, project_id, kind, entity_type, entity_id, created_at, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.ProjectID, event.Kind, event.EntityType, event.EntityID, event.CreatedAt, payloadJSON)

	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	if OnEventCreated != nil {
		OnEventCreated(db, *event)
	}
	return event, nil
}

func CreateEventTx(tx *sql.Tx, projectID, kind, entityType, entityID string, payload map[string]interface{}) (*models.Event, error) {
	resolvedProjectID := projectID
	if strings.TrimSpace(resolvedProjectID) == "" {
		resolvedProjectID = DefaultProjectID
	}

	event := &models.Event{
		ID:         "evt_" + uuid.New().String(),
		ProjectID:  resolvedProjectID,
		Kind:       kind,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  models.NowISO(),
		Payload:    payload,
	}

	payloadJSON, err := models.MarshalJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO events (id, project_id, kind, entity_type, entity_id, created_at, payload_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.ProjectID, event.Kind, event.EntityType, event.EntityID, event.CreatedAt, payloadJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	if OnEventCreatedTx != nil {
		OnEventCreatedTx(*event)
	}
	return event, nil
}

func GetEventsByProjectID(db *sql.DB, projectID string, limit int) ([]models.Event, error) {
	rows, err := db.Query(`
		SELECT id, project_id, kind, entity_type, entity_id, created_at, payload_json
		FROM events WHERE project_id = ? ORDER BY created_at DESC LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var payloadJSON sql.NullString
		err := rows.Scan(&event.ID, &event.ProjectID, &event.Kind, &event.EntityType,
			&event.EntityID, &event.CreatedAt, &payloadJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		if payloadJSON.Valid && payloadJSON.String != "" {
			if err := models.UnmarshalJSON(payloadJSON.String, &event.Payload); err != nil {
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}
		events = append(events, event)
	}
	return events, nil
}

func GetEventCreatedAtByID(db *sql.DB, eventID string) (string, error) {
	var createdAt string
	if err := db.QueryRow(`SELECT created_at FROM events WHERE id = ?`, eventID).Scan(&createdAt); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("event not found: %s", eventID)
		}
		return "", fmt.Errorf("failed to get event: %w", err)
	}
	return createdAt, nil
}

func GetEventReplayAnchor(db *sql.DB, eventID string) (string, string, error) {
	var projectID string
	var createdAt string
	if err := db.QueryRow(`SELECT project_id, created_at FROM events WHERE id = ?`, eventID).Scan(&projectID, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("event not found: %s", eventID)
		}
		return "", "", fmt.Errorf("failed to get event: %w", err)
	}
	return projectID, createdAt, nil
}

func ListEvents(db *sql.DB, projectID string, kind string, since string, taskID string, limit int, offset int) ([]models.Event, int, error) {
	baseQuery := "FROM events"
	conditions := make([]string, 0, 3)
	args := make([]interface{}, 0, 4)

	if strings.TrimSpace(projectID) != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}
	if kind != "" {
		conditions = append(conditions, "kind = ?")
		args = append(args, kind)
	}
	if since != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, since)
	}
	if taskID != "" {
		conditions = append(conditions, "entity_type = ?")
		args = append(args, "task")
		conditions = append(conditions, "entity_id = ?")
		args = append(args, taskID)
	}
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	query := "SELECT id, project_id, kind, entity_type, entity_id, created_at, payload_json " + baseQuery + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var payloadJSON sql.NullString
		err := rows.Scan(&event.ID, &event.ProjectID, &event.Kind, &event.EntityType,
			&event.EntityID, &event.CreatedAt, &payloadJSON)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}
		if payloadJSON.Valid && payloadJSON.String != "" {
			if err := models.UnmarshalJSON(payloadJSON.String, &event.Payload); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}
		events = append(events, event)
	}
	return events, total, nil
}

func PruneEvents(db *sql.DB, retentionDays int, maxRows int) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	if retentionDays <= 0 && maxRows <= 0 {
		return 0, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to start events cleanup transaction: %w", err)
	}
	defer tx.Rollback()

	var deleted int64
	if retentionDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).Format(models.LeaseTimeFormat)
		result, err := tx.Exec("DELETE FROM events WHERE created_at < ?", cutoff)
		if err != nil {
			return 0, fmt.Errorf("failed to delete old events: %w", err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("failed to read old events deleted count: %w", err)
		}
		deleted += count
	}

	if maxRows > 0 {
		result, err := tx.Exec(`
			DELETE FROM events
			WHERE id IN (
				SELECT id FROM events
				ORDER BY created_at DESC
				LIMIT -1 OFFSET ?
			)
		`, maxRows)
		if err != nil {
			return 0, fmt.Errorf("failed to delete excess events: %w", err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("failed to read excess events deleted count: %w", err)
		}
		deleted += count
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit events cleanup: %w", err)
	}

	return deleted, nil
}
