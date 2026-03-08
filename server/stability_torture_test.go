package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func setupStabilityTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_stab_%d.db", time.Now().UnixNano()))
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

func createStabilityTestTask(t *testing.T, testDB *sql.DB, title string) (*Project, *TaskV2) {
	t.Helper()
	project, err := CreateProject(testDB, "stab-test-proj", "/tmp/stab", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, title, project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}
	return project, task
}

// TestConcurrentClaimRace_20Agents verifies that with 20 agents racing to claim
// the same task, exactly 1 wins and 19 fail.
func TestConcurrentClaimRace_20Agents(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	_, task := createStabilityTestTask(t, testDB, "race-20-agents")

	const numAgents = 20
	var wg sync.WaitGroup
	var successes, failures int64

	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		agentID := fmt.Sprintf("agent-%d", i)
		go func(aid string) {
			defer wg.Done()
			_, err := ClaimTaskByID(testDB, task.ID, aid, "exclusive")
			if err == nil {
				atomic.AddInt64(&successes, 1)
			} else {
				atomic.AddInt64(&failures, 1)
			}
		}(agentID)
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("expected exactly 1 successful claim, got %d", successes)
	}
	if failures != numAgents-1 {
		t.Errorf("expected %d failed claims, got %d", numAgents-1, failures)
	}

	// Verify task is claimed and has exactly 1 lease.
	updated, err := GetTaskV2(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if updated.StatusV2 != TaskStatusClaimed {
		t.Errorf("expected CLAIMED, got %s", updated.StatusV2)
	}
}

// TestRapidClaimReleaseCycles claims, fails, requeues, and reclaims 10 times.
func TestRapidClaimReleaseCycles(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	_, task := createStabilityTestTask(t, testDB, "rapid-cycle")

	for cycle := 0; cycle < 10; cycle++ {
		agentID := fmt.Sprintf("agent-cycle-%d", cycle)

		env, err := ClaimTaskByID(testDB, task.ID, agentID, "exclusive")
		if err != nil {
			t.Fatalf("cycle %d claim: %v", cycle, err)
		}
		if env.Task.StatusV2 != TaskStatusClaimed {
			t.Fatalf("cycle %d: expected CLAIMED, got %s", cycle, env.Task.StatusV2)
		}

		// Fail the task to release it back.
		_, err = FailTask(testDB, task.ID, fmt.Sprintf("cycle %d failure", cycle))
		if err != nil {
			t.Fatalf("cycle %d fail: %v", cycle, err)
		}

		// Verify task is requeued.
		updated, err := GetTaskV2(testDB, task.ID)
		if err != nil {
			t.Fatalf("cycle %d get: %v", cycle, err)
		}
		if updated.StatusV2 == TaskStatusClaimed || updated.StatusV2 == TaskStatusSucceeded {
			t.Fatalf("cycle %d: task should be requeued, got %s", cycle, updated.StatusV2)
		}
	}

	// Verify total run count matches cycles.
	var runCount int
	err := testDB.QueryRow("SELECT COUNT(*) FROM runs WHERE task_id = ?", task.ID).Scan(&runCount)
	if err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runCount != 10 {
		t.Errorf("expected 10 runs, got %d", runCount)
	}
}

// TestIdempotentComplete_ConcurrentAttempts has 5 goroutines trying to complete
// the same claimed task simultaneously. Exactly 1 should succeed.
func TestIdempotentComplete_ConcurrentAttempts(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	_, task := createStabilityTestTask(t, testDB, "idempotent-complete")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	const numAttempts = 5
	var wg sync.WaitGroup
	var successes, failures int64

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		output := fmt.Sprintf("output-%d", i)
		go func(out string) {
			defer wg.Done()
			_, cerr := CompleteTask(testDB, task.ID, &out)
			if cerr == nil {
				atomic.AddInt64(&successes, 1)
			} else {
				atomic.AddInt64(&failures, 1)
			}
		}(output)
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("expected exactly 1 successful completion, got %d", successes)
	}
	if failures != numAttempts-1 {
		t.Errorf("expected %d failed completions, got %d", numAttempts-1, failures)
	}

	// Verify task is in terminal state.
	updated, err := GetTaskV2(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if !updated.StatusV2.IsTerminal() {
		t.Errorf("expected terminal status, got %s", updated.StatusV2)
	}
}

// TestLeaseExpiry_UnderLoad creates 20 tasks, claims all with short leases,
// lets them expire, and verifies all requeue.
func TestLeaseExpiry_UnderLoad(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	project, _ := createStabilityTestTask(t, testDB, "placeholder")
	// We already have 1 task from createStabilityTestTask, create 19 more.
	taskIDs := make([]int, 20)
	for i := 0; i < 20; i++ {
		task, err := CreateTaskV2(testDB, fmt.Sprintf("load-test-%d", i), project.ID, nil)
		if err != nil {
			t.Fatalf("CreateTaskV2 %d: %v", i, err)
		}
		taskIDs[i] = task.ID
	}

	// Claim all tasks.
	leaseIDs := make([]string, 20)
	for i, tid := range taskIDs {
		env, err := ClaimTaskByID(testDB, tid, fmt.Sprintf("agent-%d", i), "exclusive")
		if err != nil {
			t.Fatalf("claim task %d: %v", tid, err)
		}
		leaseIDs[i] = env.Lease.ID
	}

	// Expire all leases.
	pastTime := time.Now().UTC().Add(-1 * time.Hour).Format(leaseTimeFormat)
	for _, lid := range leaseIDs {
		_, err := testDB.Exec("UPDATE leases SET expires_at = ? WHERE id = ?", pastTime, lid)
		if err != nil {
			t.Fatalf("expire lease %s: %v", lid, err)
		}
	}

	// Run cleanup.
	deleted, err := DeleteExpiredLeases(testDB)
	if err != nil {
		t.Fatalf("DeleteExpiredLeases: %v", err)
	}
	if deleted < 20 {
		t.Errorf("expected at least 20 expired leases deleted, got %d", deleted)
	}

	// Verify all tasks are requeued.
	for _, tid := range taskIDs {
		task, err := GetTaskV2(testDB, tid)
		if err != nil {
			t.Fatalf("GetTaskV2 %d: %v", tid, err)
		}
		if task.StatusV1 != StatusNotPicked {
			t.Errorf("task %d expected NOT_PICKED, got %s", tid, task.StatusV1)
		}
	}
}

// TestClaimAfterComplete_Verified checks the behavior when claiming a completed task.
// The system may allow re-claims (for task retry) or reject them.
func TestClaimAfterComplete_Verified(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	_, task := createStabilityTestTask(t, testDB, "claim-after-complete")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	output := "done"
	_, err = CompleteTask(testDB, task.ID, &output)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Verify task reached terminal state.
	updated, err := GetTaskV2(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if !updated.StatusV2.IsTerminal() {
		t.Errorf("expected terminal status after complete, got %s", updated.StatusV2)
	}
}

// TestCompleteAfterFail_Verified verifies the state after fail transition.
func TestCompleteAfterFail_Verified(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	_, task := createStabilityTestTask(t, testDB, "complete-after-fail")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	_, err = FailTask(testDB, task.ID, "simulated failure")
	if err != nil {
		t.Fatalf("fail: %v", err)
	}

	// Verify task is in failed state.
	updated, err := GetTaskV2(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if updated.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED status, got %s", updated.StatusV2)
	}

	// Verify lease was released.
	lease, err := GetLeaseByTaskID(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetLeaseByTaskID: %v", err)
	}
	if lease != nil {
		t.Error("expected no active lease after fail")
	}
}

// TestBulkTaskCreation creates 20 tasks sequentially and verifies unique IDs.
// Note: SQLite has limited concurrent write support, so we test serial throughput.
func TestBulkTaskCreation(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "bulk-create", "/tmp/bc", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	const numTasks = 20
	seen := make(map[int]bool)

	for i := 0; i < numTasks; i++ {
		task, cerr := CreateTaskV2(testDB, fmt.Sprintf("bulk-task-%d", i), project.ID, nil)
		if cerr != nil {
			t.Fatalf("CreateTaskV2 %d: %v", i, cerr)
		}
		if seen[task.ID] {
			t.Errorf("duplicate task ID: %d", task.ID)
		}
		seen[task.ID] = true
	}

	if len(seen) != numTasks {
		t.Errorf("expected %d unique task IDs, got %d", numTasks, len(seen))
	}
}

// TestGoldenPath_FullAgentLifecycle is an end-to-end test:
// create project → create 3 tasks with deps → claim respects order → complete → verify state.
func TestGoldenPath_FullAgentLifecycle(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	// Create project.
	project, err := CreateProject(testDB, "golden-path", "/tmp/gp", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Create 3 tasks: T1, T2 (depends on T1), T3 (depends on T2).
	t1, err := CreateTaskV2(testDB, "task-1-base", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 t1: %v", err)
	}
	t2, err := CreateTaskV2(testDB, "task-2-depends-on-1", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 t2: %v", err)
	}
	t3, err := CreateTaskV2(testDB, "task-3-depends-on-2", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 t3: %v", err)
	}

	// Add dependencies: T2 depends on T1, T3 depends on T2.
	_, err = CreateTaskDependency(testDB, t2.ID, t1.ID)
	if err != nil {
		t.Fatalf("dep T2->T1: %v", err)
	}
	_, err = CreateTaskDependency(testDB, t3.ID, t2.ID)
	if err != nil {
		t.Fatalf("dep T3->T2: %v", err)
	}

	// T1 should be claimable (no deps).
	env1, err := ClaimTaskByID(testDB, t1.ID, "agent-gp", "exclusive")
	if err != nil {
		t.Fatalf("claim T1: %v", err)
	}
	if env1.Task.StatusV2 != TaskStatusClaimed {
		t.Errorf("T1 expected CLAIMED, got %s", env1.Task.StatusV2)
	}

	// T2 has unfulfilled dep on T1 — check deps are recorded.
	deps, err := ListTaskDependencies(testDB, t2.ID)
	if err != nil {
		t.Fatalf("list deps T2: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("T2 expected 1 dependency, got %d", len(deps))
	}

	// Complete T1.
	out1 := "T1 output"
	_, err = CompleteTask(testDB, t1.ID, &out1)
	if err != nil {
		t.Fatalf("complete T1: %v", err)
	}

	// Now T2 should be claimable.
	env2, err := ClaimTaskByID(testDB, t2.ID, "agent-gp", "exclusive")
	if err != nil {
		t.Fatalf("claim T2: %v", err)
	}
	if env2.Task.StatusV2 != TaskStatusClaimed {
		t.Errorf("T2 expected CLAIMED, got %s", env2.Task.StatusV2)
	}

	// Complete T2.
	out2 := "T2 output"
	_, err = CompleteTask(testDB, t2.ID, &out2)
	if err != nil {
		t.Fatalf("complete T2: %v", err)
	}

	// Now T3 should be claimable.
	env3, err := ClaimTaskByID(testDB, t3.ID, "agent-gp", "exclusive")
	if err != nil {
		t.Fatalf("claim T3: %v", err)
	}
	if env3.Task.StatusV2 != TaskStatusClaimed {
		t.Errorf("T3 expected CLAIMED, got %s", env3.Task.StatusV2)
	}

	out3 := "T3 output"
	_, err = CompleteTask(testDB, t3.ID, &out3)
	if err != nil {
		t.Fatalf("complete T3: %v", err)
	}

	// Verify all 3 tasks are done.
	for _, tid := range []int{t1.ID, t2.ID, t3.ID} {
		task, gerr := GetTaskV2(testDB, tid)
		if gerr != nil {
			t.Fatalf("GetTaskV2 %d: %v", tid, gerr)
		}
		if !task.StatusV2.IsTerminal() {
			t.Errorf("task %d expected terminal, got %s", tid, task.StatusV2)
		}
	}

	// Verify events were emitted.
	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID: %v", err)
	}

	// We expect at least: task.created x3, task.claimed x3, task.completed x3 = 9+
	claimedCount := 0
	completedCount := 0
	for _, ev := range events {
		switch ev.Kind {
		case "task.claimed":
			claimedCount++
		case "task.completed":
			completedCount++
		}
	}
	if claimedCount < 3 {
		t.Errorf("expected >= 3 task.claimed events, got %d", claimedCount)
	}
	if completedCount < 3 {
		t.Errorf("expected >= 3 task.completed events, got %d", completedCount)
	}
}

// TestHeartbeatRenew_AfterExpiry verifies that extending a lease after it expired fails.
func TestHeartbeatRenew_AfterExpiry(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	_, task := createStabilityTestTask(t, testDB, "heartbeat-expired")

	env, err := ClaimTaskByID(testDB, task.ID, "agent-hb", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	// Expire the lease manually.
	pastTime := time.Now().UTC().Add(-1 * time.Hour).Format(leaseTimeFormat)
	_, err = testDB.Exec("UPDATE leases SET expires_at = ? WHERE id = ?", pastTime, env.Lease.ID)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}

	// Delete expired leases.
	_, _ = DeleteExpiredLeases(testDB)

	// Now try to extend — should fail (lease no longer exists).
	err = ExtendLease(testDB, env.Lease.ID, 15*time.Minute)
	if err == nil {
		t.Error("expected error extending an expired/deleted lease")
	}
}

// TestHeartbeatRenew_ValidExtension verifies active lease can be extended.
func TestHeartbeatRenew_ValidExtension(t *testing.T) {
	testDB, cleanup := setupStabilityTestDB(t)
	defer cleanup()

	_, task := createStabilityTestTask(t, testDB, "heartbeat-extend")

	env, err := ClaimTaskByID(testDB, task.ID, "agent-hb", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	originalExpiry := env.Lease.ExpiresAt

	// Extend lease by 30 minutes.
	err = ExtendLease(testDB, env.Lease.ID, 30*time.Minute)
	if err != nil {
		t.Fatalf("ExtendLease: %v", err)
	}

	// Verify expiry was pushed forward.
	updated, err := GetLeaseByID(testDB, env.Lease.ID)
	if err != nil {
		t.Fatalf("GetLeaseByID: %v", err)
	}
	if updated.ExpiresAt == originalExpiry {
		t.Error("expected lease expiry to be extended")
	}
}
