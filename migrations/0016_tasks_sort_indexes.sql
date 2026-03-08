-- Migration 0016: Task sort indexes
-- Purpose: Add indexes to speed up task list sorting and filtering.

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
