# Schema Migrations System Documentation

## Overview

The Cocopilot application now includes a robust schema migrations system for managing database schema changes in a safe, repeatable, and version-controlled manner.

## Features

- **Versioned Migrations**: Track applied migrations in a `schema_migrations` table
- **Forward-Only Migrations**: SQLite-friendly approach with ascending version numbers
- **Idempotent Operations**: Safe to run migrations multiple times
- **CLI Management**: Easy-to-use command-line interface for migration operations
- **Automatic Application**: Migrations run automatically on server startup
- **Transaction Safety**: Each migration runs in its own transaction (fail-fast on errors)

## Architecture

### Migration Files

Migration files are stored in `migrations/` with the naming convention:

```
<version>_<descriptive_name>.sql
```

Example:
- `0001_schema_migrations.sql`
- `0002_tasks_v1_compat.sql`
- `0003_projects.sql`
- `0004_tasks_add_project_id.sql`
- `0013_task_dependencies.sql`
- `0014_events_project_id_backfill.sql`
- `0015_events_filter_indexes.sql`
- `0016_tasks_sort_indexes.sql`
- `0017_tasks_updated_at.sql`
- `0018_policies.sql`

### Migration Tracking

The `schema_migrations` table tracks which migrations have been applied:

```sql
CREATE TABLE schema_migrations (
  version     INTEGER PRIMARY KEY,
  applied_at  TEXT NOT NULL
);
```

### Migration Engine

The migration engine ([migrations.go](migrations.go)):
1. Ensures `schema_migrations` table exists
2. Loads all migration files from the migrations directory
3. Checks which migrations have been applied
4. Applies pending migrations in ascending version order
5. Records each successful migration in `schema_migrations`
6. Fails fast if any migration encounters an error

## CLI Commands

### Apply Migrations

Apply all pending migrations:

```powershell
.\cocopilot.exe migrate up
```

Or simply:

```powershell
.\cocopilot.exe migrate
```

### Check Migration Status

View which migrations are applied and which are pending:

```powershell
.\cocopilot.exe migrate status
```

Output example:
```
Migration Status
================
Total migrations: 18
Applied: 18
Pending: 0

Version | Status  | Name
--------|---------|-----
0001    | applied | schema_migrations
0002    | applied | tasks_v1_compat
0003    | applied | projects
0004    | applied | tasks_add_project_id
0005    | applied | tasks_v2_enhancements
0006    | applied | runs
0007    | applied | leases
0008    | applied | events
0009    | applied | memory
0010    | applied | context_packs
0011    | applied | tasks_project_fk
0012    | applied | agents
0013    | applied | task_dependencies
0014    | applied | events_project_id_backfill
0015    | applied | events_filter_indexes
0016    | applied | tasks_sort_indexes
0017    | applied | tasks_updated_at
0018    | applied | policies
```

### Rollback Migration

Rollback the most recent migration (tracking only):

```powershell
.\cocopilot.exe migrate down
```

**⚠️ IMPORTANT**: SQLite doesn't support complex schema rollbacks. The `migrate down` command only removes the migration tracking entry. You must manually reverse schema changes if needed.

### Help

Display help information:

```powershell
.\cocopilot.exe help
```

## Server Mode

When you start the server without CLI arguments, migrations are automatically applied on startup:

```powershell
.\cocopilot.exe
```

The server will:
1. Apply any pending migrations
2. Start the web server on http://127.0.0.1:8080

This ensures the database schema is always up to date before serving requests.

## Creating New Migrations

### Step 1: Create Migration File

Create a new SQL file in `migrations/` with the next version number:

```sql
-- 0005_add_agents_table.sql
-- Purpose: Add agents table for tracking connected agents

CREATE TABLE IF NOT EXISTS agents (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  status        TEXT NOT NULL,
  capabilities  TEXT,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_agents_status
  ON agents(status);
```

### Step 2: Test the Migration

Check status to see the new migration:

```powershell
.\cocopilot.exe migrate status
```

Apply the migration:

```powershell
.\cocopilot.exe migrate up
```

Verify it was applied:

```powershell
.\cocopilot.exe migrate status
```

## Best Practices

### 1. Always Use Idempotent SQL

Use `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, etc.:

```sql
-- Good
CREATE TABLE IF NOT EXISTS my_table (
  id INTEGER PRIMARY KEY
);

-- Avoid
CREATE TABLE my_table (
  id INTEGER PRIMARY KEY
);
```

### 2. One Logical Change Per Migration

Keep migrations focused on a single logical change:
- ✅ Good: `0005_add_agents_table.sql`
- ❌ Avoid: `0005_add_agents_and_runs_and_fix_tasks.sql`

### 3. Never Modify Applied Migrations

Once a migration has been applied to production, never modify it. Create a new migration instead.

### 4. Test Migrations Thoroughly

Before committing:
1. Apply the migration on a test database
2. Verify the schema changes
3. Test the application functionality
4. Consider edge cases and existing data

### 5. Document Your Migrations

Include comments in your migration files explaining:
- Purpose of the migration
- Why the change is needed
- Any special considerations
- Impact on existing data

### 6. Use Transactions Wisely

Each migration runs in its own transaction automatically. If you need multiple operations to succeed together, they'll be rolled back as a unit if any fail.

### 7. Handle Data Migrations Carefully

When migrating existing data:

```sql
-- Good: Safe backfill with NULL checks
UPDATE tasks
SET project_id = 'proj_default'
WHERE project_id IS NULL OR project_id = '';

-- Safe: Use NOT EXISTS to avoid duplicates
INSERT INTO projects (id, name, workdir)
SELECT 'proj_default', 'Default', ''
WHERE NOT EXISTS (SELECT 1 FROM projects WHERE id='proj_default');
```

## SQLite-Specific Considerations

### Forward-Only Migrations

SQLite has limited ALTER TABLE support. The migration system uses a forward-only approach:
- Migrations are applied in ascending order
- Rollback only removes the tracking entry, not the schema changes
- For destructive changes, create shadow tables or use application-level logic

### Recommended PRAGMA Settings

Applied automatically on database connection:

```sql
PRAGMA journal_mode=WAL;      -- Better concurrency
PRAGMA synchronous=NORMAL;    -- Good balance of safety/performance
```

Consider adding when foreign keys are used:

```sql
PRAGMA foreign_keys=ON;
```

### ALTER TABLE Limitations

SQLite doesn't support:
- Dropping columns (before SQLite 3.35.0)
- Modifying column types
- Adding constraints to existing columns

Workaround: Create new table, copy data, drop old table, rename new table.

## Troubleshooting

### "Duplicate column name" Error

This happens when:
1. A migration was rolled back (tracking removed)
2. The schema changes weren't manually reversed
3. The migration is applied again

**Solution**: Either manually reverse the schema changes or start with a fresh database for development.

### "Failed to read migrations directory"

Ensure the migrations directory exists:

```powershell
ls migrations\
```

### Migration Fails Midway

Migrations run in transactions and fail fast. If a migration fails:
1. Check the error message
2. Fix the migration file
3. Roll back if needed (tracking only)
4. Apply the corrected migration

### Port Already in Use

If you see "bind: Only one usage of each socket address":

```powershell
# Find process using port 8080
netstat -ano | findstr :8080

# Kill the process (replace PID with actual process ID)
Stop-Process -Id <PID> -Force
```

## Testing the Migration System

### Test Idempotency

```powershell
# Apply migrations
.\cocopilot.exe migrate up

# Apply again - should report "All migrations are up to date"
.\cocopilot.exe migrate up
```

### Test Rollback

```powershell
# Check current status
.\cocopilot.exe migrate status

# Rollback last migration
.\cocopilot.exe migrate down

# Check status again
.\cocopilot.exe migrate status

# Note: Schema changes are still there, only tracking is removed
```

### Test Fresh Database

```powershell
# Remove database
Remove-Item tasks.db

# Apply all migrations
.\cocopilot.exe migrate up

# Verify all migrations applied
.\cocopilot.exe migrate status
```

## Implementation Details

### Code Structure

- **migrations.go**: Migration engine implementation
- **main.go**: CLI command handling and server integration
- **migrations/**: Migration files (workspace root)

### Key Functions

- `runMigrations(db)`: Apply all pending migrations
- `getMigrationStatus(db)`: Show migration status
- `rollbackLastMigration(db)`: Rollback last migration (tracking only)
- `loadMigrations()`: Load migration files from disk
- `applyMigration(db, migration)`: Apply a single migration in a transaction

### Concurrency

- Migrations are protected by a mutex (`migrationMutex`)
- Single-process boot ensures migrations run once
- For multi-process deployments, consider adding file-based locking

## Future Enhancements

Potential improvements documented in the epic:

1. **Down Migrations**: Add explicit down SQL for true rollback support
2. **Migration Generator**: CLI command to generate migration templates
3. **Dry Run**: Preview what migrations would be applied
4. **Seed Data**: Support for seeding test/development data
5. **Migration Dependencies**: Handle branching/merging in version control
6. **Checksum Validation**: Detect if applied migrations have been modified

## V1 Compatibility

The migration system maintains full V1 compatibility:

- Existing `tasks` table schema is preserved
- V1 status values (`NOT_PICKED`, `IN_PROGRESS`, `COMPLETE`) are unchanged
- V1 API endpoints continue to work
- New features (projects, etc.) are additive

## References

- Migration files: [migrations/](migrations/)
- Main application: [main.go](main.go)
- Migration engine: [migrations.go](migrations.go)
