package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2ConfigReturnsSafeRuntimeConfig(t *testing.T) {
	cfg := runtimeConfig{
		DBPath:                   "C:\\secret\\tasks.db",
		RequireAPIKey:            true,
		RequireAPIKeyReads:       true,
		SSEHeartbeatSeconds:      42,
		SSEReplayLimitMax:        120,
		EventsRetentionDays:      7,
		EventsRetentionMax:       500,
		EventsPruneIntervalSeconds: 900,
		AuthIdentities: []authIdentity{
			{
				ID:     "reader",
				Type:   "service",
				APIKey: "reader-key",
				Scopes: map[string]struct{}{"v2:read": {}},
			},
		},
	}

	mux, cleanup := setupV2RouteMux(t, cfg)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/config", nil)
	req.Header.Set("X-API-Key", "reader-key")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["db_path"] != "[redacted]" {
		t.Fatalf("expected db_path to be redacted, got %v", resp["db_path"])
	}
	if resp["db_path"] == cfg.DBPath {
		t.Fatalf("expected db_path to be redacted, got original value")
	}

	auth, ok := resp["auth"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected auth object, got %T", resp["auth"])
	}
	if auth["required"] != true {
		t.Fatalf("expected auth.required true, got %v", auth["required"])
	}
	if auth["require_reads"] != true {
		t.Fatalf("expected auth.require_reads true, got %v", auth["require_reads"])
	}
	if auth["identity_count"].(float64) != 1 {
		t.Fatalf("expected auth.identity_count 1, got %v", auth["identity_count"])
	}
	if auth["legacy_api_key_set"] != false {
		t.Fatalf("expected auth.legacy_api_key_set false, got %v", auth["legacy_api_key_set"])
	}

	retention, ok := resp["retention"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retention object, got %T", resp["retention"])
	}
	if retention["enabled"] != true {
		t.Fatalf("expected retention.enabled true, got %v", retention["enabled"])
	}
	if retention["interval_seconds"].(float64) != 900 {
		t.Fatalf("expected retention.interval_seconds 900, got %v", retention["interval_seconds"])
	}
	if retention["max_rows"].(float64) != 500 {
		t.Fatalf("expected retention.max_rows 500, got %v", retention["max_rows"])
	}
	if retention["days"].(float64) != 7 {
		t.Fatalf("expected retention.days 7, got %v", retention["days"])
	}

	sse, ok := resp["sse"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sse object, got %T", resp["sse"])
	}
	if sse["heartbeat_seconds"].(float64) != 42 {
		t.Fatalf("expected sse.heartbeat_seconds 42, got %v", sse["heartbeat_seconds"])
	}
	if sse["replay_limit_max"].(float64) != 120 {
		t.Fatalf("expected sse.replay_limit_max 120, got %v", sse["replay_limit_max"])
	}
}

func TestV2ConfigAuthScopeEnforced(t *testing.T) {
	cfg := runtimeConfig{
		RequireAPIKey:      true,
		RequireAPIKeyReads: true,
		AuthIdentities: []authIdentity{
			{
				ID:     "reader",
				Type:   "service",
				APIKey: "reader-key",
				Scopes: map[string]struct{}{"v2:read": {}},
			},
			{
				ID:     "writer",
				Type:   "service",
				APIKey: "writer-key",
				Scopes: map[string]struct{}{"v2:write": {}},
			},
		},
	}

	mux, cleanup := setupV2RouteMux(t, cfg)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/config", nil)
	req.Header.Set("X-API-Key", "writer-key")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for insufficient scope, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "FORBIDDEN")
	details := errField["details"].(map[string]interface{})
	if details["required_scope"] != "v2:read" {
		t.Fatalf("expected required_scope v2:read, got %v", details["required_scope"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v2/config", nil)
	req.Header.Set("X-API-Key", "reader-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for read scope, got %d body=%s", w.Code, w.Body.String())
	}
}
