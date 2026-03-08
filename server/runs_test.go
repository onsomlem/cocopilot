package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupRunsTestDB creates a fresh test database with v2 migrations
func setupRunsTestDB(t *testing.T) (*sql.DB, func()) {
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_runs_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations to set up all tables
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Replace global db with test db
	oldDB := db
	db = testDB

	cleanup := func() {
		db.Close()
		db = oldDB
		os.Remove(testDBPath)
	}

	return testDB, cleanup
}

// TestRunCreationOnTaskClaim tests that a run is created when a task is claimed
func TestRunCreationOnTaskClaim(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Log("Test: Run creation when task is claimed")

	// Step 1: Create a task
	formData := url.Values{}
	formData.Set("instructions", "Test task for run creation")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed with status %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	taskID := int(createResp["task_id"].(float64))
	t.Logf("Created task_id: %d", taskID)

	// Verify no runs exist yet
	var runCount int
	err := db.QueryRow("SELECT COUNT(*) FROM runs WHERE task_id = ?", taskID).Scan(&runCount)
	if err != nil {
		t.Fatalf("Failed to query runs: %v", err)
	}
	if runCount != 0 {
		t.Errorf("Expected 0 runs before claiming, got %d", runCount)
	}

	// Step 2: Claim the task (this should create a run)
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()

	getTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GetTask failed with status %d: %s", w.Code, w.Body.String())
	}

	// Verify a run was created
	err = db.QueryRow("SELECT COUNT(*) FROM runs WHERE task_id = ?", taskID).Scan(&runCount)
	if err != nil {
		t.Fatalf("Failed to query runs after claim: %v", err)
	}
	if runCount != 1 {
		t.Errorf("Expected 1 run after claiming, got %d", runCount)
	}

	// Verify run has correct initial state
	var runID, agentID, status, startedAt string
	var finishedAt sql.NullString
	err = db.QueryRow(`
		SELECT id, agent_id, status, started_at, finished_at 
		FROM runs WHERE task_id = ?
	`, taskID).Scan(&runID, &agentID, &status, &startedAt, &finishedAt)
	if err != nil {
		t.Fatalf("Failed to query run details: %v", err)
	}

	if status != string(RunStatusRunning) {
		t.Errorf("Expected run status RUNNING, got %s", status)
	}
	if agentID != "default-agent" {
		t.Errorf("Expected agent_id 'default-agent', got %s", agentID)
	}
	if startedAt == "" {
		t.Error("Expected started_at to be set")
	}
	if finishedAt.Valid {
		t.Error("Expected finished_at to be NULL for running task")
	}

	t.Logf("Run created successfully: id=%s, status=%s", runID, status)
}

// TestRunCompletionOnTaskSave tests that a run is completed when a task is saved
func TestRunCompletionOnTaskSave(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Log("Test: Run completion when task is saved")

	// Step 1: Create and claim a task
	formData := url.Values{}
	formData.Set("instructions", "Test task for run completion")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	taskID := int(createResp["task_id"].(float64))

	// Claim the task
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()
	getTaskHandler(w, req)

	// Get the run ID
	var runID string
	err := db.QueryRow("SELECT id FROM runs WHERE task_id = ?", taskID).Scan(&runID)
	if err != nil {
		t.Fatalf("Failed to get run ID: %v", err)
	}

	// Step 2: Complete the task (this should complete the run)
	formData = url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", taskID))
	formData.Set("message", "Task completed successfully")

	req = httptest.NewRequest("POST", "/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	saveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Save failed with status %d: %s", w.Code, w.Body.String())
	}

	// Verify run was completed
	var status string
	var finishedAt sql.NullString
	err = db.QueryRow(`
		SELECT status, finished_at 
		FROM runs WHERE id = ?
	`, runID).Scan(&status, &finishedAt)
	if err != nil {
		t.Fatalf("Failed to query completed run: %v", err)
	}

	if status != string(RunStatusSucceeded) {
		t.Errorf("Expected run status SUCCEEDED, got %s", status)
	}
	if !finishedAt.Valid {
		t.Error("Expected finished_at to be set for completed run")
	}

	t.Logf("Run completed successfully: id=%s, status=%s, finished_at=%s", runID, status, finishedAt.String)
}

// TestGetRunEndpoint tests the GET /api/v2/runs/:id endpoint
func TestGetRunEndpoint(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Log("Test: GET /api/v2/runs/:id endpoint")

	// Step 1: Create and claim a task to generate a run
	formData := url.Values{}
	formData.Set("instructions", "Test task for run API")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	taskID := int(createResp["task_id"].(float64))

	// Claim the task to create a run
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()
	getTaskHandler(w, req)

	// Get the run ID
	var runID string
	err := db.QueryRow("SELECT id FROM runs WHERE task_id = ?", taskID).Scan(&runID)
	if err != nil {
		t.Fatalf("Failed to get run ID: %v", err)
	}

	// Step 2: Test GET /api/v2/runs/:id
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v2/runs/%s", runID), nil)
	w = httptest.NewRecorder()

	v2GetRunHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GetRun failed with status %d: %s", w.Code, w.Body.String())
	}

	// Verify response structure
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	run, ok := resp["run"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response should contain 'run' object")
	}

	if run["id"] != runID {
		t.Errorf("Expected run id %s, got %v", runID, run["id"])
	}
	if run["task_id"] != float64(taskID) {
		t.Errorf("Expected task_id %d, got %v", taskID, run["task_id"])
	}
	if run["agent_id"] != "default-agent" {
		t.Errorf("Expected agent_id 'default-agent', got %v", run["agent_id"])
	}
	if run["status"] != string(RunStatusRunning) {
		t.Errorf("Expected status RUNNING, got %v", run["status"])
	}
	if run["started_at"] == "" {
		t.Error("Expected started_at to be set")
	}

	t.Logf("GET /api/v2/runs/:id successful: %+v", run)
}

// TestGetRunEndpointIncludesDetails verifies steps/logs/artifacts/tool invocations are included.
func TestGetRunEndpointIncludesDetails(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("instructions", "Test task for run details")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed with status %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	taskID := int(createResp["task_id"].(float64))
	run, err := CreateRun(db, taskID, "agent-detail")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	if _, err := CreateRunStep(db, run.ID, "Initialize", StepStatusStarted, map[string]interface{}{"phase": "init"}); err != nil {
		t.Fatalf("CreateRunStep failed: %v", err)
	}
	if err := CreateRunLog(db, run.ID, "stdout", "hello"); err != nil {
		t.Fatalf("CreateRunLog failed: %v", err)
	}
	if _, err := CreateArtifact(db, run.ID, "file", "/tmp/output.txt", nil, nil, map[string]interface{}{"filename": "output.txt"}); err != nil {
		t.Fatalf("CreateArtifact failed: %v", err)
	}
	invocation, err := CreateToolInvocation(db, run.ID, "grep_search", map[string]interface{}{"query": "main.go"})
	if err != nil {
		t.Fatalf("CreateToolInvocation failed: %v", err)
	}
	if err := UpdateToolInvocationOutput(db, invocation.ID, map[string]interface{}{"matches": 1}); err != nil {
		t.Fatalf("UpdateToolInvocationOutput failed: %v", err)
	}

	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v2/runs/%s", run.ID), nil)
	w = httptest.NewRecorder()
	v2GetRunHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GetRun failed with status %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result, ok := resp["run"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response should contain 'run' object")
	}

	steps, ok := result["steps"].([]interface{})
	if !ok || len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %v", result["steps"])
	}
	logs, ok := result["logs"].([]interface{})
	if !ok || len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %v", result["logs"])
	}
	artifacts, ok := result["artifacts"].([]interface{})
	if !ok || len(artifacts) != 1 {
		t.Fatalf("Expected 1 artifact, got %v", result["artifacts"])
	}
	invocations, ok := result["tool_invocations"].([]interface{})
	if !ok || len(invocations) != 1 {
		t.Fatalf("Expected 1 tool invocation, got %v", result["tool_invocations"])
	}

	logEntry, ok := logs[0].(map[string]interface{})
	if !ok || logEntry["stream"] != "stdout" {
		t.Fatalf("Expected log stream stdout, got %v", logs[0])
	}
}

func TestV2AddRunStepSuccess(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("instructions", "Test task for run step")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed with status %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	taskID := int(createResp["task_id"].(float64))
	run, err := CreateRun(db, taskID, "agent-step")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	reqBody := `{"name":"Initialize","status":"STARTED","details":{"phase":"init"}}`
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v2/runs/%s/steps", run.ID), strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Add run step failed with status %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	step, ok := resp["step"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response should contain 'step' object")
	}

	if step["name"] != "Initialize" {
		t.Errorf("Expected step name Initialize, got %v", step["name"])
	}
	if step["status"] != string(StepStatusStarted) {
		t.Errorf("Expected status STARTED, got %v", step["status"])
	}
	if step["run_id"] != run.ID {
		t.Errorf("Expected run_id %s, got %v", run.ID, step["run_id"])
	}

	steps, err := GetRunSteps(db, run.ID)
	if err != nil {
		t.Fatalf("GetRunSteps failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}
	if steps[0].Name != "Initialize" {
		t.Errorf("Expected stored step name Initialize, got %s", steps[0].Name)
	}
}

func TestV2AddRunStepValidation(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/steps", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_JSON")
	})

	t.Run("invalid_status", func(t *testing.T) {
		reqBody := `{"name":"Initialize","status":"DONE"}`
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/steps", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})

	t.Run("invalid_run_id", func(t *testing.T) {
		reqBody := `{"name":"Initialize","status":"STARTED"}`
		req := httptest.NewRequest("POST", "/api/v2/runs//steps", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})
}

func TestV2AddRunStepNotFound(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	reqBody := `{"name":"Initialize","status":"STARTED"}`
	req := httptest.NewRequest("POST", "/api/v2/runs/run_missing/steps", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2RunStepsMethodNotAllowed(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/runs/run_123/steps", nil)
	w := httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodGet {
		t.Fatalf("expected details.method %s, got %v", http.MethodGet, details["method"])
	}
	allowed, ok := details["allowed_methods"].([]interface{})
	if !ok || len(allowed) != 1 || allowed[0] != http.MethodPost {
		t.Fatalf("expected allowed_methods [%s], got %v", http.MethodPost, details["allowed_methods"])
	}
}

func TestV2AddRunLogSuccess(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("instructions", "Test task for run logs")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed with status %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	taskID := int(createResp["task_id"].(float64))
	run, err := CreateRun(db, taskID, "agent-log")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	reqBody := `{"stream":"stdout","chunk":"hello"}`
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v2/runs/%s/logs", run.ID), strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Add run log failed with status %d: %s", w.Code, w.Body.String())
	}

	logs, err := GetRunLogs(db, run.ID)
	if err != nil {
		t.Fatalf("GetRunLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}
	if logs[0].Stream != "stdout" {
		t.Errorf("Expected stream stdout, got %s", logs[0].Stream)
	}
	if logs[0].Chunk != "hello" {
		t.Errorf("Expected chunk hello, got %s", logs[0].Chunk)
	}
}

func TestV2AddRunLogValidation(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/logs", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_JSON")
	})

	t.Run("invalid_stream", func(t *testing.T) {
		reqBody := `{"stream":"debug","chunk":"hello"}`
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/logs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})

	t.Run("missing_chunk", func(t *testing.T) {
		reqBody := `{"stream":"stdout","chunk":""}`
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/logs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})

	t.Run("invalid_run_id", func(t *testing.T) {
		reqBody := `{"stream":"stdout","chunk":"hello"}`
		req := httptest.NewRequest("POST", "/api/v2/runs//logs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})
}

func TestV2AddRunLogNotFound(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	reqBody := `{"stream":"stdout","chunk":"hello"}`
	req := httptest.NewRequest("POST", "/api/v2/runs/run_missing/logs", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2RunLogsMethodNotAllowed(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/runs/run_123/logs", nil)
	w := httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodGet {
		t.Fatalf("expected details.method %s, got %v", http.MethodGet, details["method"])
	}
	allowed, ok := details["allowed_methods"].([]interface{})
	if !ok || len(allowed) != 1 || allowed[0] != http.MethodPost {
		t.Fatalf("expected allowed_methods [%s], got %v", http.MethodPost, details["allowed_methods"])
	}
}

func TestV2AddRunArtifactSuccess(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("instructions", "Test task for run artifacts")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed with status %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	taskID := int(createResp["task_id"].(float64))
	run, err := CreateRun(db, taskID, "agent-artifact")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	reqBody := `{"kind":"file","storage_ref":"/tmp/output.txt","size":12,"sha256":"abc123","metadata":{"filename":"output.txt"}}`
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/v2/runs/%s/artifacts", run.ID), strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Add run artifact failed with status %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	artifact, ok := resp["artifact"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response should contain 'artifact' object")
	}

	if artifact["run_id"] != run.ID {
		t.Errorf("Expected run_id %s, got %v", run.ID, artifact["run_id"])
	}
	if artifact["kind"] != "file" {
		t.Errorf("Expected kind file, got %v", artifact["kind"])
	}
	if artifact["storage_ref"] != "/tmp/output.txt" {
		t.Errorf("Expected storage_ref /tmp/output.txt, got %v", artifact["storage_ref"])
	}
	if artifact["size"] != float64(12) {
		t.Errorf("Expected size 12, got %v", artifact["size"])
	}

	artifacts, err := GetArtifactsByRunID(db, run.ID)
	if err != nil {
		t.Fatalf("GetArtifactsByRunID failed: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("Expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].StorageRef != "/tmp/output.txt" {
		t.Errorf("Expected stored storage_ref /tmp/output.txt, got %s", artifacts[0].StorageRef)
	}
}

func TestV2AddRunArtifactValidation(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/artifacts", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_JSON")
	})

	t.Run("invalid_kind", func(t *testing.T) {
		reqBody := `{"kind":"binary","storage_ref":"/tmp/out","size":1}`
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/artifacts", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})

	t.Run("missing_storage_ref", func(t *testing.T) {
		reqBody := `{"kind":"file","storage_ref":"","size":1}`
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/artifacts", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})

	t.Run("missing_size", func(t *testing.T) {
		reqBody := `{"kind":"file","storage_ref":"/tmp/out"}`
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/artifacts", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})

	t.Run("negative_size", func(t *testing.T) {
		reqBody := `{"kind":"file","storage_ref":"/tmp/out","size":-1}`
		req := httptest.NewRequest("POST", "/api/v2/runs/run_123/artifacts", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})

	t.Run("invalid_run_id", func(t *testing.T) {
		reqBody := `{"kind":"file","storage_ref":"/tmp/out","size":1}`
		req := httptest.NewRequest("POST", "/api/v2/runs//artifacts", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		v2RunsRouteHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	})
}

func TestV2AddRunArtifactNotFound(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	reqBody := `{"kind":"file","storage_ref":"/tmp/out","size":1}`
	req := httptest.NewRequest("POST", "/api/v2/runs/run_missing/artifacts", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2RunArtifactsMethodNotAllowed(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/runs/run_123/artifacts", nil)
	w := httptest.NewRecorder()

	v2RunsRouteHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodGet {
		t.Fatalf("expected details.method %s, got %v", http.MethodGet, details["method"])
	}
	allowed, ok := details["allowed_methods"].([]interface{})
	if !ok || len(allowed) != 1 || allowed[0] != http.MethodPost {
		t.Fatalf("expected allowed_methods [%s], got %v", http.MethodPost, details["allowed_methods"])
	}
}

// TestGetRunEndpointNotFound tests 404 handling for non-existent runs
func TestGetRunEndpointNotFound(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Log("Test: GET /api/v2/runs/:id with non-existent run")

	req := httptest.NewRequest("GET", "/api/v2/runs/run_nonexistent", nil)
	w := httptest.NewRecorder()

	v2GetRunHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object in response")
	}

	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("Expected error code NOT_FOUND, got %v", errObj["code"])
	}
}

// TestMultipleRunsPerTask tests that multiple runs can be created for a task
func TestMultipleRunsPerTask(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Log("Test: Multiple runs for the same task")

	// Create a task
	formData := url.Values{}
	formData.Set("instructions", "Test task for multiple runs")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	taskID := int(createResp["task_id"].(float64))

	// Claim the task (creates first run)
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()
	getTaskHandler(w, req)

	// Save the task (completes first run)
	formData = url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", taskID))
	formData.Set("message", "First attempt")
	req = httptest.NewRequest("POST", "/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	saveHandler(w, req)

	// Manually mark task as NOT_PICKED again for second attempt
	_, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", StatusNotPicked, taskID)
	if err != nil {
		t.Fatalf("Failed to reset task status: %v", err)
	}

	// Claim again (should create second run)
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()
	getTaskHandler(w, req)

	// Verify two runs exist for this task
	var runCount int
	err = db.QueryRow("SELECT COUNT(*) FROM runs WHERE task_id = ?", taskID).Scan(&runCount)
	if err != nil {
		t.Fatalf("Failed to count runs: %v", err)
	}
	if runCount != 2 {
		t.Errorf("Expected 2 runs for task, got %d", runCount)
	}

	// Verify one is completed, one is running
	var succeededCount, runningCount int
	db.QueryRow("SELECT COUNT(*) FROM runs WHERE task_id = ? AND status = ?", taskID, RunStatusSucceeded).Scan(&succeededCount)
	db.QueryRow("SELECT COUNT(*) FROM runs WHERE task_id = ? AND status = ?", taskID, RunStatusRunning).Scan(&runningCount)

	if succeededCount != 1 {
		t.Errorf("Expected 1 succeeded run, got %d", succeededCount)
	}
	if runningCount != 1 {
		t.Errorf("Expected 1 running run, got %d", runningCount)
	}

	t.Logf("Multiple runs test successful: %d total, %d succeeded, %d running", runCount, succeededCount, runningCount)
}

// TestRunsEndToEnd tests the complete lifecycle
func TestRunsEndToEnd(t *testing.T) {
	_, cleanup := setupRunsTestDB(t)
	defer cleanup()

	t.Log("Test: End-to-end runs lifecycle")

	// 1. Create task
	formData := url.Values{}
	formData.Set("instructions", "E2E test task")
	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	taskID := int(createResp["task_id"].(float64))
	t.Logf("✓ Task created: %d", taskID)

	// 2. Claim task (should create run)
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()
	getTaskHandler(w, req)

	var runID string
	err := db.QueryRow("SELECT id FROM runs WHERE task_id = ? AND status = ?", taskID, RunStatusRunning).Scan(&runID)
	if err != nil {
		t.Fatalf("Failed to find running run: %v", err)
	}
	t.Logf("✓ Run created on claim: %s", runID)

	// 3. Retrieve run via API
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v2/runs/%s", runID), nil)
	w = httptest.NewRecorder()
	v2GetRunHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to get run via API: %s", w.Body.String())
	}
	t.Logf("✓ Run retrieved via API")

	// 4. Complete task (should complete run)
	formData = url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", taskID))
	formData.Set("message", "E2E test completed")
	req = httptest.NewRequest("POST", "/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	saveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to save task: %s", w.Body.String())
	}
	t.Logf("✓ Task saved")

	// 5. Verify run is completed
	var status string
	var finishedAt sql.NullString
	err = db.QueryRow("SELECT status, finished_at FROM runs WHERE id = ?", runID).Scan(&status, &finishedAt)
	if err != nil {
		t.Fatalf("Failed to get final run status: %v", err)
	}

	if status != string(RunStatusSucceeded) {
		t.Errorf("Expected final status SUCCEEDED, got %s", status)
	}
	if !finishedAt.Valid {
		t.Error("Expected finished_at to be set")
	}
	t.Logf("✓ Run completed: status=%s, finished_at=%s", status, finishedAt.String)

	// 6. Retrieve completed run via API
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v2/runs/%s", runID), nil)
	w = httptest.NewRecorder()
	v2GetRunHandler(w, req)

	var finalResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&finalResp)
	run := finalResp["run"].(map[string]interface{})

	if run["status"] != string(RunStatusSucceeded) {
		t.Errorf("API should show status SUCCEEDED, got %v", run["status"])
	}
	if run["finished_at"] == nil {
		t.Error("API should show finished_at")
	}
	t.Logf("✓ Completed run verified via API")

	t.Log("✅ End-to-end test passed!")
}
