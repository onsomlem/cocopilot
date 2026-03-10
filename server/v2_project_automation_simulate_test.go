package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2ProjectAutomationSimulateSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Automation Sim", "/tmp/automation-sim", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	title := "Parent task"
	parentType := TaskTypeAnalyze
	parentPriority := 7
	parentTags := []string{"parent"}
	parentTask, err := CreateTaskV2WithMeta(testDB, "Finish report", project.ID, nil, &title, &parentType, &parentPriority, parentTags, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2WithMeta failed: %v", err)
	}

	enabled := true
	actionTitle := "Review ${task_title}"
	actionType := "REVIEW"
	actionPriority := 5
	rules := []automationRule{ {
		Name:    "Followup",
		Enabled: &enabled,
		Trigger: "task.completed",
		Actions: []automationAction{{
			Type: "create_task",
			Task: automationTaskSpec{
				Title:        &actionTitle,
				Instructions: "Follow up ${task_id}",
				Type:         &actionType,
				Priority:     &actionPriority,
				Tags:         []string{"auto", "review"},
			},
		}},
	} }

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})
	defer setAutomationRules(nil)

	payload := fmt.Sprintf(`{"event":{"kind":"task.completed","entity_id":"%d","payload":{"task_id":%d}}}`, parentTask.ID, parentTask.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/automation/simulate", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Actions []struct {
			RuleName string `json:"rule_name"`
			Type     string `json:"type"`
			Task     struct {
				ProjectID    string   `json:"project_id"`
				ParentTaskID *int     `json:"parent_task_id"`
				Title        *string  `json:"title"`
				Instructions string   `json:"instructions"`
				Type         string   `json:"type"`
				Priority     int      `json:"priority"`
				Tags         []string `json:"tags"`
			} `json:"task"`
		} `json:"actions"`
		Tasks []struct {
			ProjectID    string   `json:"project_id"`
			ParentTaskID *int     `json:"parent_task_id"`
			Title        *string  `json:"title"`
			Instructions string   `json:"instructions"`
			Type         string   `json:"type"`
			Priority     int      `json:"priority"`
			Tags         []string `json:"tags"`
		} `json:"tasks_that_would_be_created"`
	}

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(resp.Actions))
	}
	if len(resp.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
	}

	actionTask := resp.Actions[0].Task
	if actionTask.ProjectID != project.ID {
		t.Fatalf("expected project_id %s, got %s", project.ID, actionTask.ProjectID)
	}
	if actionTask.ParentTaskID == nil || *actionTask.ParentTaskID != parentTask.ID {
		t.Fatalf("expected parent_task_id %d, got %v", parentTask.ID, actionTask.ParentTaskID)
	}
	if actionTask.Instructions != fmt.Sprintf("Follow up %d", parentTask.ID) {
		t.Fatalf("unexpected instructions: %s", actionTask.Instructions)
	}
	if actionTask.Title == nil || *actionTask.Title != "Review Parent task" {
		t.Fatalf("unexpected title: %v", actionTask.Title)
	}
	if actionTask.Type != "REVIEW" {
		t.Fatalf("expected type REVIEW, got %s", actionTask.Type)
	}
	if actionTask.Priority != 5 {
		t.Fatalf("expected priority 5, got %d", actionTask.Priority)
	}
	if len(actionTask.Tags) != 2 || actionTask.Tags[0] != "auto" || actionTask.Tags[1] != "review" {
		t.Fatalf("unexpected tags: %v", actionTask.Tags)
	}
}

func TestV2ProjectAutomationSimulateUnsupportedKind(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Automation Sim Kind", "/tmp/automation-sim-kind", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/automation/simulate", strings.NewReader(`{"event":{"kind":"run.completed"}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}
