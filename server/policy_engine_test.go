package server

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// EvaluatePolicy – no policies → allowed
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_NoPolicies(t *testing.T) {
	ctx := PolicyContext{
		ProjectID:    "proj_1",
		AgentID:      "agent_1",
		Action:       "create_task",
		ResourceType: "task",
		Timestamp:    time.Now().UTC(),
	}

	allowed, violations := EvaluatePolicy(ctx, nil, nil, nil)
	if !allowed {
		t.Fatalf("expected allowed with no policies, got violations: %v", violations)
	}
	if len(violations) != 0 {
		t.Fatalf("expected zero violations, got %d", len(violations))
	}
}

// ---------------------------------------------------------------------------
// EvaluatePolicy – disabled policies → allowed
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_DisabledPolicies(t *testing.T) {
	ctx := PolicyContext{
		ProjectID: "proj_1",
		AgentID:   "agent_1",
		Action:    "create_task",
		Timestamp: time.Now().UTC(),
	}

	policies := []Policy{
		{
			ID:        "pol_1",
			ProjectID: "proj_1",
			Name:      "Disabled Rate Limit",
			Enabled:   false,
			Rules: []PolicyRule{
				{"type": "rate_limit", "action": "create_task", "limit": float64(1), "window": "1h"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, newPolicyRateTracker(), nil)
	if !allowed {
		t.Fatalf("expected allowed when all policies disabled, got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// Time window – allowed hours
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_TimeWindow_Allowed(t *testing.T) {
	// At 14:00 UTC, window 9-17 should allow
	ts := time.Date(2026, 2, 17, 14, 0, 0, 0, time.UTC)

	ctx := PolicyContext{
		ProjectID: "proj_1",
		AgentID:   "agent_1",
		Action:    "create_task",
		Timestamp: ts,
	}

	policies := []Policy{
		{
			ID:      "pol_tw",
			Name:    "Business Hours",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(9), "end_hour": float64(17), "timezone": "UTC"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if !allowed {
		t.Fatalf("expected allowed at hour 14, got violations: %v", violations)
	}
	if len(violations) != 0 {
		t.Fatalf("expected zero violations, got %d", len(violations))
	}
}

// ---------------------------------------------------------------------------
// Time window – blocked hours
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_TimeWindow_Blocked(t *testing.T) {
	// At 03:00 UTC, window 9-17 should block
	ts := time.Date(2026, 2, 17, 3, 0, 0, 0, time.UTC)

	ctx := PolicyContext{
		ProjectID: "proj_1",
		AgentID:   "agent_1",
		Action:    "create_task",
		Timestamp: ts,
	}

	policies := []Policy{
		{
			ID:      "pol_tw",
			Name:    "Business Hours",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(9), "end_hour": float64(17), "timezone": "UTC"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if allowed {
		t.Fatal("expected blocked at hour 3, but was allowed")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].PolicyID != "pol_tw" {
		t.Fatalf("expected policy ID pol_tw, got %s", violations[0].PolicyID)
	}
}

// ---------------------------------------------------------------------------
// Time window – overnight window
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_TimeWindow_Overnight(t *testing.T) {
	// Overnight window 22-6: hour 23 should be allowed, hour 10 should be blocked
	tsAllowed := time.Date(2026, 2, 17, 23, 0, 0, 0, time.UTC)
	tsBlocked := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)

	policy := Policy{
		ID:      "pol_night",
		Name:    "Night Shift",
		Enabled: true,
		Rules: []PolicyRule{
			{"type": "time_window", "start_hour": float64(22), "end_hour": float64(6), "timezone": "UTC"},
		},
	}

	ctx1 := PolicyContext{ProjectID: "proj_1", Timestamp: tsAllowed}
	allowed, _ := EvaluatePolicy(ctx1, []Policy{policy}, nil, nil)
	if !allowed {
		t.Fatal("expected allowed at hour 23 in overnight window 22-6")
	}

	ctx2 := PolicyContext{ProjectID: "proj_1", Timestamp: tsBlocked}
	allowed2, violations := EvaluatePolicy(ctx2, []Policy{policy}, nil, nil)
	if allowed2 {
		t.Fatal("expected blocked at hour 10 in overnight window 22-6")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

// ---------------------------------------------------------------------------
// Workflow constraint – allowed transition
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_WorkflowConstraint_Allowed(t *testing.T) {
	ctx := PolicyContext{
		ProjectID:     "proj_1",
		AgentID:       "agent_1",
		Action:        "RUNNING",
		CurrentStatus: "CLAIMED",
		Timestamp:     time.Now().UTC(),
	}

	policies := []Policy{
		{
			ID:      "pol_wf",
			Name:    "Task Workflow",
			Enabled: true,
			Rules: []PolicyRule{
				{
					"type":                "workflow_constraint",
					"from_status":         "CLAIMED",
					"allowed_transitions": []interface{}{"RUNNING", "CANCELLED"},
				},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if !allowed {
		t.Fatalf("expected allowed transition CLAIMED->RUNNING, got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// Workflow constraint – blocked transition
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_WorkflowConstraint_Blocked(t *testing.T) {
	ctx := PolicyContext{
		ProjectID:     "proj_1",
		AgentID:       "agent_1",
		Action:        "SUCCEEDED",
		CurrentStatus: "QUEUED",
		Timestamp:     time.Now().UTC(),
	}

	policies := []Policy{
		{
			ID:      "pol_wf",
			Name:    "Task Workflow",
			Enabled: true,
			Rules: []PolicyRule{
				{
					"type":                "workflow_constraint",
					"from_status":         "QUEUED",
					"allowed_transitions": []interface{}{"CLAIMED", "CANCELLED"},
				},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if allowed {
		t.Fatal("expected blocked transition QUEUED->SUCCEEDED, but was allowed")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

// ---------------------------------------------------------------------------
// Rate tracker – basic counting
// ---------------------------------------------------------------------------

func TestPolicyRateTracker_BasicCounting(t *testing.T) {
	rt := newPolicyRateTracker()
	now := time.Now().UTC()
	window := 1 * time.Hour
	key := "proj_1:agent_1:create_task"

	// Initial count should be 0
	c := rt.Count(key, now, window)
	if c != 0 {
		t.Fatalf("expected initial count 0, got %d", c)
	}

	// Record 3 actions
	rt.Record(key, now.Add(-30*time.Minute), window)
	rt.Record(key, now.Add(-20*time.Minute), window)
	rt.Record(key, now.Add(-10*time.Minute), window)

	c = rt.Count(key, now, window)
	if c != 3 {
		t.Fatalf("expected count 3, got %d", c)
	}

	// Record returns the count including the new action
	n := rt.Record(key, now, window)
	if n != 4 {
		t.Fatalf("expected record to return 4, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// Rate tracker – expired entries are pruned
// ---------------------------------------------------------------------------

func TestPolicyRateTracker_WindowExpiry(t *testing.T) {
	rt := newPolicyRateTracker()
	key := "proj_1:agent_1:create_task"
	window := 1 * time.Hour
	now := time.Now().UTC()

	// Record an action 2 hours ago – should be outside the window
	rt.Record(key, now.Add(-2*time.Hour), window)

	c := rt.Count(key, now, window)
	if c != 0 {
		t.Fatalf("expected count 0 for expired entry, got %d", c)
	}
}

// ---------------------------------------------------------------------------
// Rate limit policy – within limit
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_RateLimit_WithinLimit(t *testing.T) {
	rt := newPolicyRateTracker()
	now := time.Now().UTC()

	ctx := PolicyContext{
		ProjectID: "proj_1",
		AgentID:   "agent_1",
		Action:    "create_task",
		Timestamp: now,
	}

	policies := []Policy{
		{
			ID:      "pol_rl",
			Name:    "Create Task Limit",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "rate_limit", "action": "create_task", "limit": float64(5), "window": "1h"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if !allowed {
		t.Fatalf("expected allowed (first action within limit), got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// Rate limit policy – exceeds limit
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_RateLimit_ExceedsLimit(t *testing.T) {
	rt := newPolicyRateTracker()
	now := time.Now().UTC()
	key := "proj_1:agent_1:create_task"
	window := 1 * time.Hour

	// Pre-fill with 5 actions
	for i := 0; i < 5; i++ {
		rt.Record(key, now.Add(-time.Duration(i)*time.Minute), window)
	}

	ctx := PolicyContext{
		ProjectID: "proj_1",
		AgentID:   "agent_1",
		Action:    "create_task",
		Timestamp: now,
	}

	policies := []Policy{
		{
			ID:      "pol_rl",
			Name:    "Create Task Limit",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "rate_limit", "action": "create_task", "limit": float64(5), "window": "1h"},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if allowed {
		t.Fatal("expected blocked (rate limit exceeded), but was allowed")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

// ---------------------------------------------------------------------------
// Resource quota – no DB (should be permissive)
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_ResourceQuota_NoDB(t *testing.T) {
	ctx := PolicyContext{
		ProjectID:    "proj_1",
		ResourceType: "task",
		Timestamp:    time.Now().UTC(),
	}

	policies := []Policy{
		{
			ID:      "pol_rq",
			Name:    "Task Quota",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "resource_quota", "resource_type": "task", "max_count": float64(10)},
			},
		},
	}

	// Without a DB, resource quota should be permissive
	allowed, violations := EvaluatePolicy(ctx, policies, nil, nil)
	if !allowed {
		t.Fatalf("expected allowed without DB (permissive), got violations: %v", violations)
	}
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

// ---------------------------------------------------------------------------
// PolicyEngine struct usage
// ---------------------------------------------------------------------------

func TestPolicyEngine_Evaluate(t *testing.T) {
	engine := NewPolicyEngine(nil)

	ctx := PolicyContext{
		ProjectID: "proj_1",
		AgentID:   "agent_1",
		Action:    "create_task",
		Timestamp: time.Now().UTC(),
	}

	// No policies → allowed
	allowed, violations := engine.Evaluate(ctx, nil)
	if !allowed {
		t.Fatalf("expected allowed, got violations: %v", violations)
	}

	// With a time window policy during allowed hours
	ts := time.Date(2026, 2, 17, 12, 0, 0, 0, time.UTC)
	ctx.Timestamp = ts

	policies := []Policy{
		{
			ID:      "pol_1",
			Name:    "Business Hours",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(9), "end_hour": float64(17), "timezone": "UTC"},
			},
		},
	}

	allowed, violations = engine.Evaluate(ctx, policies)
	if !allowed {
		t.Fatalf("expected allowed during business hours, got violations: %v", violations)
	}
}

// ---------------------------------------------------------------------------
// Multiple rules in a single policy
// ---------------------------------------------------------------------------

func TestPolicyEvaluate_MultipleRules(t *testing.T) {
	rt := newPolicyRateTracker()
	// At 03:00 UTC – outside business hours
	ts := time.Date(2026, 2, 17, 3, 0, 0, 0, time.UTC)

	ctx := PolicyContext{
		ProjectID:     "proj_1",
		AgentID:       "agent_1",
		Action:        "SUCCEEDED",
		CurrentStatus: "QUEUED",
		Timestamp:     ts,
	}

	policies := []Policy{
		{
			ID:      "pol_multi",
			Name:    "Multi-Rule",
			Enabled: true,
			Rules: []PolicyRule{
				{"type": "time_window", "start_hour": float64(9), "end_hour": float64(17), "timezone": "UTC"},
				{
					"type":                "workflow_constraint",
					"from_status":         "QUEUED",
					"allowed_transitions": []interface{}{"CLAIMED"},
				},
			},
		},
	}

	allowed, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if allowed {
		t.Fatal("expected blocked with multiple violations")
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d: %v", len(violations), violations)
	}
}
