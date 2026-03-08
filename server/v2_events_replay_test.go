package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2ProjectEventsReplaySuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Replay", "/tmp/replay", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	insertTestEvent(t, testDB, "evt_replay_1", project.ID, "task.created", "task", "101", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_replay_2", project.ID, "task.updated", "task", "101", "2026-02-11T10:01:00.000000Z")
	insertTestEvent(t, testDB, "evt_replay_3", project.ID, "task.updated", "task", "101", "2026-02-11T10:02:00.000000Z")
	insertTestEvent(t, testDB, "evt_other", "proj_default", "task.created", "task", "102", "2026-02-11T10:03:00.000000Z")

	path := fmt.Sprintf("/api/v2/projects/%s/events/replay?since_id=evt_replay_2", project.ID)
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
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	firstEvent := events[0].(map[string]interface{})
	secondEvent := events[1].(map[string]interface{})
	if firstEvent["id"] != "evt_replay_2" {
		t.Fatalf("expected first event evt_replay_2, got %v", firstEvent["id"])
	}
	if secondEvent["id"] != "evt_replay_3" {
		t.Fatalf("expected second event evt_replay_3, got %v", secondEvent["id"])
	}
	if firstEvent["project_id"] != project.ID {
		t.Fatalf("expected project_id %s, got %v", project.ID, firstEvent["project_id"])
	}
}

func TestV2ProjectEventsReplayInvalidSinceID(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Replay", "/tmp/replay", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	insertTestEvent(t, testDB, "evt_other", "proj_default", "task.created", "task", "101", "2026-02-11T10:00:00.000000Z")

	path := fmt.Sprintf("/api/v2/projects/%s/events/replay?since_id=evt_other", project.ID)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectEventsReplayMissingSinceID(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_default/events/replay", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectEventsReplayProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing/events/replay?since_id=evt_missing", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}