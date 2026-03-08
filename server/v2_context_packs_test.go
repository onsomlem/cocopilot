package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2ProjectContextPacksCreateSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, testDB)

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := fmt.Sprintf(`{"task_id":%d,"query":"focus on auth","budget":{"max_files":5,"max_bytes":2048,"max_snippets":12}}`, taskID)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/context-packs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pack, ok := resp["context_pack"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected context_pack object, got %T", resp["context_pack"])
	}
	if int(pack["task_id"].(float64)) != taskID {
		t.Fatalf("expected task_id %d, got %v", taskID, pack["task_id"])
	}
	if pack["project_id"] != "proj_default" {
		t.Fatalf("expected project_id proj_default, got %v", pack["project_id"])
	}

	contents, ok := pack["contents"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected contents object, got %T", pack["contents"])
	}
	if contents["query"] != "focus on auth" {
		t.Fatalf("expected query focus on auth, got %v", contents["query"])
	}

	persisted, err := GetContextPackByTaskID(testDB, taskID)
	if err != nil {
		t.Fatalf("GetContextPackByTaskID failed: %v", err)
	}
	if persisted == nil {
		t.Fatal("expected context pack to be persisted")
	}
}

func TestV2ProjectContextPacksValidationError(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"task_id":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/context-packs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectContextPacksProjectNotFound(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, testDB)

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := fmt.Sprintf(`{"task_id":%d}`, taskID)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_missing/context-packs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectContextPacksTaskNotInProject(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Other Project", "/tmp/other", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	taskID := createTestTask(t, testDB)

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := fmt.Sprintf(`{"task_id":%d}`, taskID)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/projects/%s/context-packs", project.ID), strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectContextPacksMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_default/context-packs", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodGet {
		t.Fatalf("expected details.method %s, got %v", http.MethodGet, details["method"])
	}
	allowed, ok := details["allowed_methods"].([]interface{})
	if !ok || len(allowed) != 1 || allowed[0] != http.MethodPost {
		t.Fatalf("expected allowed_methods [%s], got %v", http.MethodPost, details["allowed_methods"])
	}
}

func TestV2ContextPackDetailSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, testDB)
	contents := map[string]interface{}{
		"query": "context pack detail",
		"budget": map[string]interface{}{
			"max_files":    3,
			"max_bytes":    1024,
			"max_snippets": 4,
		},
	}
	pack, err := CreateContextPack(testDB, "proj_default", taskID, "Detail pack", contents)
	if err != nil {
		t.Fatalf("CreateContextPack failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/context-packs/"+pack.ID, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	packResp, ok := resp["context_pack"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected context_pack object, got %T", resp["context_pack"])
	}
	if packResp["id"] != pack.ID {
		t.Fatalf("expected id %s, got %v", pack.ID, packResp["id"])
	}
	if packResp["project_id"] != "proj_default" {
		t.Fatalf("expected project_id proj_default, got %v", packResp["project_id"])
	}
}

func TestV2ContextPackDetailNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/context-packs/ctx_missing", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ContextPackDetailInvalidID(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/context-packs/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ContextPackDetailMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/context-packs/ctx_123", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}
