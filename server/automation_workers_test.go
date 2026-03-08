package server

import (
	"fmt"
	"testing"
	"time"
)

// TestProcessRunFailedEvent verifies that processRunFailedEvent creates a
// follow-up analysis task when a run.failed event is received.
func TestProcessRunFailedEvent(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "run-failed-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Create and claim a task so there's a parent task to reference.
	task, err := CreateTaskV2(testDB, "original task that will fail", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}
	_, err = ClaimTaskByID(testDB, task.ID, "agent_test", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	// Simulate a run.failed event.
	event := Event{
		ID:         "evt_run_failed_1",
		ProjectID:  project.ID,
		Kind:       "run.failed",
		EntityType: "task",
		EntityID:   fmt.Sprintf("%d", task.ID),
		CreatedAt:  nowISO(),
		Payload: map[string]interface{}{
			"task_id": float64(task.ID),
			"error":   "execution timed out after 300s",
		},
	}

	// Process the event.
	processRunFailedEvent(testDB, event)

	// Give async broadcast goroutines a moment.
	time.Sleep(10 * time.Millisecond)

	// A follow-up analysis task should have been created.
	tasks, _, err := ListTasksV2(testDB, project.ID, "", "", "", "", "", 10, 0, "created_at", "desc")
	if err != nil {
		t.Fatalf("listTasksV2: %v", err)
	}

	var found bool
	for _, tsk := range tasks {
		if tsk.ParentTaskID != nil && *tsk.ParentTaskID == task.ID {
			found = true
			// Verify it's tagged failure-analysis.
			hasTag := false
			for _, tag := range tsk.Tags {
				if tag == "failure-analysis" {
					hasTag = true
					break
				}
			}
			if !hasTag {
				t.Errorf("follow-up task missing 'failure-analysis' tag: %v", tsk.Tags)
			}
			break
		}
	}
	if !found {
		t.Error("processRunFailedEvent: expected follow-up analysis task to be created")
	}
}

// TestProcessRepoChangedEvent verifies that processRepoChangedEvent emits a
// context.invalidated event for the project.
func TestProcessRepoChangedEvent(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "repo-changed-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	event := Event{
		ID:         "evt_repo_changed_1",
		ProjectID:  project.ID,
		Kind:       "repo.changed",
		EntityType: "project",
		EntityID:   project.ID,
		CreatedAt:  nowISO(),
		Payload: map[string]interface{}{
			"changed_files": []string{"main.go", "README.md"},
		},
	}

	processRepoChangedEvent(testDB, event)

	// The worker emits context.invalidated via CreateEvent
	// Check that a context.invalidated event was recorded.
	// Pipeline: repo.changed -> repo.scanned -> context.invalidated
	// processRepoChangedEvent now emits repo.scanned (not context.invalidated directly)
	events, _, err := ListEvents(testDB, project.ID, "repo.scanned", "", "", 10, 0)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) == 0 {
		t.Error("processRepoChangedEvent: expected repo.scanned event to be emitted")
	}
	if events[0].Payload["trigger"] != "repo.changed" {
		t.Errorf("expected trigger=repo.changed, got %v", events[0].Payload["trigger"])
	}
}

// TestProcessContextInvalidatedEvent verifies that processContextInvalidatedEvent
// creates a context-refresh task for the project.
func TestProcessContextInvalidatedEvent(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "ctx-invalid-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	event := Event{
		ID:         "evt_ctx_invalid_1",
		ProjectID:  project.ID,
		Kind:       "context.invalidated",
		EntityType: "project",
		EntityID:   project.ID,
		CreatedAt:  nowISO(),
		Payload: map[string]interface{}{
			"trigger": "repo.changed",
		},
	}

	processContextInvalidatedEvent(testDB, event)

	time.Sleep(10 * time.Millisecond)

	// A context-refresh task should have been created.
	tasks, _, err := ListTasksV2(testDB, project.ID, "", "", "", "", "", 10, 0, "created_at", "desc")
	if err != nil {
		t.Fatalf("ListTasksV2: %v", err)
	}

	var found bool
	for _, tsk := range tasks {
		for _, tag := range tsk.Tags {
			if tag == "context-refresh" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("processContextInvalidatedEvent: expected context-refresh task to be created")
	}
}

// TestRunFailedDedup verifies that processRunFailedEvent is idempotent within
// the emission window — second call for same task should be a no-op.
func TestRunFailedDedup(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "run-failed-dedup-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	task, err := CreateTaskV2(testDB, "dedup test task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	event := Event{
		ID:        "evt_dedup_1",
		ProjectID: project.ID,
		Kind:      "run.failed",
		EntityID:  fmt.Sprintf("%d", task.ID),
		CreatedAt: nowISO(),
		Payload: map[string]interface{}{
			"task_id": float64(task.ID),
			"error":   "test error",
		},
	}

	// Call twice — second call should be dedup'd.
	processRunFailedEvent(testDB, event)
	processRunFailedEvent(testDB, event)

	time.Sleep(10 * time.Millisecond)

	tasks, _, err := ListTasksV2(testDB, project.ID, "", "", "", "", "", 10, 0, "created_at", "desc")
	if err != nil {
		t.Fatalf("ListTasksV2: %v", err)
	}

	count := 0
	for _, tsk := range tasks {
		if tsk.ParentTaskID != nil && *tsk.ParentTaskID == task.ID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 follow-up task (dedup), got %d", count)
	}
}

// TestNewTriggerTypes verifies that automation rules can be configured with
// the new supported trigger types (run.failed, repo.changed, context.invalidated).
func TestNewTriggerTypes(t *testing.T) {
	newTriggers := []string{"run.failed", "repo.changed", "context.invalidated", "task.completed"}

	for _, trigger := range newTriggers {
		t.Run(trigger, func(t *testing.T) {
			rule := automationRule{
				Name:    "test-rule-" + trigger,
				Trigger: trigger,
				Actions: []automationAction{
					{
						Type: "create_task",
						Task: automationTaskSpec{
							Instructions: "Test action for " + trigger,
						},
					},
				},
			}
			normalized, err := normalizeAutomationRule(rule)
			if err != nil {
				t.Errorf("normalizeAutomationRule(%q) returned error: %v", trigger, err)
			}
			if normalized.Trigger != trigger {
				t.Errorf("expected trigger %q, got %q", trigger, normalized.Trigger)
			}
		})
	}
}
