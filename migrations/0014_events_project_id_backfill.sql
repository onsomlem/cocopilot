-- Migration 0014: Events project_id backfill
-- Purpose: Ensure project_id is populated for legacy events.

UPDATE events
SET project_id = 'proj_default'
WHERE project_id IS NULL OR TRIM(project_id) = '';
