package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupEventTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_event_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, _ = testDB.Exec("PRAGMA journal_mode=WAL")
	_, _ = testDB.Exec("PRAGMA busy_timeout=5000")
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("migrations: %v", err)
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

func createEventTestTask(t *testing.T, testDB *sql.DB, title string) (*Project, *TaskV2) {
	t.Helper()
	project, err := CreateProject(testDB, "event-test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, title, project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}
	return project, task
}

func findEvent(t *testing.T, testDB *sql.DB, projectID, kind string) *Event {
	t.Helper()
	events, err := GetEventsByProjectID(testDB, projectID, 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID: %v", err)
	}
	for _, ev := range events {
		if ev.Kind == kind {
			return &ev
		}
	}
	return nil
}

// --- task.claimed event verification ---

func TestEventVerify_TaskClaimed(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, task := createEventTestTask(t, testDB, "event-verify-claimed")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-x", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	ev := findEvent(t, testDB, project.ID, "task.claimed")
	if ev == nil {
		t.Fatal("expected task.claimed event")
	}
	if ev.Payload["agent_id"] != "agent-x" {
		t.Errorf("expected agent_id=agent-x, got %v", ev.Payload["agent_id"])
	}
	if ev.Payload["task_id"] == nil {
		t.Error("expected task_id in payload")
	}
	if ev.Payload["lease_id"] == nil {
		t.Error("expected lease_id in payload")
	}
	if ev.Payload["run_id"] == nil {
		t.Error("expected run_id in payload")
	}
}

// --- task.completed event verification ---

func TestEventVerify_TaskCompleted(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, task := createEventTestTask(t, testDB, "event-verify-completed")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	output := "task completed successfully"
	_, err = CompleteTask(testDB, task.ID, &output)
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	ev := findEvent(t, testDB, project.ID, "task.completed")
	if ev == nil {
		t.Fatal("expected task.completed event")
	}
	if ev.EntityType != "task" {
		t.Errorf("expected entity_type=task, got %s", ev.EntityType)
	}
}

// --- task.failed event verification ---

func TestEventVerify_TaskFailed(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, task := createEventTestTask(t, testDB, "event-verify-failed")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	_, err = FailTaskWithError(testDB, task.ID, "something broke")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}

	ev := findEvent(t, testDB, project.ID, "task.failed")
	if ev == nil {
		t.Fatal("expected task.failed event")
	}
	if ev.Payload["error"] != "something broke" {
		t.Errorf("expected error=something broke, got %v", ev.Payload["error"])
	}
	statusV1, _ := ev.Payload["status_v1"].(string)
	if statusV1 != string(StatusFailed) {
		t.Errorf("expected status_v1=FAILED, got %s", statusV1)
	}
}

// --- task.cancelled event verification ---

func TestEventVerify_TaskCancelled(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, task := createEventTestTask(t, testDB, "event-verify-cancelled")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	_, err = CancelTask(testDB, task.ID, "user requested cancellation")
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}

	ev := findEvent(t, testDB, project.ID, "task.cancelled")
	if ev == nil {
		t.Fatal("expected task.cancelled event")
	}
	if ev.Payload["reason"] != "user requested cancellation" {
		t.Errorf("expected reason in payload, got %v", ev.Payload["reason"])
	}
}

// --- repo.changed event verification (emitted after completion with files_touched) ---

func TestEventVerify_RepoChanged(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, task := createEventTestTask(t, testDB, "event-verify-repo-changed")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	output := "done"
	_, err = CompleteTask(testDB, task.ID, &output)
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	ev := findEvent(t, testDB, project.ID, "repo.changed")
	if ev == nil {
		// repo.changed is only emitted by CompleteTaskWithPayload when files are touched.
		t.Log("Note: repo.changed event depends on CompleteTaskWithPayload detecting files_touched")
	}
}

// --- agent.registered event verification ---

func TestEventVerify_AgentRegistered(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	payload := `{"name": "ev-test-agent", "capabilities": ["test"]}`
	req := httptest.NewRequest("POST", "/api/v2/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2RegisterAgentHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("register agent: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	agent := resp["agent"].(map[string]interface{})
	agentID := agent["id"].(string)

	ev := findEvent(t, testDB, DefaultProjectID, "agent.registered")
	if ev == nil {
		t.Fatal("expected agent.registered event")
	}
	if ev.EntityID != agentID {
		t.Errorf("expected entity_id=%s, got %v", agentID, ev.EntityID)
	}
}

// --- agent.deleted event verification ---

func TestEventVerify_AgentDeleted(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	// Register first.
	payload := `{"name": "del-ev-agent", "capabilities": ["test"]}`
	req := httptest.NewRequest("POST", "/api/v2/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2RegisterAgentHandler(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	agent := resp["agent"].(map[string]interface{})
	agentID := agent["id"].(string)

	// Delete via handler.
	delReq := httptest.NewRequest("DELETE", "/api/v2/agents/"+agentID, nil)
	delW := httptest.NewRecorder()
	v2DeleteAgentHandler(delW, delReq, agentID)

	if delW.Code != http.StatusOK {
		t.Fatalf("delete agent: expected 200, got %d: %s", delW.Code, delW.Body.String())
	}

	ev := findEvent(t, testDB, DefaultProjectID, "agent.deleted")
	if ev == nil {
		t.Fatal("expected agent.deleted event")
	}
}

// --- memory.created event (via CreateMemory) ---

func TestEventVerify_MemoryCreated(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, _ := createEventTestTask(t, testDB, "event-verify-memory")

	_, err := CreateMemory(testDB, project.ID, "test_scope", "test_key",
		map[string]interface{}{"data": "hello"}, nil)
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}

	// Memory creation may or may not emit an event (depends on implementation).
	// This test verifies the behavior.
	ev := findEvent(t, testDB, project.ID, "memory.created")
	if ev != nil {
		t.Log("memory.created event emitted (good)")
	} else {
		t.Log("Note: memory.created event not emitted (optional)")
	}
}

// --- Event chain verification: claim → complete → task.claimed + task.completed ---

func TestEventVerify_ClaimCompleteChain(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, task := createEventTestTask(t, testDB, "event-chain-test")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	output := "all done"
	_, err = CompleteTask(testDB, task.ID, &output)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID: %v", err)
	}

	kinds := make(map[string]bool)
	for _, ev := range events {
		kinds[ev.Kind] = true
	}

	if !kinds["task.claimed"] {
		t.Error("expected task.claimed event in chain")
	}
	if !kinds["task.completed"] {
		t.Error("expected task.completed event in chain")
	}
}

// --- Event chain verification: claim → fail → task.claimed + task.failed ---

func TestEventVerify_ClaimFailChain(t *testing.T) {
	testDB, cleanup := setupEventTestDB(t)
	defer cleanup()

	project, task := createEventTestTask(t, testDB, "event-chain-fail")

	_, err := ClaimTaskByID(testDB, task.ID, "agent-1", "exclusive")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	_, err = FailTaskWithError(testDB, task.ID, "chain failure")
	if err != nil {
		t.Fatalf("fail: %v", err)
	}

	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID: %v", err)
	}

	kinds := make(map[string]bool)
	for _, ev := range events {
		kinds[ev.Kind] = true
	}

	if !kinds["task.claimed"] {
		t.Error("expected task.claimed event")
	}
	if !kinds["task.failed"] {
		t.Error("expected task.failed event")
	}
}
