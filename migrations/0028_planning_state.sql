-- 0028_planning_state.sql
-- Purpose: Add planning state, workstreams, planning cycles, and planner decisions tables
-- for the staged idle-planning pipeline.

CREATE TABLE IF NOT EXISTS planning_state (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL UNIQUE,
    planning_mode   TEXT NOT NULL DEFAULT 'standard',
    cycle_count     INTEGER NOT NULL DEFAULT 0,
    last_cycle_at   TEXT,
    goals           TEXT NOT NULL DEFAULT '[]',
    release_focus   TEXT NOT NULL DEFAULT '',
    must_not_forget TEXT NOT NULL DEFAULT '[]',
    recon_summary   TEXT NOT NULL DEFAULT '',
    planner_summary TEXT NOT NULL DEFAULT '',
    blockers        TEXT NOT NULL DEFAULT '[]',
    risks           TEXT NOT NULL DEFAULT '[]',
    priority_order  TEXT NOT NULL DEFAULT '[]',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workstreams (
    id                TEXT NOT NULL,
    project_id        TEXT NOT NULL,
    planning_state_id TEXT NOT NULL,
    title             TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'active',
    continuity_score  REAL NOT NULL DEFAULT 0.0,
    urgency_score     REAL NOT NULL DEFAULT 0.0,
    related_task_ids  TEXT NOT NULL DEFAULT '[]',
    related_run_ids   TEXT NOT NULL DEFAULT '[]',
    why               TEXT NOT NULL DEFAULT '',
    what_remains      TEXT NOT NULL DEFAULT '',
    what_next         TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (planning_state_id) REFERENCES planning_state(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS planning_cycles (
    id                    TEXT PRIMARY KEY,
    project_id            TEXT NOT NULL,
    cycle_number          INTEGER NOT NULL,
    planning_mode         TEXT NOT NULL,
    started_at            TEXT NOT NULL,
    completed_at          TEXT,
    recon_output          TEXT NOT NULL DEFAULT '{}',
    continuity_output     TEXT NOT NULL DEFAULT '{}',
    gap_output            TEXT NOT NULL DEFAULT '{}',
    prioritization_output TEXT NOT NULL DEFAULT '{}',
    synthesis_output      TEXT NOT NULL DEFAULT '{}',
    anti_drift_output     TEXT NOT NULL DEFAULT '{}',
    tasks_created         TEXT NOT NULL DEFAULT '[]',
    coherence_score       REAL NOT NULL DEFAULT 0.0,
    stage_failures        TEXT NOT NULL DEFAULT '[]',
    drift_warnings        TEXT NOT NULL DEFAULT '[]',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS planner_decisions (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    cycle_id      TEXT NOT NULL,
    stage         TEXT NOT NULL,
    decision_type TEXT NOT NULL,
    subject       TEXT NOT NULL,
    reasoning     TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (cycle_id) REFERENCES planning_cycles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workstreams_project ON workstreams(project_id);
CREATE INDEX IF NOT EXISTS idx_workstreams_status ON workstreams(project_id, status);
CREATE INDEX IF NOT EXISTS idx_planning_cycles_project ON planning_cycles(project_id);
CREATE INDEX IF NOT EXISTS idx_planner_decisions_project ON planner_decisions(project_id);
CREATE INDEX IF NOT EXISTS idx_planner_decisions_cycle ON planner_decisions(cycle_id);
