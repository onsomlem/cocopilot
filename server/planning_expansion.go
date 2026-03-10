package server

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ---- Task 28: Built-in Default Prompts ----

// BuiltInPrompt defines a default prompt template that can be seeded into a project.
type BuiltInPrompt struct {
	Role         string
	Name         string
	Description  string
	SystemPrompt string
	UserTemplate string
	OutputSchema string
}

// GetBuiltInPrompts returns the default set of prompt templates.
func GetBuiltInPrompts() []BuiltInPrompt {
	return []BuiltInPrompt{
		{
			Role:        "idle_recon",
			Name:        "Default Recon Prompt",
			Description: "Gathers current project state for planning decisions",
			SystemPrompt: "You are a project reconnaissance agent. Your job is to summarize the current state of the project: active tasks, completed work, failures, and available resources.",
			UserTemplate: "Project ID: {{project_id}}\n\nCurrent state:\n{{recon_summary}}\n\nProvide a structured assessment of the project's current situation.",
			OutputSchema: `{"type":"object","properties":{"summary":{"type":"string"},"risk_level":{"type":"string"},"recommendations":{"type":"array","items":{"type":"string"}}}}`,
		},
		{
			Role:        "idle_continuity",
			Name:        "Default Continuity Prompt",
			Description: "Evaluates workstream continuity and identifies stalled threads",
			SystemPrompt: "You are a continuity analyst. Evaluate active workstreams and determine which have momentum, which are stalled, and what the next steps should be.",
			UserTemplate: "Project: {{project_id}}\nWorkstreams: {{workstreams}}\nRecon: {{recon_summary}}\n\nEvaluate continuity of each workstream.",
			OutputSchema: `{"type":"object","properties":{"continuation_candidates":{"type":"array"},"stalled_workstreams":{"type":"array"}}}`,
		},
		{
			Role:        "idle_gap",
			Name:        "Default Gap Analysis Prompt",
			Description: "Identifies gaps between project goals and current coverage",
			SystemPrompt: "You are a gap analyst. Compare stated project goals against active workstreams and identify uncovered areas, missed opportunities, and critical failures needing attention.",
			UserTemplate: "Goals: {{goals}}\nWorkstreams: {{workstreams}}\nRecon: {{recon_summary}}\n\nIdentify gaps and opportunities.",
			OutputSchema: `{"type":"object","properties":{"identified_gaps":{"type":"array"},"unblocked_opportunities":{"type":"array"}}}`,
		},
		{
			Role:        "idle_prioritization",
			Name:        "Default Prioritization Prompt",
			Description: "Ranks items for the next planning cycle",
			SystemPrompt: "You are a prioritization engine. Given continuity candidates and identified gaps, produce a ranked list of items to address in the next planning cycle.",
			UserTemplate: "Continuity candidates: {{continuity}}\nGaps: {{gaps}}\nMode: {{planning_mode}}\n\nRank items by priority.",
			OutputSchema: `{"type":"object","properties":{"ranked_items":{"type":"array"},"selected_focus":{"type":"string"}}}`,
		},
		{
			Role:        "task_synthesis",
			Name:        "Default Task Synthesis Prompt",
			Description: "Generates concrete task proposals from prioritized items",
			SystemPrompt: "You are a task synthesizer. Convert prioritized items into concrete, actionable task descriptions with clear instructions, appropriate types, and priority levels.",
			UserTemplate: "Ranked items: {{ranked_items}}\nMax tasks: {{max_tasks}}\nExisting tasks: {{existing_tasks}}\n\nGenerate task proposals.",
			OutputSchema: `{"type":"object","properties":{"proposed_tasks":{"type":"array"},"skipped_items":{"type":"array"}}}`,
		},
		{
			Role:        "anti_drift",
			Name:        "Default Anti-Drift Prompt",
			Description: "Validates proposed tasks against project goals to prevent drift",
			SystemPrompt: "You are a drift validator. Check proposed tasks against stated project goals and must-not-forget items. Flag anything that doesn't align.",
			UserTemplate: "Proposed tasks: {{proposed_tasks}}\nGoals: {{goals}}\nMust not forget: {{must_not_forget}}\n\nValidate alignment.",
			OutputSchema: `{"type":"object","properties":{"approved_tasks":{"type":"array"},"drift_warnings":{"type":"array"},"coherence_score":{"type":"number"}}}`,
		},
		{
			Role:        "review",
			Name:        "Default Review Prompt",
			Description: "Reviews completed work and extracts lessons learned",
			SystemPrompt: "You are a retrospective analyst. Review completed tasks and extract patterns, lessons learned, and recommendations for process improvement.",
			UserTemplate: "Completed tasks: {{completed_tasks}}\nFailed tasks: {{failed_tasks}}\nCycle history: {{cycle_history}}\n\nProvide retrospective analysis.",
			OutputSchema: `{"type":"object","properties":{"lessons":{"type":"array"},"process_improvements":{"type":"array"},"quality_score":{"type":"number"}}}`,
		},
		{
			Role:        "failure_recovery",
			Name:        "Default Failure Recovery Prompt",
			Description: "Analyzes failures and generates recovery plans",
			SystemPrompt: "You are a failure recovery specialist. Analyze failed tasks, identify root causes, and propose recovery actions.",
			UserTemplate: "Failed tasks: {{failed_tasks}}\nError details: {{error_details}}\nProject context: {{recon_summary}}\n\nAnalyze failures and propose recovery.",
			OutputSchema: `{"type":"object","properties":{"root_causes":{"type":"array"},"recovery_actions":{"type":"array"},"prevention_recommendations":{"type":"array"}}}`,
		},
	}
}

// SeedBuiltInPrompts creates default prompt templates for a project if none exist.
func SeedBuiltInPrompts(db *sql.DB, projectID string) error {
	existing, err := ListPromptTemplates(db, projectID, "", false)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil // already has prompts
	}

	for _, bp := range GetBuiltInPrompts() {
		desc := bp.Description
		schema := bp.OutputSchema
		_, err := CreatePromptTemplate(db, projectID, bp.Role, bp.Name, bp.SystemPrompt, bp.UserTemplate, &desc, &schema)
		if err != nil {
			log.Printf("planning: failed to seed prompt '%s': %v", bp.Role, err)
		}
	}
	return nil
}

// ---- Task 29: Structured Handoffs ----

// HandoffContext captures the full context needed for a task handoff between agents.
type HandoffContext struct {
	TaskID           int                    `json:"task_id"`
	ProjectID        string                 `json:"project_id"`
	WorkstreamID     string                 `json:"workstream_id,omitempty"`
	PreviousRunID    *int                   `json:"previous_run_id,omitempty"`
	PlanningMode     string                 `json:"planning_mode"`
	ReconSummary     string                 `json:"recon_summary"`
	Goals            []string               `json:"goals"`
	MustNotForget    []string               `json:"must_not_forget"`
	RelatedDecisions []PlannerDecision      `json:"related_decisions,omitempty"`
	Blockers         []string               `json:"blockers,omitempty"`
	Risks            []string               `json:"risks,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// BuildHandoffContext creates a structured handoff context for a task.
func BuildHandoffContext(db *sql.DB, taskID int) (*HandoffContext, error) {
	task, err := GetTaskV2(db, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	projectID := task.ProjectID

	hc := &HandoffContext{
		TaskID:    taskID,
		ProjectID: projectID,
	}

	// Get planning state if project exists
	if projectID != "" {
		ps, err := GetPlanningState(db, projectID)
		if err == nil && ps != nil {
			hc.PlanningMode = ps.PlanningMode
			hc.ReconSummary = ps.ReconSummary
			hc.Goals = ps.Goals
			hc.MustNotForget = ps.MustNotForget
			hc.Blockers = ps.Blockers
			hc.Risks = ps.Risks
		}

		// Find the workstream this task belongs to
		workstreams, _ := ListWorkstreams(db, projectID, "")
		for _, ws := range workstreams {
			for _, tid := range ws.RelatedTaskIDs {
				if tid == taskID {
					hc.WorkstreamID = ws.ID
					break
				}
			}
			if hc.WorkstreamID != "" {
				break
			}
		}

		// Get recent related decisions
		decisions, _ := ListPlannerDecisions(db, projectID, "", 10)
		for _, d := range decisions {
			if containsIgnoreCase(d.Subject, fmt.Sprintf("Task %d", taskID)) {
				hc.RelatedDecisions = append(hc.RelatedDecisions, d)
			}
		}
	}

	return hc, nil
}

// ---- Task 30: Quality Metrics ----

// PlanningQualityMetrics captures quality indicators for the planning system.
type PlanningQualityMetrics struct {
	ProjectID              string  `json:"project_id"`
	TotalCycles            int     `json:"total_cycles"`
	TasksCreated           int     `json:"tasks_created"`
	TasksSucceeded         int     `json:"tasks_succeeded"`
	TasksFailed            int     `json:"tasks_failed"`
	AvgCoherence           float64 `json:"avg_coherence"`
	AvgTasksPerCycle       float64 `json:"avg_tasks_per_cycle"`
	StageFailureRate       float64 `json:"stage_failure_rate"`
	ActiveWorkstreams      int     `json:"active_workstreams"`
	AvgContinuityScore     float64 `json:"avg_continuity_score"`
	GoalCoverage           float64 `json:"goal_coverage"` // percentage of goals with active workstreams
	DriftWarningRate       float64 `json:"drift_warning_rate"`
	LastCycleAt            string  `json:"last_cycle_at"`
	HealthStatus           string  `json:"health_status"` // healthy, degraded, critical
}

// ComputePlanningQuality calculates quality metrics for a project's planning system.
func ComputePlanningQuality(db *sql.DB, projectID string) (*PlanningQualityMetrics, error) {
	m := &PlanningQualityMetrics{ProjectID: projectID}

	// Get planning state
	ps, err := GetPlanningState(db, projectID)
	if err != nil || ps == nil {
		m.HealthStatus = "not_initialized"
		return m, nil
	}
	m.TotalCycles = ps.CycleCount
	if ps.LastCycleAt != nil {
		m.LastCycleAt = *ps.LastCycleAt
	}

	// Get cycles for coherence and failure stats
	cycles, _ := ListPlanningCycles(db, projectID, 50)
	totalCoherence := 0.0
	totalStageFailures := 0
	totalDriftWarnings := 0
	totalTasksCreated := 0
	for _, c := range cycles {
		totalCoherence += c.CoherenceScore
		totalStageFailures += len(c.StageFailures)
		totalDriftWarnings += len(c.DriftWarnings)
		totalTasksCreated += len(c.TasksCreated)
	}
	if len(cycles) > 0 {
		m.AvgCoherence = totalCoherence / float64(len(cycles))
		m.AvgTasksPerCycle = float64(totalTasksCreated) / float64(len(cycles))
		m.StageFailureRate = float64(totalStageFailures) / float64(len(cycles)*7) // 7 stages per cycle
		m.DriftWarningRate = float64(totalDriftWarnings) / float64(len(cycles))
	}
	m.TasksCreated = totalTasksCreated

	// Count succeeded/failed auto-planned tasks
	tasks, _, _ := ListTasksV2(db, projectID, "status_v2", "", "", "", "", 500, 0, "created_at", "desc")
	for _, t := range tasks {
		if t.Tags != nil {
			for _, tag := range t.Tags {
				if tag == "auto-planned" {
					switch t.StatusV2 {
					case TaskStatusSucceeded:
						m.TasksSucceeded++
					case TaskStatusFailed:
						m.TasksFailed++
					}
					break
				}
			}
		}
	}

	// Workstream stats
	workstreams, _ := ListWorkstreams(db, projectID, "active")
	m.ActiveWorkstreams = len(workstreams)
	totalContinuity := 0.0
	for _, ws := range workstreams {
		totalContinuity += ws.ContinuityScore
	}
	if len(workstreams) > 0 {
		m.AvgContinuityScore = totalContinuity / float64(len(workstreams))
	}

	// Goal coverage
	if len(ps.Goals) > 0 {
		coveredGoals := 0
		allWorkstreams, _ := ListWorkstreams(db, projectID, "")
		for _, g := range ps.Goals {
			for _, ws := range allWorkstreams {
				if containsIgnoreCase(ws.Title, g) || containsIgnoreCase(ws.Description, g) {
					coveredGoals++
					break
				}
			}
		}
		m.GoalCoverage = float64(coveredGoals) / float64(len(ps.Goals))
	}

	// Determine health status
	m.HealthStatus = determinePlanningHealth(m, ps)

	return m, nil
}

func determinePlanningHealth(m *PlanningQualityMetrics, ps *PlanningState) string {
	issues := 0

	if m.AvgCoherence < 0.5 {
		issues += 2
	} else if m.AvgCoherence < 0.7 {
		issues++
	}

	if m.StageFailureRate > 0.3 {
		issues += 2
	} else if m.StageFailureRate > 0.1 {
		issues++
	}

	if m.GoalCoverage < 0.3 && len(ps.Goals) > 0 {
		issues += 2
	}

	if m.AvgContinuityScore < 0.2 {
		issues++
	}

	// Check staleness
	if ps.LastCycleAt != nil && *ps.LastCycleAt != "" {
		last, err := time.Parse("2006-01-02T15:04:05.999999Z", *ps.LastCycleAt)
		if err == nil && time.Since(last) > 2*time.Hour {
			issues++
		}
	}

	switch {
	case issues >= 4:
		return "critical"
	case issues >= 2:
		return "degraded"
	default:
		return "healthy"
	}
}
