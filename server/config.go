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

Your task is to follow the instructions at the ` + "`/task`" + ` endpoint.
For each instruction, you will start a sub agent using the ` + "`runSubagent`" + ` tool.
The subagent must know what task ID is given, so that he can issue a request later on.

At the end of each sub agent, the sub agent must produce a summarized output
of what it has done in a few points, the sub agent then will then issue this request using curl:

` + "```" + `
POST http://127.0.0.1:8080/save
Content-Type: application/x-www-form-urlencoded
task_id=<task_id>&message=<summarized_output_here>
` + "```" + `

After that the subagent would return the control back to the main orchestrator, which will poll
the ` + "`/task`" + ` endpoint after 15 seconds for new instructions. Use bash command to wait it out and
then issue a new request towards ` + "`/task`" + `. If there is a new task at hand follow the instructions
in the beginning again.

Important Note:

Always use ` + "`curl`" + ` to do the web requests.
`
}
