package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestV2ProjectChangesSuccess(t *testing.T) {
	requireGit(t)

	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	root, err := os.MkdirTemp("", "coco-changes-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})

	initGitRepo(t, root)
	filePath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	project, err := CreateProject(testDB, "Changes Project", root, nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/changes", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	changes, ok := resp["changes"].([]interface{})
	if !ok {
		t.Fatalf("expected changes array, got %T", resp["changes"])
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change entry, got %d", len(changes))
	}
	change, ok := changes[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected change object, got %T", changes[0])
	}
	if change["path"] != "notes.txt" {
		t.Fatalf("expected path notes.txt, got %v", change["path"])
	}
	if change["kind"] != "added" {
		t.Fatalf("expected kind added, got %v", change["kind"])
	}
	if ts, ok := change["ts"].(string); !ok || ts == "" {
		t.Fatalf("expected ts string, got %v", change["ts"])
	}
}

func TestV2ProjectChangesSinceFilters(t *testing.T) {
	requireGit(t)

	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	root, err := os.MkdirTemp("", "coco-changes-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})

	initGitRepo(t, root)
	filePath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	project, err := CreateProject(testDB, "Changes Project", root, nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	since := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/changes?since="+since, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	changes, ok := resp["changes"].([]interface{})
	if !ok {
		t.Fatalf("expected changes array, got %T", resp["changes"])
	}
	if len(changes) != 0 {
		t.Fatalf("expected 0 change entries, got %d", len(changes))
	}
}

func TestV2ProjectChangesProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing/changes", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectChangesInvalidSince(t *testing.T) {
	requireGit(t)

	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	root, err := os.MkdirTemp("", "coco-changes-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})

	initGitRepo(t, root)

	project, err := CreateProject(testDB, "Changes Project", root, nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/changes?since=not-a-time", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectChangesNotGitRepo(t *testing.T) {
	requireGit(t)

	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	root, err := os.MkdirTemp("", "coco-changes-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})

	project, err := CreateProject(testDB, "Changes Project", root, nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/changes", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectChangesMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/changes", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodPost {
		t.Fatalf("expected details.method %s, got %v", http.MethodPost, details["method"])
	}
	allowed, ok := details["allowed_methods"].([]interface{})
	if !ok || len(allowed) != 1 || allowed[0] != http.MethodGet {
		t.Fatalf("expected allowed_methods [%s], got %v", http.MethodGet, details["allowed_methods"])
	}
}

func requireGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initGitRepo(t *testing.T, dir string) {
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v output=%s", args, err, string(output))
	}
}