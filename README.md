# Cocopilot — Agentic Task Queue

A web-based task queue server for orchestrating LLM agents. Provides a Kanban-style UI and HTTP APIs (v1 + v2) for agents to poll for work and submit results. Runs as a single binary backed by SQLite — no external services required.

## Quick Start

```bash
go build -o cocopilot ./cmd/cocopilot
./cocopilot                     # starts on http://127.0.0.1:8080
# Open the URL in your browser — that's it.
```

Or with Make:

```bash
make build && ./cocopilot
```

## Build

Requires **Go 1.21+**. No CGO — uses pure-Go SQLite (`modernc.org/sqlite`).

```bash
make build          # build for current platform → ./cocopilot
make build-all      # cross-compile darwin/linux amd64/arm64 → dist/
make test           # go test -race -timeout 180s ./...
make lint           # go vet ./...
make clean          # remove build artifacts
```

Release packaging: `scripts/package.sh` builds the binary and creates a clean zip in `dist/`.

## Configuration

All settings are via environment variables. Defaults are safe for local use.

| Variable | Default | Description |
|----------|---------|-------------|
| `COCO_DB_PATH` | `./tasks.db` | SQLite database file path |
| `COCO_HTTP_ADDR` | `127.0.0.1:8080` | Server listen address |
| `COCO_REQUIRE_API_KEY` | `false` | Require API key for mutations |
| `COCO_API_KEY` | — | Shared API key (when auth enabled) |
| `COCO_NO_BROWSER` | `false` | Suppress auto-opening browser on start |
| `COCO_AUTOMATION_RULES` | — | JSON array of automation rules |
| `COCO_MAX_AUTOMATION_DEPTH` | `5` | Max automation recursion depth |
| `COCO_AUTOMATION_RATE_LIMIT` | `100` | Max automation executions/hour |
| `COCO_AUTOMATION_BURST_LIMIT` | `10` | Max automation executions/minute |
| `COCO_EVENTS_RETENTION_DAYS` | `30` | Auto-prune events older than N days |
| `COCO_EVENTS_RETENTION_MAX` | `0` | Max event rows to keep (0 = unlimited) |

## API Overview

### v1 (form-encoded, plain-text responses)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/task` | GET | Poll for next available task |
| `/create` | POST | Create a new task |
| `/save` | POST | Save task output / results |
| `/update-status` | POST | Update task status |
| `/delete` | POST | Delete a task |
| `/events` | GET | SSE stream (real-time updates) |

### v2 (JSON request/response)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v2/projects` | GET/POST | List or create projects |
| `/api/v2/projects/{id}/tasks` | GET/POST | List or create tasks in a project |
| `/api/v2/tasks/{id}` | GET/PATCH/DELETE | Task CRUD |
| `/api/v2/tasks/{id}/claim` | POST | Claim a task (lease-based) |
| `/api/v2/runs/{id}` | GET/PATCH | Run details and updates |
| `/api/v2/events` | GET | List events |
| `/api/v2/events/stream` | GET | SSE stream (v2 format) |
| `/api/v2/agents` | GET/POST | Agent registration |
| `/api/v2/health` | GET | Health check |
| `/api/v2/status` | GET | Server status overview |
| `/api/v2/metrics` | GET | Detailed metrics |
| `/api/v2/config` | GET | Runtime configuration |
| `/api/v2/backup` | GET | Download database backup |
| `/api/v2/restore` | POST | Upload database restore |

v2 errors follow the format: `{"error": {"code": "...", "message": "...", "details": {...}}}`

## Database

SQLite database auto-creates on first run. Migrations apply automatically on startup.

```bash
./cocopilot migrate status   # check migration status
./cocopilot migrate up       # apply migrations manually
```

Delete `tasks.db` to reset (migrations reapply on next start).

## Docker

```bash
# Build and run with docker compose
docker compose up -d

# Or build manually
make docker-build
docker run --rm -p 8080:8080 -v cocopilot-data:/data cocopilot:dev
```

See [docs/deployment.md](docs/deployment.md) for production deployment with systemd, nginx, and automated backups.

## Security — Local Only

Cocopilot is designed for **local / single-machine use**. It is NOT intended for public deployment.

- Default bind address is `127.0.0.1:8080` (localhost only).
- Do **not** bind to `0.0.0.0` in production or on untrusted networks.
- Enable API key auth for any non-trivial use: `COCO_REQUIRE_API_KEY=true COCO_API_KEY=<secret>`.
- The database file contains all task data — protect it accordingly.
- No TLS built in. Use a reverse proxy if you need HTTPS.

## Documentation

| File | Description |
|------|-------------|
| [docs/quickstart.md](docs/quickstart.md) | Getting started guide |
| [docs/full-setup-guide.md](docs/full-setup-guide.md) | Complete setup walkthrough |
| [docs/api/v2-summary.md](docs/api/v2-summary.md) | API v2 reference |
| [docs/features.md](docs/features.md) | Comprehensive feature list |
| [docs/deployment.md](docs/deployment.md) | Production deployment guide |
| [docs/security.md](docs/security.md) | Security guide |
| [docs/threat-model.md](docs/threat-model.md) | Threat model |
| [docs/troubleshooting.md](docs/troubleshooting.md) | Troubleshooting |
| [docs/task-authoring.md](docs/task-authoring.md) | Task writing best practices |
| [MIGRATIONS.md](MIGRATIONS.md) | Migration system details |

## Tooling

| Directory | Description |
|-----------|-------------|
| `tools/cocopilot-mcp` | MCP server for VS Code integration (Node.js) |
| `tools/cocopilot-vsix` | VS Code extension scaffold |

See each tool's README for setup instructions.

## Benchmarks

```bash
go test -bench . -benchtime 5x -timeout 60s .
```

Benchmarks cover task creation, claim throughput, concurrent claim contention,
and list endpoint performance. See `load_test.go`.
