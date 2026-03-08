package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2ProjectTaskCreateSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Project Create", "/tmp/project-create", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	payload := `{"instructions":"Ship project task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/tasks", strings.NewReader(payload))
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
	if task["instructions"] != "Ship project task" {
		t.Fatalf("expected instructions to match, got %v", task["instructions"])
	}
	if task["project_id"] != project.ID {
		t.Fatalf("expected project_id %s, got %v", project.ID, task["project_id"])
	}
	assertTaskUpdatedAtPresent(t, task)
}

func TestV2ProjectTaskCreatePolicyBlock(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Project Block", "/tmp/project-block", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	_, err = CreatePolicy(testDB, project.ID, "Block create", nil, []PolicyRule{{
		"type":   "task.create.block",
		"reason": "Project locked",
	}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	var before int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks WHERE project_id = ?", project.ID).Scan(&before); err != nil {
		t.Fatalf("failed to count tasks: %v", err)
	}

	payload := `{"instructions":"Blocked project task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/tasks", strings.NewReader(payload))
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
	if details["reason"] != "Project locked" {
		t.Fatalf("expected reason 'Project locked', got %v", details["reason"])
	}

	var after int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks WHERE project_id = ?", project.ID).Scan(&after); err != nil {
		t.Fatalf("failed to count tasks after: %v", err)
	}
	if before != after {
		t.Fatalf("expected task count %d, got %d", before, after)
	}
}

func TestV2ProjectTaskCreatePolicyAllow(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Project Allow", "/tmp/project-allow", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	_, err = CreatePolicy(testDB, project.ID, "Disabled create", nil, []PolicyRule{{
		"type":   "task.create.block",
		"reason": "Not enforced",
	}}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"instructions":"Allowed project task"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2ProjectTaskCreateProjectMismatch(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Project Mismatch", "/tmp/project-mismatch", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	payload := `{"instructions":"Wrong project","project_id":"proj_default"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectTaskCreateNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"instructions":"Missing project"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_missing/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectTaskCreateValidationError(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"instructions":"   "}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}