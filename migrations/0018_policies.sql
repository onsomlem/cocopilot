-- 0018_policies.sql
-- Purpose: Add policy engine foundation (project-scoped policies and rules)

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