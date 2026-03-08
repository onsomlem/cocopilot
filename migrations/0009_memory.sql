-- Migration 0009: Memory (Persistent Knowledge Base)
-- Purpose: Store project-level knowledge that accumulates across tasks

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
