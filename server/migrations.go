// Thin delegation layer — all migration logic lives in internal/migrate.
package server

import (
	"database/sql"

	"github.com/onsomlem/cocopilot/internal/migrate"
)

// Migration re-exports the internal type for backward compatibility.
type Migration = migrate.Migration

func runMigrations(db *sql.DB) error          { return migrate.Run(db) }
func getMigrationStatus(db *sql.DB) error     { return migrate.Status(db) }
func rollbackLastMigration(db *sql.DB) error  { return migrate.Rollback(db) }
func ensureSchemaMigrationsTable(db *sql.DB) error { return migrate.EnsureTable(db) }
func loadMigrations() ([]migrate.Migration, error) { return migrate.LoadAll() }
func getMigrationsDir() string                { return migrate.MigrationsDir }
func getAppliedMigrations(db *sql.DB) (map[int]bool, error) { return migrate.AppliedVersions(db) }
func applyMigration(db *sql.DB, m migrate.Migration) error { return migrate.Apply(db, m) }
