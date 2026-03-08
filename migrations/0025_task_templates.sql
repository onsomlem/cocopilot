-- 0020_task_templates.sql
-- Purpose: Task template system for reusable task definitions per project

CREATE TABLE IF NOT EXISTS task_templates (
  id               TEXT PRIMARY KEY,
  project_id       TEXT NOT NULL,
  name             TEXT NOT NULL,
  description      TEXT,
  instructions     TEXT NOT NULL DEFAULT '',
  default_type     TEXT,
  default_priority INTEGER NOT NULL DEFAULT 50,
  default_tags     TEXT,
  default_metadata TEXT,
  created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_task_templates_project ON task_templates(project_id);
CREATE INDEX IF NOT EXISTS idx_task_templates_name ON task_templates(project_id, name);
