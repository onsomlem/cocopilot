package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2TaskClaimSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Claim me", StatusNotPicked, TaskStatusQueued, "proj_default",
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
		"agent_id": "agent-claim",
		"mode":     "exclusive",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), bytes.NewReader(body))
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

	lease, ok := resp["lease"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected lease object, got %T", resp["lease"])
	}
	task, ok := resp["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %T", resp["task"])
	}

	if int(lease["task_id"].(float64)) != int(taskID) {
		t.Fatalf("expected lease.task_id %d, got %v", taskID, lease["task_id"])
	}
	if int(task["id"].(float64)) != int(taskID) {
		t.Fatalf("expected task.id %d, got %v", taskID, task["id"])
	}
	if task["status_v1"] != string(StatusInProgress) {
		t.Fatalf("expected task.status_v1 %s, got %v", StatusInProgress, task["status_v1"])
	}
	if task["status_v2"] != string(TaskStatusClaimed) {
		t.Fatalf("expected task.status_v2 %s, got %v", TaskStatusClaimed, task["status_v2"])
	}
	assertTaskUpdatedAtPresent(t, task)

	var updatedAt string
	if err := testDB.QueryRow("SELECT updated_at FROM tasks WHERE id = ?", taskID).Scan(&updatedAt); err != nil {
		t.Fatalf("failed to read updated_at: %v", err)
	}
	if updatedAt == "" {
		t.Fatal("expected updated_at to be set after claim")
	}
	if updatedAt == oldUpdatedAt {
		t.Fatalf("expected updated_at to change after claim, still %s", updatedAt)
	}
}

func TestV2TaskClaimConflict(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Already leased", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	if _, err := CreateLease(testDB, int(taskID), "agent-a", "exclusive"); err != nil {
		t.Fatalf("failed to create lease: %v", err)
	}

	payload := map[string]interface{}{
		"agent_id": "agent-b",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "CONFLICT")
}

func TestV2TaskClaimNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := map[string]interface{}{
		"agent_id": "agent-missing",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks/999999/claim", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2TaskClaimMethodNotAllowed(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
