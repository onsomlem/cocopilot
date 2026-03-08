# Migration System Quick Reference

## Quick Start

```powershell
# Check what migrations need to be applied
.\cocopilot.exe migrate status

# Apply all pending migrations
.\cocopilot.exe migrate up

# Start server (auto-applies migrations)
.\cocopilot.exe
```

## Common Commands

| Command | Description |
|---------|-------------|
| `.\cocopilot.exe migrate status` | Show which migrations are applied/pending |
| `.\cocopilot.exe migrate up` | Apply all pending migrations |
| `.\cocopilot.exe migrate down` | Rollback last migration (⚠️ tracking only) |
| `.\cocopilot.exe migrate` | Same as `migrate up` |
| `.\cocopilot.exe help` | Show help |
| `.\cocopilot.exe` | Start server (runs migrations first) |

## Creating a New Migration

1. Create file in `migrations/`:
   ```
   0005_your_migration_name.sql
   ```

2. Write idempotent SQL:
   ```sql
   -- 0005_your_migration_name.sql
   -- Purpose: Describe what this migration does
   
   CREATE TABLE IF NOT EXISTS your_table (
     id INTEGER PRIMARY KEY,
     name TEXT NOT NULL
   );
   
   CREATE INDEX IF NOT EXISTS idx_your_table_name
     ON your_table(name);
   ```

3. Test it:
   ```powershell
   .\cocopilot.exe migrate status  # Should show as pending
   .\cocopilot.exe migrate up      # Apply it
   .\cocopilot.exe migrate status  # Should show as applied
   ```

## Best Practices Checklist

- ✅ Use `CREATE TABLE IF NOT EXISTS`
- ✅ Use `CREATE INDEX IF NOT EXISTS`
- ✅ Test on development database first
- ✅ One logical change per migration
- ✅ Include descriptive comments
- ✅ Never modify applied migrations
- ✅ Handle existing data carefully
- ❌ Don't rely on rollback for SQLite
- ❌ Don't use column drops (SQLite limitation)

## Troubleshooting

**"Duplicate column name"**
- Rollback only removes tracking, not schema changes
- Solution: Fresh database or manually fix schema

**"Failed to read migrations directory"**
- Ensure `migrations/` exists
- Check you're in the correct directory

**"Port already in use"**
- Another server is running on port 8080
- Stop it: `Stop-Process -Id <PID> -Force`

## Migration File Locations

**Directory Notes**

- Runtime migrations are loaded from `migrations/` at the repo root.
- New migrations should be added to `migrations/` with the next numeric prefix.

```
migrations/
├── 0001_schema_migrations.sql      # Creates tracking table
├── 0002_tasks_v1_compat.sql        # Tasks table + indexes
├── 0003_projects.sql               # Projects table + default
├── 0004_tasks_add_project_id.sql   # Add project_id to tasks
├── 0005_tasks_v2_enhancements.sql  # v2 task fields and indexes
├── 0006_runs.sql                   # Runs table
├── 0007_leases.sql                 # Leases table
├── 0008_events.sql                 # Events table
├── 0009_memory.sql                 # Memory table
├── 0010_context_packs.sql          # Context packs table
├── 0011_tasks_project_fk.sql       # Project FK constraints
├── 0012_agents.sql                 # Agents table
├── 0013_task_dependencies.sql      # Task dependency graph
├── 0014_events_project_id_backfill.sql # Backfill events.project_id
├── 0015_events_filter_indexes.sql  # Events filter indexes
├── 0016_tasks_sort_indexes.sql     # Task sort indexes
├── 0017_tasks_updated_at.sql       # Ensure tasks.updated_at exists
└── 0018_policies.sql               # Policy engine foundation
```

## Status Output Example

```
Migration Status
================
Total migrations: 18
Applied: 17
Pending: 1

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
0018    | pending | policies
```

## See Full Documentation

For detailed information, see [MIGRATIONS.md](MIGRATIONS.md)

## Critical Warning ⚠️

**SQLite Rollback Limitations**

The `migrate down` command only removes the migration tracking entry from `schema_migrations`. It does NOT reverse schema changes. This is because:

1. SQLite has limited ALTER TABLE support
2. Dropping columns requires complex table rebuilding
3. Forward-only migrations are safer for production

If you need to reverse a migration:
- For development: Delete database and reapply needed migrations
- For production: Create a new migration with the reverse changes
