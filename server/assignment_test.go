package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupAssignmentTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	oldDB := db
	db = testDB
	cleanup := func() {
		db.Close()
		db = oldDB
		os.Remove(testDBPath)
	}
	return testDB, cleanup
}

func TestClaimTaskByID_Atomic(t *testing.T) {
	testDB, cleanup := setupAssignmentTestDB(t)
	defer cleanup()

	// Create a project and task.
	project, err := CreateProject(testDB, "test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, "test-task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim the task.
	env, err := ClaimTaskByID(testDB, task.ID, "agent_test", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}
	if env.Task == nil {
		t.Fatal("expected non-nil task in envelope")
	}
	if env.Lease == nil {
		t.Fatal("expected non-nil lease in envelope")
	}
	if env.Run == nil {
		t.Fatal("expected non-nil run in envelope")
	}

	// Verify task status was updated.
	if env.Task.StatusV2 != TaskStatusClaimed {
		t.Errorf("expected status_v2 CLAIMED, got %s", env.Task.StatusV2)
	}

	// Verify double-claim fails.
	_, err = ClaimTaskByID(testDB, task.ID, "agent_test2", "exclusive")
	if err == nil {
		t.Error("expected error on double claim, got nil")
	}
}

func TestCompleteTask(t *testing.T) {
	testDB, cleanup := setupAssignmentTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, "completable-task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	// Claim first.
	_, err = ClaimTaskByID(testDB, task.ID, "agent_test", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	// Complete.
	output := "done"
	completed, err := CompleteTask(testDB, task.ID, &output)
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if completed.StatusV2 != TaskStatusSucceeded {
		t.Errorf("expected SUCCEEDED, got %s", completed.StatusV2)
	}

	// Double complete should fail.
	_, err = CompleteTask(testDB, task.ID, &output)
	if err == nil {
		t.Error("expected error on double complete, got nil")
	}
}

func TestFailTask(t *testing.T) {
	testDB, cleanup := setupAssignmentTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "test-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	task, err := CreateTaskV2(testDB, "failing-task", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}

	_, err = ClaimTaskByID(testDB, task.ID, "agent_test", "exclusive")
	if err != nil {
		t.Fatalf("ClaimTaskByID: %v", err)
	}

	failed, err := FailTask(testDB, task.ID, "something broke")
	if err != nil {
		t.Fatalf("FailTask: %v", err)
	}
	if failed.StatusV2 != TaskStatusFailed {
		t.Errorf("expected StatusV2 FAILED, got %s", failed.StatusV2)
	}
	// Regression: v1 status must NOT be "COMPLETE" on failure (bug fix)
	if failed.StatusV1 == StatusComplete {
		t.Errorf("v1 status must not be COMPLETE on failure; got %s", failed.StatusV1)
	}
	if failed.StatusV1 != StatusFailed {
		t.Errorf("expected v1 status FAILED, got %s", failed.StatusV1)
	}
}

func TestValidateWorkdir(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid absolute path", "/home/user/project", false},
		{"valid nested path", "/Users/weli/code/myproject", false},
		{"empty", "", true},
		{"relative path", "relative/path", true},
		{"dot relative", "./somewhere", true},
		{"null byte", "/home/user/\x00evil", true},
		{"root dir", "/", true},
		{"etc dir", "/etc", true},
		{"proc dir", "/proc", true},
		{"proc subdir", "/proc/1/fd", true},
		{"sys dir", "/sys", true},
		{"sys subdir", "/sys/class/net", true},
		{"dev dir", "/dev", true},
		{"dev subdir", "/dev/null", true},
		{"usr dir", "/usr", true},
		{"bin dir", "/bin", true},
		{"var dir", "/var", true},
		{"tmp dir", "/tmp", true},
		{"root home", "/root", true},
		{"boot dir", "/boot", true},
		{"boot subdir", "/boot/grub", true},
		{"sbin dir", "/sbin", true},
		{"valid tmp subdir", "/tmp/myproject", false},
		{"opt dir", "/opt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkdir(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWorkdir(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestWithV1MutationAuth_Disabled(t *testing.T) {
	cfg := runtimeConfig{RequireAPIKey: false}
	called := false
	handler := withV1MutationAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/save", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Error("handler should have been called when auth is disabled")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWithV1MutationAuth_MissingKey(t *testing.T) {
	cfg := runtimeConfig{RequireAPIKey: true, APIKey: "secret"}
	handler := withV1MutationAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without API key")
	})

	req := httptest.NewRequest(http.MethodPost, "/save", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestWithV1MutationAuth_InvalidKey(t *testing.T) {
	cfg := runtimeConfig{RequireAPIKey: true, APIKey: "secret"}
	handler := withV1MutationAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with wrong API key")
	})

	req := httptest.NewRequest(http.MethodPost, "/save", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestWithV1MutationAuth_ValidKey(t *testing.T) {
	cfg := runtimeConfig{RequireAPIKey: true, APIKey: "secret"}
	called := false
	handler := withV1MutationAuth(cfg, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/save", nil)
	req.Header.Set("X-API-Key", "secret")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Error("handler should have been called with valid API key")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}
