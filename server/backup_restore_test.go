package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupBackupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_backup_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	_, _ = testDB.Exec("PRAGMA journal_mode=WAL")
	_, _ = testDB.Exec("PRAGMA busy_timeout=5000")
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

func TestExport_IncludesSchemaVersion(t *testing.T) {
	testDB, cleanup := setupBackupTestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "schema-ver-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/export", project.ID), nil)
	rr := httptest.NewRecorder()
	v2ProjectExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Export: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var archive ProjectArchive
	if err := json.NewDecoder(rr.Body).Decode(&archive); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Version field should be set.
	if archive.Version == "" {
		t.Error("expected non-empty version in export archive")
	}

	// SchemaVersion should be populated.
	if archive.SchemaVersion <= 0 {
		t.Errorf("expected SchemaVersion > 0, got %d", archive.SchemaVersion)
	}
}

func TestImport_ValidatesSchemaCompatibility(t *testing.T) {
	_, cleanup := setupBackupTestDB(t)
	defer cleanup()

	// Create an archive with a future schema version.
	archive := ProjectArchive{
		Version:       "1.0",
		SchemaVersion: 99999,
		Project:       Project{Name: "future-proj"},
		Tasks:         []TaskV2{},
		Memories:      []Memory{},
		Policies:      []Policy{},
	}

	body, _ := json.Marshal(archive)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	v2ProjectImportHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for incompatible schema, got %d: %s", rr.Code, rr.Body.String())
	}

	var errResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	errObj, ok := errResp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["code"] != "SCHEMA_INCOMPATIBLE" {
		t.Errorf("expected SCHEMA_INCOMPATIBLE code, got %v", errObj["code"])
	}
}

func TestImport_AcceptsCompatibleSchema(t *testing.T) {
	testDB, cleanup := setupBackupTestDB(t)
	defer cleanup()

	// Get current schema version.
	var currentVersion int
	err := testDB.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		t.Fatalf("schema version query: %v", err)
	}

	archive := ProjectArchive{
		Version:       "1.0",
		SchemaVersion: currentVersion,
		Project:       Project{Name: "compatible-proj"},
		Tasks:         []TaskV2{},
		Memories:      []Memory{},
		Policies:      []Policy{},
	}

	body, _ := json.Marshal(archive)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	v2ProjectImportHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestImport_SchemaVersionZeroAccepted(t *testing.T) {
	_, cleanup := setupBackupTestDB(t)
	defer cleanup()

	// Schema version 0 means legacy export (no version check).
	archive := ProjectArchive{
		Version:       "1.0",
		SchemaVersion: 0,
		Project:       Project{Name: "legacy-proj"},
		Tasks:         []TaskV2{},
		Memories:      []Memory{},
		Policies:      []Policy{},
	}

	body, _ := json.Marshal(archive)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	v2ProjectImportHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for legacy export, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExportImport_Roundtrip_PreservesData(t *testing.T) {
	testDB, cleanup := setupBackupTestDB(t)
	defer cleanup()

	// Setup: project + tasks + memory + policy.
	project, err := CreateProject(testDB, "roundtrip-proj", "/tmp/rt", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	_, err = CreateTaskV2(testDB, "rt-task-1", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2: %v", err)
	}
	_, err = CreateTaskV2(testDB, "rt-task-2", project.ID, nil)
	if err != nil {
		t.Fatalf("CreateTaskV2 2: %v", err)
	}
	_, err = CreateMemory(testDB, project.ID, "test", "mem-key", map[string]interface{}{"data": "value"}, nil)
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}
	_, err = CreatePolicy(testDB, project.ID, "test-policy", nil, []PolicyRule{{"type": "rate_limit"}}, true)
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	// Export.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/projects/%s/export", project.ID), nil)
	rr := httptest.NewRecorder()
	v2ProjectExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Export: %d: %s", rr.Code, rr.Body.String())
	}

	exportedBytes := rr.Body.Bytes()

	// Import.
	req = httptest.NewRequest(http.MethodPost, "/api/v2/projects/import", bytes.NewReader(exportedBytes))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	v2ProjectImportHandler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Import: %d: %s", rr.Code, rr.Body.String())
	}

	var importResp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&importResp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if int(importResp["tasks_imported"].(float64)) != 2 {
		t.Errorf("expected 2 tasks imported, got %v", importResp["tasks_imported"])
	}
	if int(importResp["memories_imported"].(float64)) != 1 {
		t.Errorf("expected 1 memory imported, got %v", importResp["memories_imported"])
	}
	if int(importResp["policies_imported"].(float64)) != 1 {
		t.Errorf("expected 1 policy imported, got %v", importResp["policies_imported"])
	}
}
