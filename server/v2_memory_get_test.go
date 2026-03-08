package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2ProjectMemoryGetSuccessFilters(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	_, _ = CreateMemory(testDB, "proj_default", "GLOBAL", "release_notes", map[string]interface{}{"title": "Alpha"}, nil)
	_, _ = CreateMemory(testDB, "proj_default", "MODULE", "api_patterns", map[string]interface{}{"note": "Beta"}, nil)
	_, _ = CreateMemory(testDB, "proj_default", "GLOBAL", "architecture", map[string]interface{}{"note": "Alpha Beta"}, nil)

	fetchItems := func(url string) []interface{} {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		items, ok := resp["items"].([]interface{})
		if !ok {
			t.Fatalf("expected items array, got %T", resp["items"])
		}
		return items
	}

	items := fetchItems("/api/v2/projects/proj_default/memory?scope=GLOBAL")
	if len(items) != 2 {
		t.Fatalf("expected 2 items for scope filter, got %d", len(items))
	}

	items = fetchItems("/api/v2/projects/proj_default/memory?key=architecture")
	if len(items) != 1 {
		t.Fatalf("expected 1 item for key filter, got %d", len(items))
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected item object, got %T", items[0])
	}
	if item["key"] != "architecture" {
		t.Fatalf("expected key architecture, got %v", item["key"])
	}

	items = fetchItems("/api/v2/projects/proj_default/memory?q=alpha")
	if len(items) != 2 {
		t.Fatalf("expected 2 items for query filter, got %d", len(items))
	}
}

func TestV2ProjectMemoryGetProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing/memory", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectMemoryGetMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v2/projects/proj_default/memory", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodDelete {
		t.Fatalf("expected details.method %s, got %v", http.MethodDelete, details["method"])
	}
	allowed, ok := details["allowed_methods"].([]interface{})
	if !ok || len(allowed) != 2 || allowed[0] != http.MethodGet || allowed[1] != http.MethodPut {
		t.Fatalf("expected allowed_methods [%s %s], got %v", http.MethodGet, http.MethodPut, details["allowed_methods"])
	}
}
