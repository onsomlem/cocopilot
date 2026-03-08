-- 0001_schema_migrations.sql
-- Purpose: track applied migrations (SQLite-first).
-- Notes:
-- - Use a single integer version, applied in ascending order.
-- - This table is never dropped in normal operation.

CREATE TABLE IF NOT EXISTS schema_migrations (
  version     INTEGER PRIMARY KEY,
  applied_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
