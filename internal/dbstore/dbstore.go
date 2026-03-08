// Package dbstore provides all v2 database access layer operations.
package dbstore

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/onsomlem/cocopilot/internal/models"
)

// Sentinel errors.
var (
	ErrLeaseConflict        = errors.New("lease conflict")
	ErrTaskDependencyExists = errors.New("task dependency already exists")
	ErrTaskDependencyCycle  = errors.New("task dependency cycle")
	ErrInvalidPolicyRules   = errors.New("invalid policy rules")
)

// DefaultProjectID is the fallback project identifier.
const DefaultProjectID = "proj_default"

// EventHook is called after an event is inserted via CreateEvent.
type EventHook func(db *sql.DB, event models.Event)

// EventHookTx is called after an event is inserted via CreateEventTx.
type EventHookTx func(event models.Event)

// OnEventCreated is called after every CreateEvent DB insert.
var OnEventCreated EventHook

// OnEventCreatedTx is called after every CreateEventTx DB insert.
var OnEventCreatedTx EventHookTx

// ---- internal helpers ----

func IsLeaseConflictError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrLeaseConflict) {
		return true
	}
	return IsSQLiteConstraintError(err)
}

func IsSQLiteConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}

func IsSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy")
}

func ptrInt64ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

func resolveTaskProjectID(db *sql.DB, taskID int) (string, error) {
	var projectID sql.NullString
	err := db.QueryRow("SELECT project_id FROM tasks WHERE id = ?", taskID).Scan(&projectID)
	if err == sql.ErrNoRows {
		return DefaultProjectID, nil
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such column") {
			return DefaultProjectID, nil
		}
		return "", fmt.Errorf("failed to resolve project for task %d: %w", taskID, err)
	}
	if !projectID.Valid || strings.TrimSpace(projectID.String) == "" {
		return DefaultProjectID, nil
	}
	return projectID.String, nil
}
