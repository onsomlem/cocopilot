-- 0021_repo_files.sql
-- Purpose: Add repo_files table for file metadata persistence

CREATE TABLE IF NOT EXISTS repo_files (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  path TEXT NOT NULL,
  content_hash TEXT,
  size_bytes INTEGER,
  language TEXT,
  last_modified TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  metadata_json TEXT,
  UNIQUE(project_id, path)
);

CREATE INDEX IF NOT EXISTS idx_repo_files_project_path ON repo_files(project_id, path);
CREATE INDEX IF NOT EXISTS idx_repo_files_language ON repo_files(project_id, language);
