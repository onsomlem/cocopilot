package server

import (
	"database/sql"
	"encoding/json"
	"log"
)

// stageRoleMap maps pipeline stages to prompt template roles.
var stageRoleMap = map[PipelineStage]string{
	StageRecon:          "idle_recon",
	StageContinuity:     "idle_continuity",
	StageGap:            "idle_gap",
	StagePrioritization: "idle_prioritization",
	StageSynthesis:      "task_synthesis",
	StageAntiDrift:      "anti_drift",
}

// StageExecutor defines the interface for executing a rendered prompt.
// When an LLM backend is available, implement this interface to get real
// prompt-driven planning. Without one, the pipeline records the rendered
// prompt and falls back to deterministic heuristics.
type StageExecutor interface {
	// Execute sends a system prompt and rendered user prompt to an LLM
	// and returns the raw response string. The caller handles parsing.
	Execute(systemPrompt, userPrompt string, outputSchema *string) (string, error)
}

// promptStageRunner is the internal state for running a prompt-driven stage.
type promptStageRunner struct {
	db        *sql.DB
	projectID string
	stage     PipelineStage
	executor  StageExecutor // nil = no LLM, record prompt + fallback
}

// PromptRunResult holds the outcome of a prompt-driven stage attempt.
type PromptRunResult struct {
	// UsedPrompt is true if a prompt template was found and rendered.
	UsedPrompt bool
	// PromptOutput is the parsed output from the prompt execution (if executor ran).
	PromptOutput map[string]interface{}
	// Rendered contains the rendered system/user prompts for inspection.
	Rendered *RenderedPrompt
	// FellBack is true if the stage used deterministic logic.
	FellBack bool
	// Error is set if prompt execution failed (before fallback).
	Error string
}

// RenderedPrompt captures the rendered prompt for a stage, stored in output for inspection.
type RenderedPrompt struct {
	TemplateID   string `json:"template_id"`
	Role         string `json:"role"`
	Version      int    `json:"version"`
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
}

// runWithPrompt attempts to run a stage using its prompt template.
// It loads the active prompt for the stage's role, builds context, renders
// the template, and optionally executes it via the StageExecutor.
// Returns the result indicating whether a prompt was used and any output.
func (r *promptStageRunner) runWithPrompt(ctx map[string]interface{}) *PromptRunResult {
	role, ok := stageRoleMap[r.stage]
	if !ok {
		return &PromptRunResult{FellBack: true, Error: "no role mapping for stage"}
	}

	pt, err := GetActivePromptByRole(r.db, r.projectID, role)
	if err != nil || pt == nil {
		return &PromptRunResult{FellBack: true}
	}

	// Render the template with provided context
	rendered := renderTemplate(pt.UserTemplate, ctx)

	rp := &RenderedPrompt{
		TemplateID:   pt.ID,
		Role:         pt.Role,
		Version:      pt.Version,
		SystemPrompt: pt.SystemPrompt,
		UserPrompt:   rendered,
	}

	result := &PromptRunResult{
		UsedPrompt: true,
		Rendered:   rp,
	}

	// If no executor, record the prompt and fall back to heuristics
	if r.executor == nil {
		result.FellBack = true
		return result
	}

	// Execute the prompt
	response, err := r.executor.Execute(pt.SystemPrompt, rendered, pt.OutputSchema)
	if err != nil {
		log.Printf("planning: prompt execution failed for %s/%s: %v", r.projectID, role, err)
		result.FellBack = true
		result.Error = err.Error()
		return result
	}

	// Parse response as JSON
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		log.Printf("planning: prompt response parse failed for %s/%s: %v", r.projectID, role, err)
		result.FellBack = true
		result.Error = "response parse failed: " + err.Error()
		return result
	}

	// Validate output against schema if available
	if pt.OutputSchema != nil && *pt.OutputSchema != "" {
		if !validateOutputSchema(output, *pt.OutputSchema) {
			log.Printf("planning: prompt output schema validation failed for %s/%s", r.projectID, role)
			result.FellBack = true
			result.Error = "output schema validation failed"
			return result
		}
	}

	result.PromptOutput = output
	result.FellBack = false
	return result
}

// mergePromptMetadata adds prompt execution metadata to a stage output map.
func mergePromptMetadata(output map[string]interface{}, pr *PromptRunResult) {
	if pr == nil {
		output["_stage_source"] = "heuristic"
		return
	}
	if pr.FellBack {
		output["_stage_source"] = "heuristic"
	} else {
		output["_stage_source"] = "prompt"
	}

	if pr.Rendered != nil {
		output["_prompt_template_id"] = pr.Rendered.TemplateID
		output["_prompt_role"] = pr.Rendered.Role
		output["_prompt_version"] = pr.Rendered.Version
		output["_prompt_rendered"] = pr.Rendered.UserPrompt
	}

	if pr.Error != "" {
		output["_prompt_error"] = pr.Error
	}
}

// validateOutputSchema does basic structural validation of output against a JSON schema.
// It checks that required top-level keys from the schema are present in the output.
func validateOutputSchema(output map[string]interface{}, schemaJSON string) bool {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return false
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return true // no properties to validate
	}

	// Check required fields if specified
	if required, ok := schema["required"].([]interface{}); ok {
		for _, r := range required {
			key, ok := r.(string)
			if !ok {
				continue
			}
			if _, exists := output[key]; !exists {
				return false
			}
		}
	}

	// Check that at least some expected keys are present (lenient validation)
	matchCount := 0
	for key := range props {
		if _, exists := output[key]; exists {
			matchCount++
		}
	}
	return matchCount > 0 || len(props) == 0
}

// ---- Prompt-driven stage wrappers ----

// runReconStagePrompt tries prompt-driven recon, falling back to heuristics.
func runReconStagePrompt(db *sql.DB, projectID string, ps *PlanningState, executor StageExecutor) StageResult {
	runner := &promptStageRunner{db: db, projectID: projectID, stage: StageRecon, executor: executor}

	// Build context for the prompt
	ctx := map[string]interface{}{
		"project_id":    projectID,
		"recon_summary": ps.ReconSummary,
		"goals":         formatStringSlice(ps.Goals),
		"blockers":      formatStringSlice(ps.Blockers),
		"risks":         formatStringSlice(ps.Risks),
	}

	pr := runner.runWithPrompt(ctx)

	// If prompt produced output, use it
	if pr != nil && !pr.FellBack && pr.PromptOutput != nil {
		mergePromptMetadata(pr.PromptOutput, pr)
		return StageResult{Stage: StageRecon, Success: true, Output: pr.PromptOutput}
	}

	// Fallback to deterministic
	result := runReconStage(db, projectID, ps)
	mergePromptMetadata(result.Output, pr)
	return result
}

// runContinuityStagePrompt tries prompt-driven continuity, falling back to heuristics.
func runContinuityStagePrompt(db *sql.DB, projectID string, ps *PlanningState, reconOutput map[string]interface{}, executor StageExecutor) StageResult {
	runner := &promptStageRunner{db: db, projectID: projectID, stage: StageContinuity, executor: executor}

	// Build context for the prompt
	workstreams, _ := ListWorkstreams(db, projectID, "")
	ctx := map[string]interface{}{
		"project_id":    projectID,
		"workstreams":   formatWorkstreamsForPrompt(workstreams),
		"recon_summary": mapGet(reconOutput, "summary", ""),
	}

	pr := runner.runWithPrompt(ctx)

	if pr != nil && !pr.FellBack && pr.PromptOutput != nil {
		mergePromptMetadata(pr.PromptOutput, pr)
		return StageResult{Stage: StageContinuity, Success: true, Output: pr.PromptOutput}
	}

	result := runContinuityStage(db, projectID, ps, reconOutput)
	mergePromptMetadata(result.Output, pr)
	return result
}

// runGapStagePrompt tries prompt-driven gap analysis, falling back to heuristics.
func runGapStagePrompt(db *sql.DB, projectID string, ps *PlanningState, reconOutput, continuityOutput map[string]interface{}, executor StageExecutor) StageResult {
	runner := &promptStageRunner{db: db, projectID: projectID, stage: StageGap, executor: executor}

	ctx := map[string]interface{}{
		"project_id":    projectID,
		"goals":         formatStringSlice(ps.Goals),
		"workstreams":   mapGet(continuityOutput, "continuation_candidates", "[]"),
		"recon_summary": mapGet(reconOutput, "summary", ""),
	}

	pr := runner.runWithPrompt(ctx)

	if pr != nil && !pr.FellBack && pr.PromptOutput != nil {
		mergePromptMetadata(pr.PromptOutput, pr)
		return StageResult{Stage: StageGap, Success: true, Output: pr.PromptOutput}
	}

	result := runGapStage(db, projectID, ps, reconOutput, continuityOutput)
	mergePromptMetadata(result.Output, pr)
	return result
}

// runPrioritizationStagePrompt tries prompt-driven prioritization, falling back to heuristics.
func runPrioritizationStagePrompt(db *sql.DB, projectID string, ps *PlanningState, continuityOutput, gapOutput map[string]interface{}, executor StageExecutor) StageResult {
	runner := &promptStageRunner{db: db, projectID: projectID, stage: StagePrioritization, executor: executor}

	ctx := map[string]interface{}{
		"project_id":    projectID,
		"continuity":    marshalJSONCompact(continuityOutput),
		"gaps":          marshalJSONCompact(gapOutput),
		"planning_mode": ps.PlanningMode,
	}

	pr := runner.runWithPrompt(ctx)

	if pr != nil && !pr.FellBack && pr.PromptOutput != nil {
		mergePromptMetadata(pr.PromptOutput, pr)
		return StageResult{Stage: StagePrioritization, Success: true, Output: pr.PromptOutput}
	}

	result := runPrioritizationStage(db, projectID, ps, continuityOutput, gapOutput)
	mergePromptMetadata(result.Output, pr)
	return result
}

// runSynthesisStagePrompt tries prompt-driven task synthesis, falling back to heuristics.
func runSynthesisStagePrompt(db *sql.DB, projectID string, ps *PlanningState, prioOutput map[string]interface{}, cfg PipelineConfig, executor StageExecutor) StageResult {
	runner := &promptStageRunner{db: db, projectID: projectID, stage: StageSynthesis, executor: executor}

	// Build existing tasks context for dedup
	existingTasks, _, _ := ListTasksV2(db, projectID, "status_v2", "", "", "", "", 100, 0, "created_at", "desc")
	var existingTitles []string
	for _, t := range existingTasks {
		if t.Title != nil {
			existingTitles = append(existingTitles, *t.Title)
		}
	}

	ctx := map[string]interface{}{
		"project_id":     projectID,
		"ranked_items":   marshalJSONCompact(prioOutput),
		"max_tasks":      cfg.MaxTasksPerCycle,
		"existing_tasks": formatStringSlice(existingTitles),
	}

	pr := runner.runWithPrompt(ctx)

	if pr != nil && !pr.FellBack && pr.PromptOutput != nil {
		mergePromptMetadata(pr.PromptOutput, pr)
		return StageResult{Stage: StageSynthesis, Success: true, Output: pr.PromptOutput}
	}

	result := runSynthesisStage(db, projectID, ps, prioOutput, cfg)
	mergePromptMetadata(result.Output, pr)
	return result
}

// runAntiDriftStagePrompt tries prompt-driven anti-drift validation, falling back to heuristics.
func runAntiDriftStagePrompt(db *sql.DB, projectID string, ps *PlanningState, synthOutput map[string]interface{}, executor StageExecutor) StageResult {
	runner := &promptStageRunner{db: db, projectID: projectID, stage: StageAntiDrift, executor: executor}

	ctx := map[string]interface{}{
		"project_id":      projectID,
		"proposed_tasks":  marshalJSONCompact(synthOutput),
		"goals":           formatStringSlice(ps.Goals),
		"must_not_forget": formatStringSlice(ps.MustNotForget),
	}

	pr := runner.runWithPrompt(ctx)

	if pr != nil && !pr.FellBack && pr.PromptOutput != nil {
		mergePromptMetadata(pr.PromptOutput, pr)
		return StageResult{Stage: StageAntiDrift, Success: true, Output: pr.PromptOutput}
	}

	result := runAntiDriftStage(db, projectID, ps, synthOutput)
	mergePromptMetadata(result.Output, pr)
	return result
}

// ---- Helpers ----

// formatWorkstreamsForPrompt formats workstreams for template rendering.
func formatWorkstreamsForPrompt(workstreams []Workstream) string {
	if len(workstreams) == 0 {
		return "No workstreams defined"
	}
	b, _ := json.Marshal(workstreams)
	return string(b)
}

// formatStringSlice formats a string slice for template rendering.
func formatStringSlice(items []string) string {
	if len(items) == 0 {
		return "None"
	}
	b, _ := json.Marshal(items)
	return string(b)
}

// marshalJSONCompact marshals any value to compact JSON string.
func marshalJSONCompact(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// mapGet safely extracts a string value from a map with a default fallback.
func mapGet(m map[string]interface{}, key, fallback string) string {
	if m == nil {
		return fallback
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		b, _ := json.Marshal(v)
		return string(b)
	}
	return fallback
}
