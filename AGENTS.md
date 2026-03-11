# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

Cocopilot is an **Agentic Task Queue** - a web-based task queue server for orchestrating LLM agents. It provides a Kanban-style UI for managing tasks and HTTP APIs (v1 and v2) for agents to poll for work and submit results.

## Build and Run Commands

```bash
# Build
go build -o cocopilot ./cmd/cocopilot

# Run server (defaults to http://127.0.0.1:8080)
go run ./cmd/cocopilot

# Run with custom settings
COCO_DB_PATH=./dev.db COCO_HTTP_ADDR=:9090 go run ./cmd/cocopilot

# Run with API key auth
COCO_REQUIRE_API_KEY=true COCO_API_KEY=dev-secret go run ./cmd/cocopilot
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test file
go test -v -run TestTaskCreate

# Run tests with race detection
go test -race ./...
```

## Database Migrations

Migrations are in `migrations/` and auto-apply on server startup.

```bash
# Check migration status
./cocopilot migrate status

# Apply migrations manually
./cocopilot migrate up
```

Delete `tasks.db` to reset the database (migrations reapply on next run).

## Code Architecture

### Package Layout

- `cmd/cocopilot/main.go` — Thin entry point (imports `server` package and calls `Main()`)
- `server/` (`package server`) — Handlers, routes, HTTP glue, thin wrappers to internal packages
- `internal/models/` — Data types, status constants, JSON/null helpers
- `internal/dbstore/` — All database operations (split into projects, tasks, runs, agents, events, memory, policies, leases, etc.)
- `internal/config/` — RuntimeConfig, env parsing, automation rule types
- `internal/httputil/` — HTTP response helpers (WriteV2JSON, WriteV2Error, ClientIP, etc.)
- `internal/migrate/` — Schema migration system (Run, Status, Rollback)
- `internal/policy/` — Policy evaluation engine (rate limits, workflow constraints, resource quotas, time windows)
- `internal/ratelimit/` — Sliding window rate limiter
- `internal/scanner/` — File scanning, language detection, gitignore parsing
- `internal/worker/` — Task executor interface and implementations
- `internal/notifications/` — Webhook notifier, stalled task detection

### Core Files (`server/` package)

| File | Purpose |
|------|---------|
| `main.go` | `Main()` entry, DB init, background goroutines |
| `config.go` | Thin wrapper to `internal/config` |
| `routes.go` | `registerRoutes()`, route dispatching |
| `auth.go` | Auth middleware, API key validation, policy enforcement |
| `models_v2.go` | Type aliases re-exporting `internal/models` types |
| `db_v2.go` | Thin wrapper delegating to `internal/dbstore` |
| `handlers_v2_tasks.go` | v2 task CRUD, claim, complete, dependencies |
| `handlers_v2_projects.go` | v2 project CRUD, automation, policies, memory |
| `handlers_v2_events.go` | v2 event listing, SSE streaming |
| `handlers_v1.go` | v1 legacy API handlers |
| `ui_pages.go` | Kanban board HTML rendering |
| `automation.go` | Automation rules engine, emission dedupe |
| `assignment.go` | Task claiming, context assembly |
| `finalization.go` | Task completion/failure |
| `migrations.go` | Thin wrapper to `internal/migrate` |

### Key Architectural Patterns

- **Handler files**: HTTP handlers split across `handlers_v2_*.go`, `handlers_v1.go`, and `ui_*.go` in `server/` package
- **Dual API versioning**: v1 routes (`/task`, `/create`, `/save`) coexist with v2 routes (`/api/v2/*`)
- **SQLite with pure-Go driver**: Uses `modernc.org/sqlite`, no CGO required
- **SSE for real-time**: Two subscriber systems - `sseClients` for v1, `v2EventSubscribers` for v2
- **Lease-based task claiming**: Prevents multiple agents from claiming the same task
- **CORS middleware**: `withCORS()` in `routes.go` echoes `Origin`, handles preflight OPTIONS
- **Request logging**: `withRequestLog()` logs method+path for non-static requests

### Entity Relationships

```
Project (1) ─┬─ (*) Task ─┬─ (*) Run ─┬─ (*) RunStep
             │            │           ├─ (*) RunLog
             │            │           └─ (*) Artifact
             │            └─ (*) TaskDependency
             ├─ (*) Memory
             ├─ (*) Policy
             ├─ (*) Event
             └─ (*) AutomationEmission (dedupe records)
```

### API Structure

- **v1 endpoints**: Form-encoded, plain-text responses (`/task`, `/create`, `/save`, `/events`)
- **v2 endpoints**: JSON request/response, structured errors (`/api/v2/*`)
- **v2 error format**: `{"error": {"code": "...", "message": "...", "details": {...}}}`

### Tools (separate packages)

| Directory | Purpose |
|-----------|---------|
| `tools/cocopilot-mcp` | MCP server for VS Code integration (Node.js) |
| `tools/cocopilot-vsix` | VS Code extension scaffold |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `COCO_DB_PATH` | `./tasks.db` | SQLite database path |
| `COCO_HTTP_ADDR` | `127.0.0.1:8080` | Server listen address |
| `COCO_REQUIRE_API_KEY` | `false` | Require API key for v2 mutations |
| `COCO_API_KEY` | - | Shared API key when auth enabled |
| `COCO_AUTOMATION_RULES` | - | JSON array of automation rules |
| `COCO_MAX_AUTOMATION_DEPTH` | `5` | Max recursion depth for automation |
| `COCO_AUTOMATION_RATE_LIMIT` | `100` | Max automation executions per hour |
| `COCO_AUTOMATION_BURST_LIMIT` | `10` | Max automation executions per minute |

## Conventions

- **Timestamps**: Always use `nowISO()` (RFC3339 with microseconds in UTC)
- **Error handling**: v2 handlers use `writeV2Error()` for consistent error responses
- **JSON helpers**: Use `marshalJSON()`/`unmarshalJSON()` for JSON column storage
- **Null handling**: Use `sql.NullString`/`sql.NullInt64` for nullable DB columns, convert with `nullString()`/`ptrString()`

## Documentation

See `docs/` for detailed documentation:
- `docs/api/v2-summary.md` - API v2 endpoint reference
- `docs/state/architecture.md` - System architecture
- `docs/security.md` - Security deployment guide
- `docs/threat-model.md` - Threat model and attack surface analysis
- `docs/quickstart.md` - Getting started guide
- `docs/troubleshooting.md` - Troubleshooting guide
- `docs/task-authoring.md` - Task writing best practices
- `MIGRATIONS.md` - Migration system details
- `SECURITY.md` - Vulnerability reporting policy

## Benchmarks

```bash
go test -bench . -benchtime 5x -timeout 60s .
```

Benchmark suite in `load_test.go` covers task creation, claim throughput,
concurrent claim contention, and list endpoint performance.
