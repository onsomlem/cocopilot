package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2ProjectMemoryPutSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"scope":"GLOBAL","key":"release_notes","value":{"title":"v1","count":2},"source_refs":["task_1"]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/proj_default/memory", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	item, ok := resp["item"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected item object, got %T", resp["item"])
	}
	if item["project_id"] != "proj_default" {
		t.Fatalf("expected project_id proj_default, got %v", item["project_id"])
	}
	if item["scope"] != "GLOBAL" {
		t.Fatalf("expected scope GLOBAL, got %v", item["scope"])
	}
	if item["key"] != "release_notes" {
		t.Fatalf("expected key release_notes, got %v", item["key"])
	}

	value, ok := item["value"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected value object, got %T", item["value"])
	}
	if value["title"] != "v1" {
		t.Fatalf("expected value.title v1, got %v", value["title"])
	}

	retrieved, err := GetMemory(testDB, "proj_default", "GLOBAL", "release_notes")
	if err != nil {
		t.Fatalf("GetMemory failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected memory to be persisted")
	}
	if retrieved.Value["title"] != "v1" {
		t.Fatalf("expected persisted value.title v1, got %v", retrieved.Value["title"])
	}
}

func TestV2ProjectMemoryPutValidation(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"scope":" ","key":"","value":{"title":"v1"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/proj_default/memory", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectMemoryPutProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"scope":"GLOBAL","key":"release_notes","value":{"title":"v1"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/proj_missing/memory", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectMemoryPutMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/memory", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodPost {
		t.Fatalf("expected details.method %s, got %v", http.MethodPost, details["method"])
	}
	allowed, ok := details["allowed_methods"].([]interface{})
	if !ok || len(allowed) != 2 || allowed[0] != http.MethodGet || allowed[1] != http.MethodPut {
		t.Fatalf("expected allowed_methods [%s %s], got %v", http.MethodGet, http.MethodPut, details["allowed_methods"])
	}
}