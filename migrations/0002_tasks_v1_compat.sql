-- 0002_tasks_v1_compat.sql
-- Purpose: confirm v1 baseline schema exists and add safe indexes.
-- Notes:
-- - MUST NOT change v1 behavior.
-- - If the table already exists, these statements are no-ops in SQLite.
-- - We deliberately keep the v1 'status' column and values.

CREATE TABLE IF NOT EXISTS tasks (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  instructions    TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'NOT_PICKED',
  output          TEXT,
  parent_task_id  INTEGER,
  created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- Safe indexes for scale
CREATE INDEX IF NOT EXISTS idx_tasks_status_created_at
  ON tasks(status, created_at);

CREATE INDEX IF NOT EXISTS idx_tasks_parent
  ON tasks(parent_task_id);

-- Recommended (optional) WAL for concurrency; apply from app code:
-- PRAGMA journal_mode=WAL;
-- PRAGMA synchronous=NORMAL;
