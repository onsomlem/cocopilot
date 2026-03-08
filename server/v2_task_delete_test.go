package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2TaskDeleteSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	parentResult, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Delete parent", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert parent task: %v", err)
	}
	parentID, err := parentResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read parent task id: %v", err)
	}

	childResult, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, parent_task_id, project_id) VALUES (?, ?, ?, ?, ?)",
		"Child task", StatusNotPicked, TaskStatusQueued, parentID, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert child task: %v", err)
	}
	childID, err := childResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read child task id: %v", err)
	}

	if _, err := CreateLease(testDB, int(parentID), "agent-delete", "exclusive"); err != nil {
		t.Fatalf("failed to create lease: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/tasks/%d", parentID), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	deleted, ok := resp["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %T", resp["task"])
	}
	if int(deleted["id"].(float64)) != int(parentID) {
		t.Fatalf("expected task.id %d, got %v", parentID, deleted["id"])
	}

	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", parentID).Scan(&count); err != nil {
		t.Fatalf("failed to query deleted task: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected deleted task count 0, got %d", count)
	}

	var parent sql.NullInt64
	if err := testDB.QueryRow("SELECT parent_task_id FROM tasks WHERE id = ?", childID).Scan(&parent); err != nil {
		t.Fatalf("failed to query child task parent: %v", err)
	}
	if parent.Valid {
		t.Fatalf("expected child parent_task_id to be cleared, got %v", parent.Int64)
	}

	lease, err := GetLeaseByTaskID(testDB, int(parentID))
	if err != nil {
		t.Fatalf("failed to lookup lease: %v", err)
	}
	if lease != nil {
		t.Fatalf("expected lease to be released, got %+v", lease)
	}
}

func TestV2TaskDeletePolicyBlock(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Block delete", nil, []PolicyRule{{
		"type":   "task.delete.block",
		"reason": "Deletion disabled",
	}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Blocked delete", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/tasks/%d", taskID), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "FORBIDDEN")
	details, ok := errField["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error.details object, got %T", errField["details"])
	}
	if details["reason"] != "Deletion disabled" {
		t.Fatalf("expected reason 'Deletion disabled', got %v", details["reason"])
	}

	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", taskID).Scan(&count); err != nil {
		t.Fatalf("failed to query task: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected task to remain, got count %d", count)
	}
}

func TestV2TaskDeletePolicyAllow(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Disabled delete", nil, []PolicyRule{{
		"type":   "task.delete.block",
		"reason": "Not enforced",
	}}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Allowed delete", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/tasks/%d", taskID), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2TaskDeleteNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v2/tasks/999999", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2TaskDeleteMethodNotAllowed(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Wrong method", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d", taskID), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
