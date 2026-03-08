package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func assertV2MissingFields(t *testing.T, w *httptest.ResponseRecorder, expected ...string) {
	t.Helper()
	errField := assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	details, ok := errField["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error.details object, got %T", errField["details"])
	}
	missing, ok := details["missing_fields"].([]interface{})
	if !ok {
		t.Fatalf("expected missing_fields list, got %T", details["missing_fields"])
	}
	missingSet := map[string]struct{}{}
	for _, entry := range missing {
		field, ok := entry.(string)
		if !ok {
			t.Fatalf("expected missing field string, got %T", entry)
		}
		missingSet[field] = struct{}{}
	}
	for _, field := range expected {
		if _, ok := missingSet[field]; !ok {
			t.Fatalf("expected missing %s, got %v", field, missing)
		}
	}
}

func TestV2TaskCompleteSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Complete me", StatusInProgress, TaskStatusRunning, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}
	oldUpdatedAt := "2000-01-01T00:00:00.000000Z"
	if _, err := testDB.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldUpdatedAt, taskID); err != nil {
		t.Fatalf("failed to seed updated_at: %v", err)
	}

	if _, err := CreateLease(testDB, int(taskID), "agent-complete", "exclusive"); err != nil {
		t.Fatalf("failed to create lease: %v", err)
	}
	run, err := CreateRun(testDB, int(taskID), "agent-complete")
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	payload := map[string]interface{}{
		"message": "All done",
		"result": map[string]interface{}{
			"changes_made":  []string{"Updated task completion"},
			"files_touched": []string{"main.go"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	task, ok := resp["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task object, got %T", resp["task"])
	}
	if task["status_v1"] != string(StatusComplete) {
		t.Fatalf("expected task.status_v1 %s, got %v", StatusComplete, task["status_v1"])
	}
	if task["status_v2"] != string(TaskStatusSucceeded) {
		t.Fatalf("expected task.status_v2 %s, got %v", TaskStatusSucceeded, task["status_v2"])
	}
	if task["output"] != "All done" {
		t.Fatalf("expected task.output 'All done', got %v", task["output"])
	}
	assertTaskUpdatedAtPresent(t, task)

	var updatedAt string
	if err := testDB.QueryRow("SELECT updated_at FROM tasks WHERE id = ?", taskID).Scan(&updatedAt); err != nil {
		t.Fatalf("failed to read updated_at: %v", err)
	}
	if updatedAt == "" {
		t.Fatal("expected updated_at to be set after completion")
	}
	if updatedAt == oldUpdatedAt {
		t.Fatalf("expected updated_at to change after completion, still %s", updatedAt)
	}

	releasedLease, err := GetLeaseByTaskID(testDB, int(taskID))
	if err != nil {
		t.Fatalf("failed to lookup lease: %v", err)
	}
	if releasedLease != nil {
		t.Fatalf("expected lease to be released, got %+v", releasedLease)
	}

	var status string
	var finishedAt sql.NullString
	err = testDB.QueryRow("SELECT status, finished_at FROM runs WHERE id = ?", run.ID).Scan(&status, &finishedAt)
	if err != nil {
		t.Fatalf("failed to query run: %v", err)
	}
	if status != string(RunStatusSucceeded) {
		t.Fatalf("expected run status SUCCEEDED, got %s", status)
	}
	if !finishedAt.Valid {
		t.Fatal("expected finished_at to be set")
	}

	events, err := GetEventsByProjectID(testDB, "proj_default", 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}
	completedEvent := findEventByKind(events, "task.completed")
	if completedEvent == nil {
		t.Fatal("expected task.completed event")
	}
	if completedEvent.EntityID != fmt.Sprintf("%d", taskID) {
		t.Fatalf("expected task.completed entity_id %d, got %s", taskID, completedEvent.EntityID)
	}
}

func TestV2TaskCompleteNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := map[string]interface{}{
		"message": "Missing task",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks/999999/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2TaskCompleteMethodNotAllowed(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Wrong method", StatusInProgress, TaskStatusRunning, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
}

func TestV2TaskCompletePolicyBlock(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Block completion", nil, []PolicyRule{{
		"type":   "completion.block",
		"reason": "Needs approval",
	}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Blocked completion", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "FORBIDDEN")
	details, ok := errField["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error.details object, got %T", errField["details"])
	}
	if details["reason"] != "Needs approval" {
		t.Fatalf("expected reason 'Needs approval', got %v", details["reason"])
	}

	var status string
	var statusV2 sql.NullString
	var output sql.NullString
	if err := testDB.QueryRow("SELECT status, status_v2, output FROM tasks WHERE id = ?", taskID).Scan(&status, &statusV2, &output); err != nil {
		t.Fatalf("failed to query task: %v", err)
	}
	if status != string(StatusInProgress) {
		t.Fatalf("expected status IN_PROGRESS, got %s", status)
	}
	if !statusV2.Valid || statusV2.String != string(TaskStatusRunning) {
		t.Fatalf("expected status_v2 RUNNING, got %v", statusV2.String)
	}
	if output.Valid {
		t.Fatalf("expected output to be empty, got %v", output.String)
	}
}

func TestV2TaskCompletePolicyAllow(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(testDB, "proj_default", "Disabled completion block", nil, []PolicyRule{{
		"type":   "completion.block",
		"reason": "Not enforced",
	}}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Allowed completion", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2TaskCompleteNextTasks(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Complete with followups", StatusInProgress, TaskStatusRunning, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
		"result": map[string]interface{}{
			"summary":       "Completion summary",
			"changes_made":  []string{"Added structured validation"},
			"files_touched": []string{"main.go"},
			"commands_run":  []string{"go test ./..."},
			"tests_run":     []string{"go test ./..."},
			"risks":         []string{"None"},
			"next_tasks": []map[string]interface{}{
				{
					"title":        "Write docs",
					"instructions": "Document the new completion flow",
					"type":         "DOC",
					"priority":     7,
					"tags":         []string{"docs", "api"},
				},
				{
					"title":        "Add more tests",
					"instructions": "Add more tests",
					"type":         "TEST",
					"priority":     3,
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", parentID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	rows, err := testDB.Query(
		"SELECT id, instructions, title, type, priority, tags_json, parent_task_id, project_id FROM tasks WHERE parent_task_id = ? ORDER BY id",
		parentID,
	)
	if err != nil {
		t.Fatalf("failed to query child tasks: %v", err)
	}
	defer rows.Close()

	type childRow struct {
		id          int
		instructions string
		title       sql.NullString
		typeStr     sql.NullString
		priority    int
		tagsJSON    sql.NullString
		parentID    int
		projectID   string
	}
	children := make([]childRow, 0)
	for rows.Next() {
		var child childRow
		if err := rows.Scan(&child.id, &child.instructions, &child.title, &child.typeStr, &child.priority, &child.tagsJSON, &child.parentID, &child.projectID); err != nil {
			t.Fatalf("failed to scan child task: %v", err)
		}
		children = append(children, child)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate child tasks: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 child tasks, got %d", len(children))
	}

	if children[0].instructions != "Document the new completion flow" {
		t.Fatalf("expected first child instructions, got %q", children[0].instructions)
	}
	if !children[0].title.Valid || children[0].title.String != "Write docs" {
		t.Fatalf("expected first child title 'Write docs', got %v", children[0].title.String)
	}
	if !children[0].typeStr.Valid || children[0].typeStr.String != string(TaskTypeDoc) {
		t.Fatalf("expected first child type DOC, got %v", children[0].typeStr.String)
	}
	if children[0].priority != 7 {
		t.Fatalf("expected first child priority 7, got %d", children[0].priority)
	}
	if !children[0].tagsJSON.Valid {
		t.Fatal("expected first child tags_json to be set")
	}
	var tags []string
	if err := json.Unmarshal([]byte(children[0].tagsJSON.String), &tags); err != nil {
		t.Fatalf("failed to parse tags_json: %v", err)
	}
	if len(tags) != 2 || tags[0] != "docs" || tags[1] != "api" {
		t.Fatalf("expected first child tags [docs api], got %v", tags)
	}
	if children[0].parentID != int(parentID) || children[0].projectID != "proj_default" {
		t.Fatalf("expected first child parent/project to match, got parent=%d project=%s", children[0].parentID, children[0].projectID)
	}

	if children[1].instructions != "Add more tests" {
		t.Fatalf("expected second child instructions, got %q", children[1].instructions)
	}
	if !children[1].title.Valid || children[1].title.String != "Add more tests" {
		t.Fatalf("expected second child title 'Add more tests', got %v", children[1].title.String)
	}
	if !children[1].typeStr.Valid || children[1].typeStr.String != string(TaskTypeTest) {
		t.Fatalf("expected second child type TEST, got %v", children[1].typeStr.String)
	}
	if children[1].priority != 3 {
		t.Fatalf("expected second child priority 3, got %d", children[1].priority)
	}
	if children[1].tagsJSON.Valid {
		t.Fatalf("expected second child tags_json to be empty, got %v", children[1].tagsJSON.String)
	}

	events, err := GetEventsByProjectID(testDB, "proj_default", 10)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}
	created := map[string]struct{}{}
	for _, event := range events {
		if event.Kind == "task.created" {
			created[event.EntityID] = struct{}{}
		}
	}
	if _, ok := created[fmt.Sprintf("%d", children[0].id)]; !ok {
		t.Fatalf("expected task.created event for child %d", children[0].id)
	}
	if _, ok := created[fmt.Sprintf("%d", children[1].id)]; !ok {
		t.Fatalf("expected task.created event for child %d", children[1].id)
	}
}

func TestV2TaskCompleteResultMissingFields(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Complete with missing result fields", StatusInProgress, TaskStatusRunning, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2MissingFields(t, w, "result.changes_made", "result.files_touched")
}

func TestV2TaskCompleteResultMissingTestsRunForTestType(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete tests", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeTest),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
		"result": map[string]interface{}{
			"summary": "Test run executed",
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2MissingFields(t, w, "result.tests_run")
}

func TestV2TaskCompleteResultMissingSummaryForDocType(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete docs", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeDoc),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
		"result": map[string]interface{}{
			"changes_made": []string{"Updated README"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2MissingFields(t, w, "result.summary")
}

func TestV2TaskCompleteResultMissingRisksForReviewType(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete review", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeReview),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
		"result": map[string]interface{}{
			"summary": "Reviewed implementation",
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2MissingFields(t, w, "result.risks")
}

func TestV2TaskCompleteNextTasksMissingRequiredFields(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id) VALUES (?, ?, ?, ?)",
		"Complete with invalid next tasks", StatusInProgress, TaskStatusRunning, "proj_default",
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	taskID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Done",
		"result": map[string]interface{}{
			"summary":       "Completion summary",
			"changes_made":  []string{"Added structured validation"},
			"files_touched": []string{"main.go"},
			"commands_run":  []string{"go test ./..."},
			"tests_run":     []string{"go test ./..."},
			"risks":         []string{"None"},
			"next_tasks": []map[string]interface{}{
				{
					"instructions": "Missing title and required fields",
					"type":         "DOC",
					"priority":     2,
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2TaskCompleteAutomationRules(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	defer setAutomationRules(nil)

	completionReviewTitle := "Review task ${task_id}"
	completionReviewType := "REVIEW"
	completionReviewPriority := 5
	completionOutputTitle := "Output summary"

	rules := []automationRule{
		{
			Name:    "Auto followups",
			Trigger: "task.completed",
			Actions: []automationAction{
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Title:        &completionReviewTitle,
						Instructions: "Review completion for task ${task_id}",
						Type:         &completionReviewType,
						Priority:     &completionReviewPriority,
						Tags:         []string{"auto", "needs_human"},
					},
				},
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Title:        &completionOutputTitle,
						Instructions: "Summarize output: ${task_output}",
						Tags:         []string{"auto"},
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete with automation", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Automation output",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", parentID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	rows, err := testDB.Query(
		"SELECT instructions, title, type, priority, tags_json, parent_task_id FROM tasks WHERE parent_task_id = ? ORDER BY id",
		parentID,
	)
	if err != nil {
		t.Fatalf("failed to query automation tasks: %v", err)
	}
	defer rows.Close()

	type autoRow struct {
		instructions string
		title       sql.NullString
		typeStr     sql.NullString
		priority    int
		tagsJSON    sql.NullString
		parentID    int
	}
	created := make([]autoRow, 0)
	for rows.Next() {
		var row autoRow
		if err := rows.Scan(&row.instructions, &row.title, &row.typeStr, &row.priority, &row.tagsJSON, &row.parentID); err != nil {
			t.Fatalf("failed to scan automation task: %v", err)
		}
		created = append(created, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate automation tasks: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("expected 2 automation tasks, got %d", len(created))
	}

	byInstruction := map[string]autoRow{}
	for _, row := range created {
		byInstruction[row.instructions] = row
	}

	reviewInstruction := fmt.Sprintf("Review completion for task %d", parentID)
	outputInstruction := "Summarize output: Automation output"

	reviewRow, ok := byInstruction[reviewInstruction]
	if !ok {
		t.Fatalf("expected automation instruction %q", reviewInstruction)
	}
	if reviewRow.parentID != int(parentID) {
		t.Fatalf("expected review task parent_id %d, got %d", parentID, reviewRow.parentID)
	}
	if !reviewRow.title.Valid || reviewRow.title.String != fmt.Sprintf("Review task %d", parentID) {
		t.Fatalf("expected review title to include task id, got %v", reviewRow.title.String)
	}
	if !reviewRow.typeStr.Valid || reviewRow.typeStr.String != completionReviewType {
		t.Fatalf("expected review type %s, got %v", completionReviewType, reviewRow.typeStr.String)
	}
	if reviewRow.priority != completionReviewPriority {
		t.Fatalf("expected review priority %d, got %d", completionReviewPriority, reviewRow.priority)
	}
	if !reviewRow.tagsJSON.Valid {
		t.Fatal("expected review tags_json to be set")
	}
	var reviewTags []string
	if err := json.Unmarshal([]byte(reviewRow.tagsJSON.String), &reviewTags); err != nil {
		t.Fatalf("failed to parse review tags_json: %v", err)
	}
	if len(reviewTags) != 2 || reviewTags[0] != "auto" || reviewTags[1] != "needs_human" {
		t.Fatalf("expected review tags [auto needs_human], got %v", reviewTags)
	}

	outputRow, ok := byInstruction[outputInstruction]
	if !ok {
		t.Fatalf("expected automation instruction %q", outputInstruction)
	}
	if outputRow.parentID != int(parentID) {
		t.Fatalf("expected output task parent_id %d, got %d", parentID, outputRow.parentID)
	}
	if !outputRow.title.Valid || outputRow.title.String != completionOutputTitle {
		t.Fatalf("expected output title %q, got %v", completionOutputTitle, outputRow.title.String)
	}
}

func TestV2TaskCompleteAutomationRulesPolicyAllow(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	defer setAutomationRules(nil)

	rules := []automationRule{
		{
			Name:    "Auto followups",
			Trigger: "task.completed",
			Actions: []automationAction{
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Instructions: "Follow up for ${task_id}",
					},
				},
			},
		},
	}

	_, err := CreatePolicy(testDB, "proj_default", "Allow audit", nil, []PolicyRule{}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete with policy allow", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Automation output",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", parentID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks WHERE parent_task_id = ?", parentID).Scan(&count); err != nil {
		t.Fatalf("failed to count automation tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 automation task, got %d", count)
	}
}

func TestV2TaskCompleteAutomationRulesPolicyBlock(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	defer setAutomationRules(nil)

	rules := []automationRule{
		{
			Name:    "Auto followups",
			Trigger: "task.completed",
			Actions: []automationAction{
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Instructions: "Follow up for ${task_id}",
					},
				},
			},
		},
	}

	_, err := CreatePolicy(testDB, "proj_default", "Block automation", nil, []PolicyRule{{
		"type":   "automation.block",
		"reason": "No automated followups",
	}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete with policy block", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Automation output",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", parentID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks WHERE parent_task_id = ?", parentID).Scan(&count); err != nil {
		t.Fatalf("failed to count automation tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 automation tasks, got %d", count)
	}
}

func TestV2TaskCompleteAutomationRulesNoMatch(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	defer setAutomationRules(nil)

	rules := []automationRule{
		{
			Name:    "No match trigger",
			Trigger: "task.created",
			Actions: []automationAction{
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Instructions: "Should not run",
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete without automation", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "No automation",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", parentID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var count int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM tasks WHERE parent_task_id = ?", parentID).Scan(&count); err != nil {
		t.Fatalf("failed to count automation tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 automation tasks, got %d", count)
	}
}

func TestV2TaskCompleteAutomationRulesInvalidTemplateFields(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	defer setAutomationRules(nil)

	badTitle := "Title ${unknown_field}"
	rules := []automationRule{
		{
			Name:    "Invalid templates",
			Trigger: "task.completed",
			Actions: []automationAction{
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Title:        &badTitle,
						Instructions: "Follow up ${missing} for ${task_id}",
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete with template gaps", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Template data",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", parentID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var instructions string
	var title sql.NullString
	if err := testDB.QueryRow(
		"SELECT instructions, title FROM tasks WHERE parent_task_id = ? ORDER BY id LIMIT 1",
		parentID,
	).Scan(&instructions, &title); err != nil {
		t.Fatalf("failed to query automation task: %v", err)
	}

	expectedInstructions := fmt.Sprintf("Follow up ${missing} for %d", parentID)
	if instructions != expectedInstructions {
		t.Fatalf("expected instructions %q, got %q", expectedInstructions, instructions)
	}
	if !title.Valid || title.String != badTitle {
		t.Fatalf("expected title %q, got %v", badTitle, title.String)
	}
}

func TestV2TaskCompleteAutomationRulesMultipleFollowups(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	defer setAutomationRules(nil)

	rules := []automationRule{
		{
			Name:    "Multiple followups",
			Trigger: "task.completed",
			Actions: []automationAction{
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Instructions: "Followup A for ${task_id}",
					},
				},
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Instructions: "Followup B for ${task_id}",
					},
				},
				{
					Type: "create_task",
					Task: automationTaskSpec{
						Instructions: "Followup C for ${task_id}",
					},
				},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})

	result, err := testDB.Exec(
		"INSERT INTO tasks (instructions, status, status_v2, project_id, type) VALUES (?, ?, ?, ?, ?)",
		"Complete with many followups", StatusInProgress, TaskStatusRunning, "proj_default", string(TaskTypeAnalyze),
	)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to read task id: %v", err)
	}

	payload := map[string]interface{}{
		"message": "Multiple followups",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", parentID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	rows, err := testDB.Query(
		"SELECT instructions FROM tasks WHERE parent_task_id = ? ORDER BY id",
		parentID,
	)
	if err != nil {
		t.Fatalf("failed to query automation tasks: %v", err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var instructions string
		if err := rows.Scan(&instructions); err != nil {
			t.Fatalf("failed to scan automation task: %v", err)
		}
		got = append(got, instructions)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate automation tasks: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 automation tasks, got %d", len(got))
	}

	expected := map[string]struct{}{
		fmt.Sprintf("Followup A for %d", parentID): {},
		fmt.Sprintf("Followup B for %d", parentID): {},
		fmt.Sprintf("Followup C for %d", parentID): {},
	}
	for _, instructions := range got {
		if _, ok := expected[instructions]; !ok {
			t.Fatalf("unexpected automation instruction %q", instructions)
		}
	}
}
