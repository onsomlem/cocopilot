package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// TestGoldenPath_HTTPFullLifecycle exercises the full agent lifecycle through
// the HTTP API layer: create project → create task → register agent → claim
// task → add run step → complete task → verify events, runs, agents, memory.
func TestGoldenPath_HTTPFullLifecycle(t *testing.T) {
	_, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// ── 1. Create Project ──────────────────────────────────────────────
	projBody := `{"name":"gp-http-test","workdir":"/tmp/gp-http"}`
	resp := doJSON(t, mux, http.MethodPost, "/api/v2/projects", projBody)
	requireStatus(t, resp, http.StatusCreated, "create project")
	projResp := decodeJSON(t, resp)
	project := objField(t, projResp, "project")
	projectID := strField(t, project, "id")
	if projectID == "" {
		t.Fatal("project id is empty")
	}

	// ── 2. Create Task via v2 API ──────────────────────────────────────
	taskBody := fmt.Sprintf(`{
		"instructions":"Golden path HTTP smoke test task",
		"project_id":"%s",
		"title":"GP smoke task",
		"type":"MODIFY",
		"priority":5
	}`, projectID)
	resp = doJSON(t, mux, http.MethodPost, "/api/v2/tasks", taskBody)
	requireStatus(t, resp, http.StatusCreated, "create task")
	taskResp := decodeJSON(t, resp)
	task := objField(t, taskResp, "task")
	taskID := int(numField(t, task, "id"))
	if taskID == 0 {
		t.Fatal("task id is 0")
	}

	// ── 3. Register Agent ──────────────────────────────────────────────
	agentBody := `{
		"name":"gp-smoke-agent",
		"capabilities":["go","sqlite"],
		"metadata":{"version":"1.0"}
	}`
	resp = doJSON(t, mux, http.MethodPost, "/api/v2/agents", agentBody)
	requireStatus(t, resp, http.StatusCreated, "register agent")
	agentResp := decodeJSON(t, resp)
	agent := objField(t, agentResp, "agent")
	agentID := strField(t, agent, "id")
	if agentID == "" {
		t.Fatal("agent id is empty")
	}

	// ── 4. List Agents → verify our agent appears ──────────────────────
	resp = doJSON(t, mux, http.MethodGet, "/api/v2/agents", "")
	requireStatus(t, resp, http.StatusOK, "list agents")
	agentsListResp := decodeJSON(t, resp)
	agents := arrField(t, agentsListResp, "agents")
	if len(agents) < 1 {
		t.Fatal("expected at least 1 agent in list")
	}
	found := false
	for _, a := range agents {
		m, ok := a.(map[string]interface{})
		if ok && m["id"] == agentID {
			found = true
		}
	}
	if !found {
		t.Errorf("agent %s not found in list response", agentID)
	}

	// ── 5. Claim Task ──────────────────────────────────────────────────
	claimBody := fmt.Sprintf(`{"agent_id":"%s","mode":"exclusive"}`, agentID)
	resp = doJSON(t, mux, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), claimBody)
	requireStatus(t, resp, http.StatusOK, "claim task")
	claimResp := decodeJSON(t, resp)

	// Verify lease exists in response.
	lease := objField(t, claimResp, "lease")
	leaseID := strField(t, lease, "id")
	if leaseID == "" {
		t.Fatal("lease id is empty after claim")
	}

	// Verify run was created.
	run := objField(t, claimResp, "run")
	runID := strField(t, run, "id")
	if runID == "" {
		t.Fatal("run id is empty after claim")
	}

	// ── 6. Add Run Step ────────────────────────────────────────────────
	stepBody := `{"name":"edit-main","type":"action","description":"Editing main.go","status":"SUCCEEDED"}`
	resp = doJSON(t, mux, http.MethodPost, fmt.Sprintf("/api/v2/runs/%s/steps", runID), stepBody)
	requireStatus(t, resp, http.StatusCreated, "create run step")

	// ── 7. Add Run Log ─────────────────────────────────────────────────
	logBody := `{"stream":"stdout","chunk":"Applied patch to main.go\n"}`
	resp = doJSON(t, mux, http.MethodPost, fmt.Sprintf("/api/v2/runs/%s/logs", runID), logBody)
	requireStatus(t, resp, http.StatusNoContent, "create run log")

	// ── 8. Complete Task ───────────────────────────────────────────────
	completeBody := `{
		"output":"Golden path complete",
		"result":{
			"summary":"All changes applied",
			"changes_made":["main.go"],
			"files_touched":["main.go"]
		}
	}`
	resp = doJSON(t, mux, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), completeBody)
	requireStatus(t, resp, http.StatusOK, "complete task")

	// ── 9. Verify Task status is SUCCEEDED ─────────────────────────────
	resp = doJSON(t, mux, http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d", taskID), "")
	requireStatus(t, resp, http.StatusOK, "get task detail")
	taskDetail := decodeJSON(t, resp)
	tdTask := objField(t, taskDetail, "task")
	if status := strField(t, tdTask, "status_v2"); status != "SUCCEEDED" {
		t.Errorf("task status expected SUCCEEDED, got %s", status)
	}

	// ── 10. Verify Run detail has steps and logs ───────────────────────
	resp = doJSON(t, mux, http.MethodGet, fmt.Sprintf("/api/v2/runs/%s", runID), "")
	requireStatus(t, resp, http.StatusOK, "get run detail")
	runDetail := decodeJSON(t, resp)
	rdRun := objField(t, runDetail, "run")
	if rdRun["status"] != "SUCCEEDED" {
		t.Errorf("run status expected SUCCEEDED, got %v", rdRun["status"])
	}
	steps := arrField(t, rdRun, "steps")
	if len(steps) < 1 {
		t.Error("expected at least 1 run step")
	}
	logs := arrField(t, rdRun, "logs")
	if len(logs) < 1 {
		t.Error("expected at least 1 run log")
	}

	// ── 11. Verify Events contain expected lifecycle events ────────────
	resp = doJSON(t, mux, http.MethodGet,
		fmt.Sprintf("/api/v2/events?project_id=%s&limit=50", projectID), "")
	requireStatus(t, resp, http.StatusOK, "list events")
	evResp := decodeJSON(t, resp)
	events := arrField(t, evResp, "events")

	expectedKinds := map[string]bool{
		"task.claimed":   false,
		"task.completed": false,
	}
	for _, e := range events {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if kind, ok := em["kind"].(string); ok {
			if _, want := expectedKinds[kind]; want {
				expectedKinds[kind] = true
			}
		}
	}
	for kind, seen := range expectedKinds {
		if !seen {
			t.Errorf("expected event kind %q not found in events", kind)
		}
	}

	// ── 12. Write and Read Memory ──────────────────────────────────────
	memBody := `{"scope":"project","key":"smoke-test-key","value":{"data":"smoke-test-value"}}`
	resp = doJSON(t, mux, http.MethodPut,
		fmt.Sprintf("/api/v2/projects/%s/memory", projectID), memBody)
	requireStatus(t, resp, http.StatusOK, "put memory")

	resp = doJSON(t, mux, http.MethodGet,
		fmt.Sprintf("/api/v2/projects/%s/memory?scope=project&key=smoke-test-key", projectID), "")
	requireStatus(t, resp, http.StatusOK, "get memory")
	memResp := decodeJSON(t, resp)
	items := arrField(t, memResp, "items")
	if len(items) < 1 {
		t.Error("expected at least 1 memory item")
	}

	// ── 13. List Runs global endpoint ──────────────────────────────────
	resp = doJSON(t, mux, http.MethodGet, "/api/v2/runs", "")
	requireStatus(t, resp, http.StatusOK, "list runs")
	runsResp := decodeJSON(t, resp)
	runs := arrField(t, runsResp, "runs")
	if len(runs) < 1 {
		t.Error("expected at least 1 run in global list")
	}

	// ── 14. Verify UI pages render without error ───────────────────────
	uiPages := []string{
		"/dashboard",
		"/board",
		"/agents",
		"/runs",
		"/memory",
		"/events-browser",
		"/health",
		"/context-packs",
		"/dependencies",
	}
	for _, page := range uiPages {
		t.Run("UI"+page, func(t *testing.T) {
			resp := doJSON(t, mux, http.MethodGet, page, "")
			if resp.Code == http.StatusNotFound {
				t.Errorf("UI page %s returned 404", page)
			}
			if resp.Code >= 500 {
				t.Errorf("UI page %s returned %d: %s", page, resp.Code, resp.Body.String())
			}
		})
	}
}

// TestGoldenPath_HTTPDependencyChain verifies that dependency ordering works
// through the HTTP API: T2 depends on T1 → T2 can only be claimed after T1 completes.
func TestGoldenPath_HTTPDependencyChain(t *testing.T) {
	_, cleanup := setupLifecycleTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Create project.
	projBody := `{"name":"gp-deps","workdir":"/tmp/gp-deps"}`
	resp := doJSON(t, mux, http.MethodPost, "/api/v2/projects", projBody)
	requireStatus(t, resp, http.StatusCreated, "create project")
	project := objField(t, decodeJSON(t, resp), "project")
	projectID := strField(t, project, "id")

	// Create T1.
	t1Body := fmt.Sprintf(`{"instructions":"T1","project_id":"%s","title":"T1"}`, projectID)
	resp = doJSON(t, mux, http.MethodPost, "/api/v2/tasks", t1Body)
	requireStatus(t, resp, http.StatusCreated, "create T1")
	t1ID := int(numField(t, objField(t, decodeJSON(t, resp), "task"), "id"))

	// Create T2.
	t2Body := fmt.Sprintf(`{"instructions":"T2","project_id":"%s","title":"T2"}`, projectID)
	resp = doJSON(t, mux, http.MethodPost, "/api/v2/tasks", t2Body)
	requireStatus(t, resp, http.StatusCreated, "create T2")
	t2ID := int(numField(t, objField(t, decodeJSON(t, resp), "task"), "id"))

	// Add dependency: T2 depends on T1.
	depBody := fmt.Sprintf(`{"depends_on_task_id":%d}`, t1ID)
	resp = doJSON(t, mux, http.MethodPost,
		fmt.Sprintf("/api/v2/tasks/%d/dependencies", t2ID), depBody)
	requireStatus(t, resp, http.StatusCreated, "add dependency")

	// Claim T1 → should succeed.
	resp = doJSON(t, mux, http.MethodPost,
		fmt.Sprintf("/api/v2/tasks/%d/claim", t1ID),
		`{"agent_id":"dep-agent","mode":"exclusive"}`)
	requireStatus(t, resp, http.StatusOK, "claim T1")

	// Complete T1.
	resp = doJSON(t, mux, http.MethodPost,
		fmt.Sprintf("/api/v2/tasks/%d/complete", t1ID),
		`{"output":"T1 done","result":{"summary":"ok","changes_made":["a.go"],"files_touched":["a.go"]}}`)
	requireStatus(t, resp, http.StatusOK, "complete T1")

	// Now claim T2 → should succeed (T1 is done).
	resp = doJSON(t, mux, http.MethodPost,
		fmt.Sprintf("/api/v2/tasks/%d/claim", t2ID),
		`{"agent_id":"dep-agent","mode":"exclusive"}`)
	requireStatus(t, resp, http.StatusOK, "claim T2 after T1 done")

	// Complete T2.
	resp = doJSON(t, mux, http.MethodPost,
		fmt.Sprintf("/api/v2/tasks/%d/complete", t2ID),
		`{"output":"T2 done","result":{"summary":"ok","changes_made":["b.go"],"files_touched":["b.go"]}}`)
	requireStatus(t, resp, http.StatusOK, "complete T2")

	// Verify both tasks SUCCEEDED.
	for _, id := range []int{t1ID, t2ID} {
		resp = doJSON(t, mux, http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d", id), "")
		requireStatus(t, resp, http.StatusOK, fmt.Sprintf("get task %d", id))
		status := strField(t, objField(t, decodeJSON(t, resp), "task"), "status_v2")
		if status != "SUCCEEDED" {
			t.Errorf("task %d expected SUCCEEDED, got %s", id, status)
		}
	}
}

// ── Test helpers ───────────────────────────────────────────────────────────

func doJSON(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func requireStatus(t *testing.T, resp *httptest.ResponseRecorder, want int, label string) {
	t.Helper()
	if resp.Code != want {
		t.Fatalf("%s: expected %d, got %d: %s", label, want, resp.Code, resp.Body.String())
	}
}

func decodeJSON(t *testing.T, resp *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode JSON: %v\nbody: %s", err, resp.Body.String())
	}
	return m
}

func objField(t *testing.T, m map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in %v", key, m)
	}
	obj, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("key %q is not an object: %T", key, v)
	}
	return obj
}

func arrField(t *testing.T, m map[string]interface{}, key string) []interface{} {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in response", key)
	}
	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("key %q is not an array: %T = %v", key, v, v)
	}
	return arr
}

func strField(t *testing.T, m map[string]interface{}, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in %v", key, m)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("key %q is not a string: %T", key, v)
	}
	return s
}

func numField(t *testing.T, m map[string]interface{}, key string) float64 {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in %v", key, m)
	}
	n, ok := v.(float64)
	if !ok {
		// Try string → float conversion for IDs that come as strings.
		if s, ok := v.(string); ok {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				t.Fatalf("key %q is not a number: %T", key, v)
			}
			return f
		}
		t.Fatalf("key %q is not a number: %T", key, v)
	}
	return n
}
