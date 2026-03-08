// API v2 Database Access Layer Tests
package server

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupV2TestDB creates a test database with all migrations applied (including v2)
func setupV2TestDB(t *testing.T) (*sql.DB, func()) {
	testDB, cleanup := setupTestDB(t)

	// Run migrations to add v2 tables
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return testDB, cleanup
}

// createTestTask creates a test task for testing v2 features
func createTestTask(t *testing.T, db *sql.DB) int {
	result, err := db.Exec(`
		INSERT INTO tasks (instructions, status, project_id, created_at)
		VALUES (?, ?, ?, ?)
	`, "Test task", StatusNotPicked, "proj_default", nowISO())
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// ============================================================================
// Project Tests
// ============================================================================

func TestCreateProject(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	settings := map[string]interface{}{
		"theme": "dark",
		"lang":  "en",
	}
	project, err := CreateProject(db, "Test Project", "/home/test", settings)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.ID == "" {
		t.Error("Project ID should not be empty")
	}
	if project.Name != "Test Project" {
		t.Errorf("Name mismatch: got %s, want Test Project", project.Name)
	}
	if project.Workdir != "/home/test" {
		t.Errorf("Workdir mismatch: got %s, want /home/test", project.Workdir)
	}
	if project.Settings["theme"] != "dark" {
		t.Errorf("Settings theme mismatch: got %v, want dark", project.Settings["theme"])
	}
	if project.CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
}

func TestGetProject(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	created, err := CreateProject(db, "Test Project", "/home/test", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	retrieved, err := GetProject(db, created.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, created.ID)
	}
	if retrieved.Name != "Test Project" {
		t.Errorf("Name mismatch: got %s, want Test Project", retrieved.Name)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := GetProject(db, "proj_nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestListProjects(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Should have default project
	projects, err := ListProjects(db)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	initialCount := len(projects)

	// Create two more projects
	_, err = CreateProject(db, "Project 1", "/home/proj1", nil)
	if err != nil {
		t.Fatalf("CreateProject 1 failed: %v", err)
	}
	_, err = CreateProject(db, "Project 2", "/home/proj2", nil)
	if err != nil {
		t.Fatalf("CreateProject 2 failed: %v", err)
	}

	// List should now have 2 more
	projects, err = ListProjects(db)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(projects) != initialCount+2 {
		t.Errorf("Expected %d projects, got %d", initialCount+2, len(projects))
	}
}

func TestUpdateProject(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	created, err := CreateProject(db, "Original Name", "/original/path", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Update name only
	newName := "Updated Name"
	updated, err := UpdateProject(db, created.ID, &newName, nil, nil)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Name not updated: got %s, want Updated Name", updated.Name)
	}
	if updated.Workdir != "/original/path" {
		t.Errorf("Workdir changed unexpectedly: got %s", updated.Workdir)
	}

	// Update workdir only
	newWorkdir := "/new/path"
	updated, err = UpdateProject(db, created.ID, nil, &newWorkdir, nil)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Name changed unexpectedly: got %s", updated.Name)
	}
	if updated.Workdir != "/new/path" {
		t.Errorf("Workdir not updated: got %s, want /new/path", updated.Workdir)
	}

	// Update settings
	newSettings := map[string]interface{}{"key": "value"}
	updated, err = UpdateProject(db, created.ID, nil, nil, newSettings)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	if updated.Settings["key"] != "value" {
		t.Errorf("Settings not updated: got %v", updated.Settings)
	}
}

func TestUpdateProjectNotFound(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	name := "New Name"
	_, err := UpdateProject(db, "proj_nonexistent", &name, nil, nil)
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

func TestDeleteProject(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	created, err := CreateProject(db, "To Delete", "/tmp/delete", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	err = DeleteProject(db, created.ID)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	// Verify it's deleted
	_, err = GetProject(db, created.ID)
	if err == nil {
		t.Error("Expected error for deleted project")
	}
}

func TestDeleteProjectNotFound(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	err := DeleteProject(db, "proj_nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

func TestDeleteDefaultProject(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	err := DeleteProject(db, "proj_default")
	if err == nil {
		t.Error("Expected error when deleting default project")
	}
	if !contains(err.Error(), "cannot delete default") {
		t.Errorf("Expected 'cannot delete default' error, got: %v", err)
	}
}

// ============================================================================
// Policy Tests
// ============================================================================

func TestCreatePolicy(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	description := "Basic policy"
	rules := []PolicyRule{
		{
			"type":   "automation.block",
			"reason": "Audit",
		},
	}

	policy, err := CreatePolicy(db, "proj_default", "  Default Policy ", &description, rules, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	if policy.ID == "" {
		t.Error("Policy ID should not be empty")
	}
	if policy.Name != "Default Policy" {
		t.Errorf("Policy name mismatch: got %s", policy.Name)
	}
	if policy.Description == nil || *policy.Description != "Basic policy" {
		t.Errorf("Policy description mismatch: got %v", policy.Description)
	}
	if len(policy.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(policy.Rules))
	}
	if policy.Rules[0]["type"] != "automation.block" {
		t.Errorf("Policy rules mismatch: got %v", policy.Rules[0]["type"])
	}
	if !policy.Enabled {
		t.Error("Policy should be enabled")
	}
	if policy.CreatedAt == "" {
		t.Error("Policy created_at should not be empty")
	}
}

func TestListPoliciesByProject(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	otherProject, err := CreateProject(db, "Other Project", "/tmp/other", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policyA, err := CreatePolicy(db, "proj_default", "Policy A", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	_, err = CreatePolicy(db, "proj_default", "Policy B", nil, []PolicyRule{}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	_, err = CreatePolicy(db, otherProject.ID, "Policy C", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	policies, total, err := ListPoliciesByProject(db, "proj_default", nil, 100, 0, "created_at", "asc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("Expected 2 policies, got %d", len(policies))
	}
	if total != 2 {
		t.Fatalf("Expected total 2, got %d", total)
	}

	seen := map[string]bool{}
	for _, policy := range policies {
		seen[policy.ID] = true
	}
	if !seen[policyA.ID] {
		t.Errorf("Expected policy %s to be listed", policyA.ID)
	}
}

func TestListPoliciesByProjectPaging(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(db, "proj_default", "Policy A", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	_, err = CreatePolicy(db, "proj_default", "Policy B", nil, []PolicyRule{}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	policies, total, err := ListPoliciesByProject(db, "proj_default", nil, 1, 1, "created_at", "asc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("Expected 1 policy, got %d", len(policies))
	}
	if total != 2 {
		t.Fatalf("Expected total 2, got %d", total)
	}
}

func TestListPoliciesByProjectSorting(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	policyGamma, err := CreatePolicy(db, "proj_default", "Gamma", nil, []PolicyRule{}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	policyAlpha, err := CreatePolicy(db, "proj_default", "Alpha", nil, []PolicyRule{}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	policyBeta, err := CreatePolicy(db, "proj_default", "Beta", nil, []PolicyRule{}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	base := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	updates := []struct {
		id   string
		time time.Time
	}{
		{policyAlpha.ID, base.Add(2 * time.Minute)},
		{policyBeta.ID, base.Add(1 * time.Minute)},
		{policyGamma.ID, base.Add(3 * time.Minute)},
	}
	for _, update := range updates {
		_, err := db.Exec("UPDATE policies SET created_at = ? WHERE id = ?", update.time.Format(leaseTimeFormat), update.id)
		if err != nil {
			t.Fatalf("failed to update created_at: %v", err)
		}
	}

	policies, _, err := ListPoliciesByProject(db, "proj_default", nil, 100, 0, "created_at", "asc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if len(policies) != 3 {
		t.Fatalf("Expected 3 policies, got %d", len(policies))
	}
	if policies[0].ID != policyBeta.ID || policies[1].ID != policyAlpha.ID || policies[2].ID != policyGamma.ID {
		t.Fatalf("unexpected created_at asc order: %v, %v, %v", policies[0].ID, policies[1].ID, policies[2].ID)
	}

	policies, _, err = ListPoliciesByProject(db, "proj_default", nil, 100, 0, "created_at", "desc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if policies[0].ID != policyGamma.ID || policies[1].ID != policyAlpha.ID || policies[2].ID != policyBeta.ID {
		t.Fatalf("unexpected created_at desc order: %v, %v, %v", policies[0].ID, policies[1].ID, policies[2].ID)
	}

	policies, _, err = ListPoliciesByProject(db, "proj_default", nil, 100, 0, "name", "asc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if policies[0].Name != "Alpha" || policies[1].Name != "Beta" || policies[2].Name != "Gamma" {
		t.Fatalf("unexpected name asc order: %s, %s, %s", policies[0].Name, policies[1].Name, policies[2].Name)
	}

	policies, _, err = ListPoliciesByProject(db, "proj_default", nil, 100, 0, "name", "desc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if policies[0].Name != "Gamma" || policies[1].Name != "Beta" || policies[2].Name != "Alpha" {
		t.Fatalf("unexpected name desc order: %s, %s, %s", policies[0].Name, policies[1].Name, policies[2].Name)
	}
}

func TestCreatePolicyDefaults(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	policy, err := CreatePolicy(db, "proj_default", "  Policy Defaults  ", nil, nil, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	if policy.Name != "Policy Defaults" {
		t.Fatalf("expected trimmed name, got %s", policy.Name)
	}
	if policy.Description != nil {
		t.Fatalf("expected nil description, got %v", policy.Description)
	}
	if policy.Rules == nil || len(policy.Rules) != 0 {
		t.Fatalf("expected empty rules list, got %v", policy.Rules)
	}
	if policy.Enabled {
		t.Fatal("expected policy to be disabled")
	}

	policies, _, err := ListPoliciesByProject(db, "proj_default", nil, 100, 0, "created_at", "asc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	found := false
	for _, item := range policies {
		if item.ID == policy.ID {
			found = true
			if item.Enabled {
				t.Fatalf("expected persisted policy to be disabled")
			}
			if item.Rules == nil || len(item.Rules) != 0 {
				t.Fatalf("expected persisted rules empty, got %v", item.Rules)
			}
		}
	}
	if !found {
		t.Fatalf("expected policy %s to be listed", policy.ID)
	}

}

func TestGetPolicy(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Policy Get", "/tmp/policy-get", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	description := "Policy description"
	policy, err := CreatePolicy(db, project.ID, "Policy One", &description, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	retrieved, err := GetPolicy(db, project.ID, policy.ID)
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}
	if retrieved.ID != policy.ID {
		t.Fatalf("expected policy id %s, got %s", policy.ID, retrieved.ID)
	}
	if retrieved.Name != "Policy One" {
		t.Fatalf("expected policy name Policy One, got %s", retrieved.Name)
	}
	if retrieved.Description == nil || *retrieved.Description != description {
		t.Fatalf("expected description %s, got %v", description, retrieved.Description)
	}
	if len(retrieved.Rules) != 1 || retrieved.Rules[0]["type"] != "automation.block" {
		t.Fatalf("expected rules.type automation.block, got %v", retrieved.Rules)
	}

	_, err = GetPolicy(db, "proj_default", policy.ID)
	if err == nil {
		t.Fatal("expected not found for policy with wrong project")
	}
	if !contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestListPoliciesByProjectEnabledFilter(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, err := CreatePolicy(db, "proj_default", "Policy Enabled", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	_, err = CreatePolicy(db, "proj_default", "Policy Disabled", nil, []PolicyRule{}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	enabledOnly := true
	policies, total, err := ListPoliciesByProject(db, "proj_default", &enabledOnly, 100, 0, "created_at", "asc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("Expected 1 enabled policy, got %d", len(policies))
	}
	if total != 1 {
		t.Fatalf("Expected total 1, got %d", total)
	}
	if !policies[0].Enabled {
		t.Fatalf("Expected enabled policy, got disabled")
	}

	disabledOnly := false
	policies, total, err = ListPoliciesByProject(db, "proj_default", &disabledOnly, 100, 0, "created_at", "asc")
	if err != nil {
		t.Fatalf("ListPoliciesByProject failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("Expected 1 disabled policy, got %d", len(policies))
	}
	if total != 1 {
		t.Fatalf("Expected total 1, got %d", total)
	}
	if policies[0].Enabled {
		t.Fatalf("Expected disabled policy, got enabled")
	}
}

func TestUpdatePolicy(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Policy Update", "/tmp/policy-update", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Alpha", nil, []PolicyRule{}, false)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	newName := "Policy Beta"
	newDescription := "Updated description"
	newRules := []PolicyRule{{"type": "automation.block", "reason": "Block"}}
	enabled := true

	updated, err := UpdatePolicy(db, project.ID, policy.ID, &newName, &newDescription, newRules, &enabled)
	if err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected updated name %s, got %s", newName, updated.Name)
	}
	if updated.Description == nil || *updated.Description != newDescription {
		t.Fatalf("expected updated description %s, got %v", newDescription, updated.Description)
	}
	if len(updated.Rules) != 1 || updated.Rules[0]["type"] != "automation.block" {
		t.Fatalf("expected updated rules.type automation.block, got %v", updated.Rules)
	}
	if !updated.Enabled {
		t.Fatal("expected updated policy to be enabled")
	}

	clearDescription := "   "
	updated, err = UpdatePolicy(db, project.ID, policy.ID, nil, &clearDescription, nil, nil)
	if err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}
	if updated.Description != nil {
		t.Fatalf("expected description to be cleared, got %v", updated.Description)
	}
}

func TestDeletePolicy(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Policy Delete", "/tmp/policy-delete", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	policy, err := CreatePolicy(db, project.ID, "Policy Delete", nil, []PolicyRule{{"type": "automation.block"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	if err := DeletePolicy(db, project.ID, policy.ID); err != nil {
		t.Fatalf("DeletePolicy failed: %v", err)
	}

	_, err = GetPolicy(db, project.ID, policy.ID)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// Run Tests
// ============================================================================

func TestCreateRun(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, err := CreateRun(db, taskID, "agent-001")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	if run.ID == "" {
		t.Error("Run ID should not be empty")
	}
	if run.TaskID != taskID {
		t.Errorf("Task ID mismatch: got %d, want %d", run.TaskID, taskID)
	}
	if run.AgentID != "agent-001" {
		t.Errorf("Agent ID mismatch: got %s, want agent-001", run.AgentID)
	}
	if run.Status != RunStatusRunning {
		t.Errorf("Status should be RUNNING, got %s", run.Status)
	}
}

func TestGetRun(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	created, err := CreateRun(db, taskID, "agent-001")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	retrieved, err := GetRun(db, created.ID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, created.ID)
	}
	if retrieved.TaskID != taskID {
		t.Errorf("Task ID mismatch: got %d, want %d", retrieved.TaskID, taskID)
	}
}

func TestGetRunsByTaskID(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)

	// Create multiple runs
	_, err := CreateRun(db, taskID, "agent-001")
	if err != nil {
		t.Fatalf("CreateRun 1 failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	_, err = CreateRun(db, taskID, "agent-002")
	if err != nil {
		t.Fatalf("CreateRun 2 failed: %v", err)
	}

	runs, err := GetRunsByTaskID(db, taskID)
	if err != nil {
		t.Fatalf("GetRunsByTaskID failed: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("Expected 2 runs, got %d", len(runs))
	}
}

func TestUpdateRunStatus(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, err := CreateRun(db, taskID, "agent-001")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	errorMsg := "Test error"
	err = UpdateRunStatus(db, run.ID, RunStatusFailed, &errorMsg)
	if err != nil {
		t.Fatalf("UpdateRunStatus failed: %v", err)
	}

	updated, err := GetRun(db, run.ID)
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}

	if updated.Status != RunStatusFailed {
		t.Errorf("Status should be FAILED, got %s", updated.Status)
	}
	if updated.FinishedAt == nil {
		t.Error("FinishedAt should not be nil")
	}
	if updated.Error == nil || *updated.Error != errorMsg {
		t.Errorf("Error message mismatch: got %v, want %s", updated.Error, errorMsg)
	}
}

func TestDeleteRun(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, err := CreateRun(db, taskID, "agent-001")
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	err = DeleteRun(db, run.ID)
	if err != nil {
		t.Fatalf("DeleteRun failed: %v", err)
	}

	_, err = GetRun(db, run.ID)
	if err == nil {
		t.Error("GetRun should fail after deletion")
	}
}

// ============================================================================
// RunStep Tests
// ============================================================================

func TestCreateRunStep(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	details := map[string]interface{}{"key": "value", "count": 42}
	step, err := CreateRunStep(db, run.ID, "Initialization", StepStatusStarted, details)
	if err != nil {
		t.Fatalf("CreateRunStep failed: %v", err)
	}

	if step.ID == "" {
		t.Error("Step ID should not be empty")
	}
	if step.Name != "Initialization" {
		t.Errorf("Name mismatch: got %s, want Initialization", step.Name)
	}
	if step.Status != StepStatusStarted {
		t.Errorf("Status should be STARTED, got %s", step.Status)
	}
	if step.Details["key"] != "value" {
		t.Error("Details not preserved correctly")
	}
}

func TestGetRunSteps(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	_, err := CreateRunStep(db, run.ID, "Step 1", StepStatusStarted, nil)
	if err != nil {
		t.Fatalf("CreateRunStep 1 failed: %v", err)
	}
	_, err = CreateRunStep(db, run.ID, "Step 2", StepStatusSucceeded, nil)
	if err != nil {
		t.Fatalf("CreateRunStep 2 failed: %v", err)
	}

	steps, err := GetRunSteps(db, run.ID)
	if err != nil {
		t.Fatalf("GetRunSteps failed: %v", err)
	}

	if len(steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(steps))
	}
}

func TestUpdateRunStepStatus(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")
	step, _ := CreateRunStep(db, run.ID, "Test", StepStatusStarted, nil)

	err := UpdateRunStepStatus(db, step.ID, StepStatusSucceeded)
	if err != nil {
		t.Fatalf("UpdateRunStepStatus failed: %v", err)
	}

	steps, _ := GetRunSteps(db, run.ID)
	if steps[0].Status != StepStatusSucceeded {
		t.Errorf("Status should be SUCCEEDED, got %s", steps[0].Status)
	}
}

// ============================================================================
// RunLog Tests
// ============================================================================

func TestCreateRunLog(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	err := CreateRunLog(db, run.ID, "stdout", "Hello, World!")
	if err != nil {
		t.Fatalf("CreateRunLog failed: %v", err)
	}

	logs, err := GetRunLogs(db, run.ID)
	if err != nil {
		t.Fatalf("GetRunLogs failed: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
	if logs[0].Stream != "stdout" {
		t.Errorf("Stream mismatch: got %s, want stdout", logs[0].Stream)
	}
	if logs[0].Chunk != "Hello, World!" {
		t.Errorf("Chunk mismatch: got %s, want Hello, World!", logs[0].Chunk)
	}
}

func TestGetRunLogs(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	_ = CreateRunLog(db, run.ID, "stdout", "Line 1")
	_ = CreateRunLog(db, run.ID, "stderr", "Error line")
	_ = CreateRunLog(db, run.ID, "stdout", "Line 2")

	logs, err := GetRunLogs(db, run.ID)
	if err != nil {
		t.Fatalf("GetRunLogs failed: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(logs))
	}
}

// ============================================================================
// Artifact Tests
// ============================================================================

func TestCreateArtifact(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	sha := "abc123"
	size := int64(1024)
	metadata := map[string]interface{}{"filename": "test.txt"}

	artifact, err := CreateArtifact(db, run.ID, "file", "/tmp/test.txt", &sha, &size, metadata)
	if err != nil {
		t.Fatalf("CreateArtifact failed: %v", err)
	}

	if artifact.ID == "" {
		t.Error("Artifact ID should not be empty")
	}
	if artifact.Kind != "file" {
		t.Errorf("Kind mismatch: got %s, want file", artifact.Kind)
	}
	if artifact.Sha256 == nil || *artifact.Sha256 != sha {
		t.Error("SHA256 not preserved correctly")
	}
}

func TestGetArtifactsByRunID(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	_, err := CreateArtifact(db, run.ID, "file", "/tmp/1.txt", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateArtifact 1 failed: %v", err)
	}
	_, err = CreateArtifact(db, run.ID, "log", "/tmp/2.log", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateArtifact 2 failed: %v", err)
	}

	artifacts, err := GetArtifactsByRunID(db, run.ID)
	if err != nil {
		t.Fatalf("GetArtifactsByRunID failed: %v", err)
	}

	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}
}

// ============================================================================
// ToolInvocation Tests
// ============================================================================

func TestCreateToolInvocation(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	input := map[string]interface{}{"arg1": "value1", "arg2": 123}
	invocation, err := CreateToolInvocation(db, run.ID, "grep_search", input)
	if err != nil {
		t.Fatalf("CreateToolInvocation failed: %v", err)
	}

	if invocation.ID == "" {
		t.Error("Invocation ID should not be empty")
	}
	if invocation.ToolName != "grep_search" {
		t.Errorf("ToolName mismatch: got %s, want grep_search", invocation.ToolName)
	}
}

func TestUpdateToolInvocationOutput(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")
	invocation, _ := CreateToolInvocation(db, run.ID, "test_tool", nil)

	output := map[string]interface{}{"result": "success", "data": []int{1, 2, 3}}
	err := UpdateToolInvocationOutput(db, invocation.ID, output)
	if err != nil {
		t.Fatalf("UpdateToolInvocationOutput failed: %v", err)
	}

	invocations, _ := GetToolInvocationsByRunID(db, run.ID)
	if len(invocations) != 1 {
		t.Fatalf("Expected 1 invocation, got %d", len(invocations))
	}
	if invocations[0].Output["result"] != "success" {
		t.Error("Output not preserved correctly")
	}
	if invocations[0].FinishedAt == nil {
		t.Error("FinishedAt should not be nil")
	}
}

func TestGetToolInvocationsByRunID(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	run, _ := CreateRun(db, taskID, "agent-001")

	_, _ = CreateToolInvocation(db, run.ID, "tool1", nil)
	_, _ = CreateToolInvocation(db, run.ID, "tool2", nil)

	invocations, err := GetToolInvocationsByRunID(db, run.ID)
	if err != nil {
		t.Fatalf("GetToolInvocationsByRunID failed: %v", err)
	}

	if len(invocations) != 2 {
		t.Errorf("Expected 2 invocations, got %d", len(invocations))
	}
}

// ============================================================================
// Lease Tests
// ============================================================================

func TestCreateLease(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)

	lease, err := CreateLease(db, taskID, "agent-001", "exclusive")
	if err != nil {
		t.Fatalf("CreateLease failed: %v", err)
	}

	if lease.ID == "" {
		t.Error("Lease ID should not be empty")
	}
	if lease.TaskID != taskID {
		t.Errorf("Task ID mismatch: got %d, want %d", lease.TaskID, taskID)
	}
	if lease.AgentID != "agent-001" {
		t.Errorf("Agent ID mismatch: got %s, want agent-001", lease.AgentID)
	}
}

func TestGetLeaseByTaskID(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)

	created, _ := CreateLease(db, taskID, "agent-001", "exclusive")

	retrieved, err := GetLeaseByTaskID(db, taskID)
	if err != nil {
		t.Fatalf("GetLeaseByTaskID failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Lease should not be nil")
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, created.ID)
	}
}

func TestDeleteLease(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)

	lease, _ := CreateLease(db, taskID, "agent-001", "exclusive")

	err := DeleteLease(db, lease.ID)
	if err != nil {
		t.Fatalf("DeleteLease failed: %v", err)
	}

	retrieved, _ := GetLeaseByTaskID(db, taskID)
	if retrieved != nil {
		t.Error("Lease should be nil after deletion")
	}
}

func TestDeleteExpiredLeases(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID1 := createTestTask(t, db)
	taskID2 := createTestTask(t, db)
	_, _ = db.Exec("UPDATE tasks SET status = ? WHERE id IN (?, ?)", StatusInProgress, taskID1, taskID2)

	expiredTime := time.Now().UTC().Add(-1 * time.Hour).Format(leaseTimeFormat)
	validTime := time.Now().UTC().Add(1 * time.Hour).Format(leaseTimeFormat)
	_, _ = db.Exec(`
		INSERT INTO leases (id, task_id, agent_id, mode, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)
	`,
		"lease_expired", taskID1, "agent-001", "exclusive", nowISO(), expiredTime,
		"lease_valid", taskID2, "agent-002", "exclusive", nowISO(), validTime,
	)

	count, err := DeleteExpiredLeases(db)
	if err != nil {
		t.Fatalf("DeleteExpiredLeases failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 expired lease deleted, got %d", count)
	}

	// Valid lease should still exist
	validLease, _ := GetLeaseByTaskID(db, taskID2)
	if validLease == nil {
		t.Error("Valid lease should still exist")
	}

	// Expired lease task should be requeued
	var status1, status2 string
	_ = db.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID1).Scan(&status1)
	_ = db.QueryRow("SELECT status FROM tasks WHERE id = ?", taskID2).Scan(&status2)
	if status1 != string(StatusNotPicked) {
		t.Errorf("Expected task %d to be requeued to NOT_PICKED, got %s", taskID1, status1)
	}
	if status2 != string(StatusInProgress) {
		t.Errorf("Expected task %d to remain IN_PROGRESS, got %s", taskID2, status2)
	}
}

func TestCreateLeaseEmitsCreatedEvent(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	lease, err := CreateLease(db, taskID, "agent-001", "exclusive")
	if err != nil {
		t.Fatalf("CreateLease failed: %v", err)
	}

	events, err := GetEventsByProjectID(db, "proj_default", 20)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}

	found := false
	for _, event := range events {
		if event.Kind == "lease.created" && event.EntityID == lease.ID {
			found = true
			if event.Payload["agent_id"] != "agent-001" {
				t.Fatalf("expected agent_id agent-001, got %v", event.Payload["agent_id"])
			}
			if event.Payload["task_id"] != float64(taskID) {
				t.Fatalf("expected task_id %d, got %v", taskID, event.Payload["task_id"])
			}
		}
	}
	if !found {
		t.Fatalf("expected lease.created event for lease %s", lease.ID)
	}
}

func TestDeleteExpiredLeasesEmitsExpiredEvent(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	_, _ = db.Exec("UPDATE tasks SET status = ? WHERE id = ?", StatusInProgress, taskID)

	expiredLeaseID := "lease_expired_test"
	expiredTime := time.Now().UTC().Add(-1 * time.Hour).Format(leaseTimeFormat)
	_, err := db.Exec(`
		INSERT INTO leases (id, task_id, agent_id, mode, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, expiredLeaseID, taskID, "agent-expired", "exclusive", nowISO(), expiredTime)
	if err != nil {
		t.Fatalf("failed to insert expired lease: %v", err)
	}

	deleted, err := DeleteExpiredLeases(db)
	if err != nil {
		t.Fatalf("DeleteExpiredLeases failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted lease, got %d", deleted)
	}

	events, err := GetEventsByProjectID(db, "proj_default", 20)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}

	found := false
	for _, event := range events {
		if event.Kind == "lease.expired" && event.EntityID == expiredLeaseID {
			found = true
			if event.Payload["reason"] != "expired" {
				t.Fatalf("expected reason=expired, got %v", event.Payload["reason"])
			}
		}
	}
	if !found {
		t.Fatalf("expected lease.expired event for %s", expiredLeaseID)
	}
}

func TestReleaseLeaseEmitsReleasedEvent(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	lease, err := CreateLease(db, taskID, "agent-001", "exclusive")
	if err != nil {
		t.Fatalf("CreateLease failed: %v", err)
	}

	released, _, err := ReleaseLease(db, lease.ID, "manual_release")
	if err != nil {
		t.Fatalf("ReleaseLease failed: %v", err)
	}
	if !released {
		t.Fatal("expected lease to be released")
	}

	releasedAgain, _, err := ReleaseLease(db, lease.ID, "manual_release")
	if err != nil {
		t.Fatalf("second ReleaseLease returned error: %v", err)
	}
	if releasedAgain {
		t.Fatal("expected second release to report not released")
	}

	events, err := GetEventsByProjectID(db, "proj_default", 50)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}

	releasedEventCount := 0
	for _, event := range events {
		if event.Kind == "lease.released" && event.EntityID == lease.ID {
			releasedEventCount++
			if event.Payload["reason"] != "manual_release" {
				t.Fatalf("expected release reason manual_release, got %v", event.Payload["reason"])
			}
		}
	}

	if releasedEventCount != 1 {
		t.Fatalf("expected exactly one lease.released event, got %d", releasedEventCount)
	}
}

// ============================================================================
// Event Tests
// ============================================================================

func TestCreateEvent(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	payload := map[string]interface{}{"action": "created", "user": "admin"}
	event, err := CreateEvent(db, "proj_default", "task.created", "task", "123", payload)
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	if event.ID == "" {
		t.Error("Event ID should not be empty")
	}
	if event.Kind != "task.created" {
		t.Errorf("Kind mismatch: got %s, want task.created", event.Kind)
	}
	if event.Payload["action"] != "created" {
		t.Error("Payload not preserved correctly")
	}
}

func TestGetEventsByProjectID(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, _ = CreateEvent(db, "proj_default", "task.created", "task", "1", nil)
	time.Sleep(10 * time.Millisecond)
	_, _ = CreateEvent(db, "proj_default", "task.updated", "task", "1", nil)
	_, _ = CreateEvent(db, "proj_other", "task.created", "task", "2", nil)

	events, err := GetEventsByProjectID(db, "proj_default", 100)
	if err != nil {
		t.Fatalf("GetEventsByProjectID failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events for proj_default, got %d", len(events))
	}
}

func TestPruneEventsByAge(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	oldTime := time.Now().UTC().Add(-48 * time.Hour).Format(leaseTimeFormat)
	newTime := time.Now().UTC().Add(-2 * time.Hour).Format(leaseTimeFormat)

	insertEvent := func(id, createdAt string) {
		_, err := db.Exec(
			"INSERT INTO events (id, project_id, kind, entity_type, entity_id, created_at, payload_json) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, "proj_default", "task.created", "task", "1", createdAt, "{}",
		)
		if err != nil {
			t.Fatalf("failed to insert event %s: %v", id, err)
		}
	}

	insertEvent("evt_old_1", oldTime)
	insertEvent("evt_old_2", oldTime)
	insertEvent("evt_new_1", newTime)

	deleted, err := PruneEvents(db, 1, 0)
	if err != nil {
		t.Fatalf("PruneEvents failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 events deleted, got %d", deleted)
	}

	var remaining int
	if err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&remaining); err != nil {
		t.Fatalf("failed to count events: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected 1 event remaining, got %d", remaining)
	}
}

func TestPruneEventsByMaxRows(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	base := time.Now().UTC().Add(-10 * time.Minute)
	for i := 0; i < 5; i++ {
		createdAt := base.Add(time.Duration(i) * time.Minute).Format(leaseTimeFormat)
		id := fmt.Sprintf("evt_row_%d", i)
		_, err := db.Exec(
			"INSERT INTO events (id, project_id, kind, entity_type, entity_id, created_at, payload_json) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, "proj_default", "task.updated", "task", "1", createdAt, "{}",
		)
		if err != nil {
			t.Fatalf("failed to insert event %s: %v", id, err)
		}
	}

	deleted, err := PruneEvents(db, 0, 2)
	if err != nil {
		t.Fatalf("PruneEvents failed: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("expected 3 events deleted, got %d", deleted)
	}

	rows, err := db.Query("SELECT id FROM events ORDER BY created_at DESC")
	if err != nil {
		t.Fatalf("failed to fetch remaining events: %v", err)
	}
	defer rows.Close()

	remaining := make([]string, 0, 2)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("failed to scan remaining event id: %v", err)
		}
		remaining = append(remaining, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate remaining events: %v", err)
	}

	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining events, got %d", len(remaining))
	}
	if remaining[0] != "evt_row_4" || remaining[1] != "evt_row_3" {
		t.Fatalf("unexpected remaining events order: %v", remaining)
	}
}

// ============================================================================
// Memory Tests
// ============================================================================

func TestCreateMemory(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	value := map[string]interface{}{"key": "value", "count": 10}
	sourceRefs := []string{"task_123", "task_456"}

	memory, err := CreateMemory(db, "proj_default", "variables", "api_key", value, sourceRefs)
	if err != nil {
		t.Fatalf("CreateMemory failed: %v", err)
	}

	if memory.ID == "" {
		t.Error("Memory ID should not be empty")
	}
	if memory.Key != "api_key" {
		t.Errorf("Key mismatch: got %s, want api_key", memory.Key)
	}
	if memory.Value["key"] != "value" {
		t.Error("Value not preserved correctly")
	}
	if len(memory.SourceRefs) != 2 {
		t.Errorf("Expected 2 source refs, got %d", len(memory.SourceRefs))
	}
}

func TestGetMemory(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	value := map[string]interface{}{"data": "test"}
	_, _ = CreateMemory(db, "proj_default", "config", "setting1", value, nil)

	retrieved, err := GetMemory(db, "proj_default", "config", "setting1")
	if err != nil {
		t.Fatalf("GetMemory failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Memory should not be nil")
	}
	if retrieved.Key != "setting1" {
		t.Errorf("Key mismatch: got %s, want setting1", retrieved.Key)
	}
	if retrieved.Value["data"] != "test" {
		t.Error("Value not preserved correctly")
	}
}

func TestUpdateMemory(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Create initial memory
	value1 := map[string]interface{}{"version": 1}
	_, _ = CreateMemory(db, "proj_default", "config", "settings", value1, nil)

	// Update it (upsert)
	value2 := map[string]interface{}{"version": 2}
	_, err := CreateMemory(db, "proj_default", "config", "settings", value2, nil)
	if err != nil {
		t.Fatalf("Update memory failed: %v", err)
	}

	// Retrieve and verify
	retrieved, _ := GetMemory(db, "proj_default", "config", "settings")
	if retrieved.Value["version"] != float64(2) { // JSON unmarshal makes it float64
		t.Error("Memory was not updated correctly")
	}
}

func TestGetMemoriesByScope(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, _ = CreateMemory(db, "proj_default", "variables", "var1", map[string]interface{}{"a": 1}, nil)
	_, _ = CreateMemory(db, "proj_default", "variables", "var2", map[string]interface{}{"b": 2}, nil)
	_, _ = CreateMemory(db, "proj_default", "config", "cfg1", map[string]interface{}{"c": 3}, nil)

	memories, err := GetMemoriesByScope(db, "proj_default", "variables")
	if err != nil {
		t.Fatalf("GetMemoriesByScope failed: %v", err)
	}

	if len(memories) != 2 {
		t.Errorf("Expected 2 memories in variables scope, got %d", len(memories))
	}
}

func TestDeleteMemory(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	_, _ = CreateMemory(db, "proj_default", "temp", "key1", map[string]interface{}{"x": 1}, nil)

	err := DeleteMemory(db, "proj_default", "temp", "key1")
	if err != nil {
		t.Fatalf("DeleteMemory failed: %v", err)
	}

	retrieved, _ := GetMemory(db, "proj_default", "temp", "key1")
	if retrieved != nil {
		t.Error("Memory should be nil after deletion")
	}
}

// ============================================================================
// ContextPack Tests
// ============================================================================

func TestCreateContextPack(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	contents := map[string]interface{}{"files": []string{"main.go", "test.go"}, "summary": "test context"}

	pack, err := CreateContextPack(db, "proj_default", taskID, "Test context pack", contents)
	if err != nil {
		t.Fatalf("CreateContextPack failed: %v", err)
	}

	if pack.ID == "" {
		t.Error("ContextPack ID should not be empty")
	}
	if pack.Summary != "Test context pack" {
		t.Errorf("Summary mismatch: got %s, want Test context pack", pack.Summary)
	}
	if pack.Contents["summary"] != "test context" {
		t.Error("Contents not preserved correctly")
	}
}

func TestGetContextPackByTaskID(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	taskID := createTestTask(t, db)
	contents := map[string]interface{}{"data": "test"}
	_, _ = CreateContextPack(db, "proj_default", taskID, "Summary", contents)

	pack, err := GetContextPackByTaskID(db, taskID)
	if err != nil {
		t.Fatalf("GetContextPackByTaskID failed: %v", err)
	}

	if pack == nil {
		t.Fatal("ContextPack should not be nil")
	}
	if pack.TaskID != taskID {
		t.Errorf("Task ID mismatch: got %d, want %d", pack.TaskID, taskID)
	}
	if pack.Contents["data"] != "test" {
		t.Error("Contents not preserved correctly")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestCompleteRunWorkflow(t *testing.T) {
	db, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Create a task
	taskID := createTestTask(t, db)

	// Create a lease
	lease, _ := CreateLease(db, taskID, "agent-001", "exclusive")
	if lease == nil {
		t.Fatal("Failed to create lease")
	}

	// Create a run
	run, _ := CreateRun(db, taskID, "agent-001")
	if run == nil {
		t.Fatal("Failed to create run")
	}

	// Create steps
	step1, _ := CreateRunStep(db, run.ID, "Initialize", StepStatusStarted, nil)
	_ = UpdateRunStepStatus(db, step1.ID, StepStatusSucceeded)
	_, _ = CreateRunStep(db, run.ID, "Execute", StepStatusSucceeded, nil)

	// Add logs
	_ = CreateRunLog(db, run.ID, "stdout", "Starting execution...")
	_ = CreateRunLog(db, run.ID, "stdout", "Completed successfully")

	// Create artifacts
	_, _ = CreateArtifact(db, run.ID, "file", "/tmp/output.txt", nil, nil, nil)

	// Create tool invocations
	invocation, _ := CreateToolInvocation(db, run.ID, "grep_search", map[string]interface{}{"query": "test"})
	_ = UpdateToolInvocationOutput(db, invocation.ID, map[string]interface{}{"matches": 5})

	// Complete the run
	_ = UpdateRunStatus(db, run.ID, RunStatusSucceeded, nil)

	// Release the lease
	_ = DeleteLease(db, lease.ID)

	// Create event
	_, _ = CreateEvent(db, "proj_default", "task.completed", "task", fmt.Sprintf("%d", taskID), nil)

	// Verify everything
	steps, _ := GetRunSteps(db, run.ID)
	if len(steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(steps))
	}

	logs, _ := GetRunLogs(db, run.ID)
	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}

	artifacts, _ := GetArtifactsByRunID(db, run.ID)
	if len(artifacts) != 1 {
		t.Errorf("Expected 1 artifact, got %d", len(artifacts))
	}

	invocations, _ := GetToolInvocationsByRunID(db, run.ID)
	if len(invocations) != 1 {
		t.Errorf("Expected 1 tool invocation, got %d", len(invocations))
	}

	finalRun, _ := GetRun(db, run.ID)
	if finalRun.Status != RunStatusSucceeded {
		t.Errorf("Run status should be SUCCEEDED, got %s", finalRun.Status)
	}

	releasedLease, _ := GetLeaseByTaskID(db, taskID)
	if releasedLease != nil {
		t.Error("Lease should be released")
	}
}
