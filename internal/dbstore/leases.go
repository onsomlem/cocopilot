package dbstore

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func CreateLeaseTx(tx *sql.Tx, taskID int, agentID string, mode string) (*models.Lease, error) {
	if mode == "" {
		mode = "exclusive"
	}

	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(models.LeaseTimeFormat)
	now := models.NowISO()

	lease := &models.Lease{
		ID:        "lease_" + uuid.New().String(),
		TaskID:    taskID,
		AgentID:   agentID,
		Mode:      mode,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	// Remove stale lease rows for this task so they do not block reclaims.
	if _, err := tx.Exec("DELETE FROM leases WHERE task_id = ? AND expires_at <= ?", taskID, now); err != nil {
		return nil, fmt.Errorf("failed to clean stale leases: %w", err)
	}

	_, err := tx.Exec(`
		INSERT INTO leases (id, task_id, agent_id, mode, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, lease.ID, lease.TaskID, lease.AgentID, lease.Mode, lease.CreatedAt, lease.ExpiresAt)

	if err != nil {
		if IsLeaseConflictError(err) {
			return nil, ErrLeaseConflict
		}
		return nil, fmt.Errorf("failed to create lease: %w", err)
	}

	// Emit lease.created event inside the same transaction.
	if err := emitLeaseLifecycleEventTx(tx, "lease.created", lease, nil); err != nil {
		return nil, err
	}

	return lease, nil
}

func CreateLease(db *sql.DB, taskID int, agentID string, mode string) (*models.Lease, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin lease tx: %w", err)
	}

	lease, err := CreateLeaseTx(tx, taskID, agentID, mode)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to commit lease tx: %w", err)
	}

	return lease, nil
}

func GetLeaseByTaskID(db *sql.DB, taskID int) (*models.Lease, error) {
	var lease models.Lease
	err := db.QueryRow(`
		SELECT id, task_id, agent_id, mode, created_at, expires_at
		FROM leases
		WHERE task_id = ? AND expires_at > ?
	`, taskID, models.NowISO()).Scan(&lease.ID, &lease.TaskID, &lease.AgentID, &lease.Mode, &lease.CreatedAt, &lease.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lease: %w", err)
	}
	return &lease, nil
}

func DeleteLease(db *sql.DB, leaseID string) error {
	_, err := db.Exec("DELETE FROM leases WHERE id = ?", leaseID)
	if err != nil {
		return fmt.Errorf("failed to delete lease: %w", err)
	}
	return nil
}

func ReleaseLease(db *sql.DB, leaseID string, reason string) (bool, *models.Lease, error) {
	lease, err := GetLeaseByID(db, leaseID)
	if err != nil {
		return false, nil, err
	}
	if lease == nil {
		return false, nil, nil
	}

	if err := DeleteLease(db, leaseID); err != nil {
		return false, nil, err
	}

	payload := map[string]interface{}{}
	if reason != "" {
		payload["reason"] = reason
	}
	if err := EmitLeaseLifecycleEvent(db, "lease.released", lease, payload); err != nil {
		return true, lease, err
	}

	return true, lease, nil
}

func DeleteExpiredLeases(db *sql.DB) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to start lease cleanup transaction: %w", err)
	}
	defer tx.Rollback()

	now := models.NowISO()

	rows, err := tx.Query(`
		SELECT id, task_id, agent_id, mode, created_at, expires_at
		FROM leases
		WHERE expires_at <= ?
	`, now)
	if err != nil {
		return 0, fmt.Errorf("failed to query expired leases: %w", err)
	}
	defer rows.Close()

	expiredLeases := make([]models.Lease, 0)
	for rows.Next() {
		var lease models.Lease
		if err := rows.Scan(&lease.ID, &lease.TaskID, &lease.AgentID, &lease.Mode, &lease.CreatedAt, &lease.ExpiresAt); err != nil {
			return 0, fmt.Errorf("failed to scan expired lease: %w", err)
		}
		expiredLeases = append(expiredLeases, lease)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed while iterating expired leases: %w", err)
	}

	// Requeue abandoned tasks so they can be claimed again by normal flow.
	_, err = tx.Exec(`
		UPDATE tasks
		SET status = ?
		WHERE status = ?
		  AND id IN (SELECT task_id FROM leases WHERE expires_at <= ?)
	`, models.StatusNotPicked, models.StatusInProgress, now)
	if err != nil {
		return 0, fmt.Errorf("failed to requeue tasks for expired leases: %w", err)
	}

	result, err := tx.Exec("DELETE FROM leases WHERE expires_at <= ?", now)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired leases: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read lease cleanup rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit lease cleanup transaction: %w", err)
	}

	for i := range expiredLeases {
		lease := expiredLeases[i]
		if err := EmitLeaseLifecycleEvent(db, "lease.expired", &lease, map[string]interface{}{"reason": "expired"}); err != nil {
			return count, err
		}
	}

	return count, nil
}

func GetLeaseByID(db *sql.DB, leaseID string) (*models.Lease, error) {
	lease := &models.Lease{}
	err := db.QueryRow(
		"SELECT id, task_id, agent_id, mode, created_at, expires_at FROM leases WHERE id = ?",
		leaseID,
	).Scan(&lease.ID, &lease.TaskID, &lease.AgentID, &lease.Mode, &lease.CreatedAt, &lease.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lease by ID: %w", err)
	}
	return lease, nil
}

func ExtendLease(db *sql.DB, leaseID string, duration time.Duration) error {
	var currentExpiresAt string
	err := db.QueryRow("SELECT expires_at FROM leases WHERE id = ?", leaseID).Scan(&currentExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("lease not found: %s", leaseID)
		}
		return fmt.Errorf("failed to read current lease expiration: %w", err)
	}

	parsedCurrent, err := time.Parse(time.RFC3339Nano, currentExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to parse current lease expiration: %w", err)
	}

	base := time.Now().UTC()
	if parsedCurrent.After(base) {
		base = parsedCurrent
	}

	newExpiresAt := base.Add(duration).Format(models.LeaseTimeFormat)
	_, err = db.Exec("UPDATE leases SET expires_at = ? WHERE id = ?", newExpiresAt, leaseID)
	if err != nil {
		return fmt.Errorf("failed to extend lease: %w", err)
	}
	return nil
}

func EmitLeaseLifecycleEvent(db *sql.DB, kind string, lease *models.Lease, extra map[string]interface{}) error {
	if lease == nil {
		return nil
	}

	projectID, err := resolveTaskProjectID(db, lease.TaskID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"lease_id":   lease.ID,
		"task_id":    lease.TaskID,
		"agent_id":   lease.AgentID,
		"mode":       lease.Mode,
		"created_at": lease.CreatedAt,
		"expires_at": lease.ExpiresAt,
	}
	for k, v := range extra {
		payload[k] = v
	}

	if _, err := CreateEvent(db, projectID, kind, "lease", lease.ID, payload); err != nil {
		return fmt.Errorf("failed to emit %s event: %w", kind, err)
	}
	return nil
}

func emitLeaseLifecycleEventTx(tx *sql.Tx, kind string, lease *models.Lease, extra map[string]interface{}) error {
	if lease == nil {
		return nil
	}

	// Resolve project ID from the task inside the same tx.
	var projectID sql.NullString
	err := tx.QueryRow("SELECT project_id FROM tasks WHERE id = ?", lease.TaskID).Scan(&projectID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to resolve project for task %d: %w", lease.TaskID, err)
	}
	resolvedProjectID := DefaultProjectID
	if projectID.Valid && strings.TrimSpace(projectID.String) != "" {
		resolvedProjectID = projectID.String
	}

	payload := map[string]interface{}{
		"lease_id":   lease.ID,
		"task_id":    lease.TaskID,
		"agent_id":   lease.AgentID,
		"mode":       lease.Mode,
		"created_at": lease.CreatedAt,
		"expires_at": lease.ExpiresAt,
	}
	for k, v := range extra {
		payload[k] = v
	}

	if _, err := CreateEventTx(tx, resolvedProjectID, kind, "lease", lease.ID, payload); err != nil {
		return fmt.Errorf("failed to emit %s event: %w", kind, err)
	}
	return nil
}
