-- Migration 0015: Events filter indexes
-- Purpose: Add indexes to speed up project, task, and kind filtering.

CREATE INDEX IF NOT EXISTS idx_events_project_id
  ON events(project_id);

CREATE INDEX IF NOT EXISTS idx_events_task_id_created
  ON events(entity_type, entity_id, created_at);

CREATE INDEX IF NOT EXISTS idx_events_project_kind_created
  ON events(project_id, kind, created_at);
