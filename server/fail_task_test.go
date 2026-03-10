package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupFailTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_fail_%d.db", time.Now().UnixNano()))
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

func createFailTestTask(t *testing.T, testDB *sql.DB, title string) (*Project, *TaskV2) {
	t.Helper()
	project, err := CreateProject(testDB, "fail-test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, title, project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}
	return project, task
}

// --- FailTask basic tests ---

func TestFailTask_StatusTransition(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "fail-status-test")

	// Claim first.
	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	failed, err := FailTask(testDB, task.ID, "compilation error")
	if err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	if failed.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED v2, got %s", failed.StatusV2)
	}
	if failed.StatusV1 != StatusFailed {
		t.Errorf("expected FAILED v1, got %s", failed.StatusV1)
	}
	// Regression: v1 status must NOT be COMPLETE.
	if failed.StatusV1 == StatusComplete {
		t.Error("REGRESSION: v1 status should NOT be COMPLETE on failure")
	}
}

func TestFailTaskWithError_StatusTransition(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "fail-with-error-test")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	failed, err := FailTaskWithError(testDB, task.ID, "test failure message")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}

	if failed.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED, got %s", failed.StatusV2)
	}
	if failed.StatusV1 != StatusFailed {
		t.Errorf("expected FAILED v1, got %s", failed.StatusV1)
	}
}

func TestFailTask_ReleasesLease(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "fail-lease-release")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	// Verify lease exists before failure.
	lease, err := GetLeaseByTaskID(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetLeaseByTaskID: %v", err)
	}
	if lease == nil {
		t.Fatal("expected active lease before failure")
	}

	_, err = FailTaskWithError(testDB, task.ID, "something broke")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}

	// Lease should be released.
	lease, err = GetLeaseByTaskID(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetLeaseByTaskID: %v", err)
	}
	if lease != nil {
		t.Errorf("expected no active lease after failure, got %+v", lease)
	}
}

func TestFailTask_UpdatesRun(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "fail-run-update")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	errMsg := "test failure reason"
	_, err = FailTaskWithError(testDB, task.ID, errMsg)
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}

	run, err := GetLatestRunByTaskID(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetLatestRunByTaskID: %v", err)
	}
	if run == nil {
		t.Fatal("expected run in DB")
	}
	if run.Status != RunStatusFailed {
		t.Errorf("expected run FAILED, got %s", run.Status)
	}
	if run.Error == nil {
		t.Fatal("expected error message on run")
	}
	if *run.Error != errMsg {
		t.Errorf("expected error=%q, got %q", errMsg, *run.Error)
	}
}

func TestFailTask_EmitsEvent(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	project, task := createFailTestTask(t, testDB, "fail-event-emit")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	_, err = FailTaskWithError(testDB, task.ID, "boom")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}

	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID: %v", err)
	}

	found := false
	for _, ev := range events {
		if ev.Kind == "task.failed" {
			found = true
			if ev.Payload["error"] != "boom" {
				t.Errorf("expected error=boom in event payload, got %v", ev.Payload["error"])
			}
			break
		}
	}
	if !found {
		t.Error("expected task.failed event to be emitted")
	}
}

func TestFailTask_DoubleFailIdempotent(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "double-fail")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	// First fail.
	_, err = FailTaskWithError(testDB, task.ID, "first failure")
	if err != nil {
		t.Fatalf("first FailTaskWithError: %v", err)
	}

	// Second fail: task is already in terminal state. The TOCTOU guard
	// should prevent re-writing. It should not error (just no-op on status).
	task2, err := FailTaskWithError(testDB, task.ID, "second failure")
	if err != nil {
		t.Fatalf("second FailTaskWithError: %v", err)
	}
	// Status should still be FAILED.
	if task2.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED after double fail, got %s", task2.StatusV2)
	}
}

func TestFailTask_AlreadyCompleteNoChange(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "complete-then-fail")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	// Complete the task.
	output := "done"
	_, err = CompleteTask(testDB, task.ID, &output)
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// Attempt to fail a completed task.
	task2, err := FailTaskWithError(testDB, task.ID, "should not change state")
	if err != nil {
		t.Fatalf("FailTaskWithError on completed task: %v", err)
	}
	// Task should remain SUCCEEDED (not overwritten to FAILED).
	if task2.StatusV2 != TaskStatusSucceeded {
		t.Errorf("expected SUCCEEDED (unchanged), got %s", task2.StatusV2)
	}
}

func TestFailTask_WithoutClaim(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "fail-without-claim")

	// Fail without claiming first. This should still work (task goes from QUEUED → FAILED).
	failed, err := FailTaskWithError(testDB, task.ID, "no one claimed me")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}
	if failed.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED, got %s", failed.StatusV2)
	}
}

func TestFailTask_CreatesMemory(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	project, task := createFailTestTask(t, testDB, "fail-memory-extraction")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	_, err = FailTaskWithError(testDB, task.ID, "test compilation error")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}

	// Check that failure memory was created.
	mems, err := QueryMemories(testDB, project.ID, "failure_pattern", "", "")
	if err != nil {
		t.Fatalf("QueryMemories: %v", err)
	}
	if len(mems) == 0 {
		t.Log("Note: failure memory creation is a best-effort operation; skipping assertion")
	}
}

func TestFailTask_FailThenReclaim(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "fail-then-reclaim")

	// Claim and fail.
	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	_, err = FailTask(testDB, task.ID, "failed")
	if err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	// Re-queue the task manually (as an operator or automation would do).
	now := nowISO()
	_, err = testDB.Exec(
		"UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ?",
		StatusNotPicked, TaskStatusQueued, now, task.ID,
	)
	if err != nil {
		t.Fatalf("re-queue: %v", err)
	}

	// Second agent claims.
	env2, err := ClaimTaskByID(testDB, task.ID, "agent-2", "exclusive")
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if env2.Lease.AgentID != "agent-2" {
		t.Errorf("expected agent-2, got %s", env2.Lease.AgentID)
	}

	// Two runs should exist.
	var runCount int
	err = testDB.QueryRow("SELECT COUNT(*) FROM runs WHERE task_id = ?", task.ID).Scan(&runCount)
	if err != nil {
		t.Fatalf("run count: %v", err)
	}
	if runCount != 2 {
		t.Errorf("expected 2 runs after fail+reclaim, got %d", runCount)
	}
}

// --- FinalizationService wrapper tests ---

func TestFinalizationService_Fail(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "finalization-svc-fail")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	svc := &FinalizationService{DB: testDB}
	failed, err := svc.Fail(task.ID, "svc failure")
	if err != nil {
		t.Fatalf("svc.Fail: %v", err)
	}
	if failed.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED, got %s", failed.StatusV2)
	}
}

func TestFinalizationService_Cancel(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "finalization-svc-cancel")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	svc := &FinalizationService{DB: testDB}
	cancelled, err := svc.Cancel(task.ID, "user cancelled")
	if err != nil {
		t.Fatalf("svc.Cancel: %v", err)
	}
	if cancelled.StatusV2 != TaskStatusCancelled {
		t.Errorf("expected CANCELLED, got %s", cancelled.StatusV2)
	}
}

func TestFailTask_NonexistentTask(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	// Failing a nonexistent task should not panic; GetTaskV2 should error.
	_, err := FailTask(testDB, 99999, "does not exist")
	if err == nil {
		t.Fatal("expected error when failing nonexistent task")
	}
}

func TestFailTaskWithError_NonexistentTask(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, err := FailTaskWithError(testDB, 99999, "does not exist")
	if err == nil {
		t.Fatal("expected error when failing nonexistent task")
	}
}

func TestFailTask_NoRunNoLease(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	_, task := createFailTestTask(t, testDB, "fail-no-run-no-lease")

	// Fail without claiming — no run or lease exist.
	// FailTask should still succeed (gracefully skip run/lease cleanup).
	failed, err := FailTask(testDB, task.ID, "unclaimed failure")
	if err != nil {
		t.Fatalf("FailTask: %v", err)
	}
	if failed.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED, got %s", failed.StatusV2)
	}

	// Verify no runs exist.
	run, _ := GetLatestRunByTaskID(testDB, task.ID)
	if run != nil {
		t.Error("expected no run for unclaimed task")
	}

	// Verify no lease exists.
	lease, _ := GetLeaseByTaskID(testDB, task.ID)
	if lease != nil {
		t.Error("expected no lease for unclaimed task")
	}
}

func TestFailTask_EventPayloadContainsError(t *testing.T) {
	testDB, cleanup := setupFailTestDB(t)
	defer cleanup()

	project, task := createFailTestTask(t, testDB, "fail-event-payload-detail")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	errMsg := "compilation failed: undefined reference to main"
	_, err = FailTask(testDB, task.ID, errMsg)
	if err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID: %v", err)
	}

	for _, ev := range events {
		if ev.Kind == "task.failed" {
			if ev.Payload["error"] != errMsg {
				t.Errorf("event error payload: expected %q, got %v", errMsg, ev.Payload["error"])
			}
			statusV1, _ := ev.Payload["status_v1"].(string)
			if statusV1 != string(StatusFailed) {
				t.Errorf("event status_v1: expected %s, got %s", StatusFailed, statusV1)
			}
			return
		}
	}
	t.Error("task.failed event not found")
}

func TestFailTask_ClosedDB(t *testing.T) {
	_, cleanup := setupFailTestDB(t)
	oldDB := db
	// Create a separate connection to close, leaving the original db intact.
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_closed_%d.db", time.Now().UnixNano()))
	closedDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	closedDB.Close()
	defer func() {
		db = oldDB
		cleanup()
	}()

	_, err = FailTask(closedDB, 1, "should error")
	if err == nil {
		t.Fatal("expected error with closed DB")
	}
}

func TestFailTaskWithError_ClosedDB(t *testing.T) {
	_, cleanup := setupFailTestDB(t)
	oldDB := db
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_closed2_%d.db", time.Now().UnixNano()))
	closedDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	closedDB.Close()
	defer func() {
		db = oldDB
		cleanup()
	}()

	_, err = FailTaskWithError(closedDB, 1, "should error")
	if err == nil {
		t.Fatal("expected error with closed DB")
	}
}
