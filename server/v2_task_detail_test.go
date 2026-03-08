package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2TaskDetailSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	parentResult, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Parent task", StatusComplete, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert parent task: %v", err)
	}
	parentID, err := parentResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read parent task id: %v", err)
	}

	childResult, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, parent_task_id, project_id) VALUES (?, ?, ?, ?)",
		"Child task", StatusNotPicked, parentID, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert child task: %v", err)
	}
	childID, err := childResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read child task id: %v", err)
	}

	latestRun, err := CreateRun(testDB, int(childID), "agent_x")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d", childID), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	task, ok := resp["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %T", resp["task"])
	}
	if int(task["id"].(float64)) != int(childID) {
		t.Fatalf("expected task.id %d, got %v", childID, task["id"])
	}
	assertTaskUpdatedAtPresent(t, task)

	chain, ok := resp["parent_chain"].([]interface{})
	if !ok {
		t.Fatalf("expected parent_chain array, got %T", resp["parent_chain"])
	}
	if len(chain) != 1 {
		t.Fatalf("expected 1 parent in chain, got %d", len(chain))
	}
	parentTask, ok := chain[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected parent task object, got %T", chain[0])
	}
	if int(parentTask["id"].(float64)) != int(parentID) {
		t.Fatalf("expected parent id %d, got %v", parentID, parentTask["id"])
	}
	assertTaskUpdatedAtPresent(t, parentTask)

	run, ok := resp["latest_run"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected latest_run object, got %T", resp["latest_run"])
	}
	if run["id"] != latestRun.ID {
		t.Fatalf("expected latest_run.id %s, got %v", latestRun.ID, run["id"])
	}
}

func TestV2TaskDetailNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/tasks/999999", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2TaskDetailMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks/123", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
