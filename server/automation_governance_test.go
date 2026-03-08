package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/onsomlem/cocopilot/internal/ratelimit"
)

// ===========================================================================
// B2.5: Automation Governance Tests
// ===========================================================================

// ---------------------------------------------------------------------------
// Test 1: Recursion depth limit enforcement
// ---------------------------------------------------------------------------

func TestAutomationGovernanceRecursionDepthLimit(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Save and restore global state.
	origDepth := getMaxAutomationDepth()
	defer setMaxAutomationDepth(origDepth)

	origRateLimiter := automationRateLimiter
	automationRateLimiter = ratelimit.NewSlidingWindowRateLimiter()
	defer func() { automationRateLimiter = origRateLimiter }()

	origCircuit := automationCircuit
	setAutomationCircuitBreaker(newAutomationCircuitBreaker(5, 5*time.Minute))
	defer setAutomationCircuitBreaker(origCircuit)

	origRules := getAutomationRules()
	defer setAutomationRules(origRules)

	// Set max depth to 3.
	setMaxAutomationDepth(3)

	// Set up a simple automation rule.
	enabled := true
	rules := []automationRule{{
		Name:    "depth-test",
		Enabled: &enabled,
		Trigger: "task.completed",
		Actions: []automationAction{{
			Type: "create_task",
			Task: automationTaskSpec{
				Instructions: "Follow-up for task ${task_id}",
			},
		}},
	}}
	setAutomationRules(rules)

	// Create a project.
	project, err := CreateProject(testDB, "Depth Test Project", "/tmp/depth-test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// --- Case A: Parent depth=2 → child depth=3, should be allowed ---
	taskA, err := CreateTaskV2(testDB, "Task at depth 2", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 failed: %v", err)
	}
	if err := setTaskAutomationDepth(testDB, taskA.ID, 2); err != nil {
		t.Fatalf("setTaskAutomationDepth failed: %v", err)
	}

	// Fire a task.completed event for taskA.
	processAutomationEvent(testDB, Event{
		ID:         "evt_test_depth_ok",
		ProjectID:  project.ID,
		Kind:       "task.completed",
		EntityType: "task",
		EntityID:   fmt.Sprintf("%d", taskA.ID),
		CreatedAt:  nowISO(),
		Payload: map[string]interface{}{
			"task_id": float64(taskA.ID),
		},
	})

	// Check that a child task was created (automation.triggered event exists).
	evtsA, _, err := ListEvents(testDB, project.ID, "automation.triggered", "", "", 100, 0)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(evtsA) == 0 {
		t.Fatal("Expected automation.triggered event for depth-2 task, got none")
	}

	// --- Case B: Parent depth=3 → child would be depth=4, should be blocked ---
	taskB, err := CreateTaskV2(testDB, "Task at depth 3", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 failed: %v", err)
	}
	if err := setTaskAutomationDepth(testDB, taskB.ID, 3); err != nil {
		t.Fatalf("setTaskAutomationDepth failed: %v", err)
	}

	processAutomationEvent(testDB, Event{
		ID:         "evt_test_depth_block",
		ProjectID:  project.ID,
		Kind:       "task.completed",
		EntityType: "task",
		EntityID:   fmt.Sprintf("%d", taskB.ID),
		CreatedAt:  nowISO(),
		Payload: map[string]interface{}{
			"task_id": float64(taskB.ID),
		},
	})

	// Check that an automation.blocked event was emitted with recursion_depth_exceeded.
	evtsB, _, err := ListEvents(testDB, project.ID, "automation.blocked", "", "", 100, 0)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	found := false
	for _, e := range evtsB {
		if reason, ok := e.Payload["reason"].(string); ok && reason == "recursion_depth_exceeded" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected automation.blocked event with reason=recursion_depth_exceeded, not found")
	}
}

// ---------------------------------------------------------------------------
// Test 2: Rate limiting blocks excessive automation
// ---------------------------------------------------------------------------

func TestAutomationGovernanceRateLimiting(t *testing.T) {
	rl := ratelimit.NewSlidingWindowRateLimiter()
	now, _ := mockClock(time.Now())
	rl.NowFunc = now

	limit := 3
	window := time.Hour
	projectID := "proj_rate_test"
	agentID := ""

	// First 3 calls should succeed.
	for i := 0; i < limit; i++ {
		if !rl.CheckRateLimit(projectID, agentID, limit, window) {
			t.Fatalf("Request %d should have been allowed", i+1)
		}
	}

	// 4th call should be blocked.
	if rl.CheckRateLimit(projectID, agentID, limit, window) {
		t.Fatal("4th request should have been rate-limited")
	}
}

// ---------------------------------------------------------------------------
// Test 3: Circuit breaker opens after failures
// ---------------------------------------------------------------------------

func TestAutomationGovernanceCircuitBreakerOpens(t *testing.T) {
	cb := newAutomationCircuitBreaker(3, 5*time.Minute)

	// Initially the circuit should be closed and allow execution.
	if !cb.AllowExecution("test_rule") {
		t.Fatal("Expected AllowExecution to return true for closed circuit")
	}

	// Record 3 failures to trigger circuit open.
	for i := 0; i < 3; i++ {
		cb.RecordFailure("test_rule")
	}

	// Circuit should now be open.
	if cb.AllowExecution("test_rule") {
		t.Fatal("Expected AllowExecution to return false after 3 failures")
	}

	state := cb.GetState("test_rule")
	if state != circuitOpen {
		t.Fatalf("Expected circuit state=open, got %s", state)
	}
}

// ---------------------------------------------------------------------------
// Test 4: Circuit breaker auto-reset after cooldown
// ---------------------------------------------------------------------------

func TestAutomationGovernanceCircuitBreakerAutoReset(t *testing.T) {
	cooldown := 100 * time.Millisecond
	cb := newAutomationCircuitBreaker(2, cooldown)

	// Open the circuit by recording 2 failures.
	cb.RecordFailure("test_rule")
	cb.RecordFailure("test_rule")

	// Verify circuit is open.
	if cb.AllowExecution("test_rule") {
		t.Fatal("Expected circuit to be open after 2 failures")
	}

	// Wait for cooldown to elapse.
	time.Sleep(150 * time.Millisecond)

	// Circuit should transition to half-open and allow one probe request.
	if !cb.AllowExecution("test_rule") {
		t.Fatal("Expected AllowExecution to return true after cooldown (half-open)")
	}

	// Record success to close the circuit.
	cb.RecordSuccess("test_rule")

	// Circuit should now be closed.
	state := cb.GetState("test_rule")
	if state != circuitClosed {
		t.Fatalf("Expected circuit state=closed after success, got %s", state)
	}

	// Further calls should be allowed.
	if !cb.AllowExecution("test_rule") {
		t.Fatal("Expected AllowExecution to return true after circuit closed")
	}
}

// ---------------------------------------------------------------------------
// Test 5: Audit events emitted during automation processing
// ---------------------------------------------------------------------------

func TestAutomationGovernanceAuditEvents(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Save and restore global state.
	origDepth := getMaxAutomationDepth()
	defer setMaxAutomationDepth(origDepth)

	origRateLimiter := automationRateLimiter
	automationRateLimiter = ratelimit.NewSlidingWindowRateLimiter()
	defer func() { automationRateLimiter = origRateLimiter }()

	origCircuit := automationCircuit
	setAutomationCircuitBreaker(newAutomationCircuitBreaker(5, 5*time.Minute))
	defer setAutomationCircuitBreaker(origCircuit)

	origRules := getAutomationRules()
	defer setAutomationRules(origRules)

	// Use generous limits so automation is not blocked by rate limits.
	setMaxAutomationDepth(10)
	setAutomationRateLimit(1000)
	setAutomationBurstLimit(1000)

	// Set up a rule.
	enabled := true
	rules := []automationRule{{
		Name:    "audit-test-rule",
		Enabled: &enabled,
		Trigger: "task.completed",
		Actions: []automationAction{{
			Type: "create_task",
			Task: automationTaskSpec{
				Instructions: "Audit follow-up for ${task_id}",
			},
		}},
	}}
	setAutomationRules(rules)

	// Create a project and task.
	project, err := CreateProject(testDB, "Audit Test Project", "/tmp/audit-test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	task, err := CreateTaskV2(testDB, "Complete me", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 failed: %v", err)
	}

	// Fire automation directly.
	processAutomationEvent(testDB, Event{
		ID:         "evt_audit_test",
		ProjectID:  project.ID,
		Kind:       "task.completed",
		EntityType: "task",
		EntityID:   fmt.Sprintf("%d", task.ID),
		CreatedAt:  nowISO(),
		Payload: map[string]interface{}{
			"task_id": float64(task.ID),
		},
	})

	// Query for automation.triggered events.
	events, _, err := ListEvents(testDB, project.ID, "automation.triggered", "", "", 100, 0)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("Expected at least one automation.triggered event")
	}

	// Verify payload contains expected fields.
	evt := events[0]
	if evt.Kind != "automation.triggered" {
		t.Fatalf("Expected kind=automation.triggered, got %s", evt.Kind)
	}
	if evt.ProjectID != project.ID {
		t.Fatalf("Expected project_id=%s, got %s", project.ID, evt.ProjectID)
	}
	if ruleName, ok := evt.Payload["rule_name"].(string); !ok || ruleName != "audit-test-rule" {
		t.Fatalf("Expected payload rule_name=audit-test-rule, got %v", evt.Payload["rule_name"])
	}
	if trigger, ok := evt.Payload["trigger"].(string); !ok || trigger != "task.completed" {
		t.Fatalf("Expected payload trigger=task.completed, got %v", evt.Payload["trigger"])
	}
	if evtID, ok := evt.Payload["event_id"].(string); !ok || evtID != "evt_audit_test" {
		t.Fatalf("Expected payload event_id=evt_audit_test, got %v", evt.Payload["event_id"])
	}

	// Also verify a task.created event was emitted for the child task.
	createdEvents, _, err := ListEvents(testDB, project.ID, "task.created", "", "", 100, 0)
	if err != nil {
		t.Fatalf("ListEvents for task.created failed: %v", err)
	}
	if len(createdEvents) == 0 {
		t.Fatal("Expected at least one task.created event from automation")
	}
}
