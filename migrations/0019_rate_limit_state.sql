-- 0019_rate_limit_state.sql
-- Purpose: Persistent rate limit state for optional durable rate tracking

CREATE TABLE IF NOT EXISTS rate_limit_state (
  id             TEXT PRIMARY KEY,
  project_id     TEXT NOT NULL,
  agent_id       TEXT,
  action         TEXT NOT NULL,
  window_start   TEXT NOT NULL,
  request_count  INTEGER NOT NULL DEFAULT 0,
  created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_rate_limit_state_lookup
  ON rate_limit_state(project_id, agent_id, action);
