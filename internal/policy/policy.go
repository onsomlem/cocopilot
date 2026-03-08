// Package policy implements the policy evaluation engine.
package policy

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/onsomlem/cocopilot/internal/models"
)

// PolicyContext contains the context for a policy evaluation request.
type PolicyContext struct {
	ProjectID    string
	AgentID      string
	Action       string // e.g., "create_task", "claim_task", "create_run"
	ResourceType string // e.g., "task", "run", "lease"
	ResourceID   string
	Timestamp    time.Time
	// Optional: current status for workflow constraint checks
	CurrentStatus string
}

// PolicyViolation represents a single policy rule violation.
type PolicyViolation struct {
	PolicyID   string            `json:"policy_id"`
	PolicyName string            `json:"policy_name"`
	Rule       models.PolicyRule `json:"rule"`
	Message    string            `json:"message"`
}

// ---------------------------------------------------------------------------
// In-memory rate tracking
// ---------------------------------------------------------------------------

// rateEntry tracks action timestamps for sliding window rate limiting.
type rateEntry struct {
	timestamps []time.Time
}

// RateTracker provides thread-safe sliding-window rate counting.
type RateTracker struct {
	mu      sync.Mutex
	entries map[string]*rateEntry // key: "projectID:agentID:action"
}

// NewRateTracker creates a new RateTracker.
func NewRateTracker() *RateTracker {
	return &RateTracker{
		entries: make(map[string]*rateEntry),
	}
}

// Record adds a timestamp for the given key and returns the count within the window.
func (rt *RateTracker) Record(key string, ts time.Time, window time.Duration) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	e, ok := rt.entries[key]
	if !ok {
		e = &rateEntry{}
		rt.entries[key] = e
	}

	// Append current timestamp
	e.timestamps = append(e.timestamps, ts)

	// Prune timestamps outside the window
	cutoff := ts.Add(-window)
	pruned := e.timestamps[:0]
	for _, t := range e.timestamps {
		if !t.Before(cutoff) {
			pruned = append(pruned, t)
		}
	}
	e.timestamps = pruned

	return len(pruned)
}

// Count returns the number of actions within the window without recording a new one.
func (rt *RateTracker) Count(key string, ts time.Time, window time.Duration) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	e, ok := rt.entries[key]
	if !ok {
		return 0
	}

	cutoff := ts.Add(-window)
	n := 0
	for _, t := range e.timestamps {
		if !t.Before(cutoff) {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// PolicyEngine
// ---------------------------------------------------------------------------

// PolicyEngine evaluates policies against a context.
type PolicyEngine struct {
	db          *sql.DB
	rateTracker *RateTracker
}

// NewPolicyEngine creates a new PolicyEngine.
func NewPolicyEngine(db *sql.DB) *PolicyEngine {
	return &PolicyEngine{
		db:          db,
		rateTracker: NewRateTracker(),
	}
}

// Evaluate checks the given policies against the context and returns whether
// the action is allowed along with any violations found.
func (pe *PolicyEngine) Evaluate(ctx PolicyContext, policies []models.Policy) (bool, []PolicyViolation) {
	return EvaluatePolicy(ctx, policies, pe.rateTracker, pe.db)
}

// ---------------------------------------------------------------------------
// Core evaluation function
// ---------------------------------------------------------------------------

// EvaluatePolicy iterates through enabled policies, evaluates each rule, and
// returns true if no violations are found. Violations are collected and
// returned when the action should be blocked.
func EvaluatePolicy(ctx PolicyContext, policies []models.Policy, rt *RateTracker, database *sql.DB) (bool, []PolicyViolation) {
	var violations []PolicyViolation

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		for _, rule := range policy.Rules {
			ruleType, _ := rule["type"].(string)
			var v *PolicyViolation

			switch ruleType {
			case "rate_limit":
				v = evaluateRateLimit(ctx, policy, rule, rt)
			case "workflow_constraint":
				v = evaluateWorkflowConstraint(ctx, policy, rule)
			case "resource_quota":
				v = evaluateResourceQuota(ctx, policy, rule, database)
			case "time_window":
				v = evaluateTimeWindow(ctx, policy, rule)
			default:
				// Unknown rule type – skip silently
			}

			if v != nil {
				violations = append(violations, *v)
			}
		}
	}

	return len(violations) == 0, violations
}

// ---------------------------------------------------------------------------
// Rule evaluators
// ---------------------------------------------------------------------------

// evaluateRateLimit checks whether the action count exceeds the limit in the window.
func evaluateRateLimit(ctx PolicyContext, p models.Policy, rule models.PolicyRule, rt *RateTracker) *PolicyViolation {
	if rt == nil {
		return nil
	}

	action, _ := rule["action"].(string)
	if action != "" && action != ctx.Action {
		return nil // rule applies to a different action
	}

	limitF, _ := rule["limit"].(float64) // JSON numbers decode as float64
	limit := int(limitF)
	if limit <= 0 {
		return nil
	}

	windowStr, _ := rule["window"].(string)
	window, err := time.ParseDuration(windowStr)
	if err != nil || window <= 0 {
		return nil
	}

	key := fmt.Sprintf("%s:%s:%s", ctx.ProjectID, ctx.AgentID, ctx.Action)
	ts := ctx.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	current := rt.Count(key, ts, window)
	if current >= limit {
		return &PolicyViolation{
			PolicyID:   p.ID,
			PolicyName: p.Name,
			Rule:       rule,
			Message:    fmt.Sprintf("rate limit exceeded: %d/%d actions in %s window", current, limit, windowStr),
		}
	}

	// Record this action
	rt.Record(key, ts, window)
	return nil
}

// evaluateWorkflowConstraint validates task state transitions.
func evaluateWorkflowConstraint(ctx PolicyContext, p models.Policy, rule models.PolicyRule) *PolicyViolation {
	fromStatus, _ := rule["from_status"].(string)
	if fromStatus == "" {
		return nil
	}

	// Only apply if the current status matches the rule's from_status
	if !strings.EqualFold(ctx.CurrentStatus, fromStatus) {
		return nil
	}

	allowedRaw, ok := rule["allowed_transitions"]
	if !ok {
		return nil
	}

	allowed := toStringSlice(allowedRaw)
	target := ctx.Action // the target status/action

	for _, a := range allowed {
		if strings.EqualFold(a, target) {
			return nil // transition is allowed
		}
	}

	return &PolicyViolation{
		PolicyID:   p.ID,
		PolicyName: p.Name,
		Rule:       rule,
		Message:    fmt.Sprintf("workflow constraint: transition from %q via %q is not allowed (allowed: %v)", fromStatus, target, allowed),
	}
}

// evaluateResourceQuota checks that the number of concurrent resources does not exceed max_count.
func evaluateResourceQuota(ctx PolicyContext, p models.Policy, rule models.PolicyRule, database *sql.DB) *PolicyViolation {
	if database == nil {
		return nil // cannot check without a database
	}

	resourceType, _ := rule["resource_type"].(string)
	if resourceType != "" && resourceType != ctx.ResourceType {
		return nil // rule applies to a different resource type
	}

	maxCountF, _ := rule["max_count"].(float64)
	maxCount := int(maxCountF)
	if maxCount <= 0 {
		return nil
	}

	count, err := countActiveResources(database, ctx.ProjectID, resourceType)
	if err != nil {
		// On error, be permissive – do not block the action
		return nil
	}

	if count >= maxCount {
		return &PolicyViolation{
			PolicyID:   p.ID,
			PolicyName: p.Name,
			Rule:       rule,
			Message:    fmt.Sprintf("resource quota exceeded: %d/%d %s resources", count, maxCount, resourceType),
		}
	}

	return nil
}

// evaluateTimeWindow restricts operations to allowed hours.
func evaluateTimeWindow(ctx PolicyContext, p models.Policy, rule models.PolicyRule) *PolicyViolation {
	startHourF, _ := rule["start_hour"].(float64)
	endHourF, _ := rule["end_hour"].(float64)
	startHour := int(startHourF)
	endHour := int(endHourF)

	tzName, _ := rule["timezone"].(string)
	if tzName == "" {
		tzName = "UTC"
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil // invalid timezone – skip
	}

	ts := ctx.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	localTime := ts.In(loc)
	hour := localTime.Hour()

	var allowed bool
	if startHour <= endHour {
		// Same-day window, e.g. 9–17
		allowed = hour >= startHour && hour < endHour
	} else {
		// Overnight window, e.g. 22–6 means 22,23,0,1,2,3,4,5
		allowed = hour >= startHour || hour < endHour
	}

	if !allowed {
		return &PolicyViolation{
			PolicyID:   p.ID,
			PolicyName: p.Name,
			Rule:       rule,
			Message:    fmt.Sprintf("time window violation: hour %d not in allowed range [%d, %d) tz=%s", hour, startHour, endHour, tzName),
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// toStringSlice converts an interface{} that may be []interface{} (from JSON)
// into a []string.
func toStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// countActiveResources counts resources of the given type in a project.
// It queries the database for the appropriate table based on resource type.
func countActiveResources(database *sql.DB, projectID string, resourceType string) (int, error) {
	var query string
	switch resourceType {
	case "task":
		query = `SELECT COUNT(*) FROM tasks WHERE project_id = ? AND status_v2 IN ('QUEUED','CLAIMED','RUNNING')`
	case "run":
		query = `SELECT COUNT(*) FROM runs WHERE task_id IN (SELECT id FROM tasks WHERE project_id = ?) AND status IN ('running','pending')`
	case "lease":
		query = `SELECT COUNT(*) FROM leases WHERE task_id IN (SELECT id FROM tasks WHERE project_id = ?) AND expires_at > ?`
	default:
		return 0, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	var count int
	var err error
	if resourceType == "lease" {
		err = database.QueryRow(query, projectID, models.NowISO()).Scan(&count)
	} else {
		err = database.QueryRow(query, projectID).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to count %s resources: %w", resourceType, err)
	}
	return count, nil
}
