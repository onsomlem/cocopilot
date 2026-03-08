package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2ProjectAuditListSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(testDB, "Audit", "/tmp/audit", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	insertTestEvent(t, testDB, "evt_a1", project.ID, "task.created", "task", "501", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_a2", project.ID, "task.created", "task", "501", "2026-02-11T10:02:00.000000Z")
	insertTestEvent(t, testDB, "evt_a3", "proj_default", "task.created", "task", "501", "2026-02-11T10:03:00.000000Z")

	path := "/api/v2/projects/" + project.ID + "/audit?task_id=501&type=task.created&since=2026-02-11T10:01:00.000000Z&limit=1&offset=0"
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

func TestV2ProjectAuditListRejectsMismatchedProjectID(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_default/audit?project_id=proj_other", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectAuditListNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing/audit", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectAuditAuthScopes(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Audit Scope", "/tmp/audit-scope", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	cfg := runtimeConfig{
		RequireAPIKey:      true,
		RequireAPIKeyReads: true,
		AuthIdentities: []authIdentity{
			{
				ID:     "writer",
				Type:   "service",
				APIKey: "writer-key",
				Scopes: map[string]struct{}{"v2:write": {}},
			},
			{
				ID:     "events_reader",
				Type:   "service",
				APIKey: "events-key",
				Scopes: map[string]struct{}{"events.read": {}},
			},
			{
				ID:     "audit_reader",
				Type:   "service",
				APIKey: "audit-key",
				Scopes: map[string]struct{}{"audit.read": {}},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, cfg)

	path := "/api/v2/projects/" + project.ID + "/audit"

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-API-Key", "writer-key")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "FORBIDDEN")
	details := errField["details"].(map[string]interface{})
	if details["required_scope"] != "events.read or audit.read" {
		t.Fatalf("expected required_scope events.read or audit.read, got %v", details["required_scope"])
	}

	req = httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-API-Key", "events-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 with events.read, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-API-Key", "audit-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 with audit.read, got %d body=%s", w.Code, w.Body.String())
	}
}