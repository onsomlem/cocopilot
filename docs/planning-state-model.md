# Planning-State Data Model

**Version**: 1.0  
**Status**: Active  

## Overview

Defines the per-project planning state that persists between planning cycles, enabling continuity, workstream tracking, and decision history.

---

## PlanningState

The root planning state object, stored per project.

```go
type PlanningState struct {
    ID              int64           `json:"id"`
    ProjectID       int64           `json:"project_id"`
    PlanningMode    string          `json:"planning_mode"`    // standard | focused | recovery | maintenance
    CycleCount      int             `json:"cycle_count"`
    LastCycleAt     *string         `json:"last_cycle_at"`    // ISO8601
    Goals           []string        `json:"goals"`
    ReleaseFocus    string          `json:"release_focus"`
    MustNotForget   []string        `json:"must_not_forget"`
    ReconSummary    string          `json:"recon_summary"`
    PlannerSummary  string          `json:"planner_summary"`
    Blockers        []string        `json:"blockers"`
    Risks           []string        `json:"risks"`
    PriorityOrder   []string        `json:"priority_order"`   // ordered workstream IDs
    CreatedAt       string          `json:"created_at"`
    UpdatedAt       string          `json:"updated_at"`
}
```

### Fields

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| id | int64 | No | Primary key |
| project_id | int64 | No | FK to projects, unique |
| planning_mode | string | No | Current planning mode |
| cycle_count | int | No | Number of completed planning cycles |
| last_cycle_at | string | Yes | Timestamp of last completed cycle |
| goals | []string | No | Stated project goals (JSON array) |
| release_focus | string | No | Current release or sprint focus area |
| must_not_forget | []string | No | Threads the planner must not drop |
| recon_summary | string | No | Last recon stage summary |
| planner_summary | string | No | Last planner cycle summary |
| blockers | []string | No | Known blockers (JSON array) |
| risks | []string | No | Known risks (JSON array) |
| priority_order | []string | No | Ordered workstream IDs (JSON array) |
| created_at | string | No | Creation timestamp |
| updated_at | string | No | Last update timestamp |

---

## Workstream

A logical thread of work tracked across planning cycles.

```go
type Workstream struct {
    ID              string          `json:"id"`               // slug or UUID
    ProjectID       int64           `json:"project_id"`
    PlanningStateID int64           `json:"planning_state_id"`
    Title           string          `json:"title"`
    Description     string          `json:"description"`
    Status          string          `json:"status"`           // active | paused | completed | abandoned
    ContinuityScore float64        `json:"continuity_score"` // 0.0 - 1.0
    UrgencyScore    float64         `json:"urgency_score"`    // 0.0 - 1.0
    RelatedTaskIDs  []int64         `json:"related_task_ids"` // task IDs in this workstream
    RelatedRunIDs   []int64         `json:"related_run_ids"`  // run IDs in this workstream
    Why             string          `json:"why"`              // reason this workstream exists
    WhatRemains     string          `json:"what_remains"`     // remaining work description
    WhatNext        string          `json:"what_next"`        // next concrete action
    CreatedAt       string          `json:"created_at"`
    UpdatedAt       string          `json:"updated_at"`
}
```

### Fields

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| id | string | No | Unique workstream identifier (slug) |
| project_id | int64 | No | FK to projects |
| planning_state_id | int64 | No | FK to planning_state |
| title | string | No | Human-readable title |
| description | string | No | Detailed description |
| status | string | No | active, paused, completed, abandoned |
| continuity_score | float64 | No | How important to continue (0-1) |
| urgency_score | float64 | No | Time-sensitivity (0-1) |
| related_task_ids | []int64 | No | Associated task IDs (JSON array) |
| related_run_ids | []int64 | No | Associated run IDs (JSON array) |
| why | string | No | Rationale for this workstream |
| what_remains | string | No | Description of remaining work |
| what_next | string | No | Next concrete action to take |
| created_at | string | No | Creation timestamp |
| updated_at | string | No | Last update timestamp |

---

## PlanningCycleRecord

A log of each completed planning cycle for history and diagnostics.

```go
type PlanningCycleRecord struct {
    ID                int64           `json:"id"`
    ProjectID         int64           `json:"project_id"`
    CycleNumber       int             `json:"cycle_number"`
    PlanningMode      string          `json:"planning_mode"`
    StartedAt         string          `json:"started_at"`
    CompletedAt       *string         `json:"completed_at"`
    ReconOutput       json.RawMessage `json:"recon_output"`
    ContinuityOutput  json.RawMessage `json:"continuity_output"`
    GapOutput         json.RawMessage `json:"gap_output"`
    PrioritizationOut json.RawMessage `json:"prioritization_output"`
    SynthesisOutput   json.RawMessage `json:"synthesis_output"`
    AntiDriftOutput   json.RawMessage `json:"anti_drift_output"`
    TasksCreated      []int64         `json:"tasks_created"`
    CoherenceScore    float64         `json:"coherence_score"`
    StageFailures     []string        `json:"stage_failures"`
    DriftWarnings     []string        `json:"drift_warnings"`
}
```

---

## PlannerDecision

Individual decisions made during planning, for audit and decision history.

```go
type PlannerDecision struct {
    ID          int64           `json:"id"`
    ProjectID   int64           `json:"project_id"`
    CycleID     int64           `json:"cycle_id"`
    Stage       string          `json:"stage"`       // recon | continuity | gap | prioritization | synthesis | anti_drift | state_update
    DecisionType string         `json:"decision_type"` // continued | deferred | created | rejected | approved | abandoned
    Subject     string          `json:"subject"`     // what was decided about
    Reasoning   string          `json:"reasoning"`   // why
    CreatedAt   string          `json:"created_at"`
}
```

---

## SQL Schema

```sql
CREATE TABLE IF NOT EXISTS planning_state (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id    INTEGER NOT NULL UNIQUE,
    planning_mode TEXT NOT NULL DEFAULT 'standard',
    cycle_count   INTEGER NOT NULL DEFAULT 0,
    last_cycle_at TEXT,
    goals         TEXT NOT NULL DEFAULT '[]',
    release_focus TEXT NOT NULL DEFAULT '',
    must_not_forget TEXT NOT NULL DEFAULT '[]',
    recon_summary TEXT NOT NULL DEFAULT '',
    planner_summary TEXT NOT NULL DEFAULT '',
    blockers      TEXT NOT NULL DEFAULT '[]',
    risks         TEXT NOT NULL DEFAULT '[]',
    priority_order TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workstreams (
    id               TEXT NOT NULL,
    project_id       INTEGER NOT NULL,
    planning_state_id INTEGER NOT NULL,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'active',
    continuity_score REAL NOT NULL DEFAULT 0.0,
    urgency_score    REAL NOT NULL DEFAULT 0.0,
    related_task_ids TEXT NOT NULL DEFAULT '[]',
    related_run_ids  TEXT NOT NULL DEFAULT '[]',
    why              TEXT NOT NULL DEFAULT '',
    what_remains     TEXT NOT NULL DEFAULT '',
    what_next        TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (planning_state_id) REFERENCES planning_state(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS planning_cycles (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id          INTEGER NOT NULL,
    cycle_number        INTEGER NOT NULL,
    planning_mode       TEXT NOT NULL,
    started_at          TEXT NOT NULL,
    completed_at        TEXT,
    recon_output        TEXT NOT NULL DEFAULT '{}',
    continuity_output   TEXT NOT NULL DEFAULT '{}',
    gap_output          TEXT NOT NULL DEFAULT '{}',
    prioritization_output TEXT NOT NULL DEFAULT '{}',
    synthesis_output    TEXT NOT NULL DEFAULT '{}',
    anti_drift_output   TEXT NOT NULL DEFAULT '{}',
    tasks_created       TEXT NOT NULL DEFAULT '[]',
    coherence_score     REAL NOT NULL DEFAULT 0.0,
    stage_failures      TEXT NOT NULL DEFAULT '[]',
    drift_warnings      TEXT NOT NULL DEFAULT '[]',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS planner_decisions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id    INTEGER NOT NULL,
    cycle_id      INTEGER NOT NULL,
    stage         TEXT NOT NULL,
    decision_type TEXT NOT NULL,
    subject       TEXT NOT NULL,
    reasoning     TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (cycle_id) REFERENCES planning_cycles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workstreams_project ON workstreams(project_id);
CREATE INDEX IF NOT EXISTS idx_workstreams_status ON workstreams(project_id, status);
CREATE INDEX IF NOT EXISTS idx_planning_cycles_project ON planning_cycles(project_id);
CREATE INDEX IF NOT EXISTS idx_planner_decisions_project ON planner_decisions(project_id);
CREATE INDEX IF NOT EXISTS idx_planner_decisions_cycle ON planner_decisions(cycle_id);
```
