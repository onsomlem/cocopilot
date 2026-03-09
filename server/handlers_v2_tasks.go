package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// API v2 Task Handlers
// ============================================================================

type v2RunSummary struct {
	ID         string    `json:"id"`
	Status     RunStatus `json:"status"`
	StartedAt  string    `json:"started_at"`
	FinishedAt *string   `json:"finished_at,omitempty"`
}

var v2TaskListStatusV1 = map[string]struct{}{
	string(StatusNotPicked):  {},
	string(StatusInProgress): {},
	string(StatusComplete):   {},
}

var v2TaskListStatusV2 = map[string]struct{}{
	string(TaskStatusQueued):      {},
	string(TaskStatusClaimed):     {},
	string(TaskStatusRunning):     {},
	string(TaskStatusSucceeded):   {},
	string(TaskStatusFailed):      {},
	string(TaskStatusNeedsReview): {},
	string(TaskStatusCancelled):   {},
}

var v2TaskListTypeFilter = map[string]struct{}{
	string(TaskTypeAnalyze):  {},
	string(TaskTypeModify):   {},
	string(TaskTypeTest):     {},
	string(TaskTypeReview):   {},
	string(TaskTypeDoc):      {},
	string(TaskTypeRelease):  {},
	string(TaskTypeRollback): {},
}

const (
	v2TaskListDefaultLimit   = 100
	v2TaskListMaxLimit       = 500
	v2AgentListDefaultLimit  = 100
	v2AgentListMaxLimit      = 500
	v2PolicyListDefaultLimit = 100
	v2PolicyListMaxLimit     = 500
)

func resolveV2PolicyListSort(raw string) (string, string, error) {
	sort := strings.ToLower(strings.TrimSpace(raw))
	if sort == "" {
		return "created_at", "asc", nil
	}

	switch sort {
	case "created_at:asc":
		return "created_at", "asc", nil
	case "created_at:desc":
		return "created_at", "desc", nil
	case "name:asc":
		return "name", "asc", nil
	case "name:desc":
		return "name", "desc", nil
	default:
		return "", "", fmt.Errorf("invalid sort option")
	}
}

func resolveV2TaskListStatusFilter(raw string) (string, string, error) {
	status := strings.TrimSpace(raw)
	if status == "" {
		return "", "", nil
	}

	status = strings.ToUpper(status)
	if _, ok := v2TaskListStatusV2[status]; ok {
		return "status_v2", status, nil
	}
	if _, ok := v2TaskListStatusV1[status]; ok {
		return "status", status, nil
	}

	return "", "", fmt.Errorf("invalid status filter")
}

func resolveV2TaskListSort(raw string) (string, string, error) {
	sort := strings.ToLower(strings.TrimSpace(raw))
	if sort == "" {
		return "created_at", "asc", nil
	}

	switch sort {
	case "created_at:asc":
		return "created_at", "asc", nil
	case "created_at:desc":
		return "created_at", "desc", nil
	case "updated_at:asc":
		return "updated_at", "asc", nil
	case "updated_at:desc":
		return "updated_at", "desc", nil
	default:
		return "", "", fmt.Errorf("invalid sort option")
	}
}

func resolveV2TaskListTypeFilter(query url.Values) (string, error) {
	values, ok := query["type"]
	if !ok {
		return "", nil
	}
	value := ""
	if len(values) > 0 {
		value = strings.TrimSpace(values[0])
	}
	if value == "" {
		return "", fmt.Errorf("type cannot be empty")
	}
	value = strings.ToUpper(value)
	if _, ok := v2TaskListTypeFilter[value]; !ok {
		return "", fmt.Errorf("invalid type filter")
	}
	return value, nil
}

func resolveV2TaskListTagFilter(query url.Values) (string, error) {
	values, ok := query["tag"]
	if !ok {
		return "", nil
	}
	value := ""
	if len(values) > 0 {
		value = strings.TrimSpace(values[0])
	}
	if value == "" {
		return "", fmt.Errorf("tag cannot be empty")
	}
	if strings.Contains(value, "\"") {
		return "", fmt.Errorf("tag contains invalid characters")
	}
	return value, nil
}

func resolveV2TaskListQueryFilter(query url.Values) (string, error) {
	values, ok := query["q"]
	if !ok {
		return "", nil
	}
	value := ""
	if len(values) > 0 {
		value = strings.TrimSpace(values[0])
	}
	if value == "" {
		return "", fmt.Errorf("q cannot be empty")
	}
	return value, nil
}

func v2ListTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	query := r.URL.Query()
	projectID := strings.TrimSpace(query.Get("project_id"))
	statusColumn, statusValue, err := resolveV2TaskListStatusFilter(query.Get("status"))
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid status filter", map[string]interface{}{
			"status": query.Get("status"),
		})
		return
	}

	typeFilter, err := resolveV2TaskListTypeFilter(query)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), map[string]interface{}{
			"type": query.Get("type"),
		})
		return
	}

	tagFilter, err := resolveV2TaskListTagFilter(query)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), map[string]interface{}{
			"tag": query.Get("tag"),
		})
		return
	}

	queryFilter, err := resolveV2TaskListQueryFilter(query)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), map[string]interface{}{
			"q": query.Get("q"),
		})
		return
	}

	sortField, sortDirection, err := resolveV2TaskListSort(query.Get("sort"))
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "sort must be created_at:asc, created_at:desc, updated_at:asc, or updated_at:desc", map[string]interface{}{
			"sort": query.Get("sort"),
		})
		return
	}

	limit := v2TaskListDefaultLimit
	offset := 0
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
				"limit": rawLimit,
			})
			return
		}
		if parsed > v2TaskListMaxLimit {
			limit = v2TaskListMaxLimit
		} else {
			limit = parsed
		}
	}
	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil || parsed < 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "offset must be a non-negative integer", map[string]interface{}{
				"offset": rawOffset,
			})
			return
		}
		offset = parsed
	}

	tasks, total, err := ListTasksV2(db, projectID, statusColumn, statusValue, typeFilter, tagFilter, queryFilter, limit, offset, sortField, sortDirection)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"tasks": tasks,
		"total": total,
	})
}

func v2CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		Instructions string `json:"instructions"`
		ParentTaskID *int   `json:"parent_task_id"`
		ProjectID    string `json:"project_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	instructions := strings.TrimSpace(req.Instructions)
	if instructions == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "instructions is required", map[string]interface{}{
			"instructions": req.Instructions,
		})
		return
	}

	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		projectID = DefaultProjectID
	}

	if req.ParentTaskID != nil && *req.ParentTaskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id must be a positive integer", map[string]interface{}{
			"parent_task_id": *req.ParentTaskID,
		})
		return
	}

	project, err := GetProject(db, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id not found", map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	if req.ParentTaskID != nil {
		parentTask, err := GetTaskV2(db, *req.ParentTaskID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id not found", map[string]interface{}{
					"parent_task_id": *req.ParentTaskID,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
			})
			return
		}
		if parentTask.ProjectID != project.ID {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent task must be in the same project", map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
				"project_id":     project.ID,
			})
			return
		}
	}

	blocked, reason, err := isTaskCreateBlockedByPolicies(db, project.ID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": project.ID,
		})
		return
	}
	if blocked {
		message := "Task creation blocked by policy"
		if reason != "" {
			message = fmt.Sprintf("Task creation blocked by policy: %s", reason)
		}
		writeV2Error(w, http.StatusForbidden, "FORBIDDEN", message, map[string]interface{}{
			"project_id": project.ID,
			"reason":     reason,
		})
		return
	}

	task, err := CreateTaskV2(db, instructions, project.ID, req.ParentTaskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": project.ID,
		})
		return
	}

	go broadcastUpdate(v1EventTypeTasks)

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"task": task,
	})
}

func v2GetTaskDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Extract task ID from URL path (/api/v2/tasks/:id)
	rawID := strings.TrimPrefix(r.URL.Path, "/api/v2/tasks/")
	if rawID == "" || strings.Contains(rawID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawID,
		})
		return
	}

	taskID, err := strconv.Atoi(rawID)
	if err != nil || taskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawID,
		})
		return
	}

	task, err := GetTaskV2(db, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	parentChain, err := GetTaskParentChain(db, task.ParentTaskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	latestRun, err := GetLatestRunByTaskID(db, taskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	response := map[string]interface{}{
		"task": task,
	}
	if len(parentChain) > 0 {
		response["parent_chain"] = parentChain
	}
	if latestRun != nil {
		response["latest_run"] = v2RunSummary{
			ID:         latestRun.ID,
			Status:     latestRun.Status,
			StartedAt:  latestRun.StartedAt,
			FinishedAt: latestRun.FinishedAt,
		}
	}

	writeV2JSON(w, http.StatusOK, response)
}

func v2TaskDependenciesHandler(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
		return
	}

	taskID, err := strconv.Atoi(rawID)
	if err != nil || taskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawID,
		})
		return
	}

	if r.Method == http.MethodGet {
		exists, err := TaskExists(db, taskID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		if !exists {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("task not found: %d", taskID), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}

		deps, err := ListTaskDependencies(db, taskID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"dependencies": deps,
		})
		return
	}

	var req struct {
		DependsOnTaskID int `json:"depends_on_task_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if req.DependsOnTaskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "depends_on_task_id must be a positive integer", map[string]interface{}{
			"depends_on_task_id": req.DependsOnTaskID,
		})
		return
	}
	if req.DependsOnTaskID == taskID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "task cannot depend on itself", map[string]interface{}{
			"task_id":            taskID,
			"depends_on_task_id": req.DependsOnTaskID,
		})
		return
	}

	exists, err := TaskExists(db, taskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}
	if !exists {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("task not found: %d", taskID), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	dependsExists, err := TaskExists(db, req.DependsOnTaskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"depends_on_task_id": req.DependsOnTaskID,
		})
		return
	}
	if !dependsExists {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("task not found: %d", req.DependsOnTaskID), map[string]interface{}{
			"depends_on_task_id": req.DependsOnTaskID,
		})
		return
	}

	createsCycle, err := TaskDependencyCreatesCycle(db, taskID, req.DependsOnTaskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id":            taskID,
			"depends_on_task_id": req.DependsOnTaskID,
		})
		return
	}
	if createsCycle {
		writeV2Error(w, http.StatusConflict, "CONFLICT", "Task dependency would create a cycle", map[string]interface{}{
			"task_id":            taskID,
			"depends_on_task_id": req.DependsOnTaskID,
		})
		return
	}

	dep, err := CreateTaskDependency(db, taskID, req.DependsOnTaskID)
	if err != nil {
		if errors.Is(err, ErrTaskDependencyExists) {
			writeV2Error(w, http.StatusConflict, "CONFLICT", "Task dependency already exists", map[string]interface{}{
				"task_id":            taskID,
				"depends_on_task_id": req.DependsOnTaskID,
			})
			return
		}
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry dependency creation", map[string]interface{}{
				"task_id":            taskID,
				"depends_on_task_id": req.DependsOnTaskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id":            taskID,
			"depends_on_task_id": req.DependsOnTaskID,
		})
		return
	}

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"dependency": dep,
	})
}

func v2TaskDependencyDetailHandler(w http.ResponseWriter, r *http.Request, rawTaskID string, rawDependsID string) {
	if r.Method != http.MethodDelete {
		writeV2MethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	taskID, err := strconv.Atoi(rawTaskID)
	if err != nil || taskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawTaskID,
		})
		return
	}

	dependsOnTaskID, err := strconv.Atoi(rawDependsID)
	if err != nil || dependsOnTaskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "depends_on_task_id must be a positive integer", map[string]interface{}{
			"depends_on_task_id": rawDependsID,
		})
		return
	}

	exists, err := TaskExists(db, taskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}
	if !exists {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("task not found: %d", taskID), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	dependsExists, err := TaskExists(db, dependsOnTaskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"depends_on_task_id": dependsOnTaskID,
		})
		return
	}
	if !dependsExists {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("task not found: %d", dependsOnTaskID), map[string]interface{}{
			"depends_on_task_id": dependsOnTaskID,
		})
		return
	}

	deleted, err := DeleteTaskDependency(db, taskID, dependsOnTaskID)
	if err != nil {
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry dependency removal", map[string]interface{}{
				"task_id":            taskID,
				"depends_on_task_id": dependsOnTaskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id":            taskID,
			"depends_on_task_id": dependsOnTaskID,
		})
		return
	}
	if !deleted {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Task dependency not found", map[string]interface{}{
			"task_id":            taskID,
			"depends_on_task_id": dependsOnTaskID,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"dependency": TaskDependency{TaskID: taskID, DependsOnTaskID: dependsOnTaskID},
	})
}

func v2UpdateTaskHandler(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodPatch {
		writeV2MethodNotAllowed(w, r, http.MethodPatch)
		return
	}

	taskID, err := strconv.Atoi(rawID)
	if err != nil || taskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawID,
		})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}
	if strings.TrimSpace(string(bodyBytes)) == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "At least one field is required", nil)
		return
	}

	var req struct {
		Instructions *string `json:"instructions"`
		Status       *string `json:"status"`
		ProjectID    *string `json:"project_id"`
		ParentTaskID *int    `json:"parent_task_id"`
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}
	if req.Instructions == nil && req.Status == nil && req.ProjectID == nil && req.ParentTaskID == nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "At least one field is required", nil)
		return
	}

	var instructions *string
	if req.Instructions != nil {
		trimmed := strings.TrimSpace(*req.Instructions)
		if trimmed == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "instructions cannot be empty", map[string]interface{}{
				"instructions": *req.Instructions,
			})
			return
		}
		instructions = &trimmed
	}

	var statusV1 *TaskStatus
	var statusV2 *TaskStatusV2
	if req.Status != nil {
		statusRaw := strings.TrimSpace(*req.Status)
		if statusRaw == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "status cannot be empty", map[string]interface{}{
				"status": *req.Status,
			})
			return
		}
		switch statusRaw {
		case string(StatusNotPicked), string(StatusInProgress), string(StatusComplete):
			parsedV1 := TaskStatus(statusRaw)
			parsedV2 := mapTaskStatusV1ToV2(parsedV1)
			statusV1 = &parsedV1
			statusV2 = &parsedV2
		case string(TaskStatusQueued), string(TaskStatusClaimed), string(TaskStatusRunning),
			string(TaskStatusSucceeded), string(TaskStatusFailed), string(TaskStatusNeedsReview), string(TaskStatusCancelled):
			parsedV2 := TaskStatusV2(statusRaw)
			parsedV1 := mapTaskStatusV2ToV1(parsedV2)
			statusV2 = &parsedV2
			statusV1 = &parsedV1
		default:
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid status", map[string]interface{}{
				"status": statusRaw,
			})
			return
		}
	}

	task, err := GetTaskV2(db, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	var projectID *string
	if req.ProjectID != nil {
		trimmed := strings.TrimSpace(*req.ProjectID)
		if trimmed == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id cannot be empty", map[string]interface{}{
				"project_id": *req.ProjectID,
			})
			return
		}
		project, err := GetProject(db, trimmed)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id not found", map[string]interface{}{
					"project_id": trimmed,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": trimmed,
			})
			return
		}
		projectID = &project.ID
	}

	targetProjectID := task.ProjectID
	if projectID != nil {
		targetProjectID = *projectID
	}

	if req.ParentTaskID != nil {
		if *req.ParentTaskID <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id must be a positive integer", map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
			})
			return
		}
		if *req.ParentTaskID == taskID {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id cannot reference the task itself", map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
			})
			return
		}
		parentTask, err := GetTaskV2(db, *req.ParentTaskID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id not found", map[string]interface{}{
					"parent_task_id": *req.ParentTaskID,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
			})
			return
		}
		if parentTask.ProjectID != targetProjectID {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent task must be in the same project", map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
				"project_id":     targetProjectID,
			})
			return
		}
	} else if projectID != nil && task.ParentTaskID != nil {
		parentTask, err := GetTaskV2(db, *task.ParentTaskID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id not found", map[string]interface{}{
					"parent_task_id": *task.ParentTaskID,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"parent_task_id": *task.ParentTaskID,
			})
			return
		}
		if parentTask.ProjectID != targetProjectID {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent task must be in the same project", map[string]interface{}{
				"parent_task_id": *task.ParentTaskID,
				"project_id":     targetProjectID,
			})
			return
		}
	}

	blocked, reason, err := isTaskUpdateBlockedByPolicies(db, targetProjectID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id":    taskID,
			"project_id": targetProjectID,
		})
		return
	}
	if blocked {
		message := "Task update blocked by policy"
		if reason != "" {
			message = fmt.Sprintf("Task update blocked by policy: %s", reason)
		}
		writeV2Error(w, http.StatusForbidden, "FORBIDDEN", message, map[string]interface{}{
			"task_id":    taskID,
			"project_id": targetProjectID,
			"reason":     reason,
		})
		return
	}

	updatedTask, err := UpdateTaskV2(db, taskID, instructions, statusV1, statusV2, projectID, req.ParentTaskID)
	if err != nil {
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry update", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	go broadcastUpdate(v1EventTypeTasks)

	CreateEvent(db, updatedTask.ProjectID, "task.updated", "task", fmt.Sprintf("%d", taskID), map[string]interface{}{
		"task_id":   taskID,
		"status_v2": string(updatedTask.StatusV2),
	})

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"task": updatedTask,
	})
}

func v2DeleteTaskHandler(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodDelete {
		writeV2MethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	taskID, err := strconv.Atoi(rawID)
	if err != nil || taskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawID,
		})
		return
	}

	task, err := GetTaskV2(db, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	blocked, reason, err := isTaskDeleteBlockedByPolicies(db, task.ProjectID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id":    taskID,
			"project_id": task.ProjectID,
		})
		return
	}
	if blocked {
		message := "Task delete blocked by policy"
		if reason != "" {
			message = fmt.Sprintf("Task delete blocked by policy: %s", reason)
		}
		writeV2Error(w, http.StatusForbidden, "FORBIDDEN", message, map[string]interface{}{
			"task_id":    taskID,
			"project_id": task.ProjectID,
			"reason":     reason,
		})
		return
	}

	lease, err := GetLeaseByTaskID(db, taskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}
	if lease != nil {
		if _, _, err := ReleaseLease(db, lease.ID, "task_deleted"); err != nil {
			if isSQLiteBusyError(err) {
				writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry delete", map[string]interface{}{
					"task_id":  taskID,
					"lease_id": lease.ID,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"task_id":  taskID,
				"lease_id": lease.ID,
			})
			return
		}
	}

	_, err = db.Exec("UPDATE tasks SET parent_task_id = NULL WHERE parent_task_id = ?", taskID)
	if err != nil {
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry delete", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry delete", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	go broadcastUpdate(v1EventTypeTasks)

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"task": task,
	})
}

func v2TaskDetailRouteHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "claim" {
		v2TaskClaimHandler(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "complete" {
		v2TaskCompleteHandler(w, r, parts[0])
		return
	}
	if len(parts) == 3 && parts[1] == "dependencies" {
		v2TaskDependencyDetailHandler(w, r, parts[0], parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "dependencies" {
		v2TaskDependenciesHandler(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "approve" {
		v2TaskApproveHandler(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "reject" {
		v2TaskRejectHandler(w, r, parts[0])
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			v2GetTaskDetailHandler(w, r)
		case http.MethodPatch:
			v2UpdateTaskHandler(w, r, parts[0])
		case http.MethodDelete:
			v2DeleteTaskHandler(w, r, parts[0])
		default:
			writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
		}
		return
	}

	writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Not found", map[string]interface{}{
		"path": r.URL.Path,
	})
}

// claimNextTaskTx finds the next claimable task in a project inside the given
// transaction.  It uses priority-aware ordering so high-priority tasks are
// picked before lower ones.  Returns (taskID, true, nil) when a candidate is
// found, (0, false, nil) when the queue is empty, or (0, false, err) on DB
// errors.
func claimNextTaskTx(tx *sql.Tx, projectID string, now string) (int, bool, error) {
	var taskID int
	var statusBucket int
	err := tx.QueryRow(`
		SELECT t.id,
		       CASE WHEN t.status = ? THEN 0 ELSE 1 END AS status_bucket
		FROM tasks t
		LEFT JOIN leases l ON t.id = l.task_id AND l.expires_at > ?
		WHERE t.project_id = ? AND t.status IN (?, ?) AND l.id IS NULL
		  AND (COALESCE(t.requires_approval, 0) = 0 OR t.approval_status = 'approved')
		ORDER BY status_bucket, t.priority DESC, t.id ASC
		LIMIT 1
	`, StatusNotPicked, now, projectID, StatusNotPicked, StatusInProgress).Scan(&taskID, &statusBucket)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return taskID, true, nil
}

// claimNextTaskTxWithCaps finds the next claimable task, optionally preferring
// tasks whose type matches the agent's declared capabilities.
func claimNextTaskTxWithCaps(tx *sql.Tx, projectID string, now string, capabilities []string) (int, bool, error) {
	if len(capabilities) == 0 {
		return claimNextTaskTx(tx, projectID, now)
	}

	// Build a CASE expression that boosts tasks matching agent capabilities.
	// Capabilities are matched against the task type column.
	placeholders := make([]string, len(capabilities))
	args := []interface{}{StatusNotPicked, now, projectID, StatusNotPicked, StatusInProgress}
	for i, cap := range capabilities {
		placeholders[i] = "?"
		args = append(args, strings.ToUpper(cap))
	}
	capMatch := strings.Join(placeholders, ",")

	query := fmt.Sprintf(`
		SELECT t.id,
		       CASE WHEN t.status = ? THEN 0 ELSE 1 END AS status_bucket,
		       CASE WHEN UPPER(COALESCE(t.type, '')) IN (%s) THEN 0 ELSE 1 END AS cap_bucket
		FROM tasks t
		LEFT JOIN leases l ON t.id = l.task_id AND l.expires_at > ?
		WHERE t.project_id = ? AND t.status IN (?, ?) AND l.id IS NULL
		  AND (COALESCE(t.requires_approval, 0) = 0 OR t.approval_status = 'approved')
		ORDER BY status_bucket, cap_bucket, t.priority DESC, t.id ASC
		LIMIT 1
	`, capMatch)

	// Reorder args: StatusNotPicked(status_bucket), caps..., now, projectID, StatusNotPicked, StatusInProgress
	reorderedArgs := make([]interface{}, 0, len(args))
	reorderedArgs = append(reorderedArgs, StatusNotPicked) // status_bucket CASE
	for _, cap := range capabilities {
		reorderedArgs = append(reorderedArgs, strings.ToUpper(cap)) // cap_bucket IN(...)
	}
	reorderedArgs = append(reorderedArgs, now)                               // leases expires_at
	reorderedArgs = append(reorderedArgs, projectID)                         // project_id
	reorderedArgs = append(reorderedArgs, StatusNotPicked, StatusInProgress) // status IN

	var taskID, statusBucket, capBucket int
	err := tx.QueryRow(query, reorderedArgs...).Scan(&taskID, &statusBucket, &capBucket)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return taskID, true, nil
}

// spawnIdlePlannerTx creates an idle-planner task inside the supplied
// transaction.  It deduplicates via automation_emissions and checks that no
// planner task is already open.  Returns (newTaskID, true, nil) on success,
// (0, false, nil) when dedupe/planner-check prevents creation, or
// (0, false, err) on DB errors.  The caller must commit or rollback the tx.
func spawnIdlePlannerTx(tx *sql.Tx, projectID string) (int64, bool, error) {
	dedupeKey := computeEmissionDedupeKey(projectID, "idle_planner", GetEmissionWindow())
	nowUnix := time.Now().Unix()
	emResult, emErr := tx.Exec(`
		INSERT OR IGNORE INTO automation_emissions (dedupe_key, project_id, kind, task_id, created_at)
		VALUES (?, ?, ?, NULL, ?)
	`, dedupeKey, projectID, "idle_planner", nowUnix)
	if emErr != nil {
		return 0, false, fmt.Errorf("emission insert: %w", emErr)
	}

	rowsAffected, raErr := emResult.RowsAffected()
	if raErr != nil {
		return 0, false, fmt.Errorf("RowsAffected: %w", raErr)
	}
	if rowsAffected == 0 {
		return 0, false, nil // dedupe hit
	}

	// Hard planner-already-open check
	var openPlannerCount int
	if err := tx.QueryRow(`
		SELECT COUNT(*) FROM tasks
		WHERE project_id = ? AND tags_json LIKE '%"planner"%'
		  AND status IN (?, ?)
	`, projectID, StatusNotPicked, StatusInProgress).Scan(&openPlannerCount); err != nil {
		return 0, false, fmt.Errorf("planner check: %w", err)
	}
	if openPlannerCount > 0 {
		log.Printf("idle-planner: planner already open in %s (count=%d)", projectID, openPlannerCount)
		return 0, false, nil
	}

	idlePlannerInstructions := "You are the Idle Planner. The queue is empty. Your job is to generate the next actionable tasks for this project.\n" +
		"Scan recent events, open tasks, repo state, and policies.\n" +
		"Create 3-10 concrete tasks that move the project forward (implementation, tests, cleanup, docs).\n" +
		"If you cannot identify meaningful work, create a single task asking the operator what to prioritize next and then stop.\n" +
		"Do not generate another idle-planner task."

	tagsJSON := `["auto","idle","planner"]`
	createdNow := nowISO()
	taskResult, taskErr := tx.Exec(`
		INSERT INTO tasks (title, instructions, status, status_v2, type, priority, project_id, tags_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "Idle Planner", idlePlannerInstructions, StatusNotPicked, TaskStatusQueued, TaskTypePlan, 100, projectID, tagsJSON, createdNow, createdNow)
	if taskErr != nil {
		return 0, false, fmt.Errorf("task insert: %w", taskErr)
	}

	newTaskID, liErr := taskResult.LastInsertId()
	if liErr != nil {
		return 0, false, fmt.Errorf("LastInsertId: %w", liErr)
	}

	if _, emUpErr := tx.Exec(`UPDATE automation_emissions SET task_id = ? WHERE dedupe_key = ?`, newTaskID, dedupeKey); emUpErr != nil {
		return 0, false, fmt.Errorf("emission update: %w", emUpErr)
	}

	return newTaskID, true, nil
}

func v2TaskClaimHandler(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	taskID, err := strconv.Atoi(rawID)
	if err != nil || taskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawID,
		})
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
		Mode    string `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if strings.TrimSpace(req.AgentID) == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "agent_id is required", map[string]interface{}{
			"agent_id": req.AgentID,
		})
		return
	}

	if req.Mode == "" {
		req.Mode = "exclusive"
	}

	if _, err := GetTaskV2(db, taskID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	// Use canonical assignment service for consistent claim behavior.
	envelope, claimErr := ClaimTaskByID(db, taskID, req.AgentID, req.Mode)
	if claimErr != nil {
		if errors.Is(claimErr, ErrLeaseConflict) || isLeaseConflictError(claimErr) {
			writeV2Error(w, http.StatusConflict, "CONFLICT", "Task is already leased by another agent", map[string]interface{}{
				"task_id":  taskID,
				"agent_id": req.AgentID,
			})
			return
		}
		if isSQLiteBusyError(claimErr) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry claim", map[string]interface{}{
				"task_id":  taskID,
				"agent_id": req.AgentID,
			})
			return
		}
		log.Printf("Error claiming task: %v", claimErr)
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Internal server error", map[string]interface{}{
			"task_id":  taskID,
			"agent_id": req.AgentID,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"lease":   envelope.Lease,
		"task":    envelope.Task,
		"run":     envelope.Run,
		"context": envelope.Context,
	})
}

// v2ProjectTasksClaimNextHandler handles POST /api/v2/projects/:id/tasks/claim-next
// It finds the next available task in the project, claims it atomically in a
// single transaction (select + lease + status update + run creation), and returns
// the claimed {task, lease, run}.
// If no task is available, it attempts idle planner emission + creation + one retry.
// Returns 204 No Content when nothing is claimable and idle planner cannot spawn.
// Returns 202 Accepted when idle planner was spawned but could not be claimed in
// the same call (rare edge case).
func v2ProjectTasksClaimNextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Extract project ID from URL path (/api/v2/projects/:id/tasks/claim-next)
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project ID", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	projectID := parts[0]

	var req struct {
		AgentID      string   `json:"agent_id"`
		Mode         string   `json:"mode"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}
	if strings.TrimSpace(req.AgentID) == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "agent_id is required", map[string]interface{}{
			"agent_id": req.AgentID,
		})
		return
	}
	if req.Mode == "" {
		req.Mode = "exclusive"
	}

	// If capabilities not provided in request, look up from agent profile
	agentCaps := req.Capabilities
	if len(agentCaps) == 0 {
		if agent, err := GetAgent(db, req.AgentID); err == nil && len(agent.Capabilities) > 0 {
			agentCaps = agent.Capabilities
		}
	}

	// Auto-register or touch agent record
	_ = EnsureAgent(db, req.AgentID)

	idlePlannerAttempted := false
	idlePlannerSpawned := false
	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Jittered backoff on retries (10-50ms) — only after first attempt
		if attempt > 0 {
			jitter := time.Duration(10+rand.Intn(41)) * time.Millisecond
			time.Sleep(jitter)
		}

		// Find next claimable task (read-only transaction — always rolled back).
		findTx, txErr := db.Begin()
		if txErr != nil {
			if isSQLiteBusyError(txErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to begin transaction", nil)
			return
		}

		// Step 1: SELECT candidate task inside read-only tx
		now := nowISO()
		taskID, found, qErr := claimNextTaskTxWithCaps(findTx, projectID, now, agentCaps)
		_ = findTx.Rollback() // always rollback — we only needed it for the read

		if qErr != nil {
			if isSQLiteBusyError(qErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query tasks", map[string]interface{}{
				"project_id": projectID,
			})
			return
		}

		if !found {
			// No task available — attempt idle planner spawn (once)
			if idlePlannerAttempted {
				if idlePlannerSpawned {
					// Spawned idle planner but couldn't claim it — 202
					writeV2JSON(w, http.StatusAccepted, map[string]interface{}{
						"message": "Idle planner task spawned but could not be claimed in this call",
					})
					return
				}
				CreateEvent(db, projectID, "project.idle", "project", projectID, map[string]interface{}{
					"agent_id": req.AgentID,
				})
				w.WriteHeader(http.StatusNoContent)
				return
			}
			idlePlannerAttempted = true

			// Policy gate
			blocked, reason, pErr := isAutomationBlockedByPolicies(db, projectID)
			if pErr != nil {
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Policy check failed", map[string]interface{}{
					"project_id": projectID,
					"error":      pErr.Error(),
				})
				return
			}
			if blocked {
				log.Printf("idle-planner-v2: blocked by policy: %s", reason)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			spawnTx, spawnTxErr := db.Begin()
			if spawnTxErr != nil {
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to begin transaction", nil)
				return
			}
			newTaskID, created, spawnErr := spawnIdlePlannerTx(spawnTx, projectID)
			if spawnErr != nil {
				_ = spawnTx.Rollback()
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Idle planner spawn failed", map[string]interface{}{
					"project_id": projectID,
					"error":      spawnErr.Error(),
				})
				return
			}
			if !created {
				_ = spawnTx.Rollback()
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if cmErr := spawnTx.Commit(); cmErr != nil {
				_ = spawnTx.Rollback()
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to commit idle planner", nil)
				return
			}
			idlePlannerSpawned = true
			log.Printf("idle-planner-v2: created task %d in %s", newTaskID, projectID)
			continue // retry claim to pick up the newly spawned task
		}

		// Step 2: Claim via canonical assignment service (handles lease, run, event, context).
		envelope, claimErr := ClaimTaskByID(db, taskID, req.AgentID, req.Mode)
		if claimErr != nil {
			if errors.Is(claimErr, ErrLeaseConflict) || isLeaseConflictError(claimErr) || isSQLiteBusyError(claimErr) {
				continue
			}
			log.Printf("claim-next: failed to claim task %d: %v", taskID, claimErr)
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to claim task", map[string]interface{}{
				"task_id":  taskID,
				"agent_id": req.AgentID,
			})
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"task":    envelope.Task,
			"lease":   envelope.Lease,
			"run":     envelope.Run,
			"context": envelope.Context,
		})
		return
	}

	// Exhausted attempts
	if idlePlannerSpawned {
		// We spawned idle planner but couldn't claim after retries
		writeV2JSON(w, http.StatusAccepted, map[string]interface{}{
			"message": "Idle planner task spawned but could not be claimed in this call",
		})
		return
	}
	writeV2Error(w, http.StatusConflict, "CONFLICT", "Could not claim a task after multiple attempts", map[string]interface{}{
		"project_id": projectID,
	})
}

// v2TryIdlePlannerSpawn attempts to create an idle planner task for the project.
// Returns (true, nil) if a planner was created, (false, nil) if dedupe/policy blocked it,
// or (false, err) on unexpected errors.
// Delegates to the shared spawnIdlePlannerTx helper.
func v2TryIdlePlannerSpawn(projectID string) (bool, error) {
	blocked, reason, pErr := isAutomationBlockedByPolicies(db, projectID)
	if pErr != nil {
		return false, fmt.Errorf("policy check: %w", pErr)
	}
	if blocked {
		log.Printf("idle-planner-v2: blocked by policy: %s", reason)
		return false, nil
	}

	tx, txErr := db.Begin()
	if txErr != nil {
		return false, fmt.Errorf("begin tx: %w", txErr)
	}

	newTaskID, created, err := spawnIdlePlannerTx(tx, projectID)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if !created {
		_ = tx.Rollback()
		return false, nil
	}
	if cmErr := tx.Commit(); cmErr != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("commit: %w", cmErr)
	}

	log.Printf("idle-planner-v2: created task %d in %s", newTaskID, projectID)
	return true, nil
}

func v2TaskCompleteHandler(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	taskID, err := strconv.Atoi(rawID)
	if err != nil || taskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", map[string]interface{}{
			"task_id": rawID,
		})
		return
	}

	type completionNextTask struct {
		Title        *string  `json:"title"`
		Instructions string   `json:"instructions"`
		Type         *string  `json:"type"`
		Priority     *int     `json:"priority"`
		Tags         []string `json:"tags"`
	}
	type completionResult struct {
		Summary      string               `json:"summary"`
		ChangesMade  []string             `json:"changes_made"`
		FilesTouched []string             `json:"files_touched"`
		CommandsRun  []string             `json:"commands_run"`
		TestsRun     []string             `json:"tests_run"`
		Risks        []string             `json:"risks"`
		NextTasks    []completionNextTask `json:"next_tasks"`
	}
	type normalizedNextTask struct {
		title        *string
		instructions string
		taskType     *TaskType
		priority     *int
		tags         []string
	}
	var req struct {
		Output  *string           `json:"output"`
		Message *string           `json:"message"`
		Result  *completionResult `json:"result"`
	}

	nextTasks := make([]normalizedNextTask, 0)
	var rawResult map[string]interface{}
	var hasResult bool

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}
	if strings.TrimSpace(string(bodyBytes)) != "" {
		var raw map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &raw); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
				"content_type": r.Header.Get("Content-Type"),
			})
			return
		}

		if rawResultValue, ok := raw["result"]; ok {
			resultMap, ok := rawResultValue.(map[string]interface{})
			if !ok {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "result must be an object", nil)
				return
			}
			rawResult = resultMap
			hasResult = true
		}

		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
				"content_type": r.Header.Get("Content-Type"),
			})
			return
		}
	}

	task, err := GetTaskV2(db, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	if task.StatusV1 == StatusComplete {
		writeV2Error(w, http.StatusConflict, "CONFLICT", "Task is already complete", map[string]interface{}{
			"task_id": taskID,
			"status":  task.StatusV1,
		})
		return
	}

	requiredFields := make([]string, 0)
	switch task.Type {
	case TaskTypeModify:
		requiredFields = append(requiredFields, "changes_made", "files_touched")
	case TaskTypeTest:
		requiredFields = append(requiredFields, "tests_run")
	case TaskTypeDoc:
		requiredFields = append(requiredFields, "summary")
	case TaskTypeReview:
		requiredFields = append(requiredFields, "summary", "risks")
	}

	if len(requiredFields) > 0 {
		missingFields := make([]string, 0)
		if !hasResult {
			for _, field := range requiredFields {
				missingFields = append(missingFields, "result."+field)
			}
		} else {
			for _, field := range requiredFields {
				value, ok := rawResult[field]
				if !ok || value == nil {
					missingFields = append(missingFields, "result."+field)
				}
			}
		}
		if len(missingFields) > 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Missing required completion result fields", map[string]interface{}{
				"missing_fields": missingFields,
				"task_type":      task.Type,
			})
			return
		}
	}

	if hasResult {
		validateStringArray := func(field string, value interface{}) bool {
			items, ok := value.([]interface{})
			if !ok {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "result."+field+" must be an array of strings", nil)
				return false
			}
			for idx, item := range items {
				if _, ok := item.(string); !ok {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "result."+field+" must be an array of strings", map[string]interface{}{
						"index": idx,
					})
					return false
				}
			}
			return true
		}
		validateNonEmptyString := func(field string, value interface{}) bool {
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) == "" {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "result."+field+" must be a non-empty string", nil)
				return false
			}
			return true
		}

		if value, ok := rawResult["summary"]; ok {
			if !validateNonEmptyString("summary", value) {
				return
			}
		}
		if value, ok := rawResult["changes_made"]; ok {
			if !validateStringArray("changes_made", value) {
				return
			}
		}
		if value, ok := rawResult["files_touched"]; ok {
			if !validateStringArray("files_touched", value) {
				return
			}
		}
		if value, ok := rawResult["commands_run"]; ok {
			if !validateStringArray("commands_run", value) {
				return
			}
		}
		if value, ok := rawResult["tests_run"]; ok {
			if !validateStringArray("tests_run", value) {
				return
			}
		}
		if value, ok := rawResult["risks"]; ok {
			if !validateStringArray("risks", value) {
				return
			}
		}

		if nextTasksRaw, ok := rawResult["next_tasks"]; ok {
			nextTasksList, ok := nextTasksRaw.([]interface{})
			if !ok {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "result.next_tasks must be an array", nil)
				return
			}

			nextTasks = make([]normalizedNextTask, 0, len(nextTasksList))
			for idx, rawTask := range nextTasksList {
				taskMap, ok := rawTask.(map[string]interface{})
				if !ok {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks entries must be objects", map[string]interface{}{
						"index": idx,
					})
					return
				}

				rawTitle, ok := taskMap["title"]
				if !ok {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.title is required", map[string]interface{}{
						"index": idx,
					})
					return
				}
				titleStr, ok := rawTitle.(string)
				if !ok || strings.TrimSpace(titleStr) == "" {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.title is required", map[string]interface{}{
						"index": idx,
					})
					return
				}
				trimmedTitle := strings.TrimSpace(titleStr)

				rawInstructions, ok := taskMap["instructions"]
				if !ok {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.instructions is required", map[string]interface{}{
						"index": idx,
					})
					return
				}
				instructionsStr, ok := rawInstructions.(string)
				if !ok || strings.TrimSpace(instructionsStr) == "" {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.instructions is required", map[string]interface{}{
						"index": idx,
					})
					return
				}
				trimmedInstructions := strings.TrimSpace(instructionsStr)

				rawType, ok := taskMap["type"]
				if !ok {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.type is required", map[string]interface{}{
						"index": idx,
					})
					return
				}
				typeStr, ok := rawType.(string)
				if !ok || strings.TrimSpace(typeStr) == "" {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.type is required", map[string]interface{}{
						"index": idx,
					})
					return
				}
				trimmedType := strings.ToUpper(strings.TrimSpace(typeStr))
				if _, ok := v2TaskListTypeFilter[trimmedType]; !ok {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.type is invalid", map[string]interface{}{
						"index": idx,
						"type":  typeStr,
					})
					return
				}
				parsedType := TaskType(trimmedType)

				rawPriority, ok := taskMap["priority"]
				if !ok {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.priority is required", map[string]interface{}{
						"index": idx,
					})
					return
				}
				priorityValue, ok := rawPriority.(float64)
				if !ok || priorityValue < 0 || math.Trunc(priorityValue) != priorityValue {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.priority must be a non-negative integer", map[string]interface{}{
						"index":    idx,
						"priority": rawPriority,
					})
					return
				}
				priority := int(priorityValue)

				var tags []string
				if rawTags, ok := taskMap["tags"]; ok {
					list, ok := rawTags.([]interface{})
					if !ok {
						writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.tags must be an array of strings", map[string]interface{}{
							"index": idx,
						})
						return
					}
					for _, tagItem := range list {
						tagStr, ok := tagItem.(string)
						if !ok {
							writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "next_tasks.tags must be an array of strings", map[string]interface{}{
								"index": idx,
							})
							return
						}
						trimmed := strings.TrimSpace(tagStr)
						if trimmed != "" {
							tags = append(tags, trimmed)
						}
					}
				}

				nextTasks = append(nextTasks, normalizedNextTask{
					title:        &trimmedTitle,
					instructions: trimmedInstructions,
					taskType:     &parsedType,
					priority:     &priority,
					tags:         tags,
				})
			}
		}
	}

	blocked, reason, err := isCompletionBlockedByPolicies(db, task.ProjectID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}
	if blocked {
		message := "Task completion blocked by policy"
		if reason != "" {
			message = fmt.Sprintf("Task completion blocked by policy: %s", reason)
		}
		writeV2Error(w, http.StatusForbidden, "FORBIDDEN", message, map[string]interface{}{
			"task_id": taskID,
			"reason":  reason,
		})
		return
	}

	output := task.Output
	if req.Output != nil {
		output = req.Output
	} else if req.Message != nil {
		output = req.Message
	}

	// Use canonical completion path with structured summary extraction.
	// CompleteTaskWithPayload: status update, run, lease, event, memory extraction, SSE.
	completedTask, runSummary, completeErr := CompleteTaskWithPayload(db, taskID, output)
	if completeErr != nil {
		if isSQLiteBusyError(completeErr) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry completion", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		if strings.Contains(completeErr.Error(), "already completed") {
			writeV2Error(w, http.StatusConflict, "CONFLICT", "Task was already completed by another agent", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", completeErr.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	if len(nextTasks) > 0 {
		for _, nextTask := range nextTasks {
			child, err := CreateTaskV2WithMeta(db, nextTask.instructions, completedTask.ProjectID, &taskID, nextTask.title, nextTask.taskType, nextTask.priority, nextTask.tags)
			if err != nil {
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
					"task_id": taskID,
				})
				return
			}

			payload := buildTaskCreatedPayload(child, &taskID)
			if _, err := CreateEvent(db, completedTask.ProjectID, "task.created", "task", fmt.Sprintf("%d", child.ID), payload); err != nil {
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
					"task_id": taskID,
				})
				return
			}
		}

		go broadcastUpdate(v1EventTypeTasks)
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"task":    completedTask,
		"summary": runSummary,
	})
}
