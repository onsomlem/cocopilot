# Cocopilot Feature List

Comprehensive reference of all features in the Cocopilot Agentic Task Queue.

---

## Core Infrastructure

- **Single-binary Go server** — no external dependencies, pure-Go SQLite driver (`modernc.org/sqlite`)
- **Automatic schema migrations** — 27 migrations auto-apply on startup
- **Environment variable configuration** — all settings via `COCO_*` env vars
- **CORS middleware** — echoes `Origin`, handles preflight `OPTIONS`
- **Request logging** — structured logging for non-static requests

## Projects & Workspaces

- Multi-project support with isolated scopes
- Default project (`proj_default`) for v1 backward compatibility
- Project-scoped tasks, events, memory, and policies
- Project tree snapshots for repository awareness
- File change tracking with git-style status
- Project audit log with filtering and export

## Task Management

- Full CRUD (create, read, update, delete)
- Task types: `ANALYZE`, `MODIFY`, `TEST`, `REVIEW`, `DOC`, `RELEASE`, `ROLLBACK`
- Numeric priority ordering
- Tag-based categorization and full-text search
- Task dependency graph (DAG) with blocking semantics
- Task templates with variable interpolation
- Parent-child relationships
- Pagination, sorting (`created_at`, `updated_at`), and status filtering

## Task Execution

- **Execution ledger** — every execution attempt recorded as a Run
- Run steps with status tracking
- Structured logging (stdout/stderr capture)
- Artifact attachment (diffs, patches, test results, reports)
- Artifact comments for review feedback
- Tool invocation tracking per run

## Agent Coordination

- **Lease-based task claiming** — exclusive claiming prevents conflicts
- Time-bound leases with configurable expiration
- Heartbeat mechanism for lease renewal
- Agent registration with capabilities and metadata
- Agent activity monitoring (`last_seen` timestamps)
- Context assembly from parent tasks and project memory

## Automation Engine

- Event-driven automation rules (`task.completed` trigger → create follow-up)
- Template support with variable interpolation (`{{task.title}}`, etc.)
- Configurable rate limiting (per-hour and per-minute burst)
- Circuit breaker for failure protection
- Automation depth limiting (default: 5 levels)
- Simulation endpoint for dry-run testing
- Replay capability for event re-processing
- Deduplicated emission tracking

## Real-Time Events

- **Server-Sent Events (SSE)** for push-based updates
- v1 and v2 independent SSE subscriber systems
- Event replay with `since` parameter (RFC3339 timestamp or ID)
- Project-scoped event streaming
- Event type filtering
- Configurable heartbeat interval and replay limits

## Event System

- Append-only event log with full audit trail
- Event types: `task.created`, `task.completed`, `task.updated`, `run.*`, `memory.*`, `agent.*`, etc.
- Filtering by type, project, task, entity, and time range
- Pagination with total counts
- Configurable retention (max age in days, max row count)
- Background pruning goroutine

## Persistent Memory

- Project-level key-value knowledge storage
- Memory scopes: `GLOBAL`, `MODULE`, `FILE`, `TASK`, `RUN`
- Source reference tracking
- Queryable via project-scoped API

## Context Packs

- Automated context generation for LLM agents
- Budget-controlled file inclusion (max files, bytes, snippets)
- File snippets with line ranges
- Related task context inclusion
- Decision audit trail
- Repository state snapshot (git HEAD, dirty flag)

## Policies & Governance

- Project-scoped policy rules
- Rule types: `automation.block`, `completion.block`, `task.create.block`, `task.update.block`, `task.delete.block`
- Enable/disable toggle per policy
- Rate limit and workflow constraint evaluation

## Authentication & Security

- Optional API key authentication (`COCO_REQUIRE_API_KEY`)
- Scope-based authorization (read, write, admin)
- Auth decision logging to event stream
- Localhost-only binding by default

## APIs

### v1 (Form-encoded, plain-text responses)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/task` | GET | Poll for next available task |
| `/create` | POST | Create a task |
| `/save` | POST | Save task output/result |
| `/update-status` | POST | Update task status |
| `/delete` | POST | Delete a task |
| `/events` | GET | SSE event stream |
| `/api/tasks` | GET | List tasks with filters |
| `/instructions` | GET | Agent setup instructions |
| `/` | GET | Kanban board UI |

### v2 (JSON request/response, structured errors)

| Resource | Endpoints |
|----------|-----------|
| **Projects** | CRUD, automation rules, simulation, replay, policies, memory, context packs, tree, changes, audit, dashboard, notifications, templates, files |
| **Tasks** | CRUD, claim, complete, dependencies, list with filters/pagination |
| **Runs** | Detail, steps, logs, artifacts |
| **Leases** | Create, heartbeat, release |
| **Events** | List with filters, SSE stream with replay |
| **Agents** | Register, list, detail, delete |
| **System** | Health, status, metrics, config, version, backup, restore |

## Data Operations

- **Database backup** — `GET /api/v2/backup` streams raw SQLite file
- **Database restore** — `POST /api/v2/restore` replaces database
- **Project export** — JSON archive of all project data
- **Project import** — restore from JSON archive

## Web UI

- Kanban-style drag-and-drop board (Alpine.js)
- Dashboard with task distribution overview
- Health dashboard with system metrics
- Project management page
- Agent monitoring view
- Run viewer with steps and logs
- Task dependency graph visualization
- Repository file graph
- Diff viewer
- Event browser with filters
- Policy editor
- Settings and automation configuration
- Memory browser
- Context pack builder
- Audit log viewer
- VS Code-themed dark styling

## Developer Tools

- **MCP server** (`tools/cocopilot-mcp/`) — Model Context Protocol integration for VS Code
- **VS Code extension** (`tools/cocopilot-vsix/`) — extension scaffold
- **OpenAPI spec** — v2 API specification in `docs/schema/`

## Operations

- Database migration CLI (`migrate status`, `migrate up`)
- Background lease cleanup goroutine
- Background event retention pruning
- Stalled task detection
- Server uptime tracking
