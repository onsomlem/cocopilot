// Package migrate provides SQLite schema migration management.
package migrate

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Migration represents a single database migration.
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

var mu sync.Mutex

// MigrationsDir is the path to the migrations directory.
var MigrationsDir = filepath.Join(".", "migrations")

// EnsureTable creates the schema_migrations tracking table if it doesn't exist.
func EnsureTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     INTEGER PRIMARY KEY,
			applied_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)
	`)
	return err
}

// AppliedVersions returns a map of applied migration versions.
func AppliedVersions(db *sql.DB) (map[int]bool, error) {
	applied := make(map[int]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, nil
}

// LoadAll reads all migration files from MigrationsDir.
func LoadAll() ([]Migration, error) {
	files, err := os.ReadDir(MigrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %v", err)
	}
	var migrations []Migration
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(file.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(MigrationsDir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %v", file.Name(), err)
		}
		name := strings.TrimSuffix(parts[1], ".sql")
		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			UpSQL:   string(content),
			DownSQL: "",
		})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

// Apply applies a single migration within a transaction.
func Apply(db *sql.DB, migration Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()
	statements := SplitSQLStatements(migration.UpSQL)
	for _, statement := range statements {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			if migration.Version == 17 && strings.Contains(err.Error(), "duplicate column name: updated_at") &&
				strings.Contains(strings.ToLower(stmt), "alter table tasks add column updated_at") {
				continue
			}
			return fmt.Errorf("failed to execute migration %04d: %v", migration.Version, err)
		}
	}
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
		migration.Version,
		time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	); err != nil {
		return fmt.Errorf("failed to record migration %04d: %v", migration.Version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %04d: %v", migration.Version, err)
	}
	return nil
}

// SplitSQLStatements splits a SQL text into individual statements.
func SplitSQLStatements(sqlText string) []string {
	var cleaned []string
	scanner := bufio.NewScanner(strings.NewReader(sqlText))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		cleaned = append(cleaned, line)
	}
	return strings.Split(strings.Join(cleaned, "\n"), ";")
}

// Run applies all pending migrations.
func Run(db *sql.DB) error {
	mu.Lock()
	defer mu.Unlock()
	if err := EnsureTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %v", err)
	}
	applied, err := AppliedVersions(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %v", err)
	}
	migrations, err := LoadAll()
	if err != nil {
		return err
	}
	appliedCount := 0
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		fmt.Printf("Applying migration %04d: %s\n", migration.Version, migration.Name)
		if err := Apply(db, migration); err != nil {
			return err
		}
		appliedCount++
	}
	if appliedCount == 0 {
		fmt.Println("All migrations are up to date")
	} else {
		fmt.Printf("Successfully applied %d migration(s)\n", appliedCount)
	}
	return nil
}

// Status prints the current migration status.
func Status(db *sql.DB) error {
	mu.Lock()
	defer mu.Unlock()
	if err := EnsureTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %v", err)
	}
	applied, err := AppliedVersions(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %v", err)
	}
	migrations, err := LoadAll()
	if err != nil {
		return err
	}
	fmt.Println("Migration Status")
	fmt.Println("================")
	fmt.Printf("Total migrations: %d\n", len(migrations))
	fmt.Printf("Applied: %d\n", len(applied))
	fmt.Printf("Pending: %d\n\n", len(migrations)-len(applied))
	if len(migrations) == 0 {
		fmt.Println("No migrations found in migrations directory")
		return nil
	}
	fmt.Println("Version | Status  | Name")
	fmt.Println("--------|---------|-----")
	for _, migration := range migrations {
		status := "pending"
		if applied[migration.Version] {
			status = "applied"
		}
		fmt.Printf("%04d    | %-7s | %s\n", migration.Version, status, migration.Name)
	}
	return nil
}

// Rollback removes the tracking entry for the most recent migration.
func Rollback(db *sql.DB) error {
	mu.Lock()
	defer mu.Unlock()
	if err := EnsureTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %v", err)
	}
	var version int
	err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err == sql.ErrNoRows || version == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get latest migration: %v", err)
	}
	fmt.Printf("Rolling back migration %04d\n", version)
	fmt.Println("WARNING: SQLite doesn't support complex schema rollbacks.")
	fmt.Println("This will only remove the migration tracking entry.")
	fmt.Println("You must manually reverse schema changes if needed.")
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version); err != nil {
		return fmt.Errorf("failed to remove migration %04d: %v", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback %04d: %v", version, err)
	}
	fmt.Printf("Successfully rolled back migration %04d (tracking only)\n", version)
	return nil
}
