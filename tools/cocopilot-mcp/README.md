# Cocopilot MCP Server

Minimal MCP server that exposes Cocopilot tools backed by the existing HTTP API.

## Installation

### From npm (recommended)

```bash
npm install -g cocopilot-mcp
```

### From source

```bash
git clone https://github.com/onsomlem/cocopilot.git
cd cocopilot/tools/cocopilot-mcp
npm install
npm run build
npm link   # optional: makes 'cocopilot-mcp' available globally
```

## Requirements

- Node.js 18+
- Cocopilot API running (default: http://localhost:8080)

## Setup

```bash
npm install
```

## Run (dev)

```bash
npm run dev
```

## Build and run

```bash
npm run build
npm run start
```

## Build/release checklist

- Prereqs: Node.js 18+ with npm available (`node -v`, `npm -v`)
- If `npm` is missing, install/reinstall Node.js (or use nvm) and ensure npm is on PATH
- Build the server: `npm run build`
- Run the release helper (build + tools.json sync check): `npm run release:check`
- Verify `tools.json` matches the current routes and input schemas
- Confirm env setup: `COCO_API_BASE`, `COCO_API_KEY` or `COCO_API_HEADERS`, `COCO_PROJECT_ID`

## Configuration

- `COCO_API_BASE` (optional): Base URL for the Cocopilot API. Defaults to http://localhost:8080 and must be a valid http(s) URL.
- `COCO_API_KEY` (optional): API key used for `Authorization: Bearer <key>` and `X-API-Key: <key>` headers.
- `COCO_PROJECT_ID` (optional): Default project id for project-scoped tools.
- `COCO_API_HEADERS` (optional): JSON object of headers to send with every request (overrides `COCO_API_KEY`).

See `.env.example` for a starter set of environment variables.

Example `.vscode/mcp.json` entry:

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

Example:

```bash
export COCO_API_KEY="your-key"
```

```bash
export COCO_API_HEADERS='{"Authorization":"Bearer your-key","X-API-Key":"your-key"}'
```

## Tools

See [tools.json](tools.json) for the MCP tool input schemas.

Keep `tools.json` in sync with the server routes and request/response contracts. When adding a tool, update `tools.json` to include the new tool name, its input schema, and any required fields so MCP clients validate arguments correctly.

- `coco.project.list` -> `GET /api/v2/projects`
- `coco.project.create` -> `POST /api/v2/projects`
- `coco.project.update` -> `PATCH /api/v2/projects/{projectId}`
- `coco.project.get` -> `GET /api/v2/projects/{projectId}`
- `coco.project.delete` -> `DELETE /api/v2/projects/{projectId}`
- `coco.config.get` -> `GET /api/v2/config`
- `coco.version.get` -> `GET /api/v2/version`
- `coco.health.get` -> `GET /api/v2/health`
- `coco.agent.list` -> `GET /api/v2/agents`
- `coco.agent.get` -> `GET /api/v2/agents/{agentId}`
- `coco.agent.delete` -> `DELETE /api/v2/agents/{agentId}`
- `coco.project.tasks.list` -> `GET /api/v2/projects/{projectId}/tasks`
- `coco.project.memory.query` -> `GET /api/v2/projects/{projectId}/memory`
- `coco.project.memory.put` -> `PUT /api/v2/projects/{projectId}/memory`
- `coco.project.audit.list` -> `GET /api/v2/projects/{projectId}/audit` (optional `type`, `since`, `limit`, `offset`)
- `coco.project.events.replay` -> `GET /api/v2/projects/{projectId}/events/replay` (requires `since_id`, optional `limit`)
- `coco.project.automation.rules` -> `GET /api/v2/projects/{projectId}/automation/rules`
- `coco.project.automation.simulate` -> `POST /api/v2/projects/{projectId}/automation/simulate`
- `coco.project.automation.replay` -> `POST /api/v2/projects/{projectId}/automation/replay` (requires `since_event_id`, optional `limit`)
- `coco.policy.list` -> `GET /api/v2/projects/{projectId}/policies`
- `coco.policy.get` -> `GET /api/v2/projects/{projectId}/policies/{policyId}`
- `coco.policy.create` -> `POST /api/v2/projects/{projectId}/policies`
- `coco.policy.update` -> `PATCH /api/v2/projects/{projectId}/policies/{policyId}`
- `coco.policy.delete` -> `DELETE /api/v2/projects/{projectId}/policies/{policyId}`
- `coco.context_pack.create` -> `POST /api/v2/projects/{projectId}/context-packs`
- `coco.context_pack.get` -> `GET /api/v2/context-packs/{packId}`
- `coco.task.create` -> `POST /api/v2/tasks`
- `coco.task.list` -> `GET /api/v2/tasks`
- `coco.task.complete` -> `POST /api/v2/tasks/{taskId}/complete`
- `coco.task.update` -> `PATCH /api/v2/tasks/{taskId}`
- `coco.task.delete` -> `DELETE /api/v2/tasks/{taskId}`
- `coco.task.dependencies.list` -> `GET /api/v2/tasks/{taskId}/dependencies`
- `coco.task.dependencies.create` -> `POST /api/v2/tasks/{taskId}/dependencies`
- `coco.task.dependencies.delete` -> `DELETE /api/v2/tasks/{taskId}/dependencies/{dependsOnTaskId}`
- `coco.lease.create` -> `POST /api/v2/leases`
- `coco.lease.heartbeat` -> `POST /api/v2/leases/{leaseId}/heartbeat`
- `coco.lease.release` -> `POST /api/v2/leases/{leaseId}/release`
- `coco.events.list` -> `GET /api/v2/events`
- `coco.run.get` -> `GET /api/v2/runs/{runId}`
- `coco.run.steps` -> `POST /api/v2/runs/{runId}/steps`
- `coco.run.logs` -> `POST /api/v2/runs/{runId}/logs`
- `coco.run.artifacts` -> `POST /api/v2/runs/{runId}/artifacts`
- `coco.task.claim` -> `GET /task`
- `coco.task.claim.v2` -> `POST /api/v2/tasks/{taskId}/claim`
- `coco.task.save` -> `POST /save`

## Usage notes

- `coco.task.claim` returns the raw text response from the v1 `GET /task` endpoint.
- `coco.task.claim.v2` returns the JSON response from `POST /api/v2/tasks/{taskId}/claim`.
- `coco.task.save` expects `task_id` and `message`, submits form-encoded data, and returns the raw text response.
- Tool failures include HTTP status and response body when available.
- JSON parse failures surface the response body to help debug invalid payloads.
- Basic request/error logs are emitted to stderr.

## Examples

List tasks for a project with filters:

```json
{
	"name": "coco.task.list",
	"arguments": {
		"project_id": "proj_123",
		"status": "OPEN",
		"limit": 25,
		"offset": 0
	}
}
```

Create a project:

```json
{
	"name": "coco.project.create",
	"arguments": {
		"name": "Agent Research",
		"description": "Notes and tasks for agent experiments",
		"status": "ACTIVE"
	}
}
```

Update a project:

```json
{
	"name": "coco.project.update",
	"arguments": {
		"project_id": "proj_123",
		"name": "Agent Research v2",
		"description": "Expanded scope",
		"status": "PAUSED"
	}
}
```

Fetch a project by id:

```json
{
	"name": "coco.project.get",
	"arguments": {
		"project_id": "proj_123"
	}
}
```

Delete a project by id:

```json
{
	"name": "coco.project.delete",
	"arguments": {
		"project_id": "proj_123"
	}
}
```

List agents with filters and sorting:

```json
{
	"name": "coco.agent.list",
	"arguments": {
		"status": "active",
		"since": "2026-02-10T00:00:00Z",
		"limit": 50,
		"offset": 0,
		"sort": "last_seen:desc"
	}
}
```

Fetch server config:

```json
{
	"name": "coco.config.get",
	"arguments": {}
}
```

Fetch server version info:

```json
{
	"name": "coco.version.get",
	"arguments": {}
}
```

Fetch server health info:

```json
{
	"name": "coco.health.get",
	"arguments": {}
}
```

Fetch an agent by id:

```json
{
	"name": "coco.agent.get",
	"arguments": {
		"agent_id": "agent_123"
	}
}
```

Delete an agent by id:

```json
{
	"name": "coco.agent.delete",
	"arguments": {
		"agent_id": "agent_123"
	}
}
```

Update a task with new instructions and status:

```json
{
	"name": "coco.task.update",
	"arguments": {
		"task_id": 123,
		"instructions": "Refine the migration plan",
		"status": "RUNNING"
	}
}
```

Delete a task by id:

```json
{
	"name": "coco.task.delete",
	"arguments": {
		"task_id": 123
	}
}
```

List task dependencies:

```json
{
	"name": "coco.task.dependencies.list",
	"arguments": {
		"task_id": 123
	}
}
```

Create a task dependency:

```json
{
	"name": "coco.task.dependencies.create",
	"arguments": {
		"task_id": 123,
		"depends_on_task_id": 456
	}
}
```

Delete a task dependency:

```json
{
	"name": "coco.task.dependencies.delete",
	"arguments": {
		"task_id": 123,
		"depends_on_task_id": 456
	}
}
```

Claim a task by id (v2):

```json
{
	"name": "coco.task.claim.v2",
	"arguments": {
		"task_id": 123
	}
}
```

Create a lease for a task claim:

```json
{
	"name": "coco.lease.create",
	"arguments": {
		"task_id": 123,
		"agent_id": "agent_123",
		"mode": "exclusive"
	}
}
```

Heartbeat a lease:

```json
{
	"name": "coco.lease.heartbeat",
	"arguments": {
		"lease_id": "lease_xyz"
	}
}
```

Release a lease with a reason:

```json
{
	"name": "coco.lease.release",
	"arguments": {
		"lease_id": "lease_xyz",
		"reason": "task completed"
	}
}
```

List project tasks with project-specific filters:

```json
{
	"name": "coco.project.tasks.list",
	"arguments": {
		"project_id": "proj_123",
		"status": "OPEN",
		"type": "REVIEW",
		"tag": "agent",
		"q": "planner",
		"limit": 25,
		"offset": 0
	}
}
```

Query project memory with filters:

```json
{
	"name": "coco.project.memory.query",
	"arguments": {
		"project_id": "proj_123",
		"scope": "agent",
		"key": "notes",
		"q": "search term"
	}
}
```

Store project memory with optional tags:

```json
{
	"name": "coco.project.memory.put",
	"arguments": {
		"project_id": "proj_123",
		"scope": "GLOBAL",
		"key": "release_notes",
		"value": {
			"title": "v1",
			"count": 2
		},
		"tags": ["task_1", "task_2"]
	}
}
```

List project audit events with filters and pagination:

```json
{
	"name": "coco.project.audit.list",
	"arguments": {
		"project_id": "proj_123",
		"type": "task.created",
		"since": "2026-02-10T00:00:00Z",
		"limit": 25,
		"offset": 0
	}
}
```

List project policies with filters, pagination, and sorting:

```json
{
	"name": "coco.policy.list",
	"arguments": {
		"project_id": "proj_123",
		"enabled": true,
		"limit": 50,
		"offset": 0,
		"sort": "created_at:desc"
	}
}
```

Fetch a project policy:

```json
{
	"name": "coco.policy.get",
	"arguments": {
		"project_id": "proj_123",
		"policy_id": "pol_456"
	}
}
```

Create a project policy:

```json
{
	"name": "coco.policy.create",
	"arguments": {
		"project_id": "proj_123",
		"name": "Default Policy",
		"description": "Enforce task audit",
		"rules": [
			{
				"type": "automation.block",
				"reason": "No automated followups"
			}
		],
		"enabled": true
	}
}
```

Update a project policy:

```json
{
	"name": "coco.policy.update",
	"arguments": {
		"project_id": "proj_123",
		"policy_id": "pol_456",
		"name": "Updated Policy",
		"description": "Updated policy description",
		"rules": [
			{
				"type": "automation.block",
				"reason": "No automated followups"
			}
		],
		"enabled": false
	}
}
```

Delete a project policy:

```json
{
	"name": "coco.policy.delete",
	"arguments": {
		"project_id": "proj_123",
		"policy_id": "pol_456"
	}
}
```

Replay project events since an event id:

```json
{
	"name": "coco.project.events.replay",
	"arguments": {
		"project_id": "proj_123",
		"since_id": "evt_456",
		"limit": 100
	}
}
```

Create a context pack (optional fields: `query`, `budget.max_files`, `budget.max_bytes`, `budget.max_snippets`):

```json
{
	"name": "coco.context_pack.create",
	"arguments": {
		"project_id": "proj_123",
		"task_id": 384,
		"query": "memory and context packs endpoints",
		"budget": {
			"max_files": 20,
			"max_bytes": 250000,
			"max_snippets": 80
		}
	}
}
```

List recent tasks without filters:

```json
{
	"name": "coco.task.list",
	"arguments": {
		"limit": 50
	}
}
```

Complete a task with status and structured result:

```json
{
	"name": "coco.task.complete",
	"arguments": {
		"task_id": 123,
		"status": "SUCCEEDED",
		"result": {
			"summary": "Completed the migration step",
			"changes_made": ["Updated schema for v2"],
			"files_touched": ["migrations/0018_policies.sql"],
			"tests_run": ["go test ./..."],
			"risks": ["None"]
		}
	}
}
```

List events with filters:

```json
{
	"name": "coco.events.list",
	"arguments": {
		"type": "task.completed",
		"since": "2026-02-10T00:00:00Z",
		"project_id": "proj_123",
		"limit": 50,
		"offset": 0
	}
}
```

Fetch run details:

```json
{
	"name": "coco.run.get",
	"arguments": {
		"run_id": "run_abc123"
	}
}
```

Append a run step:

```json
{
	"name": "coco.run.steps",
	"arguments": {
		"run_id": "run_abc123",
		"name": "Run unit tests",
		"status": "SUCCEEDED",
		"details": {
			"command": "go test ./..."
		}
	}
}
```

Append a run log entry:

```json
{
	"name": "coco.run.logs",
	"arguments": {
		"run_id": "run_abc123",
		"stream": "info",
		"chunk": "Tests started",
		"ts": "2026-02-11T10:36:10Z"
	}
}
```

Append a run artifact:

```json
{
	"name": "coco.run.artifacts",
	"arguments": {
		"run_id": "run_abc123",
		"name": "coverage-report",
		"kind": "report",
		"uri": "s3://ci-artifacts/run_abc123/coverage.html",
		"metadata": {
			"size_bytes": 48213,
			"sha256": "f2cce8f6a4b6f30f90af4f88e45b5fc81dc05b0d2c3e7c53f9c4d1e3c7e8b9aa"
		}
	}
}
```

## Testing

### With Claude Desktop

Add the server to your Claude Desktop MCP config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "cocopilot": {
      "command": "node",
      "args": ["/absolute/path/to/cocopilot/tools/cocopilot-mcp/dist/index.js"],
      "env": {
        "COCO_API_BASE": "http://localhost:8080",
        "COCO_API_KEY": "your-api-key"
      }
    }
  }
}
```

### MCP ↔ Cocopilot Communication

1. Start the Go server: `go run .`
2. Start the MCP server: `npm run dev` (or `node dist/index.js`)
3. Use MCP tools (e.g. `coco.task.list`, `coco.project.list`) and verify they return data from the running Cocopilot server.

## Version Bumping

Use npm's built-in version commands:

```bash
npm version patch   # 0.1.0 → 0.1.1
npm version minor   # 0.1.0 → 0.2.0
npm version major   # 0.1.0 → 1.0.0
```

This updates `package.json` and creates a git tag. Push the tag to trigger the CI publish workflow.
