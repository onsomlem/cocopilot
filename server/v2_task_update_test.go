package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2TaskUpdateSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Update Project", "/tmp/update", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	parentResult, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Parent task", StatusNotPicked, TaskStatusQueued, project.ID,
	)
	if err != nil {
		t.Fatalf("failed to insert parent task: %v", err)
	}
	parentID, err := parentResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read parent task id: %v", err)
	}

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Original task", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}
	oldUpdatedAt := "2000-01-01T00:00:00.000000Z"
	if _, err := testDB.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldUpdatedAt, taskID); err != nil {
		t.Fatalf("failed to seed updated_at: %v", err)
	}

	payload := map[string]interface{}{
		"instructions":   "Updated instructions",
		"status":         "RUNNING",
		"project_id":     project.ID,
		"parent_task_id": parentID,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v2/tasks/%d", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

	if task["instructions"] != "Updated instructions" {
		t.Fatalf("expected instructions to update, got %v", task["instructions"])
	}
	if task["status_v1"] != string(StatusInProgress) {
		t.Fatalf("expected status_v1 %s, got %v", StatusInProgress, task["status_v1"])
	}
	if task["status_v2"] != string(TaskStatusRunning) {
		t.Fatalf("expected status_v2 %s, got %v", TaskStatusRunning, task["status_v2"])
	}
	if task["project_id"] != project.ID {
		t.Fatalf("expected project_id %s, got %v", project.ID, task["project_id"])
	}
	if int(task["parent_task_id"].(float64)) != int(parentID) {
		t.Fatalf("expected parent_task_id %d, got %v", parentID, task["parent_task_id"])
	}
	assertTaskUpdatedAtPresent(t, task)

	var updatedAt string
	if err := testDB.QueryRow("SELECT updated_at FROM tasks WHERE id = ?", taskID).Scan(&updatedAt); err != nil {
		t.Fatalf("failed to read updated_at: %v", err)
	}
	if updatedAt == "" {
		t.Fatal("expected updated_at to be set after update")
	}
	if updatedAt == oldUpdatedAt {
		t.Fatalf("expected updated_at to change after update, still %s", updatedAt)
	}
}

func TestV2TaskUpdatePolicyBlock(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Block update", nil, []PolicyRule{{
		"type":   "task.update.block",
		"reason": "Updates paused",
	}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Original task", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"instructions": "Blocked update",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v2/tasks/%d", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
	if details["reason"] != "Updates paused" {
		t.Fatalf("expected reason 'Updates paused', got %v", details["reason"])
	}

	var instructions string
	if err := testDB.QueryRow("SELECT instructions FROM tasks WHERE id = ?", taskID).Scan(&instructions); err != nil {
		t.Fatalf("failed to query task: %v", err)
	}
	if instructions != "Original task" {
		t.Fatalf("expected instructions to remain Original task, got %s", instructions)
	}
}

func TestV2TaskUpdatePolicyAllow(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Disabled update", nil, []PolicyRule{{
		"type":   "task.update.block",
		"reason": "Not enforced",
	}}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Original task", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"instructions": "Allowed update",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v2/tasks/%d", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2TaskUpdateValidationError(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Invalid update", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"status": "NOT_A_STATUS",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v2/tasks/%d", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2TaskUpdateNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := map[string]interface{}{
		"instructions": "Missing task",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/v2/tasks/999999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2TaskUpdateMethodNotAllowed(t *testing.T) {
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
