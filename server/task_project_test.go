package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestTaskProjectAssociation tests that tasks are properly associated with projects
func TestTaskProjectAssociation(t *testing.T) {
	// Setup test database with migrations
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Create a test project
	project, err := CreateProject(testDB, "Test Project", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Create task with explicit project_id
	form := url.Values{}
	form.Add("instructions", "Test task in specific project")
	form.Add("project_id", project.ID)

	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify task was created with correct project_id
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	taskID := int(result["task_id"].(float64))

	var projectID string
	err = testDB.QueryRow("SELECT project_id FROM tasks WHERE id = ?", taskID).Scan(&projectID)
	if err != nil {
		t.Fatalf("Failed to query task: %v", err)
	}

	if projectID != project.ID {
		t.Errorf("Expected project_id %s, got %s", project.ID, projectID)
	}
}

// TestTaskCreationDefaultsToDefaultProject tests that tasks default to proj_default
func TestTaskCreationDefaultsToDefaultProject(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Create task without specifying project_id
	form := url.Values{}
	form.Add("instructions", "Test task without project")

	req := httptest.NewRequest(http.MethodPost, "/create", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify task was created with default project_id
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	taskID := int(result["task_id"].(float64))

	var projectID string
	err := testDB.QueryRow("SELECT project_id FROM tasks WHERE id = ?", taskID).Scan(&projectID)
	if err != nil {
		t.Fatalf("Failed to query task: %v", err)
	}

	if projectID != "proj_default" {
		t.Errorf("Expected project_id 'proj_default', got %s", projectID)
	}
}

// TestListProjectTasks tests getting tasks scoped to a project
func TestListProjectTasks(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Create two projects
	project1, err := CreateProject(testDB, "Project 1", "/tmp/p1", nil)
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}

	project2, err := CreateProject(testDB, "Project 2", "/tmp/p2", nil)
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	// Create tasks in different projects
	_, err = testDB.Exec("INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Task in project 1", StatusNotPicked, project1.ID)
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Task in project 2", StatusNotPicked, project2.ID)
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Another task in project 1", StatusNotPicked, project1.ID)
	if err != nil {
		t.Fatalf("Failed to create task 3: %v", err)
	}

	// Test getting tasks for project 1
	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project1.ID+"/tasks", nil)
	w := httptest.NewRecorder()

	v2ListProjectTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	tasks := result["tasks"].([]interface{})

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks for project 1, got %d", len(tasks))
	}
}

// TestListProjectTasksNotFound tests getting tasks for non-existent project
func TestListProjectTasksNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/nonexistent/tasks", nil)
	w := httptest.NewRecorder()

	v2ListProjectTasksHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestGetTasksByProject tests the getTasksByProjectJSON function
func TestGetTasksByProject(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Create a test project
	project, err := CreateProject(testDB, "Test Project", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Create tasks in the project
	_, err = testDB.Exec("INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Task 1", StatusNotPicked, project.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Task 2", StatusInProgress, project.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Query tasks for this project
	tasks, err := getTasksByProjectJSON(project.ID)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Verify tasks are from the correct project by checking instructions
	foundTask1 := false
	foundTask2 := false
	for _, task := range tasks {
		if task.Instructions == "Task 1" {
			foundTask1 = true
		}
		if task.Instructions == "Task 2" {
			foundTask2 = true
		}
	}

	if !foundTask1 || !foundTask2 {
		t.Error("Not all expected tasks were returned")
	}
}

// TestTaskIsolationBetweenProjects tests that tasks are properly isolated
func TestTaskIsolationBetweenProjects(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	// Create two projects
	project1, err := CreateProject(testDB, "Project 1", "/tmp/p1", nil)
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}

	project2, err := CreateProject(testDB, "Project 2", "/tmp/p2", nil)
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	// Create tasks in project 1
	_, err = testDB.Exec("INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Task in project 1", StatusNotPicked, project1.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Create tasks in project 2
	_, err = testDB.Exec("INSERT INTO tasks (instructions, status, project_id) VALUES (?, ?, ?)",
		"Task in project 2", StatusNotPicked, project2.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Get tasks for project 1
	tasks1, err := getTasksByProjectJSON(project1.ID)
	if err != nil {
		t.Fatalf("Failed to get tasks for project 1: %v", err)
	}

	// Get tasks for project 2
	tasks2, err := getTasksByProjectJSON(project2.ID)
	if err != nil {
		t.Fatalf("Failed to get tasks for project 2: %v", err)
	}

	// Verify isolation
	if len(tasks1) != 1 {
		t.Errorf("Expected 1 task in project 1, got %d", len(tasks1))
	}

	if len(tasks2) != 1 {
		t.Errorf("Expected 1 task in project 2, got %d", len(tasks2))
	}

	// Verify tasks don't overlap
	if len(tasks1) > 0 && len(tasks2) > 0 {
		if tasks1[0].Instructions == tasks2[0].Instructions {
			t.Error("Tasks from different projects should not be the same")
		}
	}
}
