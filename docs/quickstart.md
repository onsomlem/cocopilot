# Quickstart Guide

## Install

**Option A: Download binary**
```bash
# Download from GitHub Releases (replace with actual URL)
curl -L -o cocopilot https://github.com/onsomlem/cocopilot/releases/latest/download/cocopilot-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x cocopilot
```

**Option B: Build from source**
```bash
git clone https://github.com/onsomlem/cocopilot.git
cd cocopilot
go build -o cocopilot ./cmd/cocopilot
```

## Start the Server

```bash
./cocopilot
# Server starts on http://127.0.0.1:8080
# Dashboard available at http://127.0.0.1:8080
```

Or use quickstart mode (creates default project + opens browser):
```bash
./cocopilot quickstart
```

## Create a Task

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "instructions": "Review the main.go file and suggest improvements",
    "title": "Code review",
    "type": "review",
    "priority": 50
  }'
```

## Claim a Task (Agent Loop)

```bash
# Claim the next available task
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/tasks/claim-next \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "my-agent"}'
```

The response includes:
- Task details and instructions
- Lease (your claim token)
- Run (tracking your execution)
- Context (memories, policies, dependencies, repo files)
- Completion contract (what outputs are expected)

## Complete a Task

```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks/1/complete \
  -H "Content-Type: application/json" \
  -d '{
    "status": "SUCCEEDED",
    "output": "Reviewed main.go — found 3 areas for improvement",
    "summary": "Code review complete"
  }'
```

## Built-in Worker

For automated task processing:
```bash
./cocopilot worker --project=proj_default
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `COCO_DB_PATH` | `./tasks.db` | SQLite database path |
| `COCO_HTTP_ADDR` | `127.0.0.1:8080` | Listen address |
| `COCO_REQUIRE_API_KEY` | `false` | Require API key for mutations |
| `COCO_API_KEY` | — | Shared API key |

See the full list in the README.
