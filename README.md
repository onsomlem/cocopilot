# Cocopilot — Local Agent Orchestration Console

Cocopilot is an operator-facing console for orchestrating LLM agents on your machine. It manages **tasks, runs, agents, context, and approvals** through a real-time dashboard — backed by a single binary and SQLite. No external services required.

**What you get:**
- **Dashboard** — Kanban board with real-time status, filters, and project switching
- **Task lifecycle** — Create, prioritize, assign, claim, complete, fail tasks with full audit trail
- **Agent management** — Register agents, track heartbeats, monitor active leases
- **Run tracking** — Every task execution is a run with steps, logs, and artifacts
- **Context packs** — Attach repo files, memories, and policies to agent claims
- **Automation** — Rules engine for auto-creating follow-up tasks with governance (rate limits, circuit breaker, recursion depth)
- **Events & SSE** — Real-time event stream for all state changes (19 event families)
- **Task dependencies** — DAG-based ordering with cycle detection and visual graph
- **Planning pipeline** — Automated planning cycles with quality scoring and seed prompts
- **File scanning** — Repository file indexing with .gitignore support
- **Memory** — Project-scoped key-value storage for cross-task knowledge
- **Policies** — Governance rules for rate limits, workflow constraints, resource quotas
- **Audit trail** — Full compliance logging with per-project export
- **Templates & prompts** — Reusable task templates and prompt management
- **MCP & VSIX** — VS Code integration via MCP server (48 tools) and extension

## Quick Start

```bash
# Build and launch
go build -o cocopilot ./cmd/cocopilot
./cocopilot
```

Your browser opens to `http://127.0.0.1:8080`. From the dashboard:

1. **Open a project** — a default project is created automatically
2. **Seed demo data** — click "Seed Demo" to populate sample tasks and agents
3. **Watch the board** — tasks appear on the Kanban board with live status updates

To connect an agent or use the built-in worker, see [Getting Started](docs/quickstart.md).

## Build

Requires **Go 1.21+**. No CGO — uses pure-Go SQLite (`modernc.org/sqlite`).

```bash
make build          # build for current platform → ./cocopilot
make build-all      # cross-compile darwin/linux amd64/arm64 → dist/
make test           # go test -race -timeout 180s ./...
make lint           # go vet ./...
make release        # build + package clean release zip
make verify-release # validate release zip contents
```

## Configuration

All settings are via environment variables. Defaults are safe for local use.

| Variable | Default | Description |
|----------|---------|-------------|
| `COCO_DB_PATH` | `./tasks.db` | SQLite database file path |
| `COCO_HTTP_ADDR` | `127.0.0.1:8080` | Server listen address |
| `COCO_REQUIRE_API_KEY` | `false` | Require API key for mutations |
| `COCO_API_KEY` | — | Shared API key (when auth enabled) |
| `COCO_NO_BROWSER` | `false` | Suppress auto-opening browser on start |

See [docs/full-setup-guide.md](docs/full-setup-guide.md) for the full configuration reference.

## The Operator Workflow

This is the real way to use Cocopilot — not just raw API calls.

### 1. Launch and open a project

```bash
./cocopilot                     # dashboard opens at http://127.0.0.1:8080
```

### 2. Create tasks from the UI or API

Use the dashboard "New Task" button, or:

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Review auth module", "instructions": "Check for security issues", "priority": 70}'
```

### 3. Connect an agent

Agents claim tasks via the v2 API. The simplest loop:

```bash
# Claim → work → complete
CLAIM=$(curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/tasks/claim-next \
  -H "Content-Type: application/json" -d '{"agent_id": "my-agent"}')
# ... agent does work ...
TASK_ID=$(echo "$CLAIM" | jq -r '.task.id')
curl -s -X POST "http://127.0.0.1:8080/api/v2/tasks/$TASK_ID/complete" \
  -H "Content-Type: application/json" -d '{"output": "Done", "summary": "Reviewed auth module"}'
```

### 4. Monitor everything in the dashboard

- **Board** — drag tasks between columns, filter by status/type/agent
- **Agents** — see registered agents, last heartbeat, active runs
- **Runs** — drill into each execution with steps, logs, and artifacts
- **Events** — real-time feed of all state changes (SSE-powered)

## Security — Local Only

Cocopilot is designed for **local / single-machine use**.

- Default bind address is `127.0.0.1:8080` (localhost only)
- Do **not** bind to `0.0.0.0` on untrusted networks
- Enable API key auth for shared environments: `COCO_REQUIRE_API_KEY=true COCO_API_KEY=<secret>`
- No TLS built in — use a reverse proxy if you need HTTPS

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/quickstart.md) | First-run walkthrough |
| [Full Setup Guide](docs/full-setup-guide.md) | Complete setup with MCP, VSIX, and Docker |
| [Task Authoring](docs/task-authoring.md) | Writing effective tasks for agents |

| Reference | Description |
|-----------|-------------|
| [API v2 Reference](docs/api/v2-summary.md) | Full endpoint documentation |
| [Deployment](docs/deployment.md) | systemd, nginx, Docker production setup |
| [Security](docs/security.md) | Security model and hardening |
| [Threat Model](docs/threat-model.md) | Attack surface analysis |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and fixes |
| [Migrations](MIGRATIONS.md) | Database migration system |

## Tooling

| Directory | Description |
|-----------|-------------|
| `tools/cocopilot-mcp` | MCP server — exposes Cocopilot tools to VS Code Copilot Chat |
| `tools/cocopilot-vsix` | VS Code extension — sidebar panel for task management |

## Docker

```bash
docker compose up -d
# Or: make docker-build && docker run --rm -p 8080:8080 -v cocopilot-data:/data cocopilot:dev
```

See [docs/deployment.md](docs/deployment.md) for production deployment.
