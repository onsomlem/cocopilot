package policy

import (
	"testing"
	"time"

	"github.com/onsomlem/cocopilot/internal/models"
)

func TestRateTracker_Record(t *testing.T) {
	rt := NewRateTracker()
	now := time.Now().UTC()
	window := time.Minute

	count := rt.Record("k1", now, window)
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
	count = rt.Record("k1", now.Add(time.Second), window)
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestRateTracker_Count(t *testing.T) {
	rt := NewRateTracker()
	now := time.Now().UTC()
	window := time.Minute

	if c := rt.Count("k1", now, window); c != 0 {
		t.Fatalf("expected 0, got %d", c)
	}

	rt.Record("k1", now, window)
	if c := rt.Count("k1", now.Add(time.Second), window); c != 1 {
		t.Fatalf("expected 1, got %d", c)
	}
}

func TestRateTracker_WindowExpiry(t *testing.T) {
	rt := NewRateTracker()
	now := time.Now().UTC()
	window := time.Minute

	rt.Record("k1", now, window)
	// Check after window expires
	later := now.Add(2 * time.Minute)
	if c := rt.Count("k1", later, window); c != 0 {
		t.Fatalf("expected 0 after window, got %d", c)
	}
}

func TestEvaluatePolicy_NoViolations(t *testing.T) {
	ctx := PolicyContext{
		ProjectID: "p1",
		AgentID:   "a1",
		Action:    "create_task",
		Timestamp: time.Now().UTC(),
	}
	policies := []models.Policy{
		{
			ID:      "pol1",
			Name:    "test",
			Enabled: true,
			Rules: []models.PolicyRule{
				{"type": "rate_limit", "action": "create_task", "limit": float64(100), "window": "1m"},
			},
		},
	}
	rt := NewRateTracker()
	ok, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if !ok || len(violations) > 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestEvaluatePolicy_RateLimitExceeded(t *testing.T) {
	rt := NewRateTracker()
	now := time.Now().UTC()
	ctx := PolicyContext{
		ProjectID: "p1",
		AgentID:   "a1",
		Action:    "claim_task",
		Timestamp: now,
	}
	policies := []models.Policy{
		{
			ID:      "pol1",
			Name:    "rate-test",
			Enabled: true,
			Rules: []models.PolicyRule{
				{"type": "rate_limit", "action": "claim_task", "limit": float64(2), "window": "1m"},
			},
		},
	}
	// First two should pass
	for i := 0; i < 2; i++ {
		ctx.Timestamp = now.Add(time.Duration(i) * time.Second)
		ok, _ := EvaluatePolicy(ctx, policies, rt, nil)
		if !ok {
			t.Fatalf("call %d should pass", i)
		}
	}
	// Third should violate
	ctx.Timestamp = now.Add(3 * time.Second)
	ok, violations := EvaluatePolicy(ctx, policies, rt, nil)
	if ok {
		t.Fatal("expected rate limit violation")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

func TestEvaluatePolicy_DisabledPolicy(t *testing.T) {
	ctx := PolicyContext{Action: "create_task", Timestamp: time.Now().UTC()}
	policies := []models.Policy{
		{ID: "pol1", Name: "disabled", Enabled: false, Rules: []models.PolicyRule{
			{"type": "rate_limit", "action": "create_task", "limit": float64(0), "window": "1m"},
		}},
	}
	ok, _ := EvaluatePolicy(ctx, policies, NewRateTracker(), nil)
	if !ok {
		t.Fatal("disabled policy should not block")
	}
}

func TestEvaluatePolicy_WorkflowConstraint(t *testing.T) {
	ctx := PolicyContext{
		Action:        "complete",
		CurrentStatus: "QUEUED",
	}
	policies := []models.Policy{
		{
			ID: "pol1", Name: "wf", Enabled: true,
			Rules: []models.PolicyRule{
				{
					"type":                "workflow_constraint",
					"from_status":         "QUEUED",
					"allowed_transitions": []interface{}{"claim"},
				},
			},
		},
	}
	ok, violations := EvaluatePolicy(ctx, policies, NewRateTracker(), nil)
	if ok {
		t.Fatal("expected workflow violation")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

func TestEvaluatePolicy_TimeWindow_Allowed(t *testing.T) {
	// Use a timestamp at noon UTC
	noon := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := PolicyContext{Action: "create_task", Timestamp: noon}
	policies := []models.Policy{
		{
			ID: "pol1", Name: "tw", Enabled: true,
			Rules: []models.PolicyRule{
				{"type": "time_window", "start_hour": float64(9), "end_hour": float64(17), "timezone": "UTC"},
			},
		},
	}
	ok, _ := EvaluatePolicy(ctx, policies, NewRateTracker(), nil)
	if !ok {
		t.Fatal("noon should be within 9-17 window")
	}
}

func TestEvaluatePolicy_TimeWindow_Blocked(t *testing.T) {
	midnight := time.Date(2025, 1, 1, 3, 0, 0, 0, time.UTC)
	ctx := PolicyContext{Action: "create_task", Timestamp: midnight}
	policies := []models.Policy{
		{
			ID: "pol1", Name: "tw", Enabled: true,
			Rules: []models.PolicyRule{
				{"type": "time_window", "start_hour": float64(9), "end_hour": float64(17), "timezone": "UTC"},
			},
		},
	}
	ok, violations := EvaluatePolicy(ctx, policies, NewRateTracker(), nil)
	if ok {
		t.Fatal("3am should be outside 9-17 window")
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}
