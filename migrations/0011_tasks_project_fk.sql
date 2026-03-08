-- 0011_tasks_project_fk.sql
-- Purpose: Add foreign key constraint to tasks.project_id and make it NOT NULL
-- Notes:
-- - SQLite requires table recreation to add foreign key constraint
-- - All existing tasks should already have project_id set to 'proj_default'

-- Create new tasks table with foreign key constraint
CREATE TABLE tasks_new (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  instructions     TEXT NOT NULL,
  status           TEXT NOT NULL DEFAULT 'NOT_PICKED',
  output           TEXT,
  parent_task_id   INTEGER,
  created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  title            TEXT,
  type             TEXT,
  priority         INTEGER DEFAULT 50,
  tags_json        TEXT,
  status_v2        TEXT DEFAULT 'QUEUED',
  updated_at       TEXT,
  project_id       TEXT NOT NULL DEFAULT 'proj_default',
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_task_id) REFERENCES tasks(id)
);

-- Copy data from old table
INSERT INTO tasks_new 
  SELECT id, instructions, status, output, parent_task_id, created_at, 
         title, type, priority, tags_json, status_v2, updated_at,
         COALESCE(project_id, 'proj_default')
  FROM tasks;

-- Drop old table
DROP TABLE tasks;

-- Rename new table
ALTER TABLE tasks_new RENAME TO tasks;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_tasks_project_status_created_at
  ON tasks(project_id, status, created_at);

CREATE INDEX IF NOT EXISTS idx_tasks_status_v2
  ON tasks(status_v2);

CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id
  ON tasks(parent_task_id);
