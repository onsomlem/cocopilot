-- 0017_tasks_updated_at.sql
-- Purpose: Ensure tasks.updated_at exists and backfill when missing

ALTER TABLE tasks ADD COLUMN updated_at TEXT;

UPDATE tasks
SET updated_at = created_at
WHERE updated_at IS NULL OR updated_at = '';
