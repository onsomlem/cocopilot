package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsomlem/cocopilot/internal/config"
)

// ---------------------------------------------------------------------------
// Type aliases for backward compatibility.
// ---------------------------------------------------------------------------

type runtimeConfig = config.RuntimeConfig
type authIdentity = config.AuthIdentity
type automationRule = config.AutomationRule
type automationAction = config.AutomationAction
type automationTaskSpec = config.AutomationTaskSpec

// ---------------------------------------------------------------------------
// Constant aliases for backward compatibility.
// ---------------------------------------------------------------------------

const (
	defaultSSEHeartbeatSeconds        = config.DefaultSSEHeartbeatSeconds
	minSSEHeartbeatSeconds            = config.MinSSEHeartbeatSeconds
	maxSSEHeartbeatSeconds            = config.MaxSSEHeartbeatSeconds
	defaultDBPath                     = config.DefaultDBPath
	defaultHTTPAddr                   = config.DefaultHTTPAddr
	defaultEventsRetentionDays        = config.DefaultEventsRetentionDays
	defaultEventsRetentionMax         = config.DefaultEventsRetentionMax
	defaultEventsPruneIntervalSeconds = config.DefaultEventsPruneIntervalSeconds
	minEventsPruneIntervalSeconds     = config.MinEventsPruneIntervalSeconds
	maxEventsPruneIntervalSeconds     = config.MaxEventsPruneIntervalSeconds
	v1EventTypeTasks                  = config.V1EventTypeTasks

	v1TasksListDefaultLimit       = config.V1TasksListDefaultLimit
	v1TasksListMaxLimit           = config.V1TasksListMaxLimit
	v1EventsReplayLimitMaxDefault = config.V1EventsReplayLimitMaxDefault
	DefaultProjectID              = config.DefaultProjectID
)

// ---------------------------------------------------------------------------
// v1 compat struct (stays in root; depends on TaskStatus from models).
// ---------------------------------------------------------------------------

type Task struct {
	ID           int        `json:"id"`
	Instructions string     `json:"instructions"`
	Status       TaskStatus `json:"status"`
	Output       *string    `json:"output"`
	ParentTaskID *int       `json:"parent_task_id"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// Function wrappers for backward compatibility.
// ---------------------------------------------------------------------------

func getEnvConfigValue(name, fallback string) (string, error) {
	return config.GetEnvConfigValue(name, fallback)
}

func getEnvBoolValue(name string, fallback bool) (bool, error) {
	return config.GetEnvBoolValue(name, fallback)
}

func parseScopeSet(raw string) (map[string]struct{}, error) {
	return config.ParseScopeSet(raw)
}

func parseAuthIdentities(raw string) ([]authIdentity, error) {
	return config.ParseAuthIdentities(raw)
}

func normalizeV1EventType(raw string) (string, bool) {
	return config.NormalizeV1EventType(raw)
}

func loadRuntimeConfig() (runtimeConfig, error) {
	return config.LoadRuntimeConfig(parseAutomationRules)
}

func resolveSSEHeartbeatInterval(cfg runtimeConfig) time.Duration {
	return config.ResolveSSEHeartbeatInterval(cfg)
}

func resolveV1EventsReplayLimitMax(cfg runtimeConfig) int {
	return config.ResolveV1EventsReplayLimitMax(cfg)
}

func resolveSSEReplayLimitMax(cfg runtimeConfig) int {
	return config.ResolveSSEReplayLimitMax(cfg)
}

func resolveEventsPruneInterval(cfg runtimeConfig) time.Duration {
	return config.ResolveEventsPruneInterval(cfg)
}

// ---------------------------------------------------------------------------
// getInstructions stays in root (depends on workdirMu/workdir from main.go).
// ---------------------------------------------------------------------------

// buildProjectsSection queries the DB for all projects and renders a section
// listing them with their IDs, names, and workdirs so the agent knows which
// project(s) to target.
func buildProjectsSection(baseURL string) string {
	if db == nil {
		return ""
	}
	projects, err := ListProjects(db)
	if err != nil || len(projects) == 0 {
		return fmt.Sprintf(`## Active Projects

No projects found. Create one first:
`+"```"+`
curl -s -X POST %s/api/v2/projects -H "Content-Type: application/json" \
  -d '{"name":"my-project","workdir":"/path/to/workdir"}'
`+"```"+`
`, baseURL)
	}

	var sb strings.Builder
	sb.WriteString("## Active Projects\n\n")
	sb.WriteString("| ID | Name | Working Directory |\n")
	sb.WriteString("|----|------|-------------------|\n")
	for _, p := range projects {
		wd := p.Workdir
		if wd == "" {
			wd = "(not set)"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", p.ID, p.Name, wd))
	}
	sb.WriteString("\n")

	// Identify the primary project (first non-default, or default if only one)
	primary := projects[0]
	for _, p := range projects {
		if p.ID != DefaultProjectID {
			primary = p
			break
		}
	}
	sb.WriteString(fmt.Sprintf("**Primary project:** `%s` (%s)\n\n", primary.ID, primary.Name))
	sb.WriteString("Use the project ID in all project-scoped API calls below.\n\n")

	return sb.String()
}

func getInstructions(baseURL string) string {
	workdirMu.RLock()
	wd := workdir
	workdirMu.RUnlock()

	projectsSection := buildProjectsSection(baseURL)

	return `# Cocopilot Agent Instructions

You are connected to a Cocopilot orchestrator — an agentic task queue that manages
projects, tasks, runs, memory, dependencies, and automation.

## Working Directory

All work MUST happen in: ` + "`" + wd + "`" + `

` + "```" + `
cd ` + wd + `
` + "```" + `

` + projectsSection + `

## Quick Reference (v2 JSON API)

Base URL: ` + baseURL + `

### Discover projects
` + "```" + `
GET /api/v2/projects
` + "```" + `

### Register agent (once)
` + "```" + `
POST /api/v2/agents  {"id":"copilot","name":"GitHub Copilot","capabilities":["analyze","modify","test"]}
` + "```" + `

### Create tasks in a project
` + "```" + `
POST /api/v2/projects/<project_id>/tasks  {"title":"...","instructions":"...","type":"MODIFY","priority":50,"tags":["feature"]}
` + "```" + `

### Claim next task from a project
` + "```" + `
POST /api/v2/projects/<project_id>/tasks/claim-next  {"agent_id":"copilot"}
` + "```" + `
Response: {task, lease, run, context}. Save run_id and lease_id.
If 404, no tasks available — poll again in 15s.

### Log progress
` + "```" + `
POST /api/v2/runs/<run_id>/steps    {"name":"step-name","status":"STARTED","details":"..."}
POST /api/v2/runs/<run_id>/logs     {"stream":"stdout","chunk":"log line"}
` + "```" + `

### Heartbeat (every 30s for long tasks)
` + "```" + `
POST /api/v2/leases/<lease_id>/heartbeat
` + "```" + `

### Complete or fail
` + "```" + `
POST /api/v2/tasks/<id>/complete  {"output":"summary","result":{"summary":"...","changes_made":[...],"files_touched":[...]}}
POST /api/v2/tasks/<id>/fail      {"error":"what failed","output":"partial output"}
` + "```" + `

### Memory (persist knowledge across tasks)
` + "```" + `
PUT /api/v2/projects/<project_id>/memory  {"scope":"agent","key":"learnings","value":{"notes":"..."},"source_refs":["task_123"]}
GET /api/v2/projects/<project_id>/memory?scope=agent
` + "```" + `

### Dependencies (task ordering)
` + "```" + `
POST /api/v2/tasks/<id>/dependencies  {"depends_on_task_id":42}
` + "```" + `
Tasks with unmet dependencies cannot be claimed.

### Context packs (assembled context for a task)
` + "```" + `
POST /api/v2/projects/<project_id>/context-packs  {"task_id":42,"summary":"...","contents":{...}}
` + "```" + `

### Policies (governance rules)
` + "```" + `
POST /api/v2/projects/<project_id>/policies  {"name":"require-tests","rules":{...},"enabled":true}
GET  /api/v2/projects/<project_id>/policies
` + "```" + `

### Automation (event-driven task creation)
` + "```" + `
GET  /api/v2/projects/<project_id>/automation/rules
POST /api/v2/projects/<project_id>/automation/simulate  {"event":{"kind":"task.completed","entity_id":"42"}}
` + "```" + `

### Events (real-time)
` + "```" + `
GET /api/v2/events?type=task.completed&limit=10
GET /api/v2/events/stream  (SSE — real-time push)
GET /api/v2/projects/<project_id>/events/stream  (project-scoped SSE)
` + "```" + `

### Runs & artifacts
` + "```" + `
GET  /api/v2/runs/<run_id>
POST /api/v2/runs/<run_id>/steps      {"name":"...","status":"STARTED","details":"..."}
POST /api/v2/runs/<run_id>/logs       {"stream":"stdout","chunk":"..."}
POST /api/v2/runs/<run_id>/artifacts  {"kind":"diff","storage_ref":"...","metadata":{...}}
` + "```" + `

### Agents & leases
` + "```" + `
GET  /api/v2/agents
POST /api/v2/agents  {"id":"my-agent","name":"My Agent","capabilities":["analyze","modify","test"]}
POST /api/v2/leases/<lease_id>/heartbeat
POST /api/v2/leases/<lease_id>/release
` + "```" + `

### Repo & files
` + "```" + `
GET  /api/v2/projects/<project_id>/files
POST /api/v2/projects/<project_id>/files/scan
POST /api/v2/projects/<project_id>/files/sync
GET  /api/v2/projects/<project_id>/tree?depth=3
GET  /api/v2/projects/<project_id>/changes?since=2025-01-01T00:00:00Z
` + "```" + `

### Artifacts (diff viewer, run outputs)
` + "```" + `
GET  /api/v2/artifacts/<artifact_id>           (metadata)
GET  /api/v2/artifacts/<artifact_id>/content   (raw content)
GET  /api/v2/artifacts/<artifact_id>/comments  (line comments)
POST /api/v2/artifacts/<artifact_id>/comments  {"line_number":10,"body":"...","author":"reviewer"}
` + "```" + `

### Planning (project planning pipeline)
` + "```" + `
GET  /api/v2/projects/<project_id>/planning
POST /api/v2/projects/<project_id>/planning/run
GET  /api/v2/projects/<project_id>/planning/cycles
GET  /api/v2/projects/<project_id>/planning/decisions
GET  /api/v2/projects/<project_id>/planning/quality
GET  /api/v2/projects/<project_id>/planning/seed-prompts
` + "```" + `

### Templates & prompts
` + "```" + `
GET  /api/v2/projects/<project_id>/templates
GET  /api/v2/projects/<project_id>/prompts
` + "```" + `

### Audit (compliance trail)
` + "```" + `
GET  /api/v2/audit                                      (global)
GET  /api/v2/projects/<project_id>/audit                 (project-scoped)
GET  /api/v2/projects/<project_id>/audit/export          (export)
` + "```" + `

### System
` + "```" + `
GET  /api/v2/health
GET  /api/v2/status
GET  /api/v2/metrics
GET  /api/v2/version
GET  /api/v2/config
POST /api/v2/backup
POST /api/v2/restore
` + "```" + `

## Task Types

ANALYZE, MODIFY, TEST, REVIEW, DOC, RELEASE, ROLLBACK, PLAN

## Task Status Lifecycle

QUEUED → CLAIMED → RUNNING → SUCCEEDED / FAILED / NEEDS_REVIEW

## Workflow

1. List projects (GET /api/v2/projects) to find your target project
2. Claim a task via claim-next using the project ID
3. Read the task instructions and assembled context
4. Execute the work in the working directory
5. Log progress via run steps and logs
6. Complete or fail the task with a summary
7. Store learnings in project memory for future tasks
8. Poll for the next task after 15 seconds

Always use ` + "`curl`" + ` for API calls.

For the full API reference (detailed examples, automation rules, template variables),
see ` + baseURL + `/instructions-detailed
`
}

// ---------------------------------------------------------------------------
// getDetailedInstructions returns a comprehensive reference for an AI agent
// to fully set up, manage, and orchestrate projects autonomously.
// ---------------------------------------------------------------------------

func getDetailedInstructions(baseURL string) string {
	workdirMu.RLock()
	wd := workdir
	workdirMu.RUnlock()

	projectsSection := buildProjectsSection(baseURL)

	body := `# Cocopilot — Detailed Agent Instructions

## What is Cocopilot?

Cocopilot is an **Agentic Task Queue** — a server that orchestrates LLM agents by
providing a Kanban-style task queue with HTTP APIs. Agents poll for work, claim tasks,
execute them, and report results. The server manages projects, tasks, runs, memory,
automation, policies, and real-time events.

**Base URL:** ` + baseURL + `
**Working Directory:** ` + "`" + wd + "`" + `

---

` + projectsSection + `

---

## Quick Start (Full Autonomous Setup)

### Step 0: Discover Projects

` + "```bash" + `
curl -s ` + baseURL + `/api/v2/projects | jq .
` + "```" + `

Use the project ID from the response in all project-scoped API calls.

### Step 1: Create a Project (if needed)

` + "```bash" + `
curl -s -X POST http://127.0.0.1:8080/api/v2/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-project",
    "workdir": "` + wd + `"
  }' | jq .
` + "```" + `

Response: ` + "`" + `{"project": {"id": "proj_xxx", "name": "my-project", ...}}` + "`" + `

Save the project ID for all subsequent calls.

### Step 2: Create Tasks

` + "```bash" + `
# Create a task in the project
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/PROJECT_ID/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Analyze codebase",
    "instructions": "Review the project and summarize architecture",
    "type": "ANALYZE",
    "priority": 50,
    "tags": ["onboarding"]
  }' | jq .
` + "```" + `

### Step 3: Claim and Execute Tasks

` + "```bash" + `
# Claim the next available task (highest priority first)
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/PROJECT_ID/tasks/claim-next \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "my-agent"}' | jq .
` + "```" + `

Response includes: task details, run ID, lease, assembled context (memory, dependencies).

### Step 4: Complete Tasks

` + "```bash" + `
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks/TASK_ID/complete \
  -H "Content-Type: application/json" \
  -d '{
    "output": "Summary of work done",
    "result": {
      "summary": "Brief description",
      "changes_made": ["change 1", "change 2"],
      "files_touched": ["file1.go"]
    }
  }' | jq .
` + "```" + `

### Step 5: Poll Loop (Agent Pattern)

` + "```bash" + `
while true; do
  RESPONSE=$(curl -s -X POST http://127.0.0.1:8080/api/v2/projects/PROJECT_ID/tasks/claim-next \
    -H "Content-Type: application/json" \
    -d '{"agent_id": "my-agent"}')

  if echo "$RESPONSE" | jq -e '.task' > /dev/null 2>&1; then
    TASK_ID=$(echo "$RESPONSE" | jq -r '.task.id')
    INSTRUCTIONS=$(echo "$RESPONSE" | jq -r '.task.instructions')
    # ... do work based on INSTRUCTIONS ...
    curl -s -X POST "http://127.0.0.1:8080/api/v2/tasks/$TASK_ID/complete" \
      -H "Content-Type: application/json" \
      -d "{\"output\": \"work done\"}"
  fi
  sleep 15
done
` + "```" + `

---

## Complete API Reference

### Projects

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | /api/v2/projects | Create project (body: {name, workdir, settings?}) |
| GET | /api/v2/projects | List all projects |
| GET | /api/v2/projects/{id} | Get project details |
| PUT | /api/v2/projects/{id} | Update project (body: {name?, workdir?, settings?}) |
| DELETE | /api/v2/projects/{id} | Delete project |
| GET | /api/v2/projects/{id}/dashboard | Project dashboard summary |
| GET | /api/v2/projects/{id}/tree | File tree snapshot (?depth=N) |
| GET | /api/v2/projects/{id}/changes | Repo changes (?since=RFC3339) |

### Tasks

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | /api/v2/tasks | Create task globally (body: {instructions, project_id?, title?, type?, priority?, tags?, parent_task_id?}) |
| GET | /api/v2/tasks | List tasks (?project_id, ?status, ?type, ?tag, ?q, ?sort, ?limit, ?offset) |
| GET | /api/v2/tasks/{id} | Get task details |
| PATCH | /api/v2/tasks/{id} | Update task (body: {title?, instructions?, type?, priority?, tags?, output?}) |
| DELETE | /api/v2/tasks/{id} | Delete task |
| POST | /api/v2/tasks/{id}/claim | Claim specific task (body: {agent_id}) |
| POST | /api/v2/tasks/{id}/complete | Complete task (body: {output?, result?}) |
| POST | /api/v2/tasks/{id}/fail | Fail task (body: {error, output?}) |
| POST | /api/v2/tasks/{id}/approve | Approve reviewed task |
| POST | /api/v2/tasks/{id}/reject | Reject reviewed task |

### Project-Scoped Tasks

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | /api/v2/projects/{id}/tasks | Create task in project |
| GET | /api/v2/projects/{id}/tasks | List project tasks |
| POST | /api/v2/projects/{id}/tasks/claim-next | Claim next available (highest priority) |

### Task Dependencies

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/tasks/{id}/dependencies | List dependencies |
| POST | /api/v2/tasks/{id}/dependencies | Add dependency (body: {depends_on_task_id}) |
| DELETE | /api/v2/tasks/{id}/dependencies/{depId} | Remove dependency |

### Runs (Execution Ledger)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/runs/{runId} | Get run details (steps, logs, artifacts) |
| POST | /api/v2/runs/{runId}/steps | Log step (body: {name, status, details?}) |
| POST | /api/v2/runs/{runId}/logs | Stream log (body: {stream, chunk}) |
| POST | /api/v2/runs/{runId}/artifacts | Attach artifact (body: {kind, storage_ref, sha256?, size?, metadata?}) |

### Agents & Leases

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | /api/v2/agents | Register agent (body: {id, name, capabilities?, metadata?}) |
| GET | /api/v2/agents | List agents |
| GET | /api/v2/agents/{id} | Get agent details |
| DELETE | /api/v2/agents/{id} | Unregister agent |
| POST | /api/v2/leases | Create lease (body: {task_id, agent_id}) |
| GET | /api/v2/leases/{id} | Get lease |
| POST | /api/v2/leases/{id}/heartbeat | Renew lease |
| POST | /api/v2/leases/{id}/release | Release lease |

### Memory (Persistent Knowledge)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/projects/{id}/memory | List memories (?scope, ?key) |
| PUT | /api/v2/projects/{id}/memory | Store memory (body: {scope, key, value, source_refs?}) |
| GET | /api/v2/projects/{id}/memory/{memId} | Get memory item |
| DELETE | /api/v2/projects/{id}/memory/{memId} | Delete memory |

### Policies (Governance)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/projects/{id}/policies | List policies (?enabled) |
| POST | /api/v2/projects/{id}/policies | Create policy (body: {name, description?, rules, enabled}) |
| GET | /api/v2/projects/{id}/policies/{pId} | Get policy |
| PATCH | /api/v2/projects/{id}/policies/{pId} | Update policy |
| DELETE | /api/v2/projects/{id}/policies/{pId} | Delete policy |

### Events & Streaming

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/events | List events (?type, ?since, ?task_id, ?project_id, ?limit) |
| GET | /api/v2/events/stream | SSE event stream (?type, ?project_id) |
| GET | /api/v2/projects/{id}/events/stream | Project-scoped SSE stream |
| GET | /api/v2/projects/{id}/events/replay | Replay events (?since_event_id, ?limit) |

### Automation

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/projects/{id}/automation/rules | List automation rules |
| GET | /api/v2/projects/{id}/automation/stats | Automation stats |
| POST | /api/v2/projects/{id}/automation/simulate | Simulate event (body: {event: {kind, entity_id, payload}}) |
| POST | /api/v2/projects/{id}/automation/replay | Replay automation (?since_event_id, ?limit) |

### Context Packs

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/projects/{id}/context-packs | List context packs |
| POST | /api/v2/projects/{id}/context-packs | Create pack (body: {task_id, summary, contents}) |
| GET | /api/v2/context-packs/{packId} | Get context pack |
| DELETE | /api/v2/context-packs/{packId} | Delete context pack |

### Repo / Files

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/projects/{id}/files | List tracked files |
| POST | /api/v2/projects/{id}/files/scan | Scan workdir for files |
| POST | /api/v2/projects/{id}/files/sync | Sync files to DB |

### System

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /api/v2/health | Health check |
| GET | /api/v2/status | System status |
| GET | /api/v2/metrics | Performance metrics |
| GET | /api/v2/version | Server version |
| GET | /api/v2/config | Runtime config (redacted) |
| POST | /api/v2/backup | Create DB backup |
| POST | /api/v2/restore | Restore from backup |

---

## Task Types

| Type | Purpose |
|------|---------|
| ANALYZE | Code analysis, review, investigation |
| MODIFY | Code changes, feature implementation |
| TEST | Write or run tests |
| REVIEW | Code review |
| DOC | Documentation |
| RELEASE | Release management |
| ROLLBACK | Revert changes |
| PLAN | Planning, architecture |

## Task Status Lifecycle

` + "```" + `
QUEUED (initial)
  → CLAIMED (agent claims via /claim or /claim-next)
    → RUNNING (agent logs progress via run steps)
      → SUCCEEDED (/complete)
      → FAILED (/fail)
      → NEEDS_REVIEW (if policy requires review)
        → SUCCEEDED (/approve)
        → QUEUED (/reject — goes back to queue)
` + "```" + `

Priority: 0-100 (higher = claimed first by claim-next).

## Automation Rules

Set via COCO_AUTOMATION_RULES env var (JSON array). Rules trigger on events
like task.completed and automatically create follow-up tasks.

` + "```json" + `
[{
  "name": "Auto-review modifications",
  "enabled": true,
  "trigger": "task.completed",
  "actions": [{
    "type": "create_task",
    "task": {
      "title": "Review: ${task_title}",
      "instructions": "Review changes from: ${task_output}",
      "type": "REVIEW",
      "priority": 55,
      "tags": ["auto-review"],
      "parent": "completed"
    }
  }]
}]
` + "```" + `

Template variables: ${event_id}, ${event_kind}, ${project_id}, ${task_id},
${task_title}, ${task_instructions}, ${task_output}, ${task_status_v1}, ${task_status_v2}

Safety limits: max 100 automations/hour, 10/minute, max depth 5.

## Memory System

Store persistent knowledge that gets assembled into task context on claim:

` + "```bash" + `
# Store a fact
curl -s -X PUT http://127.0.0.1:8080/api/v2/projects/PROJECT_ID/memory \
  -H "Content-Type: application/json" \
  -d '{"scope": "architecture", "key": "stack", "value": {"lang": "Go", "db": "SQLite"}}'

# Retrieve all memory
curl -s http://127.0.0.1:8080/api/v2/projects/PROJECT_ID/memory | jq .
` + "```" + `

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| COCO_DB_PATH | ./tasks.db | SQLite database path |
| COCO_HTTP_ADDR | 127.0.0.1:8080 | Server listen address |
| COCO_REQUIRE_API_KEY | false | Require API key for v2 mutations |
| COCO_API_KEY | - | Shared API key when auth enabled |
| COCO_AUTOMATION_RULES | - | JSON array of automation rules |
| COCO_MAX_AUTOMATION_DEPTH | 5 | Max recursion depth for automation |
| COCO_AUTOMATION_RATE_LIMIT | 100 | Max automation executions per hour |
| COCO_AUTOMATION_BURST_LIMIT | 10 | Max automation executions per minute |

## Error Format

All v2 errors return:
` + "```json" + `
{"error": {"code": "not_found", "message": "task not found", "details": {}}}
` + "```" + `

## Tips for Autonomous Agents

1. **Discover projects first** — GET /api/v2/projects to find your target project
2. **Store knowledge in memory** — it gets assembled into context when tasks are claimed
3. **Use priorities** — higher priority tasks get claimed first (0-100)
4. **Use task types** — helps organize and filter work
5. **Add dependencies** — declare task ordering via POST /api/v2/tasks/{id}/dependencies
6. **Log run steps** — POST /api/v2/runs/{runId}/steps to track progress
7. **Attach artifacts** — POST /api/v2/runs/{runId}/artifacts for diffs, patches, test results
8. **Monitor events** — GET /api/v2/events/stream for real-time updates (SSE)
9. **Use automation** — set up rules to auto-create follow-up tasks
10. **Check health** — GET /api/v2/health to verify server is up
`

	return strings.ReplaceAll(body, "http://127.0.0.1:8080", baseURL)
}
