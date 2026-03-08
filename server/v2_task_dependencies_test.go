package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func findEventByKind(events []Event, kind string) *Event {
	for i := range events {
		if events[i].Kind == kind {
			return &events[i]
		}
	}
	return nil
}

func TestV2TaskDependenciesCreateAndList(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	resultA, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task A", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task A: %v", err)
	}
	taskID, err := resultA.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task A id: %v", err)
	}

	resultB, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task B", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task B: %v", err)
	}
	dependsID, err := resultB.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task B id: %v", err)
	}

	payload := map[string]interface{}{"depends_on_task_id": dependsID}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	dep, ok := createResp["dependency"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected dependency object, got %T", createResp["dependency"])
	}
	if int(dep["task_id"].(float64)) != int(taskID) {
		t.Fatalf("expected task_id %d, got %v", taskID, dep["task_id"])
	}
	if int(dep["depends_on_task_id"].(float64)) != int(dependsID) {
		t.Fatalf("expected depends_on_task_id %d, got %v", dependsID, dep["depends_on_task_id"])
	}

	events, err := GetEventsByProjectID(testDB, "proj_default", 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}
	createdEvent := findEventByKind(events, "task.dependency.created")
	if createdEvent == nil {
		t.Fatal("expected task.dependency.created event")
	}
	if createdEvent.EntityType != "task_dependency" {
		t.Fatalf("expected entity_type task_dependency, got %s", createdEvent.EntityType)
	}
	if createdEvent.EntityID != fmt.Sprintf("%d:%d", taskID, dependsID) {
		t.Fatalf("expected entity_id %d:%d, got %s", taskID, dependsID, createdEvent.EntityID)
	}
	if int(createdEvent.Payload["task_id"].(float64)) != int(taskID) {
		t.Fatalf("expected event task_id %d, got %v", taskID, createdEvent.Payload["task_id"])
	}
	if int(createdEvent.Payload["depends_on_id"].(float64)) != int(dependsID) {
		t.Fatalf("expected event depends_on_id %d, got %v", dependsID, createdEvent.Payload["depends_on_id"])
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskID), nil)
	listW := httptest.NewRecorder()
	mux.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", listW.Code, listW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	deps, ok := listResp["dependencies"].([]interface{})
	if !ok {
		t.Fatalf("expected dependencies array, got %T", listResp["dependencies"])
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
}

func TestV2TaskDependenciesValidation(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task A", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	selfPayload := map[string]interface{}{"depends_on_task_id": taskID}
	selfBody, _ := json.Marshal(selfPayload)
	selfReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskID), bytes.NewReader(selfBody))
	selfReq.Header.Set("Content-Type", "application/json")
	selfW := httptest.NewRecorder()
	mux.ServeHTTP(selfW, selfReq)

	if selfW.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", selfW.Code, selfW.Body.String())
	}
	assertV2ErrorEnvelope(t, selfW, "INVALID_ARGUMENT")

	missingPayload := map[string]interface{}{"depends_on_task_id": 999999}
	missingBody, _ := json.Marshal(missingPayload)
	missingReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskID), bytes.NewReader(missingBody))
	missingReq.Header.Set("Content-Type", "application/json")
	missingW := httptest.NewRecorder()
	mux.ServeHTTP(missingW, missingReq)

	if missingW.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", missingW.Code, missingW.Body.String())
	}
	assertV2ErrorEnvelope(t, missingW, "NOT_FOUND")
}

func TestV2TaskDependenciesDuplicateConflict(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	resultA, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task A", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task A: %v", err)
	}
	taskID, err := resultA.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task A id: %v", err)
	}

	resultB, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task B", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task B: %v", err)
	}
	dependsID, err := resultB.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task B id: %v", err)
	}

	payload := map[string]interface{}{"depends_on_task_id": dependsID}
	body, _ := json.Marshal(payload)
	path := fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskID)

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}

	dupeReq := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	dupeReq.Header.Set("Content-Type", "application/json")
	dupeW := httptest.NewRecorder()
	mux.ServeHTTP(dupeW, dupeReq)

	if dupeW.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", dupeW.Code, dupeW.Body.String())
	}
	assertV2ErrorEnvelope(t, dupeW, "CONFLICT")
}

func TestV2TaskDependenciesCycleConflict(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	resultA, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task A", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task A: %v", err)
	}
	resultB, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task B", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task B: %v", err)
	}
	resultC, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task C", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task C: %v", err)
	}

	taskAID, err := resultA.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task A id: %v", err)
	}
	taskBID, err := resultB.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task B id: %v", err)
	}
	taskCID, err := resultC.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task C id: %v", err)
	}

	chain := []struct{
		taskID     int64
		dependsID  int64
	}{
		{taskID: taskAID, dependsID: taskBID},
		{taskID: taskBID, dependsID: taskCID},
	}
	for _, step := range chain {
		payload := map[string]interface{}{"depends_on_task_id": step.dependsID}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dependencies", step.taskID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
		}
	}

	cyclePayload := map[string]interface{}{"depends_on_task_id": taskAID}
	cycleBody, _ := json.Marshal(cyclePayload)
	cycleReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskCID), bytes.NewReader(cycleBody))
	cycleReq.Header.Set("Content-Type", "application/json")
	cycleW := httptest.NewRecorder()
	mux.ServeHTTP(cycleW, cycleReq)

	if cycleW.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", cycleW.Code, cycleW.Body.String())
	}
	assertV2ErrorEnvelope(t, cycleW, "CONFLICT")
}

func TestV2TaskDependenciesDeleteSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	resultA, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task A", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task A: %v", err)
	}
	taskID, err := resultA.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task A id: %v", err)
	}

	resultB, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task B", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task B: %v", err)
	}
	dependsID, err := resultB.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task B id: %v", err)
	}

	payload := map[string]interface{}{"depends_on_task_id": dependsID}
	body, _ := json.Marshal(payload)
	createReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskID), bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", createW.Code, createW.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/tasks/%d/dependencies/%d", taskID, dependsID), nil)
	deleteW := httptest.NewRecorder()
	mux.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", deleteW.Code, deleteW.Body.String())
	}

	var deleteResp map[string]interface{}
	if err := json.NewDecoder(deleteW.Body).Decode(&deleteResp); err != nil {
		t.Fatalf("failed to decode delete response: %v", err)
	}
	dep, ok := deleteResp["dependency"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected dependency object, got %T", deleteResp["dependency"])
	}
	if int(dep["task_id"].(float64)) != int(taskID) {
		t.Fatalf("expected task_id %d, got %v", taskID, dep["task_id"])
	}
	if int(dep["depends_on_task_id"].(float64)) != int(dependsID) {
		t.Fatalf("expected depends_on_task_id %d, got %v", dependsID, dep["depends_on_task_id"])
	}

	events, err := GetEventsByProjectID(testDB, "proj_default", 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}
	deletedEvent := findEventByKind(events, "task.dependency.deleted")
	if deletedEvent == nil {
		t.Fatal("expected task.dependency.deleted event")
	}
	if deletedEvent.EntityType != "task_dependency" {
		t.Fatalf("expected entity_type task_dependency, got %s", deletedEvent.EntityType)
	}
	if deletedEvent.EntityID != fmt.Sprintf("%d:%d", taskID, dependsID) {
		t.Fatalf("expected entity_id %d:%d, got %s", taskID, dependsID, deletedEvent.EntityID)
	}
	if int(deletedEvent.Payload["task_id"].(float64)) != int(taskID) {
		t.Fatalf("expected event task_id %d, got %v", taskID, deletedEvent.Payload["task_id"])
	}
	if int(deletedEvent.Payload["depends_on_id"].(float64)) != int(dependsID) {
		t.Fatalf("expected event depends_on_id %d, got %v", dependsID, deletedEvent.Payload["depends_on_id"])
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d/dependencies", taskID), nil)
	listW := httptest.NewRecorder()
	mux.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", listW.Code, listW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	deps, ok := listResp["dependencies"].([]interface{})
	if !ok {
		t.Fatalf("expected dependencies array, got %T", listResp["dependencies"])
	}
	if len(deps) != 0 {
		t.Fatalf("expected 0 dependencies, got %d", len(deps))
	}
}

func TestV2TaskDependenciesDeleteNotFound(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	resultA, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task A", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task A: %v", err)
	}
	taskID, err := resultA.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task A id: %v", err)
	}

	resultB, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Task B", StatusNotPicked, TaskStatusQueued, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task B: %v", err)
	}
	dependsID, err := resultB.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task B id: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/tasks/%d/dependencies/%d", taskID, dependsID), nil)
	deleteW := httptest.NewRecorder()
	mux.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", deleteW.Code, deleteW.Body.String())
	}
	assertV2ErrorEnvelope(t, deleteW, "NOT_FOUND")
}

func TestV2TaskDependenciesDeleteMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPut, "/api/v2/tasks/123/dependencies/456", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}

func TestV2TaskDependenciesMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPut, "/api/v2/tasks/123/dependencies", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
