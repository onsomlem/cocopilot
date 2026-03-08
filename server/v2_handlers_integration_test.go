package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------- claim-next ----------

func TestClaimNextTask_Success(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "claim-next-proj", "/tmp", nil)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	task1, err := CreateTaskV2(db, "task one", proj.ID, nil)
	if err != nil {
		t.Fatalf("create task1: %v", err)
	}
	task2, err := CreateTaskV2(db, "task two", proj.ID, nil)
	if err != nil {
		t.Fatalf("create task2: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"agent_id": "agent-cn-1", "mode": "exclusive"})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v2/projects/"+proj.ID+"/tasks/claim-next",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	taskObj, ok := resp["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object in response, got %T", resp["task"])
	}
	claimedID := int(taskObj["id"].(float64))
	if claimedID != task1.ID && claimedID != task2.ID {
		t.Fatalf("claimed task ID %d not one of [%d, %d]", claimedID, task1.ID, task2.ID)
	}

	if _, ok := resp["lease"]; !ok {
		t.Fatal("expected lease in response")
	}
	if _, ok := resp["run"]; !ok {
		t.Fatal("expected run in response")
	}
}

func TestClaimNextTask_EmptyProject(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "empty-proj", "/tmp", nil)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"agent_id": "agent-empty", "mode": "exclusive"})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v2/projects/"+proj.ID+"/tasks/claim-next",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// The idle planner may spawn and auto-claim a task (200), or
	// return 204 No Content if no idle planner fires. Both are valid.
	switch w.Code {
	case http.StatusNoContent:
		// No task available and idle planner did not spawn — OK
	case http.StatusOK:
		// Idle planner spawned and was claimed
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		taskObj, ok := resp["task"].(map[string]interface{})
		if !ok {
			t.Fatal("expected task object in idle planner response")
		}
		if taskObj["title"] != "Idle Planner" {
			t.Fatalf("expected Idle Planner task, got title=%v", taskObj["title"])
		}
	case http.StatusAccepted:
		// Idle planner spawned but couldn't be claimed — OK
	default:
		t.Fatalf("expected 200, 202, or 204, got %d body=%s", w.Code, w.Body.String())
	}
}

// ---------- metrics ----------

func TestMetrics_ReturnsData(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "metrics-proj", "/tmp", nil)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := CreateTaskV2(db, "metric task", proj.ID, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/metrics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	totals, ok := resp["totals"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected totals map, got %T", resp["totals"])
	}
	if totals["tasks"].(float64) < 1 {
		t.Fatal("expected at least 1 task in totals")
	}
	if totals["projects"].(float64) < 1 {
		t.Fatal("expected at least 1 project in totals")
	}

	byStatus, ok := resp["tasks_by_status"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tasks_by_status map, got %T", resp["tasks_by_status"])
	}
	if _, exists := byStatus["queued"]; !exists {
		t.Fatal("expected queued key in tasks_by_status")
	}

	if _, exists := resp["schema_version"]; !exists {
		t.Fatal("expected schema_version in response")
	}
}

// ---------- status ----------

func TestStatus_ReturnsVersion(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if _, ok := resp["version"]; !ok {
		t.Fatal("expected version in response")
	}
	if _, ok := resp["uptime_seconds"]; !ok {
		t.Fatal("expected uptime_seconds in response")
	}
	if _, ok := resp["active_projects"]; !ok {
		t.Fatal("expected active_projects in response")
	}
	if _, ok := resp["schema_version"]; !ok {
		t.Fatal("expected schema_version in response")
	}
}

// ---------- audit/export ----------

func TestAuditExport_ReturnsEvents(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "audit-proj", "/tmp", nil)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Seed a couple of events
	if _, err := CreateEvent(db, proj.ID, "task.created", "task", "1", map[string]interface{}{"title": "t1"}); err != nil {
		t.Fatalf("create event: %v", err)
	}
	if _, err := CreateEvent(db, proj.ID, "task.completed", "task", "1", map[string]interface{}{"result": "ok"}); err != nil {
		t.Fatalf("create event: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/v2/projects/"+proj.ID+"/audit/export?format=json",
		nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp["project_id"] != proj.ID {
		t.Fatalf("expected project_id %s, got %v", proj.ID, resp["project_id"])
	}
	total := int(resp["total"].(float64))
	if total < 2 {
		t.Fatalf("expected total >= 2, got %d", total)
	}
	events, ok := resp["events"].([]interface{})
	if !ok || len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// Verify event structure
	ev0 := events[0].(map[string]interface{})
	if _, ok := ev0["id"]; !ok {
		t.Fatal("expected id field in event")
	}
	if _, ok := ev0["kind"]; !ok {
		t.Fatal("expected kind field in event")
	}
}

func TestAuditExport_CSV(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "audit-csv-proj", "/tmp", nil)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	if _, err := CreateEvent(db, proj.ID, "task.created", "task", "1", nil); err != nil {
		t.Fatalf("create event: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/v2/projects/"+proj.ID+"/audit/export?format=csv",
		nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct == "" || ct != "text/csv; charset=utf-8" {
		t.Fatalf("expected text/csv content-type, got %q", ct)
	}
}

func TestAuditExport_NotFound(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet,
		"/api/v2/projects/nonexistent/audit/export",
		nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}
