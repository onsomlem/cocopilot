-- Artifact line comments for diff viewer
CREATE TABLE IF NOT EXISTS artifact_comments (
    id          TEXT PRIMARY KEY,
    artifact_id TEXT NOT NULL,
    project_id  TEXT NOT NULL DEFAULT '',
    line_number INTEGER NOT NULL,
    body        TEXT NOT NULL,
    author      TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
CREATE INDEX IF NOT EXISTS idx_artifact_comments_artifact ON artifact_comments(artifact_id);
