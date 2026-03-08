// Enhanced finalization service for centralized task completion/failure/cancellation.
// This service ensures that all completion paths (v1, v2, automation) produce
// identical database state and trigger consistent downstream actions.
package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// FinalizationService is the single canonical service for task finalization.
// All completion, failure, and cancellation paths (v1, v2, automation) must
// route through this service so that status transitions, run updates, lease
// releases, memory extraction, event emission, and automation hooks happen
// consistently.
type FinalizationService struct {
	DB *sql.DB
}

// Complete is the canonical SUCCESS path for a task.
func (svc *FinalizationService) Complete(taskID int, output *string) (*TaskV2, *CompletionRunSummary, error) {
	return CompleteTaskWithPayload(svc.DB, taskID, output)
}

// Fail is the canonical FAILURE path for a task.
func (svc *FinalizationService) Fail(taskID int, errMsg string) (*TaskV2, error) {
	return FailTaskWithError(svc.DB, taskID, errMsg)
}

// Cancel is the canonical CANCELLATION path for a task.
func (svc *FinalizationService) Cancel(taskID int, reason string) (*TaskV2, error) {
	return CancelTask(svc.DB, taskID, reason)
}

// CompletionPayload represents a structured completion result.
// Agents can submit either plain-text output or structured completion data.
type CompletionPayload struct {
	Summary      string                   `json:"summary,omitempty"`
	ChangesMade  []string                 `json:"changes_made,omitempty"`
	FilesTouched []string                 `json:"files_touched,omitempty"`
	CommandsRun  []string                 `json:"commands_run,omitempty"`
	TestsRun     []string                 `json:"tests_run,omitempty"`
	Risks        []string                 `json:"risks,omitempty"`
	NextTasks    []NextTaskProposal       `json:"next_tasks,omitempty"`
	Artifacts    []string                 `json:"artifacts,omitempty"`
}

// NextTaskProposal represents a task that should be created after this one completes.
type NextTaskProposal struct {
	Title        string   `json:"title"`
	Instructions string   `json:"instructions"`
	Type         *string  `json:"type,omitempty"`
	Priority     *int     `json:"priority,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// CompletionRunSummary represents extracted structured data from a task completion.
// This is persisted to enable memory extraction and planning.
type CompletionRunSummary struct {
	RunID        string    `json:"run_id"`
	TaskID       int       `json:"task_id"`
	Status       RunStatus `json:"status"`
	Summary      string    `json:"summary,omitempty"`
	ChangesMade  []string  `json:"changes_made,omitempty"`
	FilesTouched []string  `json:"files_touched,omitempty"`
	CommandsRun  []string  `json:"commands_run,omitempty"`
	TestsRun     []string  `json:"tests_run,omitempty"`
	Risks        []string  `json:"risks,omitempty"`
	CreatedAt    string    `json:"created_at"`
}

// CompleteTaskWithPayload is the canonical SUCCESS path.
// It accepts either plain-text output or a structured CompletionPayload.
// It:
//   1. Updates task status to SUCCEEDED
//   2. Updates latest run to SUCCEEDED
//   3. Extracts and stores structured summary
//   4. Releases lease
//   5. Extracts memory for future context
//   6. Emits task.completed event (triggers automation)
//   7. Broadcasts SSE
//
// If output is plain text, it's stored as-is. If it's JSON matching
// CompletionPayload struct, it's parsed and stored structurally.
func CompleteTaskWithPayload(database *sql.DB, taskID int, output *string) (*TaskV2, *CompletionRunSummary, error) {
	now := nowISO()

	// 1. Update task status (TOCTOU-safe: AND status != COMPLETE).
	result, err := database.Exec(
		"UPDATE tasks SET output = ?, status = ?, status_v2 = ?, updated_at = ? WHERE id = ? AND status != ?",
		nullString(output), StatusComplete, TaskStatusSucceeded, now, taskID, StatusComplete,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("task status update: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, nil, fmt.Errorf("task %d already completed", taskID)
	}

	// 2. Get latest run and update its status.
	latestRun, err := GetLatestRunByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get run for task %d: %v", taskID, err)
	}
	
	var summary *CompletionRunSummary
	if latestRun != nil {
		if err := UpdateRunStatus(database, latestRun.ID, RunStatusSucceeded, nil); err != nil {
			log.Printf("Warning: failed to update run %s: %v", latestRun.ID, err)
		}

		// 3. Extract structured summary from output
		summary = extractSummaryFromCompletion(latestRun.ID, taskID, output, now)
		if summary != nil {
			if err := storeSummary(database, summary); err != nil {
				log.Printf("Warning: failed to store summary for task %d: %v", taskID, err)
			}
		}
	}

	// 4. Release lease.
	lease, err := GetLeaseByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get lease for task %d: %v", taskID, err)
	} else if lease != nil {
		if _, _, err := ReleaseLease(database, lease.ID, "task_completed"); err != nil {
			log.Printf("Warning: failed to release lease for task %d: %v", taskID, err)
		}
	}

	// 5. Fetch task for response
	task, err := GetTaskV2(database, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch task: %w", err)
	}

	// 6. Extract memory from completion (learning loop)
	if summary != nil && summary.Summary != "" {
		if err := createMemoryFromSummary(database, task.ProjectID, taskID, summary); err != nil {
			log.Printf("Warning: failed to create memory from task %d: %v", taskID, err)
		}
	}

	// 7. Emit task.completed event (triggers automation).
	payload := map[string]interface{}{
		"task_id":   taskID,
		"status_v1": string(StatusComplete),
		"status_v2": string(TaskStatusSucceeded),
	}
	if output != nil {
		payload["output"] = *output
	}
	if summary != nil {
		payload["summary"] = summary.Summary
		if len(summary.ChangesMade) > 0 {
			payload["changes_made_count"] = len(summary.ChangesMade)
		}
		if len(summary.Risks) > 0 {
			payload["has_risks"] = true
		}
	}
	if _, err := CreateEvent(database, task.ProjectID, "task.completed", "task", fmt.Sprintf("%d", taskID), payload); err != nil {
		log.Printf("Warning: failed to emit task.completed event for task %d: %v", taskID, err)
	}

	// 8a. Emit repo.changed event if files were touched during this task.
	if summary != nil && len(summary.FilesTouched) > 0 {
		if _, err := CreateEvent(database, task.ProjectID, "repo.changed", "repo", task.ProjectID, map[string]interface{}{
			"task_id":       taskID,
			"changed_files": summary.FilesTouched,
			"trigger":       "task.completed",
		}); err != nil {
			log.Printf("Warning: failed to emit repo.changed event for task %d: %v", taskID, err)
		}
	}

	// 8b. SSE broadcast.
	go broadcastUpdate(v1EventTypeTasks)

	return task, summary, nil
}

// FailTaskWithError is the canonical FAILURE path.
// It:
//   1. Updates task status to FAILED
//   2. Updates latest run to FAILED with error message
//   3. Releases lease
//   4. Extracts memory from failure (learning opportunity)
//   5. Emits task.failed event (triggers automation)
//   6. Broadcasts SSE
func FailTaskWithError(database *sql.DB, taskID int, errMsg string) (*TaskV2, error) {
	now := nowISO()

	// 1. Update task status (TOCTOU-safe: AND status not already terminal).
	_, err := database.Exec(
		"UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ? AND status != ? AND status != ?",
		StatusFailed, TaskStatusFailed, now, taskID, StatusComplete, StatusFailed,
	)
	if err != nil {
		return nil, fmt.Errorf("task status update: %w", err)
	}

	// 2. Update latest run with error.
	latestRun, err := GetLatestRunByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get run for task %d: %v", taskID, err)
	} else if latestRun != nil {
		if err := UpdateRunStatus(database, latestRun.ID, RunStatusFailed, &errMsg); err != nil {
			log.Printf("Warning: failed to update run %s: %v", latestRun.ID, err)
		}
	}

	// 3. Release lease.
	lease, err := GetLeaseByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get lease for task %d: %v", taskID, err)
	} else if lease != nil {
		if _, _, err := ReleaseLease(database, lease.ID, "task_failed"); err != nil {
			log.Printf("Warning: failed to release lease for task %d: %v", taskID, err)
		}
	}

	// 4. Fetch task for response
	task, err := GetTaskV2(database, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch task: %w", err)
	}

	// 5. Extract error memory (learning opportunity)
	if err := createMemoryFromFailure(database, task.ProjectID, taskID, errMsg); err != nil {
		log.Printf("Warning: failed to create error memory for task %d: %v", taskID, err)
	}

	// 6. Emit task.failed event (triggers automation for retry/escalation).
	payload := map[string]interface{}{
		"task_id":   taskID,
		"status_v1": string(StatusFailed),
		"status_v2": string(TaskStatusFailed),
		"error":     errMsg,
	}
	if _, err := CreateEvent(database, task.ProjectID, "task.failed", "task", fmt.Sprintf("%d", taskID), payload); err != nil {
		log.Printf("Warning: failed to emit task.failed event for task %d: %v", taskID, err)
	}

	// 7. SSE broadcast.
	go broadcastUpdate(v1EventTypeTasks)

	return task, nil
}

// CancelTask is the canonical CANCELLATION path.
// It marks a task as cancelled (user-initiated or automation-triggered).
func CancelTask(database *sql.DB, taskID int, reason string) (*TaskV2, error) {
	now := nowISO()

	// Update task status to CANCELLED.
	_, err := database.Exec(
		"UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ? AND status != ?",
		StatusComplete, TaskStatusCancelled, now, taskID, StatusComplete,
	)
	if err != nil {
		return nil, fmt.Errorf("task status update: %w", err)
	}

	// Update run if active.
	latestRun, err := GetLatestRunByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get run for task %d: %v", taskID, err)
	} else if latestRun != nil && !latestRun.Status.IsTerminal() {
		if err := UpdateRunStatus(database, latestRun.ID, RunStatusCancelled, &reason); err != nil {
			log.Printf("Warning: failed to update run %s: %v", latestRun.ID, err)
		}
	}

	// Release lease.
	lease, err := GetLeaseByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get lease for task %d: %v", taskID, err)
	} else if lease != nil {
		if _, _, err := ReleaseLease(database, lease.ID, "task_cancelled"); err != nil {
			log.Printf("Warning: failed to release lease for task %d: %v", taskID, err)
		}
	}

	// Fetch task for response
	task, err := GetTaskV2(database, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch task: %w", err)
	}

	// Emit task.cancelled event.
	payload := map[string]interface{}{
		"task_id":   taskID,
		"status_v2": string(TaskStatusCancelled),
		"reason":    reason,
	}
	if _, err := CreateEvent(database, task.ProjectID, "task.cancelled", "task", fmt.Sprintf("%d", taskID), payload); err != nil {
		log.Printf("Warning: failed to emit task.cancelled event for task %d: %v", taskID, err)
	}

	// SSE broadcast.
	go broadcastUpdate(v1EventTypeTasks)

	return task, nil
}

// ===== Internal Helper Functions =====

// extractSummaryFromCompletion parses completion output and extracts structured summary.
func extractSummaryFromCompletion(runID string, taskID int, output *string, now string) *CompletionRunSummary {
	if output == nil || *output == "" {
		return nil
	}

	summary := &CompletionRunSummary{
		RunID:     runID,
		TaskID:    taskID,
		Status:    RunStatusSucceeded,
		CreatedAt: now,
	}

	// Try to parse as JSON (structured payload)
	var payload CompletionPayload
	if err := json.Unmarshal([]byte(*output), &payload); err == nil {
		// Successfully parsed as structured JSON
		summary.Summary = payload.Summary
		summary.ChangesMade = payload.ChangesMade
		summary.FilesTouched = payload.FilesTouched
		summary.CommandsRun = payload.CommandsRun
		summary.TestsRun = payload.TestsRun
		summary.Risks = payload.Risks
		return summary
	}

	// Fall back to treating entire output as summary
	summary.Summary = *output
	return summary
}

// storeSummary persists the run summary to the run_summaries table.
func storeSummary(database *sql.DB, summary *CompletionRunSummary) error {
	if summary == nil {
		return nil
	}
	changesMade, _ := marshalJSON(summary.ChangesMade)
	filesTouched, _ := marshalJSON(summary.FilesTouched)
	commandsRun, _ := marshalJSON(summary.CommandsRun)
	testsRun, _ := marshalJSON(summary.TestsRun)
	risks, _ := marshalJSON(summary.Risks)

	_, err := database.Exec(
		`INSERT INTO run_summaries (run_id, task_id, status, summary, changes_made, files_touched, commands_run, tests_run, risks, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		summary.RunID, summary.TaskID, string(summary.Status), summary.Summary,
		changesMade, filesTouched, commandsRun, testsRun, risks, summary.CreatedAt,
	)
	return err
}

// MemoryCandidate represents a potential memory to extract from a run summary.
type MemoryCandidate struct {
	Scope      string
	Key        string
	Confidence float64 // 0.0–1.0
	Value      map[string]interface{}
	SourceRefs []string
}

// createMemoryFromSummary creates Memory records from a successful completion.
// It extracts multiple memory candidates from the summary, scores them by
// confidence, redacts secrets, deduplicates by content hash, and persists
// with scope + tags + links to run/task.
func createMemoryFromSummary(database *sql.DB, projectID string, taskID int, summary *CompletionRunSummary) error {
	if summary == nil || summary.Summary == "" {
		return nil
	}

	cleaned := redactSecrets(summary.Summary)
	sourceRefs := []string{
		fmt.Sprintf("task:%d", taskID),
		fmt.Sprintf("run:%s", summary.RunID),
	}

	// Extract memory candidates of different types.
	candidates := extractMemoryCandidates(cleaned, taskID, summary, sourceRefs)

	// Sort by confidence descending.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	var firstErr error
	for _, c := range candidates {
		// Dedupe: skip if a memory with identical content hash already exists.
		contentHash := memoryContentHash(c.Value)
		c.Value["content_hash"] = contentHash
		c.Value["confidence"] = c.Confidence

		if isDuplicateMemory(database, projectID, c.Scope, contentHash) {
			continue
		}

		// B1: Skip creation if a recent memory with the same key already exists (<24h).
		if existing, err := GetMemory(database, projectID, c.Scope, c.Key); err == nil && existing != nil {
			cutoff := time.Now().Add(-24 * time.Hour)
			if t, parseErr := time.Parse(time.RFC3339Nano, existing.UpdatedAt); parseErr == nil && t.After(cutoff) {
				continue
			}
		}

		if _, err := CreateMemory(database, projectID, c.Scope, c.Key, c.Value, c.SourceRefs); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// extractMemoryCandidates produces memory candidates from a run summary.
// Types: fix_pattern, failure_pattern, convention, constraint, completion.
func extractMemoryCandidates(cleaned string, taskID int, summary *CompletionRunSummary, sourceRefs []string) []MemoryCandidate {
	var candidates []MemoryCandidate
	lower := strings.ToLower(cleaned)

	baseValue := func() map[string]interface{} {
		v := map[string]interface{}{
			"summary":       cleaned,
			"task_id":       taskID,
			"run_id":        summary.RunID,
			"changes_count": len(summary.ChangesMade),
			"files_touched": len(summary.FilesTouched),
			"has_risks":     len(summary.Risks) > 0,
			"timestamp":     summary.CreatedAt,
		}
		if len(summary.ChangesMade) > 0 {
			v["changes"] = summary.ChangesMade
		}
		if len(summary.FilesTouched) > 0 {
			v["files"] = summary.FilesTouched
		}
		if len(summary.Risks) > 0 {
			v["risks"] = summary.Risks
		}
		return v
	}

	// fix_pattern: if summary mentions fixing/resolving/patching
	if strings.Contains(lower, "fixed") || strings.Contains(lower, "resolved") ||
		strings.Contains(lower, "patched") || strings.Contains(lower, "corrected") ||
		strings.Contains(lower, "bug") {
		conf := 0.7
		if len(summary.ChangesMade) > 0 {
			conf = 0.85
		}
		candidates = append(candidates, MemoryCandidate{
			Scope:      "fix_pattern",
			Key:        fmt.Sprintf("fix_%d", taskID),
			Confidence: conf,
			Value:      baseValue(),
			SourceRefs: sourceRefs,
		})
	}

	// convention: if summary describes patterns, naming, style
	if strings.Contains(lower, "convention") || strings.Contains(lower, "pattern") ||
		strings.Contains(lower, "style") || strings.Contains(lower, "naming") ||
		strings.Contains(lower, "standard") {
		candidates = append(candidates, MemoryCandidate{
			Scope:      "convention",
			Key:        fmt.Sprintf("convention_%d", taskID),
			Confidence: 0.6,
			Value:      baseValue(),
			SourceRefs: sourceRefs,
		})
	}

	// constraint: if summary mentions constraints, policies, must/must-not
	if strings.Contains(lower, "constraint") || strings.Contains(lower, "policy") ||
		strings.Contains(lower, "must not") || strings.Contains(lower, "required") ||
		strings.Contains(lower, "forbidden") {
		candidates = append(candidates, MemoryCandidate{
			Scope:      "constraint",
			Key:        fmt.Sprintf("constraint_%d", taskID),
			Confidence: 0.8,
			Value:      baseValue(),
			SourceRefs: sourceRefs,
		})
	}

	// failure_pattern: if summary mentions failures even in a success context
	if strings.Contains(lower, "failure") || strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") {
		if len(summary.Risks) > 0 {
			candidates = append(candidates, MemoryCandidate{
				Scope:      "failure_pattern",
				Key:        fmt.Sprintf("failure_insight_%d", taskID),
				Confidence: 0.65,
				Value:      baseValue(),
				SourceRefs: sourceRefs,
			})
		}
	}

	// Always create a general completion memory (lower confidence).
	candidates = append(candidates, MemoryCandidate{
		Scope:      "completion",
		Key:        fmt.Sprintf("task_%d_completion", taskID),
		Confidence: 0.4,
		Value:      baseValue(),
		SourceRefs: sourceRefs,
	})

	return candidates
}

// memoryContentHash computes a SHA-256 hash of the memory value for dedup.
func memoryContentHash(value map[string]interface{}) string {
	// Use summary + changes_count + files_touched as dedupe signal.
	summary, _ := value["summary"].(string)
	raw := fmt.Sprintf("%s:%v:%v", summary, value["changes_count"], value["files_touched"])
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash[:16]) // 128-bit prefix is sufficient
}

// isDuplicateMemory checks if any memory in the project+scope has the same content hash.
func isDuplicateMemory(database *sql.DB, projectID, scope, contentHash string) bool {
	mems, err := QueryMemories(database, projectID, scope, "", "")
	if err != nil {
		return false
	}
	for _, m := range mems {
		if h, ok := m.Value["content_hash"].(string); ok && h == contentHash {
			return true
		}
	}
	return false
}


// redactSecrets strips patterns that look like API keys, bearer tokens,
// AWS credentials, GitHub tokens, passwords in URLs, and other common secrets.
func redactSecrets(s string) string {
	// Pass 1: prefix-based redaction for known secret patterns.
	patterns := []struct{ prefix, replacement string }{
		// OpenAI / Stripe-style keys
		{"sk-", "sk-[REDACTED]"},
		// Bearer tokens
		{"Bearer ", "Bearer [REDACTED]"},
		// URL query params
		{"token=", "token=[REDACTED]"},
		{"api_key=", "api_key=[REDACTED]"},
		{"apikey=", "apikey=[REDACTED]"},
		{"access_token=", "access_token=[REDACTED]"},
		{"secret=", "secret=[REDACTED]"},
		{"password=", "password=[REDACTED]"},
		{"passwd=", "passwd=[REDACTED]"},
		// AWS keys (AKIA...)
		{"AKIA", "[AWS_KEY_REDACTED]"},
		// GitHub tokens (ghp_, gho_, ghu_, ghs_, ghr_)
		{"ghp_", "ghp_[REDACTED]"},
		{"gho_", "gho_[REDACTED]"},
		{"ghu_", "ghu_[REDACTED]"},
		{"ghs_", "ghs_[REDACTED]"},
		{"ghr_", "ghr_[REDACTED]"},
		// npm tokens
		{"npm_", "npm_[REDACTED]"},
		// Slack tokens
		{"xoxb-", "xoxb-[REDACTED]"},
		{"xoxp-", "xoxp-[REDACTED]"},
		{"xoxs-", "xoxs-[REDACTED]"},
	}
	result := s
	for _, p := range patterns {
		searchFrom := 0
		for {
			idx := strings.Index(result[searchFrom:], p.prefix)
			if idx < 0 {
				break
			}
			idx += searchFrom
			end := idx + len(p.prefix)
			// Find end of the token (whitespace, quotes, delimiters, or end of string).
			for end < len(result) && result[end] != ' ' && result[end] != '\n' && result[end] != '"' && result[end] != '\'' && result[end] != ',' && result[end] != ';' {
				end++
			}
			result = result[:idx] + p.replacement + result[end:]
			searchFrom = idx + len(p.replacement)
		}
	}

	// Pass 2: redact passwords embedded in URLs (scheme://user:password@host).
	result = redactURLPasswords(result)

	return result
}

// redactURLPasswords strips passwords from URLs matching scheme://user:pass@host.
func redactURLPasswords(s string) string {
	schemes := []string{"https://", "http://", "postgres://", "postgresql://", "mysql://", "mongodb://", "redis://"}
	result := s
	for _, scheme := range schemes {
		for {
			idx := strings.Index(result, scheme)
			if idx < 0 {
				break
			}
			rest := result[idx+len(scheme):]
			// Look for user:pass@host pattern.
			atIdx := strings.Index(rest, "@")
			if atIdx < 0 {
				break
			}
			userPass := rest[:atIdx]
			colonIdx := strings.Index(userPass, ":")
			if colonIdx < 0 {
				break
			}
			// Replace password portion with [REDACTED].
			user := userPass[:colonIdx]
			newUserPass := user + ":[REDACTED]"
			result = result[:idx+len(scheme)] + newUserPass + result[idx+len(scheme)+atIdx:]
			break // one replacement per scheme per pass
		}
	}
	return result
}

// createMemoryFromFailure creates Memory records from task failures.
// Uses scope "failure_pattern" with a key derived from the error type.
func createMemoryFromFailure(database *sql.DB, projectID string, taskID int, errMsg string) error {
	if errMsg == "" {
		return nil
	}

	cleaned := redactSecrets(errMsg)
	scope := "failure_pattern"
	key := fmt.Sprintf("failure_%s", classifyErrorKey(cleaned))

	// B1: Skip creation if a recent memory with the same key already exists (<24h).
	if existing, err := GetMemory(database, projectID, scope, key); err == nil && existing != nil {
		cutoff := time.Now().Add(-24 * time.Hour)
		if t, parseErr := time.Parse(time.RFC3339Nano, existing.UpdatedAt); parseErr == nil && t.After(cutoff) {
			return nil
		}
	}

	value := map[string]interface{}{
		"error":   cleaned,
		"task_id": taskID,
		"timestamp": nowISO(),
	}

	sourceRefs := []string{fmt.Sprintf("task:%d", taskID)}
	_, err := CreateMemory(database, projectID, scope, key, value, sourceRefs)
	return err
}

// classifyErrorKey returns a short key derived from the error type.
func classifyErrorKey(errMsg string) string {
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "timeout"):
		return "timeout"
	case strings.Contains(lower, "permission") || strings.Contains(lower, "forbidden") || strings.Contains(lower, "unauthorized"):
		return "permission"
	case strings.Contains(lower, "not found") || strings.Contains(lower, "404"):
		return "not_found"
	case strings.Contains(lower, "compile") || strings.Contains(lower, "syntax") || strings.Contains(lower, "parse"):
		return "compile_error"
	case strings.Contains(lower, "test") || strings.Contains(lower, "assert"):
		return "test_failure"
	case strings.Contains(lower, "network") || strings.Contains(lower, "connection"):
		return "network"
	default:
		return "general"
	}
}
