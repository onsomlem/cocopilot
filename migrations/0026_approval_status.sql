-- 0021_approval_status.sql
-- Purpose: Add approval gate fields to tasks for human-in-the-loop workflows

ALTER TABLE tasks ADD COLUMN requires_approval INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN approval_status TEXT;
