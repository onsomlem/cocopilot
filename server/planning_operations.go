package server

import (
	"database/sql"
	"log"
	"time"
)

// ---- Task 25: Pipeline as the Only Path ----

// ValidateTaskCreationThroughPipeline checks whether a task creation should be allowed
// based on the project's planning mode. In strict modes, only pipeline-created tasks
// (tagged "auto-planned") are allowed through automation.
func ValidateTaskCreationThroughPipeline(db *sql.DB, projectID string, tags []string) (bool, string) {
	ps, err := GetPlanningState(db, projectID)
	if err != nil || ps == nil {
		return true, "" // no planning state = no restriction
	}

	// In focused or recovery mode, enforced pipeline-only for automation-created tasks
	switch ps.PlanningMode {
	case "focused", "recovery":
		// Allow manually-tagged tasks and pipeline tasks
		for _, t := range tags {
			if t == "auto-planned" || t == "manual" || t == "user-created" {
				return true, ""
			}
		}
		return false, "In " + ps.PlanningMode + " mode, new tasks should go through the planning pipeline"
	default:
		return true, ""
	}
}

// ---- Task 26: Autonomy Modes ----

// AutonomyLevel defines how much the planning system can do without human confirmation.
type AutonomyLevel string

const (
	AutonomySuggestOnly AutonomyLevel = "suggest_only" // Pipeline proposes but doesn't create tasks
	AutonomySemiAuto    AutonomyLevel = "semi_auto"    // Creates tasks but marks them for review
	AutonomyFullAuto    AutonomyLevel = "full_auto"    // Creates and queues tasks immediately
)

// AutonomyConfig holds settings for the autonomy level.
type AutonomyConfig struct {
	Level               AutonomyLevel `json:"level"`
	MaxAutoTasksPerDay  int           `json:"max_auto_tasks_per_day"`
	RequireReviewAbove  int           `json:"require_review_above"` // priority threshold requiring human review
}

// DefaultAutonomyConfig returns sensible defaults (semi-auto).
func DefaultAutonomyConfig() AutonomyConfig {
	return AutonomyConfig{
		Level:              AutonomySemiAuto,
		MaxAutoTasksPerDay: 20,
		RequireReviewAbove: 80,
	}
}

// ShouldAutoCreateTask determines if a proposed task should be auto-created based on autonomy settings.
func ShouldAutoCreateTask(cfg AutonomyConfig, priority int) (create bool, needsReview bool) {
	switch cfg.Level {
	case AutonomySuggestOnly:
		return false, false
	case AutonomySemiAuto:
		return true, priority >= cfg.RequireReviewAbove
	case AutonomyFullAuto:
		return true, false
	default:
		return true, true
	}
}

// ---- Task 27: Continuity Triggers ----

// StartPlanningTrigger starts a background goroutine that periodically checks
// if conditions are right to trigger a planning cycle.
func StartPlanningTrigger(db *sql.DB, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			triggerPlanningCyclesIfNeeded(db)
		}
	}()
	log.Printf("planning: background trigger started (interval: %v)", interval)
}

// triggerPlanningCyclesIfNeeded checks all projects and runs planning cycles where conditions are met.
func triggerPlanningCyclesIfNeeded(db *sql.DB) {
	projects, err := ListProjects(db)
	if err != nil {
		log.Printf("planning trigger: failed to list projects: %v", err)
		return
	}

	for _, proj := range projects {
		shouldTrigger, reason := shouldTriggerPlanning(db, proj.ID)
		if !shouldTrigger {
			continue
		}

		log.Printf("planning trigger: running cycle for project %s (%s)", proj.ID, reason)
		cfg := DefaultPipelineConfig()
		result, err := RunPlanningPipeline(db, proj.ID, cfg)
		if err != nil {
			log.Printf("planning trigger: cycle failed for %s: %v", proj.ID, err)
			continue
		}
		log.Printf("planning trigger: cycle %s completed for %s (%d tasks created, %d failures)",
			result.CycleID, proj.ID, len(result.TasksCreated), len(result.StageFailures))
	}
}

// shouldTriggerPlanning evaluates whether a project needs a planning cycle.
func shouldTriggerPlanning(db *sql.DB, projectID string) (bool, string) {
	ps, err := GetPlanningState(db, projectID)
	if err != nil || ps == nil {
		return false, "" // no planning state = not opted in
	}

	// Check if mode is maintenance — lower frequency
	if ps.PlanningMode == "maintenance" {
		// Only trigger if cycle count is still 0 or last cycle was >1 hour ago
		if ps.CycleCount > 0 && ps.LastCycleAt != nil && *ps.LastCycleAt != "" {
			last, err := time.Parse("2006-01-02T15:04:05.999999Z", *ps.LastCycleAt)
			if err == nil && time.Since(last) < 1*time.Hour {
				return false, ""
			}
		}
	}

	// Check for idle agents (trigger: agents waiting with no work)
	tasks, _, _ := ListTasksV2(db, projectID, "status_v2", "", "", "", "", 100, 0, "created_at", "desc")
	queuedCount := 0
	for _, t := range tasks {
		if t.StatusV2 == TaskStatusQueued {
			queuedCount++
		}
	}
	if queuedCount == 0 {
		return true, "no queued tasks available"
	}

	// Check for high failure rate (trigger: >50% of recent tasks failed)
	failedCount := 0
	recentCount := 0
	for _, t := range tasks {
		if t.StatusV2 == TaskStatusFailed || t.StatusV2 == TaskStatusSucceeded {
			recentCount++
			if t.StatusV2 == TaskStatusFailed {
				failedCount++
			}
		}
	}
	if recentCount > 0 && failedCount*2 > recentCount {
		return true, "high failure rate detected"
	}

	// Check for stalled workstreams (trigger: active workstreams with low continuity)
	workstreams, _ := ListWorkstreams(db, projectID, "active")
	stalledCount := 0
	for _, ws := range workstreams {
		if ws.ContinuityScore < 0.2 {
			stalledCount++
		}
	}
	if stalledCount > 0 && stalledCount == len(workstreams) {
		return true, "all workstreams stalled"
	}

	return false, ""
}
