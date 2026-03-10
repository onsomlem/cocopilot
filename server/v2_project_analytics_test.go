package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2ProjectAutomationStats(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "stats-test", "", nil)

	url := "/api/v2/projects/" + proj.ID + "/automation-stats"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	v2ProjectAutomationStatsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("automation stats returned %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["project_id"].(string) != proj.ID {
		t.Errorf("expected project_id %s, got %s", proj.ID, resp["project_id"])
	}
}

func TestV2ProjectGraphTasks(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "graph-test", "", nil)
	// Create some tasks
	CreateTaskV2(db, "task 1", proj.ID, nil)
	CreateTaskV2(db, "task 2", proj.ID, nil)

	url := "/api/v2/projects/" + proj.ID + "/graphs/tasks"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	v2ProjectGraphTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("graph tasks returned %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	nodes := resp["nodes"].([]interface{})
	if len(nodes) < 2 {
		t.Errorf("expected at least 2 nodes, got %d", len(nodes))
	}
}

func TestV2ProjectIDESignals(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "ide-test", "", nil)

	url := "/api/v2/projects/" + proj.ID + "/ide-signals"
	body := `{"kind":"file_saved","data":{"file":"main.go"}}`
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectIDESignalsHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("IDE signals returned %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["kind"].(string) != "file_saved" {
		t.Errorf("expected kind 'file_saved', got %s", resp["kind"])
	}
	if resp["project_id"].(string) != proj.ID {
		t.Errorf("expected project_id %s", proj.ID)
	}
}

func TestV2ProjectIDESignals_MissingKind(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "ide-test2", "", nil)

	url := "/api/v2/projects/" + proj.ID + "/ide-signals"
	body := `{"data":{"file":"main.go"}}`
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectIDESignalsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing kind, got %d", w.Code)
	}
}

func TestV2AuditHandler(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "audit-test", "", nil)

	// Create some audit events
	CreateEvent(db, proj.ID, "audit.task_created", "task", "1", map[string]interface{}{"title": "t1"})
	CreateEvent(db, proj.ID, "audit.task_updated", "task", "1", map[string]interface{}{"title": "t1"})

	url := "/api/v2/audit?project_id=" + proj.ID
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	v2AuditHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("audit query returned %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	events := resp["events"].([]interface{})
	if len(events) < 2 {
		t.Errorf("expected at least 2 audit events, got %d", len(events))
	}
}

func TestV2AuditHandler_WithTypeFilter(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "audit-filter", "", nil)
	CreateEvent(db, proj.ID, "audit.task_created", "task", "1", nil)
	CreateEvent(db, proj.ID, "audit.task_updated", "task", "2", nil)
	CreateEvent(db, proj.ID, "task.completed", "task", "3", nil)

	url := "/api/v2/audit?project_id=" + proj.ID + "&type=audit.task_created"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	v2AuditHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("audit with type filter returned %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	events := resp["events"].([]interface{})
	if len(events) != 1 {
		t.Errorf("expected 1 audit event with type filter, got %d", len(events))
	}
}

func TestV2AuditHandler_Pagination(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "audit-page", "", nil)
	for i := 0; i < 5; i++ {
		CreateEvent(db, proj.ID, "audit.test", "task", "1", nil)
	}

	url := "/api/v2/audit?project_id=" + proj.ID + "&limit=2&offset=0"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	v2AuditHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("audit pagination returned %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	events := resp["events"].([]interface{})
	if len(events) != 2 {
		t.Errorf("expected 2 events with limit=2, got %d", len(events))
	}
}
