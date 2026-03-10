package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV2ProjectAutomationReplaySuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	defer setAutomationRules(nil)

	project, err := CreateProject(testDB, "Automation Replay", "/tmp/automation-replay", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	parentTask, err := CreateTaskV2WithMeta(testDB, "Finish report", project.ID, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2WithMeta failed: %v", err)
	}

	instruction := "Review ${task_id}"
	rules := []automationRule{{
		Name:    "Followup",
		Trigger: "task.completed",
		Actions: []automationAction{{
			Type: "create_task",
			Task: automationTaskSpec{
				Instructions: instruction,
			},
		}},
	}}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{AutomationRules: rules})

	idValue := fmt.Sprintf("%d", parentTask.ID)
	insertTestEvent(t, testDB, "evt_auto_1", project.ID, "task.completed", "task", idValue, "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_auto_2", project.ID, "task.updated", "task", idValue, "2026-02-11T10:01:00.000000Z")
	insertTestEvent(t, testDB, "evt_auto_3", project.ID, "task.completed", "task", idValue, "2026-02-11T10:02:00.000000Z")

	path := fmt.Sprintf("/api/v2/projects/%s/automation/replay?since_event_id=evt_auto_1", project.ID)
	req := httptest.NewRequest(http.MethodPost, path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if replayed, ok := resp["events_replayed"].(float64); !ok || int(replayed) != 3 {
		t.Fatalf("expected events_replayed 3, got %v", resp["events_replayed"])
	}
	if matched, ok := resp["task_completed_events"].(float64); !ok || int(matched) != 2 {
		t.Fatalf("expected task_completed_events 2, got %v", resp["task_completed_events"])
	}

	rows, err := testDB.Query("SELECT instructions FROM tasks WHERE parent_task_id = ? ORDER BY id", parentTask.ID)
	if err != nil {
		t.Fatalf("failed to query replayed tasks: %v", err)
	}
	defer rows.Close()

	created := 0
	for rows.Next() {
		created++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate replayed tasks: %v", err)
	}
	if created != 2 {
		t.Fatalf("expected 2 automation tasks, got %d", created)
	}
}

func TestV2ProjectAutomationReplayMissingSinceEventID(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/automation/replay", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectAutomationReplaySinceEventWrongProject(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "Automation Replay", "/tmp/automation-replay-wrong", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	insertTestEvent(t, testDB, "evt_other_project", "proj_default", "task.completed", "task", "101", "2026-02-11T10:00:00.000000Z")

	path := fmt.Sprintf("/api/v2/projects/%s/automation/replay?since_event_id=evt_other_project", project.ID)
	req := httptest.NewRequest(http.MethodPost, path, nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}
