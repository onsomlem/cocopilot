# Cocopilot — Full Setup Guide

Complete start-to-finish guide for setting up Cocopilot in a new project.

---

## Prerequisites

- **Go 1.21+** — [https://go.dev/dl/](https://go.dev/dl/)
- **Node.js 18+** — required for MCP server and VS Code extension
- **VS Code** — for MCP and VSIX integration

---

## Step 1: Install Cocopilot

### Option A: Build from source

```bash
git clone https://github.com/onsomlem/cocopilot.git
cd cocopilot
go build -o cocopilot ./cmd/cocopilot
```

### Option B: Use Make

```bash
git clone https://github.com/onsomlem/cocopilot.git
cd cocopilot
make build
```

### Option C: Download binary

```bash
curl -L -o cocopilot https://github.com/onsomlem/cocopilot/releases/latest/download/cocopilot-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x cocopilot
```

---

## Step 2: Start the Server

```bash
./cocopilot
```

- Server starts on `http://127.0.0.1:8080`
- SQLite database auto-creates at `./tasks.db`
- All 27 migrations apply automatically on first run
- Kanban UI opens in your browser

### Optional: Custom configuration

```bash
# Custom port
COCO_HTTP_ADDR=:9090 ./cocopilot

# Custom database path
COCO_DB_PATH=./myproject.db ./cocopilot

# Suppress browser auto-open
COCO_NO_BROWSER=true ./cocopilot

# Enable API key authentication
COCO_REQUIRE_API_KEY=true COCO_API_KEY=$(openssl rand -hex 32) ./cocopilot
```

---

## Step 3: Create a Project

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-project",
    "description": "My new project",
    "workdir": "/path/to/your/repo"
  }' | jq .
```

Note the `id` in the response (e.g., `proj_abc123`). You'll use this for all project-scoped operations.

A `default` project (`proj_default`) is created automatically on first run.

---

## Step 4: Create Tasks

```bash
# Simple task
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Review codebase architecture",
    "instructions": "Analyze the project structure and suggest improvements for modularity and testability.",
    "type": "REVIEW",
    "priority": 50
  }' | jq .

# Task with tags
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Add input validation to API endpoints",
    "instructions": "Add request body validation for all POST/PATCH handlers. Use JSON schema or manual checks.",
    "type": "MODIFY",
    "priority": 70,
    "tags": ["security", "api"]
  }' | jq .
```

### Task types

| Type | Use case |
|------|----------|
| `MODIFY` | Code changes, feature implementation |
| `REVIEW` | Code review, architecture analysis |
| `ANALYZE` | Investigation, research |
| `TEST` | Writing or running tests |
| `PLAN` | Planning, task decomposition |

### Priority

0 (lowest) to 100 (highest). Higher priority tasks are claimed first.

---

## Step 5: Claim & Complete Tasks (Agent Workflow)

### Claim the next available task

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/tasks/claim-next \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "my-agent"}' | jq .
```

The response includes:
- **task** — Full task details with instructions
- **run** — A tracking record for this execution
- **context** — Memories, policies, repo files, dependencies
- **completion_contract** — Expected outputs for this task type

### Complete a task

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks/1/complete \
  -H "Content-Type: application/json" \
  -d '{
    "output": "Reviewed the architecture. Found 3 areas for improvement.",
    "result": {
      "summary": "Architecture review complete",
      "changes_made": ["Refactored handler structure", "Added service layer"],
      "files_touched": ["server/handlers.go", "server/services.go"]
    }
  }' | jq .
```

### Fail a task

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks/1/fail \
  -H "Content-Type: application/json" \
  -d '{
    "error": "Build failed after changes",
    "output": "Attempted refactor but tests broke"
  }' | jq .
```

---

## Step 6: Set Up MCP Server (VS Code Integration)

The MCP server exposes all Cocopilot tools to VS Code's Copilot Chat.

### Install

```bash
cd tools/cocopilot-mcp
npm install
npm run build
```

### Configure in VS Code

Create `.vscode/mcp.json` in your project root:

```json
{
  "servers": {
    "cocopilot": {
      "command": "node",
      "args": ["tools/cocopilot-mcp/dist/index.js"],
      "env": {
        "COCO_API_BASE": "http://localhost:8080"
      }
    }
  }
}
```

If API key auth is enabled, add:

```json
{
  "servers": {
    "cocopilot": {
      "command": "node",
      "args": ["tools/cocopilot-mcp/dist/index.js"],
      "env": {
        "COCO_API_BASE": "http://localhost:8080",
        "COCO_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Or install globally from npm

```bash
npm install -g cocopilot-mcp
```

Then use in `.vscode/mcp.json`:

```json
{
  "servers": {
    "cocopilot": {
      "command": "cocopilot-mcp",
      "env": {
        "COCO_API_BASE": "http://localhost:8080"
      }
    }
  }
}
```

### Available MCP Tools

Once configured, these tools are available in VS Code Copilot Chat:

| Tool | Description |
|------|-------------|
| `coco.project.list` | List all projects |
| `coco.project.create` | Create a project |
| `coco.task.create` | Create a task |
| `coco.task.list` | List tasks with filters |
| `coco.task.claim` | Claim next available task |
| `coco.task.complete` | Complete a task |
| `coco.task.save` | Complete task with output |
| `coco.run.get` | Get run details |
| `coco.run.steps` | Log a run step |
| `coco.run.logs` | Log run output |
| `coco.events.list` | List events |
| `coco.project.memory.query` | Query project memory |
| `coco.project.memory.put` | Store project memory |
| `coco.policy.list` | List policies |
| `coco.context_pack.create` | Create a context pack |

---

## Step 7: Set Up VS Code Extension (VSIX)

The extension provides a sidebar panel and commands for managing tasks directly in VS Code.

### Install from source

```bash
cd tools/cocopilot-vsix
npm install
npm run build
```

Then install the VSIX:

```bash
code --install-extension dist/cocopilot-vsix.vsix
```

### Or build and install in one step

```bash
cd tools/cocopilot-vsix
npm install && npm run build
code --install-extension dist/cocopilot-vsix.vsix
```

### Configuration

In VS Code settings (`settings.json`):

```json
{
  "cocopilot.apiBase": "http://localhost:8080",
  "cocopilot.apiKey": "your-api-key",
  "cocopilot.autoStartMcpServer": true
}
```

### Features

- **Sidebar panel** — View and manage tasks in the VS Code sidebar
- **Task creation** — Create tasks from the command palette
- **Status bar** — See active task count
- **Auto-start MCP** — Optionally start the MCP server automatically

---

## Step 8: Enable Repo Scanning

Scan your project files for language detection and context assembly:

```bash
# Trigger a scan
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/files/scan \
  -H "Content-Type: application/json" \
  -d '{"purge": true}' | jq .

# View discovered files
curl -s http://127.0.0.1:8080/api/v2/projects/proj_default/files | jq .
```

The scanner respects `.gitignore` and ignores common directories (`.git`, `node_modules`, `vendor`, `.venv`).

Scanned file metadata is included in the context when agents claim tasks.

---

## Step 9: Set Up Automation Rules (Optional)

Automation rules trigger actions when events occur (e.g., auto-create follow-up tasks on failure).

```bash
COCO_AUTOMATION_RULES='[
  {
    "event": "task.failed",
    "action": "create_task",
    "config": {
      "title_template": "Investigate failure: {{.TaskTitle}}",
      "instructions_template": "The task \"{{.TaskTitle}}\" failed. Investigate and fix.",
      "type": "ANALYZE",
      "priority": 80
    }
  }
]' ./cocopilot
```

### Automation events

| Event | When |
|-------|------|
| `task.created` | A new task is created |
| `task.completed` | A task is completed |
| `task.failed` | A task fails |
| `run.completed` | A run finishes |
| `run.failed` | A run fails |
| `repo.changed` | Files changed in repo |
| `repo.scanned` | Repo scan completed |
| `context.invalidated` | Context packs need refresh |

---

## Step 10: Use Policies (Optional)

Policies enforce constraints on task execution:

```bash
# Create a rate limit policy
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "claim-rate-limit",
    "type": "rate_limit",
    "config": {
      "max_claims_per_hour": 50,
      "max_claims_per_minute": 5
    },
    "enabled": true
  }' | jq .
```

---

## Step 11: Use Memory (Optional)

Store and retrieve knowledge that persists across tasks:

```bash
# Store a memory
curl -s -X PUT http://127.0.0.1:8080/api/v2/projects/proj_default/memory \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "repo",
    "key": "architecture-notes",
    "value": "The project uses a layered architecture with handlers -> services -> database."
  }' | jq .

# Query memories
curl -s "http://127.0.0.1:8080/api/v2/projects/proj_default/memory?scope=repo" | jq .
```

Memories are automatically included in the context when agents claim tasks.

---

## Step 12: Task Dependencies (Optional)

Create dependency chains between tasks:

```bash
# Task 2 depends on Task 1 (Task 2 won't be claimable until Task 1 is complete)
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks/2/dependencies \
  -H "Content-Type: application/json" \
  -d '{"depends_on_task_id": 1}' | jq .
```

---

## Complete Agent Loop Example

Here's a minimal agent loop in bash:

```bash
#!/bin/bash
API="http://127.0.0.1:8080"
PROJECT="proj_default"
AGENT_ID="my-agent"

while true; do
  # Claim next task
  RESPONSE=$(curl -s -X POST "$API/api/v2/projects/$PROJECT/tasks/claim-next" \
    -H "Content-Type: application/json" \
    -d "{\"agent_id\": \"$AGENT_ID\"}")

  # Check if a task was returned
  TASK_ID=$(echo "$RESPONSE" | jq -r '.task.id // empty')

  if [ -z "$TASK_ID" ]; then
    echo "No tasks available. Waiting 15s..."
    sleep 15
    continue
  fi

  INSTRUCTIONS=$(echo "$RESPONSE" | jq -r '.task.instructions')
  echo "Claimed task $TASK_ID: $INSTRUCTIONS"

  # --- Do work here ---
  RESULT="Completed the task successfully."

  # Complete the task
  curl -s -X POST "$API/api/v2/tasks/$TASK_ID/complete" \
    -H "Content-Type: application/json" \
    -d "{\"output\": \"$RESULT\", \"result\": {\"summary\": \"$RESULT\"}}"

  echo "Task $TASK_ID completed."
  sleep 5
done
```

---

## Monitoring

### Health check

```bash
curl -s http://127.0.0.1:8080/api/v2/health | jq .
```

### Server status

```bash
curl -s http://127.0.0.1:8080/api/v2/status | jq .
```

### Metrics

```bash
curl -s http://127.0.0.1:8080/api/v2/metrics | jq .
```

### Event stream (real-time)

```bash
curl -s http://127.0.0.1:8080/api/v2/events/stream
```

### Kanban UI

Open `http://127.0.0.1:8080` in your browser for the visual task board.

---

## Environment Variables Reference

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

---

## Troubleshooting

### "address already in use"

Another process is using port 8080. Either kill it or use a different port:

```bash
# Find what's using the port
lsof -i :8080

# Use a different port
COCO_HTTP_ADDR=:9090 ./cocopilot
```

### Database reset

Delete the database file and restart — migrations reapply automatically:

```bash
rm tasks.db
./cocopilot
```

### MCP server not connecting

1. Ensure the Go server is running on the configured port
2. Check `COCO_API_BASE` matches the server address
3. Run `node tools/cocopilot-mcp/dist/index.js` manually to see error output
