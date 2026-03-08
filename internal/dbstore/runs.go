package dbstore

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

func CreateRun(db *sql.DB, taskID int, agentID string) (*models.Run, error) {
	run := &models.Run{
		ID:        "run_" + uuid.New().String(),
		TaskID:    taskID,
		AgentID:   agentID,
		Status:    models.RunStatusRunning,
		StartedAt: models.NowISO(),
	}

	_, err := db.Exec(`
		INSERT INTO runs (id, task_id, agent_id, status, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, run.ID, run.TaskID, run.AgentID, run.Status, run.StartedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	task, taskErr := GetTaskV2(db, taskID)
	if taskErr == nil {
		CreateEvent(db, task.ProjectID, "run.started", "run", run.ID, map[string]interface{}{
			"task_id":  taskID,
			"agent_id": agentID,
			"run_id":   run.ID,
		})
	}

	return run, nil
}

func GetRun(db *sql.DB, runID string) (*models.Run, error) {
	var run models.Run
	var finishedAt, errorMsg sql.NullString

	err := db.QueryRow(`
		SELECT id, task_id, agent_id, status, started_at, finished_at, error
		FROM runs WHERE id = ?
	`, runID).Scan(&run.ID, &run.TaskID, &run.AgentID, &run.Status, &run.StartedAt, &finishedAt, &errorMsg)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", runID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	run.FinishedAt = models.PtrString(finishedAt)
	run.Error = models.PtrString(errorMsg)
	return &run, nil
}

func GetRunsByTaskID(db *sql.DB, taskID int) ([]models.Run, error) {
	rows, err := db.Query(`
		SELECT id, task_id, agent_id, status, started_at, finished_at, error
		FROM runs WHERE task_id = ? ORDER BY started_at DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query runs: %w", err)
	}
	defer rows.Close()

	var runs []models.Run
	for rows.Next() {
		var run models.Run
		var finishedAt, errorMsg sql.NullString
		err := rows.Scan(&run.ID, &run.TaskID, &run.AgentID, &run.Status, &run.StartedAt, &finishedAt, &errorMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}
		run.FinishedAt = models.PtrString(finishedAt)
		run.Error = models.PtrString(errorMsg)
		runs = append(runs, run)
	}
	return runs, nil
}

func UpdateRunStatus(db *sql.DB, runID string, status models.RunStatus, errorMsg *string) error {
	finishedAt := models.NowISO()
	_, err := db.Exec(`
		UPDATE runs SET status = ?, finished_at = ?, error = ?
		WHERE id = ?
	`, status, finishedAt, models.NullString(errorMsg), runID)

	if err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	run, runErr := GetRun(db, runID)
	if runErr == nil {
		eventKind := "run.completed"
		if status == models.RunStatusFailed {
			eventKind = "run.failed"
		}
		task, taskErr := GetTaskV2(db, run.TaskID)
		if taskErr == nil {
			payload := map[string]interface{}{
				"run_id":  runID,
				"task_id": run.TaskID,
				"status":  string(status),
			}
			if errorMsg != nil {
				payload["error"] = *errorMsg
			}
			CreateEvent(db, task.ProjectID, eventKind, "run", runID, payload)
		}
	}

	return nil
}

func DeleteRun(db *sql.DB, runID string) error {
	_, err := db.Exec("DELETE FROM runs WHERE id = ?", runID)
	if err != nil {
		return fmt.Errorf("failed to delete run: %w", err)
	}
	return nil
}

func CreateRunStep(db *sql.DB, runID, name string, status models.StepStatus, details map[string]interface{}) (*models.RunStep, error) {
	step := &models.RunStep{
		ID:        "step_" + uuid.New().String(),
		RunID:     runID,
		Name:      name,
		Status:    status,
		Details:   details,
		CreatedAt: models.NowISO(),
	}

	detailsJSON, err := models.MarshalJSON(details)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal details: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO run_steps (id, run_id, name, status, details_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, step.ID, step.RunID, step.Name, step.Status, detailsJSON, step.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create run step: %w", err)
	}
	return step, nil
}

func GetRunSteps(db *sql.DB, runID string) ([]models.RunStep, error) {
	rows, err := db.Query(`
		SELECT id, run_id, name, status, details_json, created_at
		FROM run_steps WHERE run_id = ? ORDER BY created_at ASC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query run steps: %w", err)
	}
	defer rows.Close()

	var steps []models.RunStep
	for rows.Next() {
		var step models.RunStep
		var detailsJSON sql.NullString
		err := rows.Scan(&step.ID, &step.RunID, &step.Name, &step.Status, &detailsJSON, &step.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run step: %w", err)
		}
		if detailsJSON.Valid && detailsJSON.String != "" {
			if err := models.UnmarshalJSON(detailsJSON.String, &step.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func UpdateRunStepStatus(db *sql.DB, stepID string, status models.StepStatus) error {
	_, err := db.Exec("UPDATE run_steps SET status = ? WHERE id = ?", status, stepID)
	if err != nil {
		return fmt.Errorf("failed to update run step status: %w", err)
	}
	return nil
}

func CreateRunLog(db *sql.DB, runID, stream, chunk string) error {
	_, err := db.Exec(`
		INSERT INTO run_logs (run_id, stream, chunk, ts)
		VALUES (?, ?, ?, ?)
	`, runID, stream, chunk, models.NowISO())

	if err != nil {
		return fmt.Errorf("failed to create run log: %w", err)
	}
	return nil
}

func GetRunLogs(db *sql.DB, runID string) ([]models.RunLog, error) {
	rows, err := db.Query(`
		SELECT id, run_id, stream, chunk, ts
		FROM run_logs WHERE run_id = ? ORDER BY ts ASC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query run logs: %w", err)
	}
	defer rows.Close()

	var logs []models.RunLog
	for rows.Next() {
		var log models.RunLog
		err := rows.Scan(&log.ID, &log.RunID, &log.Stream, &log.Chunk, &log.Ts)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run log: %w", err)
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func CreateArtifact(db *sql.DB, runID, kind, storageRef string, sha256 *string, size *int64, metadata map[string]interface{}) (*models.Artifact, error) {
	artifact := &models.Artifact{
		ID:         "art_" + uuid.New().String(),
		RunID:      runID,
		Kind:       kind,
		StorageRef: storageRef,
		Sha256:     sha256,
		Size:       size,
		Metadata:   metadata,
		CreatedAt:  models.NowISO(),
	}

	metadataJSON, err := models.MarshalJSON(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO artifacts (id, run_id, kind, storage_ref, sha256, size, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, artifact.ID, artifact.RunID, artifact.Kind, artifact.StorageRef,
		models.NullString(artifact.Sha256), ptrInt64ToNullInt64(artifact.Size), metadataJSON, artifact.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create artifact: %w", err)
	}
	return artifact, nil
}

func GetArtifactsByRunID(db *sql.DB, runID string) ([]models.Artifact, error) {
	rows, err := db.Query(`
		SELECT id, run_id, kind, storage_ref, sha256, size, metadata_json, created_at
		FROM artifacts WHERE run_id = ? ORDER BY created_at ASC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var artifact models.Artifact
		var sha256, metadataJSON sql.NullString
		var size sql.NullInt64
		err := rows.Scan(&artifact.ID, &artifact.RunID, &artifact.Kind, &artifact.StorageRef,
			&sha256, &size, &metadataJSON, &artifact.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		artifact.Sha256 = models.PtrString(sha256)
		artifact.Size = models.PtrInt64(size)
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := models.UnmarshalJSON(metadataJSON.String, &artifact.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func CreateToolInvocation(db *sql.DB, runID, toolName string, input map[string]interface{}) (*models.ToolInvocation, error) {
	invocation := &models.ToolInvocation{
		ID:        "tool_" + uuid.New().String(),
		RunID:     runID,
		ToolName:  toolName,
		Input:     input,
		StartedAt: models.NowISO(),
	}

	inputJSON, err := models.MarshalJSON(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO tool_invocations (id, run_id, tool_name, input_json, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, invocation.ID, invocation.RunID, invocation.ToolName, inputJSON, invocation.StartedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create tool invocation: %w", err)
	}
	return invocation, nil
}

func UpdateToolInvocationOutput(db *sql.DB, invocationID string, output map[string]interface{}) error {
	outputJSON, err := models.MarshalJSON(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	_, err = db.Exec(`
		UPDATE tool_invocations SET output_json = ?, finished_at = ?
		WHERE id = ?
	`, outputJSON, models.NowISO(), invocationID)

	if err != nil {
		return fmt.Errorf("failed to update tool invocation: %w", err)
	}
	return nil
}

func GetToolInvocationsByRunID(db *sql.DB, runID string) ([]models.ToolInvocation, error) {
	rows, err := db.Query(`
		SELECT id, run_id, tool_name, input_json, output_json, started_at, finished_at
		FROM tool_invocations WHERE run_id = ? ORDER BY started_at ASC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool invocations: %w", err)
	}
	defer rows.Close()

	var invocations []models.ToolInvocation
	for rows.Next() {
		var invocation models.ToolInvocation
		var inputJSON, outputJSON, finishedAt sql.NullString
		err := rows.Scan(&invocation.ID, &invocation.RunID, &invocation.ToolName,
			&inputJSON, &outputJSON, &invocation.StartedAt, &finishedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool invocation: %w", err)
		}
		if inputJSON.Valid && inputJSON.String != "" {
			if err := models.UnmarshalJSON(inputJSON.String, &invocation.Input); err != nil {
				return nil, fmt.Errorf("failed to unmarshal input: %w", err)
			}
		}
		if outputJSON.Valid && outputJSON.String != "" {
			if err := models.UnmarshalJSON(outputJSON.String, &invocation.Output); err != nil {
				return nil, fmt.Errorf("failed to unmarshal output: %w", err)
			}
		}
		invocation.FinishedAt = models.PtrString(finishedAt)
		invocations = append(invocations, invocation)
	}
	return invocations, nil
}
