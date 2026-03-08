package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2ProjectAutomationRulesSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Automation Rules", "/tmp/automation-rules", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	enabled := true
	rules := []automationRule{{
		Name:    "Followup",
		Enabled: &enabled,
		Trigger: "task.completed",
		Actions: []automationAction{{
			Type: "create_task",
			Task: automationTaskSpec{
				Instructions: "Review ${task_id}",
			},
		}},
	}}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})
	defer setAutomationRules(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/automation/rules", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	gotRules, ok := resp["rules"].([]interface{})
	if !ok {
		t.Fatalf("expected rules array, got %T", resp["rules"])
	}
	if len(gotRules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(gotRules))
	}

	rule, ok := gotRules[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected rule object, got %T", gotRules[0])
	}
	if rule["name"] != "Followup" {
		t.Fatalf("expected rule name Followup, got %v", rule["name"])
	}
}

func TestV2ProjectAutomationRulesEmpty(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Automation Rules Empty", "/tmp/automation-rules-empty", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})
	defer setAutomationRules(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/automation/rules", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	gotRules, ok := resp["rules"].([]interface{})
	if !ok {
		t.Fatalf("expected rules array, got %T", resp["rules"])
	}
	if len(gotRules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(gotRules))
	}
}

func TestV2ProjectAutomationRulesProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing/automation/rules", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectAutomationRulesMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/automation/rules", nil)
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
	if !ok || len(allowed) != 1 || allowed[0] != http.MethodGet {
		t.Fatalf("expected allowed_methods [%s], got %v", http.MethodGet, details["allowed_methods"])
	}
}
