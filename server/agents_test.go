package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupAgentTestDB creates a fresh test database for agent tests
func setupAgentTestDB(t *testing.T) (*sql.DB, func()) {
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_agents_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations to set up all tables
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Replace global db with test db
	oldDB := db
	db = testDB

	cleanup := func() {
		db.Close()
		db = oldDB
		os.Remove(testDBPath)
	}

	return testDB, cleanup
}

// TestAgentRegistration tests agent registration endpoint
func TestAgentRegistration(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	payload := `{
		"name": "Test Agent",
		"capabilities": ["code", "analysis"],
		"metadata": {"version": "1.0.0", "type": "llm"}
	}`

	req := httptest.NewRequest("POST", "/api/v2/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2RegisterAgentHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	agent, ok := response["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("No agent in response")
	}

	if agent["name"] != "Test Agent" {
		t.Errorf("Expected agent name 'Test Agent', got %v", agent["name"])
	}

	if agent["status"] != "ONLINE" {
		t.Errorf("Expected agent status 'ONLINE', got %v", agent["status"])
	}

	// Verify agent was saved to database
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM agents WHERE name = ?", "Test Agent").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 agent in database, got %d", count)
	}
}

// TestAgentRegistrationValidation tests validation in agent registration
func TestAgentRegistrationValidation(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	// Test missing name
	payload := `{"capabilities": ["code"]}`
	req := httptest.NewRequest("POST", "/api/v2/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2RegisterAgentHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing name, got %d", http.StatusBadRequest, w.Code)
	}

	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

// TestAgentListing tests agent listing endpoint
func TestAgentListing(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	// Register a few agents first
	_, err := RegisterAgent(db, "Agent One", []string{"code"}, map[string]interface{}{"type": "llm"})
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	_, err = RegisterAgent(db, "Agent Two", []string{"analysis"}, map[string]interface{}{"type": "assistant"})
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v2/agents", nil)
	w := httptest.NewRecorder()

	v2ListAgentsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	agents, ok := response["agents"].([]interface{})
	if !ok {
		t.Fatal("No agents in response")
	}

	if len(agents) < 2 {
		t.Errorf("Expected at least 2 agents, got %d", len(agents))
	}

	// Check that we have Agent One and Agent Two (plus possibly the default agent)
	names := make(map[string]bool)
	for _, agentInterface := range agents {
		agent := agentInterface.(map[string]interface{})
		names[agent["name"].(string)] = true
	}

	if !names["Agent One"] || !names["Agent Two"] {
		t.Error("Missing expected agents in response")
	}
}

func TestAgentListingPaging(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	agentOne, err := RegisterAgent(db, "Agent One", []string{"code"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	agentTwo, err := RegisterAgent(db, "Agent Two", []string{"analysis"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	agentThree, err := RegisterAgent(db, "Agent Three", []string{"analysis"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	base := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	defaultTime := base.Add(-1 * time.Minute).Format(leaseTimeFormat)
	oneTime := base.Format(leaseTimeFormat)
	twoTime := base.Add(1 * time.Minute).Format(leaseTimeFormat)
	threeTime := base.Add(2 * time.Minute).Format(leaseTimeFormat)

	_, err = db.Exec("UPDATE agents SET registered_at = ? WHERE id = ?", defaultTime, "agent_default")
	if err != nil {
		t.Fatalf("Failed to update default agent timestamp: %v", err)
	}
	_, err = db.Exec("UPDATE agents SET registered_at = ? WHERE id = ?", oneTime, agentOne.ID)
	if err != nil {
		t.Fatalf("Failed to update agent one timestamp: %v", err)
	}
	_, err = db.Exec("UPDATE agents SET registered_at = ? WHERE id = ?", twoTime, agentTwo.ID)
	if err != nil {
		t.Fatalf("Failed to update agent two timestamp: %v", err)
	}
	_, err = db.Exec("UPDATE agents SET registered_at = ? WHERE id = ?", threeTime, agentThree.ID)
	if err != nil {
		t.Fatalf("Failed to update agent three timestamp: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v2/agents?limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	v2ListAgentsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	agents, ok := response["agents"].([]interface{})
	if !ok {
		t.Fatal("No agents in response")
	}
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent, got %d", len(agents))
	}
	if total, ok := response["total"].(float64); !ok || int(total) != 4 {
		t.Fatalf("Expected total 4, got %v", response["total"])
	}

	agent := agents[0].(map[string]interface{})
	if agent["name"] != "Agent One" {
		t.Errorf("Expected Agent One, got %v", agent["name"])
	}
}

func TestAgentListingSorting(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	agentOne, err := RegisterAgent(db, "Agent One", []string{"code"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	agentTwo, err := RegisterAgent(db, "Agent Two", []string{"analysis"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}
	agentThree, err := RegisterAgent(db, "Agent Three", []string{"analysis"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	base := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)

	defaultRegistered := base.Add(-4 * time.Minute).Format(leaseTimeFormat)
	oneRegistered := base.Add(-3 * time.Minute).Format(leaseTimeFormat)
	twoRegistered := base.Add(-2 * time.Minute).Format(leaseTimeFormat)
	threeRegistered := base.Add(-1 * time.Minute).Format(leaseTimeFormat)

	defaultSeen := base.Add(-3 * time.Minute).Format(leaseTimeFormat)
	oneSeen := base.Add(-2 * time.Minute).Format(leaseTimeFormat)
	twoSeen := base.Add(-1 * time.Minute).Format(leaseTimeFormat)
	threeSeen := base.Add(1 * time.Minute).Format(leaseTimeFormat)

	_, err = db.Exec("UPDATE agents SET registered_at = ?, last_seen = ? WHERE id = ?", defaultRegistered, defaultSeen, "agent_default")
	if err != nil {
		t.Fatalf("Failed to update default agent timestamps: %v", err)
	}
	_, err = db.Exec("UPDATE agents SET registered_at = ?, last_seen = ? WHERE id = ?", oneRegistered, oneSeen, agentOne.ID)
	if err != nil {
		t.Fatalf("Failed to update agent one timestamps: %v", err)
	}
	_, err = db.Exec("UPDATE agents SET registered_at = ?, last_seen = ? WHERE id = ?", twoRegistered, twoSeen, agentTwo.ID)
	if err != nil {
		t.Fatalf("Failed to update agent two timestamps: %v", err)
	}
	_, err = db.Exec("UPDATE agents SET registered_at = ?, last_seen = ? WHERE id = ?", threeRegistered, threeSeen, agentThree.ID)
	if err != nil {
		t.Fatalf("Failed to update agent three timestamps: %v", err)
	}

	getNames := func(path string) []string {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		v2ListAgentsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		agents, ok := response["agents"].([]interface{})
		if !ok {
			t.Fatal("No agents in response")
		}

		names := make([]string, 0, len(agents))
		for _, agentInterface := range agents {
			agent := agentInterface.(map[string]interface{})
			names = append(names, agent["name"].(string))
		}
		return names
	}

	namesAsc := getNames("/api/v2/agents?sort=last_seen:asc")
	if len(namesAsc) < 4 {
		t.Fatalf("Expected at least 4 agents, got %d", len(namesAsc))
	}

	expectedAsc := []string{"Default Agent", "Agent One", "Agent Two", "Agent Three"}
	for i, expected := range expectedAsc {
		if namesAsc[i] != expected {
			t.Fatalf("Expected %s at position %d, got %s", expected, i, namesAsc[i])
		}
	}

	namesDesc := getNames("/api/v2/agents?sort=last_seen:desc")
	if len(namesDesc) < 4 {
		t.Fatalf("Expected at least 4 agents, got %d", len(namesDesc))
	}

	expectedDesc := []string{"Agent Three", "Agent Two", "Agent One", "Default Agent"}
	for i, expected := range expectedDesc {
		if namesDesc[i] != expected {
			t.Fatalf("Expected %s at position %d, got %s", expected, i, namesDesc[i])
		}
	}

	namesCreated := getNames("/api/v2/agents?sort=created_at")
	if len(namesCreated) < 4 {
		t.Fatalf("Expected at least 4 agents, got %d", len(namesCreated))
	}

	expectedCreated := []string{"Default Agent", "Agent One", "Agent Two", "Agent Three"}
	for i, expected := range expectedCreated {
		if namesCreated[i] != expected {
			t.Fatalf("Expected %s at position %d, got %s", expected, i, namesCreated[i])
		}
	}
}

func TestAgentListingInvalidPagingParams(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	tests := []string{
		"/api/v2/agents?limit=0",
		"/api/v2/agents?limit=-1",
		"/api/v2/agents?limit=abc",
		"/api/v2/agents?offset=-2",
		"/api/v2/agents?offset=xyz",
	}

	for _, path := range tests {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		v2ListAgentsHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	}
}

func TestAgentListingInvalidSortParam(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	tests := []string{
		"/api/v2/agents?sort=last_seen",
		"/api/v2/agents?sort=created_at:desc",
		"/api/v2/agents?sort=unknown",
	}

	for _, path := range tests {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		v2ListAgentsHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	}
}

func TestAgentListingFilters(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	agentActiveRecent, err := RegisterAgent(db, "Active Recent", []string{"code"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	agentActiveOld, err := RegisterAgent(db, "Active Old", []string{"analysis"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	agentStale, err := RegisterAgent(db, "Stale Agent", []string{"analysis"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	now := time.Now().UTC()
	recentSeen := now.Add(-2 * time.Minute).Format(leaseTimeFormat)
	oldSeen := now.Add(-2 * time.Hour).Format(leaseTimeFormat)
	recentRegistered := now.Add(-1 * time.Minute).Format(leaseTimeFormat)

	_, err = db.Exec("UPDATE agents SET registered_at = ?, last_seen = ? WHERE id = ?", oldSeen, oldSeen, "agent_default")
	if err != nil {
		t.Fatalf("Failed to update default agent timestamps: %v", err)
	}

	_, err = db.Exec("UPDATE agents SET status = ?, last_seen = ?, registered_at = ? WHERE id = ?",
		AgentStatusOnline, recentSeen, recentRegistered, agentActiveRecent.ID)
	if err != nil {
		t.Fatalf("Failed to update active recent agent: %v", err)
	}

	_, err = db.Exec("UPDATE agents SET status = ?, last_seen = ? WHERE id = ?",
		AgentStatusOnline, oldSeen, agentActiveOld.ID)
	if err != nil {
		t.Fatalf("Failed to update active old agent: %v", err)
	}

	_, err = db.Exec("UPDATE agents SET status = ?, last_seen = ? WHERE id = ?",
		AgentStatusOffline, oldSeen, agentStale.ID)
	if err != nil {
		t.Fatalf("Failed to update stale agent: %v", err)
	}

	// Filter by status=active
	req := httptest.NewRequest("GET", "/api/v2/agents?status=active", nil)
	w := httptest.NewRecorder()
	v2ListAgentsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	agents, ok := response["agents"].([]interface{})
	if !ok {
		t.Fatal("No agents in response")
	}

	activeIDs := map[string]bool{}
	for _, agentInterface := range agents {
		agent := agentInterface.(map[string]interface{})
		if agent["status"] == "OFFLINE" {
			t.Errorf("Expected no OFFLINE agents in active filter, got %v", agent["id"])
		}
		activeIDs[agent["id"].(string)] = true
	}

	if !activeIDs[agentActiveRecent.ID] || !activeIDs[agentActiveOld.ID] {
		t.Error("Expected active agents to be included in active filter")
	}
	if activeIDs[agentStale.ID] {
		t.Error("Expected stale agent to be excluded from active filter")
	}

	// Filter by status=stale
	req = httptest.NewRequest("GET", "/api/v2/agents?status=stale", nil)
	w = httptest.NewRecorder()
	v2ListAgentsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	response = map[string]interface{}{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	agents, ok = response["agents"].([]interface{})
	if !ok {
		t.Fatal("No agents in response")
	}

	staleIDs := map[string]bool{}
	for _, agentInterface := range agents {
		agent := agentInterface.(map[string]interface{})
		if agent["status"] != "OFFLINE" {
			t.Errorf("Expected only OFFLINE agents in stale filter, got %v", agent["id"])
		}
		staleIDs[agent["id"].(string)] = true
	}

	if !staleIDs[agentStale.ID] {
		t.Error("Expected stale agent to be included in stale filter")
	}
	if staleIDs[agentActiveRecent.ID] || staleIDs[agentActiveOld.ID] {
		t.Error("Expected active agents to be excluded from stale filter")
	}

	// Filter by since
	since := now.Add(-10 * time.Minute).Format(time.RFC3339Nano)
	req = httptest.NewRequest("GET", "/api/v2/agents?since="+url.QueryEscape(since), nil)
	w = httptest.NewRecorder()
	v2ListAgentsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	response = map[string]interface{}{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	agents, ok = response["agents"].([]interface{})
	if !ok {
		t.Fatal("No agents in response")
	}

	sinceIDs := map[string]bool{}
	for _, agentInterface := range agents {
		agent := agentInterface.(map[string]interface{})
		sinceIDs[agent["id"].(string)] = true
	}

	if !sinceIDs[agentActiveRecent.ID] {
		t.Error("Expected recent agent to be included in since filter")
	}
	if sinceIDs[agentActiveOld.ID] || sinceIDs[agentStale.ID] {
		t.Error("Expected old agents to be excluded from since filter")
	}
}

func TestAgentListingFilterValidation(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	tests := []struct {
		name string
		path string
	}{
		{name: "invalid status", path: "/api/v2/agents?status=unknown"},
		{name: "empty status", path: "/api/v2/agents?status="},
		{name: "invalid since", path: "/api/v2/agents?since=not-a-time"},
		{name: "empty since", path: "/api/v2/agents?since="},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()
			v2ListAgentsHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
		})
	}
}

// TestAgentHeartbeat tests agent heartbeat endpoint
func TestAgentHeartbeat(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	// Register an agent first
	agent, err := RegisterAgent(db, "Test Agent", []string{"code"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// Set agent to offline first
	err = UpdateAgentStatus(db, agent.ID, AgentStatusOffline)
	if err != nil {
		t.Fatalf("Failed to update agent status: %v", err)
	}

	// Send heartbeat
	req := httptest.NewRequest("POST", "/api/v2/agents/"+agent.ID+"/heartbeat", nil)
	w := httptest.NewRecorder()

	v2AgentActionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success: true in response")
	}

	// Verify agent is now online in database
	updatedAgent, err := GetAgent(db, agent.ID)
	if err != nil {
		t.Fatalf("Failed to get updated agent: %v", err)
	}

	if updatedAgent.Status != AgentStatusOnline {
		t.Errorf("Expected agent status ONLINE, got %v", updatedAgent.Status)
	}

	if updatedAgent.LastSeen == nil {
		t.Error("Expected LastSeen to be set after heartbeat")
	}
}

// TestAgentHeartbeatNotFound tests heartbeat for non-existent agent
func TestAgentHeartbeatNotFound(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/v2/agents/agent_nonexistent/heartbeat", nil)
	w := httptest.NewRecorder()

	v2AgentActionHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestAgentHeartbeatMethodNotAllowed(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v2/agents/agent_abc/heartbeat", nil)
	w := httptest.NewRecorder()

	v2AgentActionHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodGet {
		t.Fatalf("expected details.method GET, got %v", details["method"])
	}
}

// TestStaleAgentCleanup tests the background job for marking stale agents offline
func TestStaleAgentCleanup(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	// Register an agent
	agent, err := RegisterAgent(db, "Stale Agent", []string{"code"}, nil)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// Set agent status to online and last seen to 15 minutes ago
	oldTime := time.Now().UTC().Add(-15 * time.Minute)
	oldTimeISO := oldTime.Format("2006-01-02T15:04:05.000Z")

	_, err = db.Exec("UPDATE agents SET status = ?, last_seen = ? WHERE id = ?",
		AgentStatusOnline, oldTimeISO, agent.ID)
	if err != nil {
		t.Fatalf("Failed to update agent timestamp: %v", err)
	}

	// Run stale agent cleanup with 10 minute threshold
	err = MarkStaleAgentsOffline(db, 10)
	if err != nil {
		t.Fatalf("Failed to mark stale agents offline: %v", err)
	}

	// Verify agent is now offline
	updatedAgent, err := GetAgent(db, agent.ID)
	if err != nil {
		t.Fatalf("Failed to get updated agent: %v", err)
	}

	if updatedAgent.Status != AgentStatusOffline {
		t.Errorf("Expected agent to be marked offline, got status %v", updatedAgent.Status)
	}
}

// TestAgentEndToEnd tests the complete agent lifecycle
func TestAgentEndToEnd(t *testing.T) {
	_, cleanup := setupAgentTestDB(t)
	defer cleanup()

	t.Log("Test: Complete agent lifecycle")

	// 1. Register agent
	payload := `{
		"name": "E2E Test Agent",
		"capabilities": ["code", "test"],
		"metadata": {"version": "1.0.0"}
	}`

	req := httptest.NewRequest("POST", "/api/v2/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2RegisterAgentHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Agent registration failed with status %d", w.Code)
	}

	var regResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &regResponse)
	agent := regResponse["agent"].(map[string]interface{})
	agentID := agent["id"].(string)

	t.Logf("✓ Agent registered: %s", agentID)

	// 2. List agents and verify it's included
	req = httptest.NewRequest("GET", "/api/v2/agents", nil)
	w = httptest.NewRecorder()
	v2ListAgentsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Agent listing failed with status %d", w.Code)
	}

	var listResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResponse)
	agents := listResponse["agents"].([]interface{})

	found := false
	for _, agentInterface := range agents {
		a := agentInterface.(map[string]interface{})
		if a["id"].(string) == agentID {
			found = true
			if a["status"] != "ONLINE" {
				t.Errorf("Expected initial status ONLINE, got %v", a["status"])
			}
			break
		}
	}
	if !found {
		t.Fatal("Registered agent not found in listing")
	}

	t.Log("✓ Agent found in listing with ONLINE status")

	// 3. Send heartbeat
	req = httptest.NewRequest("POST", "/api/v2/agents/"+agentID+"/heartbeat", nil)
	w = httptest.NewRecorder()
	v2AgentActionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Heartbeat failed with status %d", w.Code)
	}

	t.Log("✓ Heartbeat successful")

	// 4. Mark as stale and verify cleanup
	// Set last seen to old time
	oldTime := time.Now().UTC().Add(-20 * time.Minute)
	oldTimeISO := oldTime.Format("2006-01-02T15:04:05.000Z")
	db.Exec("UPDATE agents SET last_seen = ? WHERE id = ?", oldTimeISO, agentID)

	err := MarkStaleAgentsOffline(db, 10)
	if err != nil {
		t.Fatalf("Stale agent cleanup failed: %v", err)
	}

	// Verify agent is offline
	updatedAgent, err := GetAgent(db, agentID)
	if err != nil {
		t.Fatalf("Failed to get agent after cleanup: %v", err)
	}

	if updatedAgent.Status != AgentStatusOffline {
		t.Errorf("Expected agent to be offline after cleanup, got %v", updatedAgent.Status)
	}

	t.Log("✓ Stale agent marked offline")
	t.Log("✅ End-to-end test passed!")
}
