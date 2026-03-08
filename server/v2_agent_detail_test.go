package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2AgentDetailSuccess(t *testing.T) {
	mux, cleanup := setupV2RouteMux(t, runtimeConfig{})
	defer cleanup()

	agent, err := RegisterAgent(db, "Detail Agent", []string{"analysis"}, map[string]interface{}{"tier": "standard"})
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/agents/"+agent.ID, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	agentField, ok := resp["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected agent in response")
	}

	if agentField["id"] != agent.ID {
		t.Fatalf("expected agent id %s, got %v", agent.ID, agentField["id"])
	}
	if agentField["name"] != agent.Name {
		t.Fatalf("expected agent name %s, got %v", agent.Name, agentField["name"])
	}
}

func TestV2AgentDetailNotFound(t *testing.T) {
	mux, cleanup := setupV2RouteMux(t, runtimeConfig{})
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/agents/agent_missing", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}

	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2AgentDetailMethodNotAllowed(t *testing.T) {
	mux, cleanup := setupV2RouteMux(t, runtimeConfig{})
	defer cleanup()

	agent, err := RegisterAgent(db, "Method Agent", nil, nil)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/agents/"+agent.ID, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}

	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
