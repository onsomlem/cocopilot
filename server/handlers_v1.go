package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getAllTasksJSON() ([]Task, error) {
	return getAllTasksJSONWithFilters(nil, nil, nil)
}

func getAllTasksJSONWithFilters(status *TaskStatus, updatedSince *string, projectID *string) ([]Task, error) {
	baseQuery, args := buildTaskListQuery(status, updatedSince, projectID)
	query := baseQuery + " ORDER BY created_at DESC"
	return fetchTaskList(query, args...)
}

func getAllTasksJSONWithFiltersLimit(status *TaskStatus, updatedSince *string, projectID *string, limit int) ([]Task, error) {
	baseQuery, args := buildTaskListQuery(status, updatedSince, projectID)
	query := baseQuery + " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)
	return fetchTaskList(query, args...)
}

func getAllTasksJSONWithFiltersPage(status *TaskStatus, updatedSince *string, projectID *string, limit int, offset int, sortField string, sortDirection string) ([]Task, int, error) {
	total, err := countTasksWithFilters(status, updatedSince, projectID)
	if err != nil {
		return nil, 0, err
	}

	baseQuery, args := buildTaskListQuery(status, updatedSince, projectID)
	orderDirection := strings.ToLower(strings.TrimSpace(sortDirection))
	if orderDirection == "" {
		orderDirection = "desc"
	}
	if orderDirection != "asc" && orderDirection != "desc" {
		return nil, 0, fmt.Errorf("invalid sort direction")
	}

	orderBy := "created_at " + strings.ToUpper(orderDirection)
	switch strings.ToLower(strings.TrimSpace(sortField)) {
	case "", "created_at":
		orderBy = "created_at " + strings.ToUpper(orderDirection)
	case "updated_at":
		orderBy = "COALESCE(updated_at, created_at) " + strings.ToUpper(orderDirection)
	default:
		return nil, 0, fmt.Errorf("invalid sort field")
	}

	query := baseQuery + " ORDER BY " + orderBy + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	tasks, err := fetchTaskList(query, args...)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func buildTaskListQuery(status *TaskStatus, updatedSince *string, projectID *string) (string, []interface{}) {
	query := strings.Builder{}
	query.WriteString("SELECT id, instructions, status, output, parent_task_id, created_at, updated_at FROM tasks")

	filters := make([]string, 0, 3)
	args := make([]interface{}, 0, 3)
	if status != nil {
		filters = append(filters, "status = ?")
		args = append(args, *status)
	}
	if updatedSince != nil {
		filters = append(filters, "COALESCE(updated_at, created_at) >= ?")
		args = append(args, *updatedSince)
	}
	if projectID != nil {
		filters = append(filters, "project_id = ?")
		args = append(args, *projectID)
	}
	if len(filters) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(filters, " AND "))
	}

	return query.String(), args
}

func countTasksWithFilters(status *TaskStatus, updatedSince *string, projectID *string) (int, error) {
	query, args := buildTaskListQuery(status, updatedSince, projectID)
	countQuery := strings.Builder{}
	countQuery.WriteString("SELECT COUNT(1) FROM (")
	countQuery.WriteString(query)
	countQuery.WriteString(") AS filtered")

	var total int
	if err := db.QueryRow(countQuery.String(), args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func fetchTaskList(query string, args ...interface{}) ([]Task, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		var output sql.NullString
		var parentID sql.NullInt64
		var updatedAt sql.NullString
		err := rows.Scan(&t.ID, &t.Instructions, &t.Status, &output, &parentID, &t.CreatedAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		if output.Valid {
			outputStr := strings.ReplaceAll(output.String, "\\n", "\n")
			t.Output = &outputStr
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			t.ParentTaskID = &pid
		}
		if updatedAt.Valid && updatedAt.String != "" {
			t.UpdatedAt = updatedAt.String
		} else {
			t.UpdatedAt = t.CreatedAt
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func parseTaskStatusFilter(raw string) (TaskStatus, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(StatusNotPicked):
		return StatusNotPicked, true
	case string(StatusInProgress):
		return StatusInProgress, true
	case string(StatusComplete):
		return StatusComplete, true
	case string(StatusFailed):
		return StatusFailed, true
	default:
		return "", false
	}
}

func resolveV1TaskListSort(raw string) (string, string, error) {
	sort := strings.ToLower(strings.TrimSpace(raw))
	if sort == "" {
		return "created_at", "desc", nil
	}

	switch sort {
	case "created_at:asc":
		return "created_at", "asc", nil
	case "created_at:desc":
		return "created_at", "desc", nil
	case "updated_at":
		return "updated_at", "desc", nil
	default:
		return "", "", fmt.Errorf("invalid sort option")
	}
}

func getTasksByProjectJSON(projectID string) ([]Task, error) {
	rows, err := db.Query("SELECT id, instructions, status, output, parent_task_id, created_at, updated_at FROM tasks WHERE project_id = ? ORDER BY created_at DESC", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		var output sql.NullString
		var parentID sql.NullInt64
		var updatedAt sql.NullString
		err := rows.Scan(&t.ID, &t.Instructions, &t.Status, &output, &parentID, &t.CreatedAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		if output.Valid {
			outputStr := strings.ReplaceAll(output.String, "\\n", "\n")
			t.Output = &outputStr
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			t.ParentTaskID = &pid
		}
		if updatedAt.Valid && updatedAt.String != "" {
			t.UpdatedAt = updatedAt.String
		} else {
			t.UpdatedAt = t.CreatedAt
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func apiTasksHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var projectIDFilter *string
	if rawValues, ok := query["project_id"]; ok {
		rawProjectID := ""
		if len(rawValues) > 0 {
			rawProjectID = strings.TrimSpace(rawValues[0])
		}
		if rawProjectID == "" {
			http.Error(w, "project_id cannot be empty", http.StatusBadRequest)
			return
		}
		if _, err := GetProject(db, rawProjectID); err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "project_id not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		projectIDFilter = &rawProjectID
	}

	var statusFilter *TaskStatus
	if rawValues, ok := query["status"]; ok {
		rawStatus := ""
		if len(rawValues) > 0 {
			rawStatus = strings.TrimSpace(rawValues[0])
		}
		if rawStatus == "" {
			http.Error(w, "status cannot be empty", http.StatusBadRequest)
			return
		}
		parsed, ok := parseTaskStatusFilter(rawStatus)
		if !ok {
			http.Error(w, "status must be NOT_PICKED, IN_PROGRESS, COMPLETE, or FAILED", http.StatusBadRequest)
			return
		}
		statusFilter = &parsed
	}

	var updatedSinceFilter *string
	if rawValues, ok := query["updated_since"]; ok {
		rawSince := ""
		if len(rawValues) > 0 {
			rawSince = strings.TrimSpace(rawValues[0])
		}
		if rawSince == "" {
			http.Error(w, "updated_since cannot be empty", http.StatusBadRequest)
			return
		}
		parsed, err := time.Parse(time.RFC3339Nano, rawSince)
		if err != nil {
			http.Error(w, "updated_since must be RFC3339", http.StatusBadRequest)
			return
		}
		formatted := parsed.UTC().Format(leaseTimeFormat)
		updatedSinceFilter = &formatted
	}

	limit := v1TasksListDefaultLimit
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		if parsed > v1TasksListMaxLimit {
			limit = v1TasksListMaxLimit
		} else {
			limit = parsed
		}
	}

	offset := 0
	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil || parsed < 0 {
			http.Error(w, "offset must be a non-negative integer", http.StatusBadRequest)
			return
		}
		offset = parsed
	}

	var sortField string
	var sortDirection string
	if rawValues, ok := query["sort"]; ok {
		rawSort := ""
		if len(rawValues) > 0 {
			rawSort = strings.TrimSpace(rawValues[0])
		}
		if rawSort == "" {
			http.Error(w, "sort cannot be empty", http.StatusBadRequest)
			return
		}
		var err error
		sortField, sortDirection, err = resolveV1TaskListSort(rawSort)
		if err != nil {
			http.Error(w, "sort must be created_at:asc, created_at:desc, or updated_at", http.StatusBadRequest)
			return
		}
	} else {
		var err error
		sortField, sortDirection, err = resolveV1TaskListSort("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	tasks, total, err := getAllTasksJSONWithFiltersPage(statusFilter, updatedSinceFilter, projectIDFilter, limit, offset, sortField, sortDirection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
		"total": total,
	})
}

func eventsHandler(heartbeatInterval time.Duration, replayLimitMax int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		var eventTypeFilter string
		if rawValues, ok := query["type"]; ok {
			rawType := ""
			if len(rawValues) > 0 {
				rawType = strings.TrimSpace(rawValues[0])
			}
			if rawType == "" {
				http.Error(w, "type cannot be empty", http.StatusBadRequest)
				return
			}
			normalized, ok := normalizeV1EventType(rawType)
			if !ok {
				http.Error(w, "type is not supported", http.StatusBadRequest)
				return
			}
			eventTypeFilter = normalized
		}
		var projectIDFilter *string
		if rawValues, ok := query["project_id"]; ok {
			rawProjectID := ""
			if len(rawValues) > 0 {
				rawProjectID = strings.TrimSpace(rawValues[0])
			}
			if rawProjectID == "" {
				http.Error(w, "project_id cannot be empty", http.StatusBadRequest)
				return
			}
			if _, err := GetProject(db, rawProjectID); err != nil {
				if strings.Contains(err.Error(), "not found") {
					http.Error(w, "project_id not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			projectIDFilter = &rawProjectID
		}

		var updatedSinceFilter *string
		if rawValues, ok := query["since"]; ok {
			rawSince := ""
			if len(rawValues) > 0 {
				rawSince = strings.TrimSpace(rawValues[0])
			}
			if rawSince == "" {
				http.Error(w, "since cannot be empty", http.StatusBadRequest)
				return
			}
			parsed, err := time.Parse(time.RFC3339Nano, rawSince)
			if err != nil {
				http.Error(w, "since must be RFC3339", http.StatusBadRequest)
				return
			}
			formatted := parsed.UTC().Format(leaseTimeFormat)
			updatedSinceFilter = &formatted
		}

		if replayLimitMax <= 0 {
			replayLimitMax = v1EventsReplayLimitMaxDefault
		}

		replayLimit := 0
		if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
			if updatedSinceFilter == nil {
				http.Error(w, "limit requires since", http.StatusBadRequest)
				return
			}
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed <= 0 {
				http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
				return
			}
			if parsed > replayLimitMax {
				replayLimit = replayLimitMax
			} else {
				replayLimit = parsed
			}
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		clientChan := make(chan string, 10)
		sseMutex.Lock()
		sseClients = append(sseClients, v1SSEClient{ch: clientChan, projectID: projectIDFilter, eventType: eventTypeFilter})
		sseMutex.Unlock()

		defer func() {
			sseMutex.Lock()
			for i, c := range sseClients {
				if c.ch == clientChan {
					sseClients = append(sseClients[:i], sseClients[i+1:]...)
					break
				}
			}
			sseMutex.Unlock()
			close(clientChan)
		}()

		// Send initial data
		if eventTypeFilter == "" || eventTypeFilter == v1EventTypeTasks {
			var tasks []Task
			var err error
			if updatedSinceFilter != nil {
				if replayLimit == 0 {
					replayLimit = replayLimitMax
				}
				tasks, err = getAllTasksJSONWithFiltersLimit(nil, updatedSinceFilter, projectIDFilter, replayLimit)
			} else {
				tasks, err = getAllTasksJSONWithFilters(nil, updatedSinceFilter, projectIDFilter)
			}
			if err == nil {
				data, _ := json.Marshal(tasks)
				fmt.Fprintf(w, "event: %s\n", v1EventTypeTasks)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}

		heartbeat := time.NewTicker(heartbeatInterval)
		defer heartbeat.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case eventType, ok := <-clientChan:
				if !ok {
					return
				}
				if eventType != v1EventTypeTasks {
					continue
				}
				tasks, err := getAllTasksJSONWithFilters(nil, nil, projectIDFilter)
				if err == nil {
					data, _ := json.Marshal(tasks)
					fmt.Fprintf(w, "event: %s\n", v1EventTypeTasks)
					fmt.Fprintf(w, "data: %s\n\n", data)
					flusher.Flush()
				}
			case <-heartbeat.C:
				// Keep-alive ping
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	}
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	contextDepthStr := r.URL.Query().Get("context_depth")
	contextDepth := 3
	if contextDepthStr != "" {
		if d, err := strconv.Atoi(contextDepthStr); err == nil {
			contextDepth = d
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		agentID = "default-agent"
	}

	var taskID int
	var status TaskStatus
	var instructions string
	var parentTaskID sql.NullInt64
	var createdAt string
	var updatedAt sql.NullString
	var updatedAtValue string

	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		projectID = DefaultProjectID
	}

	const maxClaimAttempts = 20
	claimed := false
	idlePlannerAttempted := false
	for attempt := 0; attempt < maxClaimAttempts; attempt++ {
		now := nowISO()

		// Use shared helper to find next claimable task
		findTx, txErr := db.Begin()
		if txErr != nil {
			if isSQLiteBusyError(txErr) {
				continue
			}
			http.Error(w, txErr.Error(), http.StatusInternalServerError)
			return
		}
		foundID, found, qErr := claimNextTaskTx(findTx, projectID, now)
		_ = findTx.Rollback() // read-only, always rollback

		if qErr != nil {
			if isSQLiteBusyError(qErr) {
				continue
			}
			http.Error(w, qErr.Error(), http.StatusInternalServerError)
			return
		}

		if !found {
			// Attempt idle planner spawn (once)
			if idlePlannerAttempted {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
				return
			}
			idlePlannerAttempted = true

			blocked, reason, pErr := isAutomationBlockedByPolicies(db, projectID)
			if pErr != nil {
				log.Printf("idle-planner: policy check error: %v", pErr)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
				return
			}
			if blocked {
				log.Printf("idle-planner: blocked by policy: %s", reason)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
				return
			}

			tx, txErr := db.Begin()
			if txErr != nil {
				log.Printf("idle-planner: begin tx error: %v", txErr)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
				return
			}

			newTaskID, created, spawnErr := spawnIdlePlannerTx(tx, projectID)
			if spawnErr != nil {
				_ = tx.Rollback()
				log.Printf("idle-planner: spawn error: %v", spawnErr)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
				return
			}
			if !created {
				_ = tx.Rollback()
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
				return
			}
			if cmErr := tx.Commit(); cmErr != nil {
				_ = tx.Rollback()
				log.Printf("idle-planner: commit error: %v", cmErr)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "No tasks available.\nWait 15 secs for new instructions.")
				return
			}
			log.Printf("idle-planner: created task %d in %s", newTaskID, projectID)
			// Retry claim so the agent picks up the new planner task
			continue
		}
		taskID = foundID

		// Use canonical assignment service for consistent claim behavior.
		envelope, claimErr := ClaimTaskByID(db, taskID, agentID, "exclusive")
		if claimErr != nil {
			if errors.Is(claimErr, ErrLeaseConflict) || isLeaseConflictError(claimErr) || isSQLiteBusyError(claimErr) {
				continue
			}
			http.Error(w, claimErr.Error(), http.StatusInternalServerError)
			return
		}

		status = StatusInProgress
		if envelope.Task != nil && envelope.Task.UpdatedAt != nil {
			updatedAtValue = *envelope.Task.UpdatedAt
		} else {
			updatedAtValue = nowISO()
		}

		go broadcastUpdate(v1EventTypeTasks)
		claimed = true
		break
	}

	if !claimed {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, "Task was claimed by another agent. Try again for next available task.")
		return
	}

	// Fetch task details for response (instructions, parent_task_id needed for context)
	err := db.QueryRow("SELECT instructions, parent_task_id, created_at, updated_at FROM tasks WHERE id = ?", taskID).
		Scan(&instructions, &parentTaskID, &createdAt, &updatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build context from ancestors
	var contextParts []string
	currentParentID := parentTaskID
	depth := 0

	for currentParentID.Valid && depth < contextDepth {
		var parentID int
		var parentOutput sql.NullString
		var nextParentID sql.NullInt64
		var parentCreatedAt string
		var parentUpdatedAt sql.NullString

		err := db.QueryRow("SELECT id, output, parent_task_id, created_at, updated_at FROM tasks WHERE id = ?", currentParentID.Int64).
			Scan(&parentID, &parentOutput, &nextParentID, &parentCreatedAt, &parentUpdatedAt)
		if err != nil {
			break
		}

		if parentOutput.Valid && parentOutput.String != "" {
			parentUpdatedAtValue := parentCreatedAt
			if parentUpdatedAt.Valid && strings.TrimSpace(parentUpdatedAt.String) != "" {
				parentUpdatedAtValue = parentUpdatedAt.String
			}
			contextParts = append([]string{fmt.Sprintf("=== Task #%d (UPDATED_AT: %s) Output ===\n%s", parentID, parentUpdatedAtValue, parentOutput.String)}, contextParts...)
		}

		currentParentID = nextParentID
		depth++
	}

	contextBlock := ""
	if len(contextParts) > 0 {
		contextBlock = "CONTEXT FROM PREVIOUS TASKS:\n" + strings.Join(contextParts, "\n\n") + "\n\n---\n\n"
	}

	if updatedAtValue == "" {
		if updatedAt.Valid && strings.TrimSpace(updatedAt.String) != "" {
			updatedAtValue = updatedAt.String
		} else {
			updatedAtValue = createdAt
		}
	}

	fmt.Fprintf(w, "AVAILABLE TASK ID: %d\nTASK_STATUS: %s\nUPDATED_AT: %s\n\n%sInstructions:\n%s", taskID, status, updatedAtValue, contextBlock, instructions)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	taskIDStr := r.FormValue("task_id")
	message := r.FormValue("message")

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		http.Error(w, "Invalid task_id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Check if task exists
	var exists int
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Task %d not found.", taskID)
		return
	}

	// Use canonical FinalizationService: updates status, run, lease, emits event, extracts memory.
	var output *string
	if message != "" {
		output = &message
	}
	completedTask, _, completeErr := CompleteTaskWithPayload(db, taskID, output)
	if completeErr != nil {
		http.Error(w, completeErr.Error(), http.StatusInternalServerError)
		return
	}

	updatedAt := ""
	if completedTask != nil && completedTask.UpdatedAt != nil {
		updatedAt = *completedTask.UpdatedAt
	}
	fmt.Fprintf(w, "Task %d saved and marked as COMPLETE.\nUPDATED_AT: %s", taskID, updatedAt)
}

func updateStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	taskIDStr := r.FormValue("task_id")
	status := r.FormValue("status")

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task_id"})
		return
	}

	// Route terminal status transitions through canonical finalization paths.
	switch TaskStatus(status) {
	case StatusComplete:
		task, _, completeErr := CompleteTaskWithPayload(db, taskID, nil)
		if completeErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": completeErr.Error()})
			return
		}
		updatedAt := ""
		if task != nil && task.UpdatedAt != nil {
			updatedAt = *task.UpdatedAt
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "updated_at": updatedAt})
		return
	case StatusFailed:
		errMsg := r.FormValue("error")
		if errMsg == "" {
			errMsg = "Marked as failed via status update"
		}
		task, failErr := FailTaskWithError(db, taskID, errMsg)
		if failErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": failErr.Error()})
			return
		}
		updatedAt := ""
		if task != nil && task.UpdatedAt != nil {
			updatedAt = *task.UpdatedAt
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "updated_at": updatedAt})
		return
	case StatusNotPicked, StatusInProgress:
		// Non-terminal transitions: update directly (re-queue or mark in-progress).
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid status. Valid values: NOT_PICKED, IN_PROGRESS, COMPLETE, FAILED"})
		return
	}

	// Check if task exists for non-terminal transitions.
	var exists int
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	// Map v1 status to v2 status for consistency.
	statusV2 := TaskStatusQueued
	if TaskStatus(status) == StatusInProgress {
		statusV2 = TaskStatusClaimed
	}

	updatedAt := nowISO()
	_, err = db.Exec("UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ?", status, string(statusV2), updatedAt, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	broadcastUpdate(v1EventTypeTasks)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "updated_at": updatedAt})
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		// Fall back to regular form parsing
		err = r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
	}

	instructions := r.FormValue("instructions")
	parentTaskIDStr := r.FormValue("parent_task_id")
	projectID := r.FormValue("project_id")

	// Default to 'proj_default' if not specified
	if projectID == "" {
		projectID = DefaultProjectID
	}

	var parentTaskID *int
	if parentTaskIDStr != "" {
		if id, err := strconv.Atoi(parentTaskIDStr); err == nil {
			parentTaskID = &id
		}
	}

	// Use v2 service for consistent task creation (status, status_v2, timestamps).
	task, err := CreateTaskV2(db, instructions, projectID, parentTaskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Emit task.created event for automation/SSE consistency.
	if _, evErr := CreateEvent(db, projectID, "task.created", "task", fmt.Sprintf("%d", task.ID), map[string]interface{}{
		"task_id":    task.ID,
		"project_id": projectID,
		"source":     "v1",
	}); evErr != nil {
		log.Printf("Warning: failed to emit task.created event: %v", evErr)
	}

	broadcastUpdate(v1EventTypeTasks)

	updatedAt := ""
	if task.UpdatedAt != nil {
		updatedAt = *task.UpdatedAt
	} else {
		updatedAt = task.CreatedAt
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "task_id": task.ID, "updated_at": updatedAt})
}

func instructionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusSeeOther)
	fmt.Fprint(w, getInstructions())
}

func instructionsDetailedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, getDetailedInstructions())
}

func getWorkdirHandler(w http.ResponseWriter, r *http.Request) {
	workdirMu.RLock()
	wd := workdir
	workdirMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"workdir": wd})
}

func setWorkdirHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	newWorkdir := r.FormValue("workdir")
	if err := validateWorkdir(newWorkdir); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	workdirMu.Lock()
	oldWorkdir := workdir
	workdir = filepath.Clean(newWorkdir)
	workdirMu.Unlock()

	// Emit audit event for workdir change (Section 8: sensitive op audit).
	CreateEvent(db, DefaultProjectID, "audit.workdir.changed", "config", "workdir", map[string]interface{}{
		"old_workdir": oldWorkdir,
		"new_workdir": filepath.Clean(newWorkdir),
		"client_ip":   r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "workdir": newWorkdir})
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		r.ParseForm()
	}

	taskIDStr := r.FormValue("task_id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task_id"})
		return
	}

	// Check if task exists
	var exists int
	err = db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	// Delete the task
	// Fetch project_id before deletion for audit event.
	var projectID string
	_ = db.QueryRow("SELECT COALESCE(project_id, ?) FROM tasks WHERE id = ?", DefaultProjectID, taskID).Scan(&projectID)

	_, err = db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Emit audit event for task deletion (Section 8: sensitive op audit).
	CreateEvent(db, projectID, "audit.task.deleted", "task", fmt.Sprintf("%d", taskID), map[string]interface{}{
		"task_id":   taskID,
		"client_ip": r.RemoteAddr,
		"source":    "v1",
	})

	broadcastUpdate(v1EventTypeTasks)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
