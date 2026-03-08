# Next Steps

**Last Updated**: 2026-02-12
**References**: [COMPLETION_SUMMARY.md](../../COMPLETION_SUMMARY.md)

## Recently Completed

### TASK-716: db_v2_test.go parse error (duplicate)
- Status: Complete (2026-02-12)
- Notes: Fix already present in db_v2_test.go; full suite passes (`go test ./...`).

### TASK-752: Env isolation test stabilization
- Status: Complete (2026-02-12)
- Notes: Env isolation fix applied; full suite passes (`go test -v ./...`).

### NEXT-000: VSIX Scaffold
- Status: Complete
- Notes: VS Code extension scaffold is in place under `tools/cocopilot-vsix/`.

### NEXT-001: POC Regression Suite
- Status: Complete
- Notes: Regression tests are implemented in Go and running in CI-style local flow (`go test ./...`).

### NEXT-002: Migration System
- Status: Complete
- Notes: Boot-time migration runner active; migrations `0001`-`0017` applied and tracked.

### NEXT-003: v2 Health/Version Endpoints
- Status: Complete (2026-02-11)
- Notes: `/api/v2/health` and `/api/v2/version` implemented with contract tests and schema-version verification.

### NEXT-004: Projects Table and API
- Status: Complete
- Notes: CRUD-style project handlers and default project support are in place.

### NEXT-005: Task Project Association
- Status: Complete (core backend)
- Notes: `tasks.project_id` exists and is used by task creation.

### NEXT-006: Basic Runs Table
- Status: Complete (core backend)
- Notes: Claims create runs, save completes runs, and run retrieval endpoint exists.

### NEXT-007: Agent Registration
- Status: Complete (API/backend)
- Notes: Agent registration, listing, heartbeat, and stale-agent background handling are implemented.

### NEXT-008: Lease-Based Task Claiming
- Status: Complete (2026-02-11)
- Delivered:
  - Lease migration (`0007_leases.sql`) active
  - `GET /task` now lease-backed for both fresh and reclaimed tasks
  - Conflict-safe concurrent claiming (single winner)
  - Lease expiration cleanup job requeues abandoned tasks
  - Lease APIs: `POST /api/v2/leases`, `POST /api/v2/leases/{id}/heartbeat`
  - Completion flow releases leases
  - Added concurrent claiming test coverage

### NEXT-009: Lease Release API + Lease Events
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/leases/{id}/release`
  - Added lifecycle event emission on lease create/expire/release
  - Updated lease cleanup to emit `lease.expired` after requeue/deletion
  - Added API + DB tests for release behavior and event persistence

### NEXT-010: v2 Health + Version Endpoints
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /api/v2/health` stable JSON response
  - `GET /api/v2/version` with schema migration version
  - Method-not-allowed behavior returns JSON error envelope
  - Endpoint tests updated for contract and schema parity

### NEXT-011: Runtime Configuration via Environment
- Status: Complete (2026-02-11)
- Delivered:
  - Added `COCO_DB_PATH` runtime override
  - Added `COCO_HTTP_ADDR` runtime override with validation
  - Server and CLI DB init now use runtime config loader
  - Added unit tests for defaults, overrides, and invalid env values
  - Updated README and state docs

### NEXT-012: Standardize v2 Error Envelope
- Status: Complete (2026-02-11)
- Delivered:
  - Added shared v2 JSON error envelope writer: `{"error":{"code","message","details?"}}`
  - Applied consistent 4xx/5xx error responses across v2 handlers:
    - health/version
    - projects (+ project tasks)
    - agents (+ heartbeat action)
    - runs lookup
    - leases (+ heartbeat/release/action router)
  - Replaced v2 plain-text `http.Error` responses with JSON envelope responses
  - Added representative schema tests for projects, leases, agents, and health/version method failures
  - Updated API docs with canonical error examples
  - Verified success-path compatibility and full test pass (`go test ./...`)

### NEXT-013: OpenAPI Parity for Implemented v2 Endpoints
- Status: Complete (2026-02-11)
- Delivered:
  - Updated `docs/api/openapi-v2.yaml` to match shipped v2 handlers:
    - added `/api/v2/agents`, `/api/v2/agents/{agentId}/heartbeat`, and `/api/v2/leases`
    - aligned `/api/v2/version` response fields with runtime (`version` included)
    - added `PATCH /api/v2/projects/{projectId}`
    - aligned lease heartbeat/release response shapes and status codes
    - aligned run get response to envelope form (`{ "run": ... }`)
  - Marked design-only endpoints with `x-runtime-status: planned` for clear implemented vs planned separation
  - Added canonical error examples for representative invalid-argument/conflict/not-found cases
  - Validated OpenAPI YAML syntax and re-ran test suite (`go test ./...`)

### NEXT-014: Contract-Level API Response Tests
- Status: Complete (2026-02-11)
- Delivered:
  - Extracted reusable route wiring into `registerRoutes(mux)` with named v2 route dispatch handlers
  - Updated server startup to use explicit mux registration (no behavior change intended)
  - Added route-level v2 contract tests covering:
    - method-not-allowed envelope behavior across health/version/projects/agents/leases/runs routes
    - project not-found envelope through route dispatch
    - health success contract through route dispatch
  - Verified full suite passes (`go test ./...`)

### NEXT-015: Authn/Authz Foundation (Read-Only Guardrails First)
- Status: Complete (2026-02-11)
- Delivered:
  - Added runtime auth config flags:
    - `COCO_REQUIRE_API_KEY` (mutating v2 endpoints)
    - `COCO_REQUIRE_API_KEY_READS` (optional v2 read protection)
    - `COCO_API_KEY` (legacy wildcard identity for backward compatibility)
  - Added v2 API-key middleware in route registration with constant-time key comparison
  - Unauthorized requests now return standard v2 error envelope (`UNAUTHORIZED`)
  - Added tests for:
    - config defaults/overrides/validation for new auth flags
    - routed unauthorized/authorized behavior on mutating and read endpoints
  - Updated README runtime configuration docs
  - Verified suite remains green (`go test ./...`)

### NEXT-016: Auth Scope and Identity Hardening
- Status: Complete (2026-02-11)
- Delivered:
  - Added scoped identity model (`agent|user|service`) via `COCO_API_IDENTITIES`
  - Added endpoint-level scope enforcement with required-scope mapping by path/method
  - Preserved `UNAUTHORIZED` for missing/invalid key and added `FORBIDDEN` for insufficient scope
  - Added tests for scoped auth matrix and forbidden/allowed behavior through route dispatch
  - Updated API docs/OpenAPI to document scoped auth format and forbidden examples
  - Verified suite remains green (`go test ./...`)

### NEXT-017: Auth Audit and Policy Observability
- Status: Complete (2026-02-11)
- Delivered:
  - Added structured auth decision logs with identity, scope, result, and endpoint context
  - Persisted auth denial events to the events stream for operational visibility
  - Added regression tests for `UNAUTHORIZED` and `FORBIDDEN` audit emission
  - Documented key rotation and scope rollout playbook

### NEXT-018: v2 Task Detail Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/tasks/{taskId}` to return task details
  - Included parent chain and latest run in the response payload
  - Standard v2 error envelope for not-found and method-not-allowed responses
  - Added route-level tests for success, not-found, and method handling

### NEXT-019: v2 Tasks List Endpoint + Filters
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/tasks` list endpoint ordered by `created_at` ascending
  - Added optional query filters for `project_id` and `status`
  - Invalid filter values return standard v2 error envelope

### NEXT-020: Policy Engine Foundation
- Status: Complete (2026-02-11)
- Delivered:
  - Added policies table migration with project scope, rules JSON, and enabled flag
  - Added policy model helpers for create/list by project
  - Added migration and DB helper tests for policy storage
  - Documented schema addition in API and README docs

### NEXT-072: v2 Task Complete next_tasks Payload
- Status: Complete (2026-02-11)
- Delivered:
  - `POST /api/v2/tasks/{taskId}/complete` includes `next_tasks` when available
  - Response contract updated to document the optional `next_tasks` array

### NEXT-073: v2 Child Task Creation
- Status: Complete (2026-02-11)
- Delivered:
  - `POST /api/v2/tasks` accepts `parent_task_id` to create child tasks
  - Child tasks inherit parent context for downstream agent execution

### NEXT-025: v2 Tasks List Pagination + Total
- Status: Complete (2026-02-11)
- Delivered:
  - Added `limit` and `offset` query params to `GET /api/v2/tasks`
  - Responses include `total` count alongside `tasks`

### NEXT-026: v2 Project Tasks List Endpoint + Paging
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/projects/{projectId}/tasks` to list tasks scoped to a project
  - Supports `status` filter plus `limit`/`offset` pagination
  - Responses include `total` count alongside `tasks`
  - Added tests for success, paging, not-found, and method-not-allowed behavior

### NEXT-077: v2 Project Task Create Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/projects/{projectId}/tasks` to create tasks scoped to a project
  - Accepts the same task payload as `POST /api/v2/tasks` with `project_id` implied by the path
  - Standard v2 error envelope for invalid payloads or unknown projects

### NEXT-027: v2 Task Dependencies Endpoints
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/tasks/{taskId}/dependencies` to create a dependency via `depends_on_task_id`
  - Added `GET /api/v2/tasks/{taskId}/dependencies` to list dependencies for a task
  - Standard v2 error envelope for invalid IDs, not-found tasks, duplicate dependencies, and method handling
  - Added tests for create/list, validation, conflict, and method-not-allowed behavior

### NEXT-028: Migration Directory + Task Dependencies Migration
- Status: Complete (2026-02-11)
- Delivered:
  - Migration runner now uses `migrations/` as the source of SQL files
  - `0013_task_dependencies.sql` applied in boot-time migrations

### NEXT-029: Migration Directory Alignment Docs
- Status: Complete (2026-02-11)
- Delivered:
  - Updated roadmap snapshot/current state notes to reflect `migrations/` alignment
  - Confirmed migration directory references in state docs match runtime

### NEXT-030: v2 Task Dependency Removal Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `DELETE /api/v2/tasks/{taskId}/dependencies/{dependsOnTaskId}`
  - Standard v2 error envelope for not-found and method-not-allowed responses
  - Added delete coverage in v2 task dependencies tests

### NEXT-031: Dependency Cycle Detection
- Status: Complete (2026-02-11)
- Delivered:
  - Dependency creation now rejects cycles in the task graph
  - Cycle attempts return 409 with the standard v2 error envelope
  - Added cycle conflict coverage in v2 task dependencies tests

### NEXT-032: Dependency Event Emission
- Status: Complete (2026-02-11)
- Delivered:
  - Emit dependency lifecycle events on create/delete
  - Persist `task.dependency.created` and `task.dependency.deleted` events
  - Added tests covering dependency event emission

### NEXT-033: v2 Events List Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/events` to list events in reverse-chronological order
  - Supports `type`, `since`, and `limit` query filters
  - Invalid params and method mismatches return standard v2 error envelopes
  - Added tests for success, filtering, validation, and method handling

### NEXT-034: v2 Events List Pagination + Total
- Status: Complete (2026-02-11)
- Delivered:
  - Added `limit` and `offset` query params to `GET /api/v2/events`
  - Responses include `total` count alongside `events`

### NEXT-035: v2 Events List task_id Filter
- Status: Complete (2026-02-11)
- Delivered:
  - Added `task_id` query filter to `GET /api/v2/events`
  - Updated validation to allow filtering by task id
  - Added tests for task-filtered event list responses

### NEXT-036: v2 Events List project_id Filter + Backfill
- Status: Complete (2026-02-11)
- Delivered:
  - Added `project_id` query filter to `GET /api/v2/events`
  - Backfilled existing events with `project_id` for filtering parity
  - Added tests for project-filtered event list responses

### NEXT-062: v1 /save updated_at Response
- Status: Complete (2026-02-11)
- Delivered:
  - `POST /save` now returns `updated_at` in the v1 response payload
  - Documentation updated to reflect v1 `/save` updated_at parity

### NEXT-063: v1 /update-status updated_at Response
- Status: Complete (2026-02-11)
- Delivered:
  - `POST /update-status` now returns `updated_at` in the v1 response payload
  - Documentation updated to reflect v1 `/update-status` updated_at parity

### NEXT-064: v1 /api/tasks Filters
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /api/tasks` supports `status` filtering (`NOT_PICKED`, `IN_PROGRESS`, `COMPLETE`)
  - `GET /api/tasks` supports `updated_since` filtering (RFC3339)
  - Invalid or empty filter values return 400 responses

### NEXT-065: v1 /api/tasks Pagination + Total
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /api/tasks` supports `limit` and `offset` pagination
  - Responses include a `total` count alongside `tasks`

### NEXT-066: v1 /api/tasks Sorting
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /api/tasks` supports `sort` values `created_at:asc`, `created_at:desc`, `updated_at`
  - `updated_at` sort returns most-recent-first ordering
  - Invalid `sort` values return 400 responses

### NEXT-067: v1 /api/tasks project_id Filter
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /api/tasks` supports `project_id` filtering
  - Invalid or empty `project_id` values return 400 responses
  - Documentation updated to reflect the new filter

### NEXT-068: v1 /events project_id Filter
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /events` supports `project_id` filtering
  - Invalid or empty `project_id` values return 400 responses
  - Documentation updated to reflect the new filter

### NEXT-069: v1 /events type Filter
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /events` supports `type=tasks` filtering
  - Documentation updated to reflect the new filter

### NEXT-070: v1 /events since Filter
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /events` supports `since` RFC3339 replay filtering
  - Invalid or empty `since` values return 400 responses
  - Documentation updated to reflect the new filter

### NEXT-071: v1 /events replay limit
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /events` supports optional `limit` to cap replay size
  - Replay limit is capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`
  - Documentation updated to note the replay limit behavior

### NEXT-037: Migration 0014 Docs/Test Updates
- Status: Complete (2026-02-11)
- Delivered:
  - Documented migration `0014` event `project_id` backfill coverage
  - Noted doc/test updates for migration `0014` alignment

### NEXT-038: v2 Events SSE Stream
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/events/stream` SSE endpoint scoped by `project_id`
  - Supports `type` filter for event kinds
  - SSE payload emits `event:` and `data:` lines with event JSON
  - Added tests for headers, filtering, and payload format

### NEXT-039: v2 Events SSE Auth Coverage
- Status: Complete (2026-02-11)
- Delivered:
  - v2 auth guardrails now cover `GET /api/v2/events/stream` when enabled
  - Scoped auth enforcement applied to SSE stream reads (`v2:read`)
  - Added tests for authorized, unauthorized, and forbidden SSE access

### NEXT-040: v2 Events SSE since Filter + Replay
- Status: Complete (2026-02-11)
- Delivered:
  - Added `since` query filter support for `GET /api/v2/events/stream`
  - Server replays matching events since the provided cursor before live streaming

### NEXT-041: v2 Events SSE Replay Limit
- Status: Complete (2026-02-11)
- Delivered:
  - Added replay `limit` support for `GET /api/v2/events/stream` to cap backlog
  - Documented the replay limit behavior in state and roadmap notes

### NEXT-042: SSE Heartbeat Config
- Status: Complete (2026-02-11)
- Delivered:
  - Added `COCO_SSE_HEARTBEAT_SECONDS` to configure SSE stream heartbeat interval
  - Documented the SSE heartbeat configuration in state and roadmap notes

### NEXT-043: SSE Replay Limit Cap Config
- Status: Complete (2026-02-11)
- Delivered:
  - Added `COCO_SSE_REPLAY_LIMIT_MAX` to cap requested SSE replay `limit`
  - Documented the replay limit cap in state and roadmap notes

### NEXT-044: Events Filter Indexes
- Status: Complete (2026-02-11)
- Delivered:
  - Added migration `0015_events_filter_indexes.sql` to index event filters
  - Added indexes for `project_id`, `entity_type/entity_id + created_at`, and `project_id/kind + created_at`

### NEXT-045: Events Retention Cleanup
- Status: Complete (2026-02-11)
- Delivered:
  - Added hourly background pruning when retention config is enabled
  - Added retention controls via `COCO_EVENTS_RETENTION_DAYS` and `COCO_EVENTS_RETENTION_MAX_ROWS`
  - Pruning deletes events by age and caps max rows

### NEXT-046: Events Prune Interval Config
- Status: Complete (2026-02-11)
- Delivered:
  - Added `COCO_EVENTS_PRUNE_INTERVAL_SECONDS` to configure the prune interval
  - Default remains hourly when retention pruning is enabled
  - Documented the prune interval configuration in roadmap and state notes

### NEXT-047: Events Prune Logging
- Status: Complete (2026-02-11)
- Delivered:
  - Log prune completion with deleted count and duration
  - Log skipped prune runs when SQLite is busy
  - Log prune failures with duration context

### NEXT-048: Version Endpoint Retention Snapshot
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /api/v2/version` now includes a `retention` config snapshot
  - Documented retention fields: `enabled`, `interval_seconds`, `max_rows`, `days`

### NEXT-049: v2 Config Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/config` to return a redacted runtime config snapshot
  - Snapshot includes auth, retention, and SSE config fields (db path redacted)
  - Endpoint requires `v2:read` scope when v2 auth read guardrails are enabled

### NEXT-050: v2 Agent Detail Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/agents/{agentId}` to return agent details
  - Standard v2 error envelope for not-found and method-not-allowed responses
  - Added tests for detail success and error paths

### NEXT-051: v2 Agent Delete Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `DELETE /api/v2/agents/{agentId}` to remove agents
  - Standard v2 error envelope for not-found and method-not-allowed responses
  - Added tests for delete success, not-found, and method handling

### NEXT-052: v2 Agents List Filters
- Status: Complete (2026-02-11)
- Delivered:
  - Added `status` filter (`active` or `stale`) to `GET /api/v2/agents`
  - Added `since` RFC3339 filter for `last_seen` or `registered_at` on `GET /api/v2/agents`
  - Added filter validation coverage for invalid or empty query values

### NEXT-053: v2 Agents List Pagination + Total
- Status: Complete (2026-02-11)
- Delivered:
  - Added `limit` and `offset` query params to `GET /api/v2/agents`
  - Responses include `total` count alongside `agents`
  - Added tests for paging and total count behavior

### NEXT-054: v2 Agents List Sorting
- Status: Complete (2026-02-11)
- Delivered:
  - Added `sort` query for `GET /api/v2/agents` (`created_at` default, `last_seen:asc`, `last_seen:desc`)
  - `last_seen` sorting falls back to `registered_at` when `last_seen` is null
  - Added tests for sort order and invalid `sort` values

### NEXT-055: v2 Tasks List Sorting
- Status: Complete (2026-02-11)
- Delivered:
  - Added `sort` query for `GET /api/v2/tasks` (`created_at:asc` default, `created_at:desc`, `updated_at:asc`, `updated_at:desc`)
  - Added `sort` query for `GET /api/v2/projects/{projectId}/tasks` with the same options
  - Added tests for sort ordering and invalid `sort` values

### NEXT-056: Task Sort Indexes
- Status: Complete (2026-02-11)
- Delivered:
  - Added migration `0016_tasks_sort_indexes.sql` to index task sort keys
  - Boot-time migrations apply task sort indexes used by task list sorting

### NEXT-057: Task Updated-At Tracking
- Status: Complete (2026-02-11)
- Delivered:
  - Added migration `0017_tasks_updated_at.sql` to ensure `tasks.updated_at` exists
  - Backfilled `tasks.updated_at` to `created_at` for existing rows
  - Task mutations now maintain `tasks.updated_at` on claim, status changes, and completion

### NEXT-058: Task Updated-At in Responses
- Status: Complete (2026-02-11)
- Delivered:
  - Added `updated_at` to v2 task response payloads
  - Documented `updated_at` response availability in roadmap/state notes

### NEXT-059: v1 Tasks List Updated-At
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /api/tasks` now includes `updated_at` for each task
  - v1 list payload aligns with updated-at tracking

### NEXT-060: v1 Task Claim Updated-At
- Status: Complete (2026-02-11)
- Delivered:
  - `GET /task` now includes `updated_at` in the task response
  - v1 claim payload aligns with updated-at tracking

### NEXT-061: v1 Task Create Updated-At
- Status: Complete (2026-02-11)
- Delivered:
  - `POST /create` now returns `updated_at` in the response payload
  - v1 create response aligns with updated-at tracking

### NEXT-020: v2 Task Claim Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/tasks/{taskId}/claim` for explicit v2 task claiming
  - Returns lease + run details on successful claim
  - Conflict responses use the standard v2 error envelope

### NEXT-021: v2 Task Complete Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/tasks/{taskId}/complete` for v2 task completion
  - Marks tasks as COMPLETE, releases active leases, and completes the latest run
  - Standard v2 error envelope for invalid, not-found, and conflict responses
  - Added tests for success, not-found, and method handling

### NEXT-022: v2 Task Create Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/tasks` for v2 task creation
  - Supports optional `project_id` and `parent_task_id` inputs
  - Standard v2 error envelope for validation failures
  - Added tests for success and invalid payload handling

### NEXT-023: v2 Task Update Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `PATCH /api/v2/tasks/{taskId}` for v2 task updates
  - Standard v2 error envelope for invalid or not-found updates
  - Added tests for success, validation failures, and method handling

### NEXT-024: v2 Task Delete Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `DELETE /api/v2/tasks/{taskId}` for v2 task deletion
  - Standard v2 error envelope for not-found and method handling
  - Added tests for success, not-found, and method handling

### NEXT-072: v2 Runs Sub-Resource Endpoints
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/runs/{runId}/steps`, `POST /api/v2/runs/{runId}/logs`, and `POST /api/v2/runs/{runId}/artifacts`
  - Run detail now returns steps, logs, artifacts, and tool invocations
  - Added validation and error-envelope coverage for run sub-resources

### NEXT-073: v2 Project Memory Endpoints
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/projects/{projectId}/memory` and `PUT /api/v2/projects/{projectId}/memory`
  - Supports `scope`, `key`, and `q` filters on memory queries
  - Added tests for success, validation, and not-found cases

### NEXT-074: v2 Context Packs Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `POST /api/v2/projects/{projectId}/context-packs`
  - Validates project/task ownership and persists context packs
  - Added tests for success, validation, and not-found cases

### NEXT-075: v2 Project Tree Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/projects/{projectId}/tree`
  - Returns a shallow workdir snapshot with validation
  - Added tests for success, validation, and method-not-allowed cases

### NEXT-076: v2 Project Changes Endpoint
- Status: Complete (2026-02-11)
- Delivered:
  - Added `GET /api/v2/projects/{projectId}/changes`
  - Returns git status-based change feed with optional `since`

### NEXT-078: MCP Server Scaffold
- Status: Complete (2026-02-12)
- Delivered:
  - Added MCP server scaffold under `tools/cocopilot-mcp`
  - Implemented initial tool set
  - Added MCP `tools.json` manifest

### NEXT-079: VSIX MCP Config/Start Commands
- Status: In Progress (2026-02-12)
- Notes: Tracking VSIX integration for MCP config/start commands; wiring and UX polish remain.
- Follow-up: automate `tools.json` manifest generation.

## Active Priorities
- Define the automation engine workflow (task graph planning, orchestration, retries, rollback policy).
- Extend repository change tracking to include commit diffs and richer timestamps when needed.
- Add end-to-end tests that tie automation runs to repo diffs (create tasks -> apply changes -> verify diff output).
- Add MCP/VSIX packaging and publishing tasks (signed builds, release pipeline, distribution docs).

## Current Gaps and Drift Notes
- Completion reality is closer to ~60%: core backend + v2 APIs are strong, but UI, governance, packaging, and ops are materially behind.
- UI expansion remains incomplete: project selector, run viewer, memory panel, dependency visualization, agent dashboard.
- Governance and automation are foundational only: `COCO_AUTOMATION_RULES` baseline exists, but rule management, loop guards, quotas, policy enforcement, and operator tooling are missing.
- Packaging and release are not production-ready: VSIX + MCP scaffolds exist, but signed builds, distribution, telemetry, and deployment playbooks are absent.
- Operational tooling gaps persist: metrics/dashboards, backup/restore automation, retention reports, and health runbooks.
- Drift callout: older docs and snapshot notes implied ~75% completion; align narrative to the current assessment and missing areas above.

## Ongoing Risks
- SQLite write contention under heavy parallel claiming
- Auth model uses API keys and scoped identities; no user session or mTLS support
