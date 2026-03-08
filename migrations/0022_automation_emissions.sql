-- 0022_automation_emissions.sql
-- Purpose: Dedupe + throttle mechanism for automation emissions
-- Prevents infinite task spawning from "always-next-task" patterns

CREATE TABLE IF NOT EXISTS automation_emissions (
  dedupe_key   TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL,
  kind         TEXT NOT NULL,
  task_id      INTEGER,
  created_at   INTEGER NOT NULL,
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_automation_emissions_project_kind 
  ON automation_emissions(project_id, kind);
CREATE INDEX IF NOT EXISTS idx_automation_emissions_created_at 
  ON automation_emissions(created_at);
