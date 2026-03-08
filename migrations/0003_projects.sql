-- 0003_projects.sql
-- Purpose: add projects so state can be scoped; seed a default project.
-- Notes:
-- - v1 remains "single project" via a seeded default.
-- - workdir is stored on the project and v1 /set-workdir updates default.

CREATE TABLE IF NOT EXISTS projects (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  workdir       TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  settings_json TEXT
);

-- Seed default project if none exists.
-- Use a deterministic ID to simplify mapping.
INSERT INTO projects (id, name, workdir, settings_json)
SELECT 'proj_default', 'Default', '', NULL
WHERE NOT EXISTS (SELECT 1 FROM projects WHERE id='proj_default');
