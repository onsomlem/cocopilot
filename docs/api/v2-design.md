# API v2 Contract Design Document

## Executive Summary

This document describes the design of Cocopilot's second-generation API (v2), which introduces multi-project support, execution ledger tracking, lease-based task claiming, and structured completion reporting while maintaining full backward compatibility with v1.

**Version:** 2.0.0  
**Date:** February 12, 2026  
**Status:** Implemented (core v2 endpoints live)

## Goals and Principles

### Primary Goals

1. **Multi-project support**: Enable multiple isolated workspaces within a single instance
2. **Execution observability**: Track runs, steps, artifacts, and tool invocations in detail
3. **Agent coordination**: Support multiple agents with lease-based task claiming
4. **Structured metadata**: Move from plain text to rich, structured task completion data
5. **Backward compatibility**: Keep v1 endpoints stable and functioning

### Design Principles

- **v1 Stability**: All v1 endpoints remain unchanged at root level
- **v2 Namespacing**: All new features under `/api/v2/*` prefix
- **Progressive Enhancement**: v2 capabilities built on top of v1 data model
- **Feature Flags**: Risky features (leases, tool execution) can be disabled
- **Data Compatibility**: v1 and v2 can coexist, sharing the same task records

## Architecture Overview

### API Versioning Strategy

```
Root Level (v1 - Legacy)
├── GET  /task
├── POST /create
├── POST /save
├── GET  /events (SSE)
├── POST /set-workdir
└── GET  /api/tasks

/api/v2/* (Modern)
├── Health & Version
├── Projects (multi-workspace)
├── Tasks (enhanced with metadata)
├── Leases (agent coordination)
├── Runs (execution ledger)
├── Events (project-scoped SSE)
├── Memory (persistent knowledge)
└── Context Packs (automated context)
```

### Domain Model

```
Project (1) ─────┬───> (N) Task
                 │
                 ├───> (N) Memory Items
                 │
                 └───> (N) Events

Task (1) ────────┬───> (N) Runs
                 │
                 ├───> (0..1) Active Lease
                 │
                 └───> (0..1) Parent Task

Run (1) ─────────┬───> (N) Steps
                 │
                 ├───> (N) Logs
                 │
                 ├───> (N) Artifacts
                 │
                 └───> (N) Tool Invocations

Lease (1) ───────> (1) Task
              └───> (1) Agent
```

## Core Endpoints Design

### 1. Projects

Projects provide workspace isolation, allowing multiple independent contexts.

**Key Features:**
- Each project has its own `workdir`, settings, and task scope
- Default project (`proj_default`) maintains v1 compatibility
- Projects can be created, updated, and deleted
- Settings JSON allows flexible configuration per project

**Endpoints:**
- `POST /api/v2/projects` - Create new project
- `GET /api/v2/projects` - List all projects
- `GET /api/v2/projects/{projectId}` - Get project details
- `PUT /api/v2/projects/{projectId}` - Update project
- `DELETE /api/v2/projects/{projectId}` - Delete project

**Example Use Cases:**
- Separate frontend and backend projects
- Per-customer workspaces in a multi-tenant scenario
- Temporary experimental workspaces

### 2. Tasks (Enhanced)

Tasks gain project scoping, type classification, tagging, and dual status tracking.

**Key Features:**
- `project_id` associates task with a project
- `type` field (ANALYZE, MODIFY, TEST, REVIEW, DOC, RELEASE, ROLLBACK)
- `tags` for flexible categorization
- `status_v1` maintains v1 compatibility (NOT_PICKED, IN_PROGRESS, COMPLETE)
- `status_v2` adds granularity (QUEUED, CLAIMED, RUNNING, SUCCEEDED, FAILED, NEEDS_REVIEW, CANCELLED)
- `title` for human-readable summary
- `priority` for ordering

**Endpoints:**
- `POST /api/v2/tasks` - Create task (optionally scoped with `project_id`)
- `POST /api/v2/projects/{projectId}/tasks` - Create task in project
- `GET /api/v2/tasks` - List/filter/search tasks across projects
- `GET /api/v2/projects/{projectId}/tasks` - List/filter/search tasks within project
- `GET /api/v2/tasks/{taskId}` - Get task details with latest run
- `PATCH /api/v2/tasks/{taskId}` - Update task fields
- `DELETE /api/v2/tasks/{taskId}` - Delete task

**Query Parameters:**
- `status` - Filter by v1 or v2 status
- `type` - Filter by task type
- `tag` - Filter by tag
- `q` - Full-text search
- `project_id` - Filter by project (applies to `/api/v2/tasks`)
- `sort` - Sort by `created_at` or `updated_at` with `:asc`/`:desc`
- `limit`, `offset` - Pagination

**Status Mapping:**

| v1 Status      | v2 Status Equivalents                      |
|----------------|--------------------------------------------|
| NOT_PICKED     | QUEUED                                     |
| IN_PROGRESS    | CLAIMED, RUNNING                           |
| COMPLETE       | SUCCEEDED, FAILED, NEEDS_REVIEW, CANCELLED |

### 3. Leases (Agent Coordination)

Leases enable exclusive task claiming with expiration and heartbeat.

**Key Features:**
- Prevents multiple agents from working on the same task
- Time-bound with configurable expiration
- Heartbeat mechanism to keep lease alive
- Explicit release for clean termination
- Returns 409 CONFLICT if task already claimed

**Endpoints:**
- `POST /api/v2/tasks/{taskId}/claim` - Claim task for execution
- `POST /api/v2/leases/{leaseId}/heartbeat` - Renew lease
- `POST /api/v2/leases/{leaseId}/release` - Release lease

**Claim Flow:**
1. Agent requests claim with `agent_id`
2. Server creates lease (15-minute TTL)
3. Server updates task status to `IN_PROGRESS`/`CLAIMED`
4. Agent receives lease and updated task

**Heartbeat Pattern:**
- Agent sends heartbeat every ~1-2 minutes
- Server extends expiration timestamp by 15 minutes (from max(now, current expiry))
- On heartbeat failure, lease expires automatically
- Expired leases allow task to be reclaimed

### 4. Runs (Execution Ledger)

Runs are the audit log for task execution attempts.

**Key Features:**
- One task can have multiple runs (retries, different agents, etc.)
- Tracks `started_at`, `finished_at`, final `status`
- Associated with `agent_id` for accountability
- Contains structured steps, logs, artifacts, tool invocations

**Endpoints:**
- `GET /api/v2/runs/{runId}` - Get run details
- `POST /api/v2/runs/{runId}/steps` - Log execution step
- `POST /api/v2/runs/{runId}/logs` - Stream logs (stdout/stderr/info)
- `POST /api/v2/runs/{runId}/artifacts` - Attach artifacts (diffs, patches, reports)

**Note:** Tool invocations are returned only in run detail (`GET /api/v2/runs/{runId}`) today; there is no separate tool-invocations endpoint yet.

**Run Lifecycle:**
1. Created when execution begins with status `RUNNING` (current runtime: v2 claim does not auto-create runs; v1 `/task` claim does)
2. Agent logs steps as work progresses
3. Agent streams logs for observability
4. Agent attaches artifacts (diffs, test results, etc.)
5. Finalized on task completion with status `SUCCEEDED` or `FAILED`

**Steps:**
- Name: e.g., "Analyze codebase", "Run tests", "Generate patch"
- Status: STARTED, SUCCEEDED, FAILED
- Details: Structured JSON for step-specific data

**Artifacts:**
- Kind: diff, patch, log, report, file
- Storage reference (could be local path, S3 URL, etc.)
- SHA256 hash for integrity
- Metadata for custom fields

### 5. Completion (Structured)

Replaces v1's plain-text `output` with rich, structured result data.

**Key Features:**
- `summary` for human readability
- Lists of `changes_made`, `files_touched`, `commands_run`
- Test results in `tests_run`
- Risk assessment in `risks`
- Suggestion of `next_tasks` for workflow automation
- Follow-up creation from `next_tasks` is governed by configurable task.completed rules (enable/disable, limits, allowlists, dependency behavior)

**Task-Type Validation:**
- `MODIFY` requires `changes_made` and `files_touched`
- `TEST` requires `tests_run`
- `DOC` requires `summary`
- `REVIEW` requires `summary` and `risks`
- Other fields remain optional and are validated when present

**Initial policy rule shape (project-scoped):**

Policy rules live in `policies.rules` JSON. A policy rule can block automation follow-up creation with:

```json
{
  "type": "automation.block",
  "reason": "Optional human-readable reason"
}
```

Task completion can be blocked with:

```json
{
  "type": "completion.block",
  "reason": "Optional human-readable reason"
}
```

Task creation can be blocked with:

```json
{
  "type": "task.create.block",
  "reason": "Optional human-readable reason"
}
```

Task updates can be blocked with:

```json
{
  "type": "task.update.block",
  "reason": "Optional human-readable reason"
}
```

Task deletes can be blocked with:

```json
{
  "type": "task.delete.block",
  "reason": "Optional human-readable reason"
}
```

**Endpoint:**
- `POST /api/v2/tasks/{taskId}/complete`

**Compatibility Behavior:**
- Server generates human-readable synopsis and writes to v1 `tasks.output`
- v1 clients see completion as before
- v2 clients get full structured data

**Example Payload:**
```json
{
  "run_id": "run_abc123",
  "status": "SUCCEEDED",
  "result": {
    "summary": "Implemented login feature with JWT authentication",
    "changes_made": [
      "Added POST /api/login endpoint",
      "Integrated JWT library",
      "Added middleware for authentication"
    ],
    "files_touched": [
      "src/auth/login.go",
      "src/middleware/auth.go",
      "go.mod"
    ],
    "commands_run": [
      "go get github.com/golang-jwt/jwt/v5",
      "go test ./src/auth/...",
      "go build"
    ],
    "tests_run": [
      "TestLoginSuccess: PASS",
      "TestLoginInvalidCredentials: PASS",
      "TestLoginMissingParams: PASS"
    ],
    "risks": [
      "JWT secret should be moved to environment variable",
      "Rate limiting not yet implemented"
    ],
    "next_tasks": [
      {
        "title": "Add rate limiting to login endpoint",
        "instructions": "Implement rate limiting middleware to prevent brute force attacks",
        "type": "MODIFY",
        "priority": 8
      }
    ]
  }
}
```

### 6. Events (Project-Scoped SSE)

Events enable real-time updates scoped to specific projects.

**Key Features:**
- List endpoint with filters: `/api/v2/events`
- SSE stream with filters: `/api/v2/events/stream`
- Event types: task, run, lease, memory, auth, repo changes
- Event IDs for idempotency and ordering

**Event Kinds:**
- `task.created`, `task.updated`, `task.deleted`
- `run.started`, `run.updated`, `run.completed`
- `lease.created`, `lease.expired`
- `memory.updated`
- `repo.file_changed`

**SSE Format:**
```
id: evt_123
event: task.created
data: {"id": "evt_123", "kind": "task.created", ...}
```

**Replay Flow:**
1. Client connects to `/api/v2/events/stream`
2. Client supplies `since` (RFC3339 or event id) to replay backlog
3. Server replays events in the stream before switching to live events

### 7. Memory (Persistent Knowledge)

Memory stores project-level knowledge that accumulates across tasks.

**Key Features:**
- Scoped: GLOBAL, MODULE, FILE, TASK, RUN
- Key-value store with structured JSON values
- Source references track where knowledge came from
- Queryable by scope, key, or full-text search

**Endpoints:**
- `PUT /api/v2/projects/{projectId}/memory` - Store/update memory
- `GET /api/v2/projects/{projectId}/memory` - Query memory

**Use Cases:**
- Architecture decisions: `scope=GLOBAL, key=architecture`
- API conventions: `scope=MODULE, key=api_patterns`
- Known issues: `scope=FILE, key=src/auth/login.go`
- Task-specific context: `scope=TASK`

### 8. Context Packs (Automated Context)

Context packs automatically gather relevant context for a task.

**Key Features:**
- Budget-controlled (max files, bytes, snippets)
- Includes file snippets, related tasks, decisions, repo state
- Generates during task claim or on demand
- Reduces manual context gathering

**Endpoint:**
- `POST /api/v2/projects/{projectId}/context-packs`

**Contents:**
- **Files**: Paths with line-range snippets
- **Related tasks**: IDs of parent/child/similar tasks
- **Decisions**: Architecture decisions from memory
- **Repo state**: Git HEAD, dirty status

## Implementation Notes (Current Runtime)

- **Auth guardrails:** v2 endpoints honor `X-API-Key` when `COCO_REQUIRE_API_KEY` is enabled. If `COCO_REQUIRE_API_KEY_READS` is true, reads also require keys. Identities come from `COCO_API_IDENTITIES` (`id|type|api_key|scope1,scope2`), with `COCO_API_KEY` acting as a legacy wildcard scope. Scopes map to endpoints (`tasks:read`, `tasks:write`, `projects:write`, `policy.read`, `policy.write`, `leases:write`, `agents:write`, `runs:write`, `v2:read`, `v2:write`); claim accepts `tasks:write` or `leases:write`. Auth denials emit `auth.denied` events.
- **Leases behavior:** leases are created with a 15-minute TTL. Heartbeats extend by 15 minutes (from max(now, current expiry)) and return 410 if already expired. v2 claim only creates the lease and updates task status; it does not auto-create runs or context packs. Lease release is triggered on task completion and task deletion. Expired leases are cleaned up and tasks are re-queued to `NOT_PICKED`.
- **Runs sub-resources:** `/api/v2/runs/{runId}/steps`, `/logs`, and `/artifacts` are live. Steps require `name` and `status` (`STARTED`, `SUCCEEDED`, `FAILED`) with optional `details`. Logs require `stream` (`stdout`, `stderr`, `info`) and non-empty `chunk` and return 204. Artifacts require `kind` (`diff`, `patch`, `log`, `report`, `file`), `storage_ref`, and non-negative `size`; optional `sha256` and `metadata` are stored.
- **Completion next_tasks:** when `/api/v2/tasks/{taskId}/complete` includes `result.next_tasks`, each entry must provide non-empty `instructions`; `title`, `type` (validated against v2 types), `priority` (non-negative), and `tags` are optional. Follow-up creation honors configurable task.completed rules (enable/disable, max follow-ups, allowlisted types/tags, and whether to chain dependencies). When allowed, each entry becomes a child task in the same project with `parent_task_id` set and emits a `task.created` event (v1 task list broadcast fires after creation).
- **Task list filters:** `/api/v2/tasks` and `/api/v2/projects/{projectId}/tasks` support `status` (v1 or v2 values), `type`, `tag`, `q`, paging (`limit`, `offset`), and `sort` (`created_at:asc`, `created_at:desc`, `updated_at:asc`, `updated_at:desc`). `/api/v2/tasks` also accepts `project_id`. Defaults are `limit=100`, `sort=created_at:asc`, and `offset=0` with max `limit=500`.
- **Memory endpoints:** `/api/v2/projects/{projectId}/memory` supports `GET` with `scope`, `key`, and `q` filters plus `PUT` for upserts. `PUT` requires `scope`, `key`, and `value`; missing projects return 404.
- **Context packs:** `/api/v2/projects/{projectId}/context-packs` accepts `POST` only. It requires `task_id` and optionally `query` plus `budget` (`max_files`, `max_bytes`, `max_snippets`). The task must belong to the project; packs are persisted with a generated summary.
- **Project tree/changes:** `/api/v2/projects/{projectId}/tree` is implemented and returns a shallow workdir snapshot rooted at `.` with nested children. `/api/v2/projects/{projectId}/changes` is implemented, accepts optional `since` (RFC3339), and returns git status-based working tree changes.
- **Events filters:** `/api/v2/events` lists events with `project_id`, `type` (kind), `since` (RFC3339), `task_id`, `limit`, and `offset`. `/api/v2/events/stream` supports `project_id`, `type`, `since` (RFC3339 or event id), and `limit` (replay cap). Stream events use `event: <kind>` and `id: <event_id>`.
- **updated_at propagation:** `updated_at` is set on task create, update, claim, completion, v1 save, and v1 status updates. Sorting uses `COALESCE(updated_at, created_at)` and v2 responses always return a non-empty `updated_at`.

## Backward Compatibility Plan

### v1 Endpoint Preservation

All v1 endpoints remain **completely unchanged**:

- `GET /task` - Returns next task, marks IN_PROGRESS
- `POST /create` - Creates task with `instructions`, `parent_task_id`
- `POST /save` - Saves task output
- `GET /events` - SSE for all tasks (cross-project)
- `POST /set-workdir` - Updates default project workdir
- `GET /api/tasks` - Lists all tasks

### Data Model Bridge

v1 and v2 share the same underlying `tasks` table:

```sql
-- v1 columns (unchanged)
id, instructions, status, output, parent_task_id, created_at

-- v2 additions (nullable, backfilled to defaults)
project_id, title, type, priority, tags_json, status_v2
```

**Mapping Strategy:**
- All v1 tasks automatically belong to `proj_default`
- `status_v1` and `status_v2` are kept in sync
- v1 `output` is populated from v2 completion summary
- v2 reads v1 fields seamlessly

### v1 to v2 Status Sync

| Action                     | v1 Status      | v2 Status       |
|----------------------------|----------------|-----------------|
| Task created via v1        | NOT_PICKED     | QUEUED          |
| GET /task picks up         | IN_PROGRESS    | RUNNING         |
| POST /save                 | COMPLETE       | SUCCEEDED       |
| Task created via v2        | NOT_PICKED     | QUEUED          |
| v2 claim                   | IN_PROGRESS    | CLAIMED         |
| v2 complete (SUCCEEDED)    | COMPLETE       | SUCCEEDED       |
| v2 complete (FAILED)       | IN_PROGRESS    | FAILED          |
| v2 complete (NEEDS_REVIEW) | IN_PROGRESS    | NEEDS_REVIEW    |

**Rule:** v1 status is always derivable from v2 status, but not vice versa.

### Migration Path for Clients

**Phase 1: v1 clients continue working**
- No changes required
- They operate on `proj_default` implicitly

**Phase 2: New clients adopt v2**
- Use `/api/v2/*` endpoints
- Benefit from projects, leases, runs, structured completion

**Phase 3: Gradual v1 client migration (optional)**
- Migrate to v2 endpoints incrementally
- v1 endpoints remain supported indefinitely (or until explicitly deprecated)

### Workdir Compatibility

- v1 stores workdir globally (singleton)
- v2 stores workdir per project
- `/set-workdir` updates `proj_default` workdir for v1 compatibility
- v2 clients set workdir when creating projects

## Database Schema Additions

### Existing Tables (from migrations)

```sql
-- 0001: schema_migrations table
CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY);

-- 0002: tasks table (v1 schema)
CREATE TABLE tasks (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  instructions    TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'NOT_PICKED',
  output          TEXT,
  parent_task_id  INTEGER,
  created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- 0003: projects table
CREATE TABLE projects (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  workdir       TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  settings_json TEXT
);

-- 0004: tasks.project_id column
ALTER TABLE tasks ADD COLUMN project_id TEXT;
```

### New Tables Required for v2

#### Migration 0005: Tasks v2 Enhancements

```sql
-- Add v2 task fields
ALTER TABLE tasks ADD COLUMN title TEXT;
ALTER TABLE tasks ADD COLUMN type TEXT DEFAULT 'MODIFY';
ALTER TABLE tasks ADD COLUMN priority INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN status_v2 TEXT DEFAULT 'QUEUED';
ALTER TABLE tasks ADD COLUMN tags_json TEXT;
ALTER TABLE tasks ADD COLUMN updated_at TEXT;

-- Backfill status_v2 from status (v1)
UPDATE tasks SET status_v2 = 
  CASE status
    WHEN 'NOT_PICKED' THEN 'QUEUED'
    WHEN 'IN_PROGRESS' THEN 'RUNNING'
    WHEN 'COMPLETE' THEN 'SUCCEEDED'
  END;

-- Index for v2 queries
CREATE INDEX idx_tasks_v2_status_priority 
  ON tasks(project_id, status_v2, priority DESC, created_at ASC);

CREATE INDEX idx_tasks_type ON tasks(type);
```

#### Migration 0006: Runs (Execution Ledger)

```sql
CREATE TABLE runs (
  id          TEXT PRIMARY KEY,
  task_id     INTEGER NOT NULL,
  agent_id    TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'RUNNING',
  started_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  finished_at TEXT,
  error       TEXT,
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE INDEX idx_runs_task_id ON runs(task_id);
CREATE INDEX idx_runs_status ON runs(status);

CREATE TABLE run_steps (
  id          TEXT PRIMARY KEY,
  run_id      TEXT NOT NULL,
  name        TEXT NOT NULL,
  status      TEXT NOT NULL,
  details_json TEXT,
  created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id)
);

CREATE INDEX idx_run_steps_run_id ON run_steps(run_id);

CREATE TABLE run_logs (
  id         TEXT PRIMARY KEY,
  run_id     TEXT NOT NULL,
  stream     TEXT NOT NULL,
  chunk      TEXT NOT NULL,
  ts         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id)
);

CREATE INDEX idx_run_logs_run_id_ts ON run_logs(run_id, ts);

CREATE TABLE artifacts (
  id           TEXT PRIMARY KEY,
  run_id       TEXT NOT NULL,
  kind         TEXT NOT NULL,
  storage_ref  TEXT NOT NULL,
  sha256       TEXT,
  size         INTEGER,
  metadata_json TEXT,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (run_id) REFERENCES runs(id)
);

CREATE INDEX idx_artifacts_run_id ON artifacts(run_id);

CREATE TABLE tool_invocations (
  id          TEXT PRIMARY KEY,
  run_id      TEXT NOT NULL,
  tool_name   TEXT NOT NULL,
  input_json  TEXT,
  output_json TEXT,
  started_at  TEXT NOT NULL,
  finished_at TEXT,
  FOREIGN KEY (run_id) REFERENCES runs(id)
);

CREATE INDEX idx_tool_invocations_run_id ON tool_invocations(run_id);
```

#### Migration 0007: Leases

```sql
CREATE TABLE leases (
  id         TEXT PRIMARY KEY,
  task_id    INTEGER NOT NULL,
  agent_id   TEXT NOT NULL,
  mode       TEXT NOT NULL DEFAULT 'exclusive',
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  expires_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE UNIQUE INDEX idx_leases_task_id ON leases(task_id);
CREATE INDEX idx_leases_expires_at ON leases(expires_at);
CREATE INDEX idx_leases_agent_id ON leases(agent_id);
```

#### Migration 0008: Events

```sql
CREATE TABLE events (
  id           TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL,
  kind         TEXT NOT NULL,
  entity_type  TEXT NOT NULL,
  entity_id    TEXT NOT NULL,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  payload_json TEXT,
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX idx_events_project_created ON events(project_id, created_at);
CREATE INDEX idx_events_kind ON events(kind);
```

#### Migration 0009: Memory

```sql
CREATE TABLE memory (
  id            TEXT PRIMARY KEY,
  project_id    TEXT NOT NULL,
  scope         TEXT NOT NULL,
  key           TEXT NOT NULL,
  value_json    TEXT NOT NULL,
  source_refs_json TEXT,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE UNIQUE INDEX idx_memory_project_scope_key 
  ON memory(project_id, scope, key);

CREATE INDEX idx_memory_scope ON memory(scope);
```

#### Migration 0010: Context Packs

```sql
CREATE TABLE context_packs (
  id           TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL,
  task_id      INTEGER NOT NULL,
  summary      TEXT NOT NULL,
  contents_json TEXT NOT NULL,
  created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  FOREIGN KEY (project_id) REFERENCES projects(id),
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE INDEX idx_context_packs_task_id ON context_packs(task_id);
CREATE INDEX idx_context_packs_project_id ON context_packs(project_id);
```

## Error Handling

All v2 endpoints return consistent error structure:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Task not found",
    "details": {
      "task_id": 123
    }
  }
}
```

### Standard Error Codes

- `INVALID_ARGUMENT` (400) - Bad request parameters
- `UNAUTHORIZED` (401) - Authentication required
- `FORBIDDEN` (403) - Insufficient permissions
- `NOT_FOUND` (404) - Resource doesn't exist
- `CONFLICT` (409) - Resource conflict (e.g., task already claimed)
- `INTERNAL` (500) - Server error

## Security Considerations

### Authentication (Current)

v2 supports API key auth via `X-API-Key` with optional read enforcement:

- `COCO_REQUIRE_API_KEY=true` enables auth for v2 endpoints
- `COCO_REQUIRE_API_KEY_READS=true` enforces auth for reads as well
- `COCO_API_IDENTITIES` defines scoped identities (`id|type|api_key|scopes`)
- `COCO_API_KEY` is accepted as a legacy wildcard identity

### Authorization (Current)

- Scope checks map to endpoints (`tasks:read`, `tasks:write`, `projects:write`, `policy.read`, `policy.write`, `leases:write`, `agents:write`, `runs:write`, `v2:read`, `v2:write`)
- Task claim allows either `tasks:write` or `leases:write`
- Auth failures emit `auth.denied` events for visibility

### Input Validation

- All JSON fields validated against schema
- Task instructions sanitized (no code injection)
- File paths validated against workdir boundaries
- Lease expiration bounded (max 1 hour)

## Performance Considerations

### Indexing Strategy

Key indexes for v2:
- Tasks: `(project_id, status_v2, priority, created_at)`
- Runs: `(task_id)`, `(status)`
- Leases: `(task_id)` UNIQUE, `(expires_at)`, `(agent_id)`
- Events: `(project_id, created_at)`
- Memory: `(project_id, scope, key)` UNIQUE

### Scalability

- SQLite performs well up to 100K tasks, 1M events
- SSE scales to ~1K concurrent connections per project
- For higher scale, consider:
  - PostgreSQL migration
  - Redis for events broker
  - Object storage (S3) for artifacts

### Query Optimization

- Task listing with filters uses covering indexes
- Event replay uses timestamp-based pagination
- Logs streamed in chunks, not loaded fully
- Context packs cached per task

## Testing Strategy

### Unit Tests

- Endpoint handlers with mock DB
- Status mapping logic
- Lease expiration logic
- Context pack generation

### Integration Tests

- Full API flow: create project → create task → claim → run → complete
- v1 to v2 status sync
- Lease heartbeat and expiration
- SSE event delivery

### Compatibility Tests

- v1 endpoints work unchanged
- v1 tasks appear in v2 queries
- v2 completions populate v1 output field

## Implementation Roadmap

Note (2026-02-12): Phases 1-6 are complete in code; remaining drift is primarily UI, governance, packaging, and ops work.

### Phase 1: Core v2 Endpoints (Week 1)

- Implement `/api/v2/health`, `/api/v2/version`
- Implement `/api/v2/projects` CRUD
- Implement `/api/v2/projects/{projectId}/tasks` CREATE and LIST
- Implement `/api/v2/tasks/{taskId}` GET
- Run database migrations 0005

**Deliverable:** Basic multi-project task management

### Phase 2: Execution Ledger (Week 2)

- Run migrations 0006 (runs, steps, logs, artifacts)
- Implement `/api/v2/runs/{runId}` GET
- Implement `/api/v2/runs/{runId}/steps` POST
- Implement `/api/v2/runs/{runId}/logs` POST
- Implement `/api/v2/runs/{runId}/artifacts` POST

**Deliverable:** Full execution tracking

### Phase 3: Leases & Claiming (Week 2-3)

- Run migration 0007 (leases)
- Implement `/api/v2/tasks/{taskId}/claim`
- Implement lease heartbeat and expiration
- Implement `/api/v2/leases/{leaseId}/heartbeat`
- Implement `/api/v2/leases/{leaseId}/release`
- Add background job for lease expiration cleanup

**Deliverable:** Multi-agent coordination

### Phase 4: Structured Completion (Week 3)

- Implement `/api/v2/tasks/{taskId}/complete`
- Add logic to generate v1 `output` from v2 `result`
- Implement status_v2 → status_v1 mapping
- Update tests for v1/v2 compatibility

**Deliverable:** Rich completion metadata

### Phase 5: Events & Memory (Week 4)

- Run migrations 0008 (events), 0009 (memory)
- Implement project-scoped SSE
- Implement event replay endpoint
- Implement memory PUT/GET endpoints
- Integrate event publishing across all mutations

**Deliverable:** Real-time updates and persistent knowledge

### Phase 6: Context Packs (Week 4)

- Run migration 0010 (context_packs)
- Implement context pack generation algorithm
- Implement `/api/v2/projects/{projectId}/context-packs` POST
- Integrate into claim flow (optional generation)

**Deliverable:** Automated context gathering

### Phase 7: Polish & Documentation (Week 5)

- Add comprehensive error handling
- Validate OpenAPI spec with Swagger UI
- Write API usage examples
- Performance testing and optimization
- Security audit

**Deliverable:** Production-ready v2 API

## Future Enhancements

### Authentication & Authorization

- JWT-based authentication
- Role-based access control (RBAC)
- Project-level permissions
- Agent identity management

### Advanced Features

- Webhooks for external integrations
- GraphQL endpoint for flexible querying
- Bulk operations (create multiple tasks, etc.)
- Task templates and workflows
- Scheduled/recurring tasks

### Observability

- Prometheus metrics export
- Distributed tracing (OpenTelemetry)
- Structured logging
- Performance profiling endpoints

### Repo Perception

- Deep tree snapshots (filtering, paging, ignore rules)
- Real file change tracking for `/api/v2/projects/{projectId}/changes` (git or filesystem scan)
- Git integration for diffs and blame info
- Git integration for diffs and blame info

## Conclusion

API v2 represents a significant evolution of Cocopilot, enabling:

1. **Multi-project workspaces** for better organization
2. **Detailed execution tracking** for observability
3. **Agent coordination** for multi-agent scenarios
4. **Structured metadata** for automation and analysis
5. **Full backward compatibility** with v1

The design is pragmatic, focusing on incremental adoption and compatibility while laying the foundation for future advanced features. Implementation can proceed in focused phases, delivering value at each milestone.

---

**Status:** Implemented  
**Note:** All phases complete. See v2-summary.md for current API reference.  
**Related Documents:**
- [OpenAPI Spec](../api/openapi-v2.yaml)
- [Database Schema](../schema/v2-migrations.sql)
