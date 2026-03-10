package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestContract_InstructionsEndpointsExist verifies that every endpoint
// referenced in the /instructions page is actually registered and reachable.
func TestContract_InstructionsEndpointsExist(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Create default project so claim-next can resolve
	_, err := CreateProject(testDB, "Default", ".", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Get the default project ID
	var projID string
	_ = testDB.QueryRow("SELECT id FROM projects LIMIT 1").Scan(&projID)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		// From instructions:
		// 1. Register agent
		{http.MethodPost, "/api/v2/agents", `{"id":"contract-agent","name":"Contract","capabilities":["test"]}`},
		// 2. Claim-next (project scoped)
		{http.MethodPost, fmt.Sprintf("/api/v2/projects/%s/tasks/claim-next", projID), `{"agent_id":"contract-agent"}`},
		// 3. Run steps (need a run ID — just verify route exists with dummy)
		{http.MethodPost, "/api/v2/runs/run_nonexistent/steps", `{"name":"step1","status":"running"}`},
		// 4. Lease heartbeat
		{http.MethodPost, "/api/v2/leases/lease_nonexistent/heartbeat", ""},
		// 5. Complete task
		{http.MethodPost, "/api/v2/tasks/999/complete", `{"output":"done","result":{"summary":"ok","changes_made":[],"files_touched":[]}}`},
		// 6. Fail task
		{http.MethodPost, "/api/v2/tasks/999/fail", `{"error":"test failure"}`},
		// 7. List tasks
		{http.MethodGet, "/api/v2/tasks", ""},
		// Also verify instructions itself
		{http.MethodGet, "/instructions", ""},
		{http.MethodGet, "/instructions-detailed", ""},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var body *strings.Reader
			if ep.body != "" {
				body = strings.NewReader(ep.body)
			} else {
				body = strings.NewReader("")
			}
			req := httptest.NewRequest(ep.method, ep.path, body)
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			// We expect anything other than 404 (route not found) or 405 (method not allowed).
			// Some endpoints will return 404 for nonexistent entities or 409 for conflicts,
			// which is fine — the route IS registered.
			if rr.Code == http.StatusNotFound {
				// Check if it's a "route not found" vs "entity not found"
				var errResp map[string]interface{}
				if json.Unmarshal(rr.Body.Bytes(), &errResp) == nil {
					if errObj, ok := errResp["error"].(map[string]interface{}); ok {
						code, _ := errObj["code"].(string)
						// Entity not found is OK — route exists
						if code == "NOT_FOUND" || code == "TASK_NOT_FOUND" || code == "LEASE_NOT_FOUND" || code == "RUN_NOT_FOUND" {
							return
						}
					}
				}
				t.Errorf("endpoint %s %s returned raw 404 — route not registered", ep.method, ep.path)
			}
			if rr.Code == http.StatusMethodNotAllowed {
				t.Errorf("endpoint %s %s returned 405 — method not supported", ep.method, ep.path)
			}
		})
	}
}

// TestContract_ClaimResponseShape verifies the claim-next response
// includes the fields the worker/agent expects.
func TestContract_ClaimResponseShape(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Setup: project + task
	proj, err := CreateProject(testDB, "contract-proj", ".", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	_, err = CreateTaskV2(testDB, "Contract test task", proj.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Register agent
	agentBody := `{"id":"contract-agent","name":"Contract Agent","capabilities":["test"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/agents", strings.NewReader(agentBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register agent: %d %s", rr.Code, rr.Body.String())
	}

	// Claim
	claimBody := `{"agent_id":"contract-agent"}`
	req = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v2/projects/%s/tasks/claim-next", proj.ID),
		strings.NewReader(claimBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("claim: %d %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode claim response: %v", err)
	}

	// Verify expected top-level keys per instructions
	requiredKeys := []string{"task", "lease", "run", "context"}
	for _, key := range requiredKeys {
		if resp[key] == nil {
			t.Errorf("claim response missing required key %q", key)
		}
	}

	// Verify task has expected fields
	task, ok := resp["task"].(map[string]interface{})
	if !ok {
		t.Fatal("claim response 'task' is not an object")
	}
	taskFields := []string{"id", "instructions"}
	for _, f := range taskFields {
		if task[f] == nil {
			t.Errorf("task missing field %q", f)
		}
	}

	// Verify lease has expected fields
	lease, ok := resp["lease"].(map[string]interface{})
	if !ok {
		t.Fatal("claim response 'lease' is not an object")
	}
	if lease["id"] == nil {
		t.Error("lease missing 'id'")
	}
	if lease["expires_at"] == nil {
		t.Error("lease missing 'expires_at'")
	}

	// Verify run has expected fields
	run, ok := resp["run"].(map[string]interface{})
	if !ok {
		t.Fatal("claim response 'run' is not an object")
	}
	if run["id"] == nil {
		t.Error("run missing 'id'")
	}
}
