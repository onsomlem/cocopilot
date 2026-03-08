-- Migration 0008: Events (Real-time Notifications)
-- Purpose: Store events for SSE replay and project-scoped notifications

CREATE TABLE IF NOT EXISTS events (
  id           TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL,
  kind         TEXT NOT NULL,
  entity_type  TEXT NOT NULL,
  entity_id    TEXT NOT NULL,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  payload_json TEXT,
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Index for project-scoped event queries and SSE
CREATE INDEX IF NOT EXISTS idx_events_project_created 
  ON events(project_id, created_at);

-- Index for filtering by event kind
CREATE INDEX IF NOT EXISTS idx_events_kind ON events(kind);

-- Index for entity lookup
CREATE INDEX IF NOT EXISTS idx_events_entity 
  ON events(entity_type, entity_id);
