package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2TaskCreateSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"instructions":"Draft the release notes"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	task, ok := resp["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %T", resp["task"])
	}

	if task["instructions"] != "Draft the release notes" {
		t.Fatalf("expected instructions to match, got %v", task["instructions"])
	}
	if task["project_id"] != "proj_default" {
		t.Fatalf("expected project_id proj_default, got %v", task["project_id"])
	}
	if task["status_v1"] != string(StatusNotPicked) {
		t.Fatalf("expected status_v1 %s, got %v", StatusNotPicked, task["status_v1"])
	}
	if task["status_v2"] != string(TaskStatusQueued) {
		t.Fatalf("expected status_v2 %s, got %v", TaskStatusQueued, task["status_v2"])
	}
	assertTaskUpdatedAtPresent(t, task)
}

func TestV2TaskCreatePolicyBlock(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Block create", nil, []PolicyRule{{
		"type":   "task.create.block",
		"reason": "No new tasks",
	}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	var before int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&before); err != nil {
		t.Fatalf("failed to count tasks: %v", err)
	}

	payload := `{"instructions":"Blocked task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(payload))
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
	if details["reason"] != "No new tasks" {
		t.Fatalf("expected reason 'No new tasks', got %v", details["reason"])
	}

	var after int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&after); err != nil {
		t.Fatalf("failed to count tasks after: %v", err)
	}
	if before != after {
		t.Fatalf("expected task count %d, got %d", before, after)
	}
}

func TestV2TaskCreatePolicyAllow(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Disabled create", nil, []PolicyRule{{
		"type":   "task.create.block",
		"reason": "Not enforced",
	}}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"instructions":"Allowed task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2TaskCreateValidationError(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"instructions":"   "}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2TaskCreateMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPatch, "/api/v2/tasks", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
