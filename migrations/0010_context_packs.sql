-- Migration 0010: Context Packs (Automated Context Generation)
-- Purpose: Store pre-generated context bundles for tasks

CREATE TABLE IF NOT EXISTS context_packs (
  id            TEXT PRIMARY KEY,
  project_id    TEXT NOT NULL,
  task_id       INTEGER NOT NULL,
  summary       TEXT NOT NULL,
  contents_json TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Index for task-centric queries (get context pack for task)
CREATE INDEX IF NOT EXISTS idx_context_packs_task_id ON context_packs(task_id);

-- Index for project-level queries
CREATE INDEX IF NOT EXISTS idx_context_packs_project_id ON context_packs(project_id);
