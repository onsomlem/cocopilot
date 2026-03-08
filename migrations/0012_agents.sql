-- Migration 0012: Agents (Agent Registration)
-- Purpose: Track registered agents with their capabilities and status

CREATE TABLE IF NOT EXISTS agents (
  id                 TEXT PRIMARY KEY,
  name               TEXT NOT NULL,
  capabilities_json  TEXT,
  metadata_json      TEXT,
  status             TEXT NOT NULL DEFAULT 'OFFLINE',
  last_seen          TEXT,
  registered_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_registered_at ON agents(registered_at);
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);

-- Create a default agent for existing runs
INSERT OR IGNORE INTO agents (id, name, status, registered_at)
VALUES ('agent_default', 'Default Agent', 'OFFLINE', strftime('%Y-%m-%dT%H:%M:%fZ','now'));