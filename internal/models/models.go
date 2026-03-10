// Package models contains all v2 data model types and helpers.
package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// LeaseTimeFormat is the canonical timestamp format used across the system.
const LeaseTimeFormat = "2006-01-02T15:04:05.999999Z"

// TaskStatus represents the v1 task status.
type TaskStatus string

const (
	StatusNotPicked  TaskStatus = "NOT_PICKED"
	StatusInProgress TaskStatus = "IN_PROGRESS"
	StatusComplete   TaskStatus = "COMPLETE"
	StatusFailed     TaskStatus = "FAILED"
)

// Project represents a workspace isolation unit.
type Project struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Workdir   string                 `json:"workdir"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

// TreeNode represents a file or directory in a project tree snapshot.
type TreeNode struct {
	Path     string     `json:"path"`
	Kind     string     `json:"kind"`
	Size     *int64     `json:"size,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// FileChange represents a repo change entry for project change feeds.
type FileChange struct {
	Path   string  `json:"path"`
	Kind   string  `json:"kind"`
	Sha256 *string `json:"sha256,omitempty"`
	Ts     string  `json:"ts"`
}

// TaskStatusV2 represents the detailed v2 status.
type TaskStatusV2 string

const (
	TaskStatusQueued      TaskStatusV2 = "QUEUED"
	TaskStatusClaimed     TaskStatusV2 = "CLAIMED"
	TaskStatusRunning     TaskStatusV2 = "RUNNING"
	TaskStatusSucceeded   TaskStatusV2 = "SUCCEEDED"
	TaskStatusFailed      TaskStatusV2 = "FAILED"
	TaskStatusNeedsReview TaskStatusV2 = "NEEDS_REVIEW"
	TaskStatusCancelled   TaskStatusV2 = "CANCELLED"
)

// IsTerminal returns true if the task status represents a final state.
func (s TaskStatusV2) IsTerminal() bool {
	switch s {
	case TaskStatusSucceeded, TaskStatusFailed, TaskStatusCancelled:
		return true
	}
	return false
}

// IsActive returns true if the task is currently being worked on.
func (s TaskStatusV2) IsActive() bool {
	return s == TaskStatusClaimed || s == TaskStatusRunning
}

// TaskType represents the type of task.
type TaskType string

const (
	TaskTypeAnalyze  TaskType = "ANALYZE"
	TaskTypeModify   TaskType = "MODIFY"
	TaskTypeTest     TaskType = "TEST"
	TaskTypeReview   TaskType = "REVIEW"
	TaskTypeDoc      TaskType = "DOC"
	TaskTypeRelease  TaskType = "RELEASE"
	TaskTypeRollback TaskType = "ROLLBACK"
	TaskTypePlan     TaskType = "PLAN"
)

// TaskV2 represents an enhanced task with v2 metadata.
type TaskV2 struct {
	ID               int          `json:"id"`
	ProjectID        string       `json:"project_id"`
	Title            *string      `json:"title,omitempty"`
	Instructions     string       `json:"instructions"`
	Type             TaskType     `json:"type"`
	Priority         int          `json:"priority"`
	Tags             []string     `json:"tags,omitempty"`
	StatusV1         TaskStatus   `json:"status_v1"`
	StatusV2         TaskStatusV2 `json:"status_v2"`
	ParentTaskID     *int         `json:"parent_task_id,omitempty"`
	Output           *string      `json:"output,omitempty"`
	CreatedAt        string       `json:"created_at"`
	UpdatedAt        *string      `json:"updated_at,omitempty"`
	AutomationDepth  int          `json:"automation_depth"`
	RequiresApproval bool         `json:"requires_approval,omitempty"`
	ApprovalStatus   *string      `json:"approval_status,omitempty"`
	LoopAnchorPrompt *string      `json:"loop_anchor_prompt,omitempty"`
}

// DefaultLoopAnchorPrompt is the system-wide default execution anchor prompt.
const DefaultLoopAnchorPrompt = "Stay focused on this task's actual objective. Continue the current line of work until you have made meaningful progress, completed the task, or hit a real blocker. Before stopping, check what remains, avoid drifting into unrelated work, and note the next logical step if anything is still unfinished."

// TaskDependency represents a dependency between two tasks.
type TaskDependency struct {
	TaskID          int `json:"task_id"`
	DependsOnTaskID int `json:"depends_on_task_id"`
}

// RunStatus represents the status of a run.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusSucceeded RunStatus = "SUCCEEDED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusCancelled RunStatus = "CANCELLED"
)

// IsTerminal returns true if the run status represents a final state.
func (s RunStatus) IsTerminal() bool {
	switch s {
	case RunStatusSucceeded, RunStatusFailed, RunStatusCancelled:
		return true
	}
	return false
}

// Run represents a task execution attempt.
type Run struct {
	ID         string    `json:"id"`
	TaskID     int       `json:"task_id"`
	AgentID    string    `json:"agent_id"`
	Status     RunStatus `json:"status"`
	StartedAt  string    `json:"started_at"`
	FinishedAt *string   `json:"finished_at,omitempty"`
	Error      *string   `json:"error,omitempty"`
}

// RunDetail represents a run with its related execution data.
type RunDetail struct {
	Run
	Steps           []RunStep        `json:"steps,omitempty"`
	Logs            []RunLog         `json:"logs,omitempty"`
	Artifacts       []Artifact       `json:"artifacts,omitempty"`
	ToolInvocations []ToolInvocation `json:"tool_invocations,omitempty"`
}

// StepStatus represents the status of a run step.
type StepStatus string

const (
	StepStatusStarted   StepStatus = "STARTED"
	StepStatusSucceeded StepStatus = "SUCCEEDED"
	StepStatusFailed    StepStatus = "FAILED"
)

// RunStep represents a major phase within a run.
type RunStep struct {
	ID        string                 `json:"id"`
	RunID     string                 `json:"run_id"`
	Name      string                 `json:"name"`
	Status    StepStatus             `json:"status"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

// RunLog represents a log entry from a run.
type RunLog struct {
	ID     int    `json:"id"`
	RunID  string `json:"run_id"`
	Stream string `json:"stream"`
	Chunk  string `json:"chunk"`
	Ts     string `json:"ts"`
}

// Artifact represents a file produced during execution.
type Artifact struct {
	ID         string                 `json:"id"`
	RunID      string                 `json:"run_id"`
	Kind       string                 `json:"kind"`
	StorageRef string                 `json:"storage_ref"`
	Sha256     *string                `json:"sha256,omitempty"`
	Size       *int64                 `json:"size,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  string                 `json:"created_at"`
}

// ToolInvocation represents a tool call during execution.
type ToolInvocation struct {
	ID         string                 `json:"id"`
	RunID      string                 `json:"run_id"`
	ToolName   string                 `json:"tool_name"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Output     map[string]interface{} `json:"output,omitempty"`
	StartedAt  string                 `json:"started_at"`
	FinishedAt *string                `json:"finished_at,omitempty"`
}

// Lease represents a task claim by an agent.
type Lease struct {
	ID        string `json:"id"`
	TaskID    int    `json:"task_id"`
	AgentID   string `json:"agent_id"`
	Mode      string `json:"mode"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

// AgentStatus represents the status of an agent.
type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "ONLINE"
	AgentStatusOffline AgentStatus = "OFFLINE"
	AgentStatusBusy    AgentStatus = "BUSY"
	AgentStatusIdle    AgentStatus = "IDLE"
)

// Agent represents a registered agent in the system.
type Agent struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Capabilities []string               `json:"capabilities,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Status       AgentStatus            `json:"status"`
	LastSeen     *string                `json:"last_seen,omitempty"`
	RegisteredAt string                 `json:"registered_at"`
}

// Event represents a system event for real-time notifications.
type Event struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	Kind       string                 `json:"kind"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	CreatedAt  string                 `json:"created_at"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// Memory represents a persistent knowledge item.
type Memory struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	Scope      string                 `json:"scope"`
	Key        string                 `json:"key"`
	Value      map[string]interface{} `json:"value"`
	SourceRefs []string               `json:"source_refs,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// PolicyRule represents a single policy rule.
type PolicyRule map[string]interface{}

// Policy represents a policy engine rule set scoped to a project.
type Policy struct {
	ID          string       `json:"id"`
	ProjectID   string       `json:"project_id"`
	Name        string       `json:"name"`
	Description *string      `json:"description,omitempty"`
	Rules       []PolicyRule `json:"rules"`
	Enabled     bool         `json:"enabled"`
	CreatedAt   string       `json:"created_at"`
}

// ContextPack represents a pre-generated context bundle.
type ContextPack struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	TaskID    int                    `json:"task_id"`
	Summary   string                 `json:"summary"`
	Contents  map[string]interface{} `json:"contents"`
	CreatedAt string                 `json:"created_at"`
	Stale     bool                   `json:"stale,omitempty"`
}

// RunSummary is a lightweight summary of a run for context assembly.
type RunSummary struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	StartedAt    string   `json:"started_at"`
	FinishedAt   string   `json:"finished_at,omitempty"`
	Summary      string   `json:"summary,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
	FilesTouched []string `json:"files_touched,omitempty"`
}

// TaskTemplate represents a reusable task definition for a project.
type TaskTemplate struct {
	ID                string                 `json:"id"`
	ProjectID         string                 `json:"project_id"`
	Name              string                 `json:"name"`
	Description       *string                `json:"description,omitempty"`
	Instructions      string                 `json:"instructions"`
	DefaultType       *string                `json:"default_type,omitempty"`
	DefaultPriority   int                    `json:"default_priority"`
	DefaultTags       []string               `json:"default_tags,omitempty"`
	DefaultMetadata   map[string]interface{} `json:"default_metadata,omitempty"`
	DefaultLoopAnchor *string                `json:"default_loop_anchor,omitempty"`
	CreatedAt         string                 `json:"created_at"`
	UpdatedAt         string                 `json:"updated_at"`
}

// ApprovalStatus constants for human-in-the-loop.
const (
	ApprovalPending  = "pending_approval"
	ApprovalApproved = "approved"
	ApprovalRejected = "rejected"
)

// RepoFile represents a file in a project's repository.
type RepoFile struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	Path         string                 `json:"path"`
	ContentHash  *string                `json:"content_hash,omitempty"`
	SizeBytes    *int64                 `json:"size_bytes,omitempty"`
	Language     *string                `json:"language,omitempty"`
	LastModified *string                `json:"last_modified,omitempty"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ListRepoFilesOpts holds filtering/pagination options for listing repo files.
type ListRepoFilesOpts struct {
	Language *string
	Limit    int
	Offset   int
}

// ArtifactComment represents a line comment on an artifact (diff viewer).
type ArtifactComment struct {
	ID         string `json:"id"`
	ArtifactID string `json:"artifact_id"`
	ProjectID  string `json:"project_id,omitempty"`
	LineNumber int    `json:"line_number"`
	Body       string `json:"body"`
	Author     string `json:"author,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// DashboardData holds a precomputed project dashboard snapshot.
type DashboardData struct {
	ProjectID        string                    `json:"project_id"`
	TaskCounts       DashboardTaskCounts       `json:"task_counts"`
	ActiveRuns       []DashboardRun            `json:"active_runs"`
	RecentChanges    []DashboardRepoChange     `json:"recent_changes"`
	RecentFailures   []DashboardFailure        `json:"recent_failures"`
	AutomationEvents []DashboardAutoEvent      `json:"automation_events"`
	Recommendations  []DashboardRecommendation `json:"recommendations"`
	StalledTasks     []DashboardStalledTask    `json:"stalled_tasks"`
}

// DashboardTaskCounts contains per-bucket task counts.
type DashboardTaskCounts struct {
	Queued     int `json:"queued"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}

// DashboardRun summarises an active run for the dashboard.
type DashboardRun struct {
	RunID     string `json:"run_id"`
	TaskTitle string `json:"task_title"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

// DashboardRepoChange summarises a recent repo file change.
type DashboardRepoChange struct {
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
}

// DashboardFailure summarises a recently failed task.
type DashboardFailure struct {
	TaskID    int    `json:"task_id"`
	Title     string `json:"title,omitempty"`
	Error     string `json:"error,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

// DashboardAutoEvent summarises an automation event for the dashboard.
type DashboardAutoEvent struct {
	Kind      string `json:"kind"`
	CreatedAt string `json:"created_at"`
}

// DashboardRecommendation suggests a task to work on next, with a reason.
type DashboardRecommendation struct {
	TaskID   int    `json:"task_id"`
	Title    string `json:"title,omitempty"`
	Priority int    `json:"priority"`
	Type     string `json:"type,omitempty"`
	Why      string `json:"why"`
}

// DashboardStalledTask represents a task that appears stuck.
type DashboardStalledTask struct {
	TaskID    int    `json:"task_id"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status"`
	AgentID   string `json:"agent_id,omitempty"`
	ClaimedAt string `json:"claimed_at"`
	Reason    string `json:"reason"`
}

// ===== Helper Functions =====

// MarshalJSON marshals v to a JSON string.
func MarshalJSON(v interface{}) (string, error) {
	if v == nil {
		return "", nil
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// UnmarshalJSON unmarshals a JSON string into v.
func UnmarshalJSON(s string, v interface{}) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}

// NullString converts a *string to sql.NullString.
func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// PtrString converts a sql.NullString to *string.
func PtrString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// PtrInt64 converts a sql.NullInt64 to *int64.
func PtrInt64(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}

// NowISO returns the current time as an ISO 8601 string with microsecond precision.
func NowISO() string {
	return time.Now().UTC().Format(LeaseTimeFormat)
}

// ===== Status Canonicalization Helpers =====

// TaskStatusBuckets returns the canonical grouping of task statuses for metrics/reporting.
func TaskStatusBuckets() map[string][]TaskStatusV2 {
	return map[string][]TaskStatusV2{
		"active":    {TaskStatusClaimed, TaskStatusRunning},
		"queued":    {TaskStatusQueued},
		"terminal":  {TaskStatusSucceeded, TaskStatusFailed, TaskStatusNeedsReview, TaskStatusCancelled},
		"success":   {TaskStatusSucceeded},
		"failed":    {TaskStatusFailed},
		"review":    {TaskStatusNeedsReview},
		"cancelled": {TaskStatusCancelled},
	}
}

// RunStatusBuckets returns the canonical grouping of run statuses for metrics/reporting.
func RunStatusBuckets() map[string][]RunStatus {
	return map[string][]RunStatus{
		"active":    {RunStatusRunning},
		"terminal":  {RunStatusSucceeded, RunStatusFailed, RunStatusCancelled},
		"success":   {RunStatusSucceeded},
		"failed":    {RunStatusFailed},
		"cancelled": {RunStatusCancelled},
	}
}

// IsTaskQueued returns true if task is waiting to be claimed.
func IsTaskQueued(status TaskStatusV2) bool {
	return status == TaskStatusQueued
}

// IsTaskActive returns true if task is currently being worked on.
func IsTaskActive(status TaskStatusV2) bool {
	return status == TaskStatusClaimed || status == TaskStatusRunning
}

// IsTaskTerminal returns true if task is in a final state.
func IsTaskTerminal(status TaskStatusV2) bool {
	return status.IsTerminal()
}

// BucketForStatus returns the bucket name for a given task status.
func BucketForStatus(status TaskStatusV2) string {
	switch status {
	case TaskStatusQueued:
		return "queued"
	case TaskStatusClaimed, TaskStatusRunning:
		return "active"
	case TaskStatusSucceeded:
		return "success"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusNeedsReview:
		return "review"
	case TaskStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// IsRunActive returns true if run is currently executing.
func IsRunActive(status RunStatus) bool {
	return status == RunStatusRunning
}

// IsRunTerminal returns true if run is in a final state.
func IsRunTerminal(status RunStatus) bool {
	return status.IsTerminal()
}

// IsLeaseActiveAt checks if a lease is active at a given timestamp string.
func IsLeaseActiveAt(expiresAt string, at string) bool {
	if expiresAt == "" {
		return false
	}
	return at < expiresAt
}

// IsLeaseActive checks if a lease is currently active (not expired).
func IsLeaseActive(expiresAt string) bool {
	return IsLeaseActiveAt(expiresAt, NowISO())
}

// TaskStatusIsSuccessful returns true if task completed successfully.
func TaskStatusIsSuccessful(status TaskStatusV2) bool {
	return status == TaskStatusSucceeded
}

// TaskStatusIsFailed returns true if task failed.
func TaskStatusIsFailed(status TaskStatusV2) bool {
	return status == TaskStatusFailed
}

// RunStatusIsSuccessful returns true if run completed successfully.
func RunStatusIsSuccessful(status RunStatus) bool {
	return status == RunStatusSucceeded
}

// RunStatusIsFailed returns true if run failed.
func RunStatusIsFailed(status RunStatus) bool {
	return status == RunStatusFailed
}

// ===== Planning State Types =====

// PlanningMode represents how the planner should behave.
type PlanningMode string

const (
	PlanningModeStandard    PlanningMode = "standard"
	PlanningModeFocused     PlanningMode = "focused"
	PlanningModeRecovery    PlanningMode = "recovery"
	PlanningModeMaintenance PlanningMode = "maintenance"
)

// PlanningState holds the per-project planning state for the staged planner.
type PlanningState struct {
	ID             string   `json:"id"`
	ProjectID      string   `json:"project_id"`
	PlanningMode   string   `json:"planning_mode"`
	CycleCount     int      `json:"cycle_count"`
	LastCycleAt    *string  `json:"last_cycle_at,omitempty"`
	Goals          []string `json:"goals"`
	ReleaseFocus   string   `json:"release_focus"`
	MustNotForget  []string `json:"must_not_forget"`
	ReconSummary   string   `json:"recon_summary"`
	PlannerSummary string   `json:"planner_summary"`
	Blockers       []string `json:"blockers"`
	Risks          []string `json:"risks"`
	PriorityOrder  []string `json:"priority_order"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// WorkstreamStatus constants.
const (
	WorkstreamStatusActive    = "active"
	WorkstreamStatusPaused    = "paused"
	WorkstreamStatusCompleted = "completed"
	WorkstreamStatusAbandoned = "abandoned"
)

// Workstream represents a logical thread of work tracked across planning cycles.
type Workstream struct {
	ID              string  `json:"id"`
	ProjectID       string  `json:"project_id"`
	PlanningStateID string  `json:"planning_state_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Status          string  `json:"status"`
	ContinuityScore float64 `json:"continuity_score"`
	UrgencyScore    float64 `json:"urgency_score"`
	RelatedTaskIDs  []int   `json:"related_task_ids"`
	RelatedRunIDs   []int   `json:"related_run_ids"`
	Why             string  `json:"why"`
	WhatRemains     string  `json:"what_remains"`
	WhatNext        string  `json:"what_next"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// PlanningCycle records input/output of a completed planning cycle.
type PlanningCycle struct {
	ID                  string                 `json:"id"`
	ProjectID           string                 `json:"project_id"`
	CycleNumber         int                    `json:"cycle_number"`
	PlanningMode        string                 `json:"planning_mode"`
	StartedAt           string                 `json:"started_at"`
	CompletedAt         *string                `json:"completed_at,omitempty"`
	ReconOutput         map[string]interface{} `json:"recon_output"`
	ContinuityOutput    map[string]interface{} `json:"continuity_output"`
	GapOutput           map[string]interface{} `json:"gap_output"`
	PrioritizationOutput map[string]interface{} `json:"prioritization_output"`
	SynthesisOutput     map[string]interface{} `json:"synthesis_output"`
	AntiDriftOutput     map[string]interface{} `json:"anti_drift_output"`
	TasksCreated        []int                  `json:"tasks_created"`
	CoherenceScore      float64                `json:"coherence_score"`
	StageFailures       []string               `json:"stage_failures"`
	DriftWarnings       []string               `json:"drift_warnings"`
}

// PlannerDecision records a single decision made during a planning cycle.
type PlannerDecision struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	CycleID      string `json:"cycle_id"`
	Stage        string `json:"stage"`
	DecisionType string `json:"decision_type"`
	Subject      string `json:"subject"`
	Reasoning    string `json:"reasoning"`
	CreatedAt    string `json:"created_at"`
}

// PromptTemplate represents a versioned prompt template for a specific role.
type PromptTemplate struct {
	ID           string  `json:"id"`
	ProjectID    string  `json:"project_id"`
	Role         string  `json:"role"`
	Version      int     `json:"version"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	SystemPrompt string  `json:"system_prompt"`
	UserTemplate string  `json:"user_template"`
	OutputSchema *string `json:"output_schema,omitempty"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}
