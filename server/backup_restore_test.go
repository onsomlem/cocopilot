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

func setupBackupTestDB(t *testing.T) (*sql.DB, string, func()) {
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
	return testDB, testDBPath, cleanup
}

func TestExport_IncludesSchemaVersion(t *testing.T) {
	testDB, _, cleanup := setupBackupTestDB(t)
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
	_, _, cleanup := setupBackupTestDB(t)
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
	testDB, _, cleanup := setupBackupTestDB(t)
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
	_, _, cleanup := setupBackupTestDB(t)
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
	testDB, _, cleanup := setupBackupTestDB(t)
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

// --- v2BackupHandler tests ---

func TestV2BackupHandler_Success(t *testing.T) {
	testDB, dbPath, cleanup := setupBackupTestDB(t)
	defer cleanup()

	// Insert some data to back up.
	_, err := CreateProject(testDB, "backup-proj", "/tmp/b", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	cfg := runtimeConfig{DBPath: dbPath}
	handler := v2BackupHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/backup", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("backup: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("expected Content-Type application/octet-stream, got %s", ct)
	}

	cd := rr.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header")
	}

	// Body should start with SQLite magic bytes.
	body := rr.Body.Bytes()
	if len(body) < 16 {
		t.Fatalf("backup too small: %d bytes", len(body))
	}
	if string(body[:15]) != "SQLite format 3" {
		t.Errorf("backup doesn't start with SQLite header: %q", string(body[:16]))
	}
}

func TestV2BackupHandler_WrongMethod(t *testing.T) {
	_, _, cleanup := setupBackupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{DBPath: "/nonexistent"}
	handler := v2BackupHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/backup", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestV2BackupHandler_MissingDB(t *testing.T) {
	_, _, cleanup := setupBackupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{DBPath: "/tmp/nonexistent_db_" + fmt.Sprintf("%d", time.Now().UnixNano())}
	handler := v2BackupHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/backup", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for missing DB, got %d", rr.Code)
	}
}

// --- v2RestoreHandler tests ---

func TestV2RestoreHandler_WrongMethod(t *testing.T) {
	_, _, cleanup := setupBackupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{DBPath: "/tmp/test.db"}
	handler := v2RestoreHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/restore", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestV2RestoreHandler_TooSmall(t *testing.T) {
	_, _, cleanup := setupBackupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{DBPath: filepath.Join(t.TempDir(), "restore.db")}
	handler := v2RestoreHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/restore", bytes.NewReader([]byte("too small")))
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestV2RestoreHandler_InvalidFormat(t *testing.T) {
	_, _, cleanup := setupBackupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{DBPath: filepath.Join(t.TempDir(), "restore.db")}
	handler := v2RestoreHandler(cfg)

	// 32 bytes of non-SQLite data.
	fakeData := bytes.Repeat([]byte("x"), 32)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/restore", bytes.NewReader(fakeData))
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid format, got %d", rr.Code)
	}

	var errResp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&errResp)
	errObj, _ := errResp["error"].(map[string]interface{})
	if errObj["code"] != "INVALID_FORMAT" {
		t.Errorf("expected INVALID_FORMAT code, got %v", errObj["code"])
	}
}

func TestV2RestoreHandler_ValidSQLite(t *testing.T) {
	testDB, dbPath, cleanup := setupBackupTestDB(t)
	defer cleanup()

	// Read the actual test SQLite DB to use as restore payload.
	// Force WAL checkpoint so data is in main file.
	testDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	dbContent, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read DB: %v", err)
	}

	restorePath := filepath.Join(t.TempDir(), "restored.db")
	cfg := runtimeConfig{DBPath: restorePath}
	handler := v2RestoreHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/restore", bytes.NewReader(dbContent))
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["restored"] != true {
		t.Errorf("expected restored=true, got %v", resp["restored"])
	}
	if resp["message"] == nil {
		t.Error("expected message in response")
	}

	// Verify restored file exists and is valid SQLite.
	restoredContent, err := os.ReadFile(restorePath)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if string(restoredContent[:15]) != "SQLite format 3" {
		t.Error("restored file is not valid SQLite")
	}
}

func TestV2BackupRestore_Roundtrip(t *testing.T) {
	testDB, dbPath, cleanup := setupBackupTestDB(t)
	defer cleanup()

	// Seed data.
	project, _ := CreateProject(testDB, "roundtrip-proj", "/tmp/rt", nil)
	CreateTaskV2(testDB, "task-alpha", project.ID, nil)
	CreateTaskV2(testDB, "task-beta", project.ID, nil)

	// Checkpoint WAL so backup gets full data.
	testDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	// Backup.
	cfg := runtimeConfig{DBPath: dbPath}
	backupHandler := v2BackupHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v2/backup", nil)
	rr := httptest.NewRecorder()
	backupHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("backup: %d", rr.Code)
	}
	backupData := rr.Body.Bytes()

	// Restore to new path.
	restorePath := filepath.Join(t.TempDir(), "roundtrip_restore.db")
	restoreCfg := runtimeConfig{DBPath: restorePath}
	restoreHandler := v2RestoreHandler(restoreCfg)
	req = httptest.NewRequest(http.MethodPost, "/api/v2/restore", bytes.NewReader(backupData))
	rr = httptest.NewRecorder()
	restoreHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("restore: %d: %s", rr.Code, rr.Body.String())
	}

	// Open restored DB and verify data.
	restoredDB, err := sql.Open("sqlite", restorePath)
	if err != nil {
		t.Fatalf("open restored: %v", err)
	}
	defer restoredDB.Close()

	var taskCount int
	restoredDB.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
	if taskCount != 2 {
		t.Errorf("expected 2 tasks in restored DB, got %d", taskCount)
	}
}
