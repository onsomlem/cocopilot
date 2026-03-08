package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2TasksListSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	_, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, created_at) VALUES (?, ?, ?, ?, ?)",
		"First task", StatusNotPicked, TaskStatusQueued, "proj_default", "2026-02-11T10:00:00.000Z",
	)
	if err != nil {
		t.Fatalf("failed to insert first task: %v", err)
	}
	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, created_at) VALUES (?, ?, ?, ?, ?)",
		"Second task", StatusNotPicked, TaskStatusQueued, "proj_default", "2026-02-11T10:01:00.000Z",
	)
	if err != nil {
		t.Fatalf("failed to insert second task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/tasks", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tasks, ok := resp["tasks"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks array, got %T", resp["tasks"])
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if total, ok := resp["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("expected total 2, got %v", resp["total"])
	}

	firstTask, ok := tasks[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first task object, got %T", tasks[0])
	}
	secondTask, ok := tasks[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected second task object, got %T", tasks[1])
	}

	if firstTask["instructions"] != "First task" {
		t.Fatalf("expected first task instructions 'First task', got %v", firstTask["instructions"])
	}
	if secondTask["instructions"] != "Second task" {
		t.Fatalf("expected second task instructions 'Second task', got %v", secondTask["instructions"])
	}
	assertTaskUpdatedAtPresent(t, firstTask)
	assertTaskUpdatedAtPresent(t, secondTask)
}

func TestV2TasksListFilter(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Filter Project", "/tmp/filter", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Filtered task", StatusInProgress, TaskStatusRunning, project.ID,
	)
	if err != nil {
		t.Fatalf("failed to insert filtered task: %v", err)
	}
	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Different status", StatusNotPicked, TaskStatusQueued, project.ID,
	)
	if err != nil {
		t.Fatalf("failed to insert queued task: %v", err)
	}
	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Other project", StatusInProgress, TaskStatusRunning, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert other project task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/tasks?project_id="+project.ID+"&status=RUNNING", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tasks, ok := resp["tasks"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks array, got %T", resp["tasks"])
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if total, ok := resp["total"].(float64); !ok || int(total) != 1 {
		t.Fatalf("expected total 1, got %v", resp["total"])
	}

	firstTask, ok := tasks[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %T", tasks[0])
	}
	if firstTask["instructions"] != "Filtered task" {
		t.Fatalf("expected filtered task, got %v", firstTask["instructions"])
	}
}

func TestV2TasksListTypeTagQueryFilters(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	backendTags, err := marshalJSON([]string{"backend", "api"})
	if err != nil {
		t.Fatalf("failed to marshal tags: %v", err)
	}
	frontendTags, err := marshalJSON([]string{"frontend"})
	if err != nil {
		t.Fatalf("failed to marshal tags: %v", err)
	}

	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type, tags_json, title) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Investigate memory leak", StatusNotPicked, TaskStatusQueued, "proj_default", string(TaskTypeAnalyze), backendTags, "Memory leak",
	)
	if err != nil {
		t.Fatalf("failed to insert analyze task: %v", err)
	}
	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type, tags_json, title) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Add button", StatusNotPicked, TaskStatusQueued, "proj_default", string(TaskTypeModify), frontendTags, "UI update",
	)
	if err != nil {
		t.Fatalf("failed to insert modify task: %v", err)
	}

	assertSingleInstruction := func(path string, expected string) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		tasks, ok := resp["tasks"].([]interface{})
		if !ok {
			t.Fatalf("expected tasks array, got %T", resp["tasks"])
		}
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		task := tasks[0].(map[string]interface{})
		if task["instructions"] != expected {
			t.Fatalf("expected instructions %q, got %v", expected, task["instructions"])
		}
	}

	assertSingleInstruction("/api/v2/tasks?type=ANALYZE", "Investigate memory leak")
	assertSingleInstruction("/api/v2/tasks?tag=frontend", "Add button")
	assertSingleInstruction("/api/v2/tasks?q=memory", "Investigate memory leak")
}

func TestV2TasksListPaging(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	_, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, created_at) VALUES (?, ?, ?, ?, ?)",
		"First task", StatusNotPicked, TaskStatusQueued, "proj_default", "2026-02-11T10:00:00.000Z",
	)
	if err != nil {
		t.Fatalf("failed to insert first task: %v", err)
	}
	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, created_at) VALUES (?, ?, ?, ?, ?)",
		"Second task", StatusNotPicked, TaskStatusQueued, "proj_default", "2026-02-11T10:01:00.000Z",
	)
	if err != nil {
		t.Fatalf("failed to insert second task: %v", err)
	}
	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, created_at) VALUES (?, ?, ?, ?, ?)",
		"Third task", StatusNotPicked, TaskStatusQueued, "proj_default", "2026-02-11T10:02:00.000Z",
	)
	if err != nil {
		t.Fatalf("failed to insert third task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/tasks?limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tasks, ok := resp["tasks"].([]interface{})
	if !ok {
		t.Fatalf("expected tasks array, got %T", resp["tasks"])
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if total, ok := resp["total"].(float64); !ok || int(total) != 3 {
		t.Fatalf("expected total 3, got %v", resp["total"])
	}

	task, ok := tasks[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %T", tasks[0])
	}
	if task["instructions"] != "Second task" {
		t.Fatalf("expected second task, got %v", task["instructions"])
	}
}

func TestV2TasksListSorting(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	_, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"Older created, newer updated", StatusNotPicked, TaskStatusQueued, "proj_default", "2026-02-11T10:00:00.000Z", "2026-02-11T10:05:00.000Z",
	)
	if err != nil {
		t.Fatalf("failed to insert first task: %v", err)
	}
	_, err = testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		"Newer created, older updated", StatusNotPicked, TaskStatusQueued, "proj_default", "2026-02-11T10:01:00.000Z", "2026-02-11T10:02:00.000Z",
	)
	if err != nil {
		t.Fatalf("failed to insert second task: %v", err)
	}

	getInstructions := func(path string) []string {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		tasks, ok := resp["tasks"].([]interface{})
		if !ok {
			t.Fatalf("expected tasks array, got %T", resp["tasks"])
		}

		instructions := make([]string, 0, len(tasks))
		for _, task := range tasks {
			item := task.(map[string]interface{})
			instructions = append(instructions, item["instructions"].(string))
		}
		return instructions
	}

	createdDesc := getInstructions("/api/v2/tasks?sort=created_at:desc")
	if len(createdDesc) < 2 || createdDesc[0] != "Newer created, older updated" {
		t.Fatalf("expected created_at desc ordering, got %v", createdDesc)
	}

	updatedDesc := getInstructions("/api/v2/tasks?sort=updated_at:desc")
	if len(updatedDesc) < 2 || updatedDesc[0] != "Older created, newer updated" {
		t.Fatalf("expected updated_at desc ordering, got %v", updatedDesc)
	}
}

func TestV2TasksListInvalidPagingParams(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	tests := []string{
		"/api/v2/tasks?limit=-1",
		"/api/v2/tasks?limit=abc",
		"/api/v2/tasks?offset=-2",
		"/api/v2/tasks?offset=xyz",
	}

	for _, path := range tests {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	}
}

func TestV2TasksListInvalidSortParam(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/tasks?sort=priority:asc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2TasksListInvalidFilterParams(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	paths := []string{
		"/api/v2/tasks?type=UNKNOWN",
		"/api/v2/tasks?tag=",
		"/api/v2/tasks?q=",
	}

	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	}
}

func TestV2TasksListMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v2/tasks", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
