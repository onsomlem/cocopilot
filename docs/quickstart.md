# Getting Started

## Install and Launch

```bash
git clone https://github.com/onsomlem/cocopilot.git
cd cocopilot
go build -o cocopilot ./cmd/cocopilot
./cocopilot
```

Your browser opens to `http://127.0.0.1:8080` — you'll see the dashboard.

## Your First 5 Minutes

### 1. Open the default project

A default project (`proj_default`) is created on first run. Click it in the sidebar to open the board.

### 2. Seed demo data

Click **Seed Demo** in the dashboard header. This populates sample tasks, agents, and runs so you can see the full UI immediately.

### 3. Explore the board

- **Kanban columns** show tasks by status (Pending → In Progress → Completed / Failed)
- **Filters** let you narrow by type, priority, agent, or tags
- **Click any task** to see full details, run history, and context

### 4. Watch real-time updates

The dashboard uses SSE (Server-Sent Events) — task status changes, agent heartbeats, and new events appear instantly without page refresh.

### 5. Create your first real task

Click **New Task** on the board, or from the terminal:

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Review README", "instructions": "Check for outdated info", "priority": 50}'
```

## Connect an Agent

Agents are programs that poll for tasks, do work, and report results.

### Simple agent loop

```bash
# 1. Claim a task
CLAIM=$(curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/tasks/claim-next \
  -H "Content-Type: application/json" -d '{"agent_id": "my-agent"}')

# 2. Read the instructions
echo "$CLAIM" | jq '.task.instructions'

# 3. Do the work, then complete
TASK_ID=$(echo "$CLAIM" | jq -r '.task.id')
curl -s -X POST "http://127.0.0.1:8080/api/v2/tasks/$TASK_ID/complete" \
  -H "Content-Type: application/json" \
  -d '{"output": "Reviewed and updated", "summary": "README looks good"}'
```

The claim response includes everything the agent needs:
- **task** — full details and instructions
- **run** — tracking record for this execution
- **context** — project memories, policies, repo files
- **completion_contract** — expected output format

### Built-in worker

For automated processing without writing your own agent:

```bash
./cocopilot worker --project=proj_default
```

### MCP integration (VS Code)

Connect Cocopilot tools directly to VS Code Copilot Chat:

```bash
cd tools/cocopilot-mcp && npm install && npm run build
```

Then add to `.vscode/mcp.json`:

```json
{
  "servers": {
    "cocopilot": {
      "command": "node",
      "args": ["tools/cocopilot-mcp/dist/index.js"],
      "env": { "COCO_API_BASE": "http://localhost:8080" }
    }
  }
}
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `COCO_DB_PATH` | `./tasks.db` | SQLite database path |
| `COCO_HTTP_ADDR` | `127.0.0.1:8080` | Listen address |
| `COCO_REQUIRE_API_KEY` | `false` | Require API key for mutations |
| `COCO_API_KEY` | — | Shared API key |
| `COCO_NO_BROWSER` | `false` | Suppress auto-open browser |

## Validation

### Run tests

```bash
go test ./...                    # all tests
go test -run TestGoldenPath ./server/  # golden path lifecycle tests
go test -run TestSmoke ./server/       # UI & route smoke tests
```

### Verify repo hygiene

```bash
make verify-repo     # no binaries, .vsix, node_modules in git index
make verify-release  # check a release zip for leaking artifacts
```

## Next Steps

- [Full Setup Guide](full-setup-guide.md) — MCP, VSIX, Docker, production deployment
- [Task Authoring](task-authoring.md) — Writing effective tasks for agents
- [API Reference](api/v2-summary.md) — Full v2 endpoint documentation
