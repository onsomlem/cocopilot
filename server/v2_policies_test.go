package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestV2ProjectPoliciesCreateSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Project", "/tmp/policy-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	payload := `{"name":"  Policy Alpha  ","description":"  Basic policy  ","rules":[{"type":"automation.block","reason":"Audit"}],"enabled":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies", strings.NewReader(payload))
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

	policy, ok := resp["policy"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected policy object, got %T", resp["policy"])
	}
	if policy["project_id"] != project.ID {
		t.Fatalf("expected project_id %s, got %v", project.ID, policy["project_id"])
	}
	if policy["name"] != "Policy Alpha" {
		t.Fatalf("expected trimmed name, got %v", policy["name"])
	}
	if policy["description"] != "Basic policy" {
		t.Fatalf("expected trimmed description, got %v", policy["description"])
	}
	if policy["enabled"] != false {
		t.Fatalf("expected enabled false, got %v", policy["enabled"])
	}

	rules, ok := policy["rules"].([]interface{})
	if !ok {
		t.Fatalf("expected rules array, got %T", policy["rules"])
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	first, ok := rules[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected rule object, got %T", rules[0])
	}
	if first["type"] != "automation.block" {
		t.Fatalf("expected rules[0].type automation.block, got %v", first["type"])
	}

	policyID, ok := policy["id"].(string)
	if !ok || policyID == "" {
		t.Fatalf("expected policy id, got %v", policy["id"])
	}

	events, err := GetEventsByProjectID(db, project.ID, 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}
	createdEvent := findEventByKind(events, "policy.created")
	if createdEvent == nil {
		t.Fatal("expected policy.created event")
	}
	if createdEvent.ProjectID != project.ID {
		t.Fatalf("expected policy.created project_id %s, got %s", project.ID, createdEvent.ProjectID)
	}
	if createdEvent.EntityType != "policy" {
		t.Fatalf("expected policy.created entity_type policy, got %s", createdEvent.EntityType)
	}
	if createdEvent.EntityID != policyID {
		t.Fatalf("expected policy.created entity_id %s, got %s", policyID, createdEvent.EntityID)
	}
	if createdEvent.Payload["policy_id"] != policyID {
		t.Fatalf("expected policy.created payload policy_id %s, got %v", policyID, createdEvent.Payload["policy_id"])
	}
}

func TestV2ProjectPoliciesAuthScopes(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	cfg := runtimeConfig{
		RequireAPIKey:      true,
		RequireAPIKeyReads: true,
		AuthIdentities: []authIdentity{
			{
				ID:     "policy_reader",
				Type:   "service",
				APIKey: "read-key",
				Scopes: map[string]struct{}{"policy.read": {}},
			},
			{
				ID:     "policy_writer",
				Type:   "service",
				APIKey: "write-key",
				Scopes: map[string]struct{}{"policy.write": {}},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, cfg)

	project, err := CreateProject(db, "Policy Auth", "/tmp/policy-auth", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/policies", nil)
	req.Header.Set("X-API-Key", "write-key")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for write-only scope on list, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "FORBIDDEN")
	details := errField["details"].(map[string]interface{})
	if details["required_scope"] != "policy.read" {
		t.Fatalf("expected required_scope policy.read, got %v", details["required_scope"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/policies", nil)
	req.Header.Set("X-API-Key", "read-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for read scope on list, got %d body=%s", w.Code, w.Body.String())
	}

	payload := `{"name":"Policy Scope","rules":[{"type":"automation.block"}],"enabled":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "read-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for read-only scope on create, got %d body=%s", w.Code, w.Body.String())
	}
	errField = assertV2ErrorEnvelope(t, w, "FORBIDDEN")
	details = errField["details"].(map[string]interface{})
	if details["required_scope"] != "policy.write" {
		t.Fatalf("expected required_scope policy.write, got %v", details["required_scope"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "write-key")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for write scope on create, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2ProjectPoliciesListSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy List", "/tmp/policy-list", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	_, _ = CreatePolicy(db, project.ID, "Policy A", nil, []PolicyRule{{"type": "automation.block"}}, true)
	_, _ = CreatePolicy(db, project.ID, "Policy B", nil, []PolicyRule{}, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/policies", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	policies, ok := resp["policies"].([]interface{})
	if !ok {
		t.Fatalf("expected policies array, got %T", resp["policies"])
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}
	if total, ok := resp["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("expected total 2, got %v", resp["total"])
	}
}

func TestV2ProjectPoliciesListEnabledFilter(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy List Filter", "/tmp/policy-list-filter", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	_, _ = CreatePolicy(db, project.ID, "Policy Enabled", nil, []PolicyRule{{"type": "automation.block"}}, true)
	_, _ = CreatePolicy(db, project.ID, "Policy Disabled", nil, []PolicyRule{}, false)

	requestWithFilter := func(enabledValue string) []interface{} {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/policies?enabled="+enabledValue, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		policies, ok := resp["policies"].([]interface{})
		if !ok {
			t.Fatalf("expected policies array, got %T", resp["policies"])
		}
		if total, ok := resp["total"].(float64); !ok || int(total) != 1 {
			t.Fatalf("expected total 1, got %v", resp["total"])
		}
		return policies
	}

	policies := requestWithFilter("true")
	if len(policies) != 1 {
		t.Fatalf("expected 1 enabled policy, got %d", len(policies))
	}

	policies = requestWithFilter("false")
	if len(policies) != 1 {
		t.Fatalf("expected 1 disabled policy, got %d", len(policies))
	}
}

func TestV2ProjectPoliciesListPaging(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy List Paging", "/tmp/policy-list-paging", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	_, _ = CreatePolicy(db, project.ID, "Policy A", nil, []PolicyRule{{"type": "automation.block"}}, true)
	_, _ = CreatePolicy(db, project.ID, "Policy B", nil, []PolicyRule{}, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/policies?limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	policies, ok := resp["policies"].([]interface{})
	if !ok {
		t.Fatalf("expected policies array, got %T", resp["policies"])
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	if total, ok := resp["total"].(float64); !ok || int(total) != 2 {
		t.Fatalf("expected total 2, got %v", resp["total"])
	}
}

func TestV2ProjectPoliciesListSorting(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy List Sorting", "/tmp/policy-list-sorting", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policyGamma, err := CreatePolicy(db, project.ID, "Gamma", nil, []PolicyRule{}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	policyAlpha, err := CreatePolicy(db, project.ID, "Alpha", nil, []PolicyRule{}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	policyBeta, err := CreatePolicy(db, project.ID, "Beta", nil, []PolicyRule{}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	base := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	updates := []struct {
		id   string
		time time.Time
	}{
		{policyAlpha.ID, base.Add(2 * time.Minute)},
		{policyBeta.ID, base.Add(1 * time.Minute)},
		{policyGamma.ID, base.Add(3 * time.Minute)},
	}
	for _, update := range updates {
		_, err := db.Exec("UPDATE policies SET created_at = ? WHERE id = ?", update.time.Format(leaseTimeFormat), update.id)
		if err != nil {
			t.Fatalf("failed to update created_at: %v", err)
		}
	}

	getNames := func(path string) []string {
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
		policies, ok := resp["policies"].([]interface{})
		if !ok {
			t.Fatalf("expected policies array, got %T", resp["policies"])
		}
		names := make([]string, 0, len(policies))
		for _, item := range policies {
			policy, ok := item.(map[string]interface{})
			if !ok {
				t.Fatalf("expected policy object, got %T", item)
			}
			name, _ := policy["name"].(string)
			names = append(names, name)
		}
		return names
	}

	createdDesc := getNames("/api/v2/projects/" + project.ID + "/policies?sort=created_at:desc")
	if len(createdDesc) != 3 || createdDesc[0] != "Gamma" || createdDesc[1] != "Alpha" || createdDesc[2] != "Beta" {
		t.Fatalf("unexpected created_at desc order: %v", createdDesc)
	}

	nameAsc := getNames("/api/v2/projects/" + project.ID + "/policies?sort=name:asc")
	if len(nameAsc) != 3 || nameAsc[0] != "Alpha" || nameAsc[1] != "Beta" || nameAsc[2] != "Gamma" {
		t.Fatalf("unexpected name asc order: %v", nameAsc)
	}
}

func TestV2ProjectPoliciesListInvalidSort(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy List Invalid Sort", "/tmp/policy-list-invalid-sort", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	invalidPaths := []string{
		"/api/v2/projects/" + project.ID + "/policies?sort=created_at",
		"/api/v2/projects/" + project.ID + "/policies?sort=name",
		"/api/v2/projects/" + project.ID + "/policies?sort=unknown:asc",
		"/api/v2/projects/" + project.ID + "/policies?sort=created_at:up",
	}

	for _, path := range invalidPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for %s, got %d body=%s", path, w.Code, w.Body.String())
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	}
}

func TestV2ProjectPoliciesListInvalidPaging(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy List Invalid Paging", "/tmp/policy-list-invalid", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	invalidPaths := []string{
		"/api/v2/projects/" + project.ID + "/policies?limit=0",
		"/api/v2/projects/" + project.ID + "/policies?limit=-1",
		"/api/v2/projects/" + project.ID + "/policies?limit=bad",
		"/api/v2/projects/" + project.ID + "/policies?offset=-2",
		"/api/v2/projects/" + project.ID + "/policies?offset=bogus",
	}

	for _, path := range invalidPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for %s, got %d body=%s", path, w.Code, w.Body.String())
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	}
}

func TestV2ProjectPoliciesValidationError(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"name":"   ","rules":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectPoliciesNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing/policies", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectPoliciesInvalidRules(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Invalid Rules", "/tmp/policy-invalid-rules", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	payload := `{"name":"Policy Bad","rules":[{"reason":"missing type"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectPoliciesMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v2/projects/proj_default/policies", nil)
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
	if !ok || len(allowed) != 2 || allowed[0] != http.MethodGet || allowed[1] != http.MethodPost {
		t.Fatalf("expected allowed_methods [%s %s], got %v", http.MethodGet, http.MethodPost, details["allowed_methods"])
	}
}

func TestV2ProjectPolicyDetailSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Detail", "/tmp/policy-detail", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Detail", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/policies/"+policy.ID, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	policyResp, ok := resp["policy"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected policy object, got %T", resp["policy"])
	}
	if policyResp["id"] != policy.ID {
		t.Fatalf("expected policy id %s, got %v", policy.ID, policyResp["id"])
	}
}

func TestV2ProjectPolicyDetailNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Missing", "/tmp/policy-missing", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/policies/pol_missing", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectPolicyUpdateSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Update", "/tmp/policy-update-route", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Alpha", nil, []PolicyRule{}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	payload := `{"name":"  Policy Beta  ","description":"  Updated  ","rules":[{"type":"automation.block","reason":"Block"}],"enabled":true}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v2/projects/"+project.ID+"/policies/"+policy.ID, strings.NewReader(payload))
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
	policyResp, ok := resp["policy"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected policy object, got %T", resp["policy"])
	}
	if policyResp["name"] != "Policy Beta" {
		t.Fatalf("expected updated name, got %v", policyResp["name"])
	}
	if policyResp["description"] != "Updated" {
		t.Fatalf("expected updated description, got %v", policyResp["description"])
	}
	if policyResp["enabled"] != true {
		t.Fatalf("expected enabled true, got %v", policyResp["enabled"])
	}

	events, err := GetEventsByProjectID(db, project.ID, 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}
	updatedEvent := findEventByKind(events, "policy.updated")
	if updatedEvent == nil {
		t.Fatal("expected policy.updated event")
	}
	if updatedEvent.ProjectID != project.ID {
		t.Fatalf("expected policy.updated project_id %s, got %s", project.ID, updatedEvent.ProjectID)
	}
	if updatedEvent.EntityType != "policy" {
		t.Fatalf("expected policy.updated entity_type policy, got %s", updatedEvent.EntityType)
	}
	if updatedEvent.EntityID != policy.ID {
		t.Fatalf("expected policy.updated entity_id %s, got %s", policy.ID, updatedEvent.EntityID)
	}
	if updatedEvent.Payload["policy_id"] != policy.ID {
		t.Fatalf("expected policy.updated payload policy_id %s, got %v", policy.ID, updatedEvent.Payload["policy_id"])
	}
}

func TestV2ProjectPolicyUpdateValidationError(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Update Invalid", "/tmp/policy-update-invalid", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Alpha", nil, []PolicyRule{}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v2/projects/"+project.ID+"/policies/"+policy.ID, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectPolicyUpdateInvalidRules(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Update Invalid Rules", "/tmp/policy-update-invalid-rules", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Alpha", nil, []PolicyRule{}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	payload := `{"rules":[{"type":"unknown.rule"}]}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v2/projects/"+project.ID+"/policies/"+policy.ID, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectPolicyDeleteSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Delete", "/tmp/policy-delete", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Delete", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v2/projects/"+project.ID+"/policies/"+policy.ID, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", w.Code, w.Body.String())
	}

	events, err := GetEventsByProjectID(db, project.ID, 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}
	deletedEvent := findEventByKind(events, "policy.deleted")
	if deletedEvent == nil {
		t.Fatal("expected policy.deleted event")
	}
	if deletedEvent.ProjectID != project.ID {
		t.Fatalf("expected policy.deleted project_id %s, got %s", project.ID, deletedEvent.ProjectID)
	}
	if deletedEvent.EntityType != "policy" {
		t.Fatalf("expected policy.deleted entity_type policy, got %s", deletedEvent.EntityType)
	}
	if deletedEvent.EntityID != policy.ID {
		t.Fatalf("expected policy.deleted entity_id %s, got %s", policy.ID, deletedEvent.EntityID)
	}
	if deletedEvent.Payload["policy_id"] != policy.ID {
		t.Fatalf("expected policy.deleted payload policy_id %s, got %v", policy.ID, deletedEvent.Payload["policy_id"])
	}
}

func TestV2ProjectPolicyDetailMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Policy Method", "/tmp/policy-method", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Method", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/policies/"+policy.ID, nil)
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
}
