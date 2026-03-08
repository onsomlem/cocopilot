package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// B1.5: Comprehensive Policy Enforcement Tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 1. EvaluatePolicy – multiple policies with different rule types
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_MultiplePoliciesDifferentTypes(t *testing.T) {
	rt := newPolicyRateTracker()

	ctx := PolicyContext{
		ProjectID:    "proj_1",
		AgentID:      "agent_1",
		Action:       "create_task",
		ResourceType: "task",
		Timestamp:    time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC), // 14:00 UTC
	}

	policies := []Policy{
		{
			ID: "pol_tw", ProjectID: "proj_1", Name: "Time Window",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(8), "end_hour": float64(18), "timezone": "UTC"},
			},
		},
		{
			ID: "pol_rl", ProjectID: "proj_1", Name: "Rate Limit",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "rate_limit", "action": "create_task", "limit": float64(100), "window": "1h"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if !allowed {
		t.Fatalf("expected allowed, got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// 2. Policy with multiple rules – some pass, some fail
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_MixedRulesPartialViolation(t *testing.T) {
	rt := newPolicyRateTracker()

	ctx := PolicyContext{
		ProjectID:    "proj_1",
		AgentID:      "agent_1",
		Action:       "create_task",
		ResourceType: "task",
		Timestamp:    time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC), // 03:00 UTC – outside 8-18
	}

	policies := []Policy{
		{
			ID: "pol_mixed", ProjectID: "proj_1", Name: "Mixed Rules",
			Enabled: true,
			Rules: []PolicyRule{
				// This should pass (within rate limit)
				{"type": "rate_limit", "action": "create_task", "limit": float64(100), "window": "1h"},
				// This should fail (outside time window)
				{"type": "time_window", "start_hour": float64(8), "end_hour": float64(18), "timezone": "UTC"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if allowed {
		t.Fatal("expected blocked due to time_window violation")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].PolicyID != "pol_mixed" {
		t.Errorf("expected violation from pol_mixed, got %s", violations[0].PolicyID)
	}
}

// ---------------------------------------------------------------------------
// 3. All 4 rule types in a single evaluation
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_AllFourRuleTypes(t *testing.T) {
	rt := newPolicyRateTracker()

	ctx := PolicyContext{
		ProjectID:     "proj_1",
		AgentID:       "agent_1",
		Action:        "create_task",
		ResourceType:  "task",
		CurrentStatus: "QUEUED",
		Timestamp:     time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	policies := []Policy{
		{
			ID: "pol_all", ProjectID: "proj_1", Name: "All Rules",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "rate_limit", "action": "create_task", "limit": float64(50), "window": "1h"},
				{"type": "workflow_constraint", "from_status": "QUEUED", "allowed_transitions": []interface{}{"CLAIMED", "CANCELLED", "create_task"}},
				{"type": "resource_quota", "resource_type": "task", "max_count": float64(1000)},
				{"type": "time_window", "start_hour": float64(6), "end_hour": float64(22), "timezone": "UTC"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if !allowed {
		t.Fatalf("expected all rules to pass, got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// 4. Edge cases: empty rules, nil rules, unknown rule type
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_EmptyRulesArray(t *testing.T) {
	ctx := PolicyContext{
		ProjectID: "proj_1", AgentID: "agent_1", Action: "create_task",
		Timestamp: time.Now().UTC(),
	}

	policies := []Policy{
		{ID: "pol_empty", ProjectID: "proj_1", Name: "Empty Rules", Enabled: true, Rules: []PolicyRule{}},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if !allowed {
		t.Fatalf("expected allowed with empty rules, got violations: %v", violations)
	}
	if len(violations) != 0 {
		t.Fatalf("expected zero violations, got %d", len(violations))
	}
}

func TestPolicyEvaluate_NilRulesSlice(t *testing.T) {
	ctx := PolicyContext{
		ProjectID: "proj_1", AgentID: "agent_1", Action: "create_task",
		Timestamp: time.Now().UTC(),
	}

	policies := []Policy{
		{ID: "pol_nil", ProjectID: "proj_1", Name: "Nil Rules", Enabled: true, Rules: nil},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if !allowed {
		t.Fatalf("expected allowed with nil rules, got violations: %v", violations)
	}
}

func TestPolicyEvaluate_UnknownRuleType(t *testing.T) {
	ctx := PolicyContext{
		ProjectID: "proj_1", AgentID: "agent_1", Action: "create_task",
		Timestamp: time.Now().UTC(),
	}

	policies := []Policy{
		{
			ID: "pol_unknown", ProjectID: "proj_1", Name: "Unknown Type", Enabled: true,
			Rules: []PolicyRule{
				{"type": "nonexistent_rule_type", "foo": "bar"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if !allowed {
		t.Fatalf("expected allowed (unknown types skipped), got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// 5. Integration: middleware blocking POST with a time_window violation
// ---------------------------------------------------------------------------

func TestPolicyEnforcement_MiddlewareBlocksPost(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	project, err := CreateProject(db, "Enforcement Test", "/tmp/enforce", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Create a time_window policy that blocks all hours (start=end)
	enabled := true
	_, err = CreatePolicy(db, project.ID, "Block All Hours", nil, []PolicyRule{
		{"type": "time_window", "start_hour": float64(25), "end_hour": float64(25), "timezone": "UTC"},
	}, enabled)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	handler := policyEnforcementMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks?project_id="+project.ID, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj, _ := resp["error"].(map[string]interface{})
	if errObj == nil {
		t.Fatal("expected error object in response")
	}
	if code, _ := errObj["code"].(string); code != "POLICY_VIOLATION" {
		t.Errorf("expected POLICY_VIOLATION code, got %s", code)
	}
}

// ---------------------------------------------------------------------------
// 6. Integration: middleware allows POST when policy passes
// ---------------------------------------------------------------------------

func TestPolicyEnforcement_MiddlewareAllowsPost(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	project, err := CreateProject(db, "Allow Test", "/tmp/allow", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Create a permissive rate limit policy
	enabled := true
	_, err = CreatePolicy(db, project.ID, "Permissive Rate Limit", nil, []PolicyRule{
		{"type": "rate_limit", "action": "create_task", "limit": float64(10000), "window": "1h"},
	}, enabled)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	called := false
	handler := policyEnforcementMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks?project_id="+project.ID, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Fatal("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 7. Middleware with mixed enabled/disabled policies
// ---------------------------------------------------------------------------

func TestPolicyEnforcement_MixedEnabledDisabled(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldPE := policyEngine
	policyEngine = NewPolicyEngine(db)
	defer func() { policyEngine = oldPE }()

	project, err := CreateProject(db, "Mixed Policies", "/tmp/mixed", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Create a DISABLED restrictive policy (should be ignored)
	disabled := false
	_, err = CreatePolicy(db, project.ID, "Disabled Blocker", nil, []PolicyRule{
		{"type": "time_window", "start_hour": float64(25), "end_hour": float64(25), "timezone": "UTC"},
	}, disabled)
	if err != nil {
		t.Fatalf("CreatePolicy disabled: %v", err)
	}

	// Create an ENABLED permissive policy
	enabled := true
	_, err = CreatePolicy(db, project.ID, "Enabled Permissive", nil, []PolicyRule{
		{"type": "rate_limit", "action": "create_task", "limit": float64(10000), "window": "1h"},
	}, enabled)
	if err != nil {
		t.Fatalf("CreatePolicy enabled: %v", err)
	}

	called := false
	handler := policyEnforcementMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks?project_id="+project.ID, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Fatal("expected handler to be called (disabled policies should be skipped)")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 8. Enable/disable behavior – disabled policy not evaluated
// ---------------------------------------------------------------------------

func TestPolicyEnforcement_DisabledNotEvaluated(t *testing.T) {
	rt := newPolicyRateTracker()

	// Restrictive policy but disabled
	policies := []Policy{
		{
			ID: "pol_dis", ProjectID: "proj_1", Name: "Disabled Restrictive",
			Enabled: false,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(25), "end_hour": float64(25), "timezone": "UTC"},
			},
		},
	}

	ctx := PolicyContext{
		ProjectID: "proj_1", AgentID: "agent_1", Action: "create_task",
		Timestamp: time.Now().UTC(),
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if !allowed {
		t.Fatalf("disabled policy should not block, got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// 9. Enable then evaluate – re-enabled policy should block
// ---------------------------------------------------------------------------

func TestPolicyEnforcement_EnabledPolicyBlocks(t *testing.T) {
	rt := newPolicyRateTracker()

	// Same restrictive policy but enabled
	policies := []Policy{
		{
			ID: "pol_en", ProjectID: "proj_1", Name: "Enabled Restrictive",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(25), "end_hour": float64(25), "timezone": "UTC"},
			},
		},
	}

	ctx := PolicyContext{
		ProjectID: "proj_1", AgentID: "agent_1", Action: "create_task",
		Timestamp: time.Now().UTC(),
	}

	allowed, _ := EvaluatePolicy(ctx, policies, rt, nil)
	if allowed {
		t.Fatal("enabled restrictive policy should block")
	}
}

// ---------------------------------------------------------------------------
// 10. Rate limit window expiration – blocks then allows
// ---------------------------------------------------------------------------

func TestPolicyEnforcement_RateLimitWindowExpires(t *testing.T) {
	rt := newPolicyRateTracker()

	policies := []Policy{
		{
			ID: "pol_rl", ProjectID: "proj_1", Name: "Tight Rate Limit",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "rate_limit", "action": "create_task", "limit": float64(2), "window": "100ms"},
			},
		},
	}

	ctx := PolicyContext{
		ProjectID: "proj_1", AgentID: "agent_1", Action: "create_task",
		ResourceType: "task",
		Timestamp:    time.Now().UTC(),
	}

	// First two should be allowed
	allowed1, _ := EvaluatePolicy(ctx, policies, rt, nil)
	if !allowed1 {
		t.Fatal("first request should be allowed")
	}

	ctx.Timestamp = time.Now().UTC()
	allowed2, _ := EvaluatePolicy(ctx, policies, rt, nil)
	if !allowed2 {
		t.Fatal("second request should be allowed")
	}

	// Third should be blocked
	ctx.Timestamp = time.Now().UTC()
	allowed3, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if allowed3 {
		t.Fatal("third request should be blocked (rate limit exceeded)")
	}
	if len(violations) == 0 {
		t.Fatal("expected violations for rate limit")
	}

	// Wait for the window to expire
	time.Sleep(150 * time.Millisecond)

	// Now should be allowed again
	ctx.Timestamp = time.Now().UTC()
	allowed4, _ := EvaluatePolicy(ctx, policies, rt, nil)
	if !allowed4 {
		t.Fatal("request after window expiry should be allowed")
	}
}

// ---------------------------------------------------------------------------
// 11. Multiple violations from multiple policies
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_MultipleViolationsFromMultiplePolicies(t *testing.T) {
	rt := newPolicyRateTracker()

	ctx := PolicyContext{
		ProjectID:    "proj_1",
		AgentID:      "agent_1",
		Action:       "create_task",
		ResourceType: "task",
		Timestamp:    time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC), // 03:00 – outside window
	}

	policies := []Policy{
		{
			ID: "pol_tw", ProjectID: "proj_1", Name: "Time Window Policy",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(8), "end_hour": float64(18), "timezone": "UTC"},
			},
		},
		{
			ID: "pol_tw2", ProjectID: "proj_1", Name: "Another Time Window",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(9), "end_hour": float64(17), "timezone": "UTC"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if allowed {
		t.Fatal("expected blocked by both time window policies")
	}
	if len(violations) < 2 {
		t.Fatalf("expected at least 2 violations, got %d", len(violations))
	}

	// Verify violations come from different policies
	policyIDs := make(map[string]bool)
	for _, v := range violations {
		policyIDs[v.PolicyID] = true
	}
	if len(policyIDs) < 2 {
		t.Errorf("expected violations from 2 different policies, got %d", len(policyIDs))
	}
}
