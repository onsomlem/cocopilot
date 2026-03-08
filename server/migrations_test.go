package server

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrationSystem(t *testing.T) {
	// Setup: Remove test database if it exists
	testDB := "test_migrations.db"
	defer os.Remove(testDB)

	t.Run("EnsureSchemaMigrationsTable", func(t *testing.T) {
		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Ensure schema_migrations table can be created
		if err := ensureSchemaMigrationsTable(db); err != nil {
			t.Fatalf("Failed to create schema_migrations table: %v", err)
		}

		// Verify table exists
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		if err != nil {
			t.Fatalf("schema_migrations table doesn't exist: %v", err)
		}

		// Should be empty initially
		if count != 0 {
			t.Errorf("Expected 0 migrations, got %d", count)
		}
	})

	t.Run("LoadMigrations", func(t *testing.T) {
		migrations, err := loadMigrations()
		if err != nil {
			t.Fatalf("Failed to load migrations: %v", err)
		}

		if len(migrations) == 0 {
			t.Error("No migrations found, expected at least 4")
		}

		// Verify migrations are sorted
		for i := 0; i < len(migrations)-1; i++ {
			if migrations[i].Version >= migrations[i+1].Version {
				t.Errorf("Migrations not sorted: %d >= %d", migrations[i].Version, migrations[i+1].Version)
			}
		}

		// Verify first migration is schema_migrations
		if len(migrations) > 0 && migrations[0].Version != 1 {
			t.Errorf("Expected first migration to be version 1, got %d", migrations[0].Version)
		}
	})

	t.Run("ApplyMigrations", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Apply migrations
		if err := runMigrations(db); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		// Verify migrations were applied
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query schema_migrations: %v", err)
		}

		if count == 0 {
			t.Error("No migrations were applied")
		}

		// Verify expected tables exist
		tables := []string{"schema_migrations", "tasks", "projects", "task_dependencies", "policies"}
		for _, table := range tables {
			var exists int
			query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
			err = db.QueryRow(query, table).Scan(&exists)
			if err != nil || exists == 0 {
				t.Errorf("Expected table %s to exist", table)
			}
		}

		// Verify policies.rules_json column exists
		var hasRulesJSON bool
		policyRows, err := db.Query("PRAGMA table_info(policies)")
		if err != nil {
			t.Fatalf("Failed to read policies table info: %v", err)
		}
		defer policyRows.Close()

		for policyRows.Next() {
			var cid int
			var name string
			var columnType string
			var notNull int
			var defaultValue sql.NullString
			var pk int
			if err := policyRows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
				t.Fatalf("Failed to scan policies column info: %v", err)
			}
			if name == "rules_json" {
				hasRulesJSON = true
				break
			}
		}
		if err := policyRows.Err(); err != nil {
			t.Fatalf("Failed to iterate policies table info: %v", err)
		}
		if !hasRulesJSON {
			t.Error("Expected policies.rules_json column to exist")
		}

		// Verify events.project_id column exists
		var hasProjectID bool
		rows, err := db.Query("PRAGMA table_info(events)")
		if err != nil {
			t.Fatalf("Failed to read events table info: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name string
			var columnType string
			var notNull int
			var defaultValue sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
				t.Fatalf("Failed to scan events column info: %v", err)
			}
			if name == "project_id" {
				hasProjectID = true
				break
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Failed to iterate events table info: %v", err)
		}
		if !hasProjectID {
			t.Error("Expected events.project_id column to exist")
		}

		// Verify tasks.updated_at column exists
		var hasUpdatedAt bool
		taskRows, err := db.Query("PRAGMA table_info(tasks)")
		if err != nil {
			t.Fatalf("Failed to read tasks table info: %v", err)
		}
		defer taskRows.Close()

		for taskRows.Next() {
			var cid int
			var name string
			var columnType string
			var notNull int
			var defaultValue sql.NullString
			var pk int
			if err := taskRows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
				t.Fatalf("Failed to scan tasks column info: %v", err)
			}
			if name == "updated_at" {
				hasUpdatedAt = true
				break
			}
		}
		if err := taskRows.Err(); err != nil {
			t.Fatalf("Failed to iterate tasks table info: %v", err)
		}
		if !hasUpdatedAt {
			t.Error("Expected tasks.updated_at column to exist")
		}
	})

	t.Run("EventsFilterIndexes", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		if err := runMigrations(db); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		rows, err := db.Query("PRAGMA index_list(events)")
		if err != nil {
			t.Fatalf("Failed to list event indexes: %v", err)
		}
		defer rows.Close()

		indexNames := make(map[string]struct{})
		for rows.Next() {
			var seq int
			var name string
			var unique int
			var origin string
			var partial int
			if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				t.Fatalf("Failed to scan event index: %v", err)
			}
			indexNames[name] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Failed to iterate event indexes: %v", err)
		}

		requiredIndexes := []string{
			"idx_events_project_id",
			"idx_events_task_id_created",
			"idx_events_project_kind_created",
		}
		for _, name := range requiredIndexes {
			if _, ok := indexNames[name]; !ok {
				t.Errorf("Expected events index %s to exist", name)
			}
		}
	})

	t.Run("TasksSortIndexes", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		if err := runMigrations(db); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		rows, err := db.Query("PRAGMA index_list(tasks)")
		if err != nil {
			t.Fatalf("Failed to list task indexes: %v", err)
		}
		defer rows.Close()

		indexNames := make(map[string]struct{})
		for rows.Next() {
			var seq int
			var name string
			var unique int
			var origin string
			var partial int
			if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				t.Fatalf("Failed to scan task index: %v", err)
			}
			indexNames[name] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Failed to iterate task indexes: %v", err)
		}

		requiredIndexes := []string{
			"idx_tasks_created_at",
			"idx_tasks_updated_at",
			"idx_tasks_project_created_at",
			"idx_tasks_project_updated_at",
			"idx_tasks_status_updated_at",
			"idx_tasks_status_v2_updated_at",
			"idx_tasks_project_status_updated_at",
			"idx_tasks_project_status_v2_updated_at",
		}
		for _, name := range requiredIndexes {
			if _, ok := indexNames[name]; !ok {
				t.Errorf("Expected tasks index %s to exist", name)
			}
		}
	})

	t.Run("IdempotentMigrations", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Apply migrations first time
		if err := runMigrations(db); err != nil {
			t.Fatalf("Failed to run migrations first time: %v", err)
		}

		// Get count after first run
		var count1 int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count1)

		// Apply migrations second time - should be idempotent
		if err := runMigrations(db); err != nil {
			t.Fatalf("Failed to run migrations second time: %v", err)
		}

		// Get count after second run
		var count2 int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count2)

		if count1 != count2 {
			t.Errorf("Migrations not idempotent: first=%d, second=%d", count1, count2)
		}
	})

	t.Run("GetAppliedMigrations", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Setup
		ensureSchemaMigrationsTable(db)

		// Initially should be empty
		applied, err := getAppliedMigrations(db)
		if err != nil {
			t.Fatalf("Failed to get applied migrations: %v", err)
		}
		if len(applied) != 0 {
			t.Errorf("Expected 0 applied migrations, got %d", len(applied))
		}

		// Apply migrations
		runMigrations(db)

		// Should have some applied now
		applied, err = getAppliedMigrations(db)
		if err != nil {
			t.Fatalf("Failed to get applied migrations: %v", err)
		}
		if len(applied) == 0 {
			t.Error("Expected some applied migrations")
		}
	})

	t.Run("MigrationWithProjects", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Apply migrations
		if err := runMigrations(db); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		// Verify default project was seeded
		var projectID string
		err = db.QueryRow("SELECT id FROM projects WHERE id='proj_default'").Scan(&projectID)
		if err != nil {
			t.Fatalf("Default project not found: %v", err)
		}

		if projectID != "proj_default" {
			t.Errorf("Expected project ID 'proj_default', got '%s'", projectID)
		}
	})

	t.Run("EventsProjectIDBackfill", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		if err := ensureSchemaMigrationsTable(db); err != nil {
			t.Fatalf("Failed to create schema_migrations table: %v", err)
		}

		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS projects (
			  id TEXT PRIMARY KEY,
			  name TEXT NOT NULL,
			  workdir TEXT NOT NULL DEFAULT ''
			);
			CREATE TABLE IF NOT EXISTS events (
			  id TEXT PRIMARY KEY,
			  project_id TEXT,
			  kind TEXT NOT NULL,
			  entity_type TEXT NOT NULL,
			  entity_id TEXT NOT NULL,
			  created_at TEXT NOT NULL,
			  payload_json TEXT
			);
			INSERT INTO projects (id, name, workdir)
			VALUES ('proj_default', 'Default', '');
			INSERT INTO events (id, project_id, kind, entity_type, entity_id, created_at)
			VALUES
			  ('evt_null', NULL, 'test', 'task', 't1', '2024-01-01T00:00:00Z'),
			  ('evt_empty', '', 'test', 'task', 't2', '2024-01-01T00:00:00Z'),
			  ('evt_keep', 'proj_keep', 'test', 'task', 't3', '2024-01-01T00:00:00Z');
		`)
		if err != nil {
			t.Fatalf("Failed to seed legacy events: %v", err)
		}

		migrations, err := loadMigrations()
		if err != nil {
			t.Fatalf("Failed to load migrations: %v", err)
		}

		var backfill Migration
		found := false
		for _, migration := range migrations {
			if migration.Version == 14 {
				backfill = migration
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Migration 0014 not found")
		}

		if err := applyMigration(db, backfill); err != nil {
			t.Fatalf("Failed to apply migration 0014: %v", err)
		}

		var backfilled int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM events WHERE id IN ('evt_null', 'evt_empty') AND project_id = 'proj_default'",
		).Scan(&backfilled)
		if err != nil {
			t.Fatalf("Failed to verify backfilled events: %v", err)
		}
		if backfilled != 2 {
			t.Errorf("Expected 2 events to be backfilled, got %d", backfilled)
		}

		var keepProjectID string
		if err := db.QueryRow(
			"SELECT project_id FROM events WHERE id = 'evt_keep'",
		).Scan(&keepProjectID); err != nil {
			t.Fatalf("Failed to load preserved event: %v", err)
		}
		if keepProjectID != "proj_keep" {
			t.Errorf("Expected preserved project_id 'proj_keep', got '%s'", keepProjectID)
		}
	})

	t.Run("RollbackMigration", func(t *testing.T) {
		os.Remove(testDB) // Fresh start

		db, err := sql.Open("sqlite", testDB)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Apply migrations
		if err := runMigrations(db); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		// Get initial count
		var countBefore int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&countBefore)

		if countBefore == 0 {
			t.Skip("No migrations to rollback")
		}

		// Rollback last migration
		if err := rollbackLastMigration(db); err != nil {
			t.Fatalf("Failed to rollback migration: %v", err)
		}

		// Verify count decreased
		var countAfter int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&countAfter)

		if countAfter != countBefore-1 {
			t.Errorf("Rollback didn't work: before=%d, after=%d", countBefore, countAfter)
		}
	})
}

func TestMigrationsDir(t *testing.T) {
	// Verify migrations directory exists
	migrationsDir := getMigrationsDir()
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		t.Fatalf("Migrations directory doesn't exist: %s", migrationsDir)
	}

	// Verify it contains SQL files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	sqlCount := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			sqlCount++
		}
	}

	if sqlCount == 0 {
		t.Error("No .sql migration files found")
	}

	t.Logf("Found %d migration files", sqlCount)
}

func TestMigrationFileNaming(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	for _, migration := range migrations {
		// Verify version is positive
		if migration.Version <= 0 {
			t.Errorf("Migration has invalid version: %d", migration.Version)
		}

		// Verify name is not empty
		if migration.Name == "" {
			t.Errorf("Migration %d has empty name", migration.Version)
		}

		// Verify SQL is not empty
		if migration.UpSQL == "" {
			t.Errorf("Migration %d has empty SQL", migration.Version)
		}

		t.Logf("Migration %04d: %s", migration.Version, migration.Name)
	}
}
