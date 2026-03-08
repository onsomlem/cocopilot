-- Migration 0006: Runs (Execution Ledger)
-- Purpose: Track task execution attempts with detailed steps, logs, and artifacts

-- Main runs table: one run per task execution attempt
CREATE TABLE IF NOT EXISTS runs (
  id          TEXT PRIMARY KEY,
  task_id     INTEGER NOT NULL,
  agent_id    TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'RUNNING',
  started_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  finished_at TEXT,
  error       TEXT,
  FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_agent_id ON runs(agent_id);

-- Run steps: track major phases within a run
CREATE TABLE IF NOT EXISTS run_steps (
  id           TEXT PRIMARY KEY,
  run_id       TEXT NOT NULL,
  name         TEXT NOT NULL,
  status       TEXT NOT NULL,
  details_json TEXT,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_steps_run_id ON run_steps(run_id);

-- Run logs: stream stdout/stderr/info logs from agent
CREATE TABLE IF NOT EXISTS run_logs (
  id     INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL,
  stream TEXT NOT NULL,
  chunk  TEXT NOT NULL,
  ts     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_logs_run_id_ts ON run_logs(run_id, ts);

-- Artifacts: attach files produced during execution
CREATE TABLE IF NOT EXISTS artifacts (
  id            TEXT PRIMARY KEY,
  run_id        TEXT NOT NULL,
  kind          TEXT NOT NULL,
  storage_ref   TEXT NOT NULL,
  sha256        TEXT,
  size          INTEGER,
  metadata_json TEXT,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifacts_run_id ON artifacts(run_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_kind ON artifacts(kind);

-- Tool invocations: track which tools were called during execution
CREATE TABLE IF NOT EXISTS tool_invocations (
  id          TEXT PRIMARY KEY,
  run_id      TEXT NOT NULL,
  tool_name   TEXT NOT NULL,
  input_json  TEXT,
  output_json TEXT,
  started_at  TEXT NOT NULL,
  finished_at TEXT,
  FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tool_invocations_run_id ON tool_invocations(run_id);
CREATE INDEX IF NOT EXISTS idx_tool_invocations_tool_name ON tool_invocations(tool_name);
