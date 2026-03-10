-- 0030_task_loop_anchor.sql
-- Purpose: Add loop_anchor_prompt field to tasks and task_templates

ALTER TABLE tasks ADD COLUMN loop_anchor_prompt TEXT;
ALTER TABLE task_templates ADD COLUMN default_loop_anchor TEXT;
