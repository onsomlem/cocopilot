package server

import (
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/onsomlem/cocopilot/internal/ratelimit"
)

// automationRule, automationAction, automationTaskSpec are type-aliased
// from internal/config via config.go.

var automationRulesMu sync.RWMutex
var automationRules []automationRule

var maxAutomationDepthMu sync.RWMutex
var maxAutomationDepthVal = 5

var automationRateLimiter *ratelimit.SlidingWindowRateLimiter
var automationRateLimitHour int = 100  // default 100/hour
var automationBurstLimitMinute int = 10 // default 10/minute
var automationRateMu sync.RWMutex

// Circuit breaker types for automation governance.

type circuitState int

const (
    circuitClosed   circuitState = iota // normal operation
    circuitOpen                          // blocking all requests
    circuitHalfOpen                      // testing if service recovered
)

func (s circuitState) String() string {
    switch s {
    case circuitClosed:
        return "closed"
    case circuitOpen:
        return "open"
    case circuitHalfOpen:
        return "half-open"
    default:
        return "unknown"
    }
}

type automationCircuitBreaker struct {
    mu               sync.Mutex
    states           map[string]*circuitEntry // key: rule name
    maxFailures      int                      // consecutive failures before opening (default: 5)
    cooldownDuration time.Duration            // time before half-open (default: 5 minutes)
}

type circuitEntry struct {
    state               circuitState
    consecutiveFailures int
    lastFailureTime     time.Time
    openedAt            time.Time
}

var automationCircuit *automationCircuitBreaker

func newAutomationCircuitBreaker(maxFailures int, cooldown time.Duration) *automationCircuitBreaker {
    if maxFailures <= 0 {
        maxFailures = 5
    }
    if cooldown <= 0 {
        cooldown = 5 * time.Minute
    }
    return &automationCircuitBreaker{
        states:           make(map[string]*circuitEntry),
        maxFailures:      maxFailures,
        cooldownDuration: cooldown,
    }
}

func (cb *automationCircuitBreaker) getOrCreate(ruleName string) *circuitEntry {
    entry, ok := cb.states[ruleName]
    if !ok {
        entry = &circuitEntry{state: circuitClosed}
        cb.states[ruleName] = entry
    }
    return entry
}

// AllowExecution checks if the circuit for a rule allows execution.
func (cb *automationCircuitBreaker) AllowExecution(ruleName string) bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    entry := cb.getOrCreate(ruleName)

    switch entry.state {
    case circuitClosed:
        return true
    case circuitOpen:
        if time.Since(entry.openedAt) >= cb.cooldownDuration {
            entry.state = circuitHalfOpen
            return true
        }
        return false
    case circuitHalfOpen:
        // In half-open state, only one request is allowed through.
        // The next call to RecordSuccess or RecordFailure will transition.
        // Block additional requests until the probe completes.
        return false
    default:
        return true
    }
}

// RecordSuccess records a successful execution, resetting failure count.
func (cb *automationCircuitBreaker) RecordSuccess(ruleName string) {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    entry := cb.getOrCreate(ruleName)
    entry.state = circuitClosed
    entry.consecutiveFailures = 0
}

// RecordFailure records a failure, potentially opening the circuit.
// Returns true if the circuit just transitioned to open state.
func (cb *automationCircuitBreaker) RecordFailure(ruleName string) bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    entry := cb.getOrCreate(ruleName)
    entry.consecutiveFailures++
    entry.lastFailureTime = time.Now()

    switch entry.state {
    case circuitClosed:
        if entry.consecutiveFailures >= cb.maxFailures {
            entry.state = circuitOpen
            entry.openedAt = time.Now()
            log.Printf("Automation circuit opened for rule %q after %d consecutive failures", ruleName, entry.consecutiveFailures)
            return true
        }
    case circuitHalfOpen:
        // Probe failed, reopen.
        entry.state = circuitOpen
        entry.openedAt = time.Now()
        log.Printf("Automation circuit re-opened for rule %q after half-open probe failure", ruleName)
        return true
    }
    return false
}

// GetState returns the current circuit state for a rule.
func (cb *automationCircuitBreaker) GetState(ruleName string) circuitState {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    entry := cb.getOrCreate(ruleName)

    // Transparently transition open -> half-open if cooldown elapsed.
    if entry.state == circuitOpen && time.Since(entry.openedAt) >= cb.cooldownDuration {
        entry.state = circuitHalfOpen
    }
    return entry.state
}

func setAutomationCircuitBreaker(cb *automationCircuitBreaker) {
    automationCircuit = cb
}

func init() {
    automationRateLimiter = ratelimit.NewSlidingWindowRateLimiter()
}

func setAutomationRateLimit(limit int) {
    automationRateMu.Lock()
    defer automationRateMu.Unlock()
    automationRateLimitHour = limit
}

func getAutomationRateLimit() int {
    automationRateMu.RLock()
    defer automationRateMu.RUnlock()
    return automationRateLimitHour
}

func setAutomationBurstLimit(limit int) {
    automationRateMu.Lock()
    defer automationRateMu.Unlock()
    automationBurstLimitMinute = limit
}

func getAutomationBurstLimit() int {
    automationRateMu.RLock()
    defer automationRateMu.RUnlock()
    return automationBurstLimitMinute
}

func setMaxAutomationDepth(d int) {
    maxAutomationDepthMu.Lock()
    defer maxAutomationDepthMu.Unlock()
    maxAutomationDepthVal = d
}

func getMaxAutomationDepth() int {
    maxAutomationDepthMu.RLock()
    defer maxAutomationDepthMu.RUnlock()
    return maxAutomationDepthVal
}

func setAutomationRules(rules []automationRule) {
    automationRulesMu.Lock()
    defer automationRulesMu.Unlock()
    if len(rules) == 0 {
        automationRules = nil
        return
    }
    automationRules = append([]automationRule(nil), rules...)
}

func getAutomationRules() []automationRule {
    automationRulesMu.RLock()
    defer automationRulesMu.RUnlock()
    if len(automationRules) == 0 {
        return nil
    }
    return append([]automationRule(nil), automationRules...)
}

func parseAutomationRules(raw string) ([]automationRule, error) {
    trimmed := strings.TrimSpace(raw)
    if trimmed == "" {
        return nil, nil
    }

    var rules []automationRule
    if err := json.Unmarshal([]byte(trimmed), &rules); err != nil {
        return nil, fmt.Errorf("invalid COCO_AUTOMATION_RULES: %w", err)
    }

    for i := range rules {
        normalized, err := normalizeAutomationRule(rules[i])
        if err != nil {
            return nil, fmt.Errorf("invalid COCO_AUTOMATION_RULES[%d]: %w", i, err)
        }
        rules[i] = normalized
    }

    return rules, nil
}

func normalizeAutomationRule(rule automationRule) (automationRule, error) {
    rule.Name = strings.TrimSpace(rule.Name)
    rule.Trigger = strings.ToLower(strings.TrimSpace(rule.Trigger))
    if rule.Trigger == "" {
        return rule, fmt.Errorf("trigger is required")
    }
    allowedTriggers := map[string]bool{
        "task.completed":       true,
        "task.failed":          true,
        "run.failed":           true,
        "lease.expired":        true,
        "repo.changed":         true,
        "repo.scanned":         true,
        "context.invalidated":  true,
        "memory.created":       true,
        "dependency.unblocked": true,
        "project.idle":         true,
    }
    if !allowedTriggers[rule.Trigger] {
        return rule, fmt.Errorf("unsupported trigger %q", rule.Trigger)
    }
    if len(rule.Actions) == 0 {
        return rule, fmt.Errorf("actions cannot be empty")
    }

    for i := range rule.Actions {
        action := rule.Actions[i]
        action.Type = strings.ToLower(strings.TrimSpace(action.Type))
        if action.Type == "" {
            return rule, fmt.Errorf("actions[%d].type is required", i)
        }
        if action.Type != "create_task" {
            return rule, fmt.Errorf("actions[%d].type must be create_task", i)
        }

        action.Task.Instructions = strings.TrimSpace(action.Task.Instructions)
        if action.Task.Instructions == "" {
            return rule, fmt.Errorf("actions[%d].task.instructions is required", i)
        }

        if action.Task.Type != nil {
            trimmedType := strings.ToUpper(strings.TrimSpace(*action.Task.Type))
            if trimmedType != "" {
                if _, ok := v2TaskListTypeFilter[trimmedType]; !ok {
                    return rule, fmt.Errorf("actions[%d].task.type is invalid", i)
                }
                action.Task.Type = &trimmedType
            } else {
                action.Task.Type = nil
            }
        }

        if action.Task.Priority != nil && *action.Task.Priority < 0 {
            return rule, fmt.Errorf("actions[%d].task.priority must be non-negative", i)
        }

        action.Task.Tags = normalizeAutomationTags(action.Task.Tags)

        if action.Task.Parent != nil {
            parentValue := strings.ToLower(strings.TrimSpace(*action.Task.Parent))
            if parentValue == "" {
                action.Task.Parent = nil
            } else if parentValue != "completed" && parentValue != "none" {
                return rule, fmt.Errorf("actions[%d].task.parent must be completed or none", i)
            } else {
                action.Task.Parent = &parentValue
            }
        }

        rule.Actions[i] = action
    }

    return rule, nil
}

func normalizeAutomationTags(tags []string) []string {
    if len(tags) == 0 {
        return nil
    }
    normalized := make([]string, 0, len(tags))
    for _, tag := range tags {
        trimmed := strings.TrimSpace(tag)
        if trimmed == "" {
            continue
        }
        normalized = append(normalized, trimmed)
    }
    if len(normalized) == 0 {
        return nil
    }
    return normalized
}

func isAutomationRuleEnabled(rule automationRule) bool {
    if rule.Enabled == nil {
        return true
    }
    return *rule.Enabled
}

func isAutomationBlockedByPolicies(db *sql.DB, projectID string) (bool, string, error) {
    policies, _, err := ListPoliciesByProject(db, projectID, nil, 0, 0, "created_at", "asc")
    if err != nil {
        return false, "", err
    }

    for _, policy := range policies {
        if !policy.Enabled {
            continue
        }
        if blocked, reason := policyBlocksAutomation(policy.Rules); blocked {
            emitPolicyDeniedEvent(db, projectID, "automation", reason)
            return true, reason, nil
        }
    }

    return false, "", nil
}

func policyBlocksAutomation(rules []PolicyRule) (bool, string) {
    if len(rules) == 0 {
        return false, ""
    }

    for _, rule := range rules {
        rawType, ok := rule["type"].(string)
        if !ok {
            continue
        }
        if strings.ToLower(strings.TrimSpace(rawType)) != "automation.block" {
            continue
        }

        reason, _ := rule["reason"].(string)
        return true, strings.TrimSpace(reason)
    }

    return false, ""
}

func isCompletionBlockedByPolicies(db *sql.DB, projectID string) (bool, string, error) {
    policies, _, err := ListPoliciesByProject(db, projectID, nil, 0, 0, "created_at", "asc")
    if err != nil {
        return false, "", err
    }

    for _, policy := range policies {
        if !policy.Enabled {
            continue
        }
        if blocked, reason := policyBlocksCompletion(policy.Rules); blocked {
            emitPolicyDeniedEvent(db, projectID, "task.complete", reason)
            return true, reason, nil
        }
    }

    return false, "", nil
}

func policyBlocksCompletion(rules []PolicyRule) (bool, string) {
    if len(rules) == 0 {
        return false, ""
    }

    for _, rule := range rules {
        rawType, ok := rule["type"].(string)
        if !ok {
            continue
        }
        if strings.ToLower(strings.TrimSpace(rawType)) != "completion.block" {
            continue
        }

        reason, _ := rule["reason"].(string)
        return true, strings.TrimSpace(reason)
    }

    return false, ""
}

func isTaskCreateBlockedByPolicies(db *sql.DB, projectID string) (bool, string, error) {
    policies, _, err := ListPoliciesByProject(db, projectID, nil, 0, 0, "created_at", "asc")
    if err != nil {
        return false, "", err
    }

    for _, policy := range policies {
        if !policy.Enabled {
            continue
        }
        if blocked, reason := policyBlocksTaskCreate(policy.Rules); blocked {
            emitPolicyDeniedEvent(db, projectID, "task.create", reason)
            return true, reason, nil
        }
    }

    return false, "", nil
}

func isTaskUpdateBlockedByPolicies(db *sql.DB, projectID string) (bool, string, error) {
    policies, _, err := ListPoliciesByProject(db, projectID, nil, 0, 0, "created_at", "asc")
    if err != nil {
        return false, "", err
    }

    for _, policy := range policies {
        if !policy.Enabled {
            continue
        }
        if blocked, reason := policyBlocksTaskUpdate(policy.Rules); blocked {
            emitPolicyDeniedEvent(db, projectID, "task.update", reason)
            return true, reason, nil
        }
    }

    return false, "", nil
}

func policyBlocksTaskUpdate(rules []PolicyRule) (bool, string) {
    if len(rules) == 0 {
        return false, ""
    }

    for _, rule := range rules {
        rawType, ok := rule["type"].(string)
        if !ok {
            continue
        }
        if strings.ToLower(strings.TrimSpace(rawType)) != "task.update.block" {
            continue
        }

        reason, _ := rule["reason"].(string)
        return true, strings.TrimSpace(reason)
    }

    return false, ""
}

func isTaskDeleteBlockedByPolicies(db *sql.DB, projectID string) (bool, string, error) {
    policies, _, err := ListPoliciesByProject(db, projectID, nil, 0, 0, "created_at", "asc")
    if err != nil {
        return false, "", err
    }

    for _, policy := range policies {
        if !policy.Enabled {
            continue
        }
        if blocked, reason := policyBlocksTaskDelete(policy.Rules); blocked {
            emitPolicyDeniedEvent(db, projectID, "task.delete", reason)
            return true, reason, nil
        }
    }

    return false, "", nil
}

func policyBlocksTaskDelete(rules []PolicyRule) (bool, string) {
    if len(rules) == 0 {
        return false, ""
    }

    for _, rule := range rules {
        rawType, ok := rule["type"].(string)
        if !ok {
            continue
        }
        if strings.ToLower(strings.TrimSpace(rawType)) != "task.delete.block" {
            continue
        }

        reason, _ := rule["reason"].(string)
        return true, strings.TrimSpace(reason)
    }

    return false, ""
}

func policyBlocksTaskCreate(rules []PolicyRule) (bool, string) {
    if len(rules) == 0 {
        return false, ""
    }

    for _, rule := range rules {
        rawType, ok := rule["type"].(string)
        if !ok {
            continue
        }
        if strings.ToLower(strings.TrimSpace(rawType)) != "task.create.block" {
            continue
        }

        reason, _ := rule["reason"].(string)
        return true, strings.TrimSpace(reason)
    }

    return false, ""
}

// emitPolicyDeniedEvent emits a policy.denied event for audit trail.
func emitPolicyDeniedEvent(db *sql.DB, projectID, action, reason string) {
    CreateEvent(db, projectID, "policy.denied", "policy", action, map[string]interface{}{
        "action": action,
        "reason": reason,
    })
}

func processAutomationEvent(db *sql.DB, event Event) {
    // Route system-level events to built-in workers first.
    switch event.Kind {
    case "run.failed":
        processRunFailedEvent(db, event)
    case "task.failed":
        processRunFailedEvent(db, event)
    case "task.completed":
        processTaskCompletedDependencies(db, event)
    case "repo.changed":
        processRepoChangedEvent(db, event)
    case "repo.scanned":
        processRepoScannedEvent(db, event)
    case "context.invalidated":
        processContextInvalidatedEvent(db, event)
    case "memory.created":
        processMemoryCreatedEvent(db, event)
    case "lease.expired":
        processLeaseExpiredEvent(db, event)
    case "project.idle":
        processProjectIdleEvent(db, event)
    }

    // Route all events (including above) through user-defined automation rules.
    allowedForRules := map[string]bool{
        "task.completed":       true,
        "task.failed":          true,
        "run.failed":           true,
        "lease.expired":        true,
        "repo.changed":         true,
        "repo.scanned":         true,
        "context.invalidated":  true,
        "memory.created":       true,
        "dependency.unblocked": true,
        "project.idle":         true,
    }
    if !allowedForRules[event.Kind] {
        return
    }

    rules := getAutomationRules()
    if len(rules) == 0 {
        return
    }

    taskID, ok := parseEventTaskID(event.EntityID)
    if !ok {
        if fromPayload, ok := taskIDFromPayload(event.Payload); ok {
            taskID = fromPayload
        } else {
            log.Printf("Automation skipped: unable to resolve task id for event %s", event.ID)
            return
        }
    }

    task, err := GetTaskV2(db, taskID)
    if err != nil {
        log.Printf("Automation skipped: failed to load task %d for event %s: %v", taskID, event.ID, err)
        return
    }

    parentDepth := task.AutomationDepth
    maxDepth := getMaxAutomationDepth()
    if parentDepth+1 > maxDepth {
        log.Printf("Automation blocked: recursion depth %d exceeds max %d for task %d", parentDepth+1, maxDepth, task.ID)
        CreateEvent(db, task.ProjectID, "automation.blocked", "automation", "", map[string]interface{}{
            "reason":  "recursion_depth_exceeded",
            "task_id": task.ID,
            "depth":   parentDepth + 1,
            "max":     maxDepth,
        })
		createAutomationEscalateTask(db, task, "Automation recursion depth exceeded (depth "+fmt.Sprintf("%d", parentDepth+1)+" > max "+fmt.Sprintf("%d", maxDepth)+")")
		return
	}

	// Check hourly rate limit
	hourlyLimit := getAutomationRateLimit()
	if !automationRateLimiter.CheckRateLimit(task.ProjectID, "", hourlyLimit, time.Hour) {
		log.Printf("Automation blocked: hourly rate limit (%d/hour) exceeded for project %s", hourlyLimit, task.ProjectID)
		CreateEvent(db, task.ProjectID, "automation.blocked", "automation", "", map[string]interface{}{
			"reason":  "rate_limit_exceeded",
			"task_id": task.ID,
			"limit":   hourlyLimit,
			"window":  "hour",
		})
		createAutomationReviewTask(db, task, fmt.Sprintf("Automation hourly rate limit exceeded (%d/hour) for project %s", hourlyLimit, task.ProjectID))
		return
	}

	// Check burst limit
	burstLimit := getAutomationBurstLimit()
	if !automationRateLimiter.CheckRateLimit(task.ProjectID, "burst", burstLimit, time.Minute) {
		log.Printf("Automation blocked: burst rate limit (%d/minute) exceeded for project %s", burstLimit, task.ProjectID)
		CreateEvent(db, task.ProjectID, "automation.blocked", "automation", "", map[string]interface{}{
			"reason":  "rate_limit_exceeded",
			"task_id": task.ID,
			"limit":   burstLimit,
			"window":  "minute",
		})
		createAutomationReviewTask(db, task, fmt.Sprintf("Automation burst rate limit exceeded (%d/minute) for project %s", burstLimit, task.ProjectID))
		return
	}

	blocked, reason, err := isAutomationBlockedByPolicies(db, task.ProjectID)
    if err != nil {
        log.Printf("Automation skipped: failed to evaluate policies for project %s: %v", task.ProjectID, err)
        return
    }
    if blocked {
        if reason == "" {
            log.Printf("Automation blocked by policy for project %s", task.ProjectID)
        } else {
            log.Printf("Automation blocked by policy for project %s: %s", task.ProjectID, reason)
        }
        CreateEvent(db, task.ProjectID, "automation.blocked", "automation", "", map[string]interface{}{
            "reason":  "policy_blocked",
            "task_id": task.ID,
            "detail":  reason,
        })
        return
    }

    templateData := buildAutomationTemplateData(event, task)
    createdCount := 0

    for _, rule := range rules {
        if !isAutomationRuleEnabled(rule) || rule.Trigger != event.Kind {
            continue
        }

        // Circuit breaker check
        if automationCircuit != nil && !automationCircuit.AllowExecution(rule.Name) {
            log.Printf("Automation circuit open for rule %q, skipping", rule.Name)
            CreateEvent(db, task.ProjectID, "automation.blocked", "automation", rule.Name, map[string]interface{}{
                "rule_name": rule.Name,
                "reason":    "circuit_open",
                "task_id":   task.ID,
            })
            continue
        }

        // Emit automation.triggered event
        CreateEvent(db, task.ProjectID, "automation.triggered", "automation", rule.Name, map[string]interface{}{
            "rule_name": rule.Name,
            "trigger":   rule.Trigger,
            "task_id":   task.ID,
            "event_id":  event.ID,
        })

        for _, action := range rule.Actions {
            if action.Type != "create_task" {
                continue
            }

            parentID := resolveAutomationParentID(action.Task.Parent, taskID)
            instructions := applyAutomationTemplate(action.Task.Instructions, templateData)
            if strings.TrimSpace(instructions) == "" {
                log.Printf("Automation skipped: empty instructions after template for rule %q", rule.Name)
                continue
            }

            var title *string
            if action.Task.Title != nil {
                renderedTitle := strings.TrimSpace(applyAutomationTemplate(*action.Task.Title, templateData))
                if renderedTitle != "" {
                    title = &renderedTitle
                }
            }

            var taskType *TaskType
            if action.Task.Type != nil {
                parsed := TaskType(strings.ToUpper(strings.TrimSpace(*action.Task.Type)))
                if parsed != "" {
                    taskType = &parsed
                }
            }

            tags := normalizeAutomationTags(action.Task.Tags)

            // Dedupe follow-up tasks by (project + parent + purpose/title).
            // Don't create duplicate follow-ups if one already exists and is non-terminal.
            if title != nil && parentID != nil {
                if isFollowUpDuplicate(db, task.ProjectID, *parentID, *title) {
                    log.Printf("Automation skipped: duplicate follow-up %q for parent %d in project %s", *title, *parentID, task.ProjectID)
                    continue
                }
            }

            child, err := CreateTaskV2WithMeta(db, instructions, task.ProjectID, parentID, title, taskType, action.Task.Priority, tags, nil)
            if err != nil {
                log.Printf("Automation failed: create task for rule %q failed: %v", rule.Name, err)
                if automationCircuit != nil {
                    justOpened := automationCircuit.RecordFailure(rule.Name)
                    if justOpened {
                        CreateEvent(db, task.ProjectID, "automation.circuit_opened", "automation", rule.Name, map[string]interface{}{
                            "rule_name": rule.Name,
                            "task_id":   task.ID,
                        })
                        createAutomationEscalateTask(db, task, fmt.Sprintf("Automation circuit breaker opened for rule %q after repeated failures", rule.Name))
                    }
                }
                continue
            }

            if automationCircuit != nil {
                automationCircuit.RecordSuccess(rule.Name)
            }

            childDepth := parentDepth + 1
            if err := setTaskAutomationDepth(db, child.ID, childDepth); err != nil {
                log.Printf("Automation warning: failed to set automation depth for task %d: %v", child.ID, err)
            }

            payload := buildTaskCreatedPayload(child, parentID)
            if _, err := CreateEvent(db, task.ProjectID, "task.created", "task", fmt.Sprintf("%d", child.ID), payload); err != nil {
                log.Printf("Automation failed: create task.created event for task %d: %v", child.ID, err)
                continue
            }
            createdCount++
        }
    }

    if createdCount > 0 {
        go broadcastUpdate(v1EventTypeTasks)
    }
}

func resolveAutomationParentID(parent *string, taskID int) *int {
    if parent == nil || *parent == "" || *parent == "completed" {
        return &taskID
    }
    if *parent == "none" {
        return nil
    }
    return &taskID
}

func buildAutomationTemplateData(event Event, task *TaskV2) map[string]string {
    data := map[string]string{
        "event_id":       event.ID,
        "event_kind":     event.Kind,
        "project_id":     event.ProjectID,
        "task_id":        fmt.Sprintf("%d", task.ID),
        "task_instructions": task.Instructions,
        "task_status_v1": string(task.StatusV1),
        "task_status_v2": string(task.StatusV2),
    }
    if task.Output != nil {
        data["task_output"] = *task.Output
    } else {
        data["task_output"] = ""
    }
    if task.Title != nil {
        data["task_title"] = *task.Title
    } else {
        data["task_title"] = ""
    }
    return data
}

func applyAutomationTemplate(input string, data map[string]string) string {
    if input == "" {
        return ""
    }
    replacer := strings.NewReplacer(
        "${event_id}", data["event_id"],
        "${event_kind}", data["event_kind"],
        "${project_id}", data["project_id"],
        "${task_id}", data["task_id"],
        "${task_instructions}", data["task_instructions"],
        "${task_output}", data["task_output"],
        "${task_status_v1}", data["task_status_v1"],
        "${task_status_v2}", data["task_status_v2"],
        "${task_title}", data["task_title"],
    )
    return replacer.Replace(input)
}

func buildTaskCreatedPayload(task *TaskV2, parentTaskID *int) map[string]interface{} {
    payload := map[string]interface{}{
        "task_id":      task.ID,
        "instructions": task.Instructions,
        "priority":     task.Priority,
    }
    if parentTaskID != nil {
        payload["parent_task_id"] = *parentTaskID
    }
    if task.Title != nil {
        payload["title"] = *task.Title
    }
    if task.Type != "" {
        payload["type"] = task.Type
    }
    if len(task.Tags) > 0 {
        payload["tags"] = task.Tags
    }
    return payload
}

// ============================================================================
// Automation Emission Dedupe
// ============================================================================

// Default emission window is 5 minutes (300 seconds)
const defaultEmissionWindowSeconds = 300

var emissionWindowSeconds = defaultEmissionWindowSeconds
var emissionWindowMu sync.RWMutex

// SetEmissionWindow sets the dedupe window in seconds.
func SetEmissionWindow(seconds int) {
    emissionWindowMu.Lock()
    defer emissionWindowMu.Unlock()
    if seconds <= 0 {
        seconds = defaultEmissionWindowSeconds
    }
    emissionWindowSeconds = seconds
}

// GetEmissionWindow returns the current dedupe window in seconds.
func GetEmissionWindow() int {
    emissionWindowMu.RLock()
    defer emissionWindowMu.RUnlock()
    return emissionWindowSeconds
}

// computeEmissionDedupeKey generates a dedupe key for the given project, kind, and time window.
// Key format: sha256(projectID + ":" + kind + ":" + windowNumber)
func computeEmissionDedupeKey(projectID, kind string, windowSeconds int) string {
    nowUnix := time.Now().Unix()
    window := nowUnix / int64(windowSeconds)
    raw := fmt.Sprintf("%s:%s:%d", projectID, kind, window)
    hash := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(hash[:])
}

// TryRecordEmission attempts to insert an emission record with the given dedupe key.
// Returns true if the emission was recorded (i.e., this is the first emission in the window).
// Returns false if a duplicate key exists (emission already recorded in this window).
// The taskID parameter is optional and can be nil.
func TryRecordEmission(db *sql.DB, projectID, kind string, taskID *int) (bool, error) {
    windowSeconds := GetEmissionWindow()
    dedupeKey := computeEmissionDedupeKey(projectID, kind, windowSeconds)
    nowUnix := time.Now().Unix()

    var taskIDValue interface{}
    if taskID != nil {
        taskIDValue = *taskID
    } else {
        taskIDValue = nil
    }

    // Use INSERT OR IGNORE to atomically check and insert
    result, err := db.Exec(`
        INSERT OR IGNORE INTO automation_emissions (dedupe_key, project_id, kind, task_id, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, dedupeKey, projectID, kind, taskIDValue, nowUnix)
    if err != nil {
        return false, fmt.Errorf("failed to record emission: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return false, fmt.Errorf("failed to check rows affected: %w", err)
    }

    // If rowsAffected == 1, the insert succeeded (first emission in window)
    // If rowsAffected == 0, the key already existed (duplicate)
    return rowsAffected == 1, nil
}

// CheckEmissionAllowed checks if an emission is allowed without recording it.
// Returns true if no emission exists for this project/kind in the current window.
func CheckEmissionAllowed(db *sql.DB, projectID, kind string) (bool, error) {
    windowSeconds := GetEmissionWindow()
    dedupeKey := computeEmissionDedupeKey(projectID, kind, windowSeconds)

    var count int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM automation_emissions WHERE dedupe_key = ?
    `, dedupeKey).Scan(&count)
    if err != nil {
        return false, fmt.Errorf("failed to check emission: %w", err)
    }

    return count == 0, nil
}

// CleanupOldEmissions removes emission records older than the given age.
// Recommended to run periodically (e.g., hourly) to prevent table bloat.
func CleanupOldEmissions(db *sql.DB, maxAgeSeconds int64) (int64, error) {
    cutoff := time.Now().Unix() - maxAgeSeconds

    result, err := db.Exec(`
        DELETE FROM automation_emissions WHERE created_at < ?
    `, cutoff)
    if err != nil {
        return 0, fmt.Errorf("failed to cleanup emissions: %w", err)
    }

    return result.RowsAffected()
}

// createAutomationEscalateTask creates an ESCALATE task tagged needs_human when
// automation governance detects repeated failures or recursion limits.
func createAutomationEscalateTask(db *sql.DB, parentTask *TaskV2, reason string) {
    title := "Automation Escalation: Review Required"
    instructions := fmt.Sprintf("Automation escalation for task %d in project %s.\n\nReason: %s\n\nPlease review the automation configuration and resolve the underlying issue.", parentTask.ID, parentTask.ProjectID, reason)
    taskType := "REVIEW"
    tags := []string{"needs_human", "escalation", "auto"}
    parentID := parentTask.ID

    child, err := CreateTaskV2WithMeta(db, instructions, parentTask.ProjectID, &parentID, &title, (*TaskType)(&taskType), nil, tags, nil)
    if err != nil {
        log.Printf("Automation escalation: failed to create ESCALATE task: %v", err)
        return
    }
    CreateEvent(db, parentTask.ProjectID, "task.created", "task", fmt.Sprintf("%d", child.ID), map[string]interface{}{
        "task_id":      child.ID,
        "instructions": instructions,
        "tags":         tags,
        "auto":         true,
        "escalation":   true,
    })
    go broadcastUpdate(v1EventTypeTasks)
}

// createAutomationReviewTask creates a REVIEW task tagged needs_human when
// automation rate limits or quotas are exceeded.
func createAutomationReviewTask(db *sql.DB, parentTask *TaskV2, reason string) {
    title := "Automation Review: Limits Exceeded"
    instructions := fmt.Sprintf("Automation limits exceeded for task %d in project %s.\n\nReason: %s\n\nPlease review automation rules and adjust limits if needed.", parentTask.ID, parentTask.ProjectID, reason)
    taskType := "REVIEW"
    tags := []string{"needs_human", "review", "auto"}
    parentID := parentTask.ID

    child, err := CreateTaskV2WithMeta(db, instructions, parentTask.ProjectID, &parentID, &title, (*TaskType)(&taskType), nil, tags, nil)
    if err != nil {
        log.Printf("Automation review: failed to create REVIEW task: %v", err)
        return
    }
    CreateEvent(db, parentTask.ProjectID, "task.created", "task", fmt.Sprintf("%d", child.ID), map[string]interface{}{
        "task_id":      child.ID,
        "instructions": instructions,
        "tags":         tags,
        "auto":         true,
    })
    go broadcastUpdate(v1EventTypeTasks)
}

// ============================================================================
// Built-in Event Workers (Phase 4: new event trigger handlers)
// ============================================================================

// maxFollowUpDepth is the maximum number of automatic follow-up tasks that
// can be created when a run fails. This prevents infinite investigation chains.
const maxFollowUpDepth = 3

// processRunFailedEvent handles run.failed events by:
// 1. Storing a structured failure summary as a failure_pattern memory.
// 2. Proposing a follow-up analysis task so agents can investigate and retry.
func processRunFailedEvent(db *sql.DB, event Event) {
    // Resolve task_id from event payload or entity_id.
    taskID, ok := parseEventTaskID(event.EntityID)
    if !ok {
        if fromPayload, ok := taskIDFromPayload(event.Payload); ok {
            taskID = fromPayload
        } else {
            return
        }
    }

    task, err := GetTaskV2(db, taskID)
    if err != nil {
        log.Printf("run.failed worker: failed to load task %d: %v", taskID, err)
        return
    }

    errMsg := ""
    if v, ok := event.Payload["error"]; ok {
        if s, ok := v.(string); ok {
            errMsg = s
        }
    }

    // Use task title if available, otherwise fall back to task ID.
    taskLabel := fmt.Sprintf("task %d", taskID)
    if task.Title != nil && *task.Title != "" {
        taskLabel = *task.Title
    }

    // --- T168-6: Store structured failure summary as failure_pattern memory ---
    errorType := classifyErrorType(errMsg)
    memoryKey := "failure_" + errorType
    var filesTouched []string
    if v, ok := event.Payload["files_touched"]; ok {
        if files, ok := v.([]interface{}); ok {
            for _, f := range files {
                if s, ok := f.(string); ok {
                    filesTouched = append(filesTouched, s)
                }
            }
        }
    }
    memoryValue := map[string]interface{}{
        "task_title":     taskLabel,
        "error_message":  errMsg,
        "files_touched":  filesTouched,
        "timestamp":      nowISO(),
    }
    scope := "failure_pattern"
    if _, memErr := CreateMemory(db, task.ProjectID, scope, memoryKey, memoryValue, nil); memErr != nil {
        log.Printf("run.failed worker: failed to store failure memory for task %d: %v", taskID, memErr)
    }
    // --------------------------------------------------------------------------

    // Depth guard: don't create investigation tasks beyond maxFollowUpDepth.
    if task.AutomationDepth >= maxFollowUpDepth {
        log.Printf("run.failed worker: follow-up depth %d >= max %d for task %d, skipping", task.AutomationDepth, maxFollowUpDepth, taskID)
        return
    }

    // Cooldown dedup: only one follow-up per task per emission window.
    kind := fmt.Sprintf("run.failed.followup.%d", taskID)
    recorded, err := TryRecordEmission(db, task.ProjectID, kind, &taskID)
    if err != nil || !recorded {
        return
    }

    title := fmt.Sprintf("Investigate failure: %s", taskLabel)

    // Dedupe follow-up: don't create if a non-terminal task with same title+parent exists.
    if isFollowUpDuplicate(db, task.ProjectID, taskID, title) {
        log.Printf("run.failed worker: duplicate follow-up for task %d, skipping", taskID)
        return
    }

    instructions := fmt.Sprintf(
        "Task %d (%s) failed. Please analyse the failure and propose a fix or retry.\n\nTask instructions:\n%s\n\nError:\n%s",
        taskID, taskLabel, task.Instructions, errMsg,
    )
    taskType := TaskTypeAnalyze
    tags := []string{"auto", "failure-analysis"}

    child, err := CreateTaskV2WithMeta(db, instructions, task.ProjectID, &taskID, &title, &taskType, nil, tags, nil)
    if err != nil {
        log.Printf("run.failed worker: failed to create analysis task: %v", err)
        return
    }

    // Set follow-up depth = parent depth + 1 to enforce chain limit.
    childDepth := task.AutomationDepth + 1
    if err := setTaskAutomationDepth(db, child.ID, childDepth); err != nil {
        log.Printf("run.failed worker: failed to set automation depth for child task %d: %v", child.ID, err)
    }

    CreateEvent(db, task.ProjectID, "task.created", "task", fmt.Sprintf("%d", child.ID), map[string]interface{}{
        "task_id":        child.ID,
        "parent_task_id": taskID,
        "auto":           true,
        "trigger":        "run.failed",
    })
    go broadcastUpdate(v1EventTypeTasks)
}

// classifyErrorType derives a short error type slug from an error message.
// Used as part of the memory key when storing failure patterns.
func classifyErrorType(errMsg string) string {
    lower := strings.ToLower(errMsg)
    switch {
    case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
        return "timeout"
    case strings.Contains(lower, "compile") || strings.Contains(lower, "syntax error") || strings.Contains(lower, "build failed"):
        return "compilation_error"
    case strings.Contains(lower, "assertion") || strings.Contains(lower, "expected") || strings.Contains(lower, "assert"):
        return "assertion_failure"
    case strings.Contains(lower, "connection refused") || strings.Contains(lower, "network") || strings.Contains(lower, "dial"):
        return "network_error"
    case strings.Contains(lower, "permission denied") || strings.Contains(lower, "access denied"):
        return "permission_error"
    case strings.Contains(lower, "not found") || strings.Contains(lower, "no such file"):
        return "not_found"
    default:
        return "runtime_error"
    }
}

// processRepoChangedEvent handles repo.changed events by:
// 1. Persisting changed files to repo_files table via UpsertRepoFile
// 2. Emitting a context.invalidated event so dependent workers can react.
func processRepoChangedEvent(db *sql.DB, event Event) {
    if event.ProjectID == "" {
        return
    }

    // Persist each changed file to the repo_files table.
    if files, ok := event.Payload["changed_files"]; ok {
        switch v := files.(type) {
        case []interface{}:
            for _, item := range v {
                path, ok := item.(string)
                if !ok || path == "" {
                    continue
                }
                now := nowISO()
                rf := RepoFile{
                    ProjectID:    event.ProjectID,
                    Path:         path,
                    LastModified: &now,
                }
                if _, err := UpsertRepoFile(db, rf); err != nil {
                    log.Printf("repo.changed worker: failed to upsert repo file %q: %v", path, err)
                }
            }
        case []string:
            for _, path := range v {
                if path == "" {
                    continue
                }
                now := nowISO()
                rf := RepoFile{
                    ProjectID:    event.ProjectID,
                    Path:         path,
                    LastModified: &now,
                }
                if _, err := UpsertRepoFile(db, rf); err != nil {
                    log.Printf("repo.changed worker: failed to upsert repo file %q: %v", path, err)
                }
            }
        }
    }

    // Dedup: one repo.scanned emission per project per window.
    // repo.changed → repo.scanned → context.invalidated is the canonical pipeline.
    kind := "repo.scanned"
    recorded, err := TryRecordEmission(db, event.ProjectID, kind, nil)
    if err != nil || !recorded {
        return
    }

    payload := map[string]interface{}{
        "project_id":      event.ProjectID,
        "trigger":         "repo.changed",
        "source_event_id": event.ID,
    }
    if v, ok := event.Payload["changed_files"]; ok {
        payload["changed_files"] = v
    }

    // Perform a real filesystem scan for richer metadata if project has a workdir.
    proj, projErr := GetProject(db, event.ProjectID)
    if projErr == nil && proj != nil && proj.Workdir != "" {
        scannedFiles, scanErr := ScanProjectFiles(event.ProjectID, proj.Workdir)
        if scanErr == nil {
            // Fetch existing repo files for diff computation.
            existing, _, _ := ListRepoFiles(db, event.ProjectID, ListRepoFilesOpts{})
            existingMap := make(map[string]RepoFile, len(existing))
            for _, f := range existing {
                existingMap[f.Path] = f
            }

            var newFiles, modifiedFiles, deletedFiles []string
            scannedMap := make(map[string]bool, len(scannedFiles))
            langCounts := make(map[string]int)
            var totalSize int64
            for _, sf := range scannedFiles {
                scannedMap[sf.Path] = true
                if sf.Language != nil {
                    langCounts[*sf.Language]++
                }
                if sf.SizeBytes != nil {
                    totalSize += *sf.SizeBytes
                }
                if ef, exists := existingMap[sf.Path]; !exists {
                    newFiles = append(newFiles, sf.Path)
                } else if sf.ContentHash != nil && ef.ContentHash != nil && *sf.ContentHash != *ef.ContentHash {
                    modifiedFiles = append(modifiedFiles, sf.Path)
                }
                // Upsert scanned file into repo_files table.
                if _, uErr := UpsertRepoFile(db, sf); uErr != nil {
                    log.Printf("repo.changed worker: scan upsert %q: %v", sf.Path, uErr)
                }
            }
            for _, ef := range existing {
                if !scannedMap[ef.Path] {
                    deletedFiles = append(deletedFiles, ef.Path)
                    _ = DeleteRepoFile(db, event.ProjectID, ef.Path)
                }
            }

            // Attach diff summary and richer metadata to the event payload.
            payload["scan_summary"] = map[string]interface{}{
                "total_files":      len(scannedFiles),
                "new_files":        len(newFiles),
                "modified_files":   len(modifiedFiles),
                "deleted_files":    len(deletedFiles),
                "total_size_bytes": totalSize,
                "languages":        langCounts,
            }
            if len(newFiles) > 0 {
                payload["new_files"] = newFiles
            }
            if len(modifiedFiles) > 0 {
                payload["modified_files"] = modifiedFiles
            }
            if len(deletedFiles) > 0 {
                payload["deleted_files"] = deletedFiles
            }
        } else {
            log.Printf("repo.changed worker: scan failed for project %s: %v", event.ProjectID, scanErr)
        }
    }

    if _, err := CreateEvent(db, event.ProjectID, "repo.scanned", "project", event.ProjectID, payload); err != nil {
        log.Printf("repo.changed worker: failed to emit repo.scanned: %v", err)
    }
}

// processContextInvalidatedEvent handles context.invalidated events by:
// 1. Marking all context packs for the project as stale
// 2. Auto-rebuilding context packs for queued tasks (up to 5)
// 3. Creating a context-refresh task for broader regeneration if needed.
func processContextInvalidatedEvent(db *sql.DB, event Event) {
    if event.ProjectID == "" {
        return
    }

    // Mark existing context packs as stale so agents know to regenerate them.
    if err := MarkContextPacksStale(db, event.ProjectID); err != nil {
        log.Printf("context.invalidated worker: failed to mark context packs stale: %v", err)
    }

    // Auto-rebuild context packs for queued tasks so they have fresh context at claim time.
    const maxRefresh = 5
    if tasks, _, err := ListTasksV2(db, event.ProjectID, "status_v2", string(TaskStatusQueued), "", "", "", maxRefresh, 0, "priority", "desc"); err == nil {
        for i := range tasks {
            task := &tasks[i]
            ctx := assembleContext(db, task)
            contents := map[string]interface{}{
                "memories":     ctx.Memories,
                "policies":     ctx.Policies,
                "dependencies": ctx.Dependencies,
                "repo_files":   ctx.RepoFiles,
            }
            if _, err := RefreshContextPack(db, event.ProjectID, task.ID, "Auto-refreshed by context.invalidated worker", contents); err != nil {
                log.Printf("context.invalidated worker: failed to refresh pack for task %d: %v", task.ID, err)
            }
        }
    }

    // Dedup: only queue one context refresh per project per window.
    kind := "context.refresh.queued"
    recorded, err := TryRecordEmission(db, event.ProjectID, kind, nil)
    if err != nil || !recorded {
        return
    }

    title := "Refresh project context"
    instructions := fmt.Sprintf(
        "The project context for project %s has been invalidated (trigger: %s).\n\nPlease:\n1. Scan the project working directory\n2. Regenerate context packs for queued/active tasks\n3. Update relevant memory entries with new file state",
        event.ProjectID, event.Payload["trigger"],
    )
    taskType := TaskTypeAnalyze
    tags := []string{"auto", "context-refresh"}

    child, err := CreateTaskV2WithMeta(db, instructions, event.ProjectID, nil, &title, &taskType, nil, tags, nil)
    if err != nil {
        log.Printf("context.invalidated worker: failed to create refresh task: %v", err)
        return
    }
    CreateEvent(db, event.ProjectID, "task.created", "task", fmt.Sprintf("%d", child.ID), map[string]interface{}{
        "task_id": child.ID,
        "auto":    true,
        "trigger": "context.invalidated",
    })
    go broadcastUpdate(v1EventTypeTasks)
}

// processLeaseExpiredEvent handles lease.expired events by re-queuing the task
// so it becomes available for another agent to claim.
func processLeaseExpiredEvent(db *sql.DB, event Event) {
    taskID, ok := parseEventTaskID(event.EntityID)
    if !ok {
        if fromPayload, ok := taskIDFromPayload(event.Payload); ok {
            taskID = fromPayload
        } else {
            log.Printf("lease.expired worker: cannot resolve task_id from event %s", event.ID)
            return
        }
    }

    task, err := GetTaskV2(db, taskID)
    if err != nil {
        log.Printf("lease.expired worker: failed to load task %d: %v", taskID, err)
        return
    }

    // Only re-queue tasks that are still in a claimable/active state.
    if task.StatusV2 != TaskStatusClaimed && task.StatusV2 != TaskStatusRunning {
        return
    }

    now := nowISO()
    _, err = db.Exec(
        "UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ? AND status_v2 IN (?, ?)",
        StatusNotPicked, TaskStatusQueued, now, taskID, string(TaskStatusClaimed), string(TaskStatusRunning),
    )
    if err != nil {
        log.Printf("lease.expired worker: failed to re-queue task %d: %v", taskID, err)
        return
    }

    CreateEvent(db, task.ProjectID, "task.requeued", "task", fmt.Sprintf("%d", taskID), map[string]interface{}{
        "task_id": taskID,
        "trigger": "lease.expired",
    })
    log.Printf("lease.expired worker: re-queued task %d", taskID)
    go broadcastUpdate(v1EventTypeTasks)
}

// processRepoScannedEvent handles repo.scanned events by walking the project
// workdir, updating repo_files with real metadata, and emitting context.invalidated.
func processRepoScannedEvent(db *sql.DB, event Event) {
    if event.ProjectID == "" {
        return
    }

    // Fetch project to get workdir.
    project, err := GetProject(db, event.ProjectID)
    if err != nil {
        log.Printf("repo.scanned worker: failed to fetch project %s: %v", event.ProjectID, err)
        return
    }

    workdir := project.Workdir
    if workdir == "" {
        log.Printf("repo.scanned worker: project %s has no workdir, skipping scan", event.ProjectID)
        // Still emit context.invalidated even without a scan.
        emitContextInvalidatedFromScan(db, event)
        return
    }

    // Validate workdir exists and is a directory.
    info, err := os.Stat(workdir)
    if err != nil || !info.IsDir() {
        log.Printf("repo.scanned worker: workdir %q not accessible: %v", workdir, err)
        emitContextInvalidatedFromScan(db, event)
        return
    }

    // Snapshot existing repo_files to compute diff summary.
    previousFiles := map[string]string{} // path → content_hash
    if existing, _, lErr := ListRepoFiles(db, event.ProjectID, ListRepoFilesOpts{Limit: 10000}); lErr == nil {
        for _, f := range existing {
            hash := ""
            if f.ContentHash != nil {
                hash = *f.ContentHash
            }
            previousFiles[f.Path] = hash
        }
    }

    // Walk the workdir, collecting file metadata.
    const maxFiles = 1000
    // Directories to skip entirely.
    skipDirs := map[string]bool{
        ".git": true, "node_modules": true, "__pycache__": true,
        ".venv": true, "vendor": true, "dist": true, ".next": true,
        "__MACOSX": true, ".idea": true, ".vscode": true,
    }

    var fileCount int
    var totalSize int64
    langCounts := map[string]int{}
    seenPaths := map[string]bool{}
    var filesAdded, filesModified int

    filepath.Walk(workdir, func(path string, fi os.FileInfo, walkErr error) error {
        if walkErr != nil {
            return nil // skip inaccessible paths
        }
        if fi.IsDir() {
            if skipDirs[fi.Name()] {
                return filepath.SkipDir
            }
            return nil
        }
        if fileCount >= maxFiles {
            return filepath.SkipAll
        }

        // Skip hidden files.
        if strings.HasPrefix(fi.Name(), ".") {
            return nil
        }

        fileCount++
        size := fi.Size()
        totalSize += size
        modTime := fi.ModTime().UTC().Format("2006-01-02T15:04:05.000000Z")

        // Detect language from extension.
        lang := detectLanguageFromExt(filepath.Ext(fi.Name()))
        if lang != "" {
            langCounts[lang]++
        }

        // Compute a lightweight content hash from size + mod time.
        contentHash := fmt.Sprintf("%d:%s", size, modTime)

        // Relative path from workdir.
        relPath, relErr := filepath.Rel(workdir, path)
        if relErr != nil {
            relPath = path
        }

        seenPaths[relPath] = true

        // Track diff: added vs modified.
        if prevHash, existed := previousFiles[relPath]; !existed {
            filesAdded++
        } else if prevHash != contentHash {
            filesModified++
        }

        // Upsert into repo_files table.
        rf := RepoFile{
            ProjectID:    event.ProjectID,
            Path:         relPath,
            ContentHash:  &contentHash,
            SizeBytes:    &size,
            Language:     strPtrIfNotEmpty(lang),
            LastModified: &modTime,
        }
        if _, uErr := UpsertRepoFile(db, rf); uErr != nil {
            log.Printf("repo.scanned worker: failed to upsert %q: %v", relPath, uErr)
        }

        return nil
    })

    // Detect deleted files: in previous snapshot but not seen during walk.
    var filesDeleted int
    for prevPath := range previousFiles {
        if !seenPaths[prevPath] {
            filesDeleted++
            _ = DeleteRepoFile(db, event.ProjectID, prevPath)
        }
    }

    // Build scan summary with diff.
    languages := make([]string, 0, len(langCounts))
    for lang := range langCounts {
        languages = append(languages, lang)
    }

    log.Printf("repo.scanned worker: scanned %d files (%d bytes) in %s for project %s [+%d ~%d -%d]",
        fileCount, totalSize, workdir, event.ProjectID, filesAdded, filesModified, filesDeleted)

    // Store scan summary (with diff) as a memory entry.
    summaryValue := map[string]interface{}{
        "file_count":     fileCount,
        "total_size":     totalSize,
        "languages":      languages,
        "files_added":    filesAdded,
        "files_modified": filesModified,
        "files_deleted":  filesDeleted,
        "scanned_at":     nowISO(),
    }
    CreateMemory(db, event.ProjectID, "repo", "repo_scan_summary", summaryValue, []string{"event:" + event.ID})

    // Emit context.invalidated so downstream workers refresh.
    emitContextInvalidatedFromScan(db, event)
}

// emitContextInvalidatedFromScan deduplicates and emits context.invalidated from a repo.scanned event.
func emitContextInvalidatedFromScan(db *sql.DB, event Event) {
    kind := "context.invalidated"
    recorded, err := TryRecordEmission(db, event.ProjectID, kind, nil)
    if err != nil || !recorded {
        return
    }
    payload := map[string]interface{}{
        "project_id":      event.ProjectID,
        "trigger":         "repo.scanned",
        "source_event_id": event.ID,
    }
    if _, err := CreateEvent(db, event.ProjectID, "context.invalidated", "project", event.ProjectID, payload); err != nil {
        log.Printf("repo.scanned worker: failed to emit context.invalidated: %v", err)
    }
}

// detectLanguageFromExt maps file extensions to language names.
func detectLanguageFromExt(ext string) string {
    switch strings.ToLower(ext) {
    case ".go":
        return "go"
    case ".py":
        return "python"
    case ".js":
        return "javascript"
    case ".ts":
        return "typescript"
    case ".jsx":
        return "javascript"
    case ".tsx":
        return "typescript"
    case ".rs":
        return "rust"
    case ".rb":
        return "ruby"
    case ".java":
        return "java"
    case ".c", ".h":
        return "c"
    case ".cpp", ".cc", ".cxx", ".hpp":
        return "cpp"
    case ".cs":
        return "csharp"
    case ".swift":
        return "swift"
    case ".kt":
        return "kotlin"
    case ".sql":
        return "sql"
    case ".sh", ".bash":
        return "shell"
    case ".md":
        return "markdown"
    case ".json":
        return "json"
    case ".yaml", ".yml":
        return "yaml"
    case ".toml":
        return "toml"
    case ".html", ".htm":
        return "html"
    case ".css":
        return "css"
    case ".xml":
        return "xml"
    default:
        return ""
    }
}

// strPtrIfNotEmpty returns a string pointer if the string is non-empty, nil otherwise.
func strPtrIfNotEmpty(s string) *string {
    if s == "" {
        return nil
    }
    return &s
}

// processMemoryCreatedEvent handles memory.created events by emitting
// context.invalidated when a high-relevance memory is created.
func processMemoryCreatedEvent(db *sql.DB, event Event) {
    if event.ProjectID == "" {
        return
    }
    // High-relevance scopes: convention, constraint, fix_pattern.
    scope, _ := event.Payload["scope"].(string)
    highRelevance := scope == "convention" || scope == "constraint" || scope == "fix_pattern"
    if !highRelevance {
        return
    }

    // Dedup: one context.invalidated per project per window.
    kind := "context.invalidated"
    recorded, err := TryRecordEmission(db, event.ProjectID, kind, nil)
    if err != nil || !recorded {
        return
    }
    if _, err := CreateEvent(db, event.ProjectID, "context.invalidated", "project", event.ProjectID, map[string]interface{}{
        "project_id": event.ProjectID,
        "trigger":    "memory.created",
        "scope":      scope,
        "source_event_id": event.ID,
    }); err != nil {
        log.Printf("memory.created worker: failed to emit context.invalidated: %v", err)
    }
}

// processProjectIdleEvent handles project.idle events by emitting a
// notification event so external systems can react (e.g. send alerts).
func processProjectIdleEvent(db *sql.DB, event Event) {
    if event.ProjectID == "" {
        return
    }
    // Dedup: one notification per project per window.
    kind := "project.idle.notified"
    recorded, err := TryRecordEmission(db, event.ProjectID, kind, nil)
    if err != nil || !recorded {
        return
    }
    CreateEvent(db, event.ProjectID, "notification", "project", event.ProjectID, map[string]interface{}{
        "project_id": event.ProjectID,
        "message":    fmt.Sprintf("Project %s has been idle with no active tasks.", event.ProjectID),
        "trigger":    "project.idle",
    })
}

// processTaskCompletedDependencies checks if completing a task unblocks any
// dependent tasks. For each task that was waiting on this one, if all its
// dependencies are now fulfilled, a dependency.unblocked event is emitted.
func processTaskCompletedDependencies(db *sql.DB, event Event) {
    completedTaskID, ok := parseEventTaskID(event.EntityID)
    if !ok {
        if fromPayload, ok := taskIDFromPayload(event.Payload); ok {
            completedTaskID = fromPayload
        } else {
            return
        }
    }

    dependents, err := GetTasksDependingOn(db, completedTaskID)
    if err != nil {
        log.Printf("dependency worker: GetTasksDependingOn(%d): %v", completedTaskID, err)
        return
    }

    for _, dep := range dependents {
        // Only consider tasks that are still waiting (QUEUED or not yet claimed).
        if dep.StatusV2.IsTerminal() || dep.StatusV2 == TaskStatusClaimed || dep.StatusV2 == TaskStatusRunning {
            continue
        }
        fulfilled, err := AreAllDependenciesFulfilled(db, dep.ID)
        if err != nil {
            log.Printf("dependency worker: AreAllDependenciesFulfilled(%d): %v", dep.ID, err)
            continue
        }
        if !fulfilled {
            continue
        }
        // All dependencies completed: emit dependency.unblocked event.
        CreateEvent(db, dep.ProjectID, "dependency.unblocked", "task", fmt.Sprintf("%d", dep.ID), map[string]interface{}{
            "task_id":             dep.ID,
            "unblocked_by_task_id": completedTaskID,
        })
        log.Printf("dependency worker: task %d unblocked by completed task %d", dep.ID, completedTaskID)
    }
}

// isFollowUpDuplicate checks if a non-terminal task already exists with the
// same project, parent, and title. This prevents automation from creating
// duplicate follow-up tasks (loop guard per Canvas 5.3).
func isFollowUpDuplicate(db *sql.DB, projectID string, parentID int, title string) bool {
    var count int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM tasks
        WHERE project_id = ?
          AND parent_task_id = ?
          AND title = ?
          AND status_v2 NOT IN (?, ?, ?)
    `, projectID, parentID, title,
        string(TaskStatusSucceeded), string(TaskStatusFailed), string(TaskStatusCancelled),
    ).Scan(&count)
    if err != nil {
        log.Printf("isFollowUpDuplicate: query error: %v", err)
        return false
    }
    return count > 0
}

