-- Migration 0007: Leases (Agent Coordination)
-- Purpose: Enable exclusive task claiming with expiration and heartbeat

CREATE TABLE IF NOT EXISTS leases (
  id         TEXT PRIMARY KEY,
  task_id    INTEGER NOT NULL,
  agent_id   TEXT NOT NULL,
  mode       TEXT NOT NULL DEFAULT 'exclusive',
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  expires_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Unique constraint: only one active lease per task
CREATE UNIQUE INDEX IF NOT EXISTS idx_leases_task_id ON leases(task_id);

-- Index for expiration cleanup job
CREATE INDEX IF NOT EXISTS idx_leases_expires_at ON leases(expires_at);

-- Index for agent-centric queries
CREATE INDEX IF NOT EXISTS idx_leases_agent_id ON leases(agent_id);
