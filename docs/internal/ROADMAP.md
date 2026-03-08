# Cocopilot Project Roadmap
**Complete Atomic Reference Guide**

Generated: 2026-02-10

---

## Implementation Snapshot (2026-03-04)

- Plan completion estimate: ~75% (backend v2 is robust with policy enforcement, automation governance, repo_files, and MCP/VSIX tooling all complete; UI expansion, DAG visualization, marketplace publishing, and operational tooling remain).
- `go test ./...` passes including race detection.
- CHECKLIST.md: 101/101 items complete (B1-B4 all done).
- MCP server scaffold is in place under `tools/cocopilot-mcp/` (functional with build configs and CI/CD workflows).
- VSIX scaffold is in place under `tools/cocopilot-vsix/` (functional with build configs and CI/CD workflows).
- MCP and VSIX READMEs now include release checklists.
- VSIX commands cover MCP configure (open `mcp.json`) plus start/stop MCP server from commands or status bar.
- VSIX commands now include Open API Docs, Open API Summary, and Open OpenAPI Spec shortcuts (plus related API doc openers).
- MCP tool coverage is documented and wired for tasks (create/list/update/complete/claim/save/dependencies), projects (CRUD, tasks, tree, changes, audit, events replay), events list, runs, leases, agents, config/version/health, memory, policies, context packs (list/detail), and automation tools (rules/simulate/replay).
- Basic UI pages now fetch data for agents, audit, memory, runs, graphs, and context packs, including a context pack detail view.
- Migrations `0001`-`0018` are implemented and applied by boot-time migration runner, now sourced from `migrations/`.
- Migration `0015_events_filter_indexes.sql` adds indexes for event filtering.
- Migration `0016_tasks_sort_indexes.sql` adds indexes for task list sorting.
- Migration `0017_tasks_updated_at.sql` ensures `tasks.updated_at` exists and is backfilled.
- Migration `0018_policies.sql` adds the policies table for policy engine storage.
- Migration `0019_rate_limit_state.sql` adds rate_limit_state table for policy rate limiting.
- Migration `0020_automation_depth.sql` adds automation_depth column for recursion tracking.
- Migration `0021_repo_files.sql` adds repo_files table for file metadata persistence.
- Migration `0022_automation_emissions.sql` adds automation_emissions table for emission deduplication.
- Docs now align migration directory references to `migrations/` across roadmap/state notes.
- Migration `0014` docs/test notes updated for events `project_id` backfill.
- Projects, runs, agents, and leases are active in backend code.
- v2 runs sub-resources are implemented:
  - `GET /api/v2/runs/{runId}` returns run detail
  - `POST /api/v2/runs/{runId}/steps` records run steps
  - `POST /api/v2/runs/{runId}/logs` appends run logs
  - `POST /api/v2/runs/{runId}/artifacts` attaches artifacts
- v2 project memory endpoints are implemented:
  - `PUT /api/v2/projects/{projectId}/memory` stores memory entries
  - `GET /api/v2/projects/{projectId}/memory` supports `scope`, `key`, and `q` filters
- v2 context packs are implemented:
  - `POST /api/v2/projects/{projectId}/context-packs` creates context packs for tasks
  - `GET /api/v2/projects/{projectId}/context-packs/{contextPackId}` returns context pack detail
- v2 project tree endpoint is implemented:
  - `GET /api/v2/projects/{projectId}/tree` returns a shallow workdir snapshot
- v2 project changes endpoint is implemented:
  - `GET /api/v2/projects/{projectId}/changes` returns git status-based working tree changes with optional `since`
- v2 project audit endpoint is implemented:
  - `GET /api/v2/projects/{projectId}/audit` supports filters and pagination with `total`
- v2 agent detail endpoint is implemented:
  - `GET /api/v2/agents/{agentId}` returns agent details
- v2 agent delete endpoint is implemented:
  - `DELETE /api/v2/agents/{agentId}` deletes an agent
- v2 agent list filters are implemented:
  - `GET /api/v2/agents` supports `status` (`active`/`stale`), `since`, `limit`, and `offset` filters with `total` count
- v2 agent list sorting is implemented:
  - `GET /api/v2/agents` supports `sort` values `created_at`, `last_seen:asc`, and `last_seen:desc`
- Lease-based claiming is implemented:
  - `GET /task` acquires exclusive leases before returning work.
  - Concurrent claims resolve safely to a single winner.
  - Expired leases are cleaned and abandoned tasks are requeued.
  - Lease APIs implemented: `POST /api/v2/leases`, `POST /api/v2/leases/{leaseId}/heartbeat`, `POST /api/v2/leases/{leaseId}/release`.
  - Lease lifecycle events emitted: `lease.created`, `lease.expired`, `lease.released`.
- Standardized v2 error envelope is implemented across current v2 handlers:
  - shape: `{"error":{"code","message","details?"}}`
  - no remaining plain-text error responses on v2 paths
- OpenAPI runtime parity pass completed for shipped v2 endpoints:
  - added agents + lease-create endpoint coverage in OpenAPI
  - aligned version/project-patch/lease/run response contracts with runtime
  - marked design-only endpoints with `x-runtime-status: planned`
- `GET /api/v2/version` now includes a retention config snapshot (`retention.enabled`, `interval_seconds`, `max_rows`, `days`).
- v2 config endpoint is implemented: `GET /api/v2/config` returns a redacted runtime config snapshot (auth, retention, SSE).
- Route-level v2 contract tests are now in place through mux dispatch:
  - method-not-allowed envelope behavior validated across key v2 routes
  - representative routed success + not-found contracts validated
- Auth foundation is now implemented for v2 routes (opt-in):
  - shared API key enforcement on mutating endpoints
  - optional read-endpoint enforcement toggle
  - unauthorized responses follow standard v2 error envelope
- v2 events SSE stream is covered by auth guardrails when v2 read protection is enabled
- Scoped auth identities are now supported:
  - identity format: `id|type|api_key|scope1,scope2;...`
  - endpoint-level scope checks produce `FORBIDDEN` on insufficient permissions
- Auth audit and policy observability are implemented:
  - structured auth decision logs with identity/scope/result context
  - auth denials persisted to events with key rotation + scope rollout playbook
- Policy engine foundation is in place with persistence for policy definitions; runtime enforcement is still pending.
- B1: Policy Runtime Enforcement is complete:
  - PolicyEngine evaluates policies with scope/action matching and rate limiting.
  - Rate limiter middleware enforces request rate limits per policy.
  - Policy enable/disable endpoints allow toggling policies at runtime.
  - Enforcement integrated into v2 task mutation handlers.
- B2: Automation Governance is complete:
  - Recursion depth tracking prevents infinite automation loops (COCO_MAX_AUTOMATION_DEPTH).
  - Rate limiting caps automation executions per hour/minute (COCO_AUTOMATION_RATE_LIMIT, COCO_AUTOMATION_BURST_LIMIT).
  - Circuit breaker halts automation on repeated failures.
  - Audit trail events emitted for all automation actions.
  - Emission deduplication prevents duplicate task creation from same trigger.
- B3: repo_files Feature is complete:
  - Schema, models, and DB operations for file metadata persistence.
  - 5 API endpoints for file CRUD and listing.
  - File scanner with .gitignore support, language detection, and SHA256 hashing.
  - Context pack integration for including file metadata in context.
  - Server-side scan endpoint: POST /api/v2/projects/{id}/files/scan.
- B4: MCP/VSIX Packaging is complete:
  - MCP tools/resources/prompts fully implemented.
  - VSIX commands/config wired and functional.
  - Build configs (tsconfig, webpack, esbuild) in place.
  - CI/CD workflows configured for both packages.
  - Documentation and release checklists updated.
- v2 project policies endpoints are implemented (list/detail/create/update/delete) with filters, pagination, and sorting.
- v2 task detail endpoint is implemented:
  - `GET /api/v2/tasks/{taskId}` returns task detail with parent chain and latest run
- v2 task update endpoint is implemented:
  - `PATCH /api/v2/tasks/{taskId}` updates task fields via the v2 API
- v2 task create endpoint is implemented:
  - `POST /api/v2/tasks` creates a task with optional project and parent inputs
- v2 task list endpoint is implemented:
  - `GET /api/v2/tasks` supports `project_id`, `status`, `type`, `tag`, and `q` filters
  - supports `limit`/`offset` pagination and returns a `total` count
  - supports `sort` with `created_at:asc|desc` and `updated_at:asc|desc`
- v2 project tasks list endpoint is implemented:
  - `GET /api/v2/projects/{projectId}/tasks` supports `status`, `type`, `tag`, and `q` filters
  - supports `limit`/`offset` pagination and returns a `total` count
  - supports `sort` with `created_at:asc|desc` and `updated_at:asc|desc`
- v2 task claim endpoint is implemented:
  - `POST /api/v2/tasks/{taskId}/claim` claims a task via the v2 API
- v2 task complete endpoint is implemented:
  - `POST /api/v2/tasks/{taskId}/complete` completes a task, releases leases, and completes the latest run
- Automation engine baseline is implemented with full governance controls:
  - `task.completed` events emit configurable follow-up tasks via `COCO_AUTOMATION_RULES`
  - Automation API endpoints include rules, simulate, and replay
  - Governance: recursion depth limits, rate limiting, circuit breaker, emission dedupe
- v2 task delete endpoint is implemented:
  - `DELETE /api/v2/tasks/{taskId}` deletes a task via the v2 API
- v2 task dependencies endpoints are implemented:
  - `POST /api/v2/tasks/{taskId}/dependencies` creates a dependency
  - `GET /api/v2/tasks/{taskId}/dependencies` lists dependencies for a task
  - `DELETE /api/v2/tasks/{taskId}/dependencies/{dependsOnTaskId}` removes a dependency
- Dependency cycle detection now rejects circular task graphs (409 conflict).
- Task dependency events emitted on create/delete: `task.dependency.created`, `task.dependency.deleted`.
- v2 events list endpoint is implemented:
  - `GET /api/v2/events` supports `type`, `since`, `task_id`, `project_id`, `limit`, and `offset` filters with `total` count
  - Existing events are backfilled with `project_id` for filter parity
- v2 events SSE stream is implemented:
  - `GET /api/v2/events/stream` supports `project_id` scoping with optional `type` filter and `since` replay with `limit`
  - SSE heartbeat interval is configurable via `COCO_SSE_HEARTBEAT_SECONDS`
  - SSE replay limit cap is configurable via `COCO_SSE_REPLAY_LIMIT_MAX`
- v2 project events replay endpoint is implemented:
  - `GET /api/v2/projects/{projectId}/events/replay` replays events since `since_id` with optional `limit`
- Events retention cleanup runs hourly by default when `COCO_EVENTS_RETENTION_DAYS` or `COCO_EVENTS_RETENTION_MAX_ROWS` is enabled (prune interval configurable via `COCO_EVENTS_PRUNE_INTERVAL_SECONDS`) and logs prune outcomes with deleted counts and durations
- Task mutations now maintain `tasks.updated_at` on claim, status changes, and completion
- v2 task response payloads now include `updated_at` timestamps
- v1 task list responses now include `updated_at` timestamps
- v1 `GET /task` responses now include `updated_at` timestamps
- v1 `POST /create` responses now include `updated_at` timestamps
- v1 `POST /save` responses now include `updated_at` timestamps
- v1 `POST /update-status` responses now include `updated_at` timestamps
- v1 `GET /api/tasks` supports `status`, `updated_since`, and `project_id` filters
- v1 `GET /api/tasks` supports `sort` (`created_at:asc`, `created_at:desc`, `updated_at`)
- v1 `GET /api/tasks` supports `limit`/`offset` pagination with `total` count
- v1 `GET /events` supports `project_id`, `type=tasks`, and `since` replay with optional `limit` (capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`)

---

## Table of Contents
Progress snapshot: [COMPLETION_SUMMARY.md](COMPLETION_SUMMARY.md)
1. [Overview](#overview)
2. [Current State (PoC)](#current-state-poc)
3. [Future Vision](#future-vision)
4. [Roadmap by Phase](#roadmap-by-phase)
5. [Database Schema Evolution](#database-schema-evolution)
6. [API Specifications](#api-specifications)
7. [Automation Engine Details](#automation-engine-details)
8. [UI Implementation Details](#ui-implementation-details)
9. [MCP Server & VSIX Details](#mcp-server--vsix-details)
10. [Testing & Acceptance Criteria](#testing--acceptance-criteria)
11. [Implementation Guidelines](#implementation-guidelines)
12. [Guiding Principles](#guiding-principles)

---

## Overview

Cocopilot is a task orchestration system designed to manage and coordinate tasks for Large Language Model (LLM) agents. It serves as a mission control system, providing a Kanban-style web interface for humans to manage tasks and a RESTful API for agents to interact with tasks programmatically.

### Project Evolution
- **Current**: Proof-of-Concept (PoC) task queue with basic Kanban UI
- **Goal**: Durable, context-aware "project brain" enabling autonomous agent decisions
- **Architecture**: Go + SQLite backend, web-based frontend, real-time SSE updates

### Key Documents Reference
- `docs/api/v2-summary.md`: API v2 endpoint reference
- `docs/state/architecture.md`: System architecture
- `MIGRATIONS.md`: Migration system details
- `migrations/`: SQL migration files

---

## Current State (PoC)

### Existing Functionality
Migrations `0001`-`0018` are applied from `migrations/`, including `0013_task_dependencies.sql`, `0014_events_project_id_backfill.sql`, `0015_events_filter_indexes.sql`, `0016_tasks_sort_indexes.sql`, `0017_tasks_updated_at.sql`, and `0018_policies.sql`.
Documentation references now align to the `migrations/` directory, with `0014` docs/test notes updated and task sort indexes applied.

1. **Task Lifecycle**:
  - Create: `POST /create` with instructions and optional parent_task_id, returns `updated_at`
  - Claim: `GET /task` returns next NOT_PICKED task, marks as IN_PROGRESS, includes `updated_at`
  - Complete: `POST /save` with task_id and output, marks as COMPLETE, returns `updated_at`
  - Update Status: `POST /update-status` updates task status, returns `updated_at`
  - Monitor: `GET /events` (SSE) streams real-time updates; supports `project_id`, `type=tasks`, `since`, and optional replay `limit` (capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`)
2. **v2 Health/Version**:
  - `GET /api/v2/version` includes a retention config snapshot (`retention.enabled`, `interval_seconds`, `max_rows`, `days`)
  - `GET /api/v2/config` returns a redacted runtime config snapshot (auth, retention, SSE)
2. **v2 Task Create**:
  - `POST /api/v2/tasks` creates a task with optional project and parent inputs

3. **v2 Agent Detail**:
  - `GET /api/v2/agents/{agentId}` returns agent details

4. **v2 Agent Delete**:
  - `DELETE /api/v2/agents/{agentId}` deletes an agent

4. **v2 Agents List**:
  - `GET /api/v2/agents` supports `status` (`active`/`stale`), `since`, `limit`, `offset`, and `sort` (`created_at`, `last_seen:asc`, `last_seen:desc`) with `total` count

4. **v2 Task Detail**:
  - `GET /api/v2/tasks/{taskId}` returns task detail with parent chain and latest run

4. **v2 Task Update**:
  - `PATCH /api/v2/tasks/{taskId}` updates task fields

5. **v2 Task List**:
  - `GET /api/v2/tasks` supports `project_id` and `status` filters, plus `limit`/`offset` pagination with a `total` count
  - `GET /api/v2/tasks` supports `sort` with `created_at:asc|desc` and `updated_at:asc|desc`
  - Task responses include `updated_at` timestamps

6. **v2 Project Tasks List**:
  - `GET /api/v2/projects/{projectId}/tasks` supports `status` filters, plus `limit`/`offset` pagination with a `total` count
  - `GET /api/v2/projects/{projectId}/tasks` supports `sort` with `created_at:asc|desc` and `updated_at:asc|desc`

7. **v2 Events List**:
  - `GET /api/v2/events` supports `type`, `since`, `task_id`, `project_id`, `limit`, and `offset` filters with `total` count
  - Existing events are backfilled with `project_id` for filter parity

8. **v2 Events SSE Stream**:
  - `GET /api/v2/events/stream` supports `project_id` scoping with optional `type` filter and `since` replay with `limit`
  - Subject to v2 auth read guardrails when enabled
  - SSE heartbeat interval is configurable via `COCO_SSE_HEARTBEAT_SECONDS`
  - SSE replay limit cap is configurable via `COCO_SSE_REPLAY_LIMIT_MAX`

9. **Events Retention Cleanup**:
  - Hourly pruning by default when `COCO_EVENTS_RETENTION_DAYS` or `COCO_EVENTS_RETENTION_MAX_ROWS` is enabled (prune interval configurable via `COCO_EVENTS_PRUNE_INTERVAL_SECONDS`)
  - Prune runs log deleted counts and durations, with skip logs when SQLite is busy

7. **v2 Task Claim**:
  - `POST /api/v2/tasks/{taskId}/claim` claims a task for execution

8. **v2 Task Complete**:
  - `POST /api/v2/tasks/{taskId}/complete` completes a task and releases leases

9. **v2 Task Delete**:
  - `DELETE /api/v2/tasks/{taskId}` deletes a task

10. **v2 Task Dependencies**:
  - `POST /api/v2/tasks/{taskId}/dependencies` creates a dependency
  - `GET /api/v2/tasks/{taskId}/dependencies` lists dependencies for a task
  - `DELETE /api/v2/tasks/{taskId}/dependencies/{dependsOnTaskId}` removes a dependency
  - Dependency creation rejects cycles with a 409 conflict
  - Dependency lifecycle events are emitted on create/delete

8. **Web UI**:
   - Three-column Kanban: To Do, In Progress, Done
   - Drag-and-drop task status updates
   - Task creation modal with parent task selection
   - Real-time updates via SSE
   - Working directory management

8. **Database Schema (Current)**:
   ```sql
   CREATE TABLE tasks (
     id INTEGER PRIMARY KEY AUTOINCREMENT,
     instructions TEXT NOT NULL,
     status TEXT NOT NULL DEFAULT 'NOT_PICKED',
     output TEXT,
     parent_task_id INTEGER,
     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
     FOREIGN KEY (parent_task_id) REFERENCES tasks(id)
   );
   ```

### Existing Endpoints (v1)
- `GET /task`: Fetch next task + context from parent chain (includes `updated_at`)
- `POST /create`: Create new task (returns `updated_at`)
- `POST /save`: Complete task with output (returns `updated_at`)
- `POST /update-status`: Update task status (returns `updated_at`)
- `POST /delete`: Delete task
- `GET /api/tasks`: List all tasks (JSON, includes `updated_at` timestamps; supports `status`, `updated_since`, `project_id`, `sort`, `limit`, and `offset` with `total` count)
- `GET /events`: SSE stream for real-time updates (supports `project_id`, `type=tasks`, `since` replay, and optional `limit` capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`)
- `GET /api/workdir`: Get current working directory
- `POST /set-workdir`: Set working directory
- `GET /instructions`: Get agent setup instructions
- `GET /`: Serve Kanban UI

---

## Future Vision

### Core Capabilities
1. **Execution Ledger**: Full audit trail of runs, steps, artifacts, and events
2. **Multi-Agent Coordination**: Safe task claiming with leases and heartbeats  
3. **Durable Memory**: Persistent context storage independent of chat sessions
4. **Project Management**: Multi-project support with scoped tasks and settings
5. **Automation Engine**: Event-driven task creation and orchestration
6. **Repository Awareness**: File tree monitoring and change detection
7. **Mission Control UI**: Comprehensive monitoring and management interface
8. **IDE Integration**: VS Code extension for seamless developer workflow

### Key Innovations
- **Context Packs**: Immutable bundles of relevant code, history, and context
- **Event-First Architecture**: All state changes captured in append-only event log
- **Deterministic Automation**: Reproducible task creation based on events and rules
- **Observable Autonomy**: Complete visibility into agent actions and decisions

---

## Roadmap by Phase

## Phase 0: Foundation and Stability

### 0.1 POC Regression Suite
**Purpose**: Ensure v1 stability throughout evolution

**Test Cases**:
```bash
# POC-REG-001: Create → Claim → Save lifecycle
curl -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \
  -d '{"instructions":"POC-REG-001: say hello","parent_task_id":null}'

curl -sS http://localhost:8080/task
# Assert: Response includes id, instructions, status transition to IN_PROGRESS

curl -X POST http://localhost:8080/save \
  -H "Content-Type: application/json" \
  -d '{"id":<ID>,"output":"hello"}'
# Assert: Task marked COMPLETE, output saved

# POC-REG-002: Parent task context preservation
# Create parent P, complete with output, create child C
# Assert: Child task includes parent context block

# POC-REG-003: SSE events stream
curl -N http://localhost:8080/events
# Assert: Events received for create/update/save/delete operations

# POC-REG-004: Workdir management
curl -X POST http://localhost:8080/set-workdir \
  -d '{"workdir":"/tmp/coco-workdir"}'
curl http://localhost:8080/api/workdir
# Assert: Workdir persisted and retrievable
```

**Automation**:
- Script: `scripts/poc_regression.sh`
- CI Integration: Must pass on every PR
- Test Environment: Isolated DB (`./tmp/test.db`)

### 0.2 Schema Migrations System
**Migration Runner Requirements**:
1. On server boot: read max applied version from `schema_migrations`
2. Apply migrations from `migrations/` directory in ascending order
3. Insert version row after each successful migration
4. Fail fast on errors (no partial application)
5. Single-process boot (app-level mutex)

**Implementation**:
```go
func runMigrations() error {
    maxVersion := getMaxAppliedVersion()
    migrationFiles := listMigrationFiles("migrations/")
    
    for _, file := range migrationFiles {
        version := extractVersionFromFilename(file)
        if version > maxVersion {
            if err := executeMigration(file); err != nil {
                return err
            }
            recordAppliedVersion(version)
        }
    }
    return nil
}
```

### 0.3 v2 Health/Version Endpoints
```http
GET /api/v2/health
Response: {"ok": true}

GET /api/v2/version
Response: {
  "service": "cocopilot",
  "api": {"v1": true, "v2": true},
  "schema_version": 4
}
```

---

## Phase 1: Projects and Execution Ledger

### 1.1 Project Management
**Database Schema**:
```sql
-- Migration 0003_projects.sql
CREATE TABLE IF NOT EXISTS projects (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  workdir       TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  settings_json TEXT
);

INSERT INTO projects (id, name, workdir, settings_json)
SELECT 'proj_default', 'Default', '', NULL
WHERE NOT EXISTS (SELECT 1 FROM projects WHERE id='proj_default');

-- Migration 0004_tasks_add_project_id.sql
ALTER TABLE tasks ADD COLUMN project_id TEXT;
UPDATE tasks SET project_id = 'proj_default' WHERE project_id IS NULL;
CREATE INDEX idx_tasks_project_status_created_at ON tasks(project_id, status, created_at);
```

**API Endpoints**:
```http
POST /api/v2/projects
{
  "name": "string",
  "workdir": "string", 
  "settings": {}
}
Response: {"project": {...}}

GET /api/v2/projects
Response: {"projects": [...]}

GET /api/v2/projects/{projectId}
Response: {"project": {...}}

PUT /api/v2/projects/{projectId}
{
  "name": "string?",
  "workdir": "string?", 
  "settings": {}
}
```

**v1 Compatibility**:
- `POST /set-workdir` updates `proj_default.workdir`
- `GET /api/workdir` reads from `proj_default.workdir`
- All v1 tasks automatically associated with `proj_default`

### 1.2 Execution Ledger
**Database Schema**:
```sql
-- Migration 0006_runs.sql
CREATE TABLE runs (
  id          TEXT PRIMARY KEY,
  task_id     INTEGER NOT NULL,
  agent_id    TEXT,
  status      TEXT NOT NULL, -- RUNNING|SUCCEEDED|FAILED|CANCELLED
  started_at  TEXT NOT NULL,
  finished_at TEXT,
  exit_code   INTEGER,
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- Migration 0007_run_steps.sql  
CREATE TABLE run_steps (
  id        TEXT PRIMARY KEY,
  run_id    TEXT NOT NULL,
  sequence  INTEGER NOT NULL,
  name      TEXT NOT NULL,
  status    TEXT NOT NULL, -- STARTED|SUCCEEDED|FAILED
  details   TEXT, -- JSON
  started_at TEXT NOT NULL,
  finished_at TEXT,
  FOREIGN KEY (run_id) REFERENCES runs(id)
);

-- Migration 0008_artifacts.sql
CREATE TABLE artifacts (
  id          TEXT PRIMARY KEY,
  run_id      TEXT NOT NULL,
  kind        TEXT NOT NULL, -- diff|patch|log|report|file
  storage_ref TEXT NOT NULL,
  sha256      TEXT,
  size        INTEGER,
  metadata    TEXT, -- JSON
  created_at  TEXT NOT NULL,
  FOREIGN KEY (run_id) REFERENCES runs(id)
);
```

**API Endpoints**:
```http
GET /api/v2/runs/{runId}
Response: {
  "run": {...},
  "steps": [...],
  "artifacts": [...],
  "tool_invocations": [...]
}

POST /api/v2/runs/{runId}/steps
{
  "name": "string",
  "status": "STARTED|SUCCEEDED|FAILED",
  "details": {}
}

POST /api/v2/runs/{runId}/logs  
{
  "stream": "stdout|stderr|info",
  "chunk": "string",
  "ts": "ISO8601?"
}

POST /api/v2/runs/{runId}/artifacts
{
  "kind": "diff|patch|log|report|file",
  "storage_ref": "string", 
  "sha256": "string?",
  "size": 123,
  "metadata": {}
}
```

**v1 Integration**:
- `POST /save` creates a run record with status SUCCEEDED
- `GET /task` creates a run record with status RUNNING

### 1.3 UI Enhancements

**Project Selector**:
- Location: Header topbar dropdown
- Data Source: `GET /api/v2/projects`
- Behavior: Switching project updates Kanban task list
- Hidden when only one project exists

**Task Detail Drawer v2**:
- Trigger: Click on Kanban card
- Data Sources: 
  - `GET /api/v2/tasks/{taskId}` 
  - Fallback to `GET /api/tasks` filter by id
- UI Elements:
  - Title, type, priority, tags
  - Parent chain viewer (expand/collapse)
  - Latest run summary block
  - Link to run viewer

**Run Viewer**:
- Route: `/runs/{runId}`
- Data Sources: 
  - `GET /api/v2/runs/{runId}`
  - `GET /api/v2/projects/{projectId}/events/stream` (SSE)
- UI Elements:
  - Step timeline (ordered)
  - Live log console (tail mode)
  - Artifacts list with download links
  - Tool invocations list
- Acceptance: Live logs appear within 1s, page refresh reconstructs state

---

## Phase 2: Leases and Heartbeats

### 2.1 Safe Task Claiming
**Database Schema**:
```sql
-- Migration 0010_leases.sql
CREATE TABLE leases (
  id          TEXT PRIMARY KEY,
  task_id     INTEGER NOT NULL,
  agent_id    TEXT NOT NULL,
  expires_at  TEXT NOT NULL,
  created_at  TEXT NOT NULL,
  UNIQUE(task_id), -- Only one active lease per task
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- Migration 0005_agents.sql
CREATE TABLE agents (
  id           TEXT PRIMARY KEY,
  capabilities TEXT, -- JSON
  version      TEXT,
  last_seen    TEXT NOT NULL,
  status       TEXT NOT NULL -- ONLINE|OFFLINE
);
```

**API Endpoints**:
```http
POST /api/v2/tasks/{taskId}/claim
{
  "agent_id": "agent_x", 
  "mode": "exclusive"
}
Response: {
  "lease": {"id": "lease_x", "expires_at": "ISO8601"},
  "run": {"id": "run_x", "status": "RUNNING"}, 
  "context_pack_id": "pack_x?"
}
Errors: 409 CONFLICT if already leased

POST /api/v2/leases/{leaseId}/heartbeat
Response: {"lease": {"expires_at": "ISO8601"}}

POST /api/v2/leases/{leaseId}/release
{"reason": "string?"}
```

**Lease Management**:
- Default lease duration: 15 minutes
- Heartbeat interval: 5 minutes  
- Lease extension: +15 minutes on heartbeat
- Auto-release: Background process expires stale leases

### 2.2 Agent Dashboard
**Route**: `/agents`
**Data Sources**: 
- `GET /api/v2/agents`
- `GET /api/v2/projects/{projectId}/events/stream` (SSE)

**UI Elements**:
- Agent cards showing online/offline status
- Current run links
- Lease expiry countdown
- Last heartbeat timestamp
- Agent capabilities and version

**Agent State Transitions**:
- ONLINE: Recent heartbeat within threshold
- OFFLINE: No heartbeat beyond threshold (default: 2x heartbeat interval)

---

## Phase 3: Repo Perception

Implementation Notes (2026-02-11): `GET /api/v2/projects/{projectId}/tree` and `GET /api/v2/projects/{projectId}/changes` are implemented with a shallow workdir snapshot and git status-based working tree changes.

### 3.1 File System Monitoring
**Database Schema**:
```sql  
-- Migration 0011_repo_files.sql
CREATE TABLE repo_files (
  id           TEXT PRIMARY KEY,
  project_id   TEXT NOT NULL,
  path         TEXT NOT NULL,
  kind         TEXT NOT NULL, -- file|dir
  size         INTEGER,
  sha256       TEXT,
  last_modified TEXT NOT NULL,
  git_ref      TEXT,
  UNIQUE(project_id, path),
  FOREIGN KEY (project_id) REFERENCES projects(id)
);
```

**API Endpoints**:
```http
GET /api/v2/projects/{projectId}/tree
Response: {
  "tree": {
    "path": ".",
    "kind": "dir", 
    "children": [
      {"path": "cmd", "kind": "dir", "children": []},
      {"path": "main.go", "kind": "file", "size": 1234}
    ]
  }
}

GET /api/v2/projects/{projectId}/changes?since=ISO8601
Response: {
  "changes": [
    {"path": "main.go", "kind": "modified", "sha256": "...", "ts": "ISO8601"}
  ]
}
```

### 3.2 Repo Panel UI  
**Route**: `/repo` 
**UI Elements**:
- Tree explorer (expandable/collapsible)
- Recent changes feed with timestamps
- File metadata on selection (hash/size/language)
- "Recent tasks touching this file" (future)

**File Change Detection**:
- File watcher on project workdir
- Git integration for branch/dirty state
- Events: `repo.file_changed`, `repo.git_state_changed`
- Change feed updates within 2s of detection

---

## Phase 4: Memory and Context Packs

### 4.1 Durable Memory
**Database Schema**:
```sql
-- Migration 0012_memory.sql
CREATE TABLE memory_items (
  id          TEXT PRIMARY KEY,
  project_id  TEXT NOT NULL,
  scope       TEXT NOT NULL, -- GLOBAL|MODULE|FILE|TASK|RUN
  key         TEXT NOT NULL,
  value       TEXT NOT NULL, -- JSON
  source_refs TEXT, -- JSON array
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL,
  UNIQUE(project_id, scope, key),
  FOREIGN KEY (project_id) REFERENCES projects(id)
);
```

**API Endpoints**:
```http
PUT /api/v2/projects/{projectId}/memory
{
  "scope": "GLOBAL|MODULE|FILE|TASK|RUN",
  "key": "string",
  "value": {},
  "source_refs": [
    {"type": "task", "id": 123},
    {"type": "run", "id": "run_x"},
    {"type": "file", "path": "src/auth.go", "ref": "git:abc123"}
  ]
}

GET /api/v2/projects/{projectId}/memory?scope=&key=&q=
Response: {
  "items": [
    {
      "id": "mem_x",
      "scope": "GLOBAL", 
      "key": "architecture",
      "value": {...},
      "updated_at": "ISO"
    }
  ]
}
```

### 4.2 Context Packs
**Database Schema**:
```sql
-- Migration 0013_context_packs.sql  
CREATE TABLE context_packs (
  id          TEXT PRIMARY KEY,
  task_id     INTEGER,
  query       TEXT,
  summary     TEXT NOT NULL,
  contents    TEXT NOT NULL, -- JSON
  created_at  TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);
```

**API Endpoints**:
```http
POST /api/v2/projects/{projectId}/context-packs
{
  "task_id": 123,
  "query": "string?",
  "budget": {"max_files": 25, "max_bytes": 500000, "max_snippets": 200}
}
Response: {
  "context_pack": {
    "id": "pack_x",
    "task_id": 123,
    "summary": "string",
    "contents": {
      "files": [
        {
          "path": "src/auth/login.go",
          "snippets": [{"start": 10, "end": 80, "text": "..."}]
        }
      ],
      "related_tasks": [123, 456],
      "decisions": ["dec_x"],
      "repo_state": {"git_head": "abc123", "dirty": true}
    }
  }
}
```

**Context Pack Building Rules**:
1. **File Priority**: Active editor > Open editors > Recently modified > Git history
2. **Snippet Selection**: Function boundaries, imports, relevant classes  
3. **Budget Enforcement**: Respect max_files, max_bytes, max_snippets limits
4. **Immutability**: Packs never change after creation (audit trail)

### 4.3 Memory & Context UI

**Memory Panel** (`/memory`):
- Filter by scope (GLOBAL, MODULE, FILE, TASK, RUN)
- Search by key or content
- Edit/create memory item modal
- Source refs with links to tasks/runs/files
- Real-time updates via SSE

**Context Pack Inspector** (`/context-packs/{packId}`):
- Pack summary and metadata
- File/snippet viewer with syntax highlighting
- Related tasks list with links
- Repo state at time of pack creation
- Read-only interface (immutable audit trail)
- API gap: OpenAPI has no context pack detail endpoint; suggest `GET /api/v2/context-packs/{packId}` returning `context_pack` payload

---

## Phase 5: Automation Engine

### 5.1 Event-Driven Architecture
**Database Schema**:
```sql
-- Migration 0009_events.sql
CREATE TABLE events (
  id          TEXT PRIMARY KEY,
  project_id  TEXT NOT NULL,
  kind        TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id   TEXT NOT NULL,
  payload     TEXT NOT NULL, -- JSON
  created_at  TEXT NOT NULL,
  FOREIGN KEY (project_id) REFERENCES projects(id)
);

-- Migration 0014_automation_emissions.sql
CREATE TABLE automation_emissions (
  dedupe_key  TEXT PRIMARY KEY,
  task_id     INTEGER NOT NULL,
  created_at  TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES tasks(id)
);
```

**Event Types**:
- `task.created`, `task.updated`, `task.completed`, `task.blocked`
- `run.started`, `run.completed`, `run.failed`
- `repo.file_changed`, `repo.git_state_changed`
- `memory.updated`
- `lease.created`, `lease.expired`

### 5.2 Blocker Classification & Automation Rules

**Blocker Types**:
1. `MISSING_INFO`: Insufficient context or requirements
2. `TEST_FAIL`: Test suite failures  
3. `LINT_FAIL`: Code quality issues
4. `BUILD_FAIL`: Compilation errors
5. `ENV_SETUP`: Environment configuration issues
6. `CONFLICT`: Merge conflicts or file conflicts
7. `SCOPE_GAP`: Incomplete acceptance criteria
8. `POLICY_BLOCK`: Permission or policy violations
9. `FLAKY`: Intermittent/non-deterministic failures
10. `UNKNOWN`: Unclassified blocker

**Canonical Automation Triggers**:

**Trigger A - Run Failed → TRIAGE + FIX + VERIFY**:
```
When: run.completed with status=FAILED
Actions:
1. Create TRIAGE task (type=ANALYZE): 
   - Instructions: "Summarize failure, identify culprit files, propose fix plan"
   - Evidence: run logs, artifacts, repo state
   
2. Create FIX task (type=MODIFY):
   - Depends on: TRIAGE task
   - Instructions: "Implement fix based on triage analysis"
   
3. Create VERIFY task (type=TEST):  
   - Depends on: FIX task
   - Instructions: "Run tests to verify fix resolves issue"

Dedupe Key: sha256(project + parent_task + run_id + "run_failed")
```

**Trigger B - Task Blocked → VALIDATION**:
```
When: task.blocked or completion with status=BLOCKED  
Actions:
1. Create VALIDATION task (type=REVIEW):
   - Tag: needs_human
   - Instructions: "Review blocked task and provide guidance"
   - Keep parent BLOCKED until validation completes

Dedupe Key: hash(parent_task + normalized_questions)
```

**Trigger C - Acceptance Gap → Criterion Tasks**:
```
When: completion indicates SCOPE_GAP
Actions:  
1. For each missing criterion, create tasks:
   - DOC task for documentation gaps
   - TEST task for test coverage gaps  
   - MODIFY task for implementation gaps
   - Dependencies ensure proper ordering

Dedupe Key: hash(parent_task + criterion_type + criterion_id)
```

**Trigger D - Conflict Detected → RESOLUTION Chain**:
```
When: repo.conflict_detected (git conflict markers)
Actions:
1. Create ANALYZE task: "Analyze conflict sources and impact"
2. Create MODIFY task: "Resolve conflicts" (depends on ANALYZE)  
3. Create TEST task: "Verify resolution" (depends on MODIFY)

Dedupe Key: hash(project + conflict_files + git_refs)
```

**Trigger E - Policy Block → APPROVAL Gate**:
```  
When: tool invocation denied by policy
Actions:
1. Create APPROVAL task (type=REVIEW):
   - Tag: needs_human  
   - Instructions: "Review and approve policy exception"
   - Include: original action + justification
   
2. On approval, create RESUME task:
   - Instructions: "Continue with approved action"
   - Context: original blocked action details

Dedupe Key: hash(policy_rule + action_type + resource)
```

### 5.3 Loop Prevention & Safety

**Quotas**:
- `automation.max_auto_tasks_per_root`: 25 (default)
- `automation.max_retries_per_task`: 3 (default)
- `automation.max_emissions_per_hour`: 100 (default)

**Escalation Rules**:
- Repeated identical failures → Create ESCALATE task (needs_human)
- Quota exceeded → Create REVIEW task explaining automation limits
- Infinite loop detection → Disable automation for affected root task

**Idempotency**:
```go
func emitTask(action AutomationAction) error {
    dedupeKey := computeDedupeKey(action)
    if exists, _ := db.Query("SELECT 1 FROM automation_emissions WHERE dedupe_key = ?", dedupeKey); exists {
        return nil // Already emitted
    }
    
    taskID := createTask(action.TaskPayload)
    recordEmission(dedupeKey, taskID)
    emitEvent("task.created", taskID, map[string]interface{}{"auto": true})
    return nil
}
```

### 5.4 Automation API

```http
POST /api/v2/projects/{projectId}/automation/simulate
{
  "event": {
    "kind": "run.completed", 
    "payload": {"status": "FAILED", "run_id": "run_x"}
  }
}
Response: {
  "actions": [...],
  "tasks_that_would_be_created": [...]
}

GET /api/v2/projects/{projectId}/automation/rules
Response: {"rules": [...]} // Lists active automation rules

POST /api/v2/projects/{projectId}/automation/replay?since_event_id=evt_123
// Re-runs automation over event range for testing/recovery
```

---

## Phase 6: DAG Orchestration

### 6.1 Task Dependencies  
**Database Schema**:
```sql
-- Migration 0015_task_dependencies.sql
CREATE TABLE task_dependencies (
  id            TEXT PRIMARY KEY,
  task_id       INTEGER NOT NULL,
  depends_on_id INTEGER NOT NULL,
  created_at    TEXT NOT NULL,
  UNIQUE(task_id, depends_on_id),
  FOREIGN KEY (task_id) REFERENCES tasks(id),
  FOREIGN KEY (depends_on_id) REFERENCES tasks(id)
);
```

**API Endpoints**:
```http  
POST /api/v2/tasks/{taskId}/dependencies
{"depends_on_ids": [123, 456]}

GET /api/v2/tasks/{taskId}/dependencies  
Response: {"dependencies": [...], "dependents": [...]}

GET /graphs/tasks?project_id=proj_default
Response: {
  "nodes": [{"id": 123, "title": "...", "status": "..."}],
  "edges": [{"from": 123, "to": 456, "type": "depends_on"}]
}
```

### 6.2 Task DAG Viewer
**Route**: `/graphs/tasks`
**UI Elements**:
- Interactive graph visualization (D3.js/vis.js)
- Node colors by status (queued=gray, running=blue, done=green, blocked=red)
- Edge types (depends_on, parent_child, auto_created)  
- Filters by status, type, project
- Zoom/pan controls
- Node click → task detail drawer

**DAG Features**:
- **Blocked Task Detection**: Highlight tasks waiting on dependencies
- **Critical Path**: Show longest dependency chain
- **Cycle Detection**: Warn on circular dependencies
- **Parallel Execution**: Identify tasks that can run concurrently

---

## Phase 7: Mission Control UI

### 7.1 Diff Viewer
**Route**: `/diffs/{artifactId}` or embedded in run viewer
**Data Source**: Artifact with kind='diff' or 'patch'

**UI Elements**:
- Split-pane diff view (before/after)
- Unified diff option
- File list with changed hunks
- Syntax highlighting
- Line-by-line comments (future)
- Pagination for large diffs (>1000 lines)

**Supported Formats**:
- Git unified diff
- JSON patches  
- Custom diff formats per artifact metadata

### 7.2 Advanced Graphs

**Task Dependency Graph** (`/graphs/tasks`):
- **Nodes**: Tasks with status colors
- **Edges**: Dependencies, parent relationships
- **Layouts**: Hierarchical, force-directed, circular
- **Interactions**: Hover details, click navigation, drag reposition

**Repo/Entity Graph** (`/graphs/repo`) - Optional:
- **Nodes**: Files, classes, functions
- **Edges**: Imports, calls, references
- **Analysis**: Dead code detection, coupling metrics
- **Integration**: Tasks that touch each entity

**Performance**: 
- Virtual scrolling for large graphs (>1000 nodes)
- WebGL rendering for smooth interactions
- Server-side layout computation for complex graphs

---

## Phase 8: Governance and Security

### 8.1 Structured Completion
**API Endpoint**:
```http
POST /api/v2/tasks/{taskId}/complete
{
  "run_id": "run_x",
  "status": "SUCCEEDED|FAILED|NEEDS_REVIEW",
  "result": {
    "summary": "string",
    "changes_made": ["Modified auth logic", "Updated tests"],
    "files_touched": ["src/auth.go", "tests/auth_test.go"], 
    "commands_run": ["go test", "go build"],
    "tests_run": ["TestLogin", "TestLogout"],
    "risks": ["Breaking change to auth API"],
    "next_tasks": [
      {
        "title": "Update API documentation", 
        "instructions": "Document auth API changes",
        "type": "DOC",
        "priority": 1
      }
    ]
  }
}
```

**Validation Rules**:
- Required fields per task type
- Risk assessment for MODIFY tasks  
- Test evidence for TEST tasks
- Documentation updates for API changes

### 8.2 Policy Engine
**Database Schema**:
```sql
-- Migration 0016_policies.sql
CREATE TABLE policies (
  id          TEXT PRIMARY KEY,
  project_id  TEXT NOT NULL,
  name        TEXT NOT NULL,
  description TEXT NOT NULL,
  rules       TEXT NOT NULL, -- JSON policy rules
  enabled     BOOLEAN DEFAULT true,
  created_at  TEXT NOT NULL,
  FOREIGN KEY (project_id) REFERENCES projects(id)
);
```

**Policy Types**:
- **Tool Execution**: Restrict dangerous commands
- **File Access**: Prevent modification of critical files  
- **Resource Limits**: CPU, memory, disk usage bounds
- **Time Limits**: Maximum run duration
- **Approval Gates**: Human review requirements

**Enforcement Points**:
- Before tool execution
- Before file system operations  
- Before task completion
- Before automation task creation

### 8.3 Audit Trail
**Route**: `/audit`
**Data Source**: `GET /api/v2/projects/{projectId}/events/stream`

**UI Elements**:
- Chronological event timeline  
- Filter by entity type, user, date range
- Event details with payload expansion
- Export audit log (CSV, JSON)
- Search by entity ID or content

**Event Retention**:
- Default: 90 days  
- Configurable per project
- Archive older events to storage
- Compliance reporting capabilities

---

## Phase 9: MCP Server and VSIX

### 9.1 MCP Server Implementation

**Tool Mappings**:
```typescript
interface MCPTool {
  name: string;
  description: string;
  inputSchema: JSONSchema;
}

const tools: MCPTool[] = [
  {
    name: "coco.project.list",
    description: "List all projects",
    inputSchema: {}
  },
  {
    name: "coco.task.create", 
    description: "Create a new task",
    inputSchema: {
      type: "object",
      properties: {
        instructions: {type: "string"},
        project_id: {type: "string"},
        type: {enum: ["ANALYZE", "MODIFY", "TEST", "REVIEW", "DOC"]},
        parent_task_id: {type: "number"}
      },
      required: ["instructions"]
    }
  },
  {
    name: "coco.task.claim",
    description: "Claim a task for execution",
    inputSchema: {
      type: "object", 
      properties: {
        task_id: {type: "number"},
        agent_id: {type: "string"}
      },
      required: ["task_id", "agent_id"]
    }
  }
  // ... 12 more tools (run.get, memory.query, context_pack.build, etc.)
];
```

**Resource Mappings**:
```typescript
const resources: MCPResource[] = [
  {
    uri: "coco://project/{projectId}/summary",
    name: "Project Summary",
    mimeType: "text/plain"
  },
  {
    uri: "coco://task/{taskId}",  
    name: "Task Details",
    mimeType: "application/json"
  },
  {
    uri: "coco://context-pack/{packId}",
    name: "Context Pack Contents", 
    mimeType: "application/json"
  }
];
```

**Prompt Mappings**:
```typescript
const prompts: MCPPrompt[] = [
  {
    name: "coco.plan",
    description: "Generate subtasks and dependency DAG",
    arguments: [
      {name: "goal", description: "High-level objective", required: true},
      {name: "context", description: "Additional context", required: false}
    ]
  },
  {
    name: "coco.fix", 
    description: "Fix failing tests using context packs",
    arguments: [
      {name: "run_id", description: "Failed run ID", required: true}
    ]
  }
];
```

**Transport Configuration**:
```json
// .vscode/mcp.json
{
  "servers": {
    "cocopilot": {
      "type": "stdio",
      "command": "node", 
      "args": ["./tools/cocopilot-mcp/dist/server.js"],
      "env": {
        "COCO_API_BASE": "http://localhost:8080",
        "COCO_PROJECT_ID": "proj_default"
      }
    }
  }
}
```

### 9.2 VSIX Extension ("CocoBridge")

**Packaging + Publishing Note**:
- VSIX release tasks: finalize production build, package with `vsce` (or equivalent), verify activation events/commands, and publish to the Marketplace.
- MCP release tasks: bundle the server (Node dist + assets), publish the bundle to the chosen registry or release channel, and document install/update flow.

**Extension Features**:
1. **Auto-registration**: Contributes MCP server config automatically
2. **Bundling**: Includes MCP server binaries/scripts  
3. **IDE Signal Capture**: Active file, selection, diagnostics, git state
4. **Documentation Shortcuts**: Open API docs/summary/spec and related reference docs in-editor

**VS Code Integration**:
```typescript
// Extension activation
export function activate(context: vscode.ExtensionContext) {
  // Register MCP server provider
  const mcpProvider = new CocopilotMCPProvider();
  vscode.mcp.registerServerProvider('cocopilot', mcpProvider);
  
  // Start IDE signal streaming
  const signalManager = new IDESignalManager();
  signalManager.start();
}

class IDESignalManager {
  start() {
    // Monitor active editor changes
    vscode.window.onDidChangeActiveTextEditor(this.onActiveEditorChanged);
    
    // Monitor selection changes  
    vscode.window.onDidChangeTextEditorSelection(this.onSelectionChanged);
    
    // Monitor diagnostics updates
    vscode.languages.onDidChangeDiagnostics(this.onDiagnosticsChanged);
    
    // Monitor git status
    this.gitExtension.onDidChangeStatus(this.onGitStatusChanged);
  }
}
```

**IDE Signal Payloads**:
```typescript
interface IDESignal {
  kind: 'ide.active_file.changed' | 'ide.selection.changed' | 'ide.diagnostics.updated' | 'ide.git.state.changed';
  data: any;
}

// Examples:
{
  kind: 'ide.active_file.changed',
  data: {
    path: 'src/auth/login.go',
    selection: {start: 120, end: 260},
    language: 'go'
  }
}

{
  kind: 'ide.diagnostics.updated', 
  data: {
    path: 'src/auth/login.go',
    diagnostics: [
      {line: 45, message: 'unused variable', severity: 'warning'}
    ]
  }
}
```

**API Integration**:
```http
POST /api/v2/projects/{projectId}/ide-signals
{
  "kind": "ide.active_file.changed",
  "data": {...}
}
```

### 9.3 Acceptance Tests

**MCP Server Tests**:
```bash
# AT-MCP-1: Tools available and functional
mcp-client list-tools
# Assert: cocopilot tools appear

mcp-client call-tool coco.task.create '{"instructions": "test task"}'
# Assert: Task created successfully

# AT-MCP-2: Context pack building  
mcp-client call-tool coco.context_pack.build '{"task_id": 123}'
# Assert: Immutable pack created and retrievable
```

**VSIX Tests**:
```typescript
// AT-VSIX-1: Auto-registration
suite('VSIX Integration', () => {
  test('MCP server auto-registered', async () => {
    const servers = await vscode.mcp.getServers();
    assert(servers.some(s => s.name === 'cocopilot'));
  });
  
  // AT-VSIX-2: IDE signals
  test('Active editor signals', async () => {
    await vscode.window.showTextDocument(testDocument);
    await delay(100);
    
    const signals = await fetchIDESignals(); 
    assert(signals.some(s => s.kind === 'ide.active_file.changed'));
  });
});
```

---

## Database Schema Evolution

### Complete Migration Sequence

**0001_schema_migrations.sql**:
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version     INTEGER PRIMARY KEY,
  applied_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

**0002_tasks_v1_compat.sql**:
```sql
CREATE TABLE IF NOT EXISTS tasks (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  instructions    TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'NOT_PICKED',
  output          TEXT,
  parent_task_id  INTEGER,
  created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_created_at
  ON tasks(status, created_at);
CREATE INDEX IF NOT EXISTS idx_tasks_parent  
  ON tasks(parent_task_id);
```

**0003_projects.sql**:
```sql
CREATE TABLE IF NOT EXISTS projects (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  workdir       TEXT NOT NULL,
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  settings_json TEXT
);

INSERT INTO projects (id, name, workdir, settings_json)
SELECT 'proj_default', 'Default', '', NULL
WHERE NOT EXISTS (SELECT 1 FROM projects WHERE id='proj_default');
```

**0004_tasks_add_project_id.sql**:
```sql
ALTER TABLE tasks ADD COLUMN project_id TEXT;
UPDATE tasks SET project_id = 'proj_default' WHERE project_id IS NULL OR project_id = '';
CREATE INDEX IF NOT EXISTS idx_tasks_project_status_created_at
  ON tasks(project_id, status, created_at);
```

**Actual Migrations (0005-0018)**:
- **0005**: tasks_v2_enhancements (priority, type, tags, context_json, result_json)
- **0006**: runs table
- **0007**: leases table
- **0008**: events table
- **0009**: memory table
- **0010**: context_packs table
- **0011**: tasks_project_fk (foreign key constraint)
- **0012**: agents table
- **0013**: task_dependencies table
- **0014**: events_project_id_backfill
- **0015**: events_filter_indexes
- **0016**: tasks_sort_indexes
- **0017**: tasks_updated_at
- **0018**: policies table
- **0019**: automation_emissions (dedupe + throttle for idle planner)

> **Note**: The originally planned `repo_files` table (file system monitoring persistence) was deferred. File context is currently handled through context_packs without persistent file metadata storage.

### Performance Optimizations

**SQLite Settings**:
```sql
PRAGMA journal_mode=WAL;        -- Concurrent reads
PRAGMA synchronous=NORMAL;      -- Balanced durability/performance
PRAGMA foreign_keys=ON;         -- Referential integrity
PRAGMA cache_size=-64000;       -- 64MB cache
```

**Index Strategy**:
```sql
-- Task queries
CREATE INDEX idx_tasks_status_priority_created_at ON tasks(status, priority DESC, created_at);
CREATE INDEX idx_tasks_project_type_status ON tasks(project_id, type, status);

-- Event queries  
CREATE INDEX idx_events_project_kind_created_at ON events(project_id, kind, created_at DESC);
CREATE INDEX idx_events_entity ON events(entity_type, entity_id);

-- Memory queries
CREATE INDEX idx_memory_project_scope_key ON memory_items(project_id, scope, key);
CREATE INDEX idx_memory_updated_at ON memory_items(updated_at DESC);
```

---

## API Specifications

### Error Model (v2)
```json
{
  "error": {
    "code": "INVALID_ARGUMENT|NOT_FOUND|CONFLICT|UNAUTHORIZED|FORBIDDEN|INTERNAL",
    "message": "Human readable description", 
    "details": {
      "field": "validation error details",
      "request_id": "req_12345"
    }
  }
}
```

### SSE Event Format
```typescript
interface CocoPEvent {
  event: 'coco.event';
  data: {
    id: string;           // event_uuid  
    kind: EventKind;      // task.created, run.completed, etc.
    entity_type: string;  // task, run, lease, memory, repo_file
    entity_id: string;    // entity identifier
    created_at: string;   // ISO8601 timestamp
    payload: any;         // event-specific data
  };
}
```

### Task Status Mapping

**v1 Status Values**: `NOT_PICKED`, `IN_PROGRESS`, `COMPLETE`

**v2 Status Values**: 
- `QUEUED` → maps to v1 `NOT_PICKED`
- `RUNNING` → maps to v1 `IN_PROGRESS`  
- `SUCCEEDED` → maps to v1 `COMPLETE`
- `FAILED` → maps to v1 `COMPLETE` (with error in output)
- `BLOCKED` → maps to v1 `IN_PROGRESS`
- `NEEDS_REVIEW` → maps to v1 `IN_PROGRESS`  
- `WAITING_APPROVAL` → maps to v1 `IN_PROGRESS`

**Compatibility Layer**:
```go
func mapV2StatusToV1(v2Status string) string {
    switch v2Status {
    case "QUEUED": return "NOT_PICKED"
    case "RUNNING": return "IN_PROGRESS"
    case "SUCCEEDED": return "COMPLETE" 
    case "FAILED", "BLOCKED", "NEEDS_REVIEW", "WAITING_APPROVAL": 
        return "IN_PROGRESS"
    default: return "NOT_PICKED"
    }
}
```

---

## Testing & Acceptance Criteria

Short reference: [TEST_REGRESSION.md](TEST_REGRESSION.md)

### Regression Test Suite (scripts/poc_regression.sh)
```bash
#!/bin/bash
set -euo pipefail

# Test environment setup
export COCO_DB_PATH=./tmp/test.db  
export COCO_HTTP_ADDR=:8080
rm -f ./tmp/test.db

# Start server in background
./cocopilot &
SERVER_PID=$!
trap "kill $SERVER_PID" EXIT

# Wait for server startup
sleep 2

echo "Running POC regression tests..."

# POC-REG-001: Create → Claim → Save lifecycle
echo "POC-REG-001: Task lifecycle"
TASK_RESP=$(curl -sS -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \
  -d '{"instructions":"POC-REG-001: say hello"}')

TASK_ID=$(echo $TASK_RESP | jq -r .task_id)
test "$TASK_ID" != "null" || { echo "FAIL: Task creation"; exit 1; }

CLAIM_RESP=$(curl -sS http://localhost:8080/task)
echo "$CLAIM_RESP" | grep -q "AVAILABLE TASK ID: $TASK_ID" || { echo "FAIL: Task claiming"; exit 1; }

SAVE_RESP=$(curl -sS -X POST http://localhost:8080/save \
  -H "Content-Type: application/json" \  
  -d "{\"id\":$TASK_ID,\"output\":\"hello\"}")
echo "$SAVE_RESP" | grep -q "saved and marked as COMPLETE" || { echo "FAIL: Task completion"; exit 1; }

# POC-REG-002: Parent task context  
echo "POC-REG-002: Parent context"
PARENT_RESP=$(curl -sS -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \
  -d '{"instructions":"parent: gather info"}')
PARENT_ID=$(echo $PARENT_RESP | jq -r .task_id)

curl -sS http://localhost:8080/task > /dev/null # claim parent
curl -sS -X POST http://localhost:8080/save \
  -H "Content-Type: application/json" \
  -d "{\"id\":$PARENT_ID,\"output\":\"parent output\"}" > /dev/null

CHILD_RESP=$(curl -sS -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \  
  -d "{\"instructions\":\"child task\",\"parent_task_id\":$PARENT_ID}")
CHILD_ID=$(echo $CHILD_RESP | jq -r .task_id)

CHILD_CLAIM=$(curl -sS http://localhost:8080/task)
echo "$CHILD_CLAIM" | grep -q "parent output" || { echo "FAIL: Parent context missing"; exit 1; }

# POC-REG-003: SSE events
echo "POC-REG-003: SSE events" 
timeout 5s curl -sS -N http://localhost:8080/events > /tmp/events.log &
EVENT_PID=$!

curl -sS -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \
  -d '{"instructions":"test sse"}' > /dev/null

sleep 1
kill $EVENT_PID 2>/dev/null || true

grep -q "data:" /tmp/events.log || { echo "FAIL: No SSE events received"; exit 1; }

# POC-REG-004: Workdir management
echo "POC-REG-004: Workdir"
curl -sS -X POST http://localhost:8080/set-workdir \
  -H "Content-Type: application/json" \
  -d '{"workdir":"/tmp/coco-workdir"}' > /dev/null

WORKDIR_RESP=$(curl -sS http://localhost:8080/api/workdir)
echo "$WORKDIR_RESP" | grep -q "/tmp/coco-workdir" || { echo "FAIL: Workdir not persisted"; exit 1; }

echo "All POC regression tests passed!"
```

### Integration Test Examples

**Automation Engine Tests**:
```go
func TestAutomationEngine(t *testing.T) {
    // AT-AUTO-001: Failed run emits triage/fix/verify
    t.Run("FailedRunEmitsTasks", func(t *testing.T) {
        // Create task and run
        taskID := createTestTask(t, "test automation")
        runID := createTestRun(t, taskID, "FAILED")
        
        // Emit run.completed event
        emitEvent(t, "run.completed", runID, map[string]interface{}{
            "status": "FAILED",
            "artifacts": []string{"log_123"},
        })
        
        // Wait for automation processing
        time.Sleep(100 * time.Millisecond)
        
        // Verify TRIAGE, FIX, VERIFY tasks created
        tasks := getTasksByParent(t, taskID)
        assert.Len(t, tasks, 3)
        
        triageTask := findTaskByType(tasks, "ANALYZE") 
        assert.Contains(t, triageTask.Instructions, "summarize failure")
        assert.Contains(t, triageTask.Tags, "auto")
        
        fixTask := findTaskByType(tasks, "MODIFY")
        assert.Equal(t, triageTask.ID, fixTask.Dependencies[0])
        
        verifyTask := findTaskByType(tasks, "TEST")  
        assert.Equal(t, fixTask.ID, verifyTask.Dependencies[0])
        
        // Test idempotency - re-emit same event
        emitEvent(t, "run.completed", runID, map[string]interface{}{
            "status": "FAILED",
        })
        time.Sleep(100 * time.Millisecond)
        
        // Should not create duplicate tasks
        newTasks := getTasksByParent(t, taskID)
        assert.Len(t, newTasks, 3) // Same count
    })
}
```

**UI Component Tests**:
```typescript
// Task Detail Drawer Tests
describe('TaskDetailDrawer', () => {
  test('displays task metadata and parent chain', async () => {
    const task = createMockTask({
      id: 123,
      title: 'Test Task',
      parent_task_id: 456
    });
    
    render(<TaskDetailDrawer task={task} />);
    
    expect(screen.getByText('Test Task')).toBeInTheDocument();
    expect(screen.getByText('#123')).toBeInTheDocument();
    expect(screen.getByText('from #456')).toBeInTheDocument();
  });
  
  test('loads and displays latest run summary', async () => {
    const task = createMockTask({id: 123});
    mockApi.get(`/api/v2/tasks/123`).reply(200, {
      task,
      latest_run: {
        id: 'run_x',
        status: 'SUCCEEDED', 
        started_at: '2024-01-01T00:00:00Z'
      }
    });
    
    render(<TaskDetailDrawer task={task} />);
    
    await waitFor(() => {
      expect(screen.getByText('run_x')).toBeInTheDocument();
      expect(screen.getByText('SUCCEEDED')).toBeInTheDocument();
    });
  });
});
```

### Performance Benchmarks

**Database Performance Targets**:
- Task creation: <10ms (p95)
- Task claiming: <20ms (p95)  
- Event insertion: <5ms (p95)
- SSE event broadcast: <50ms (p95)
- Full task list load: <100ms for 1000 tasks

**API Response Time Targets**:
- `GET /api/v2/tasks`: <200ms for 1000 tasks
- `GET /api/v2/runs/{runId}`: <100ms  
- `POST /api/v2/tasks/{taskId}/claim`: <300ms
- SSE connection establishment: <1s

**Load Testing Scenarios**:
```bash
# Concurrent task creation
echo "Testing concurrent task creation..."
seq 1 100 | xargs -I{} -P 10 curl -sS -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \
  -d '{"instructions":"load test task {}"}'

# SSE connection load  
echo "Testing SSE connection load..."
seq 1 50 | xargs -I{} -P 50 timeout 60s curl -sS -N http://localhost:8080/events &

# Task claiming concurrency
echo "Testing concurrent task claiming..."
seq 1 20 | xargs -I{} -P 20 curl -sS http://localhost:8080/task
```

---

## Implementation Guidelines

### Code Organization
```
cocopilot/
├── main.go                 # Server entry point, HTTP handlers
├── internal/
│   ├── db/                 # Database layer, migrations
│   │   ├── migrations.go
│   │   ├── tasks.go
│   │   ├── projects.go
│   │   └── runs.go
│   ├── api/                # API handlers
│   │   ├── v1/             # Legacy endpoints
│   │   └── v2/             # New endpoints  
│   ├── automation/         # Automation engine
│   │   ├── engine.go
│   │   ├── rules.go
│   │   └── triggers.go
│   ├── events/             # Event system
│   │   ├── emitter.go
│   │   └── sse.go
│   └── ui/                 # Static assets, templates
├── migrations/             # SQL migration files
├── scripts/                # Build, test, deployment scripts  
├── tools/                  # External tools (MCP server, etc.)
├── docs/                   # Documentation
└── tests/                  # Integration tests
```

### Development Workflow

**Phase Implementation Order**:
1. Complete current phase before starting next
2. Each phase must pass regression tests  
3. v2 features behind feature flags initially
4. Gradual rollout with monitoring

**Database Migration Process**:
```bash
# Create new migration
./scripts/new-migration.sh "add_leases_table"
# Creates: migrations/NNNN_add_leases_table.sql

# Test migration  
COCO_DB_PATH=./tmp/test.db go run main.go --dry-run-migrations
  
# Apply migration
COCO_DB_PATH=./prod.db go run main.go
```

**Feature Flag Usage**:
```go
type FeatureFlags struct {
    EnableAutomation   bool `default:"false"`
    EnableV2API       bool `default:"true"`
    EnableLeases      bool `default:"false"`
    EnableMCP         bool `default:"false"`
}

func (h *Handler) handleTaskClaim(w http.ResponseWriter, r *http.Request) {
    if !h.flags.EnableLeases {
        http.Error(w, "Feature not enabled", http.StatusNotImplemented)
        return
    }
    // ... implementation
}
```

### Error Handling Patterns

**Database Errors**:
```go
func createTask(ctx context.Context, task *Task) error {
    if err := validateTask(task); err != nil {
        return &ValidationError{Field: "instructions", Message: err.Error()}
    }
    
    if err := db.Insert(ctx, task); err != nil {
        if isUniqueViolation(err) {
            return &ConflictError{Message: "Task already exists"}
        }
        return &InternalError{Cause: err}
    }
    
    return nil
}
```

**HTTP Error Responses**:
```go
func writeError(w http.ResponseWriter, err error) {
    var apiErr APIError
    switch e := err.(type) {
    case *ValidationError:
        apiErr = APIError{Code: "INVALID_ARGUMENT", Message: e.Error()}
        w.WriteHeader(http.StatusBadRequest)
    case *NotFoundError:
        apiErr = APIError{Code: "NOT_FOUND", Message: e.Error()}
        w.WriteHeader(http.StatusNotFound)
    case *ConflictError:
        apiErr = APIError{Code: "CONFLICT", Message: e.Error()}  
        w.WriteHeader(http.StatusConflict)
    default:
        apiErr = APIError{Code: "INTERNAL", Message: "Internal server error"}
        w.WriteHeader(http.StatusInternalServerError)
    }
    
    json.NewEncoder(w).Encode(map[string]APIError{"error": apiErr})
}
```

### Concurrency Patterns

**SSE Client Management**:
```go
type SSEManager struct {
    clients map[string]chan []byte
    mutex   sync.RWMutex
}

func (s *SSEManager) AddClient(id string) <-chan []byte {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    ch := make(chan []byte, 100) // Buffered to prevent blocking
    s.clients[id] = ch
    return ch
}

func (s *SSEManager) Broadcast(data []byte) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    for _, client := range s.clients {
        select {
        case client <- data:
        default:
            // Client slow/disconnected, don't block
        }
    }
}
```

**Database Connection Pooling**:
```go
func initDB() (*sql.DB, error) {
    db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_timeout=5000")
    if err != nil {
        return nil, err
    }
    
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5) 
    db.SetConnMaxLifetime(time.Hour)
    
    return db, nil
}
```

---

## Guiding Principles

### Backward Compatibility
- **v1 API Stability**: All existing endpoints must remain functional
- **Database Compatibility**: New migrations are additive only
- **UI Compatibility**: Existing Kanban functionality preserved
- **Agent Compatibility**: Current agent workflow continues unchanged

### Traceability & Observability
- **Audit Trail**: Every state change recorded in events table
- **Request Tracing**: Request IDs for correlating logs and errors
- **Performance Monitoring**: Response times, error rates, resource usage
- **Agent Visibility**: Complete view of agent actions and decisions

### Incremental Development
- **Phase Ordering**: Strict dependencies between phases
- **Feature Flags**: Gradual rollout of new capabilities  
- **Vertical Slices**: End-to-end functionality in each iteration
- **Continuous Testing**: Regression suite runs on every change

### Determinism & Reproducibility
- **Event Ordering**: Consistent event processing order
- **Automation Rules**: Deterministic task creation logic
- **State Management**: Immutable context packs and audit logs
- **Testing**: Reproducible test scenarios and data

### Security & Governance
- **Policy Enforcement**: Configurable rules for tool execution and resource access
- **Approval Gates**: Human review requirements for sensitive operations  
- **Access Control**: Project-scoped permissions and agent authentication
- **Data Privacy**: Secure handling of code, credentials, and sensitive context

---

This roadmap provides the complete atomic reference for implementing Cocopilot's evolution from a simple task queue to a comprehensive agentic system. Each phase builds incrementally on previous work while maintaining the stability and functionality of the existing proof-of-concept.
