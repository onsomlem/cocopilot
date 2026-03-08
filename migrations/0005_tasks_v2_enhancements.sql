-- Migration 0005: Tasks v2 Enhancements
-- Purpose: Add v2 metadata fields to tasks table

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
