package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestV2ProjectTreeSuccess(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	root, err := os.MkdirTemp("", "coco-tree-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})

	subdir := filepath.Join(root, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	filePath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	project, err := CreateProject(testDB, "Tree Project", root, nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/tree", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tree, ok := resp["tree"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tree object, got %T", resp["tree"])
	}
	if tree["path"] != "." {
		t.Fatalf("expected tree.path '.', got %v", tree["path"])
	}
	if tree["kind"] != "dir" {
		t.Fatalf("expected tree.kind 'dir', got %v", tree["kind"])
	}

	children, ok := tree["children"].([]interface{})
	if !ok {
		t.Fatalf("expected tree.children array, got %T", tree["children"])
	}

	childByPath := map[string]map[string]interface{}{}
	for _, child := range children {
		childMap, ok := child.(map[string]interface{})
		if !ok {
			continue
		}
		pathValue, _ := childMap["path"].(string)
		if pathValue != "" {
			childByPath[pathValue] = childMap
		}
	}

	fileNode, ok := childByPath["notes.txt"]
	if !ok {
		t.Fatalf("expected notes.txt entry in tree")
	}
	if fileNode["kind"] != "file" {
		t.Fatalf("expected notes.txt kind file, got %v", fileNode["kind"])
	}
	if _, ok := fileNode["size"].(float64); !ok {
		t.Fatalf("expected notes.txt size to be present")
	}

	dirNode, ok := childByPath["subdir"]
	if !ok {
		t.Fatalf("expected subdir entry in tree")
	}
	if dirNode["kind"] != "dir" {
		t.Fatalf("expected subdir kind dir, got %v", dirNode["kind"])
	}
}

func TestV2ProjectTreeProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_missing/tree", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "NOT_FOUND")
}

func TestV2ProjectTreeWorkdirMissing(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	missing := filepath.Join(os.TempDir(), "coco-tree-missing")
	project, err := CreateProject(testDB, "Missing Dir", missing, nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/tree", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectTreeMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_default/tree", nil)
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
