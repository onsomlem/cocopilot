# Task 29 — SQL Claim Query & V2 Routes

## 1. SQL Claim Query in `getTaskHandler` (`/task` endpoint)

Located in `main.go` inside `getTaskHandler`, the SQL used to find and claim the next available task is:

```sql
SELECT t.id, t.status, t.instructions, t.parent_task_id, t.created_at, t.updated_at
FROM tasks t
LEFT JOIN leases l ON t.id = l.task_id AND l.expires_at > ?
WHERE t.project_id = ? AND t.status IN (?, ?) AND l.id IS NULL
ORDER BY
    CASE WHEN t.status = ? THEN 0 ELSE 1 END,
    t.id ASC
LIMIT 1
```

Parameters bound at call-site: `nowISO()`, `projectID`, `StatusNotPicked`, `StatusInProgress`, `StatusNotPicked`.

The query selects the highest-priority unclaimed task by:
- LEFT JOINing `leases` to exclude tasks with an active (non-expired) lease.
- Filtering to the requested project and `not_picked` / `in_progress` statuses.
- Ordering `not_picked` tasks before `in_progress`, then by ascending `id`.
- Returning only 1 row.

## 2. V2 Route Registrations

All v2 routes registered in `main.go` (via `mux.HandleFunc`):

| Route | Handler(s) / Middleware | Notes |
|---|---|---|
| `GET /api/v2/health` | `v2HealthHandler` | Health check |
| `GET /api/v2/version` | `v2VersionHandler(cfg)` | Version info |
| `GET /api/v2/config` | `v2ConfigHandler(cfg)` | Server config |
| `/api/v2/projects` | `v2ProjectsRouteHandler` → POST `v2CreateProjectHandler`, GET `v2ListProjectsHandler` | Project CRUD |
| `/api/v2/projects/` | `v2ProjectRouteHandler(heartbeatInterval, sseReplayLimitMax)` | Sub-routes include tasks, events/stream, tree, audit, changes, automation rules/simulate/replay, context-packs, repo-files, memory, policies, **claim-next** (`POST /api/v2/projects/:id/tasks/claim-next`) |
| `/api/v2/tasks` | `v2TasksRouteHandler` (with `policyEnforcementMiddleware`) → POST create, GET list | Task collection |
| `/api/v2/tasks/` | `v2TaskDetailRouteHandler` (with `policyEnforcementMiddleware`) | GET/PUT/PATCH/DELETE single task; sub-routes: complete, claim, dependencies, response |
| `/api/v2/runs/` | `v2RunsRouteHandler` (with `policyEnforcementMiddleware`) | Run detail, steps, logs, artifacts |
| `/api/v2/events` | `v2ListEventsHandler` | List events |
| `/api/v2/events/stream` | `v2EventsStreamHandler(heartbeatInterval, sseReplayLimitMax)` | SSE event stream |
| `/api/v2/agents` | `v2AgentsRouteHandler` | POST register, GET list agents |
| `/api/v2/agents/` | `v2AgentActionHandler` | GET/DELETE/heartbeat for individual agents |
| `/api/v2/leases` | `v2LeaseHandler` | Lease operations |
| `/api/v2/leases/` | `v2LeaseActionHandler` | Individual lease actions (renew/release) |
| `/api/v2/context-packs/` | `v2ContextPackDetailHandler` | Context pack detail + file sub-routes |

All v2 routes are wrapped with `withV2Auth(cfg, ...)`. Task/run mutation routes additionally pass through `policyEnforcementMiddleware`.
