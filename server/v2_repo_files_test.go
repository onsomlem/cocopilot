package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestV2RepoFilesListEmpty(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	files := resp["files"].([]interface{})
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
	if resp["total"].(float64) != 0 {
		t.Fatalf("expected total 0, got %v", resp["total"])
	}
	if resp["limit"].(float64) != 100 {
		t.Fatalf("expected default limit 100, got %v", resp["limit"])
	}
	if resp["offset"].(float64) != 0 {
		t.Fatalf("expected offset 0, got %v", resp["offset"])
	}
}

func TestV2RepoFilesListProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/proj_nonexistent/files", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesPutAndGet(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// PUT a file
	payload := `{"content_hash":"abc123","size_bytes":1024,"language":"go","metadata":{"author":"test"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/"+project.ID+"/files/src/main.go", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var putResp RepoFile
	if err := json.NewDecoder(w.Body).Decode(&putResp); err != nil {
		t.Fatalf("decode PUT response failed: %v", err)
	}
	if putResp.Path != "src/main.go" {
		t.Fatalf("expected path 'src/main.go', got '%s'", putResp.Path)
	}
	if putResp.ProjectID != project.ID {
		t.Fatalf("expected project_id %s, got %s", project.ID, putResp.ProjectID)
	}
	if putResp.ContentHash == nil || *putResp.ContentHash != "abc123" {
		t.Fatalf("expected content_hash 'abc123', got %v", putResp.ContentHash)
	}
	if putResp.SizeBytes == nil || *putResp.SizeBytes != 1024 {
		t.Fatalf("expected size_bytes 1024, got %v", putResp.SizeBytes)
	}
	if putResp.Language == nil || *putResp.Language != "go" {
		t.Fatalf("expected language 'go', got %v", putResp.Language)
	}

	// GET the file
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files/src/main.go", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var getResp RepoFile
	if err := json.NewDecoder(w.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode GET response failed: %v", err)
	}
	if getResp.Path != "src/main.go" {
		t.Fatalf("expected path 'src/main.go', got '%s'", getResp.Path)
	}
	if getResp.ID != putResp.ID {
		t.Fatalf("expected same ID, got %s vs %s", getResp.ID, putResp.ID)
	}
}

func TestV2RepoFilesGetNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files/nonexistent.go", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesDelete(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// PUT a file first
	payload := `{"content_hash":"abc123","size_bytes":512,"language":"python"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/"+project.ID+"/files/app.py", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// DELETE the file
	req = httptest.NewRequest(http.MethodDelete, "/api/v2/projects/"+project.ID+"/files/app.py", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE expected 204, got %d body=%s", w.Code, w.Body.String())
	}

	// GET should now return 404
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files/app.py", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GET after DELETE expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesDeleteNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v2/projects/"+project.ID+"/files/missing.txt", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesListWithFilter(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Create some files
	files := []struct {
		path     string
		language string
	}{
		{"main.go", "go"},
		{"util.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
	}

	for _, f := range files {
		payload := `{"language":"` + f.language + `","size_bytes":100}`
		req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/"+project.ID+"/files/"+f.path, strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("PUT %s expected 200, got %d body=%s", f.path, w.Code, w.Body.String())
		}
	}

	// List all files
	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("LIST expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 4 {
		t.Fatalf("expected total 4, got %v", resp["total"])
	}

	// Filter by language=go
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files?language=go", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("LIST with filter expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["total"].(float64) != 2 {
		t.Fatalf("expected total 2 for language=go, got %v", resp["total"])
	}

	// Test pagination: limit=2, offset=0
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files?limit=2&offset=0", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("LIST with pagination expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	json.NewDecoder(w.Body).Decode(&resp)
	resultFiles := resp["files"].([]interface{})
	if len(resultFiles) != 2 {
		t.Fatalf("expected 2 files with limit=2, got %d", len(resultFiles))
	}
	if resp["total"].(float64) != 4 {
		t.Fatalf("expected total 4, got %v", resp["total"])
	}
	if resp["limit"].(float64) != 2 {
		t.Fatalf("expected limit 2, got %v", resp["limit"])
	}
}

func TestV2RepoFilesSync(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Sync files
	payload := `{
		"files": [
			{"path": "main.go", "content_hash": "hash1", "size_bytes": 100, "language": "go"},
			{"path": "util.go", "content_hash": "hash2", "size_bytes": 200, "language": "go"},
			{"path": "app.py", "content_hash": "hash3", "size_bytes": 300, "language": "python"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/files/sync", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("SYNC expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var syncResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&syncResp)
	if syncResp["synced"].(float64) != 3 {
		t.Fatalf("expected synced 3, got %v", syncResp["synced"])
	}
	if syncResp["deleted"].(float64) != 0 {
		t.Fatalf("expected deleted 0, got %v", syncResp["deleted"])
	}

	// Verify files are listed
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var listResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&listResp)
	if listResp["total"].(float64) != 3 {
		t.Fatalf("expected 3 files after sync, got %v", listResp["total"])
	}
}

func TestV2RepoFilesSyncWithPurge(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// First sync: create 3 files
	payload := `{
		"files": [
			{"path": "main.go", "content_hash": "hash1", "size_bytes": 100, "language": "go"},
			{"path": "util.go", "content_hash": "hash2", "size_bytes": 200, "language": "go"},
			{"path": "app.py", "content_hash": "hash3", "size_bytes": 300, "language": "python"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/files/sync", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("First SYNC expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// Second sync with purge: only keep main.go
	payload = `{
		"files": [
			{"path": "main.go", "content_hash": "hash1_updated", "size_bytes": 150, "language": "go"}
		],
		"purge": true
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/files/sync", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("SYNC with purge expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var syncResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&syncResp)
	if syncResp["synced"].(float64) != 1 {
		t.Fatalf("expected synced 1, got %v", syncResp["synced"])
	}
	if syncResp["deleted"].(float64) != 2 {
		t.Fatalf("expected deleted 2, got %v", syncResp["deleted"])
	}

	// Verify only main.go remains
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var listResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&listResp)
	if listResp["total"].(float64) != 1 {
		t.Fatalf("expected 1 file after purge sync, got %v", listResp["total"])
	}

	// Verify the remaining file was updated
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files/main.go", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var fileResp RepoFile
	json.NewDecoder(w.Body).Decode(&fileResp)
	if fileResp.ContentHash == nil || *fileResp.ContentHash != "hash1_updated" {
		t.Fatalf("expected updated content_hash, got %v", fileResp.ContentHash)
	}
	if fileResp.SizeBytes == nil || *fileResp.SizeBytes != 150 {
		t.Fatalf("expected updated size_bytes 150, got %v", fileResp.SizeBytes)
	}
}

func TestV2RepoFilesSyncProjectNotFound(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{"files": [{"path": "test.go"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/proj_nonexistent/files/sync", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesSyncMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files/sync", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesListMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/files", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesDetailMethodNotAllowed(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/files/test.go", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesPutUpdate(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Initial PUT
	payload := `{"content_hash":"hash1","size_bytes":100,"language":"go"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/"+project.ID+"/files/main.go", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("First PUT expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var first RepoFile
	json.NewDecoder(w.Body).Decode(&first)

	// Update PUT (same path)
	payload = `{"content_hash":"hash2","size_bytes":200,"language":"go"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v2/projects/"+project.ID+"/files/main.go", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Second PUT expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var second RepoFile
	json.NewDecoder(w.Body).Decode(&second)

	// Should be same file (upsert), content_hash updated
	if second.ContentHash == nil || *second.ContentHash != "hash2" {
		t.Fatalf("expected updated content_hash 'hash2', got %v", second.ContentHash)
	}
	if second.SizeBytes == nil || *second.SizeBytes != 200 {
		t.Fatalf("expected updated size_bytes 200, got %v", second.SizeBytes)
	}
}

func TestV2RepoFilesSyncEmptyFiles(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// Sync with empty files array
	payload := `{"files": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/files/sync", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("SYNC expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["synced"].(float64) != 0 {
		t.Fatalf("expected synced 0, got %v", resp["synced"])
	}
}

func TestV2RepoFilesSyncMissingFilesField(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	payload := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/"+project.ID+"/files/sync", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV2RepoFilesDeepPath(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(db, "Files Project", "/tmp/files-project", nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	// PUT a file with a deep path
	payload := `{"content_hash":"deep","size_bytes":50,"language":"go"}`
	deepPath := "src/internal/pkg/handler/main.go"
	req := httptest.NewRequest(http.MethodPut, "/api/v2/projects/"+project.ID+"/files/"+deepPath, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT deep path expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var putResp RepoFile
	json.NewDecoder(w.Body).Decode(&putResp)
	if putResp.Path != deepPath {
		t.Fatalf("expected path '%s', got '%s'", deepPath, putResp.Path)
	}

	// GET the deep path file
	req = httptest.NewRequest(http.MethodGet, "/api/v2/projects/"+project.ID+"/files/"+deepPath, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET deep path expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var getResp RepoFile
	json.NewDecoder(w.Body).Decode(&getResp)
	if getResp.Path != deepPath {
		t.Fatalf("expected path '%s', got '%s'", deepPath, getResp.Path)
	}

	// DELETE the deep path file
	req = httptest.NewRequest(http.MethodDelete, "/api/v2/projects/"+project.ID+"/files/"+deepPath, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE deep path expected 204, got %d body=%s", w.Code, w.Body.String())
	}
}
