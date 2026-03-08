package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupV2RouteMux(t *testing.T, cfg runtimeConfig) (*http.ServeMux, func()) {
	t.Helper()
	_, cleanup := setupTestDB(t)
	mux := http.NewServeMux()
	registerRoutes(mux, cfg)
	return mux, cleanup
}

func TestV2RoutesMethodNotAllowedEnvelope(t *testing.T) {
	mux, cleanup := setupV2RouteMux(t, runtimeConfig{})
	defer cleanup()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "health", method: http.MethodPost, path: "/api/v2/health"},
		{name: "version", method: http.MethodPost, path: "/api/v2/version"},
		{name: "projects_root", method: http.MethodDelete, path: "/api/v2/projects"},
		{name: "agents_root", method: http.MethodPatch, path: "/api/v2/agents"},
		{name: "leases_root", method: http.MethodGet, path: "/api/v2/leases"},
		{name: "runs_item", method: http.MethodPost, path: "/api/v2/runs/run_missing"},
		{name: "project_tree", method: http.MethodPost, path: "/api/v2/projects/proj_default/tree"},
		{name: "project_changes", method: http.MethodPost, path: "/api/v2/projects/proj_default/changes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
			}

			errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
			details := errField["details"].(map[string]interface{})
			if details["method"] != tt.method {
				t.Fatalf("expected details.method %s, got %v", tt.method, details["method"])
			}
		})
	}
}

func TestV2RoutesProjectNotFoundEnvelope(t *testing.T) {
	mux, cleanup := setupV2RouteMux(t, runtimeConfig{})
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}

	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2RoutesHealthSuccessContract(t *testing.T) {
	mux, cleanup := setupV2RouteMux(t, runtimeConfig{})
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("expected ok=true, got %v", resp["ok"])
	}
}

func TestV2RoutesAuthMutatingEndpoint(t *testing.T) {
	cfg := runtimeConfig{
		RequireAPIKey: true,
		APIKey:        "test-secret",
	}
	mux, cleanup := setupV2RouteMux(t, cfg)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v2/agents", strings.NewReader(`{"name":"Auth Agent"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 without key, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "UNAUTHORIZED")

	req = httptest.NewRequest(http.MethodPost, "/api/v2/agents", strings.NewReader(`{"name":"Auth Agent"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "wrong-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 with wrong key, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "UNAUTHORIZED")

	req = httptest.NewRequest(http.MethodPost, "/api/v2/agents", strings.NewReader(`{"name":"Auth Agent"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-secret")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 with valid key, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RoutesAuthReadToggle(t *testing.T) {
	muxMutatingOnly, cleanupMutatingOnly := setupV2RouteMux(t, runtimeConfig{
		RequireAPIKey: true,
		APIKey:        "test-secret",
	})
	defer cleanupMutatingOnly()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
	w := httptest.NewRecorder()
	muxMutatingOnly.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for read when COCO_REQUIRE_API_KEY_READS disabled, got %d body=%s", w.Code, w.Body.String())
	}

	muxReadsEnabled, cleanupReadsEnabled := setupV2RouteMux(t, runtimeConfig{
		RequireAPIKey:      true,
		RequireAPIKeyReads: true,
		APIKey:             "test-secret",
	})
	defer cleanupReadsEnabled()

	req = httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
	w = httptest.NewRecorder()
	muxReadsEnabled.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for read without key when reads auth enabled, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "UNAUTHORIZED")

	req = httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
	req.Header.Set("X-API-Key", "test-secret")
	w = httptest.NewRecorder()
	muxReadsEnabled.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for read with valid key when reads auth enabled, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RoutesAuthForbiddenForInsufficientScope(t *testing.T) {
	cfg := runtimeConfig{
		RequireAPIKey: true,
		AuthIdentities: []authIdentity{
			{
				ID:     "agent_runner",
				Type:   "agent",
				APIKey: "agent-key",
				Scopes: map[string]struct{}{
					"leases:write": {},
					"v2:read":      {},
				},
			},
		},
	}
	mux, cleanup := setupV2RouteMux(t, cfg)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects", strings.NewReader(`{"name":"Scoped Project","workdir":"/tmp/scoped"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "agent-key")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for insufficient scope, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "FORBIDDEN")
	details := errField["details"].(map[string]interface{})
	if details["required_scope"] != "projects:write" {
		t.Fatalf("expected required_scope projects:write, got %v", details["required_scope"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v2/leases", strings.NewReader(`{"task_id":0,"agent_id":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "agent-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for leases request after passing auth/scope checks, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2RoutesAuthDenialsEmitEvents(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{
		RequireAPIKey: true,
		AuthIdentities: []authIdentity{
			{
				ID:     "agent_runner",
				Type:   "agent",
				APIKey: "agent-key",
				Scopes: map[string]struct{}{
					"leases:write": {},
					"v2:read":      {},
				},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, cfg)

	// Missing API key -> UNAUTHORIZED
	req := httptest.NewRequest(http.MethodPost, "/api/v2/agents", strings.NewReader(`{"name":"Auth Agent"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for missing key, got %d body=%s", w.Code, w.Body.String())
	}

	// Valid key but insufficient scope -> FORBIDDEN
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects", strings.NewReader(`{"name":"Scoped Project","workdir":"/tmp/scoped"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "agent-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for insufficient scope, got %d body=%s", w.Code, w.Body.String())
	}

	events, err := GetEventsByProjectID(testDB, "proj_default", 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}

	var missingEvent *Event
	var forbiddenEvent *Event
	for i := range events {
		if events[i].Kind != "auth.denied" {
			continue
		}
		reason, _ := events[i].Payload["reason"].(string)
		switch reason {
		case "missing_api_key":
			missingEvent = &events[i]
		case "insufficient_scope":
			forbiddenEvent = &events[i]
		}
	}

	if missingEvent == nil {
		t.Fatal("expected auth.denied event for missing_api_key")
	}
	if missingEvent.Payload["required_scope"] != "agents:write" {
		t.Fatalf("expected required_scope agents:write, got %v", missingEvent.Payload["required_scope"])
	}
	if missingEvent.Payload["endpoint"] != "/api/v2/agents" {
		t.Fatalf("expected endpoint /api/v2/agents, got %v", missingEvent.Payload["endpoint"])
	}
	if missingEvent.Payload["method"] != http.MethodPost {
		t.Fatalf("expected method POST, got %v", missingEvent.Payload["method"])
	}
	if _, ok := missingEvent.Payload["identity_id"]; ok {
		t.Fatalf("did not expect identity_id on missing_api_key event, got %v", missingEvent.Payload["identity_id"])
	}

	if forbiddenEvent == nil {
		t.Fatal("expected auth.denied event for insufficient_scope")
	}
	if forbiddenEvent.Payload["required_scope"] != "projects:write" {
		t.Fatalf("expected required_scope projects:write, got %v", forbiddenEvent.Payload["required_scope"])
	}
	if forbiddenEvent.Payload["identity_id"] != "agent_runner" {
		t.Fatalf("expected identity_id agent_runner, got %v", forbiddenEvent.Payload["identity_id"])
	}
}
