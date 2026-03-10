package dbstore

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/onsomlem/cocopilot/internal/models"
)

// ---- Planning State ----

func GetPlanningState(db *sql.DB, projectID string) (*models.PlanningState, error) {
	var ps models.PlanningState
	var lastCycleAt sql.NullString
	var goalsJSON, mustNotForgetJSON, blockersJSON, risksJSON, priorityOrderJSON string

	err := db.QueryRow(`
		SELECT id, project_id, planning_mode, cycle_count, last_cycle_at,
		       goals, release_focus, must_not_forget, recon_summary, planner_summary,
		       blockers, risks, priority_order, created_at, updated_at
		FROM planning_state WHERE project_id = ?
	`, projectID).Scan(
		&ps.ID, &ps.ProjectID, &ps.PlanningMode, &ps.CycleCount, &lastCycleAt,
		&goalsJSON, &ps.ReleaseFocus, &mustNotForgetJSON, &ps.ReconSummary, &ps.PlannerSummary,
		&blockersJSON, &risksJSON, &priorityOrderJSON, &ps.CreatedAt, &ps.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get planning state: %w", err)
	}

	ps.LastCycleAt = models.PtrString(lastCycleAt)
	_ = models.UnmarshalJSON(goalsJSON, &ps.Goals)
	_ = models.UnmarshalJSON(mustNotForgetJSON, &ps.MustNotForget)
	_ = models.UnmarshalJSON(blockersJSON, &ps.Blockers)
	_ = models.UnmarshalJSON(risksJSON, &ps.Risks)
	_ = models.UnmarshalJSON(priorityOrderJSON, &ps.PriorityOrder)
	if ps.Goals == nil {
		ps.Goals = []string{}
	}
	if ps.MustNotForget == nil {
		ps.MustNotForget = []string{}
	}
	if ps.Blockers == nil {
		ps.Blockers = []string{}
	}
	if ps.Risks == nil {
		ps.Risks = []string{}
	}
	if ps.PriorityOrder == nil {
		ps.PriorityOrder = []string{}
	}

	return &ps, nil
}

func CreatePlanningState(db *sql.DB, projectID string) (*models.PlanningState, error) {
	now := models.NowISO()
	ps := &models.PlanningState{
		ID:             "plan_" + uuid.New().String(),
		ProjectID:      projectID,
		PlanningMode:   string(models.PlanningModeStandard),
		CycleCount:     0,
		Goals:          []string{},
		ReleaseFocus:   "",
		MustNotForget:  []string{},
		ReconSummary:   "",
		PlannerSummary: "",
		Blockers:       []string{},
		Risks:          []string{},
		PriorityOrder:  []string{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	goalsJSON, _ := models.MarshalJSON(ps.Goals)
	mustNotForgetJSON, _ := models.MarshalJSON(ps.MustNotForget)
	blockersJSON, _ := models.MarshalJSON(ps.Blockers)
	risksJSON, _ := models.MarshalJSON(ps.Risks)
	priorityOrderJSON, _ := models.MarshalJSON(ps.PriorityOrder)

	_, err := db.Exec(`
		INSERT INTO planning_state (id, project_id, planning_mode, cycle_count, goals,
		    release_focus, must_not_forget, recon_summary, planner_summary,
		    blockers, risks, priority_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ps.ID, ps.ProjectID, ps.PlanningMode, ps.CycleCount, goalsJSON,
		ps.ReleaseFocus, mustNotForgetJSON, ps.ReconSummary, ps.PlannerSummary,
		blockersJSON, risksJSON, priorityOrderJSON, ps.CreatedAt, ps.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create planning state: %w", err)
	}

	return ps, nil
}

func UpdatePlanningState(db *sql.DB, ps *models.PlanningState) error {
	ps.UpdatedAt = models.NowISO()
	goalsJSON, _ := models.MarshalJSON(ps.Goals)
	mustNotForgetJSON, _ := models.MarshalJSON(ps.MustNotForget)
	blockersJSON, _ := models.MarshalJSON(ps.Blockers)
	risksJSON, _ := models.MarshalJSON(ps.Risks)
	priorityOrderJSON, _ := models.MarshalJSON(ps.PriorityOrder)

	_, err := db.Exec(`
		UPDATE planning_state SET planning_mode = ?, cycle_count = ?, last_cycle_at = ?,
		    goals = ?, release_focus = ?, must_not_forget = ?,
		    recon_summary = ?, planner_summary = ?,
		    blockers = ?, risks = ?, priority_order = ?, updated_at = ?
		WHERE id = ?
	`, ps.PlanningMode, ps.CycleCount, models.NullString(ps.LastCycleAt),
		goalsJSON, ps.ReleaseFocus, mustNotForgetJSON,
		ps.ReconSummary, ps.PlannerSummary,
		blockersJSON, risksJSON, priorityOrderJSON, ps.UpdatedAt, ps.ID)
	if err != nil {
		return fmt.Errorf("failed to update planning state: %w", err)
	}
	return nil
}

func GetOrCreatePlanningState(db *sql.DB, projectID string) (*models.PlanningState, error) {
	ps, err := GetPlanningState(db, projectID)
	if err != nil {
		return nil, err
	}
	if ps != nil {
		return ps, nil
	}
	return CreatePlanningState(db, projectID)
}

// ---- Workstreams ----

func CreateWorkstream(db *sql.DB, projectID, planningStateID, title, description, why, whatNext string) (*models.Workstream, error) {
	now := models.NowISO()
	ws := &models.Workstream{
		ID:              "ws_" + uuid.New().String(),
		ProjectID:       projectID,
		PlanningStateID: planningStateID,
		Title:           strings.TrimSpace(title),
		Description:     strings.TrimSpace(description),
		Status:          models.WorkstreamStatusActive,
		ContinuityScore: 0.5,
		UrgencyScore:    0.5,
		RelatedTaskIDs:  []int{},
		RelatedRunIDs:   []int{},
		Why:             strings.TrimSpace(why),
		WhatRemains:     "",
		WhatNext:        strings.TrimSpace(whatNext),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	taskIDsJSON, _ := models.MarshalJSON(ws.RelatedTaskIDs)
	runIDsJSON, _ := models.MarshalJSON(ws.RelatedRunIDs)

	_, err := db.Exec(`
		INSERT INTO workstreams (id, project_id, planning_state_id, title, description,
		    status, continuity_score, urgency_score, related_task_ids, related_run_ids,
		    why, what_remains, what_next, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ws.ID, ws.ProjectID, ws.PlanningStateID, ws.Title, ws.Description,
		ws.Status, ws.ContinuityScore, ws.UrgencyScore, taskIDsJSON, runIDsJSON,
		ws.Why, ws.WhatRemains, ws.WhatNext, ws.CreatedAt, ws.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create workstream: %w", err)
	}

	return ws, nil
}

func GetWorkstream(db *sql.DB, projectID, workstreamID string) (*models.Workstream, error) {
	var ws models.Workstream
	var taskIDsJSON, runIDsJSON string

	err := db.QueryRow(`
		SELECT id, project_id, planning_state_id, title, description,
		       status, continuity_score, urgency_score, related_task_ids, related_run_ids,
		       why, what_remains, what_next, created_at, updated_at
		FROM workstreams WHERE project_id = ? AND id = ?
	`, projectID, workstreamID).Scan(
		&ws.ID, &ws.ProjectID, &ws.PlanningStateID, &ws.Title, &ws.Description,
		&ws.Status, &ws.ContinuityScore, &ws.UrgencyScore, &taskIDsJSON, &runIDsJSON,
		&ws.Why, &ws.WhatRemains, &ws.WhatNext, &ws.CreatedAt, &ws.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workstream: %w", err)
	}

	_ = models.UnmarshalJSON(taskIDsJSON, &ws.RelatedTaskIDs)
	_ = models.UnmarshalJSON(runIDsJSON, &ws.RelatedRunIDs)
	if ws.RelatedTaskIDs == nil {
		ws.RelatedTaskIDs = []int{}
	}
	if ws.RelatedRunIDs == nil {
		ws.RelatedRunIDs = []int{}
	}

	return &ws, nil
}

func ListWorkstreams(db *sql.DB, projectID string, statusFilter string) ([]models.Workstream, error) {
	query := `SELECT id, project_id, planning_state_id, title, description,
	                  status, continuity_score, urgency_score, related_task_ids, related_run_ids,
	                  why, what_remains, what_next, created_at, updated_at
	           FROM workstreams WHERE project_id = ?`
	args := []interface{}{projectID}

	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY continuity_score DESC, urgency_score DESC, created_at ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workstreams: %w", err)
	}
	defer rows.Close()

	var workstreams []models.Workstream
	for rows.Next() {
		var ws models.Workstream
		var taskIDsJSON, runIDsJSON string
		if err := rows.Scan(
			&ws.ID, &ws.ProjectID, &ws.PlanningStateID, &ws.Title, &ws.Description,
			&ws.Status, &ws.ContinuityScore, &ws.UrgencyScore, &taskIDsJSON, &runIDsJSON,
			&ws.Why, &ws.WhatRemains, &ws.WhatNext, &ws.CreatedAt, &ws.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workstream: %w", err)
		}
		_ = models.UnmarshalJSON(taskIDsJSON, &ws.RelatedTaskIDs)
		_ = models.UnmarshalJSON(runIDsJSON, &ws.RelatedRunIDs)
		if ws.RelatedTaskIDs == nil {
			ws.RelatedTaskIDs = []int{}
		}
		if ws.RelatedRunIDs == nil {
			ws.RelatedRunIDs = []int{}
		}
		workstreams = append(workstreams, ws)
	}
	return workstreams, rows.Err()
}

func UpdateWorkstream(db *sql.DB, ws *models.Workstream) error {
	ws.UpdatedAt = models.NowISO()
	taskIDsJSON, _ := models.MarshalJSON(ws.RelatedTaskIDs)
	runIDsJSON, _ := models.MarshalJSON(ws.RelatedRunIDs)

	_, err := db.Exec(`
		UPDATE workstreams SET title = ?, description = ?, status = ?,
		    continuity_score = ?, urgency_score = ?,
		    related_task_ids = ?, related_run_ids = ?,
		    why = ?, what_remains = ?, what_next = ?, updated_at = ?
		WHERE project_id = ? AND id = ?
	`, ws.Title, ws.Description, ws.Status,
		ws.ContinuityScore, ws.UrgencyScore,
		taskIDsJSON, runIDsJSON,
		ws.Why, ws.WhatRemains, ws.WhatNext, ws.UpdatedAt,
		ws.ProjectID, ws.ID)
	if err != nil {
		return fmt.Errorf("failed to update workstream: %w", err)
	}
	return nil
}

func DeleteWorkstream(db *sql.DB, projectID, workstreamID string) error {
	_, err := db.Exec("DELETE FROM workstreams WHERE project_id = ? AND id = ?", projectID, workstreamID)
	if err != nil {
		return fmt.Errorf("failed to delete workstream: %w", err)
	}
	return nil
}

// ---- Planning Cycles ----

func CreatePlanningCycle(db *sql.DB, projectID string, cycleNumber int, planningMode string) (*models.PlanningCycle, error) {
	pc := &models.PlanningCycle{
		ID:                   "cycle_" + uuid.New().String(),
		ProjectID:            projectID,
		CycleNumber:          cycleNumber,
		PlanningMode:         planningMode,
		StartedAt:            models.NowISO(),
		ReconOutput:          map[string]interface{}{},
		ContinuityOutput:     map[string]interface{}{},
		GapOutput:            map[string]interface{}{},
		PrioritizationOutput: map[string]interface{}{},
		SynthesisOutput:      map[string]interface{}{},
		AntiDriftOutput:      map[string]interface{}{},
		TasksCreated:         []int{},
		StageFailures:        []string{},
		DriftWarnings:        []string{},
	}

	reconJSON, _ := models.MarshalJSON(pc.ReconOutput)
	contJSON, _ := models.MarshalJSON(pc.ContinuityOutput)
	gapJSON, _ := models.MarshalJSON(pc.GapOutput)
	prioJSON, _ := models.MarshalJSON(pc.PrioritizationOutput)
	synthJSON, _ := models.MarshalJSON(pc.SynthesisOutput)
	driftJSON, _ := models.MarshalJSON(pc.AntiDriftOutput)
	tasksJSON, _ := models.MarshalJSON(pc.TasksCreated)
	failJSON, _ := models.MarshalJSON(pc.StageFailures)
	warnJSON, _ := models.MarshalJSON(pc.DriftWarnings)

	_, err := db.Exec(`
		INSERT INTO planning_cycles (id, project_id, cycle_number, planning_mode, started_at,
		    recon_output, continuity_output, gap_output, prioritization_output,
		    synthesis_output, anti_drift_output, tasks_created,
		    coherence_score, stage_failures, drift_warnings)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pc.ID, pc.ProjectID, pc.CycleNumber, pc.PlanningMode, pc.StartedAt,
		reconJSON, contJSON, gapJSON, prioJSON, synthJSON, driftJSON, tasksJSON,
		pc.CoherenceScore, failJSON, warnJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create planning cycle: %w", err)
	}

	return pc, nil
}

func CompletePlanningCycle(db *sql.DB, pc *models.PlanningCycle) error {
	now := models.NowISO()
	pc.CompletedAt = &now

	reconJSON, _ := models.MarshalJSON(pc.ReconOutput)
	contJSON, _ := models.MarshalJSON(pc.ContinuityOutput)
	gapJSON, _ := models.MarshalJSON(pc.GapOutput)
	prioJSON, _ := models.MarshalJSON(pc.PrioritizationOutput)
	synthJSON, _ := models.MarshalJSON(pc.SynthesisOutput)
	driftJSON, _ := models.MarshalJSON(pc.AntiDriftOutput)
	tasksJSON, _ := models.MarshalJSON(pc.TasksCreated)
	failJSON, _ := models.MarshalJSON(pc.StageFailures)
	warnJSON, _ := models.MarshalJSON(pc.DriftWarnings)

	_, err := db.Exec(`
		UPDATE planning_cycles SET completed_at = ?,
		    recon_output = ?, continuity_output = ?, gap_output = ?,
		    prioritization_output = ?, synthesis_output = ?, anti_drift_output = ?,
		    tasks_created = ?, coherence_score = ?,
		    stage_failures = ?, drift_warnings = ?
		WHERE id = ?
	`, now, reconJSON, contJSON, gapJSON, prioJSON, synthJSON, driftJSON,
		tasksJSON, pc.CoherenceScore, failJSON, warnJSON, pc.ID)
	if err != nil {
		return fmt.Errorf("failed to complete planning cycle: %w", err)
	}
	return nil
}

func GetPlanningCycle(db *sql.DB, cycleID string) (*models.PlanningCycle, error) {
	var pc models.PlanningCycle
	var completedAt sql.NullString
	var reconJSON, contJSON, gapJSON, prioJSON, synthJSON, driftJSON string
	var tasksJSON, failJSON, warnJSON string

	err := db.QueryRow(`
		SELECT id, project_id, cycle_number, planning_mode, started_at, completed_at,
		       recon_output, continuity_output, gap_output, prioritization_output,
		       synthesis_output, anti_drift_output, tasks_created,
		       coherence_score, stage_failures, drift_warnings
		FROM planning_cycles WHERE id = ?
	`, cycleID).Scan(
		&pc.ID, &pc.ProjectID, &pc.CycleNumber, &pc.PlanningMode,
		&pc.StartedAt, &completedAt,
		&reconJSON, &contJSON, &gapJSON, &prioJSON, &synthJSON, &driftJSON,
		&tasksJSON, &pc.CoherenceScore, &failJSON, &warnJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get planning cycle: %w", err)
	}

	pc.CompletedAt = models.PtrString(completedAt)
	_ = models.UnmarshalJSON(reconJSON, &pc.ReconOutput)
	_ = models.UnmarshalJSON(contJSON, &pc.ContinuityOutput)
	_ = models.UnmarshalJSON(gapJSON, &pc.GapOutput)
	_ = models.UnmarshalJSON(prioJSON, &pc.PrioritizationOutput)
	_ = models.UnmarshalJSON(synthJSON, &pc.SynthesisOutput)
	_ = models.UnmarshalJSON(driftJSON, &pc.AntiDriftOutput)
	_ = models.UnmarshalJSON(tasksJSON, &pc.TasksCreated)
	_ = models.UnmarshalJSON(failJSON, &pc.StageFailures)
	_ = models.UnmarshalJSON(warnJSON, &pc.DriftWarnings)

	return &pc, nil
}

func ListPlanningCycles(db *sql.DB, projectID string, limit int) ([]models.PlanningCycle, error) {
	query := `SELECT id, project_id, cycle_number, planning_mode, started_at, completed_at,
	                  recon_output, continuity_output, gap_output, prioritization_output,
	                  synthesis_output, anti_drift_output, tasks_created,
	                  coherence_score, stage_failures, drift_warnings
	           FROM planning_cycles WHERE project_id = ?
	           ORDER BY cycle_number DESC`
	args := []interface{}{projectID}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list planning cycles: %w", err)
	}
	defer rows.Close()

	var cycles []models.PlanningCycle
	for rows.Next() {
		var pc models.PlanningCycle
		var completedAt sql.NullString
		var reconJSON, contJSON, gapJSON, prioJSON, synthJSON, driftJSON string
		var tasksJSON, failJSON, warnJSON string

		if err := rows.Scan(
			&pc.ID, &pc.ProjectID, &pc.CycleNumber, &pc.PlanningMode,
			&pc.StartedAt, &completedAt,
			&reconJSON, &contJSON, &gapJSON, &prioJSON, &synthJSON, &driftJSON,
			&tasksJSON, &pc.CoherenceScore, &failJSON, &warnJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan planning cycle: %w", err)
		}

		pc.CompletedAt = models.PtrString(completedAt)
		_ = models.UnmarshalJSON(reconJSON, &pc.ReconOutput)
		_ = models.UnmarshalJSON(contJSON, &pc.ContinuityOutput)
		_ = models.UnmarshalJSON(gapJSON, &pc.GapOutput)
		_ = models.UnmarshalJSON(prioJSON, &pc.PrioritizationOutput)
		_ = models.UnmarshalJSON(synthJSON, &pc.SynthesisOutput)
		_ = models.UnmarshalJSON(driftJSON, &pc.AntiDriftOutput)
		_ = models.UnmarshalJSON(tasksJSON, &pc.TasksCreated)
		_ = models.UnmarshalJSON(failJSON, &pc.StageFailures)
		_ = models.UnmarshalJSON(warnJSON, &pc.DriftWarnings)

		cycles = append(cycles, pc)
	}
	return cycles, rows.Err()
}

// ---- Planner Decisions ----

func CreatePlannerDecision(db *sql.DB, projectID, cycleID, stage, decisionType, subject, reasoning string) (*models.PlannerDecision, error) {
	d := &models.PlannerDecision{
		ID:           "dec_" + uuid.New().String(),
		ProjectID:    projectID,
		CycleID:      cycleID,
		Stage:        stage,
		DecisionType: decisionType,
		Subject:      subject,
		Reasoning:    reasoning,
		CreatedAt:    models.NowISO(),
	}

	_, err := db.Exec(`
		INSERT INTO planner_decisions (id, project_id, cycle_id, stage, decision_type, subject, reasoning, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.ProjectID, d.CycleID, d.Stage, d.DecisionType, d.Subject, d.Reasoning, d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner decision: %w", err)
	}
	return d, nil
}

func ListPlannerDecisions(db *sql.DB, projectID string, cycleID string, limit int) ([]models.PlannerDecision, error) {
	query := `SELECT id, project_id, cycle_id, stage, decision_type, subject, reasoning, created_at
	           FROM planner_decisions WHERE project_id = ?`
	args := []interface{}{projectID}

	if cycleID != "" {
		query += " AND cycle_id = ?"
		args = append(args, cycleID)
	}
	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list planner decisions: %w", err)
	}
	defer rows.Close()

	var decisions []models.PlannerDecision
	for rows.Next() {
		var d models.PlannerDecision
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.CycleID, &d.Stage, &d.DecisionType, &d.Subject, &d.Reasoning, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan planner decision: %w", err)
		}
		decisions = append(decisions, d)
	}
	return decisions, rows.Err()
}

// ---- Seeding (Task 5) ----

// SeedPlanningState initializes the planning state for a project from existing context.
// It creates the planning state if it doesn't exist and derives initial workstreams
// from active tasks in the project.
func SeedPlanningState(db *sql.DB, projectID string) (*models.PlanningState, error) {
	ps, err := GetPlanningState(db, projectID)
	if err != nil {
		return nil, err
	}
	if ps != nil {
		return ps, nil // already seeded
	}

	ps, err = CreatePlanningState(db, projectID)
	if err != nil {
		return nil, err
	}

	// Extract initial workstreams from existing tasks
	if err := extractWorkstreamsFromTasks(db, projectID, ps.ID); err != nil {
		// Non-fatal: planning state still usable without workstreams
		_ = err
	}

	return ps, nil
}

// ---- Workstream Extraction (Task 6) ----

// extractWorkstreamsFromTasks creates initial workstreams by grouping existing active tasks.
func extractWorkstreamsFromTasks(db *sql.DB, projectID, planningStateID string) error {
	// Find active and queued tasks grouped by parent_task_id
	rows, err := db.Query(`
		SELECT id, COALESCE(title, instructions), status_v2, parent_task_id
		FROM tasks
		WHERE project_id = ?
		  AND status_v2 IN ('QUEUED', 'CLAIMED', 'RUNNING', 'NEEDS_REVIEW')
		ORDER BY priority DESC, id ASC
	`, projectID)
	if err != nil {
		return fmt.Errorf("failed to query tasks for workstream extraction: %w", err)
	}
	defer rows.Close()

	type taskInfo struct {
		ID       int
		Title    string
		Status   string
		ParentID *int
	}

	var tasks []taskInfo
	for rows.Next() {
		var t taskInfo
		var parentID sql.NullInt64
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &parentID); err != nil {
			return fmt.Errorf("failed to scan task: %w", err)
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			t.ParentID = &pid
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(tasks) == 0 {
		return nil
	}

	// Group tasks by root parent to form workstreams
	rootTasks := make(map[int][]taskInfo) // rootID -> child tasks
	for _, t := range tasks {
		rootID := t.ID
		if t.ParentID != nil {
			rootID = *t.ParentID
		}
		rootTasks[rootID] = append(rootTasks[rootID], t)
	}

	for rootID, group := range rootTasks {
		// Use the root task title as workstream title, or first task's title
		wsTitle := group[0].Title
		for _, t := range group {
			if t.ID == rootID {
				wsTitle = t.Title
				break
			}
		}

		taskIDs := make([]int, 0, len(group))
		for _, t := range group {
			taskIDs = append(taskIDs, t.ID)
		}

		ws, err := CreateWorkstream(db, projectID, planningStateID, wsTitle,
			fmt.Sprintf("Auto-extracted from %d active task(s)", len(group)),
			"Continuing existing work", "Continue active tasks")
		if err != nil {
			continue // skip this group on error
		}

		// Update with actual task IDs
		ws.RelatedTaskIDs = taskIDs
		if err := UpdateWorkstream(db, ws); err != nil {
			continue
		}
	}

	return nil
}
