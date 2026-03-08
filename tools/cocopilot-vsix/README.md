# Cocopilot VSIX

Minimal VS Code extension scaffold with Cocopilot commands, including `Cocopilot: Configure MCP`, `Cocopilot: Open Health`, `Cocopilot: Open Version`, `Cocopilot: Open Project Tree`, `Cocopilot: Open Project Audit`, `Cocopilot: Open Project Changes`, `Cocopilot: Open Project Memory`, `Cocopilot: Open Project Events Replay`, `Cocopilot: Open Policies`, `Cocopilot: Open API Roadmap`, `Cocopilot: Open OpenAPI Spec`, `Cocopilot: Open State Architecture`, `Cocopilot: List Policies`, `Cocopilot: Toggle Policy`, and `Cocopilot: Delete Policy`.

## Installation

### From VSIX file

```bash
code --install-extension cocopilot-vsix-0.0.1.vsix
```

Or in VS Code: Extensions view → `...` menu → **Install from VSIX...** → select the `.vsix` file.

### From source

```bash
cd tools/cocopilot-vsix
npm install
npm run compile
npx @vscode/vsce package -o dist/cocopilot-vsix.vsix
code --install-extension dist/cocopilot-vsix.vsix
```

## Configuration Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `cocopilot.apiBase` | `http://localhost:8080` | Cocopilot server URL |
| `cocopilot.apiKey` | _(empty)_ | Optional API key for authenticated requests |
| `cocopilot.autoStartMcpServer` | `false` | Auto-start the MCP server when the extension activates |
| `cocopilot.projectId` | _(empty)_ | Default project ID for project-scoped commands |

## Feature Overview

- **MCP Server Management**: Start/stop the MCP server from VS Code, status bar indicator, auto-start option
- **Task Commands**: Create, claim, save, update, delete, and list tasks from the Command Palette
- **Policy Management**: Create, update, toggle, delete, and list policies
- **Dashboard Access**: Open the Kanban board, health, config, events, agents, and other endpoints in your browser
- **Documentation Access**: Quick access to API docs, roadmap, migrations, state docs, and KB overview

## Build

- Install dependencies: `npm install`
- Install vsce: `npm install -g @vscode/vsce`
- Compile once: `npm run compile`
- Watch mode: `npm run watch`
- Package a VSIX: `npm run package`
- Release helper (compile + package): `npm run release:vsix`
- Output artifact: `dist/cocopilot-vsix.vsix`

## Release Checklist

- Confirm prerequisites: Node.js + npm are installed (`node -v`, `npm -v`). If missing, install the current LTS from nodejs.org or via your Node version manager.
- Confirm `vsce` is installed (`vsce --version`). If missing, run `npm install -g @vscode/vsce`.
- Confirm version bumps are ready (extension `package.json`, any referenced docs).
- Run `npm run release:vsix` (or `npm run package`) and verify `dist/cocopilot-vsix.vsix` exists.
- Publish with `vsce publish` (or `vsce publish --packagePath dist/cocopilot-vsix.vsix`).
- MCP bundle is current and included in the VSIX (updated `tools/cocopilot-mcp` build artifacts and `tools.json`).
- Smoke test `Cocopilot: Open Health` and `Cocopilot: Configure MCP` in the Extension Development Host.

## Run (Extension Development Host)

- Open this folder in VS Code: `tools/cocopilot-vsix`
- Select the Run and Debug view
- Pick `Run Extension` if prompted
- Press `F5` to launch the Extension Development Host
- In the new window, run `Cocopilot: Hello`
- Run `Cocopilot: Open Dashboard` to open the web UI in your browser (includes `project_id` when configured)
- Run `Cocopilot: Open Tasks Board` to open the Kanban tasks board in your browser (includes `project_id` when configured)
- Run `Cocopilot: Open Health` to open `/api/v2/health` in your browser
- Run `Cocopilot: Open Config` to open `/api/v2/config` in your browser
- Run `Cocopilot: Open Version` to open `/api/v2/version` in your browser
- Run `Cocopilot: Open Events` to open `/api/v2/events` in your browser (includes `project_id` when configured)
- Run `Cocopilot: Open Events Stream` to open `/api/v2/events/stream` in your browser (optional `project_id` and `type`)
- Run `Cocopilot: Open Agents` to open `/api/v2/agents` in your browser
- Run `Cocopilot: Open Leases` to open `/api/v2/leases` in your browser
- Run `Cocopilot: Open Tasks API` to open `/api/v2/tasks` in your browser (includes `project_id` when configured)
- Run `Cocopilot: Open Task Detail` to open `/api/v2/tasks/{id}` in your browser
- Run `Cocopilot: Open Run Detail` to open `/api/v2/runs/{id}` in your browser
- Run `Cocopilot: Open Project Detail` to open `/api/v2/projects/{id}` in your browser
- Run `Cocopilot: Open Project Tree` to open `/api/v2/projects/{id}/tree` in your browser
- Run `Cocopilot: Open Project Audit` to open `/api/v2/projects/{id}/audit` in your browser
- Run `Cocopilot: Open Project Changes` to open `/api/v2/projects/{id}/changes` in your browser (optionally includes `since`)
- Run `Cocopilot: Open Project Memory` to open `/api/v2/projects/{id}/memory` in your browser (optionally includes `scope`, `key`, and `q`)
- Run `Cocopilot: Open Project Events Replay` to open `/api/v2/projects/{id}/events/replay` in your browser (optionally includes `since_id` and `limit`)
- Run `Cocopilot: Open Policies` to open `/api/v2/projects/{id}/policies` in your browser
- Run `Cocopilot: Open Policy Detail` to open `/api/v2/projects/{id}/policies/{policyId}` in your browser
- Run `Cocopilot: List Policies` to fetch `/api/v2/projects/{id}/policies` and show a QuickPick
- Run `Cocopilot: Create Policy` to prompt for policy details and create one via `/api/v2/projects/{id}/policies`
- Run `Cocopilot: Update Policy` to prompt for policy details and update one via `/api/v2/projects/{id}/policies/{policyId}`
- Run `Cocopilot: Toggle Policy` to prompt for policy id and enabled flag, then patch `/api/v2/projects/{id}/policies/{policyId}`
- Run `Cocopilot: Delete Policy` to prompt for policy id and delete one via `/api/v2/projects/{id}/policies/{policyId}`
- Run `Cocopilot: Create Task` to prompt for instructions and create a task via `/create`
- Run `Cocopilot: Save Task` to prompt for task id and message, then post to `/save`
- Run `Cocopilot: Update Status` to prompt for task id and status, then post to `/update-status`
- Run `Cocopilot: Update Task` to prompt for task id and JSON payload, then patch `/api/v2/tasks/{taskId}`
- Run `Cocopilot: Delete Task` to prompt for task id, then delete via `/api/v2/tasks/{taskId}`
- Run `Cocopilot: List Tasks` to prompt for optional status/project filters and list tasks via `GET /api/v2/tasks`
- Run `Cocopilot: Claim Task` to claim the next task via `GET /task` and display instructions
- Run `Cocopilot: Quick Start` to pick a guided MCP action
- Run `Cocopilot: Open MCP Config` to open `.vscode/mcp.json` (creates it if missing)
- Run `Cocopilot: Configure MCP` to create and save `.vscode/mcp.json` with guided prompts
- Use `Cocopilot: Start MCP Server` to launch the MCP server terminal
- Use `Cocopilot: Stop MCP Server` to stop the MCP server terminal (no-op if none is running)
- Use the `MCP: Running/Stopped` status bar item to start or stop the MCP server
- Run `Cocopilot: Open API Docs` to open `docs/api/README.md` in the editor
- Run `Cocopilot: Open OpenAPI Spec` to open `docs/api/openapi-v2.yaml` in the editor
- Run `Cocopilot: Open Migrations Docs` to open `MIGRATIONS.md` in the editor
- Run `Cocopilot: Open Roadmap` to open `ROADMAP.md` in the editor
- Run `Cocopilot: Open KB Overview` to open `docs/ai/kb/00-overview.md` in the editor
- Run `Cocopilot: Open MCP README` to open `tools/cocopilot-mcp/README.md` in the editor
- Run `Cocopilot: Open MCP Tools` to open `tools/cocopilot-mcp/tools.json` in the editor
- Run `Cocopilot: Open API Summary` to open `docs/api/v2-summary.md` in the editor
- Run `Cocopilot: Open API Roadmap` to open `docs/api/v2-roadmap.md` in the editor
- Run `Cocopilot: Open API Compatibility` to open `docs/api/v2-compatibility.md` in the editor
- Run `Cocopilot: Open API Design` to open `docs/api/v2-design.md` in the editor
- Run `Cocopilot: Show Logs` to reveal the `Cocopilot` output channel
- Run `Cocopilot: Set Project ID` to update `cocopilot.projectId`
- Run `Cocopilot: Set API Base` to update `cocopilot.apiBase`
- Run `Cocopilot: Open State Docs` to open `docs/state/overview.md` in the editor
- Run `Cocopilot: Open State Architecture` to open `docs/state/architecture.md` in the editor
- Run `Cocopilot: Open State Current` to open `docs/state/current.md` in the editor
- Run `Cocopilot: Open State Next` to open `docs/state/next.md` in the editor
- Run `Cocopilot: Open State Decisions` to open `docs/state/decisions.md` in the editor
- Run `Cocopilot: Open State Risks` to open `docs/state/risks.md` in the editor
- Run `Cocopilot: Open Current State` to open `docs/state/current.md` in the editor
- Run `Cocopilot: Open Next State` to open `docs/state/next.md` in the editor

## Commands

- `Cocopilot: Hello` - Verify the extension is installed and responsive.
- `Cocopilot: Open Dashboard` - Open the Cocopilot web UI in your browser (adds `project_id` when configured).
- `Cocopilot: Open Tasks Board` - Open the Cocopilot tasks board UI in your browser (adds `project_id` when configured).
- `Cocopilot: Open Health` - Open the Cocopilot health endpoint (`/api/v2/health`) in your browser.
- `Cocopilot: Open Config` - Open the Cocopilot config endpoint (`/api/v2/config`) in your browser.
- `Cocopilot: Open Version` - Open the Cocopilot version endpoint (`/api/v2/version`) in your browser.
- `Cocopilot: Open Events` - Open the Cocopilot events endpoint (`/api/v2/events`) in your browser (adds `project_id` when configured).
- `Cocopilot: Open Events Stream` - Open the Cocopilot events stream endpoint (`/api/v2/events/stream`) in your browser with optional `project_id` and `type` parameters.
- `Cocopilot: Open Agents` - Open the Cocopilot agents endpoint (`/api/v2/agents`) in your browser.
- `Cocopilot: Open Leases` - Open the Cocopilot leases endpoint (`/api/v2/leases`) in your browser.
- `Cocopilot: Open Tasks API` - Open the Cocopilot tasks list endpoint (`/api/v2/tasks`) in your browser (adds `project_id` when configured).
- `Cocopilot: Open Task Detail` - Prompt for a task id and open `/api/v2/tasks/{id}` in your browser.
- `Cocopilot: Open Run Detail` - Prompt for a run id and open `/api/v2/runs/{id}` in your browser.
- `Cocopilot: Open Project Detail` - Prompt for a project id and open `/api/v2/projects/{id}` in your browser.
- `Cocopilot: Open Project Tree` - Prompt for a project id and open `/api/v2/projects/{id}/tree` in your browser.
- `Cocopilot: Open Project Audit` - Prompt for a project id and open `/api/v2/projects/{id}/audit` in your browser.
- `Cocopilot: Open Project Changes` - Prompt for a project id and optional since parameter, then open `/api/v2/projects/{id}/changes` in your browser.
- `Cocopilot: Open Project Memory` - Prompt for a project id and optional `scope`, `key`, and `q` parameters, then open `/api/v2/projects/{id}/memory` in your browser.
- `Cocopilot: Open Project Events Replay` - Prompt for a project id (always prompts; does not use `cocopilot.projectId`) and optional `since_id` and `limit`, then open `/api/v2/projects/{id}/events/replay` in your browser.
- `Cocopilot: Open Policies` - Open `/api/v2/projects/{id}/policies` in your browser (uses `cocopilot.projectId` or prompts).
- `Cocopilot: Open Policy Detail` - Prompt for a policy id and open `/api/v2/projects/{id}/policies/{policyId}` in your browser.
- `Cocopilot: List Policies` - Fetch `/api/v2/projects/{id}/policies` and show the results in a QuickPick.
- `Cocopilot: Create Policy` - Prompt for a policy name, description, and rules JSON, then create it via `/api/v2/projects/{id}/policies`.
- `Cocopilot: Update Policy` - Prompt for policy id, name, description, rules JSON, and enabled flag, then update it via `/api/v2/projects/{id}/policies/{policyId}`.
- `Cocopilot: Toggle Policy` - Prompt for policy id and enabled flag, then patch it via `/api/v2/projects/{id}/policies/{policyId}`.
- `Cocopilot: Delete Policy` - Prompt for policy id, then delete it via `/api/v2/projects/{id}/policies/{policyId}`.
- `Cocopilot: Create Task` - Prompt for instructions and create a task via the `/create` endpoint.
- `Cocopilot: Save Task` - Prompt for task id and message, then save the update via the `/save` endpoint.
- `Cocopilot: Update Status` - Prompt for task id and status, then update via the `/update-status` endpoint.
- `Cocopilot: Update Task` - Prompt for task id and JSON payload, then patch the `/api/v2/tasks/{taskId}` endpoint.
- `Cocopilot: Delete Task` - Prompt for task id, then delete the task via the `/api/v2/tasks/{taskId}` endpoint.
- `Cocopilot: List Tasks` - Prompt for optional status/project filters and list tasks via `GET /api/v2/tasks`.
- `Cocopilot: Claim Task` - Claim the next task via the `/task` endpoint and show instructions.
- `Cocopilot: Open MCP Config` - Open `.vscode/mcp.json` and create it if missing.
- `Cocopilot: Configure MCP` - Create or update `.vscode/mcp.json` with guided prompts.
- `Cocopilot: Start MCP Server` - Launch the MCP server in a VS Code terminal.
- `Cocopilot: Stop MCP Server` - Stop the MCP server terminal if it is running.
- `Cocopilot: Open Settings` - Open Cocopilot settings for this workspace.
- `Cocopilot: Quick Start` - Open a guided quick start picker that walks you through MCP setup (config, start server) or opens the dashboard.
- `Cocopilot: Set Project ID` - Command id `cocopilot.setProjectId`; prompts for a project id and saves it to `cocopilot.projectId`.
- `Cocopilot: Set API Base` - Command id `cocopilot.setApiBase`; prompts for an API base URL and saves it to `cocopilot.apiBase`.
- `Cocopilot: Open API Docs` - Open `docs/api/README.md` in the editor.
- `Cocopilot: Open OpenAPI Spec` - Open `docs/api/openapi-v2.yaml` in the editor.
- `Cocopilot: Open API Summary` - Open `docs/api/v2-summary.md` in the editor.
- `Cocopilot: Open API Roadmap` - Open `docs/api/v2-roadmap.md` in the editor.
- `Cocopilot: Open API Compatibility` - Open `docs/api/v2-compatibility.md` in the editor.
- `Cocopilot: Open API Design` - Open `docs/api/v2-design.md` in the editor.
- `Cocopilot: Open Migrations Docs` - Open `MIGRATIONS.md` in the editor.
- `Cocopilot: Open Roadmap` - Open `ROADMAP.md` in the editor.
- `Cocopilot: Open KB Overview` - Open `docs/ai/kb/00-overview.md` in the editor.
- `Cocopilot: Open State Docs` - Open `docs/state/overview.md` in the editor.
- `Cocopilot: Open State Architecture` - Open `docs/state/architecture.md` in the editor.
- `Cocopilot: Open State Current` - Open `docs/state/current.md` in the editor.
- `Cocopilot: Open State Next` - Open `docs/state/next.md` in the editor.
- `Cocopilot: Open State Decisions` - Open `docs/state/decisions.md` in the editor.
- `Cocopilot: Open State Risks` - Open `docs/state/risks.md` in the editor.
- `Cocopilot: Open Current State` - Open `docs/state/current.md` in the editor.
- `Cocopilot: Open Next State` - Open `docs/state/next.md` in the editor.
- `Cocopilot: Open MCP README` - Command id `cocopilot.openMcpReadme`; opens `tools/cocopilot-mcp/README.md` in the editor.
- `Cocopilot: Open MCP Tools` - Command id `cocopilot.openMcpTools`; opens `tools/cocopilot-mcp/tools.json` in the editor.
- `Cocopilot: Show Logs` - Command id `cocopilot.showLogs`; reveals the `Cocopilot` output channel.

## Cocopilot: Create Task

Use `Cocopilot: Create Task` to create a task from the Command Palette.

- Prompts for task instructions (required; blank input is rejected).
- Sends a POST to `/create` using `cocopilot.apiBase` with `application/x-www-form-urlencoded` payload.
- Includes `instructions` and, when set, `project_id` from `cocopilot.projectId`.
- On success, shows the created `task_id` in a notification and logs it to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

## Cocopilot: Claim Task

Use `Cocopilot: Claim Task` to claim the next task from the Command Palette.

- Sends a GET to `/task` using `cocopilot.apiBase`.
- On success, shows the instructions in a notification or a new untitled document.
- The v1 `/task` response is plain text. When a task is available it includes the task id, status, and updated timestamp, followed by the instructions block.
- When no tasks are available, the response is a short plain-text message telling the agent to wait.

## Cocopilot: Open Config

Use `Cocopilot: Open Config` to open the config endpoint in your browser.

- Opens `/api/v2/config` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: Open Migrations Docs

Use `Cocopilot: Open Migrations Docs` to open the migrations reference in the editor.

- Opens `MIGRATIONS.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open API Summary

Use `Cocopilot: Open API Summary` to open the API summary in the editor.

- Opens `docs/api/v2-summary.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open OpenAPI Spec

Use `Cocopilot: Open OpenAPI Spec` to open the OpenAPI spec in the editor.

- Opens `docs/api/openapi-v2.yaml` in the workspace.
- Does not prompt for input.

## Cocopilot: Open API Roadmap

Use `Cocopilot: Open API Roadmap` to open the API roadmap in the editor.

- Opens `docs/api/v2-roadmap.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open API Compatibility

Use `Cocopilot: Open API Compatibility` to open the API compatibility guide in the editor.

- Opens `docs/api/v2-compatibility.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open API Design

Use `Cocopilot: Open API Design` to open the API design doc in the editor.

- Opens `docs/api/v2-design.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open KB Overview

Use `Cocopilot: Open KB Overview` to open the AI knowledge base overview in the editor.

- Opens `docs/ai/kb/00-overview.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open Roadmap

Use `Cocopilot: Open Roadmap` to open the product roadmap in the editor.

- Opens `ROADMAP.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open State Architecture

Use `Cocopilot: Open State Architecture` to open the state architecture doc in the editor.

- Opens `docs/state/architecture.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open Current State

Use `Cocopilot: Open Current State` to open the current state doc in the editor.

- Opens `docs/state/current.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open Next State

Use `Cocopilot: Open Next State` to open the next state doc in the editor.

- Opens `docs/state/next.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open State Decisions

Use `Cocopilot: Open State Decisions` to open the state decisions doc in the editor.

- Opens `docs/state/decisions.md` in the workspace.
- Does not prompt for input.

## Cocopilot: Open Task Detail

Use `Cocopilot: Open Task Detail` to open a task detail page in your browser.

- Prompts for a task id (required; blank input is rejected).
- Opens `/api/v2/tasks/{id}` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: Open Run Detail

Use `Cocopilot: Open Run Detail` to open a run detail page in your browser.

- Prompts for a run id (required; blank input is rejected).
- Opens `/api/v2/runs/{id}` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: Open Project Detail

Use `Cocopilot: Open Project Detail` to open a project detail page in your browser.

- Prompts for a project id (required; blank input is rejected).
- Opens `/api/v2/projects/{id}` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: Open Project Tree

Use `Cocopilot: Open Project Tree` to open a project tree snapshot in your browser.

- Prompts for a project id (required; blank input is rejected).
- Opens `/api/v2/projects/{id}/tree` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: Open Project Audit

Use `Cocopilot: Open Project Audit` to open a project audit page in your browser.

- Prompts for a project id (required; blank input is rejected).
- Opens `/api/v2/projects/{id}/audit` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: Open Project Changes

Use `Cocopilot: Open Project Changes` to open a project changes page in your browser.

- Prompts for a project id (required; blank input is rejected).
- Prompts for an optional `since` parameter to include as a query string.
- `since` is passed through as typed (for example `2024-01-01T00:00:00Z`). Leave it blank to omit the query string.
- Opens `/api/v2/projects/{id}/changes` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

Example URL with `since`:

```
/api/v2/projects/proj_123/changes?since=2024-01-01T00:00:00Z
```

## Cocopilot: Open Project Memory

Use `Cocopilot: Open Project Memory` to open project memory in your browser.

- Prompts for a project id (required; blank input is rejected).
- Prompts for optional `scope`, `key`, and `q` parameters to include as query strings.
- `scope`, `key`, and `q` are passed through as typed. Leave them blank to omit the query string.
- Opens `/api/v2/projects/{id}/memory` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

Example URL with query parameters:

```
/api/v2/projects/proj_123/memory?scope=tasks&key=summary&q=retry
```

## Cocopilot: Open Project Events Replay

Use `Cocopilot: Open Project Events Replay` to open a project events replay in your browser.

- Prompts for a project id (required; blank input is rejected) and always asks, even when `cocopilot.projectId` is set.
- Prompts for optional `since_id` and `limit` parameters to include as query strings.
- `since_id` and `limit` are passed through as typed. Leave them blank to omit the query string.
- Opens `/api/v2/projects/{id}/events/replay` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

Example URL with query parameters:

```
/api/v2/projects/proj_123/events/replay?since_id=evt_123&limit=100
```

## Cocopilot: Open Policies

Use `Cocopilot: Open Policies` to open a project policies endpoint in your browser.

- Uses `cocopilot.projectId` when set, otherwise prompts for a project id (required; blank input is rejected).
- Opens `/api/v2/projects/{id}/policies` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: Open Policy Detail

Use `Cocopilot: Open Policy Detail` to open a policy detail endpoint in your browser.

- Uses `cocopilot.projectId` when set, otherwise prompts for a project id (required; blank input is rejected).
- Prompts for a policy id (required; blank input is rejected).
- Opens `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase`.
- When `cocopilot.apiBase` is set to the default UI origin (for example `http://localhost:8080`), the request resolves against that host.

## Cocopilot: List Policies

Use `Cocopilot: List Policies` to fetch project policies and show them in a QuickPick.

- Uses `cocopilot.projectId` when set, otherwise prompts for a project id (required; blank input is rejected).
- Fetches `/api/v2/projects/{id}/policies` using `cocopilot.apiBase`.
- Shows policy items in a QuickPick, using policy fields when available.
- QuickPick labels prefer `name`, `title`, `rule`, or `type` fields, falling back to the policy id or an index-based label.
- QuickPick descriptions show `status`, `mode`, `enforced`, or `enabled` values when present.
- If the response contains no policies, an informational message is shown and the raw response is logged.

## Cocopilot: Create Policy

Use `Cocopilot: Create Policy` to create a new policy from the Command Palette.

- Uses `cocopilot.projectId` when set, otherwise prompts for a project id (required; blank input is rejected).
- Prompts for a policy name, description, and rules JSON (all required).
- Sends a POST to `/api/v2/projects/{id}/policies` using `cocopilot.apiBase` with `application/json` payload.
- On success, shows the policy id (when returned) and logs the response to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example payload (JSON):

```json
{
	"name": "Task SLA",
	"description": "Require tasks to include an ETA and owner.",
	"rules": {
		"require_fields": ["eta", "owner"],
		"scope": "tasks"
	}
}
```

## Cocopilot: Update Policy

Use `Cocopilot: Update Policy` to update an existing policy from the Command Palette.

- Uses `cocopilot.projectId` when set, otherwise prompts for a project id (required; blank input is rejected).
- Prompts for policy id, name, description, rules JSON, and enabled flag (all required).
- Sends a PATCH to `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase` with `application/json` payload.
- On success, shows the policy id and logs the response to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example payload (JSON):

```json
{
	"name": "Task SLA",
	"description": "Require tasks to include an ETA and owner.",
	"rules": {
		"require_fields": ["eta", "owner"],
		"scope": "tasks"
	},
	"enabled": true
}
```

## Cocopilot: Toggle Policy

Use `Cocopilot: Toggle Policy` to update only the enabled flag on an existing policy from the Command Palette.

- Uses `cocopilot.projectId` when set, otherwise prompts for a project id (required; blank input is rejected).
- Prompts for policy id and enabled flag (both required).
- Sends a PATCH to `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase` with `application/json` payload.
- On success, shows the policy id and logs the response to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example payload (JSON):

```json
{
	"enabled": false
}
```

## Cocopilot: Delete Policy

Use `Cocopilot: Delete Policy` to delete an existing policy from the Command Palette.

- Uses `cocopilot.projectId` when set, otherwise prompts for a project id (required; blank input is rejected).
- Prompts for policy id (required; blank input is rejected).
- Sends a DELETE to `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase`.
- On success, shows the response text (if any) and logs it to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example request (path parameters):

```
DELETE /api/v2/projects/proj_123/policies/policy_123
```

## Cocopilot: Open Events

Use `Cocopilot: Open Events` to open the events endpoint in your browser.

- Opens `/api/v2/events` using `cocopilot.apiBase`.
- Includes `project_id` from `cocopilot.projectId` when configured.

## Cocopilot: Open Events Stream

Use `Cocopilot: Open Events Stream` to open the server-sent events endpoint in your browser.

- Opens `/api/v2/events/stream` using `cocopilot.apiBase`.
- Prompts for optional `project_id` and `type` filters and includes them as query parameters when provided.

## Cocopilot: Open Agents

Use `Cocopilot: Open Agents` to open the agents endpoint in your browser.

- Opens `/api/v2/agents` using `cocopilot.apiBase`.

## Cocopilot: Open Leases

Use `Cocopilot: Open Leases` to open the leases endpoint in your browser.

- Opens `/api/v2/leases` using `cocopilot.apiBase`.

## Cocopilot: Open Tasks API

Use `Cocopilot: Open Tasks API` to open the tasks list endpoint in your browser.

- Opens `/api/v2/tasks` using `cocopilot.apiBase`.
- Includes `project_id` from `cocopilot.projectId` when configured.

## Cocopilot: Save Task

Use `Cocopilot: Save Task` to post progress updates from the Command Palette.

- Prompts for `task_id` and a message (required; blank input is rejected).
- Sends a POST to `/save` using `cocopilot.apiBase` with `application/x-www-form-urlencoded` payload.
- On success, shows the response text (if any) and logs it to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example payload (form-encoded):

```
task_id=123
message=Drafted API notes and updated docs.
```

## Cocopilot: Update Status

Use `Cocopilot: Update Status` to update a task status from the Command Palette.

- Prompts for `task_id` and a status (required; blank input is rejected).
- Accepts v2 status values: `QUEUED`, `CLAIMED`, `RUNNING`, `SUCCEEDED`, `FAILED`, `NEEDS_REVIEW`, `CANCELLED`.
- Sends a POST to `/update-status` using `cocopilot.apiBase` with `application/x-www-form-urlencoded` payload.
- On success, shows the response text (if any) and logs it to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example payload (form-encoded):

```
task_id=123
status=RUNNING
```

## Cocopilot: Update Task

Use `Cocopilot: Update Task` to update task fields from the Command Palette.

- Prompts for `task_id` and a JSON payload (required; blank input is rejected).
- Payload must be a JSON object with one or more fields.
- Accepts `title`, `instructions`, `status`, `type`, and `tags` fields.
- Omitted fields are left unchanged.
- `status` uses the same v2 status values as `Cocopilot: Update Status`.
- `tags` must be an array of strings.
- Sends a PATCH to `/api/v2/tasks/{taskId}` using `cocopilot.apiBase` with `application/json` payload.
- On success, shows the response text (if any) and logs it to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example payload (JSON):

```json
{
	"title": "Refine task plan",
	"instructions": "Revise the steps to include new requirements.",
	"status": "RUNNING",
	"type": "PLANNING",
	"tags": ["ops", "plan"]
}
```

Minimal payload (JSON):

```json
{
	"status": "RUNNING"
}
```

## Cocopilot: Delete Task

Use `Cocopilot: Delete Task` to delete a task from the Command Palette.

- Prompts for `task_id` (required; blank input is rejected).
- Sends a DELETE to `/api/v2/tasks/{taskId}` using `cocopilot.apiBase`.
- On success, shows the response text (if any) and logs it to the `Cocopilot` output channel.
- On failure, shows an error and logs the response body to the output channel.

Example request (form-encoded path parameter):

```
DELETE /api/v2/tasks/123
```

## Cocopilot: List Tasks

Use `Cocopilot: List Tasks` to browse tasks from the Command Palette.

- Prompts for optional filters and then calls `GET /api/v2/tasks` using `cocopilot.apiBase`.
- `status` filter is free-form text (commonly a v2 status like `RUNNING`, `QUEUED`, `SUCCEEDED`).
- `project_id` filter defaults to `cocopilot.projectId` when set and can be cleared to omit the filter.
- Blank input for either prompt omits that filter from the request.
- On success, shows tasks in a Quick Pick list (title/instructions as label, status as description).
- On failure, shows an error and logs the response body to the `Cocopilot` output channel.

Example requests:

```
GET /api/v2/tasks
GET /api/v2/tasks?status=RUNNING
GET /api/v2/tasks?project_id=proj_123
GET /api/v2/tasks?status=RUNNING&project_id=proj_123
```

## Logging

- Run `Cocopilot: Show Logs` to open the `Cocopilot` output channel for extension activity, MCP server start/stop, and errors.

## Notes

- Activation is triggered by running the commands.
- The dashboard URL uses `cocopilot.apiBase` and defaults to `http://localhost:8080`.
- The tasks board URL uses `cocopilot.apiBase` and defaults to `http://localhost:8080`.
- The tasks board command opens `/tasks` and appends `?project_id=...` when `cocopilot.projectId` is set.
- The health URL uses `cocopilot.apiBase` and opens `/api/v2/health`.
- The version URL uses `cocopilot.apiBase` and opens `/api/v2/version`.
- The events URL uses `cocopilot.apiBase` and opens `/api/v2/events`, adding `project_id` when configured.
- The agents URL uses `cocopilot.apiBase` and opens `/api/v2/agents`.
- The run detail command opens `/api/v2/runs/{id}` using `cocopilot.apiBase`.
- The project detail command opens `/api/v2/projects/{id}` using `cocopilot.apiBase`.
- The project audit command opens `/api/v2/projects/{id}/audit` using `cocopilot.apiBase`.
- The policies command opens `/api/v2/projects/{id}/policies` using `cocopilot.apiBase`.
- The policy detail command opens `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase` and prompts for the policy id.
- The list policies command fetches `/api/v2/projects/{id}/policies` and shows a QuickPick.
- The create policy command posts to `/api/v2/projects/{id}/policies` using `cocopilot.apiBase` and requires name, description, and rules JSON.
- The update policy command patches `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase` and requires policy id, name, description, rules JSON, and enabled flag.
- The toggle policy command patches `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase` and requires policy id plus an enabled flag.
- The delete policy command deletes `/api/v2/projects/{id}/policies/{policyId}` using `cocopilot.apiBase` and requires policy id.
- The create task command posts to `/create` using `cocopilot.apiBase`, and includes `project_id` when `cocopilot.projectId` is set.
- The save task command posts to `/save` using `cocopilot.apiBase` and requires `task_id` plus a message.
- The update status command posts to `/update-status` using `cocopilot.apiBase` and requires `task_id` plus a status.
- The update task command patches `/api/v2/tasks/{taskId}` using `cocopilot.apiBase` and requires `task_id` plus a JSON payload.
- The claim task command requests `/task` using `cocopilot.apiBase` and shows instructions in a notification or a new untitled file.
- The list tasks command requests `/api/v2/tasks` using `cocopilot.apiBase` and supports optional `status` and `project_id` filters.
- When `cocopilot.projectId` is set, the dashboard URL appends `?project_id=...`.
- The `npm run package` step uses `vsce` and requires a clean build.
- The `cocopilot.setProjectId` command updates `cocopilot.projectId` in workspace settings when a folder is open; otherwise it updates user settings. Entering a blank value clears the setting.
- The `cocopilot.setApiBase` command updates `cocopilot.apiBase` in workspace settings when a folder is open; otherwise it updates user settings. Entering a blank value clears the setting.
- Set `cocopilot.autoStartMcpServer` to `true` to start the MCP server automatically when the VSIX activates. The MCP status bar item updates to show the running state.
- Use `Cocopilot: Open MCP Tools` to review or edit `tools/cocopilot-mcp/tools.json`, which defines the MCP tools exposed by the local server.
- The status bar shows the current `cocopilot.projectId`; click it to open settings.
- The `Cocopilot` output channel logs extension actions such as command execution, MCP server start/stop, and errors.
- Run `Cocopilot: Show Logs` to reveal the `Cocopilot` output channel at any time.
- The MCP status bar item shows whether the MCP server is running.
	- Text is `MCP: Running` or `MCP: Stopped`.
	- Click to toggle the server state (start when stopped, stop when running).
	- On start, it opens a VS Code terminal running the MCP server.
	- On stop, it terminates that terminal if it is active.
- You can also run `Cocopilot: Open Settings` to edit configuration.

## Testing

### Manual VSIX Testing

```bash
code --install-extension dist/cocopilot-vsix.vsix
```

Then open a workspace and run `Cocopilot: Hello` from the Command Palette to verify installation.

### MCP ↔ Cocopilot Communication

1. Start the Go server: `go run ./cmd/cocopilot`
2. Start the MCP server via `Cocopilot: Start MCP Server` or set `cocopilot.autoStartMcpServer` to `true`
3. Use MCP tools (e.g. `coco.task.list`, `coco.project.list`) and verify they return data from the running Cocopilot server.

### Extension Development Host

For development testing, press `F5` in the `tools/cocopilot-vsix` folder to launch the Extension Development Host and test commands interactively.

## Version Bumping

Use npm's built-in version commands:

```bash
npm version patch   # 0.0.1 → 0.0.2
npm version minor   # 0.0.1 → 0.1.0
npm version major   # 0.0.1 → 1.0.0
```

This updates `package.json` and creates a git tag. Push the tag to trigger the CI build workflow.
