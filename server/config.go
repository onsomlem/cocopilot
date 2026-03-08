package server

import (
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

func getInstructions() string {
	workdirMu.RLock()
	wd := workdir
	workdirMu.RUnlock()

	return `Those are your general instructions, this is an orchestrator tool for running subagents.

IMPORTANT - WORKING DIRECTORY:
All subagents MUST ALWAYS work in the following directory:
` + "`" + wd + "`" + `

Before doing any work, the subagent must first change to this directory:
` + "```" + `
cd ` + wd + `
` + "```" + `

## API (v2)

All orchestrator communication uses the v2 JSON API.

### 1. Claim a task

` + "```" + `
POST http://127.0.0.1:8080/api/v2/projects/default/tasks/claim-next
Content-Type: application/json

{"agent_id": "copilot"}
` + "```" + `

Response (200):
` + "```json" + `
{
  "task": { "id": 123, "title": "...", "instructions": "...", "status": "IN_PROGRESS", ... },
  "context": { ... },
  "run": { "id": "...", ... }
}
` + "```" + `

If no tasks are available, the response is 404. Poll again after 15 seconds.
The response contains the task ID, instructions, and assembled context.

### 2. Complete a task

` + "```" + `
POST http://127.0.0.1:8080/api/v2/tasks/<task_id>/complete
Content-Type: application/json

{
  "output": "<summarized_output_here>",
  "result": {
    "summary": "Brief description of what was done",
    "changes_made": ["change 1", "change 2"],
    "files_touched": ["file1.go", "file2.go"]
  }
}
` + "```" + `

### 3. List tasks (optional)

` + "```" + `
GET http://127.0.0.1:8080/api/v2/tasks?status=pending
` + "```" + `

## Workflow

For each instruction, start a sub agent using the ` + "`runSubagent`" + ` tool.
The subagent must know the task ID so it can complete the task via the v2 API.

At the end of each sub agent, it must:
1. Produce a summarized output of what it has done
2. Complete the task using the v2 endpoint above

After that the subagent returns control to the main orchestrator, which will
claim the next task after 15 seconds. Use bash command to wait it out and
then claim the next task. If a new task is claimed, follow these instructions again.

Important Note:

Always use ` + "`curl`" + ` to do the web requests.
`
}
