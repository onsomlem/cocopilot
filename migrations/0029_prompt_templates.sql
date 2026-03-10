-- 0029_prompt_templates.sql
-- Purpose: Prompt template registry for planning pipeline prompt roles

CREATE TABLE IF NOT EXISTS prompt_templates (
  id               TEXT PRIMARY KEY,
  project_id       TEXT NOT NULL,
  role             TEXT NOT NULL,
  version          INTEGER NOT NULL DEFAULT 1,
  name             TEXT NOT NULL,
  description      TEXT,
  system_prompt    TEXT NOT NULL DEFAULT '',
  user_template    TEXT NOT NULL DEFAULT '',
  output_schema    TEXT,
  is_active        INTEGER NOT NULL DEFAULT 1,
  created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_prompt_templates_project ON prompt_templates(project_id);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_role ON prompt_templates(project_id, role, is_active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_prompt_templates_version ON prompt_templates(project_id, role, version);
