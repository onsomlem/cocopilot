-- Database Schema Migrations for API v2
-- This file documents all schema additions needed for v2 features
-- Migrations should be applied sequentially using the migration system

-- =============================================================================
-- Migration 0005: Tasks v2 Enhancements
-- Purpose: Add v2 metadata fields to tasks table
-- =============================================================================

-- Add v2 task fields
ALTER TABLE tasks ADD COLUMN title TEXT;
ALTER TABLE tasks ADD COLUMN type TEXT DEFAULT 'MODIFY';
ALTER TABLE tasks ADD COLUMN priority INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN status_v2 TEXT DEFAULT 'QUEUED';
ALTER TABLE tasks ADD COLUMN tags_json TEXT;
ALTER TABLE tasks ADD COLUMN updated_at TEXT;

-- Backfill status_v2 from status (v1) for existing tasks
UPDATE tasks SET status_v2 = 
  CASE status
    WHEN 'NOT_PICKED' THEN 'QUEUED'
    WHEN 'IN_PROGRESS' THEN 'RUNNING'
    WHEN 'COMPLETE' THEN 'SUCCEEDED'
    ELSE 'QUEUED'
  END
WHERE status_v2 IS NULL OR status_v2 = '';

-- Index for v2 queries (status, priority, created_at)
CREATE INDEX IF NOT EXISTS idx_tasks_v2_status_priority 
  ON tasks(project_id, status_v2, priority DESC, created_at ASC);

CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);

-- =============================================================================
-- Migration 0006: Runs (Execution Ledger)
-- Purpose: Track task execution attempts with detailed steps, logs, and artifacts
-- =============================================================================

-- Main runs table: one run per task execution attempt
CREATE TABLE IF NOT EXISTS runs (
  id          TEXT PRIMARY KEY,
  task_id     INTEGER NOT NULL,
  agent_id    TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'RUNNING',
  started_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  finished_at TEXT,
  error       TEXT,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_agent_id ON runs(agent_id);

-- Run steps: track major phases within a run
CREATE TABLE IF NOT EXISTS run_steps (
  id           TEXT PRIMARY KEY,
  run_id       TEXT NOT NULL,
  name         TEXT NOT NULL,
  status       TEXT NOT NULL,
  details_json TEXT,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_steps_run_id ON run_steps(run_id);

-- Run logs: stream stdout/stderr/info logs from agent
CREATE TABLE IF NOT EXISTS run_logs (
  id     INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL,
  stream TEXT NOT NULL,
  chunk  TEXT NOT NULL,
  ts     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_logs_run_id_ts ON run_logs(run_id, ts);

-- Artifacts: attach files produced during execution
CREATE TABLE IF NOT EXISTS artifacts (
  id            TEXT PRIMARY KEY,
  run_id        TEXT NOT NULL,
  kind          TEXT NOT NULL,
  storage_ref   TEXT NOT NULL,
  sha256        TEXT,
  size          INTEGER,
  metadata_json TEXT,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifacts_run_id ON artifacts(run_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_kind ON artifacts(kind);

-- Tool invocations: track which tools were called during execution
CREATE TABLE IF NOT EXISTS tool_invocations (
  id          TEXT PRIMARY KEY,
  run_id      TEXT NOT NULL,
  tool_name   TEXT NOT NULL,
  input_json  TEXT,
  output_json TEXT,
  started_at  TEXT NOT NULL,
  finished_at TEXT,
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tool_invocations_run_id ON tool_invocations(run_id);
CREATE INDEX IF NOT EXISTS idx_tool_invocations_tool_name ON tool_invocations(tool_name);

-- =============================================================================
-- Migration 0007: Leases (Agent Coordination)
-- Purpose: Enable exclusive task claiming with expiration and heartbeat
-- =============================================================================

CREATE TABLE IF NOT EXISTS leases (
  id         TEXT PRIMARY KEY,
  task_id    INTEGER NOT NULL,
  agent_id   TEXT NOT NULL,
  mode       TEXT NOT NULL DEFAULT 'exclusive',
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  expires_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Unique constraint: only one active lease per task
CREATE UNIQUE INDEX IF NOT EXISTS idx_leases_task_id ON leases(task_id);

-- Index for expiration cleanup job
CREATE INDEX IF NOT EXISTS idx_leases_expires_at ON leases(expires_at);

-- Index for agent-centric queries
CREATE INDEX IF NOT EXISTS idx_leases_agent_id ON leases(agent_id);

-- =============================================================================
-- Migration 0008: Events (Real-time Notifications)
-- Purpose: Store events for SSE replay and project-scoped notifications
-- =============================================================================

CREATE TABLE IF NOT EXISTS events (
  id           TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL,
  kind         TEXT NOT NULL,
  entity_type  TEXT NOT NULL,
  entity_id    TEXT NOT NULL,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  payload_json TEXT,
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Index for project-scoped event queries and SSE
CREATE INDEX IF NOT EXISTS idx_events_project_created 
  ON events(project_id, created_at);

-- Index for filtering by event kind
CREATE INDEX IF NOT EXISTS idx_events_kind ON events(kind);

-- Index for entity lookup
CREATE INDEX IF NOT EXISTS idx_events_entity 
  ON events(entity_type, entity_id);

-- =============================================================================
-- Migration 0009: Memory (Persistent Knowledge Base)
-- Purpose: Store project-level knowledge that accumulates across tasks
-- =============================================================================

CREATE TABLE IF NOT EXISTS memory (
  id               TEXT PRIMARY KEY,
  project_id       TEXT NOT NULL,
  scope            TEXT NOT NULL,
  key              TEXT NOT NULL,
  value_json       TEXT NOT NULL,
  source_refs_json TEXT,
  created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Unique constraint: one memory item per (project, scope, key)
CREATE UNIQUE INDEX IF NOT EXISTS idx_memory_project_scope_key 
  ON memory(project_id, scope, key);

-- Index for scope-based queries
CREATE INDEX IF NOT EXISTS idx_memory_scope ON memory(scope);

-- Index for full-text search (if needed, can add FTS5 virtual table)
CREATE INDEX IF NOT EXISTS idx_memory_updated_at ON memory(updated_at);

-- =============================================================================
-- Migration 0010: Context Packs (Automated Context Generation)
-- Purpose: Store pre-generated context bundles for tasks
-- =============================================================================

CREATE TABLE IF NOT EXISTS context_packs (
  id            TEXT PRIMARY KEY,
  project_id    TEXT NOT NULL,
  task_id       INTEGER NOT NULL,
  summary       TEXT NOT NULL,
  contents_json TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Index for task-centric queries (get context pack for task)
CREATE INDEX IF NOT EXISTS idx_context_packs_task_id ON context_packs(task_id);

-- Index for project-level queries
CREATE INDEX IF NOT EXISTS idx_context_packs_project_id ON context_packs(project_id);

-- =============================================================================
-- Migration 0011: Tasks project foreign key
-- Purpose: Enforce tasks.project_id with a foreign key and NOT NULL constraint
-- =============================================================================

-- SQLite requires table recreation to add foreign key constraints
CREATE TABLE tasks_new (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  instructions     TEXT NOT NULL,
  status           TEXT NOT NULL DEFAULT 'NOT_PICKED',
  output           TEXT,
  parent_task_id   INTEGER,
  created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  title            TEXT,
  type             TEXT,
  priority         INTEGER DEFAULT 50,
  tags_json        TEXT,
  status_v2        TEXT DEFAULT 'QUEUED',
  updated_at       TEXT,
  project_id       TEXT NOT NULL DEFAULT 'proj_default',
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_task_id) REFERENCES tasks(id)
);

INSERT INTO tasks_new
  SELECT id, instructions, status, output, parent_task_id, created_at,
         title, type, priority, tags_json, status_v2, updated_at,
         COALESCE(project_id, 'proj_default')
  FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX IF NOT EXISTS idx_tasks_project_status_created_at
  ON tasks(project_id, status, created_at);

CREATE INDEX IF NOT EXISTS idx_tasks_status_v2
  ON tasks(status_v2);

CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id
  ON tasks(parent_task_id);

-- =============================================================================
-- Migration 0012: Agents (Agent Registration)
-- Purpose: Track registered agents with capabilities and status
-- =============================================================================

CREATE TABLE IF NOT EXISTS agents (
  id                 TEXT PRIMARY KEY,
  name               TEXT NOT NULL,
  capabilities_json  TEXT,
  metadata_json      TEXT,
  status             TEXT NOT NULL DEFAULT 'OFFLINE',
  last_seen          TEXT,
  registered_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_registered_at ON agents(registered_at);
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);

INSERT OR IGNORE INTO agents (id, name, status, registered_at)
VALUES ('agent_default', 'Default Agent', 'OFFLINE', strftime('%Y-%m-%dT%H:%M:%fZ','now'));

-- =============================================================================
-- Migration 0013: Task dependencies
-- Purpose: Track dependencies between tasks
-- =============================================================================

CREATE TABLE IF NOT EXISTS task_dependencies (
  task_id INTEGER NOT NULL,
  depends_on_task_id INTEGER NOT NULL,
  PRIMARY KEY (task_id, depends_on_task_id),
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
  FOREIGN KEY (depends_on_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
  CHECK (task_id <> depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_dependencies_task_id
  ON task_dependencies(task_id);

CREATE INDEX IF NOT EXISTS idx_task_dependencies_depends_on
  ON task_dependencies(depends_on_task_id);

-- =============================================================================
-- Migration 0014: Events project_id backfill
-- Purpose: Ensure project_id is populated for legacy events
-- =============================================================================

UPDATE events
SET project_id = 'proj_default'
WHERE project_id IS NULL OR TRIM(project_id) = '';

-- =============================================================================
-- Migration 0015: Events filter indexes
-- Purpose: Add indexes to speed up project, task, and kind filtering
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_events_project_id
  ON events(project_id);

CREATE INDEX IF NOT EXISTS idx_events_task_id_created
  ON events(entity_type, entity_id, created_at);

CREATE INDEX IF NOT EXISTS idx_events_project_kind_created
  ON events(project_id, kind, created_at);

-- =============================================================================
-- Migration 0016: Task sort indexes
-- Purpose: Add indexes for task list sorting and filtering
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_tasks_created_at
  ON tasks(created_at);

CREATE INDEX IF NOT EXISTS idx_tasks_updated_at
  ON tasks(updated_at);

CREATE INDEX IF NOT EXISTS idx_tasks_project_created_at
  ON tasks(project_id, created_at);

CREATE INDEX IF NOT EXISTS idx_tasks_project_updated_at
  ON tasks(project_id, updated_at);

CREATE INDEX IF NOT EXISTS idx_tasks_status_updated_at
  ON tasks(status, updated_at);

CREATE INDEX IF NOT EXISTS idx_tasks_status_v2_updated_at
  ON tasks(status_v2, updated_at);

CREATE INDEX IF NOT EXISTS idx_tasks_project_status_updated_at
  ON tasks(project_id, status, updated_at);

CREATE INDEX IF NOT EXISTS idx_tasks_project_status_v2_updated_at
  ON tasks(project_id, status_v2, updated_at);

-- =============================================================================
-- Migration 0017: Tasks updated_at backfill
-- Purpose: Ensure tasks.updated_at exists and populate when missing
-- =============================================================================

ALTER TABLE tasks ADD COLUMN updated_at TEXT;

UPDATE tasks
SET updated_at = created_at
WHERE updated_at IS NULL OR updated_at = '';

-- =============================================================================
-- Migration 0018: Policies (Policy Engine Foundation)
-- Purpose: Store project-scoped policies and rules
-- =============================================================================

CREATE TABLE IF NOT EXISTS policies (
  id           TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL,
  name         TEXT NOT NULL,
  description  TEXT,
  rules_json   TEXT NOT NULL,
  enabled      INTEGER NOT NULL DEFAULT 1,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_policies_project_id ON policies(project_id);
CREATE INDEX IF NOT EXISTS idx_policies_project_name ON policies(project_id, name);

-- =============================================================================
-- Migration Notes
-- =============================================================================

-- Migration Order:
-- 1. 0001_schema_migrations.sql (already applied)
-- 2. 0002_tasks_v1_compat.sql (already applied)
-- 3. 0003_projects.sql (already applied)
-- 4. 0004_tasks_add_project_id.sql (already applied)
-- 5. 0005 (tasks v2 enhancements) - THIS FILE
-- 6. 0006 (runs ledger) - THIS FILE
-- 7. 0007 (leases) - THIS FILE
-- 8. 0008 (events) - THIS FILE
-- 9. 0009 (memory) - THIS FILE
-- 10. 0010 (context_packs) - THIS FILE
-- 11. 0011 (tasks project foreign key) - THIS FILE
-- 12. 0012 (agents) - THIS FILE
-- 13. 0013 (task dependencies) - THIS FILE
-- 14. 0014 (events project_id backfill) - THIS FILE
-- 15. 0015 (events filter indexes) - THIS FILE
-- 16. 0016 (task sort indexes) - THIS FILE
-- 17. 0017 (tasks updated_at backfill) - THIS FILE
-- 18. 0018 (policies) - THIS FILE

-- ID Generation:
-- - Use UUIDs or ULID for run_id, lease_id, event_id, etc.
-- - Go code: github.com/google/uuid or github.com/oklog/ulid
-- - Format: "run_" prefix + UUID (~36 chars) or "run_" + ULID (26 chars)

-- Cascading Deletes:
-- - All foreign keys use ON DELETE CASCADE for cleanup
-- - Deleting a task removes all runs, leases, context_packs
-- - Deleting a project removes all associated data

-- JSON Columns:
-- - SQLite stores JSON as TEXT
-- - Use json_extract() for queries: json_extract(tags_json, '$.tag1')
-- - Consider json_each() for array iteration
-- - Go marshals/unmarshals with encoding/json

-- Timestamps:
-- - ISO8601 format: 2026-02-11T10:30:00.000Z
-- - SQLite function: strftime('%Y-%m-%dT%H:%M:%fZ','now')
-- - Go: time.Now().UTC().Format(time.RFC3339Nano)

-- Status Values:
-- v1 status: NOT_PICKED, IN_PROGRESS, COMPLETE
-- v2 status: QUEUED, CLAIMED, RUNNING, SUCCEEDED, FAILED, NEEDS_REVIEW, CANCELLED
-- Run status: RUNNING, SUCCEEDED, FAILED, CANCELLED
-- Step status: STARTED, SUCCEEDED, FAILED

-- Indexes:
-- - Covering indexes for common queries
-- - Compound indexes ordered by selectivity
-- - TEXT columns use case-insensitive collation by default in SQLite

-- Performance:
-- - SQLite WAL mode (PRAGMA journal_mode=WAL) for concurrency
-- - PRAGMA synchronous=NORMAL for performance
-- - Vacuum periodically to reclaim space
-- - Consider auto_vacuum=INCREMENTAL for large datasets

-- =============================================================================
-- End of Schema Additions
-- =============================================================================
