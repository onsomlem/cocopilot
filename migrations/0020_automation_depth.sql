-- 0020_automation_depth.sql
-- Purpose: Add automation recursion depth tracking to tasks

ALTER TABLE tasks ADD COLUMN automation_depth INTEGER NOT NULL DEFAULT 0;
