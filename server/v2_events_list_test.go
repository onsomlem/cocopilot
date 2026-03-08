package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func insertTestEvent(t *testing.T, db *sql.DB, id, projectID, kind, entityType, entityID, createdAt string) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO events (id, project_id, kind, entity_type, entity_id, created_at, payload_json) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id,
		projectID,
		kind,
		entityType,
		entityID,
		createdAt,
		"{}",
	)
	if err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}
}

func TestV2EventsListSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	insertTestEvent(t, testDB, "evt_1", "proj_default", "task.created", "task", "101", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_2", "proj_default", "task.updated", "task", "101", "2026-02-11T10:01:00.000000Z")

	req := httptest.NewRequest(http.MethodGet, "/api/v2/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatalf("expected events array, got %T", resp["events"])
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if resp["total"] == nil {
		t.Fatalf("expected total in response")
	}
	if total, ok := resp["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("expected total 2, got %v", resp["total"])
	}

	firstEvent, ok := events[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected event object, got %T", events[0])
	}
	if firstEvent["kind"] != "task.updated" {
		t.Fatalf("expected newest event kind task.updated, got %v", firstEvent["kind"])
	}
}

func TestV2EventsListFilter(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	insertTestEvent(t, testDB, "evt_10", "proj_default", "task.created", "task", "201", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_11", "proj_default", "task.created", "task", "201", "2026-02-11T10:02:00.000000Z")
	insertTestEvent(t, testDB, "evt_12", "proj_default", "task.updated", "task", "201", "2026-02-11T10:03:00.000000Z")

	path := "/api/v2/events?task_id=201&type=task.created&since=2026-02-11T10:01:00.000000Z&limit=1"

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

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatalf("expected events array, got %T", resp["events"])
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if total, ok := resp["total"].(float64); !ok || int(total) != 1 {
		t.Fatalf("expected total 1, got %v", resp["total"])
	}

	firstEvent, ok := events[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected event object, got %T", events[0])
	}
	if firstEvent["kind"] != "task.created" {
		t.Fatalf("expected kind task.created, got %v", firstEvent["kind"])
	}
}

func TestV2EventsListTaskIDFilter(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	insertTestEvent(t, testDB, "evt_30", "proj_default", "task.created", "task", "301", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_31", "proj_default", "task.updated", "task", "301", "2026-02-11T10:01:00.000000Z")
	insertTestEvent(t, testDB, "evt_32", "proj_default", "task.updated", "task", "302", "2026-02-11T10:02:00.000000Z")
	insertTestEvent(t, testDB, "evt_33", "proj_default", "lease.created", "lease", "lease_1", "2026-02-11T10:03:00.000000Z")

	req := httptest.NewRequest(http.MethodGet, "/api/v2/events?task_id=301", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatalf("expected events array, got %T", resp["events"])
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if total, ok := resp["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("expected total 2, got %v", resp["total"])
	}
}

func TestV2EventsListProjectIDFilter(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Second", "/tmp/second", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	insertTestEvent(t, testDB, "evt_40", project.ID, "task.created", "task", "501", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_41", "proj_default", "task.updated", "task", "501", "2026-02-11T10:01:00.000000Z")

	path := "/api/v2/events?project_id=" + project.ID
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

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatalf("expected events array, got %T", resp["events"])
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	firstEvent, ok := events[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected event object, got %T", events[0])
	}
	if firstEvent["project_id"] != project.ID {
		t.Fatalf("expected project_id %s, got %v", project.ID, firstEvent["project_id"])
	}

	if total, ok := resp["total"].(float64); !ok || int(total) != 1 {
		t.Fatalf("expected total 1, got %v", resp["total"])
	}
}

func TestV2EventsListInvalidParams(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	tests := []string{
		"/api/v2/events?limit=0",
		"/api/v2/events?offset=-1",
		"/api/v2/events?offset=bogus",
		"/api/v2/events?since=not-a-time",
		"/api/v2/events?task_id=0",
		"/api/v2/events?task_id=-1",
		"/api/v2/events?task_id=bogus",
		"/api/v2/events?project_id=",
		"/api/v2/events?project_id=proj_missing",
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

func TestV2EventsListMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}

func TestV2EventsListPaging(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	insertTestEvent(t, testDB, "evt_20", "proj_default", "task.created", "task", "401", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_21", "proj_default", "task.updated", "task", "401", "2026-02-11T10:01:00.000000Z")
	insertTestEvent(t, testDB, "evt_22", "proj_default", "task.completed", "task", "401", "2026-02-11T10:02:00.000000Z")

	req := httptest.NewRequest(http.MethodGet, "/api/v2/events?limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatalf("expected events array, got %T", resp["events"])
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	firstEvent, ok := events[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected event object, got %T", events[0])
	}
	if firstEvent["kind"] != "task.updated" {
		t.Fatalf("expected paged event kind task.updated, got %v", firstEvent["kind"])
	}

	if total, ok := resp["total"].(float64); !ok || int(total) != 3 {
		t.Fatalf("expected total 3, got %v", resp["total"])
	}
}
