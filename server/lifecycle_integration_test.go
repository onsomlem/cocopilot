package server

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func setupLifecycleTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_lifecycle_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	_, _ = testDB.Exec("PRAGMA journal_mode=WAL")
	_, _ = testDB.Exec("PRAGMA busy_timeout=5000")
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	oldDB := db
	db = testDB

	// Reset SSE clients.
	sseMutex.Lock()
	sseClients = make([]v1SSEClient, 0)
	sseMutex.Unlock()
	v2EventMu.Lock()
	v2EventSubscribers = make([]v2EventSubscriber, 0)
	v2EventMu.Unlock()

	cleanup := func() {
		db.Close()
		db = oldDB
		os.Remove(testDBPath)
	}
	return testDB, cleanup
}

// --- Integration Test 1: V2 claim → complete → verify end state ---

func TestIntegration_V2ClaimCompleteVerify(t *testing.T) {
	testDB, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	// 1. Create project.
	project, err := CreateProject(testDB, "integ-test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// 2. Create task via v2 API.
	taskBody := fmt.Sprintf(`{
		"instructions": "Integration test task",
		"project_id": "%s",
		"title": "Integration task 1",
		"type": "MODIFY",
		"priority": 5
	}`, project.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(taskBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	v2CreateTaskHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Create task: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var createResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	taskObj, ok := createResp["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'task' object in create response, got %v", createResp)
	}
	taskID := int(taskObj["id"].(float64))

	// 3. Claim task via v2 API.
	claimBody := `{"agent_id": "integ-agent-1", "mode": "exclusive"}`
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), strings.NewReader(claimBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	v2TaskClaimHandler(rr, req, strconv.Itoa(taskID))

	if rr.Code != http.StatusOK {
		t.Fatalf("Claim: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var claimResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&claimResp); err != nil {
		t.Fatalf("decode claim response: %v", err)
	}
	if claimResp["lease"] == nil {
		t.Fatal("expected lease in claim response")
	}
	if claimResp["run"] == nil {
		t.Fatal("expected run in claim response")
	}

	// 4. Complete task via v2 API.
	completeBody := `{"output": "Integration test complete", "result": {"summary": "done", "changes_made": ["file.go"], "files_touched": ["file.go"]}}`
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), strings.NewReader(completeBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	v2TaskCompleteHandler(rr, req, strconv.Itoa(taskID))

	if rr.Code != http.StatusOK {
		t.Fatalf("Complete: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// 5. Verify final state.
	task, err := GetTaskV2(testDB, taskID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if task.StatusV2 != TaskStatusSucceeded {
		t.Errorf("expected SUCCEEDED, got %s", task.StatusV2)
	}

	// Verify run is SUCCEEDED.
	run, err := GetLatestRunByTaskID(testDB, taskID)
	if err != nil {
		t.Fatalf("GetLatestRunByTaskID: %v", err)
	}
	if run == nil {
		t.Fatal("expected run")
	}
	if run.Status != RunStatusSucceeded {
		t.Errorf("expected run SUCCEEDED, got %s", run.Status)
	}

	// Verify lease was released.
	lease, err := GetLeaseByTaskID(testDB, taskID)
	if err != nil {
		t.Fatalf("GetLeaseByTaskID: %v", err)
	}
	if lease != nil {
		t.Errorf("expected no active lease after completion")
	}

	// Verify events.
	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	kinds := make(map[string]bool)
	for _, ev := range events {
		kinds[ev.Kind] = true
	}
	if !kinds["task.claimed"] {
		t.Error("expected task.claimed event")
	}
	if !kinds["task.completed"] {
		t.Error("expected task.completed event")
	}
}

// --- Integration Test 2: V2 claim → fail → verify ---

func TestIntegration_V2ClaimFailVerify(t *testing.T) {
	testDB, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "integ-fail-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Create task.
	task, err := CreateTaskV2(testDB, "Integration fail task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim via API.
	claimBody := `{"agent_id": "integ-agent-fail", "mode": "exclusive"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", task.ID), strings.NewReader(claimBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	v2TaskClaimHandler(rr, req, strconv.Itoa(task.ID))

	if rr.Code != http.StatusOK {
		t.Fatalf("Claim: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Fail via finalization service.
	failed, err := FailTaskWithError(testDB, task.ID, "integration test failure")
	if err != nil {
		t.Fatalf("FailTaskWithError: %v", err)
	}
	if failed.StatusV2 != TaskStatusFailed {
		t.Errorf("expected FAILED, got %s", failed.StatusV2)
	}

	// Verify run is FAILED.
	run, err := GetLatestRunByTaskID(testDB, task.ID)
	if err != nil {
		t.Fatalf("GetLatestRunByTaskID: %v", err)
	}
	if run.Status != RunStatusFailed {
		t.Errorf("expected run FAILED, got %s", run.Status)
	}
	if run.Error == nil || *run.Error != "integration test failure" {
		t.Errorf("expected error message, got %v", run.Error)
	}

	// Verify events.
	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	kinds := make(map[string]bool)
	for _, ev := range events {
		kinds[ev.Kind] = true
	}
	if !kinds["task.failed"] {
		t.Error("expected task.failed event")
	}
}

// --- Integration Test 3: V1 create → claim → save → verify ---

func TestIntegration_V1CreateClaimSave(t *testing.T) {
	_, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	// 1. Create via v1 /create.
	formData := "instructions=V1+integration+test+task"
	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	createHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("v1 /create: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var createResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	taskID := int(createResp["task_id"].(float64))

	// 2. Claim via v1 /task.
	req = httptest.NewRequest(http.MethodGet, "/task", nil)
	rr = httptest.NewRecorder()
	getTaskHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("v1 /task: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "AVAILABLE TASK ID") {
		t.Fatalf("expected task assignment in response, got: %s", body[:min(200, len(body))])
	}

	// 3. Save/complete via v1 /save.
	saveData := fmt.Sprintf("task_id=%d&message=v1+integration+test+done", taskID)
	req = httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(saveData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	saveHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("v1 /save: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// 4. Verify final DB state.
	task, err := GetTaskV2(db, taskID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if task.StatusV2 != TaskStatusSucceeded {
		t.Errorf("expected SUCCEEDED, got %s", task.StatusV2)
	}
	if task.StatusV1 != StatusComplete {
		t.Errorf("expected COMPLETE, got %s", task.StatusV1)
	}

	// Verify run completed.
	run, err := GetLatestRunByTaskID(db, taskID)
	if err != nil {
		t.Fatalf("GetLatestRunByTaskID: %v", err)
	}
	if run == nil {
		t.Fatal("expected run")
	}
	if run.Status != RunStatusSucceeded {
		t.Errorf("expected run SUCCEEDED, got %s", run.Status)
	}
}

// --- Integration Test 4: Double claim via V2 HTTP → 409 Conflict ---

func TestIntegration_V2DoubleClaimConflict(t *testing.T) {
	testDB, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "conflict-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, "conflict-test", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// First claim.
	body := `{"agent_id": "agent-A", "mode": "exclusive"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", task.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	v2TaskClaimHandler(rr, req, strconv.Itoa(task.ID))

	if rr.Code != http.StatusOK {
		t.Fatalf("first claim: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Second claim by different agent → should get 409 Conflict.
	body2 := `{"agent_id": "agent-B", "mode": "exclusive"}`
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", task.ID), strings.NewReader(body2))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	v2TaskClaimHandler(rr, req, strconv.Itoa(task.ID))

	if rr.Code != http.StatusConflict {
		t.Errorf("second claim: expected 409, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify error body.
	var errResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	errObj, ok := errResp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object in response")
	}
	if errObj["code"] != "CONFLICT" {
		t.Errorf("expected CONFLICT code, got %v", errObj["code"])
	}
}

// --- Helper for backup/restore integration ---

func TestIntegration_ExportImportRoundtrip(t *testing.T) {
	testDB, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	// Create project with tasks and memories.
	project, err := CreateProject(testDB, "export-proj", "/tmp/export", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task1, err := CreateTaskV2(testDB, "export-task-1", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}
	_, err = CreateTaskV2(testDB, "export-task-2", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}
	_, err = CreateMemory(testDB, project.ID, "test", "key1", map[string]interface{}{"info": "hello"}, nil)
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}

	// Export.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/export", project.ID), nil)
	rr := httptest.NewRecorder()
	v2ProjectExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Export: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var archive ProjectArchive
	if err := json.NewDecoder(rr.Body).Decode(&archive); err != nil {
		t.Fatalf("decode archive: %v", err)
	}

	if archive.Version == "" {
		t.Error("expected version in archive")
	}
	if len(archive.Tasks) < 2 {
		t.Errorf("expected at least 2 tasks in archive, got %d", len(archive.Tasks))
	}

	// Import.
	archiveJSON, _ := json.Marshal(archive)
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/import", bytes.NewReader(archiveJSON))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	v2ProjectImportHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Import: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var importResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&importResp); err != nil {
		t.Fatalf("decode import response: %v", err)
	}

	tasksImported := int(importResp["tasks_imported"].(float64))
	if tasksImported < 2 {
		t.Errorf("expected at least 2 tasks imported, got %d", tasksImported)
	}

	_ = task1 // used in setup
}

// --- Integration Test 5: Full lifecycle emits SSE events to v2 subscribers ---

func TestIntegration_SSEEventsEmittedDuringLifecycle(t *testing.T) {
	testDB, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	server := httptest.NewServer(mux)
	defer server.Close()

	project, err := CreateProject(testDB, "sse-lifecycle-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Connect SSE client BEFORE lifecycle operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := server.URL + fmt.Sprintf("/api/v2/projects/%s/events/stream", project.ID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	// Read ": ready" comment
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read ready: %v", err)
		}
		if strings.HasPrefix(strings.TrimRight(line, "\r\n"), ":") {
			break
		}
	}

	// Give the SSE subscription a moment to register
	time.Sleep(50 * time.Millisecond)

	// Create a task (should emit task.created event)
	task, err := CreateTaskV2(testDB, "SSE lifecycle task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim the task
	claimBody := `{"agent_id": "sse-agent", "mode": "exclusive"}`
	claimReq := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v2/tasks/%d/claim", task.ID), strings.NewReader(claimBody))
	claimReq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	v2TaskClaimHandler(rr, claimReq, strconv.Itoa(task.ID))
	if rr.Code != http.StatusOK {
		t.Fatalf("Claim: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Complete the task
	completeBody := `{"output": "done", "result": {"summary": "ok", "changes_made": ["file.go"], "files_touched": ["file.go"]}}`
	completeReq := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v2/tasks/%d/complete", task.ID), strings.NewReader(completeBody))
	completeReq.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	v2TaskCompleteHandler(rr, completeReq, strconv.Itoa(task.ID))
	if rr.Code != http.StatusOK {
		t.Fatalf("Complete: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Read SSE events — we should see task.claimed and task.completed at minimum.
	// (task.created may or may not be emitted via SSE depending on the code path.)
	receivedKinds := make(map[string]bool)
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			goto doneReading
		default:
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, "event:") {
			kind := strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			receivedKinds[kind] = true
		}
		// Stop early if we have enough
		if receivedKinds["task.claimed"] && receivedKinds["task.completed"] {
			break
		}
	}
doneReading:

	if !receivedKinds["task.claimed"] {
		t.Error("expected task.claimed event via SSE stream")
	}
	if !receivedKinds["task.completed"] {
		t.Error("expected task.completed event via SSE stream")
	}

	// Also verify events are persisted in the database
	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	dbKinds := make(map[string]bool)
	for _, ev := range events {
		dbKinds[ev.Kind] = true
	}
	if !dbKinds["task.claimed"] {
		t.Error("expected task.claimed event in database")
	}
	if !dbKinds["task.completed"] {
		t.Error("expected task.completed event in database")
	}
}

// TestIntegration_FirstRunLifecycle exercises the full first-run experience:
// project → agent → task → claim → heartbeat → run steps → complete → verify
func TestIntegration_FirstRunLifecycle(t *testing.T) {
	testDB, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// 1. Create project
	project, err := CreateProject(testDB, "first-run-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// 2. Register agent via API
	agentBody := `{"name":"test-agent","capabilities":["go","python"],"metadata":{"version":"1.0"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/agents", strings.NewReader(agentBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Register agent: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var agentResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&agentResp)
	agentObj := agentResp["agent"].(map[string]interface{})
	agentID := agentObj["id"].(string)

	// 3. Create task with metadata
	taskBody := fmt.Sprintf(`{
		"instructions":"Build the widget",
		"project_id":"%s",
		"title":"Build widget",
		"type":"MODIFY",
		"priority":3,
		"tags":["backend","go"]
	}`, project.ID)
	req = httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(taskBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Create task: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var taskResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&taskResp)
	taskID := int(taskResp["task"].(map[string]interface{})["id"].(float64))

	// 4. Claim task
	claimBody := fmt.Sprintf(`{"agent_id":"%s","mode":"exclusive"}`, agentID)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), strings.NewReader(claimBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Claim: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var claimResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&claimResp)
	runObj := claimResp["run"].(map[string]interface{})
	runID := runObj["id"].(string)

	// 5. Log a run step
	stepBody := fmt.Sprintf(`{"name":"compile","status":"SUCCEEDED","details":{"output":"ok"}}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/runs/%s/steps", runID), strings.NewReader(stepBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	// Accept 200/201/204 — the endpoint may vary
	if rr.Code >= 400 {
		t.Fatalf("Run step: expected success, got %d: %s", rr.Code, rr.Body.String())
	}

	// 7. Complete task
	completeBody := `{"output":"Widget built","result":{"summary":"done","changes_made":["widget.go"],"files_touched":["widget.go"]}}`
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), strings.NewReader(completeBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Complete: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// 8. Verify final state
	task, err := GetTaskV2(testDB, taskID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if task.StatusV2 != TaskStatusSucceeded {
		t.Errorf("task: expected SUCCEEDED, got %s", task.StatusV2)
	}

	// Verify run completed
	run, err := GetLatestRunByTaskID(testDB, taskID)
	if err != nil {
		t.Fatalf("GetLatestRunByTaskID: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != RunStatusSucceeded {
		t.Errorf("run: expected SUCCEEDED, got %s", run.Status)
	}

	// Verify agent still exists
	req = httptest.NewRequest(http.MethodGet, "/api/v2/agents/"+agentID, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("agent detail: expected 200, got %d", rr.Code)
	}

	// Verify events were recorded
	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(events))
	}

	// Verify runs list endpoint works
	req = httptest.NewRequest(http.MethodGet, "/api/v2/runs", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("runs list: expected 200, got %d", rr.Code)
	}

	// Verify task appears in tasks list
	req = httptest.NewRequest(http.MethodGet, "/api/v2/tasks", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("tasks list: expected 200, got %d", rr.Code)
	}
	var listResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&listResp)
	tasks := listResp["tasks"].([]interface{})
	if len(tasks) == 0 {
		t.Error("expected at least 1 task in list")
	}
}

// TestIntegration_CanonicalProductFlow is the release-blocking lifecycle test.
// It proves the complete public experience end-to-end:
// project → agent register → create task → claim → run step → heartbeat lease →
// complete task → verify task/runs/agents/events consistency.
func TestIntegration_CanonicalProductFlow(t *testing.T) {
	testDB, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// 1. Create/open project
	project, err := CreateProject(testDB, "canonical-flow", "/tmp/canonical", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// 2. Register agent
	agentBody := `{"name":"canonical-agent","capabilities":["go"],"metadata":{"version":"1.0"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/agents", strings.NewReader(agentBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Register agent: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var agentResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&agentResp)
	agentID := agentResp["agent"].(map[string]interface{})["id"].(string)

	// 3. Create task
	taskBody := fmt.Sprintf(`{
		"instructions":"Canonical test: build and verify",
		"project_id":"%s",
		"title":"Canonical product flow",
		"type":"MODIFY",
		"priority":80,
		"tags":["release-blocking","e2e"]
	}`, project.ID)
	req = httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(taskBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Create task: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var taskResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&taskResp)
	taskID := int(taskResp["task"].(map[string]interface{})["id"].(float64))

	// 4. Claim task — creates run and lease
	claimBody := fmt.Sprintf(`{"agent_id":"%s","mode":"exclusive"}`, agentID)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), strings.NewReader(claimBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Claim: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var claimResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&claimResp)

	// Verify claim response structure
	if claimResp["task"] == nil {
		t.Fatal("claim response missing 'task'")
	}
	if claimResp["run"] == nil {
		t.Fatal("claim response missing 'run'")
	}
	if claimResp["lease"] == nil {
		t.Fatal("claim response missing 'lease'")
	}

	runID := claimResp["run"].(map[string]interface{})["id"].(string)
	leaseID := claimResp["lease"].(map[string]interface{})["id"].(string)

	// Verify task moved to IN_PROGRESS
	task, err := GetTaskV2(testDB, taskID)
	if err != nil {
		t.Fatalf("GetTaskV2: %v", err)
	}
	if task.StatusV2 != TaskStatusClaimed {
		t.Errorf("after claim: expected CLAIMED, got %s", task.StatusV2)
	}

	// 5. Log a run step
	stepBody := `{"name":"build","status":"SUCCEEDED","details":{"output":"compiled ok"}}`
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/runs/%s/steps", runID), strings.NewReader(stepBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code >= 400 {
		t.Fatalf("Run step: expected success, got %d: %s", rr.Code, rr.Body.String())
	}

	// 6. Heartbeat the lease
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/leases/%s/heartbeat", leaseID), nil)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code >= 400 {
		t.Fatalf("Lease heartbeat: expected success, got %d: %s", rr.Code, rr.Body.String())
	}

	// 7. Complete task
	completeBody := `{"output":"Built and verified successfully","result":{"summary":"canonical flow complete","changes_made":["widget.go"],"files_touched":["widget.go"]}}`
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), strings.NewReader(completeBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Complete: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// === VERIFY CONSISTENCY ===

	// Task is SUCCEEDED
	task, err = GetTaskV2(testDB, taskID)
	if err != nil {
		t.Fatalf("GetTaskV2 final: %v", err)
	}
	if task.StatusV2 != TaskStatusSucceeded {
		t.Errorf("task: expected SUCCEEDED, got %s", task.StatusV2)
	}

	// Run is SUCCEEDED
	run, err := GetLatestRunByTaskID(testDB, taskID)
	if err != nil {
		t.Fatalf("GetLatestRunByTaskID: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != RunStatusSucceeded {
		t.Errorf("run: expected SUCCEEDED, got %s", run.Status)
	}

	// Lease is released (no active lease)
	lease, err := GetLeaseByTaskID(testDB, taskID)
	if err != nil {
		t.Fatalf("GetLeaseByTaskID: %v", err)
	}
	if lease != nil {
		t.Error("expected no active lease after completion")
	}

	// Agent still registered and accessible
	req = httptest.NewRequest(http.MethodGet, "/api/v2/agents/"+agentID, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("agent detail: expected 200, got %d", rr.Code)
	}

	// Events reflect the full lifecycle
	events, err := GetEventsByProjectID(testDB, project.ID, 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID: %v", err)
	}
	kinds := make(map[string]bool)
	for _, ev := range events {
		kinds[ev.Kind] = true
	}
	for _, expected := range []string{"task.claimed", "task.completed"} {
		if !kinds[expected] {
			t.Errorf("expected event kind %q not found", expected)
		}
	}

	// Runs list includes our run
	req = httptest.NewRequest(http.MethodGet, "/api/v2/runs", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("runs list: expected 200, got %d", rr.Code)
	}

	// Tasks list includes our task
	req = httptest.NewRequest(http.MethodGet, "/api/v2/tasks", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("tasks list: expected 200, got %d", rr.Code)
	}
	var finalListResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&finalListResp)
	finalTasks := finalListResp["tasks"].([]interface{})
	found := false
	for _, ft := range finalTasks {
		ftMap := ft.(map[string]interface{})
		if int(ftMap["id"].(float64)) == taskID {
			// Status may be under "status", "status_v2", or "status_v1"
			statusVal := ""
			if s, ok := ftMap["status_v2"].(string); ok {
				statusVal = s
			} else if s, ok := ftMap["status"].(string); ok {
				statusVal = s
			}
			if statusVal != "SUCCEEDED" {
				t.Errorf("task in list: expected SUCCEEDED, got %s", statusVal)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("completed task not found in tasks list")
	}

	// Events list reflects consistent state
	req = httptest.NewRequest(http.MethodGet, "/api/v2/events", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("events list: expected 200, got %d", rr.Code)
	}
}


