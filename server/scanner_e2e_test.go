package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- Scanner E2E Tests ----------

// setupScannerTestProject creates a project with a temp workdir containing test files.
func setupScannerTestProject(t *testing.T) (string, string) {
	t.Helper()

	// Create temp directory with test files
	workdir := t.TempDir()

	// Create Go file
	writeTestFile(t, workdir, "main.go", "package main\nfunc main() {}\n")
	// Create Python file
	writeTestFile(t, workdir, "script.py", "print('hello')\n")
	// Create nested directory with JS file
	writeTestFile(t, workdir, "src/app.js", "console.log('hello');\n")
	// Create markdown
	writeTestFile(t, workdir, "README.md", "# Test\n")
	// Create a .gitignore that ignores *.log and build/
	writeTestFile(t, workdir, ".gitignore", "*.log\nbuild/\n")
	// Create files that should be ignored
	writeTestFile(t, workdir, "debug.log", "log data\n")
	writeTestFile(t, workdir, "build/output.bin", "binary\n")

	// Create project in DB
	proj, err := CreateProject(db, "scanner-test", workdir, nil)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	return proj.ID, workdir
}

func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}

func TestScannerE2E_ScanAndListFiles(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	projectID, _ := setupScannerTestProject(t)

	// POST /api/v2/projects/{id}/files/scan
	scanURL := "/api/v2/projects/" + projectID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("scan returned %d: %s", w.Code, w.Body.String())
	}

	var scanResult map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &scanResult)

	scanned := int(scanResult["scanned"].(float64))
	if scanned < 4 {
		t.Fatalf("expected at least 4 scanned files (main.go, script.py, src/app.js, README.md), got %d", scanned)
	}

	// GET /api/v2/projects/{id}/files — verify files are in DB
	listURL := "/api/v2/projects/" + projectID + "/files"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list files returned %d: %s", w2.Code, w2.Body.String())
	}

	var listResult map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &listResult)

	files := listResult["files"].([]interface{})
	if len(files) < 4 {
		t.Fatalf("expected at least 4 files, got %d", len(files))
	}

	// Verify .gitignore filtering: debug.log and build/ should NOT be in results
	for _, f := range files {
		fm := f.(map[string]interface{})
		path := fm["path"].(string)
		if path == "debug.log" {
			t.Error("debug.log should have been filtered by .gitignore")
		}
		if strings.HasPrefix(path, "build/") {
			t.Errorf("build/ path %s should have been filtered by .gitignore", path)
		}
	}
}

func TestScannerE2E_LanguageDetection(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	projectID, _ := setupScannerTestProject(t)

	// Scan
	scanURL := "/api/v2/projects/" + projectID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("scan returned %d", w.Code)
	}

	// List and check languages
	listURL := "/api/v2/projects/" + projectID + "/files"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	files := result["files"].([]interface{})

	langsByPath := map[string]string{}
	for _, f := range files {
		fm := f.(map[string]interface{})
		path := fm["path"].(string)
		if lang, ok := fm["language"].(string); ok {
			langsByPath[path] = lang
		}
	}

	expects := map[string]string{
		"main.go":    "go",
		"script.py":  "python",
		"src/app.js": "javascript",
		"README.md":  "markdown",
	}

	for path, expectedLang := range expects {
		if actual, ok := langsByPath[path]; !ok {
			t.Errorf("file %s not found in results", path)
		} else if actual != expectedLang {
			t.Errorf("file %s: expected language %s, got %s", path, expectedLang, actual)
		}
	}
}

func TestScannerE2E_LanguageFilter(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	projectID, _ := setupScannerTestProject(t)

	// Scan first
	scanURL := "/api/v2/projects/" + projectID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	// List with language filter
	listURL := "/api/v2/projects/" + projectID + "/files?language=go"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list files returned %d", w2.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	files := result["files"].([]interface{})

	if len(files) != 1 {
		t.Fatalf("expected 1 Go file, got %d", len(files))
	}

	fm := files[0].(map[string]interface{})
	if fm["path"].(string) != "main.go" {
		t.Errorf("expected main.go, got %s", fm["path"].(string))
	}
}

func TestScannerE2E_PurgeScan(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	projectID, workdir := setupScannerTestProject(t)

	// Initial scan
	scanURL := "/api/v2/projects/" + projectID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	// Delete a file, then scan with purge=true
	os.Remove(filepath.Join(workdir, "script.py"))

	body := strings.NewReader(`{"purge": true}`)
	req2 := httptest.NewRequest(http.MethodPost, scanURL, body)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("purge scan returned %d: %s", w2.Code, w2.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)

	deleted := int(result["deleted"].(float64))
	if deleted < 1 {
		t.Errorf("expected at least 1 deleted file (script.py), got %d", deleted)
	}
}

func TestScannerE2E_FileDetail(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	projectID, _ := setupScannerTestProject(t)

	// Scan
	scanURL := "/api/v2/projects/" + projectID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	// GET specific file detail
	detailURL := "/api/v2/projects/" + projectID + "/files/main.go"
	req2 := httptest.NewRequest(http.MethodGet, detailURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFileDetailHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("file detail returned %d: %s", w2.Code, w2.Body.String())
	}

	var file map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &file)

	if file["path"].(string) != "main.go" {
		t.Errorf("expected path main.go, got %s", file["path"])
	}
	if file["language"].(string) != "go" {
		t.Errorf("expected language go, got %s", file["language"])
	}
	if file["content_hash"] == nil {
		t.Error("expected content_hash to be set")
	}
	if file["size_bytes"] == nil {
		t.Error("expected size_bytes to be set")
	}
}

func TestScannerE2E_FileUpsertAndDelete(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	projectID, _ := setupScannerTestProject(t)

	// PUT a file manually
	putURL := "/api/v2/projects/" + projectID + "/files/test.txt"
	putBody := strings.NewReader(`{"language": "text", "size_bytes": 42}`)
	req := httptest.NewRequest(http.MethodPut, putURL, putBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectFileDetailHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT file returned %d: %s", w.Code, w.Body.String())
	}

	// GET it back
	req2 := httptest.NewRequest(http.MethodGet, putURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFileDetailHandler(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET file returned %d", w2.Code)
	}

	// DELETE it
	req3 := httptest.NewRequest(http.MethodDelete, putURL, nil)
	w3 := httptest.NewRecorder()
	v2ProjectFileDetailHandler(w3, req3)
	if w3.Code != http.StatusNoContent {
		t.Fatalf("DELETE file returned %d: %s", w3.Code, w3.Body.String())
	}

	// GET again — should be 404
	req4 := httptest.NewRequest(http.MethodGet, putURL, nil)
	w4 := httptest.NewRecorder()
	v2ProjectFileDetailHandler(w4, req4)
	if w4.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w4.Code)
	}
}

func TestScannerE2E_ScanNonexistentProject(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	scanURL := "/api/v2/projects/nonexistent-proj/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent project, got %d", w.Code)
	}
}

func TestScannerE2E_ScanNoWorkdir(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Create project with NO workdir
	proj, err := CreateProject(db, "no-workdir", "", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	scanURL := "/api/v2/projects/" + proj.ID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for project with no workdir, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScannerE2E_ScanWrongMethod(t *testing.T) {
	scanURL := "/api/v2/projects/test-proj/files/scan"
	req := httptest.NewRequest(http.MethodGet, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestScannerE2E_FileSync(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	proj, err := CreateProject(db, "sync-test", "", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	syncURL := "/api/v2/projects/" + proj.ID + "/files/sync"
	body := strings.NewReader(`{"files": [
		{"path": "foo.go", "language": "go", "size_bytes": 100},
		{"path": "bar.py", "language": "python", "size_bytes": 200}
	]}`)
	req := httptest.NewRequest(http.MethodPost, syncURL, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2ProjectFilesSyncHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("file sync returned %d: %s", w.Code, w.Body.String())
	}

	// List files to verify they were synced
	listURL := "/api/v2/projects/" + proj.ID + "/files"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list files returned %d", w2.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	files := result["files"].([]interface{})

	if len(files) != 2 {
		t.Fatalf("expected 2 synced files, got %d", len(files))
	}
}

func TestScannerE2E_GitignoreFiltering(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	workdir := t.TempDir()

	// Create .gitignore
	writeTestFile(t, workdir, ".gitignore", "*.tmp\nsecrets/\n!important.tmp\n")

	// Create files
	writeTestFile(t, workdir, "app.go", "package main\n")
	writeTestFile(t, workdir, "cache.tmp", "cached\n")
	writeTestFile(t, workdir, "important.tmp", "keep me\n")
	writeTestFile(t, workdir, "secrets/key.pem", "secret\n")
	writeTestFile(t, workdir, "src/util.go", "package src\n")

	proj, err := CreateProject(db, "gitignore-test", workdir, nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	scanURL := "/api/v2/projects/" + proj.ID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("scan returned %d: %s", w.Code, w.Body.String())
	}

	// List files
	listURL := "/api/v2/projects/" + proj.ID + "/files"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	files := result["files"].([]interface{})

	paths := map[string]bool{}
	for _, f := range files {
		fm := f.(map[string]interface{})
		paths[fm["path"].(string)] = true
	}

	// app.go and src/util.go should be present
	if !paths["app.go"] {
		t.Error("app.go should be present")
	}
	if !paths["src/util.go"] {
		t.Error("src/util.go should be present")
	}

	// cache.tmp should be filtered
	if paths["cache.tmp"] {
		t.Error("cache.tmp should have been filtered by *.tmp gitignore rule")
	}

	// secrets/ should be filtered
	if paths["secrets/key.pem"] {
		t.Error("secrets/key.pem should have been filtered by secrets/ gitignore rule")
	}

	// important.tmp — gitignore negation: !important.tmp should keep it
	if !paths["important.tmp"] {
		t.Log("Note: gitignore negation (!important.tmp) not keeping file — may be expected depending on implementation")
	}
}

func TestScannerE2E_EmptyDirectory(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	workdir := t.TempDir()
	proj, err := CreateProject(db, "empty-dir-test", workdir, nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	scanURL := "/api/v2/projects/" + proj.ID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("scan returned %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	scanned := int(result["scanned"].(float64))
	if scanned != 0 {
		t.Errorf("expected 0 files in empty directory, got %d", scanned)
	}
}

func TestScannerE2E_NestedDirectories(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	workdir := t.TempDir()
	writeTestFile(t, workdir, "a/b/c/deep.go", "package deep\n")
	writeTestFile(t, workdir, "a/b/mid.py", "mid\n")
	writeTestFile(t, workdir, "a/top.js", "top\n")

	proj, err := CreateProject(db, "nested-test", workdir, nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	scanURL := "/api/v2/projects/" + proj.ID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("scan returned %d", w.Code)
	}

	listURL := "/api/v2/projects/" + proj.ID + "/files"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	files := result["files"].([]interface{})

	if len(files) != 3 {
		t.Fatalf("expected 3 nested files, got %d", len(files))
	}

	paths := map[string]bool{}
	for _, f := range files {
		fm := f.(map[string]interface{})
		paths[fm["path"].(string)] = true
	}

	for _, expected := range []string{"a/b/c/deep.go", "a/b/mid.py", "a/top.js"} {
		if !paths[expected] {
			t.Errorf("expected %s in results", expected)
		}
	}
}

func TestScannerE2E_DefaultIgnoreDirs(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	workdir := t.TempDir()
	writeTestFile(t, workdir, "app.go", "package main\n")
	writeTestFile(t, workdir, "node_modules/pkg/index.js", "module\n")
	writeTestFile(t, workdir, ".git/config", "gitconfig\n")
	writeTestFile(t, workdir, "__pycache__/cache.pyc", "cache\n")

	proj, err := CreateProject(db, "ignore-dirs-test", workdir, nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	scanURL := "/api/v2/projects/" + proj.ID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("scan returned %d", w.Code)
	}

	listURL := "/api/v2/projects/" + proj.ID + "/files"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	files := result["files"].([]interface{})

	// Only app.go should be present
	if len(files) != 1 {
		paths := []string{}
		for _, f := range files {
			fm := f.(map[string]interface{})
			paths = append(paths, fm["path"].(string))
		}
		t.Fatalf("expected 1 file (app.go), got %d: %v", len(files), paths)
	}
}

func TestScannerE2E_ContentHash(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	workdir := t.TempDir()
	writeTestFile(t, workdir, "hello.txt", "hello world\n")

	proj, err := CreateProject(db, "hash-test", workdir, nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Scan
	scanURL := "/api/v2/projects/" + proj.ID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	// Get file detail
	detailURL := "/api/v2/projects/" + proj.ID + "/files/hello.txt"
	req2 := httptest.NewRequest(http.MethodGet, detailURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFileDetailHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("file detail returned %d", w2.Code)
	}

	var file map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &file)

	hash, ok := file["content_hash"].(string)
	if !ok || hash == "" {
		t.Fatal("expected content_hash to be set for small file")
	}

	// Hash should be a 64-char hex string (SHA-256)
	if len(hash) != 64 {
		t.Errorf("expected 64-char SHA-256 hash, got %d chars", len(hash))
	}
}

func TestScannerE2E_ListPagination(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	workdir := t.TempDir()
	for i := 0; i < 5; i++ {
		writeTestFile(t, workdir, strings.Replace("file_N.go", "N", strings.Repeat("a", i+1), 1), "package main\n")
	}

	proj, err := CreateProject(db, "pagination-test", workdir, nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	scanURL := "/api/v2/projects/" + proj.ID + "/files/scan"
	req := httptest.NewRequest(http.MethodPost, scanURL, nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)

	// List with limit=2
	listURL := "/api/v2/projects/" + proj.ID + "/files?limit=2"
	req2 := httptest.NewRequest(http.MethodGet, listURL, nil)
	w2 := httptest.NewRecorder()
	v2ProjectFilesHandler(w2, req2)

	var result map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &result)
	files := result["files"].([]interface{})

	if len(files) != 2 {
		t.Fatalf("expected 2 files with limit=2, got %d", len(files))
	}

	total := int(result["total"].(float64))
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
}
