package server

import (
	"strings"
	"testing"
)

func TestRepoFileUpsertAndGet(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf1", "Test Project", "/tmp/test", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	lang := "go"
	hash := "abc123"
	size := int64(1024)
	mod := nowISO()

	file := RepoFile{
		ProjectID:    "proj_rf1",
		Path:         "src/main.go",
		Language:     &lang,
		ContentHash:  &hash,
		SizeBytes:    &size,
		LastModified: &mod,
		Metadata:     map[string]interface{}{"key": "value"},
	}

	result, err := UpsertRepoFile(db, file)
	if err != nil {
		t.Fatalf("UpsertRepoFile failed: %v", err)
	}

	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
	if !strings.HasPrefix(result.ID, "rf_") {
		t.Errorf("expected ID to start with rf_, got %s", result.ID)
	}
	if result.ProjectID != "proj_rf1" {
		t.Errorf("expected project_id proj_rf1, got %s", result.ProjectID)
	}
	if result.Path != "src/main.go" {
		t.Errorf("expected path src/main.go, got %s", result.Path)
	}
	if result.Language == nil || *result.Language != "go" {
		t.Error("expected language go")
	}
	if result.ContentHash == nil || *result.ContentHash != "abc123" {
		t.Error("expected content_hash abc123")
	}
	if result.SizeBytes == nil || *result.SizeBytes != 1024 {
		t.Error("expected size_bytes 1024")
	}

	// Get the same file
	got, err := GetRepoFile(db, "proj_rf1", "src/main.go")
	if err != nil {
		t.Fatalf("GetRepoFile failed: %v", err)
	}
	if got.ID != result.ID {
		t.Errorf("expected ID %s, got %s", result.ID, got.ID)
	}
	if got.Metadata == nil || got.Metadata["key"] != "value" {
		t.Error("expected metadata key=value")
	}
}

func TestRepoFileUpsertUpdatesExisting(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf2", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	lang1 := "go"
	file1 := RepoFile{
		ProjectID: "proj_rf2",
		Path:      "main.go",
		Language:  &lang1,
	}
	r1, err := UpsertRepoFile(db, file1)
	if err != nil {
		t.Fatal(err)
	}

	// Upsert same path with different data
	lang2 := "rust"
	hash := "newhash"
	file2 := RepoFile{
		ProjectID:   "proj_rf2",
		Path:        "main.go",
		Language:    &lang2,
		ContentHash: &hash,
	}
	r2, err := UpsertRepoFile(db, file2)
	if err != nil {
		t.Fatal(err)
	}

	// Should have updated the language and hash
	if r2.Language == nil || *r2.Language != "rust" {
		t.Errorf("expected language rust, got %v", r2.Language)
	}
	if r2.ContentHash == nil || *r2.ContentHash != "newhash" {
		t.Errorf("expected content_hash newhash, got %v", r2.ContentHash)
	}

	// Verify only one row exists
	_, total, err := ListRepoFiles(db, "proj_rf2", ListRepoFilesOpts{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("expected 1 file, got %d", total)
	}
	_ = r1
}

func TestRepoFileGetNotFound(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := GetRepoFile(db, "nonexistent", "nonexistent.go")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestRepoFileListWithFiltering(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf3", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	langGo := "go"
	langPy := "python"
	langJs := "javascript"

	files := []RepoFile{
		{ProjectID: "proj_rf3", Path: "main.go", Language: &langGo},
		{ProjectID: "proj_rf3", Path: "utils.go", Language: &langGo},
		{ProjectID: "proj_rf3", Path: "script.py", Language: &langPy},
		{ProjectID: "proj_rf3", Path: "app.js", Language: &langJs},
	}

	for _, f := range files {
		if _, err := UpsertRepoFile(db, f); err != nil {
			t.Fatal(err)
		}
	}

	// List all
	all, total, err := ListRepoFiles(db, "proj_rf3", ListRepoFilesOpts{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Errorf("expected total 4, got %d", total)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 files, got %d", len(all))
	}

	// Filter by language
	goFiles, goTotal, err := ListRepoFiles(db, "proj_rf3", ListRepoFilesOpts{Language: &langGo, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if goTotal != 2 {
		t.Errorf("expected 2 go files, got %d", goTotal)
	}
	if len(goFiles) != 2 {
		t.Errorf("expected 2 go files returned, got %d", len(goFiles))
	}

	// Pagination
	page1, _, err := ListRepoFiles(db, "proj_rf3", ListRepoFilesOpts{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 files in page 1, got %d", len(page1))
	}

	page2, _, err := ListRepoFiles(db, "proj_rf3", ListRepoFilesOpts{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 {
		t.Errorf("expected 2 files in page 2, got %d", len(page2))
	}

	// Verify sorted by path
	if page1[0].Path >= page1[1].Path {
		t.Errorf("expected sorted order, got %s >= %s", page1[0].Path, page1[1].Path)
	}
}

func TestRepoFileListEmptyProject(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	files, total, err := ListRepoFiles(db, "nonexistent_proj", ListRepoFilesOpts{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestRepoFileDelete(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf4", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	lang := "go"
	_, err = UpsertRepoFile(db, RepoFile{ProjectID: "proj_rf4", Path: "main.go", Language: &lang})
	if err != nil {
		t.Fatal(err)
	}

	err = DeleteRepoFile(db, "proj_rf4", "main.go")
	if err != nil {
		t.Fatalf("DeleteRepoFile failed: %v", err)
	}

	_, err = GetRepoFile(db, "proj_rf4", "main.go")
	if err == nil {
		t.Error("expected file to be deleted")
	}
}

func TestRepoFileDeleteNotFound(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	err := DeleteRepoFile(db, "nonexistent", "nonexistent.go")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestRepoFileDeleteByProject(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf5", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	lang := "go"
	for _, path := range []string{"a.go", "b.go", "c.go"} {
		if _, err := UpsertRepoFile(db, RepoFile{ProjectID: "proj_rf5", Path: path, Language: &lang}); err != nil {
			t.Fatal(err)
		}
	}

	// Verify 3 files exist
	_, total, err := ListRepoFiles(db, "proj_rf5", ListRepoFilesOpts{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("expected 3 files, got %d", total)
	}

	// Delete all
	err = DeleteRepoFilesByProject(db, "proj_rf5")
	if err != nil {
		t.Fatalf("DeleteRepoFilesByProject failed: %v", err)
	}

	_, total, err = ListRepoFiles(db, "proj_rf5", ListRepoFilesOpts{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Errorf("expected 0 files after bulk delete, got %d", total)
	}
}

func TestRepoFileNilOptionalFields(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf6", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	// Upsert with all nil optional fields
	file := RepoFile{
		ProjectID: "proj_rf6",
		Path:      "unknown_file",
	}
	result, err := UpsertRepoFile(db, file)
	if err != nil {
		t.Fatalf("UpsertRepoFile with nil fields failed: %v", err)
	}
	if result.ContentHash != nil {
		t.Error("expected nil content_hash")
	}
	if result.SizeBytes != nil {
		t.Error("expected nil size_bytes")
	}
	if result.Language != nil {
		t.Error("expected nil language")
	}
	if result.LastModified != nil {
		t.Error("expected nil last_modified")
	}
	if result.Metadata != nil {
		t.Error("expected nil metadata")
	}
}

func TestRepoFileSpecialCharactersInPath(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf7", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	specialPaths := []string{
		"path with spaces/file.go",
		"path/with-dashes/file.ts",
		"path/with.dots/file.py",
		"deeply/nested/directory/structure/file.rs",
		"src/utils/helper_test.go",
	}

	for _, p := range specialPaths {
		lang := "go"
		if _, err := UpsertRepoFile(db, RepoFile{ProjectID: "proj_rf7", Path: p, Language: &lang}); err != nil {
			t.Fatalf("failed to upsert file with path %q: %v", p, err)
		}
	}

	// Verify all were stored correctly
	for _, p := range specialPaths {
		got, err := GetRepoFile(db, "proj_rf7", p)
		if err != nil {
			t.Fatalf("failed to get file with path %q: %v", p, err)
		}
		if got.Path != p {
			t.Errorf("expected path %q, got %q", p, got.Path)
		}
	}
}

func TestRepoFileEmptyMetadata(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf8", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	// Empty map metadata
	file := RepoFile{
		ProjectID: "proj_rf8",
		Path:      "test.go",
		Metadata:  map[string]interface{}{},
	}
	result, err := UpsertRepoFile(db, file)
	if err != nil {
		t.Fatalf("UpsertRepoFile with empty metadata failed: %v", err)
	}
	// Empty map serializes to "{}" which should be stored and retrieved
	_ = result
}

func TestRepoFileRichMetadata(t *testing.T) {
	database, cleanup := setupV2TestDB(t)
	defer cleanup()
	db = database

	_, err := db.Exec("INSERT INTO projects (id, name, workdir, created_at) VALUES (?, ?, ?, ?)",
		"proj_rf9", "Test", "/tmp", nowISO())
	if err != nil {
		t.Fatal(err)
	}

	meta := map[string]interface{}{
		"tags":       []interface{}{"important", "reviewed"},
		"complexity": float64(42),
		"nested": map[string]interface{}{
			"inner": "data",
		},
	}

	file := RepoFile{
		ProjectID: "proj_rf9",
		Path:      "complex.go",
		Metadata:  meta,
	}
	_, err = UpsertRepoFile(db, file)
	if err != nil {
		t.Fatalf("UpsertRepoFile with rich metadata failed: %v", err)
	}

	got, err := GetRepoFile(db, "proj_rf9", "complex.go")
	if err != nil {
		t.Fatal(err)
	}

	if got.Metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
	if got.Metadata["complexity"] != float64(42) {
		t.Errorf("expected complexity 42, got %v", got.Metadata["complexity"])
	}
	nested, ok := got.Metadata["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested to be a map")
	}
	if nested["inner"] != "data" {
		t.Errorf("expected nested.inner = data, got %v", nested["inner"])
	}
}
