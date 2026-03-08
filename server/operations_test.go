package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupOpsTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_ops_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	_, _ = testDB.Exec("PRAGMA journal_mode=WAL")
	_, _ = testDB.Exec("PRAGMA busy_timeout=5000")
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	oldDB := db
	db = testDB
	cleanup := func() {
		db.Close()
		db = oldDB
		os.Remove(testDBPath)
	}
	return testDB, cleanup
}

func TestDetectStuckTasks_NoStuckTasks(t *testing.T) {
	testDB, cleanup := setupOpsTestDB(t)
	defer cleanup()

	stuck, err := DetectStuckTasks(testDB, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectStuckTasks: %v", err)
	}
	if len(stuck) != 0 {
		t.Errorf("expected 0 stuck tasks, got %d", len(stuck))
	}
}

func TestDetectStuckTasks_FindsStuckTask(t *testing.T) {
	testDB, cleanup := setupOpsTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "stuck-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, "stuck-task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim the task, then expire the lease and simulate time passing.
	env, err := ClaimTaskByID(testDB, task.ID, "agent-stuck", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	// Expire the lease by setting it in the past.
	pastTime := time.Now().UTC().Add(-2 * time.Hour).Format(leaseTimeFormat)
	_, err = testDB.Exec("UPDATE leases SET expires_at = ? WHERE id = ?", pastTime, env.Lease.ID)
	if err != nil {
		t.Fatalf("expire lease: %v", err)
	}

	// Set task updated_at to 1 hour ago (past default 30 min threshold).
	oldTime := time.Now().UTC().Add(-1 * time.Hour).Format(leaseTimeFormat)
	_, err = testDB.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldTime, task.ID)
	if err != nil {
		t.Fatalf("update timestamp: %v", err)
	}

	// Delete the expired lease row directly (don't use DeleteExpiredLeases which re-queues).
	_, _ = testDB.Exec("DELETE FROM leases WHERE id = ?", env.Lease.ID)

	stuck, err := DetectStuckTasks(testDB, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectStuckTasks: %v", err)
	}
	if len(stuck) == 0 {
		t.Fatal("expected at least 1 stuck task")
	}
	if stuck[0].TaskID != task.ID {
		t.Errorf("expected task %d, got %d", task.ID, stuck[0].TaskID)
	}
}

func TestRequeueStuckTasks(t *testing.T) {
	testDB, cleanup := setupOpsTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "requeue-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, "stuck-requeue-task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim, expire lease, set old timestamp.
	env, err := ClaimTaskByID(testDB, task.ID, "agent-die", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}
	pastTime := time.Now().UTC().Add(-2 * time.Hour).Format(leaseTimeFormat)
	_, _ = testDB.Exec("UPDATE leases SET expires_at = ? WHERE id = ?", pastTime, env.Lease.ID)
	oldTime := time.Now().UTC().Add(-1 * time.Hour).Format(leaseTimeFormat)
	_, _ = testDB.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldTime, task.ID)
	// Delete the lease row directly so task appears stuck (no active lease).
	_, _ = testDB.Exec("DELETE FROM leases WHERE id = ?", env.Lease.ID)

	count, err := RequeueStuckTasks(testDB, 30*time.Minute)
	if err != nil {
		t.Fatalf("RequeueStuckTasks: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 task requeued, got %d", count)
	}

	// Verify task is now QUEUED.
	updated, err := GetTaskV2(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if updated.StatusV2 != TaskStatusQueued {
		t.Errorf("expected QUEUED, got %s", updated.StatusV2)
	}
	if updated.StatusV1 != StatusNotPicked {
		t.Errorf("expected NOT_PICKED, got %s", updated.StatusV1)
	}

	// Verify task.stalled event emitted.
	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	found := false
	for _, ev := range events {
		if ev.Kind == "task.stalled" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected task.stalled event")
	}
}

func TestPruneOldEvents(t *testing.T) {
	testDB, cleanup := setupOpsTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "prune-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Create events with old timestamps.
	for i := 0; i < 5; i++ {
		_, err := CreateEvent(testDB, project.ID, "test.event", "test", fmt.Sprintf("e%d", i), map[string]interface{}{"i": i})
		if err != nil {
			t.Fatalf("CreateEvent: %v", err)
		}
	}

	// Backdate 3 events to 60 days ago.
	oldTime := time.Now().UTC().AddDate(0, 0, -60).Format(leaseTimeFormat)
	rows, err := testDB.Query("SELECT id FROM events ORDER BY created_at LIMIT 3")
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		_, _ = testDB.Exec("UPDATE events SET created_at = ? WHERE id = ?", oldTime, id)
	}

	// Prune with 30-day retention.
	deleted, err := PruneOldEvents(testDB, 30)
	if err != nil {
		t.Fatalf("PruneOldEvents: %v", err)
	}
	if deleted != 3 {
		t.Errorf("expected 3 events pruned, got %d", deleted)
	}

	// Verify remaining events.
	var remaining int
	testDB.QueryRow("SELECT COUNT(*) FROM events").Scan(&remaining)
	if remaining != 2 {
		t.Errorf("expected 2 remaining events, got %d", remaining)
	}
}

func TestDetectStuckTasks_ActiveLeaseNotStuck(t *testing.T) {
	testDB, cleanup := setupOpsTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "active-lease-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, "active-lease-task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim task — lease is active.
	_, err = ClaimTaskByID(testDB, task.ID, "agent-active", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	// Set updated_at to old (but lease is still active).
	oldTime := time.Now().UTC().Add(-1 * time.Hour).Format(leaseTimeFormat)
	_, _ = testDB.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldTime, task.ID)

	stuck, err := DetectStuckTasks(testDB, 30*time.Minute)
	if err != nil {
		t.Fatalf("DetectStuckTasks: %v", err)
	}

	// Should not be stuck since lease is still active.
	for _, st := range stuck {
		if st.TaskID == task.ID {
			t.Error("task with active lease should not be detected as stuck")
		}
	}
}
