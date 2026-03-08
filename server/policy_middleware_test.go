package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// policyEnforcementMiddleware tests
// ---------------------------------------------------------------------------

func TestPolicyMiddleware_PassthroughGET(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Ensure policyEngine is initialised for this test
	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	called := false
	handler := policyEnforcementMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/tasks?project_id=proj1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Fatal("expected handler to be called for GET request")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPolicyMiddleware_PassthroughNoPolicies(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	project, err := CreateProject(db, "MW Test Project", "/tmp/mw-test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	called := false
	handler := policyEnforcementMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks?project_id="+project.ID, strings.NewReader(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Fatal("expected handler to be called when no policies exist")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPolicyMiddleware_BlocksOnViolation(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	project, err := CreateProject(db, "MW Block Project", "/tmp/mw-block", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Insert an enabled policy with a time_window rule where start_hour==end_hour==0,
	// meaning no hour satisfies (hour >= 0 && hour < 0) – always blocks.
	rulesJSON := `[{"type":"time_window","start_hour":0,"end_hour":0,"timezone":"UTC"}]`
	_, err = db.Exec(`INSERT INTO policies (id, project_id, name, rules_json, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"pol_block", project.ID, "Block Policy", rulesJSON, 1, nowISO())
	if err != nil {
		t.Fatalf("failed to insert policy: %v", err)
	}

	called := false
	handler := policyEnforcementMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks?project_id="+project.ID, strings.NewReader(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if called {
		t.Fatal("expected handler NOT to be called when policy blocks")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object, got %v", body)
	}
	if errObj["code"] != "POLICY_VIOLATION" {
		t.Fatalf("expected code POLICY_VIOLATION, got %v", errObj["code"])
	}
}

func TestPolicyMiddleware_PassthroughNoProjectID(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	called := false
	handler := policyEnforcementMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// POST without any project_id – middleware should pass through
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(`{"title":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Fatal("expected handler to be called when project_id is absent")
	}
}

func TestPolicyMiddleware_IntegrationRoute(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	project, err := CreateProject(db, "Route MW Project", "/tmp/route-mw", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Create a task normally – no policies, should succeed
	payload := `{"title":"Task for MW","project_id":"` + project.ID + `","instructions":"test instructions"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 when no policies exist, got %d body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// extractProjectID tests
// ---------------------------------------------------------------------------

func TestExtractProjectID_QueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks?project_id=proj_abc", nil)
	if got := extractProjectID(req); got != "proj_abc" {
		t.Fatalf("expected proj_abc, got %s", got)
	}
}

func TestExtractProjectID_URLPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_xyz/policies", nil)
	if got := extractProjectID(req); got != "proj_xyz" {
		t.Fatalf("expected proj_xyz, got %s", got)
	}
}

func TestExtractProjectID_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", nil)
	if got := extractProjectID(req); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// extractAgentID tests
// ---------------------------------------------------------------------------

func TestExtractAgentID_Header(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", nil)
	req.Header.Set("X-Agent-ID", "agent_1")
	if got := extractAgentID(req); got != "agent_1" {
		t.Fatalf("expected agent_1, got %s", got)
	}
}

func TestExtractAgentID_QueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks?agent_id=agent_2", nil)
	if got := extractAgentID(req); got != "agent_2" {
		t.Fatalf("expected agent_2, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// determinePolicyAction tests
// ---------------------------------------------------------------------------

func TestDeterminePolicyAction_CreateTask(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", nil)
	if got := determinePolicyAction(req); got != "create_task" {
		t.Fatalf("expected create_task, got %s", got)
	}
}

func TestDeterminePolicyAction_ClaimTask(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks/42/claim", nil)
	if got := determinePolicyAction(req); got != "claim_task" {
		t.Fatalf("expected claim_task, got %s", got)
	}
}

func TestDeterminePolicyAction_CompleteTask(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks/42/complete", nil)
	if got := determinePolicyAction(req); got != "complete_task" {
		t.Fatalf("expected complete_task, got %s", got)
	}
}

func TestDeterminePolicyAction_UpdateTask(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/api/v2/tasks/42", nil)
	if got := determinePolicyAction(req); got != "update_task" {
		t.Fatalf("expected update_task, got %s", got)
	}
}

func TestDeterminePolicyAction_DeleteTask(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/v2/tasks/42", nil)
	if got := determinePolicyAction(req); got != "delete_task" {
		t.Fatalf("expected delete_task, got %s", got)
	}
}

func TestDeterminePolicyAction_CreateRun(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/runs/", nil)
	if got := determinePolicyAction(req); got != "create_run" {
		t.Fatalf("expected create_run, got %s", got)
	}
}

func TestDeterminePolicyAction_CreateRunStep(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/runs/run_1/steps", nil)
	if got := determinePolicyAction(req); got != "create_run_step" {
		t.Fatalf("expected create_run_step, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// determineResourceType tests
// ---------------------------------------------------------------------------

func TestDetermineResourceType_Task(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", nil)
	if got := determineResourceType(req); got != "task" {
		t.Fatalf("expected task, got %s", got)
	}
}

func TestDetermineResourceType_Run(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/runs/run_1/steps", nil)
	if got := determineResourceType(req); got != "run" {
		t.Fatalf("expected run, got %s", got)
	}
}

func TestDetermineResourceType_Policy(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_1/policies", nil)
	if got := determineResourceType(req); got != "policy" {
		t.Fatalf("expected policy, got %s", got)
	}
}

func TestDetermineResourceType_Project(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_1", nil)
	if got := determineResourceType(req); got != "project" {
		t.Fatalf("expected project, got %s", got)
	}
}
