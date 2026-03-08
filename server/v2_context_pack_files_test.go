package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContextPackIncludeFileMetadata(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = testDB

	// Create project
	_, err := testDB.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rftest", "RF Test", "/tmp", nowISO())
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create task
	result, err := testDB.Exec("INSERT INTO tasks (instructions, status, project_id, created_at) VALUES (?, ?, ?, ?)",
		"Test task", StatusNotPicked, "proj_rftest", nowISO())
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	taskID64, _ := result.LastInsertId()
	taskID := int(taskID64)

	// Upsert repo files
	sizeA := int64(1024)
	langGo := "Go"
	sizeB := int64(2048)
	langJS := "JavaScript"
	sizeC := int64(512)
	langPy := "Python"

	_, err = UpsertRepoFile(testDB, RepoFile{ProjectID: "proj_rftest", Path: "main.go", SizeBytes: &sizeA, Language: &langGo})
	if err != nil {
		t.Fatalf("upsert repo file: %v", err)
	}
	_, err = UpsertRepoFile(testDB, RepoFile{ProjectID: "proj_rftest", Path: "index.js", SizeBytes: &sizeB, Language: &langJS})
	if err != nil {
		t.Fatalf("upsert repo file: %v", err)
	}
	_, err = UpsertRepoFile(testDB, RepoFile{ProjectID: "proj_rftest", Path: "app.py", SizeBytes: &sizeC, Language: &langPy})
	if err != nil {
		t.Fatalf("upsert repo file: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	t.Run("include_file_metadata=true", func(t *testing.T) {
		payload := fmt.Sprintf(`{"task_id":%d,"include_file_metadata":true}`, taskID)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_rftest/context-packs", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pack := resp["context_pack"].(map[string]interface{})
		contents := pack["contents"].(map[string]interface{})

		repoFilesRaw, ok := contents["repo_files"]
		if !ok {
			t.Fatal("expected repo_files in contents")
		}
		repoFiles := repoFilesRaw.(map[string]interface{})

		total := int(repoFiles["total"].(float64))
		if total != 3 {
			t.Fatalf("expected total=3, got %d", total)
		}

		files := repoFiles["files"].([]interface{})
		if len(files) != 3 {
			t.Fatalf("expected 3 files, got %d", len(files))
		}

		// Verify file entries have expected fields
		for _, f := range files {
			entry := f.(map[string]interface{})
			if _, ok := entry["path"]; !ok {
				t.Error("file entry missing path")
			}
			if _, ok := entry["language"]; !ok {
				t.Error("file entry missing language")
			}
			if _, ok := entry["size_bytes"]; !ok {
				t.Error("file entry missing size_bytes")
			}
		}
	})

	t.Run("include_file_metadata=false", func(t *testing.T) {
		payload := fmt.Sprintf(`{"task_id":%d,"include_file_metadata":false}`, taskID)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_rftest/context-packs", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pack := resp["context_pack"].(map[string]interface{})
		contents := pack["contents"].(map[string]interface{})

		if _, ok := contents["repo_files"]; ok {
			t.Fatal("expected no repo_files when include_file_metadata is false")
		}
	})

	t.Run("budget_max_files_limits_results", func(t *testing.T) {
		payload := fmt.Sprintf(`{"task_id":%d,"include_file_metadata":true,"budget":{"max_files":2}}`, taskID)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_rftest/context-packs", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		pack := resp["context_pack"].(map[string]interface{})
		contents := pack["contents"].(map[string]interface{})

		repoFiles := contents["repo_files"].(map[string]interface{})
		files := repoFiles["files"].([]interface{})
		if len(files) != 2 {
			t.Fatalf("expected 2 files (budget limited), got %d", len(files))
		}
	})
}
