# Current State

**Last Updated**: 2026-02-12
**References**: [COMPLETION_SUMMARY.md](../../COMPLETION_SUMMARY.md)

## What Is True Right Now

### Working Implementation
- Go HTTP server on port `8080` with Kanban UI and SSE updates
- Latest `go test ./...` pass recorded on 2026-02-12
- SQLite persistence with boot-time migrations (`0001`-`0018`) from `migrations/`
- Plan completion estimate: ~60% (core backend is strong; UI expansion, governance tooling, and packaging remain incomplete).
- MCP server scaffold present under `tools/cocopilot-mcp/`
- VSIX extension scaffold present under `tools/cocopilot-vsix/`
- MCP and VSIX release checklists are documented in their respective README files
- MCP tool coverage includes tasks (create/list/update/complete/claim/save/dependencies), projects (CRUD/tasks/tree/changes/audit/events replay), events list, runs, leases, agents, config/version/health, memory, policies, and context packs
- VSIX commands include MCP configure (open `mcp.json`), start/stop MCP server, and OpenAPI doc shortcuts (Open API Docs, Open API Summary, Open OpenAPI Spec)
- Documentation references now align to the `migrations/` directory, including `0014` docs/test notes
- Events filter indexes added via `0015_events_filter_indexes.sql`
- Task sort indexes added via `0016_tasks_sort_indexes.sql`
- Task updated-at tracking added via `0017_tasks_updated_at.sql`
- Policies table added via `0018_policies.sql`
- UI placeholders now fetch data for agents, audit, memory, runs, graphs, and context packs
- v1 task lifecycle API (`/create`, `/task`, `/save`, `/update-status`, `/delete`)
- v1 `GET /task` includes `updated_at` in the response payload
- v1 `POST /create` returns `updated_at` in the response payload
- v1 `POST /save` returns `updated_at` in the response payload
- v1 `POST /update-status` returns `updated_at` in the response payload
- v1 task list (`GET /api/tasks`) includes `updated_at` timestamps
- v1 task list (`GET /api/tasks`) supports `status`, `updated_since`, and `project_id` filters
- v1 task list (`GET /api/tasks`) supports `sort` (`created_at:asc`, `created_at:desc`, `updated_at` desc)
- v1 task list (`GET /api/tasks`) supports `limit`/`offset` pagination with `total` count
- v1 events stream (`GET /events`) supports `project_id`, `type=tasks`, and `since` replay with optional `limit` (capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`)
- v2 project APIs (`/api/v2/projects*`)
- Run ledger support (`runs`, `run_steps`, `run_logs`, `artifacts`, `tool_invocations`)
- v2 runs API supports run detail plus step/log/artifact ingestion
- Project memory endpoints support scoped, keyed, and free-text queries
- Context pack creation endpoint persists per-task context bundles
- Project tree endpoint returns a shallow workdir snapshot
- Project changes endpoint reports git status-based working tree changes
- Agent registration + heartbeat APIs (`/api/v2/agents*`)
- Agent list filters on `GET /api/v2/agents` for `status` (`active`/`stale`), `since`, `limit`, and `offset` with `total` count
- Agent list sorting on `GET /api/v2/agents` via `sort` (`created_at`, `last_seen:asc`, `last_seen:desc`)
- Lease-based claiming:
  - `GET /task` now always acquires a lease before returning a task
  - Safe conflict handling for concurrent claimers
  - Reclaim of abandoned `IN_PROGRESS` tasks with a fresh lease
  - `POST /api/v2/leases`, `POST /api/v2/leases/{id}/heartbeat`, `POST /api/v2/leases/{id}/release`
  - Background expired-lease cleanup and task requeue
  - Lease lifecycle events persisted: `lease.created`, `lease.expired`, `lease.released`
- Task completion (`POST /save`) releases active lease and completes latest running run
- Task updates maintain `tasks.updated_at` on claim, status changes, and completion
- v2 task responses include `updated_at` timestamps
- v2 task completion includes `next_tasks` in the response when available
- v2 task creation supports child tasks via `parent_task_id`
- Automation engine baseline is available via `COCO_AUTOMATION_RULES` for `task.completed` follow-up tasks
- Automation API endpoints include rules, simulate, and replay
- Project-scoped task creation endpoint implemented (`POST /api/v2/projects/:id/tasks`)
- Dependency creation rejects cycles for v2 task dependencies
- Task dependency events persisted on create/delete: `task.dependency.created`, `task.dependency.deleted`
- Policy lifecycle events persisted: `policy.created`, `policy.updated`, `policy.deleted`
- Route registration is centralized via `registerRoutes(mux)` for consistent runtime/test dispatch
- Optional v2 API-key guardrails:
  - `COCO_REQUIRE_API_KEY=true` enforces `X-API-Key` on mutating v2 endpoints
  - `COCO_REQUIRE_API_KEY_READS=true` extends enforcement to v2 reads
  - `COCO_API_IDENTITIES` supports scoped identities (`id|type|api_key|scope1,scope2;...`)
  - `COCO_API_KEY` is accepted as a legacy wildcard identity for backward compatibility
  - endpoint-level scopes enforced (for example: `projects:write`, `leases:write`, `v2:read`)
  - unauthorized responses use the standard v2 error envelope (`UNAUTHORIZED`)
  - insufficient scope responses return `FORBIDDEN` with `required_scope` details
  - v2 events SSE stream is covered by v2 read auth when enabled
- v2 events SSE stream supports `since` replay with optional `limit` for catch-up on connect
- v2 SSE replay limit cap is configurable via `COCO_SSE_REPLAY_LIMIT_MAX` (v1 uses `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`)
- Events retention pruning runs every hour by default when `COCO_EVENTS_RETENTION_DAYS` or `COCO_EVENTS_RETENTION_MAX_ROWS` is enabled (prune interval configurable via `COCO_EVENTS_PRUNE_INTERVAL_SECONDS`) and logs prune outcomes with deleted counts and durations
- `GET /api/v2/version` returns a retention config snapshot (`retention.enabled`, `interval_seconds`, `max_rows`, `days`)
- `GET /api/v2/config` returns a redacted runtime config snapshot (auth, retention, SSE)
- Auth decision logs for v2 requests with denial events persisted to the events stream
- Policy engine foundation with persisted policy definitions
- OpenAPI docs are aligned for implemented v2 endpoints; planned endpoints are marked with `x-runtime-status: planned`

### Database Schema (Implemented Core)
```sql
CREATE TABLE tasks (..., project_id TEXT, status TEXT, ...);
CREATE TABLE projects (...);
CREATE TABLE runs (...);
CREATE TABLE run_steps (...);
CREATE TABLE run_logs (...);
CREATE TABLE artifacts (...);
CREATE TABLE tool_invocations (...);
CREATE TABLE leases (..., UNIQUE(task_id), expires_at TEXT, ...);
CREATE TABLE agents (...);
CREATE TABLE events (...);
CREATE TABLE memory_items (...);
CREATE TABLE context_packs (...);
CREATE TABLE policies (...);
CREATE TABLE schema_migrations (...);
```

### API Endpoints (Current)

**v1 (stable behavior preserved)**:
- `GET /task` (includes `updated_at` in response payload)
- `POST /create` (returns `updated_at` in response payload)
- `POST /save` (returns `updated_at` in response payload)
- `POST /update-status` (returns `updated_at` in response payload)
- `POST /delete`
- `GET /api/tasks` (includes `updated_at`; supports `status`, `updated_since`, `project_id`, `sort`, `limit`, and `offset` with `total` count)
- `GET /events` (supports `project_id`, `type=tasks`, and `since` RFC3339 replay with optional `limit`)
- `GET /api/workdir`
- `POST /set-workdir`
- `GET /instructions`

**v2 currently implemented**:
- `GET /api/v2/health`
- `GET /api/v2/version`
- `GET /api/v2/config`
- `POST /api/v2/projects`
- `GET /api/v2/projects`
- `GET /api/v2/projects/:id`
- `PATCH /api/v2/projects/:id`
- `DELETE /api/v2/projects/:id`
- `GET /api/v2/projects/:id/tasks` (filters + `limit`/`offset` pagination with `total` count, `sort` with `created_at:asc|desc` and `updated_at:asc|desc`)
- `POST /api/v2/projects/:id/tasks`
- `GET /api/v2/projects/:id/policies`
- `POST /api/v2/projects/:id/policies`
- `GET /api/v2/projects/:id/policies/:policyId`
- `PATCH /api/v2/projects/:id/policies/:policyId`
- `DELETE /api/v2/projects/:id/policies/:policyId`
- `POST /api/v2/agents`
- `GET /api/v2/agents` (filters: `status` `active`/`stale`, `since` RFC3339, `limit`, `offset` with `total` count)
- `GET /api/v2/agents/:id`
- `DELETE /api/v2/agents/:id`
- `POST /api/v2/agents/:id/heartbeat`
- `POST /api/v2/leases`
- `POST /api/v2/leases/:id/heartbeat`
- `POST /api/v2/leases/:id/release`
- `GET /api/v2/runs/:id`
- `POST /api/v2/runs/:id/steps`
- `POST /api/v2/runs/:id/logs`
- `POST /api/v2/runs/:id/artifacts`
- `GET /api/v2/events` (supports `type`, `since`, `task_id`, `project_id`, `limit`, `offset` with `total` count)
- `GET /api/v2/events/stream` (SSE stream scoped by `project_id`, optional `type`, `since`, and replay `limit`)
- `POST /api/v2/tasks`
- `GET /api/v2/tasks` (filters + `limit`/`offset` pagination with `total` count, `sort` with `created_at:asc|desc` and `updated_at:asc|desc`)
- `GET /api/v2/tasks/:id`
- `GET /api/v2/projects/:id/memory` (filters: `scope`, `key`, `q`)
- `PUT /api/v2/projects/:id/memory`
- `POST /api/v2/projects/:id/context-packs`
- `GET /api/v2/projects/:id/tree`
- `GET /api/v2/projects/:id/changes` (git status-based change feed with optional `since`)
- `POST /api/v2/tasks/:id/dependencies`
- `GET /api/v2/tasks/:id/dependencies`
- `DELETE /api/v2/tasks/:id/dependencies/:dependsOnTaskId`
- `PATCH /api/v2/tasks/:id`
- `POST /api/v2/tasks/:id/claim`
- `POST /api/v2/tasks/:id/complete`
- `DELETE /api/v2/tasks/:id`
- `GET /api/v2/projects/:id/automation/rules`
- `POST /api/v2/projects/:id/automation/simulate`

**v2 documentation status**:
- `docs/api/openapi-v2.yaml` now matches runtime for currently implemented v2 endpoints
- Future/design-only endpoints are explicitly marked `x-runtime-status: planned`

### Task States
- `NOT_PICKED`
- `IN_PROGRESS`
- `COMPLETE`

## What Works
- End-to-end create -> claim -> save flow
- Parent context propagation for child tasks
- Concurrent claiming protection via exclusive leases
- Recovery of abandoned work through lease expiration cleanup
- SSE update propagation to web clients
- Migration CLI (`migrate up|down|status`)
- Consistent v2 JSON error envelope for all v2 4xx/5xx responses:
  - shape: `{"error":{"code":"...","message":"...","details":{...}}}`
  - method failures return `METHOD_NOT_ALLOWED`

## What Is Still Missing
- Automated deployment packaging and release automation for VSIX + MCP server
- UI expansion beyond the basic Kanban board (project selector, runs, memory, dependencies, agent dashboard)
- Automation engine governance (rule management, loop detection, quotas, and operator visibility)
- Policy enforcement beyond storage (runtime evaluation and enforcement points)
- Operational tooling (metrics, dashboards, automated backup/restore)

## Known Issues / Risks
- SQLite still requires careful contention handling under high parallelism
- Auth controls are API-key based; no user session or mTLS support
- Runtime config now supports env overrides (`COCO_DB_PATH`, `COCO_HTTP_ADDR`, `COCO_SSE_HEARTBEAT_SECONDS`, `COCO_SSE_REPLAY_LIMIT_MAX`, `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`, `COCO_EVENTS_RETENTION_DAYS`, `COCO_EVENTS_RETENTION_MAX_ROWS`, `COCO_EVENTS_PRUNE_INTERVAL_SECONDS`)

## Operational Playbook

### API Key Rotation
1. Generate a new key and add it to `COCO_API_IDENTITIES` with the same identity ID/type and scopes.
2. Deploy configuration with both old and new keys present (overlap window).
3. Update clients to use the new key and monitor auth denial events for stragglers.
4. Remove the old key from `COCO_API_IDENTITIES` after the overlap window ends.

### Scope Rollout
1. Introduce new scopes by adding them to the relevant identities in `COCO_API_IDENTITIES`.
2. Deploy config with additive scopes first and verify traffic with auth decision logs.
3. Tighten scopes by removing legacy broad scopes once clients are compliant.
4. Use `auth.denied` events to identify endpoints still hitting insufficient scope.

## Testing Status
- `go test ./...` passes as of 2026-02-11
- Lease tests include:
  - duplicate claim conflict behavior
  - lease heartbeat extension
  - explicit lease release endpoint behavior
  - lease lifecycle event emission (`created`, `expired`, `released`)
  - expired lease cleanup + task requeue
  - concurrent `/task` claim scenarios (single winner)
- Regression tests cover core v1 lifecycle and parent context behavior
- Error-envelope tests assert consistent v2 error schema on representative endpoints:
  - projects validation + method failures
  - agents validation + heartbeat failure paths
  - leases validation/conflict + method failures
  - health/version method failures
- Route-level contract tests validate v2 dispatch behavior through mux paths:
  - method-not-allowed responses for health/version/projects/agents/leases/runs
  - project not-found envelope via routed request
  - health success contract via routed request
- Auth tests cover:
  - runtime config parsing/validation for auth flags
  - unauthorized/authorized routed behavior for mutating endpoints
  - optional read-endpoint protection toggle
  - forbidden responses for valid keys with insufficient scope
