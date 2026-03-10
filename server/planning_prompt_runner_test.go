package server

import (
	"fmt"
	"testing"
)

// mockExecutor is a test StageExecutor that returns a fixed JSON response.
type mockExecutor struct {
	response string
	err      error
}

func (m *mockExecutor) Execute(systemPrompt, userPrompt string, outputSchema *string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestPromptRunner_FallbackWhenNoPrompt(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "no-prompts", "", nil)

	// Run recon stage with prompt runner — no prompts seeded, should fallback
	result := runReconStagePrompt(db, proj.ID, &PlanningState{ProjectID: proj.ID}, nil)
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	source, _ := result.Output["_stage_source"].(string)
	if source != "heuristic" {
		t.Fatalf("expected source=heuristic, got %q", source)
	}
}

func TestPromptRunner_FallbackWhenNoExecutor(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "with-prompts", "", nil)
	// Seed built-in prompts
	if err := SeedBuiltInPrompts(db, proj.ID); err != nil {
		t.Fatalf("SeedBuiltInPrompts: %v", err)
	}

	// Run recon stage — prompts exist but no executor
	result := runReconStagePrompt(db, proj.ID, &PlanningState{ProjectID: proj.ID}, nil)
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	source, _ := result.Output["_stage_source"].(string)
	if source != "heuristic" {
		t.Fatalf("expected source=heuristic (no executor), got %q", source)
	}

	// Prompt metadata should still be recorded
	if _, ok := result.Output["_prompt_template_id"]; !ok {
		t.Fatal("expected _prompt_template_id to be recorded even on fallback")
	}
	if _, ok := result.Output["_prompt_rendered"]; !ok {
		t.Fatal("expected _prompt_rendered to be recorded even on fallback")
	}
}

func TestPromptRunner_ExecutorSuccess(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "executor-test", "", nil)
	SeedBuiltInPrompts(db, proj.ID)

	executor := &mockExecutor{
		response: `{"summary":"LLM says project is healthy","risk_level":"low","recommendations":["keep going"]}`,
	}

	result := runReconStagePrompt(db, proj.ID, &PlanningState{ProjectID: proj.ID}, executor)
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	source, _ := result.Output["_stage_source"].(string)
	if source != "prompt" {
		t.Fatalf("expected source=prompt, got %q", source)
	}

	summary, _ := result.Output["summary"].(string)
	if summary != "LLM says project is healthy" {
		t.Fatalf("expected prompt output summary, got %q", summary)
	}
}

func TestPromptRunner_ExecutorFailureFallback(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "exec-fail", "", nil)
	SeedBuiltInPrompts(db, proj.ID)

	executor := &mockExecutor{
		err: fmt.Errorf("LLM service unavailable"),
	}

	result := runReconStagePrompt(db, proj.ID, &PlanningState{ProjectID: proj.ID}, executor)
	if !result.Success {
		t.Fatalf("expected success (via fallback), got error: %s", result.Error)
	}

	source, _ := result.Output["_stage_source"].(string)
	if source != "heuristic" {
		t.Fatalf("expected source=heuristic after executor failure, got %q", source)
	}

	// Error should be recorded
	if _, ok := result.Output["_prompt_error"]; !ok {
		t.Fatal("expected _prompt_error to be recorded on executor failure")
	}
}

func TestPromptRunner_InvalidResponseFallback(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "bad-json", "", nil)
	SeedBuiltInPrompts(db, proj.ID)

	executor := &mockExecutor{
		response: "this is not valid JSON",
	}

	result := runReconStagePrompt(db, proj.ID, &PlanningState{ProjectID: proj.ID}, executor)
	if !result.Success {
		t.Fatalf("expected success (via fallback), got error: %s", result.Error)
	}

	source, _ := result.Output["_stage_source"].(string)
	if source != "heuristic" {
		t.Fatalf("expected source=heuristic after bad JSON, got %q", source)
	}
}

func TestPromptRunner_AllStagesHaveRoleMapping(t *testing.T) {
	stages := []PipelineStage{
		StageRecon, StageContinuity, StageGap,
		StagePrioritization, StageSynthesis, StageAntiDrift,
	}
	for _, s := range stages {
		if _, ok := stageRoleMap[s]; !ok {
			t.Errorf("stage %q has no role mapping in stageRoleMap", s)
		}
	}
}

func TestPromptRunner_ValidateOutputSchema(t *testing.T) {
	schema := `{"type":"object","properties":{"summary":{"type":"string"},"items":{"type":"array"}},"required":["summary"]}`

	// Valid output with required field
	valid := map[string]interface{}{"summary": "ok", "items": []string{"a"}}
	if !validateOutputSchema(valid, schema) {
		t.Fatal("expected valid output to pass schema validation")
	}

	// Missing required field
	missing := map[string]interface{}{"items": []string{"a"}}
	if validateOutputSchema(missing, schema) {
		t.Fatal("expected missing required field to fail schema validation")
	}
}

func TestPromptRunner_PipelineRecordsStageSource(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	proj, _ := CreateProject(db, "pipeline-source-test", "", nil)

	// Run pipeline with no executor — all stages should fall back to heuristic
	cfg := DefaultPipelineConfig()
	result, err := RunPlanningPipeline(db, proj.ID, cfg)
	if err != nil {
		t.Fatalf("RunPlanningPipeline: %v", err)
	}

	for _, stage := range result.Stages {
		if stage.Output == nil {
			continue
		}
		source, _ := stage.Output["_stage_source"].(string)
		if source != "heuristic" && source != "" {
			t.Errorf("stage %s: expected source=heuristic, got %q", stage.Stage, source)
		}
	}
}
