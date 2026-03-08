-- 0004_tasks_add_project_id.sql
-- Purpose: associate tasks with a project without changing v1 behavior.
-- Notes:
-- - Existing tasks are backfilled to 'proj_default'.
-- - We keep v1 columns unchanged.

ALTER TABLE tasks ADD COLUMN project_id TEXT;

UPDATE tasks
SET project_id = 'proj_default'
WHERE project_id IS NULL OR project_id = '';

CREATE INDEX IF NOT EXISTS idx_tasks_project_status_created_at
  ON tasks(project_id, status, created_at);
