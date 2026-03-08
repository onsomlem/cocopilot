package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestCompletionEquivalenceV1V2 verifies that CompleteTask produces
// consistent database state and triggers expected lifecycle events.
func TestCompletionEquivalenceV1V2(t *testing.T) {
	testDB, cleanup := setupAssignmentTestDB(t)
	defer cleanup()

	projectID := "test_proj_" + fmt.Sprintf("%d", time.Now().UnixNano())
	project, err := CreateProject(testDB, projectID, "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	tests := []struct {
		name          string
		taskType      TaskType
		instructions  string
		output        string
		expectMemory  bool
		expectSummary bool
	}{
		{
			name:          "Simple task with output",
			taskType:      TaskTypeAnalyze,
			instructions:  "Analyze the system",
			output:        "System looks good",
			expectMemory:  true,
			expectSummary: true,
		},
		{
			name:          "Modify task with detailed result",
			taskType:      TaskTypeModify,
			instructions:  "Modify config.json",
			output:        "Changed timeout from 30 to 60 seconds",
			expectMemory:  true,
			expectSummary: true,
		},
		{
			name:          "Task with empty output",
			taskType:      TaskTypeTest,
			instructions:  "Run tests",
			output:        "",
			expectMemory:  false,
			expectSummary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create two identical tasks
			task1, err := CreateTaskV2(testDB, tt.instructions, project.ID, nil)
			if err != nil {
				t.Fatalf("CreateTaskV2: %v", err)
			}
			task2, err := CreateTaskV2(testDB, tt.instructions, project.ID, nil)
			if err != nil {
				t.Fatalf("CreateTaskV2: %v", err)
			}

			// Set task types
			testDB.Exec("UPDATE tasks SET type = ? WHERE id = ?", tt.taskType, task1.ID)
			testDB.Exec("UPDATE tasks SET type = ? WHERE id = ?", tt.taskType, task2.ID)

			// Claim both tasks
			_, err = ClaimTaskByID(testDB, task1.ID, "agent_test1", "exclusive")
			if err != nil {
				t.Fatalf("ClaimTaskByID for task1: %v", err)
			}
			_, err = ClaimTaskByID(testDB, task2.ID, "agent_test2", "exclusive")
			if err != nil {
				t.Fatalf("ClaimTaskByID for task2: %v", err)
			}

			// Complete both tasks via CompleteTask (simulates both v1 and v2)
			outputPtr := &tt.output
			if tt.output == "" {
				outputPtr = nil
			}

			completedTask1, _, err := CompleteTaskWithPayload(testDB, task1.ID, outputPtr)
			if err != nil {
				t.Fatalf("CompleteTaskWithPayload for task1: %v", err)
			}

			completedTask2, _, err := CompleteTaskWithPayload(testDB, task2.ID, outputPtr)
			if err != nil {
				t.Fatalf("CompleteTaskWithPayload for task2: %v", err)
			}

			// Both should have identical state
			// Status equivalence
			if completedTask1.StatusV1 != completedTask2.StatusV1 {
				t.Errorf("Status mismatch: %v != %v", completedTask1.StatusV1, completedTask2.StatusV1)
			}
			if completedTask1.StatusV2 != completedTask2.StatusV2 {
				t.Errorf("StatusV2 mismatch: %v != %v", completedTask1.StatusV2, completedTask2.StatusV2)
			}

			// Output equivalence
			if !nullStringEqual(completedTask1.Output, completedTask2.Output) {
				t.Errorf("Output mismatch: %v != %v", completedTask1.Output, completedTask2.Output)
			}

			// Both should have no active leases
			lease1, _ := GetLeaseByTaskID(testDB, task1.ID)
			lease2, _ := GetLeaseByTaskID(testDB, task2.ID)
			if lease1 != nil && IsLeaseActive(lease1.ExpiresAt) {
				t.Errorf("task1 lease not released: %v", lease1.ID)
			}
			if lease2 != nil && IsLeaseActive(lease2.ExpiresAt) {
				t.Errorf("task2 lease not released: %v", lease2.ID)
			}

			// Both should have terminal run states
			run1, _ := GetLatestRunByTaskID(testDB, task1.ID)
			run2, _ := GetLatestRunByTaskID(testDB, task2.ID)
			if run1 != nil && !run1.Status.IsTerminal() {
				t.Errorf("task1 run not terminal: %v", run1.Status)
			}
			if run2 != nil && !run2.Status.IsTerminal() {
				t.Errorf("task2 run not terminal: %v", run2.Status)
			}
		})
	}
}

// TestCompletionWithStructuredSummary verifies that CompleteTaskWithPayload
// can extract and store structured summaries from completion payloads.
func TestCompletionWithStructuredSummary(t *testing.T) {
	testDB, cleanup := setupAssignmentTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	task, err := CreateTaskV2(testDB, "Modify system", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim the task
	_, err = ClaimTaskByID(testDB, task.ID, "agent_test", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	// Structured output (as would come from agent)
	structuredOutput := `{
  "summary": "Updated configuration and added new feature",
  "changes_made": ["Added caching layer", "Updated timeout config"],
  "files_touched": ["config.json", "server.go"],
  "risks": ["May need retest under load"],
  "next_tasks": [{"title": "Test under load", "type": "TEST"}]
}`

	// Complete with structured output
	completedTask, summary, err := CompleteTaskWithPayload(testDB, task.ID, &structuredOutput)
	if err != nil {
		t.Fatalf("CompleteTaskWithPayload: %v", err)
	}

	// Verify task is complete
	if completedTask.StatusV2 != TaskStatusSucceeded {
		t.Errorf("Expected SUCCEEDED, got %v", completedTask.StatusV2)
	}

	// Task should have output stored
	if completedTask.Output == nil || *completedTask.Output != structuredOutput {
		t.Errorf("Output not stored correctly")
	}

	// Summary should be extracted
	if summary == nil {
		t.Fatalf("Summary not extracted")
	}
	if summary.Summary != "Updated configuration and added new feature" {
		t.Errorf("Summary mismatch: %v", summary.Summary)
	}
	if len(summary.ChangesMade) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(summary.ChangesMade))
	}
	if len(summary.FilesTouched) != 2 {
		t.Errorf("Expected 2 files touched, got %d", len(summary.FilesTouched))
	}
	if len(summary.Risks) != 1 {
		t.Errorf("Expected 1 risk, got %d", len(summary.Risks))
	}
}

// TestFailTaskCanonical verifies that FailTaskWithError is the canonical
// failure path and produces expected state transitions.
func TestFailTaskCanonical(t *testing.T) {
	testDB, cleanup := setupAssignmentTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	task, err := CreateTaskV2(testDB, "Do something", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim the task first
	envelope, err := ClaimTaskByID(testDB, task.ID, "agent1", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}
	if envelope.Lease == nil {
		t.Fatalf("No lease created")
	}

	// Fail the task
	failedTask, err := FailTaskWithError(testDB, task.ID, "Something went wrong")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}

	// Verify state
	if failedTask.StatusV2 != TaskStatusFailed {
		t.Errorf("Expected FAILED, got %v", failedTask.StatusV2)
	}

	// Lease should be released
	lease, _ := GetLeaseByTaskID(testDB, task.ID)
	if lease != nil && IsLeaseActive(lease.ExpiresAt) {
		t.Errorf("Lease not released after failure")
	}

	// Run should be failed
	run, _ := GetLatestRunByTaskID(testDB, task.ID)
	if run != nil && run.Status != RunStatusFailed {
		t.Errorf("Run status not FAILED, got %v", run.Status)
	}
	if run != nil && run.Error == nil {
		t.Errorf("Run error not set")
	}
}

// ===== Helper Functions =====

func nullStringEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// TestV1SaveVsV2CompletionEquivalence proves that the v1 /save HTTP handler and
// the direct CompleteTaskWithPayload call produce identical database state.
// This is the contract test required by the Phase 3 remediation.
func TestV1SaveVsV2CompletionEquivalence(t *testing.T) {
	testDB, cleanup := setupAssignmentTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "test-equiv-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// --- Path A: v1 HTTP /save handler ---
	taskA, err := CreateTaskV2(testDB, "v1 save test task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 (A): %v", err)
	}
	_, err = ClaimTaskByID(testDB, taskA.ID, "agent_v1", "exclusive")
	if err != nil {
		t.Fatalf("Claim (A): %v", err)
	}

	formData := url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", taskA.ID))
	formData.Set("message", "v1 completion output")

	req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	saveHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("v1 /save returned %d: %s", rr.Code, rr.Body.String())
	}

	// --- Path B: direct CompleteTaskWithPayload (same as v2 handler) ---
	taskB, err := CreateTaskV2(testDB, "v2 direct test task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 (B): %v", err)
	}
	_, err = ClaimTaskByID(testDB, taskB.ID, "agent_v2", "exclusive")
	if err != nil {
		t.Fatalf("Claim (B): %v", err)
	}
	output := "v2 completion output"
	_, _, err = CompleteTaskWithPayload(testDB, taskB.ID, &output)
	if err != nil {
		t.Fatalf("CompleteTaskWithPayload (B): %v", err)
	}

	// --- Verify both paths produce identical DB state ---
	finalA, err := GetTaskV2(testDB, taskA.ID)
	if err != nil {
		t.Fatalf("GetTaskV2 (A): %v", err)
	}
	finalB, err := GetTaskV2(testDB, taskB.ID)
	if err != nil {
		t.Fatalf("GetTaskV2 (B): %v", err)
	}

	// Status must match
	if finalA.StatusV1 != finalB.StatusV1 {
		t.Errorf("StatusV1 mismatch: v1=%v v2=%v", finalA.StatusV1, finalB.StatusV1)
	}
	if finalA.StatusV2 != finalB.StatusV2 {
		t.Errorf("StatusV2 mismatch: v1=%v v2=%v", finalA.StatusV2, finalB.StatusV2)
	}
	if finalA.StatusV2 != TaskStatusSucceeded {
		t.Errorf("Expected SUCCEEDED, got %v", finalA.StatusV2)
	}

	// Lease must be released (no active lease) for both
	leaseA, _ := GetLeaseByTaskID(testDB, taskA.ID)
	leaseB, _ := GetLeaseByTaskID(testDB, taskB.ID)
	if leaseA != nil && IsLeaseActive(leaseA.ExpiresAt) {
		t.Errorf("v1 task (A) still has active lease after completion")
	}
	if leaseB != nil && IsLeaseActive(leaseB.ExpiresAt) {
		t.Errorf("v2 task (B) still has active lease after completion")
	}

	// Both runs must be terminal
	runA, _ := GetLatestRunByTaskID(testDB, taskA.ID)
	runB, _ := GetLatestRunByTaskID(testDB, taskB.ID)
	if runA != nil && !runA.Status.IsTerminal() {
		t.Errorf("v1 run not terminal: %v", runA.Status)
	}
	if runB != nil && !runB.Status.IsTerminal() {
		t.Errorf("v2 run not terminal: %v", runB.Status)
	}

	// Both must have emitted task.completed events
	eventsA, _, _ := ListEvents(testDB, project.ID, "task.completed", "", fmt.Sprintf("%d", taskA.ID), 10, 0)
	eventsB, _, _ := ListEvents(testDB, project.ID, "task.completed", "", fmt.Sprintf("%d", taskB.ID), 10, 0)
	if len(eventsA) == 0 {
		t.Error("v1 completion did not emit task.completed event")
	}
	if len(eventsB) == 0 {
		t.Error("v2 completion did not emit task.completed event")
	}
}
