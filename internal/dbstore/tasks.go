package dbstore

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/onsomlem/cocopilot/internal/models"
)

func MapTaskStatusV1ToV2(status models.TaskStatus) models.TaskStatusV2 {
	switch status {
	case models.StatusNotPicked:
		return models.TaskStatusQueued
	case models.StatusInProgress:
		return models.TaskStatusRunning
	case models.StatusComplete:
		return models.TaskStatusSucceeded
	default:
		return models.TaskStatusQueued
	}
}

func MapTaskStatusV2ToV1(status models.TaskStatusV2) models.TaskStatus {
	switch status {
	case models.TaskStatusQueued:
		return models.StatusNotPicked
	case models.TaskStatusClaimed, models.TaskStatusRunning:
		return models.StatusInProgress
	case models.TaskStatusSucceeded, models.TaskStatusFailed, models.TaskStatusNeedsReview, models.TaskStatusCancelled:
		return models.StatusComplete
	default:
		return models.StatusNotPicked
	}
}

func GetTaskV2(db *sql.DB, taskID int) (*models.TaskV2, error) {
	var task models.TaskV2
	var title sql.NullString
	var taskType sql.NullString
	var tagsJSON sql.NullString
	var statusV1 string
	var statusV2 sql.NullString
	var parentID sql.NullInt64
	var output sql.NullString
	var updatedAt sql.NullString
	var requiresApproval int
	var approvalStatus sql.NullString

	err := db.QueryRow(`
		SELECT id, project_id, title, instructions, type, priority, tags_json,
		       status, status_v2, parent_task_id, output, created_at, updated_at,
		       COALESCE(automation_depth, 0),
		       COALESCE(requires_approval, 0), approval_status
		FROM tasks WHERE id = ?
	`, taskID).Scan(
		&task.ID,
		&task.ProjectID,
		&title,
		&task.Instructions,
		&taskType,
		&task.Priority,
		&tagsJSON,
		&statusV1,
		&statusV2,
		&parentID,
		&output,
		&task.CreatedAt,
		&updatedAt,
		&task.AutomationDepth,
		&requiresApproval,
		&approvalStatus,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %d", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	task.StatusV1 = models.TaskStatus(statusV1)
	if statusV2.Valid && strings.TrimSpace(statusV2.String) != "" {
		task.StatusV2 = models.TaskStatusV2(statusV2.String)
	} else {
		task.StatusV2 = MapTaskStatusV1ToV2(task.StatusV1)
	}

	task.RequiresApproval = requiresApproval != 0
	if approvalStatus.Valid {
		task.ApprovalStatus = &approvalStatus.String
	}

	if title.Valid {
		task.Title = &title.String
	}
	if taskType.Valid && strings.TrimSpace(taskType.String) != "" {
		task.Type = models.TaskType(taskType.String)
	} else {
		task.Type = models.TaskTypeModify
	}
	if tagsJSON.Valid && strings.TrimSpace(tagsJSON.String) != "" {
		var tags []string
		if err := models.UnmarshalJSON(tagsJSON.String, &tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
		task.Tags = tags
	}
	if parentID.Valid {
		pid := int(parentID.Int64)
		task.ParentTaskID = &pid
	}
	if output.Valid {
		outputStr := strings.ReplaceAll(output.String, "\\n", "\n")
		task.Output = &outputStr
	}
	if updatedAt.Valid && strings.TrimSpace(updatedAt.String) != "" {
		task.UpdatedAt = &updatedAt.String
	} else {
		fallback := task.CreatedAt
		task.UpdatedAt = &fallback
	}

	return &task, nil
}

func SetTaskAutomationDepth(db *sql.DB, taskID int, depth int) error {
	_, err := db.Exec(`UPDATE tasks SET automation_depth = ? WHERE id = ?`, depth, taskID)
	if err != nil {
		return fmt.Errorf("failed to set automation depth for task %d: %w", taskID, err)
	}
	return nil
}

func CreateTaskV2(db *sql.DB, instructions string, projectID string, parentTaskID *int) (*models.TaskV2, error) {
	createdAt := models.NowISO()
	updatedAt := createdAt
	var result sql.Result
	var err error

	if parentTaskID != nil {
		result, err = db.Exec(
			`INSERT INTO tasks (instructions, status, status_v2, parent_task_id, project_id, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			instructions,
			models.StatusNotPicked,
			models.TaskStatusQueued,
			*parentTaskID,
			projectID,
			createdAt,
			updatedAt,
		)
	} else {
		result, err = db.Exec(
			`INSERT INTO tasks (instructions, status, status_v2, project_id, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			instructions,
			models.StatusNotPicked,
			models.TaskStatusQueued,
			projectID,
			createdAt,
			updatedAt,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to read task id: %w", err)
	}

	return GetTaskV2(db, int(taskID))
}

func CreateTaskV2WithMeta(db *sql.DB, instructions string, projectID string, parentTaskID *int, title *string, taskType *models.TaskType, priority *int, tags []string) (*models.TaskV2, error) {
	createdAt := models.NowISO()
	updatedAt := createdAt

	trimmedTitle := ""
	if title != nil {
		trimmedTitle = strings.TrimSpace(*title)
	}
	var titleNull sql.NullString
	if trimmedTitle != "" {
		titleNull = sql.NullString{String: trimmedTitle, Valid: true}
	}

	var typeNull sql.NullString
	if taskType != nil {
		typeNull = sql.NullString{String: string(*taskType), Valid: true}
	}

	resolvedPriority := 50
	if priority != nil {
		resolvedPriority = *priority
	}

	var tagsNull sql.NullString
	if len(tags) > 0 {
		tagsJSON, err := models.MarshalJSON(tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		tagsNull = sql.NullString{String: tagsJSON, Valid: true}
	}

	var result sql.Result
	var err error
	if parentTaskID != nil {
		result, err = db.Exec(
			`INSERT INTO tasks (instructions, status, status_v2, parent_task_id, project_id, created_at, updated_at, title, type, priority, tags_json)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			instructions,
			models.StatusNotPicked,
			models.TaskStatusQueued,
			*parentTaskID,
			projectID,
			createdAt,
			updatedAt,
			titleNull,
			typeNull,
			resolvedPriority,
			tagsNull,
		)
	} else {
		result, err = db.Exec(
			`INSERT INTO tasks (instructions, status, status_v2, project_id, created_at, updated_at, title, type, priority, tags_json)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			instructions,
			models.StatusNotPicked,
			models.TaskStatusQueued,
			projectID,
			createdAt,
			updatedAt,
			titleNull,
			typeNull,
			resolvedPriority,
			tagsNull,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to read task id: %w", err)
	}

	return GetTaskV2(db, int(taskID))
}

func UpdateTaskV2(db *sql.DB, taskID int, instructions *string, statusV1 *models.TaskStatus, statusV2 *models.TaskStatusV2, projectID *string, parentTaskID *int) (*models.TaskV2, error) {
	task, err := GetTaskV2(db, taskID)
	if err != nil {
		return nil, err
	}

	if instructions != nil {
		task.Instructions = *instructions
	}
	if statusV1 != nil {
		task.StatusV1 = *statusV1
	}
	if statusV2 != nil {
		task.StatusV2 = *statusV2
	}
	if projectID != nil {
		task.ProjectID = *projectID
	}
	if parentTaskID != nil {
		task.ParentTaskID = parentTaskID
	}

	updatedAt := models.NowISO()
	task.UpdatedAt = &updatedAt

	parentNull := sql.NullInt64{Valid: false}
	if task.ParentTaskID != nil {
		parentNull = sql.NullInt64{Int64: int64(*task.ParentTaskID), Valid: true}
	}

	_, err = db.Exec(
		`UPDATE tasks
		 SET instructions = ?, status = ?, status_v2 = ?, project_id = ?, parent_task_id = ?, updated_at = ?
		 WHERE id = ?`,
		task.Instructions,
		task.StatusV1,
		task.StatusV2,
		task.ProjectID,
		parentNull,
		updatedAt,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return GetTaskV2(db, taskID)
}

func GetTaskParentChain(db *sql.DB, parentID *int) ([]models.TaskV2, error) {
	if parentID == nil {
		return nil, nil
	}

	chain := make([]models.TaskV2, 0)
	visited := make(map[int]struct{})
	current := *parentID

	for current != 0 {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}

		parentTask, err := GetTaskV2(db, current)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				break
			}
			return nil, err
		}
		chain = append(chain, *parentTask)

		if parentTask.ParentTaskID == nil {
			break
		}
		current = *parentTask.ParentTaskID
	}

	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain, nil
}

func GetLatestRunByTaskID(db *sql.DB, taskID int) (*models.Run, error) {
	var run models.Run
	var finishedAt, errorMsg sql.NullString

	err := db.QueryRow(`
		SELECT id, task_id, agent_id, status, started_at, finished_at, error
		FROM runs WHERE task_id = ? ORDER BY started_at DESC LIMIT 1
	`, taskID).Scan(&run.ID, &run.TaskID, &run.AgentID, &run.Status, &run.StartedAt, &finishedAt, &errorMsg)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest run: %w", err)
	}

	run.FinishedAt = models.PtrString(finishedAt)
	run.Error = models.PtrString(errorMsg)
	return &run, nil
}

func ListTasksV2(db *sql.DB, projectID string, statusColumn string, statusValue string, typeFilter string, tagFilter string, queryFilter string, limit int, offset int, sortField string, sortDirection string) ([]models.TaskV2, int, error) {
	if statusValue != "" && statusColumn != "status" && statusColumn != "status_v2" {
		return nil, 0, fmt.Errorf("invalid status column: %s", statusColumn)
	}

	direction := strings.ToLower(strings.TrimSpace(sortDirection))
	if direction == "" {
		direction = "asc"
	}
	if direction != "asc" && direction != "desc" {
		return nil, 0, fmt.Errorf("invalid sort direction: %s", sortDirection)
	}

	filters := make([]string, 0, 2)
	args := make([]interface{}, 0, 4)
	if strings.TrimSpace(projectID) != "" {
		filters = append(filters, "project_id = ?")
		args = append(args, projectID)
	}
	if statusValue != "" {
		filters = append(filters, statusColumn+" = ?")
		args = append(args, statusValue)
	}
	if strings.TrimSpace(typeFilter) != "" {
		filters = append(filters, "COALESCE(type, 'MODIFY') = ?")
		args = append(args, strings.TrimSpace(typeFilter))
	}
	if strings.TrimSpace(tagFilter) != "" {
		filters = append(filters, "LOWER(tags_json) LIKE ?")
		args = append(args, "%\""+strings.ToLower(tagFilter)+"\"%")
	}
	if strings.TrimSpace(queryFilter) != "" {
		filters = append(filters, "(LOWER(instructions) LIKE ? OR LOWER(COALESCE(title, '')) LIKE ?)")
		pattern := "%" + strings.ToLower(queryFilter) + "%"
		args = append(args, pattern, pattern)
	}

	countQuery := strings.Builder{}
	countQuery.WriteString("SELECT COUNT(1) FROM tasks")
	if len(filters) > 0 {
		countQuery.WriteString(" WHERE ")
		countQuery.WriteString(strings.Join(filters, " AND "))
	}

	var total int
	if err := db.QueryRow(countQuery.String(), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	query := strings.Builder{}
	query.WriteString(`
		SELECT id, project_id, title, instructions, type, priority, tags_json,
		       status, status_v2, parent_task_id, output, created_at, updated_at
		FROM tasks`)
	if len(filters) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(filters, " AND "))
	}

	orderBy := "created_at " + strings.ToUpper(direction)
	switch strings.ToLower(strings.TrimSpace(sortField)) {
	case "", "created_at":
		orderBy = "created_at " + strings.ToUpper(direction)
	case "updated_at":
		orderBy = "COALESCE(updated_at, created_at) " + strings.ToUpper(direction)
	default:
		return nil, 0, fmt.Errorf("invalid sort field: %s", sortField)
	}

	query.WriteString(" ORDER BY ")
	query.WriteString(orderBy)
	query.WriteString(" LIMIT ? OFFSET ?")

	argsWithPaging := append(args, limit, offset)
	rows, err := db.Query(query.String(), argsWithPaging...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]models.TaskV2, 0)
	for rows.Next() {
		var task models.TaskV2
		var title sql.NullString
		var taskType sql.NullString
		var tagsJSON sql.NullString
		var statusV1 string
		var statusV2 sql.NullString
		var parentID sql.NullInt64
		var output sql.NullString
		var updatedAt sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&title,
			&task.Instructions,
			&taskType,
			&task.Priority,
			&tagsJSON,
			&statusV1,
			&statusV2,
			&parentID,
			&output,
			&task.CreatedAt,
			&updatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}

		task.StatusV1 = models.TaskStatus(statusV1)
		if statusV2.Valid && strings.TrimSpace(statusV2.String) != "" {
			task.StatusV2 = models.TaskStatusV2(statusV2.String)
		} else {
			task.StatusV2 = MapTaskStatusV1ToV2(task.StatusV1)
		}

		if title.Valid {
			task.Title = &title.String
		}
		if taskType.Valid && strings.TrimSpace(taskType.String) != "" {
			task.Type = models.TaskType(taskType.String)
		} else {
			task.Type = models.TaskTypeModify
		}
		if tagsJSON.Valid && strings.TrimSpace(tagsJSON.String) != "" {
			var tags []string
			if err := models.UnmarshalJSON(tagsJSON.String, &tags); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
			task.Tags = tags
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			task.ParentTaskID = &pid
		}
		if output.Valid {
			outputStr := strings.ReplaceAll(output.String, "\\n", "\n")
			task.Output = &outputStr
		}
		if updatedAt.Valid && strings.TrimSpace(updatedAt.String) != "" {
			task.UpdatedAt = &updatedAt.String
		} else {
			fallback := task.CreatedAt
			task.UpdatedAt = &fallback
		}

		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

func TaskExists(db *sql.DB, taskID int) (bool, error) {
	var exists int
	if err := db.QueryRow("SELECT COUNT(1) FROM tasks WHERE id = ?", taskID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check task existence: %w", err)
	}
	return exists > 0, nil
}

func CreateTaskDependency(db *sql.DB, taskID int, dependsOnTaskID int) (*models.TaskDependency, error) {
	_, err := db.Exec(
		"INSERT INTO task_dependencies (task_id, depends_on_task_id) VALUES (?, ?)",
		taskID,
		dependsOnTaskID,
	)
	if err != nil {
		if IsSQLiteConstraintError(err) {
			return nil, ErrTaskDependencyExists
		}
		return nil, fmt.Errorf("failed to create task dependency: %w", err)
	}

	if err := emitTaskDependencyEvent(db, "task.dependency.created", taskID, dependsOnTaskID); err != nil {
		return nil, err
	}

	depTask, depErr := GetTaskV2(db, dependsOnTaskID)
	if depErr == nil && depTask != nil && !depTask.StatusV2.IsTerminal() {
		task, taskErr := GetTaskV2(db, taskID)
		projectID := ""
		if taskErr == nil && task != nil {
			projectID = task.ProjectID
		}
		CreateEvent(db, projectID, "task.blocked", "task", fmt.Sprintf("%d", taskID), map[string]interface{}{
			"task_id":            taskID,
			"blocked_by_task_id": dependsOnTaskID,
		})
	}

	return &models.TaskDependency{TaskID: taskID, DependsOnTaskID: dependsOnTaskID}, nil
}

func TaskDependencyCreatesCycle(db *sql.DB, taskID int, dependsOnTaskID int) (bool, error) {
	if taskID == dependsOnTaskID {
		return true, nil
	}

	var found int
	err := db.QueryRow(`
		WITH RECURSIVE deps(depends_on_task_id) AS (
			SELECT depends_on_task_id
			FROM task_dependencies
			WHERE task_id = ?
			UNION
			SELECT td.depends_on_task_id
			FROM task_dependencies td
			JOIN deps d ON td.task_id = d.depends_on_task_id
		)
		SELECT 1 FROM deps WHERE depends_on_task_id = ? LIMIT 1
	`, dependsOnTaskID, taskID).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check dependency cycle: %w", err)
	}
	return true, nil
}

func ListTaskDependencies(db *sql.DB, taskID int) ([]models.TaskDependency, error) {
	rows, err := db.Query(
		"SELECT task_id, depends_on_task_id FROM task_dependencies WHERE task_id = ? ORDER BY depends_on_task_id ASC",
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query task dependencies: %w", err)
	}
	defer rows.Close()

	deps := make([]models.TaskDependency, 0)
	for rows.Next() {
		var dep models.TaskDependency
		if err := rows.Scan(&dep.TaskID, &dep.DependsOnTaskID); err != nil {
			return nil, fmt.Errorf("failed to scan task dependency: %w", err)
		}
		deps = append(deps, dep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating task dependencies: %w", err)
	}

	return deps, nil
}

func DeleteTaskDependency(db *sql.DB, taskID int, dependsOnTaskID int) (bool, error) {
	result, err := db.Exec(
		"DELETE FROM task_dependencies WHERE task_id = ? AND depends_on_task_id = ?",
		taskID,
		dependsOnTaskID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete task dependency: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read task dependency delete result: %w", err)
	}
	if rows > 0 {
		if err := emitTaskDependencyEvent(db, "task.dependency.deleted", taskID, dependsOnTaskID); err != nil {
			return false, err
		}
	}
	return rows > 0, nil
}

func GetTasksDependingOn(db *sql.DB, completedTaskID int) ([]models.TaskV2, error) {
	rows, err := db.Query(`
		SELECT t.id, t.project_id, t.title, t.instructions, t.type, t.priority, t.tags_json,
		       t.status, t.status_v2, t.parent_task_id, t.output, t.created_at, t.updated_at,
		       COALESCE(t.automation_depth, 0)
		FROM tasks t
		INNER JOIN task_dependencies td ON td.task_id = t.id
		WHERE td.depends_on_task_id = ? AND t.deleted_at IS NULL
	`, completedTaskID)
	if err != nil {
		return nil, fmt.Errorf("GetTasksDependingOn: query: %w", err)
	}
	defer rows.Close()

	var tasks []models.TaskV2
	for rows.Next() {
		var task models.TaskV2
		var title, taskType, tagsJSON, statusV2, output, updatedAt sql.NullString
		var parentID sql.NullInt64
		var statusV1 string
		if err := rows.Scan(
			&task.ID, &task.ProjectID, &title, &task.Instructions, &taskType,
			&task.Priority, &tagsJSON, &statusV1, &statusV2, &parentID,
			&output, &task.CreatedAt, &updatedAt, &task.AutomationDepth,
		); err != nil {
			return nil, fmt.Errorf("GetTasksDependingOn: scan: %w", err)
		}
		task.StatusV1 = models.TaskStatus(statusV1)
		if statusV2.Valid && strings.TrimSpace(statusV2.String) != "" {
			task.StatusV2 = models.TaskStatusV2(statusV2.String)
		} else {
			task.StatusV2 = MapTaskStatusV1ToV2(task.StatusV1)
		}
		if title.Valid {
			task.Title = &title.String
		}
		if taskType.Valid && strings.TrimSpace(taskType.String) != "" {
			task.Type = models.TaskType(taskType.String)
		} else {
			task.Type = models.TaskTypeModify
		}
		if tagsJSON.Valid && strings.TrimSpace(tagsJSON.String) != "" {
			var tags []string
			if err := models.UnmarshalJSON(tagsJSON.String, &tags); err == nil {
				task.Tags = tags
			}
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			task.ParentTaskID = &pid
		}
		if output.Valid {
			s := strings.ReplaceAll(output.String, "\\n", "\n")
			task.Output = &s
		}
		if updatedAt.Valid && strings.TrimSpace(updatedAt.String) != "" {
			task.UpdatedAt = &updatedAt.String
		} else {
			fallback := task.CreatedAt
			task.UpdatedAt = &fallback
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func AreAllDependenciesFulfilled(db *sql.DB, taskID int) (bool, error) {
	var unfulfilled int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM task_dependencies td
		INNER JOIN tasks t ON t.id = td.depends_on_task_id
		WHERE td.task_id = ? AND (t.status_v2 IS NULL OR t.status_v2 != 'SUCCEEDED')
	`, taskID).Scan(&unfulfilled)
	if err != nil {
		return false, fmt.Errorf("AreAllDependenciesFulfilled: %w", err)
	}
	return unfulfilled == 0, nil
}

func SetTaskApprovalStatus(db *sql.DB, taskID int, status string) (*models.TaskV2, error) {
	now := models.NowISO()
	_, err := db.Exec(`UPDATE tasks SET approval_status = ?, updated_at = ? WHERE id = ?`, status, now, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to set approval status: %w", err)
	}
	return GetTaskV2(db, taskID)
}

func emitTaskDependencyEvent(db *sql.DB, kind string, taskID int, dependsOnTaskID int) error {
	projectID, err := resolveTaskProjectID(db, taskID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"task_id":       taskID,
		"depends_on_id": dependsOnTaskID,
	}
	entityID := fmt.Sprintf("%d:%d", taskID, dependsOnTaskID)

	if _, err := CreateEvent(db, projectID, kind, "task_dependency", entityID, payload); err != nil {
		return fmt.Errorf("failed to emit %s event: %w", kind, err)
	}
	return nil
}
