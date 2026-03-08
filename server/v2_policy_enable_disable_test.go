package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2PolicyEnableSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Enable Test Project", "/tmp/enable-test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a policy with enabled=false
	payload := `{"name":"Disable Me","description":"Test","rules":[{"type":"automation.block","reason":"test"}],"enabled":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create policy: expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	policy := createResp["policy"].(map[string]interface{})
	policyID := policy["id"].(string)

	if policy["enabled"] != false {
		t.Fatalf("expected enabled=false after create, got %v", policy["enabled"])
	}

	// Enable the policy
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies/"+policyID+"/enable", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("enable: expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var enableResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&enableResp)
	enabledPolicy := enableResp["policy"].(map[string]interface{})

	if enabledPolicy["enabled"] != true {
		t.Fatalf("expected enabled=true after enable, got %v", enabledPolicy["enabled"])
	}
	if enabledPolicy["id"] != policyID {
		t.Fatalf("expected policy id %s, got %v", policyID, enabledPolicy["id"])
	}
}

func TestV2PolicyDisableSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Disable Test Project", "/tmp/disable-test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a policy with enabled=true (default)
	payload := `{"name":"Enable Me","description":"Test","rules":[{"type":"automation.block","reason":"test"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create policy: expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	policy := createResp["policy"].(map[string]interface{})
	policyID := policy["id"].(string)

	if policy["enabled"] != true {
		t.Fatalf("expected enabled=true after create, got %v", policy["enabled"])
	}

	// Disable the policy
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies/"+policyID+"/disable", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("disable: expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var disableResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&disableResp)
	disabledPolicy := disableResp["policy"].(map[string]interface{})

	if disabledPolicy["enabled"] != false {
		t.Fatalf("expected enabled=false after disable, got %v", disabledPolicy["enabled"])
	}
}

func TestV2PolicyEnableMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Method Test", "/tmp/method-test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a policy
	payload := `{"name":"Test","description":"Test","rules":[{"type":"automation.block","reason":"test"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create policy: expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	policy := createResp["policy"].(map[string]interface{})
	policyID := policy["id"].(string)

	// Try GET on enable endpoint
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req = httptest.NewRequest(method, "/api/v2/projects/"+project.ID+"/policies/"+policyID+"/enable", nil)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s /enable: expected 405, got %d body=%s", method, w.Code, w.Body.String())
		}
	}

	// Also test disable
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req = httptest.NewRequest(method, "/api/v2/projects/"+project.ID+"/policies/"+policyID+"/disable", nil)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s /disable: expected 405, got %d body=%s", method, w.Code, w.Body.String())
		}
	}
}

func TestV2PolicyEnableNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "NotFound Test", "/tmp/notfound-test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Try to enable a non-existent policy
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies/nonexistent-id/enable", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("enable nonexistent: expected 404, got %d body=%s", w.Code, w.Body.String())
	}

	// Try to disable a non-existent policy
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies/nonexistent-id/disable", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("disable nonexistent: expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2PolicyEnableNonexistentProject(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Try to enable a policy in a non-existent project
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/nonexistent-project/policies/some-policy/enable", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("enable on nonexistent project: expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}
