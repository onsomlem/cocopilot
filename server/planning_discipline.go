package server

import (
	"database/sql"
	"math"
)

// ---- Continuity Scoring (Task 14) ----

// ContinuityScorer computes continuity and urgency scores for workstreams
// based on task activity, age, and state.
type ContinuityScorer struct {
	// StaleCycleThreshold: how many cycles with no progress before a workstream is considered stalled.
	StaleCycleThreshold int
	// RecencyWeight: how much recent activity matters (0-1).
	RecencyWeight float64
	// CompletionWeight: how much task completion matters (0-1).
	CompletionWeight float64
}

// DefaultContinuityScorer returns a scorer with sensible defaults.
func DefaultContinuityScorer() ContinuityScorer {
	return ContinuityScorer{
		StaleCycleThreshold: 3,
		RecencyWeight:       0.4,
		CompletionWeight:    0.3,
	}
}

// ScoreWorkstream computes continuity and urgency scores for a single workstream.
func (s ContinuityScorer) ScoreWorkstream(db *sql.DB, ws *Workstream) (continuity float64, urgency float64) {
	if ws == nil {
		return 0, 0
	}

	// Base continuity from current score (momentum)
	continuity = ws.ContinuityScore * 0.3

	// Factor 1: task activity ratio
	totalTasks := len(ws.RelatedTaskIDs)
	if totalTasks > 0 {
		activeTasks := 0
		completedTasks := 0
		failedTasks := 0
		for _, tid := range ws.RelatedTaskIDs {
			task, err := GetTaskV2(db, tid)
			if err != nil {
				continue
			}
			switch task.StatusV2 {
			case TaskStatusClaimed, TaskStatusRunning:
				activeTasks++
			case TaskStatusSucceeded:
				completedTasks++
			case TaskStatusFailed:
				failedTasks++
			}
		}

		// Active tasks boost continuity
		if activeTasks > 0 {
			continuity += s.RecencyWeight * 0.8
		}

		// Completion rate affects continuity
		if totalTasks > 0 {
			completionRate := float64(completedTasks) / float64(totalTasks)
			continuity += s.CompletionWeight * completionRate
		}

		// Failed tasks boost urgency
		if failedTasks > 0 {
			urgency += 0.3 * float64(failedTasks) / float64(totalTasks)
		}
	} else {
		// No tasks = low continuity
		continuity += 0.1
	}

	// Factor 2: status-based scoring
	switch ws.Status {
	case "active":
		continuity += 0.2
		urgency += 0.2
	case "paused":
		continuity += 0.1
		urgency += 0.1
	case "completed":
		continuity = 0
		urgency = 0
		return
	case "abandoned":
		continuity = 0
		urgency = 0
		return
	}

	// Factor 3: whether there's a clear next action
	if ws.WhatNext != "" {
		continuity += 0.1
	}

	// Clamp to [0, 1]
	continuity = math.Min(1.0, math.Max(0.0, continuity))
	urgency = math.Min(1.0, math.Max(0.0, urgency))

	return continuity, urgency
}

// RefreshWorkstreamScores recomputes continuity and urgency scores for all active workstreams.
func RefreshWorkstreamScores(db *sql.DB, projectID string) error {
	scorer := DefaultContinuityScorer()
	workstreams, err := ListWorkstreams(db, projectID, "")
	if err != nil {
		return err
	}

	for i := range workstreams {
		ws := &workstreams[i]
		if ws.Status == "completed" || ws.Status == "abandoned" {
			continue
		}

		c, u := scorer.ScoreWorkstream(db, ws)
		ws.ContinuityScore = c
		ws.UrgencyScore = u
		if err := UpdateWorkstream(db, ws); err != nil {
			continue // best-effort
		}
	}
	return nil
}

// ---- Anti-Fragmentation Rules (Task 15) ----

// AntiFragmentationRule defines a constraint that prevents work from becoming
// too scattered across many workstreams.
type AntiFragmentationRule struct {
	Type       string `json:"type"`        // max_active_workstreams | min_completion_before_new | max_tasks_per_workstream
	Value      int    `json:"value"`
	Enforced   bool   `json:"enforced"`    // hard block vs. warning only
	Message    string `json:"message"`
}

// DefaultAntiFragmentationRules returns the standard set of anti-fragmentation rules.
func DefaultAntiFragmentationRules() []AntiFragmentationRule {
	return []AntiFragmentationRule{
		{
			Type:     "max_active_workstreams",
			Value:    5,
			Enforced: true,
			Message:  "Too many active workstreams. Complete or pause existing work before starting new threads.",
		},
		{
			Type:     "min_completion_before_new",
			Value:    30, // percent
			Enforced: false,
			Message:  "Less than 30% of existing workstream tasks are complete. Consider finishing current work first.",
		},
		{
			Type:     "max_tasks_per_workstream",
			Value:    10,
			Enforced: false,
			Message:  "Workstream has many tasks. Consider breaking into sub-workstreams.",
		},
	}
}

// FragmentationViolation records a specific rule violation.
type FragmentationViolation struct {
	Rule    AntiFragmentationRule `json:"rule"`
	Details string               `json:"details"`
}

// CheckAntiFragmentation evaluates anti-fragmentation rules against current project state.
func CheckAntiFragmentation(db *sql.DB, projectID string, rules []AntiFragmentationRule) []FragmentationViolation {
	var violations []FragmentationViolation

	workstreams, err := ListWorkstreams(db, projectID, "active")
	if err != nil {
		return violations
	}

	for _, rule := range rules {
		switch rule.Type {
		case "max_active_workstreams":
			if len(workstreams) > rule.Value {
				violations = append(violations, FragmentationViolation{
					Rule:    rule,
					Details: formatViolation("Active workstreams: %d (max: %d)", len(workstreams), rule.Value),
				})
			}

		case "min_completion_before_new":
			for _, ws := range workstreams {
				total := len(ws.RelatedTaskIDs)
				if total == 0 {
					continue
				}
				completed := 0
				for _, tid := range ws.RelatedTaskIDs {
					t, err := GetTaskV2(db, tid)
					if err == nil && t.StatusV2 == TaskStatusSucceeded {
						completed++
					}
				}
				pct := (completed * 100) / total
				if pct < rule.Value {
					violations = append(violations, FragmentationViolation{
						Rule:    rule,
						Details: formatViolation("Workstream '%s': %d%% complete (min: %d%%)", ws.Title, pct, rule.Value),
					})
				}
			}

		case "max_tasks_per_workstream":
			for _, ws := range workstreams {
				if len(ws.RelatedTaskIDs) > rule.Value {
					violations = append(violations, FragmentationViolation{
						Rule:    rule,
						Details: formatViolation("Workstream '%s': %d tasks (max: %d)", ws.Title, len(ws.RelatedTaskIDs), rule.Value),
					})
				}
			}
		}
	}

	return violations
}

func formatViolation(format string, args ...interface{}) string {
	result := format
	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			result = replaceFirst(result, "%d", intToStr(v))
		case string:
			result = replaceFirst(result, "%s", v)
		}
	}
	return result
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

// ---- Planning Modes (Task 16) ----

// PlanningModeConfig defines behavior overrides for each planning mode.
type PlanningModeConfig struct {
	Mode               string  `json:"mode"`
	MaxActiveWorkstreams int   `json:"max_active_workstreams"`
	MaxTasksPerCycle    int    `json:"max_tasks_per_cycle"`
	PrioritizeFailures  bool   `json:"prioritize_failures"`
	AllowNewWorkstreams bool   `json:"allow_new_workstreams"`
	ContinuityBoost     float64 `json:"continuity_boost"` // multiplier for continuity scores
	Description         string  `json:"description"`
}

// PlanningModeConfigs returns the configuration for all planning modes.
func PlanningModeConfigs() map[string]PlanningModeConfig {
	return map[string]PlanningModeConfig{
		"standard": {
			Mode:                "standard",
			MaxActiveWorkstreams: 5,
			MaxTasksPerCycle:    3,
			PrioritizeFailures:  false,
			AllowNewWorkstreams: true,
			ContinuityBoost:    1.0,
			Description:        "Default mode. Balanced between continuing work and starting new threads.",
		},
		"focused": {
			Mode:                "focused",
			MaxActiveWorkstreams: 1,
			MaxTasksPerCycle:    2,
			PrioritizeFailures:  false,
			AllowNewWorkstreams: false,
			ContinuityBoost:    1.5,
			Description:        "Single workstream focus. No new workstreams until current one completes.",
		},
		"recovery": {
			Mode:                "recovery",
			MaxActiveWorkstreams: 3,
			MaxTasksPerCycle:    5,
			PrioritizeFailures:  true,
			AllowNewWorkstreams: false,
			ContinuityBoost:    0.5,
			Description:        "Failure recovery mode. Prioritizes fixing failed tasks over new work.",
		},
		"maintenance": {
			Mode:                "maintenance",
			MaxActiveWorkstreams: 2,
			MaxTasksPerCycle:    1,
			PrioritizeFailures:  true,
			AllowNewWorkstreams: false,
			ContinuityBoost:    1.0,
			Description:        "Low-activity mode. Minimal new tasks, focus on keeping things running.",
		},
	}
}

// GetPlanningModeConfig returns the config for a given mode, defaulting to standard.
func GetPlanningModeConfig(mode string) PlanningModeConfig {
	configs := PlanningModeConfigs()
	if cfg, ok := configs[mode]; ok {
		return cfg
	}
	return configs["standard"]
}

// ApplyModeToConfig applies planning mode overrides to a pipeline config.
func ApplyModeToConfig(mode string, cfg *PipelineConfig) {
	modeConfig := GetPlanningModeConfig(mode)
	cfg.MaxTasksPerCycle = modeConfig.MaxTasksPerCycle
}
