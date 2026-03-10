package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// PipelineConfig holds tuning parameters for the staged planning pipeline.
type PipelineConfig struct {
	MaxTasksPerCycle     int           `json:"max_tasks_per_cycle"`
	MinCycleInterval     time.Duration `json:"min_cycle_interval"`
	ContinuityThreshold  float64       `json:"continuity_threshold"`
	CoherenceThreshold   float64       `json:"coherence_threshold"`
	EnableAntiDrift      bool          `json:"enable_anti_drift"`
}

// DefaultPipelineConfig returns sensible defaults.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		MaxTasksPerCycle:    3,
		MinCycleInterval:    5 * time.Minute,
		ContinuityThreshold: 0.3,
		CoherenceThreshold:  0.5,
		EnableAntiDrift:     true,
	}
}

// PipelineStage identifies a stage in the planning pipeline.
type PipelineStage string

const (
	StageRecon          PipelineStage = "recon"
	StageContinuity     PipelineStage = "continuity"
	StageGap            PipelineStage = "gap"
	StagePrioritization PipelineStage = "prioritization"
	StageSynthesis      PipelineStage = "synthesis"
	StageAntiDrift      PipelineStage = "anti_drift"
	StageStateUpdate    PipelineStage = "state_update"
)

// StageResult captures the output of one pipeline stage.
type StageResult struct {
	Stage   PipelineStage          `json:"stage"`
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// PipelineResult captures the result of a full planning cycle.
type PipelineResult struct {
	CycleID        string                 `json:"cycle_id"`
	ProjectID      string                 `json:"project_id"`
	Stages         []StageResult          `json:"stages"`
	TasksCreated   []int                  `json:"tasks_created"`
	CoherenceScore float64                `json:"coherence_score"`
	DriftWarnings  []string               `json:"drift_warnings"`
	StageFailures  []string               `json:"stage_failures"`
	Duration       string                 `json:"duration"`
}

// RunPlanningPipeline executes the full staged planning pipeline for a project.
func RunPlanningPipeline(db *sql.DB, projectID string, cfg PipelineConfig) (*PipelineResult, error) {
	start := time.Now()

	// Get or create planning state
	ps, err := GetOrCreatePlanningState(db, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get planning state: %w", err)
	}

	// Apply planning mode overrides
	ApplyModeToConfig(ps.PlanningMode, &cfg)

	// Check min cycle interval
	if ps.LastCycleAt != nil && *ps.LastCycleAt != "" {
		lastCycle, parseErr := time.Parse("2006-01-02T15:04:05.999999Z", *ps.LastCycleAt)
		if parseErr == nil && time.Since(lastCycle) < cfg.MinCycleInterval {
			return nil, fmt.Errorf("minimum cycle interval not reached (last: %s)", *ps.LastCycleAt)
		}
	}

	// Create cycle record
	cycle, err := CreatePlanningCycle(db, projectID, ps.CycleCount+1, ps.PlanningMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create planning cycle: %w", err)
	}

	result := &PipelineResult{
		CycleID:   cycle.ID,
		ProjectID: projectID,
	}

	// Stage 0: Recon
	reconResult := runReconStage(db, projectID, ps)
	result.Stages = append(result.Stages, reconResult)
	if !reconResult.Success {
		result.StageFailures = append(result.StageFailures, "recon")
		finalizePipeline(db, cycle, ps, result, start)
		return result, nil
	}
	cycle.ReconOutput = reconResult.Output

	// Stage 1: Continuity
	continuityResult := runContinuityStage(db, projectID, ps, reconResult.Output)
	result.Stages = append(result.Stages, continuityResult)
	if !continuityResult.Success {
		result.StageFailures = append(result.StageFailures, "continuity")
	}
	cycle.ContinuityOutput = continuityResult.Output

	// Stage 2: Gap Analysis
	gapResult := runGapStage(db, projectID, ps, reconResult.Output, continuityResult.Output)
	result.Stages = append(result.Stages, gapResult)
	if !gapResult.Success {
		result.StageFailures = append(result.StageFailures, "gap")
	}
	cycle.GapOutput = gapResult.Output

	// Stage 3: Prioritization
	prioResult := runPrioritizationStage(db, projectID, ps, continuityResult.Output, gapResult.Output)
	result.Stages = append(result.Stages, prioResult)
	if !prioResult.Success {
		result.StageFailures = append(result.StageFailures, "prioritization")
	}
	cycle.PrioritizationOutput = prioResult.Output

	// Stage 4: Task Synthesis
	synthResult := runSynthesisStage(db, projectID, ps, prioResult.Output, cfg)
	result.Stages = append(result.Stages, synthResult)
	if !synthResult.Success {
		result.StageFailures = append(result.StageFailures, "synthesis")
	}
	cycle.SynthesisOutput = synthResult.Output

	// Stage 5: Anti-Drift Validation
	var driftResult StageResult
	if cfg.EnableAntiDrift {
		driftResult = runAntiDriftStage(db, projectID, ps, synthResult.Output)
	} else {
		driftResult = StageResult{
			Stage:   StageAntiDrift,
			Success: true,
			Output:  map[string]interface{}{"skipped": true},
		}
	}
	result.Stages = append(result.Stages, driftResult)
	if !driftResult.Success {
		result.StageFailures = append(result.StageFailures, "anti_drift")
	}
	cycle.AntiDriftOutput = driftResult.Output

	// Extract drift warnings and coherence
	if warnings, ok := driftResult.Output["drift_warnings"]; ok {
		if warnList, ok := warnings.([]interface{}); ok {
			for _, w := range warnList {
				if wStr, ok := w.(string); ok {
					result.DriftWarnings = append(result.DriftWarnings, wStr)
				}
			}
		}
	}
	if score, ok := driftResult.Output["overall_coherence_score"].(float64); ok {
		result.CoherenceScore = score
		cycle.CoherenceScore = score
	}

	// Create approved tasks
	createdTaskIDs := createApprovedTasks(db, projectID, synthResult.Output, driftResult.Output, cycle.ID)
	result.TasksCreated = createdTaskIDs
	cycle.TasksCreated = createdTaskIDs

	// Stage 6: State Update
	stateResult := runStateUpdateStage(db, projectID, ps, cycle, result)
	result.Stages = append(result.Stages, stateResult)
	if !stateResult.Success {
		result.StageFailures = append(result.StageFailures, "state_update")
	}

	cycle.StageFailures = result.StageFailures
	cycle.DriftWarnings = result.DriftWarnings

	finalizePipeline(db, cycle, ps, result, start)
	result.Duration = time.Since(start).String()

	return result, nil
}

// ---- Stage Implementations ----

func runReconStage(db *sql.DB, projectID string, ps *PlanningState) StageResult {
	output := map[string]interface{}{}

	// Count tasks by status
	tasks, _, err := ListTasksV2(db, projectID, "status_v2", "", "", "", "", 1000, 0, "created_at", "desc")
	if err != nil {
		return StageResult{Stage: StageRecon, Success: false, Error: err.Error()}
	}

	var active, queued, failed, blocked int
	for _, t := range tasks {
		switch t.StatusV2 {
		case TaskStatusQueued:
			queued++
		case TaskStatusClaimed, TaskStatusRunning:
			active++
		case TaskStatusFailed:
			failed++
		}
	}

	// Check for blocked tasks (tasks with unfulfilled dependencies)
	for _, t := range tasks {
		if t.StatusV2 == TaskStatusQueued {
			fulfilled, dErr := AreAllDependenciesFulfilled(db, t.ID)
			if dErr == nil && !fulfilled {
				blocked++
			}
		}
	}

	// Count agents
	agents, _, err := ListAgents(db, "", "", 100, 0, "registered_at", "desc")
	if err != nil {
		agents = nil
	}

	agentCount := len(agents)
	var idleAgents int
	for _, a := range agents {
		if a.Status == AgentStatusIdle || a.Status == AgentStatusOnline {
			idleAgents++
		}
	}

	output["summary"] = fmt.Sprintf("Project has %d tasks (%d active, %d queued, %d failed, %d blocked), %d agents (%d idle)",
		len(tasks), active, queued, failed, blocked, agentCount, idleAgents)
	output["active_task_count"] = active
	output["queued_task_count"] = queued
	output["failed_task_count"] = failed
	output["blocked_task_count"] = blocked
	output["agent_count"] = agentCount
	output["environment_flags"] = map[string]interface{}{
		"has_failures":              failed > 0,
		"has_blocked":               blocked > 0,
		"has_idle_agents":           idleAgents > 0,
		"stale_workstreams_detected": false,
	}

	return StageResult{Stage: StageRecon, Success: true, Output: output}
}

func runContinuityStage(db *sql.DB, projectID string, ps *PlanningState, reconOutput map[string]interface{}) StageResult {
	output := map[string]interface{}{}

	// Refresh workstream continuity/urgency scores
	_ = RefreshWorkstreamScores(db, projectID)

	workstreams, err := ListWorkstreams(db, projectID, "")
	if err != nil {
		return StageResult{Stage: StageContinuity, Success: false, Error: err.Error()}
	}

	var candidates []map[string]interface{}
	var completed []string
	var stalled []string

	for _, ws := range workstreams {
		switch ws.Status {
		case "completed":
			completed = append(completed, ws.ID)
		case "abandoned":
			continue
		case "active", "paused":
			candidate := map[string]interface{}{
				"workstream_id":       ws.ID,
				"continuity_score":    ws.ContinuityScore,
				"reason":              ws.Why,
				"suggested_next_action": ws.WhatNext,
				"risk_if_abandoned":   "Work-in-progress may be lost",
			}
			candidates = append(candidates, candidate)

			// Check if stalled (no task activity)
			if len(ws.RelatedTaskIDs) == 0 {
				stalled = append(stalled, ws.ID)
			}
		}
	}

	output["continuation_candidates"] = candidates
	output["completed_workstreams"] = completed
	output["stalled_workstreams"] = stalled

	return StageResult{Stage: StageContinuity, Success: true, Output: output}
}

func runGapStage(db *sql.DB, projectID string, ps *PlanningState, reconOutput, continuityOutput map[string]interface{}) StageResult {
	output := map[string]interface{}{}

	var gaps []map[string]interface{}
	var opportunities []map[string]interface{}

	// Check if any goals have no workstreams
	workstreams, _ := ListWorkstreams(db, projectID, "active")
	coveredGoals := make(map[string]bool)
	for _, ws := range workstreams {
		for _, g := range ps.Goals {
			// Simple containment check
			if containsIgnoreCase(ws.Title, g) || containsIgnoreCase(ws.Description, g) {
				coveredGoals[g] = true
			}
		}
	}

	for _, g := range ps.Goals {
		if !coveredGoals[g] {
			gaps = append(gaps, map[string]interface{}{
				"description":        fmt.Sprintf("Goal '%s' has no active workstream", g),
				"severity":           "important",
				"related_goal":       g,
				"suggested_workstream": "",
			})
		}
	}

	// Check for failed tasks that need recovery
	failedCount := 0
	if fc, ok := reconOutput["failed_task_count"].(int); ok {
		failedCount = fc
	}
	if failedCount > 0 {
		gaps = append(gaps, map[string]interface{}{
			"description":        fmt.Sprintf("%d failed task(s) need attention", failedCount),
			"severity":           "critical",
			"related_goal":       "reliability",
			"suggested_workstream": "",
		})
	}

	// Check for idle agents as opportunity
	if flags, ok := reconOutput["environment_flags"].(map[string]interface{}); ok {
		if hasIdle, ok := flags["has_idle_agents"].(bool); ok && hasIdle {
			opportunities = append(opportunities, map[string]interface{}{
				"description":      "Idle agents available for new work",
				"reason_unblocked": "Agents are online but not assigned to tasks",
				"estimated_effort": "small",
			})
		}
	}

	output["identified_gaps"] = gaps
	output["unblocked_opportunities"] = opportunities

	return StageResult{Stage: StageGap, Success: true, Output: output}
}

func runPrioritizationStage(db *sql.DB, projectID string, ps *PlanningState, continuityOutput, gapOutput map[string]interface{}) StageResult {
	output := map[string]interface{}{}

	// Check anti-fragmentation rules
	modeConfig := GetPlanningModeConfig(ps.PlanningMode)
	rules := DefaultAntiFragmentationRules()
	// Override max active workstreams from mode config
	for i := range rules {
		if rules[i].Type == "max_active_workstreams" {
			rules[i].Value = modeConfig.MaxActiveWorkstreams
		}
	}
	violations := CheckAntiFragmentation(db, projectID, rules)
	var fragmentationWarnings []string
	for _, v := range violations {
		fragmentationWarnings = append(fragmentationWarnings, v.Details)
	}
	output["fragmentation_violations"] = fragmentationWarnings

	var ranked []map[string]interface{}
	var deferred []map[string]interface{}
	rank := 1

	// In recovery mode, prioritize failures by checking gaps first
	if modeConfig.PrioritizeFailures {
		if gaps, ok := gapOutput["identified_gaps"].([]map[string]interface{}); ok {
			for _, g := range gaps {
				if sev, ok := g["severity"].(string); ok && sev == "critical" {
					score := 0.95
					ranked = append(ranked, map[string]interface{}{
						"rank":           rank,
						"source":         "gap_recovery",
						"workstream_id":  g["suggested_workstream"],
						"description":    g["description"],
						"priority_score": score,
						"reasoning":      fmt.Sprintf("Recovery priority: %s", g["description"]),
					})
					rank++
				}
			}
		}
	}

	// Apply continuity boost from mode config
	continuityBoost := modeConfig.ContinuityBoost

	// Rank continuity candidates first
	if candidates, ok := continuityOutput["continuation_candidates"].([]map[string]interface{}); ok {
		for _, c := range candidates {
			score := 0.5
			if s, ok := c["continuity_score"].(float64); ok {
				score = s * continuityBoost
				if score > 1.0 {
					score = 1.0
				}
			}
			ranked = append(ranked, map[string]interface{}{
				"rank":           rank,
				"source":         "continuity",
				"workstream_id":  c["workstream_id"],
				"description":    c["suggested_next_action"],
				"priority_score": score,
				"reasoning":      c["reason"],
			})
			rank++
		}
	}

	// Then gap items
	if gaps, ok := gapOutput["identified_gaps"].([]map[string]interface{}); ok {
		for _, g := range gaps {
			score := 0.5
			if sev, ok := g["severity"].(string); ok {
				switch sev {
				case "critical":
					score = 0.9
				case "important":
					score = 0.7
				case "minor":
					score = 0.3
				}
			}
			ranked = append(ranked, map[string]interface{}{
				"rank":           rank,
				"source":         "gap",
				"workstream_id":  g["suggested_workstream"],
				"description":    g["description"],
				"priority_score": score,
				"reasoning":      fmt.Sprintf("Gap: %s", g["description"]),
			})
			rank++
		}
	}

	// Then opportunities
	if opps, ok := gapOutput["unblocked_opportunities"].([]map[string]interface{}); ok {
		for _, o := range opps {
			ranked = append(ranked, map[string]interface{}{
				"rank":           rank,
				"source":         "opportunity",
				"workstream_id":  nil,
				"description":    o["description"],
				"priority_score": 0.4,
				"reasoning":      o["reason_unblocked"],
			})
			rank++
		}
	}

	// In focused mode with enforced fragmentation, block new workstream items if at limit
	if !modeConfig.AllowNewWorkstreams {
		var filtered []map[string]interface{}
		for _, item := range ranked {
			src, _ := item["source"].(string)
			wsID, _ := item["workstream_id"].(string)
			if src == "opportunity" && wsID == "" {
				deferred = append(deferred, map[string]interface{}{
					"description": item["description"],
					"reason":      "New workstreams not allowed in " + ps.PlanningMode + " mode",
				})
				continue
			}
			filtered = append(filtered, item)
		}
		ranked = filtered
	}

	// Select focus
	selectedFocus := ""
	if len(ranked) > 0 {
		if desc, ok := ranked[0]["description"].(string); ok {
			selectedFocus = desc
		}
	}

	output["ranked_items"] = ranked
	output["selected_focus"] = selectedFocus
	output["deferred_items"] = deferred
	output["planning_mode"] = ps.PlanningMode

	return StageResult{Stage: StagePrioritization, Success: true, Output: output}
}

func runSynthesisStage(db *sql.DB, projectID string, ps *PlanningState, prioOutput map[string]interface{}, cfg PipelineConfig) StageResult {
	output := map[string]interface{}{}

	var proposed []map[string]interface{}
	var skipped []map[string]interface{}

	rankedItems, _ := prioOutput["ranked_items"].([]map[string]interface{})

	maxTasks := cfg.MaxTasksPerCycle
	if maxTasks <= 0 {
		maxTasks = 3
	}

	// Get existing tasks to check for duplicates
	existingTasks, _, _ := ListTasksV2(db, projectID, "status_v2", "", "", "", "", 100, 0, "created_at", "desc")
	existingTitles := make(map[string]bool)
	for _, t := range existingTasks {
		if t.Title != nil {
			existingTitles[*t.Title] = true
		}
	}

	for i, item := range rankedItems {
		if i >= maxTasks {
			skipped = append(skipped, map[string]interface{}{
				"description": item["description"],
				"reason":      "Exceeded max_tasks_per_cycle",
			})
			continue
		}

		desc, _ := item["description"].(string)
		if desc == "" {
			continue
		}

		// Skip if title already used
		if existingTitles[desc] {
			skipped = append(skipped, map[string]interface{}{
				"description": desc,
				"reason":      "Duplicate of existing task",
			})
			continue
		}

		source, _ := item["source"].(string)
		wsID, _ := item["workstream_id"].(string)

		taskType := "ANALYZE"
		if source == "gap" {
			taskType = "MODIFY"
		}

		priority := 50
		if score, ok := item["priority_score"].(float64); ok {
			priority = int(score * 100)
		}

		proposed = append(proposed, map[string]interface{}{
			"title":         desc,
			"instructions":  fmt.Sprintf("[Auto-planned] %s", desc),
			"type":          taskType,
			"priority":      priority,
			"tags":          []string{"auto-planned"},
			"workstream_id": wsID,
			"rationale":     item["reasoning"],
		})
	}

	output["proposed_tasks"] = proposed
	output["skipped_items"] = skipped

	return StageResult{Stage: StageSynthesis, Success: true, Output: output}
}

func runAntiDriftStage(db *sql.DB, projectID string, ps *PlanningState, synthOutput map[string]interface{}) StageResult {
	output := map[string]interface{}{}

	proposed, _ := synthOutput["proposed_tasks"].([]map[string]interface{})
	var approved []map[string]interface{}
	var warnings []interface{}

	coherenceScore := 1.0
	if len(ps.Goals) == 0 {
		coherenceScore = 0.5 // No goals defined = lower confidence
		warnings = append(warnings, "No project goals defined — coherence assessment limited")
	}

	for i, task := range proposed {
		entry := map[string]interface{}{
			"task_index": i,
			"approved":   true,
		}

		// Check title against project goals (basic coherence)
		title, _ := task["title"].(string)
		if len(ps.Goals) > 0 {
			goalAligned := false
			for _, g := range ps.Goals {
				if containsIgnoreCase(title, g) {
					goalAligned = true
					break
				}
			}
			if !goalAligned {
				// Doesn't align but still approve (soft warning)
				entry["modification"] = "Consider aligning with stated project goals"
				coherenceScore -= 0.1
			}
		}

		approved = append(approved, entry)
	}

	if coherenceScore < 0 {
		coherenceScore = 0
	}

	output["approved_tasks"] = approved
	output["drift_warnings"] = warnings
	output["overall_coherence_score"] = coherenceScore

	return StageResult{Stage: StageAntiDrift, Success: true, Output: output}
}

func runStateUpdateStage(db *sql.DB, projectID string, ps *PlanningState, cycle *PlanningCycle, result *PipelineResult) StageResult {
	ps.CycleCount++
	now := nowISO()
	ps.LastCycleAt = &now

	// Update recon summary
	if len(result.Stages) > 0 && result.Stages[0].Success {
		if summary, ok := result.Stages[0].Output["summary"].(string); ok {
			ps.ReconSummary = summary
		}
	}

	// Update planner summary
	ps.PlannerSummary = fmt.Sprintf("Cycle %d: %d stage(s) completed, %d task(s) created, %d failure(s)",
		ps.CycleCount, len(result.Stages), len(result.TasksCreated), len(result.StageFailures))

	if err := UpdatePlanningState(db, ps); err != nil {
		return StageResult{Stage: StageStateUpdate, Success: false, Error: err.Error()}
	}

	return StageResult{Stage: StageStateUpdate, Success: true, Output: map[string]interface{}{
		"cycle_count": ps.CycleCount,
		"updated_at":  ps.UpdatedAt,
	}}
}

// ---- Helpers ----

func createApprovedTasks(db *sql.DB, projectID string, synthOutput, driftOutput map[string]interface{}, cycleID string) []int {
	proposed, _ := synthOutput["proposed_tasks"].([]map[string]interface{})
	approvedList, _ := driftOutput["approved_tasks"].([]map[string]interface{})

	// Build approval map
	approvedSet := make(map[int]bool)
	for _, a := range approvedList {
		idx, _ := a["task_index"].(int)
		if approved, ok := a["approved"].(bool); ok && approved {
			approvedSet[idx] = true
		}
	}
	// If no explicit approval list, approve all
	if len(approvedList) == 0 {
		for i := range proposed {
			approvedSet[i] = true
		}
	}

	var created []int
	for i, task := range proposed {
		if !approvedSet[i] {
			continue
		}

		title, _ := task["title"].(string)
		instructions, _ := task["instructions"].(string)
		if instructions == "" {
			instructions = title
		}

		taskTypeStr, _ := task["type"].(string)
		taskType := TaskType(taskTypeStr)
		priority := 50
		if p, ok := task["priority"].(int); ok {
			priority = p
		}

		var tags []string
		if t, ok := task["tags"].([]string); ok {
			tags = t
		} else if t, ok := task["tags"].([]interface{}); ok {
			for _, v := range t {
				if s, ok := v.(string); ok {
					tags = append(tags, s)
				}
			}
		}

		newTask, err := CreateTaskV2WithMeta(db, instructions, projectID, nil, &title, &taskType, &priority, tags, nil)
		if err != nil {
			log.Printf("planning: failed to create task '%s': %v", title, err)
			continue
		}

		// Record decision
		rationale, _ := task["rationale"].(string)
		if _, dErr := CreatePlannerDecision(db, projectID, cycleID, "synthesis", "created",
			fmt.Sprintf("Task %d: %s", newTask.ID, title), rationale); dErr != nil {
			log.Printf("planning: failed to record decision: %v", dErr)
		}

		created = append(created, newTask.ID)
	}

	return created
}

func finalizePipeline(db *sql.DB, cycle *PlanningCycle, ps *PlanningState, result *PipelineResult, start time.Time) {
	cycle.StageFailures = result.StageFailures
	cycle.DriftWarnings = result.DriftWarnings

	if err := CompletePlanningCycle(db, cycle); err != nil {
		log.Printf("planning: failed to complete cycle: %v", err)
	}

	// Emit planning event
	payload := map[string]interface{}{
		"cycle_id":       cycle.ID,
		"cycle_number":   cycle.CycleNumber,
		"tasks_created":  len(result.TasksCreated),
		"stage_failures": len(result.StageFailures),
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	payloadJSON, _ := json.Marshal(payload)
	var payloadMap map[string]interface{}
	_ = json.Unmarshal(payloadJSON, &payloadMap)

	if _, err := CreateEvent(db, ps.ProjectID, "planning.cycle_completed", "planning_cycle", cycle.ID, payloadMap); err != nil {
		log.Printf("planning: failed to emit cycle event: %v", err)
	}
}

func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return false
	}
	sLower := planningToLower(s)
	subLower := planningToLower(substr)
	return planningContains(sLower, subLower)
}

func planningToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

func planningContains(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
