// API v2 Database Access Layer — thin wrappers delegating to internal/dbstore.
package server

import (
	"database/sql"
	"time"

	"github.com/onsomlem/cocopilot/internal/dbstore"
	"github.com/onsomlem/cocopilot/internal/models"
)

// Re-export sentinel errors so callers in the root package keep compiling.
var ErrLeaseConflict = dbstore.ErrLeaseConflict
var ErrTaskDependencyExists = dbstore.ErrTaskDependencyExists
var ErrTaskDependencyCycle = dbstore.ErrTaskDependencyCycle
var ErrInvalidPolicyRules = dbstore.ErrInvalidPolicyRules

const leaseTimeFormat = models.LeaseTimeFormat

// allowedPolicyRuleTypes re-exports the canonical map from dbstore.
var allowedPolicyRuleTypes = dbstore.AllowedPolicyRuleTypes

func init() {
	dbstore.OnEventCreated = func(db *sql.DB, event models.Event) {
		publishV2Event(event)
		NotifyEvent(event)
		processAutomationEvent(db, event)
	}
	dbstore.OnEventCreatedTx = func(event models.Event) {
		publishV2Event(event)
	}
}

// ---- Projects ----

func CreateProject(db *sql.DB, name, workdir string, settings map[string]interface{}) (*Project, error) {
	return dbstore.CreateProject(db, name, workdir, settings)
}
func GetProject(db *sql.DB, projectID string) (*Project, error) {
	return dbstore.GetProject(db, projectID)
}
func ListProjects(db *sql.DB) ([]Project, error) { return dbstore.ListProjects(db) }
func UpdateProject(db *sql.DB, projectID string, name *string, workdir *string, settings map[string]interface{}) (*Project, error) {
	return dbstore.UpdateProject(db, projectID, name, workdir, settings)
}
func DeleteProject(db *sql.DB, projectID string) error { return dbstore.DeleteProject(db, projectID) }

// ---- Task status mapping ----

func mapTaskStatusV1ToV2(status TaskStatus) TaskStatusV2 {
	return dbstore.MapTaskStatusV1ToV2(status)
}
func mapTaskStatusV2ToV1(status TaskStatusV2) TaskStatus {
	return dbstore.MapTaskStatusV2ToV1(status)
}

// ---- Tasks ----

func GetTaskV2(db *sql.DB, taskID int) (*TaskV2, error) { return dbstore.GetTaskV2(db, taskID) }
func setTaskAutomationDepth(db *sql.DB, taskID int, depth int) error {
	return dbstore.SetTaskAutomationDepth(db, taskID, depth)
}
func CreateTaskV2(db *sql.DB, instructions string, projectID string, parentTaskID *int) (*TaskV2, error) {
	return dbstore.CreateTaskV2(db, instructions, projectID, parentTaskID)
}
func CreateTaskV2WithMeta(db *sql.DB, instructions string, projectID string, parentTaskID *int, title *string, taskType *TaskType, priority *int, tags []string) (*TaskV2, error) {
	return dbstore.CreateTaskV2WithMeta(db, instructions, projectID, parentTaskID, title, taskType, priority, tags)
}
func UpdateTaskV2(db *sql.DB, taskID int, instructions *string, statusV1 *TaskStatus, statusV2 *TaskStatusV2, projectID *string, parentTaskID *int) (*TaskV2, error) {
	return dbstore.UpdateTaskV2(db, taskID, instructions, statusV1, statusV2, projectID, parentTaskID)
}
func GetTaskParentChain(db *sql.DB, parentID *int) ([]TaskV2, error) {
	return dbstore.GetTaskParentChain(db, parentID)
}
func GetLatestRunByTaskID(db *sql.DB, taskID int) (*Run, error) {
	return dbstore.GetLatestRunByTaskID(db, taskID)
}
func ListTasksV2(db *sql.DB, projectID string, statusColumn string, statusValue string, typeFilter string, tagFilter string, queryFilter string, limit int, offset int, sortField string, sortDirection string) ([]TaskV2, int, error) {
	return dbstore.ListTasksV2(db, projectID, statusColumn, statusValue, typeFilter, tagFilter, queryFilter, limit, offset, sortField, sortDirection)
}
func TaskExists(db *sql.DB, taskID int) (bool, error) { return dbstore.TaskExists(db, taskID) }

// ---- Task Dependencies ----

func CreateTaskDependency(db *sql.DB, taskID int, dependsOnTaskID int) (*TaskDependency, error) {
	return dbstore.CreateTaskDependency(db, taskID, dependsOnTaskID)
}
func TaskDependencyCreatesCycle(db *sql.DB, taskID int, dependsOnTaskID int) (bool, error) {
	return dbstore.TaskDependencyCreatesCycle(db, taskID, dependsOnTaskID)
}
func ListTaskDependencies(db *sql.DB, taskID int) ([]TaskDependency, error) {
	return dbstore.ListTaskDependencies(db, taskID)
}
func DeleteTaskDependency(db *sql.DB, taskID int, dependsOnTaskID int) (bool, error) {
	return dbstore.DeleteTaskDependency(db, taskID, dependsOnTaskID)
}
func GetTasksDependingOn(db *sql.DB, completedTaskID int) ([]TaskV2, error) {
	return dbstore.GetTasksDependingOn(db, completedTaskID)
}
func AreAllDependenciesFulfilled(db *sql.DB, taskID int) (bool, error) {
	return dbstore.AreAllDependenciesFulfilled(db, taskID)
}

// ---- Agents ----

func RegisterAgent(db *sql.DB, name string, capabilities []string, metadata map[string]interface{}) (*Agent, error) {
	return dbstore.RegisterAgent(db, name, capabilities, metadata)
}
func GetAgent(db *sql.DB, agentID string) (*Agent, error) { return dbstore.GetAgent(db, agentID) }
func EnsureAgent(db *sql.DB, agentID string) error         { return dbstore.EnsureAgent(db, agentID) }
func ListAgents(db *sql.DB, statusFilter string, since string, limit int, offset int, sortField string, sortDirection string) ([]Agent, int, error) {
	return dbstore.ListAgents(db, statusFilter, since, limit, offset, sortField, sortDirection)
}
func DeleteAgent(db *sql.DB, agentID string) (*Agent, error) { return dbstore.DeleteAgent(db, agentID) }
func UpdateAgentStatus(db *sql.DB, agentID string, status AgentStatus) error {
	return dbstore.UpdateAgentStatus(db, agentID, status)
}
func UpdateAgentHeartbeat(db *sql.DB, agentID string) error {
	return dbstore.UpdateAgentHeartbeat(db, agentID)
}
func MarkStaleAgentsOffline(db *sql.DB, staleThresholdMinutes int) error {
	return dbstore.MarkStaleAgentsOffline(db, staleThresholdMinutes)
}
func UpdateAgentCapabilities(db *sql.DB, agentID string, capabilities []string) (*Agent, error) {
	return dbstore.UpdateAgentCapabilities(db, agentID, capabilities)
}

// ---- Runs ----

func CreateRun(db *sql.DB, taskID int, agentID string) (*Run, error) {
	return dbstore.CreateRun(db, taskID, agentID)
}
func GetRun(db *sql.DB, runID string) (*Run, error) { return dbstore.GetRun(db, runID) }
func GetRunsByTaskID(db *sql.DB, taskID int) ([]Run, error) {
	return dbstore.GetRunsByTaskID(db, taskID)
}
func UpdateRunStatus(db *sql.DB, runID string, status RunStatus, errorMsg *string) error {
	return dbstore.UpdateRunStatus(db, runID, status, errorMsg)
}
func DeleteRun(db *sql.DB, runID string) error { return dbstore.DeleteRun(db, runID) }

// ---- Run Steps ----

func CreateRunStep(db *sql.DB, runID, name string, status StepStatus, details map[string]interface{}) (*RunStep, error) {
	return dbstore.CreateRunStep(db, runID, name, status, details)
}
func GetRunSteps(db *sql.DB, runID string) ([]RunStep, error) {
	return dbstore.GetRunSteps(db, runID)
}
func UpdateRunStepStatus(db *sql.DB, stepID string, status StepStatus) error {
	return dbstore.UpdateRunStepStatus(db, stepID, status)
}

// ---- Run Logs ----

func CreateRunLog(db *sql.DB, runID, stream, chunk string) error {
	return dbstore.CreateRunLog(db, runID, stream, chunk)
}
func GetRunLogs(db *sql.DB, runID string) ([]RunLog, error) {
	return dbstore.GetRunLogs(db, runID)
}

// ---- Artifacts ----

func CreateArtifact(db *sql.DB, runID, kind, storageRef string, sha256 *string, size *int64, metadata map[string]interface{}) (*Artifact, error) {
	return dbstore.CreateArtifact(db, runID, kind, storageRef, sha256, size, metadata)
}
func GetArtifactsByRunID(db *sql.DB, runID string) ([]Artifact, error) {
	return dbstore.GetArtifactsByRunID(db, runID)
}

// ---- Tool Invocations ----

func CreateToolInvocation(db *sql.DB, runID, toolName string, input map[string]interface{}) (*ToolInvocation, error) {
	return dbstore.CreateToolInvocation(db, runID, toolName, input)
}
func UpdateToolInvocationOutput(db *sql.DB, invocationID string, output map[string]interface{}) error {
	return dbstore.UpdateToolInvocationOutput(db, invocationID, output)
}
func GetToolInvocationsByRunID(db *sql.DB, runID string) ([]ToolInvocation, error) {
	return dbstore.GetToolInvocationsByRunID(db, runID)
}

// ---- Leases ----

func CreateLeaseTx(tx *sql.Tx, taskID int, agentID string, mode string) (*Lease, error) {
	return dbstore.CreateLeaseTx(tx, taskID, agentID, mode)
}
func CreateLease(db *sql.DB, taskID int, agentID string, mode string) (*Lease, error) {
	return dbstore.CreateLease(db, taskID, agentID, mode)
}
func GetLeaseByTaskID(db *sql.DB, taskID int) (*Lease, error) {
	return dbstore.GetLeaseByTaskID(db, taskID)
}
func DeleteLease(db *sql.DB, leaseID string) error { return dbstore.DeleteLease(db, leaseID) }
func ReleaseLease(db *sql.DB, leaseID string, reason string) (bool, *Lease, error) {
	return dbstore.ReleaseLease(db, leaseID, reason)
}
func DeleteExpiredLeases(db *sql.DB) (int64, error) { return dbstore.DeleteExpiredLeases(db) }
func GetLeaseByID(db *sql.DB, leaseID string) (*Lease, error) {
	return dbstore.GetLeaseByID(db, leaseID)
}
func ExtendLease(db *sql.DB, leaseID string, duration time.Duration) error {
	return dbstore.ExtendLease(db, leaseID, duration)
}
func isLeaseConflictError(err error) bool  { return dbstore.IsLeaseConflictError(err) }
func isSQLiteBusyError(err error) bool     { return dbstore.IsSQLiteBusyError(err) }

func parseEventTaskID(raw string) (int, bool) {
	return dbstore.ParseEventTaskID(raw)
}
func taskIDFromPayload(payload map[string]interface{}) (int, bool) {
	return dbstore.TaskIDFromPayload(payload)
}

// ---- Events ----

func CreateEvent(db *sql.DB, projectID, kind, entityType, entityID string, payload map[string]interface{}) (*Event, error) {
	return dbstore.CreateEvent(db, projectID, kind, entityType, entityID, payload)
}
func CreateEventTx(tx *sql.Tx, projectID, kind, entityType, entityID string, payload map[string]interface{}) (*Event, error) {
	return dbstore.CreateEventTx(tx, projectID, kind, entityType, entityID, payload)
}
func GetEventsByProjectID(db *sql.DB, projectID string, limit int) ([]Event, error) {
	return dbstore.GetEventsByProjectID(db, projectID, limit)
}
func GetEventCreatedAtByID(db *sql.DB, eventID string) (string, error) {
	return dbstore.GetEventCreatedAtByID(db, eventID)
}
func GetEventReplayAnchor(db *sql.DB, eventID string) (string, string, error) {
	return dbstore.GetEventReplayAnchor(db, eventID)
}
func ListEvents(db *sql.DB, projectID string, kind string, since string, taskID string, limit int, offset int) ([]Event, int, error) {
	return dbstore.ListEvents(db, projectID, kind, since, taskID, limit, offset)
}
func PruneEvents(db *sql.DB, retentionDays int, maxRows int) (int64, error) {
	return dbstore.PruneEvents(db, retentionDays, maxRows)
}

// ---- Memory ----

func CreateMemory(db *sql.DB, projectID, scope, key string, value map[string]interface{}, sourceRefs []string) (*Memory, error) {
	return dbstore.CreateMemory(db, projectID, scope, key, value, sourceRefs)
}
func GetMemory(db *sql.DB, projectID, scope, key string) (*Memory, error) {
	return dbstore.GetMemory(db, projectID, scope, key)
}
func GetMemoriesByScope(db *sql.DB, projectID, scope string) ([]Memory, error) {
	return dbstore.GetMemoriesByScope(db, projectID, scope)
}
func QueryMemories(db *sql.DB, projectID, scope, key, queryFilter string) ([]Memory, error) {
	return dbstore.QueryMemories(db, projectID, scope, key, queryFilter)
}
func DeleteMemory(db *sql.DB, projectID, scope, key string) error {
	return dbstore.DeleteMemory(db, projectID, scope, key)
}

// ---- Policies ----

func validatePolicyRules(rules []PolicyRule) error { return dbstore.ValidatePolicyRules(rules) }
func CreatePolicy(db *sql.DB, projectID, name string, description *string, rules []PolicyRule, enabled bool) (*Policy, error) {
	return dbstore.CreatePolicy(db, projectID, name, description, rules, enabled)
}
func ListPoliciesByProject(db *sql.DB, projectID string, enabledFilter *bool, limit, offset int, sortField, sortDirection string) ([]Policy, int, error) {
	return dbstore.ListPoliciesByProject(db, projectID, enabledFilter, limit, offset, sortField, sortDirection)
}
func GetPolicy(db *sql.DB, projectID, policyID string) (*Policy, error) {
	return dbstore.GetPolicy(db, projectID, policyID)
}
func UpdatePolicy(db *sql.DB, projectID, policyID string, name *string, description *string, rules []PolicyRule, enabled *bool) (*Policy, error) {
	return dbstore.UpdatePolicy(db, projectID, policyID, name, description, rules, enabled)
}
func DeletePolicy(db *sql.DB, projectID, policyID string) error {
	return dbstore.DeletePolicy(db, projectID, policyID)
}

// ---- Context Packs ----

func CreateContextPack(db *sql.DB, projectID string, taskID int, summary string, contents map[string]interface{}) (*ContextPack, error) {
	return dbstore.CreateContextPack(db, projectID, taskID, summary, contents)
}
func GetContextPackByTaskID(db *sql.DB, taskID int) (*ContextPack, error) {
	return dbstore.GetContextPackByTaskID(db, taskID)
}
func GetContextPackByID(db *sql.DB, packID string) (*ContextPack, error) {
	return dbstore.GetContextPackByID(db, packID)
}
func MarkContextPacksStale(db *sql.DB, projectID string) error {
	return dbstore.MarkContextPacksStale(db, projectID)
}
func RefreshContextPack(db *sql.DB, projectID string, taskID int, summary string, contents map[string]interface{}) (*ContextPack, error) {
	return dbstore.RefreshContextPack(db, projectID, taskID, summary, contents)
}

// ---- Repo Files ----

func UpsertRepoFile(db *sql.DB, file RepoFile) (*RepoFile, error) {
	return dbstore.UpsertRepoFile(db, file)
}
func GetRepoFile(db *sql.DB, projectID, path string) (*RepoFile, error) {
	return dbstore.GetRepoFile(db, projectID, path)
}
func ListRepoFiles(db *sql.DB, projectID string, opts ListRepoFilesOpts) ([]RepoFile, int, error) {
	return dbstore.ListRepoFiles(db, projectID, opts)
}
func DeleteRepoFile(db *sql.DB, projectID, path string) error {
	return dbstore.DeleteRepoFile(db, projectID, path)
}
func DeleteRepoFilesByProject(db *sql.DB, projectID string) error {
	return dbstore.DeleteRepoFilesByProject(db, projectID)
}
func ListRecentRepoFiles(db *sql.DB, projectID string, limit int) ([]RepoFile, error) {
	return dbstore.ListRecentRepoFiles(db, projectID, limit)
}

// ---- Artifact Comments ----

func CreateArtifactComment(db *sql.DB, artifactID string, projectID string, lineNumber int, body string, author string) (*ArtifactComment, error) {
	return dbstore.CreateArtifactComment(db, artifactID, projectID, lineNumber, body, author)
}
func ListArtifactComments(db *sql.DB, artifactID string) ([]ArtifactComment, error) {
	return dbstore.ListArtifactComments(db, artifactID)
}
func DeleteArtifactComment(db *sql.DB, commentID string) error {
	return dbstore.DeleteArtifactComment(db, commentID)
}

// ---- Dashboard ----

func GetProjectDashboardData(db *sql.DB, projectID string) (*DashboardData, error) {
	return dbstore.GetProjectDashboardData(db, projectID)
}

// ---- Templates ----

func CreateTaskTemplate(db *sql.DB, projectID, name string, description *string, instructions string, defaultType *string, defaultPriority int, defaultTags []string, defaultMetadata map[string]interface{}) (*TaskTemplate, error) {
	return dbstore.CreateTaskTemplate(db, projectID, name, description, instructions, defaultType, defaultPriority, defaultTags, defaultMetadata)
}
func GetTaskTemplate(db *sql.DB, templateID string) (*TaskTemplate, error) {
	return dbstore.GetTaskTemplate(db, templateID)
}
func ListTaskTemplates(db *sql.DB, projectID string) ([]TaskTemplate, error) {
	return dbstore.ListTaskTemplates(db, projectID)
}
func UpdateTaskTemplate(db *sql.DB, templateID string, name *string, description *string, instructions *string, defaultType *string, defaultPriority *int, defaultTags []string, defaultMetadata map[string]interface{}) (*TaskTemplate, error) {
	return dbstore.UpdateTaskTemplate(db, templateID, name, description, instructions, defaultType, defaultPriority, defaultTags, defaultMetadata)
}
func DeleteTaskTemplate(db *sql.DB, templateID string) error {
	return dbstore.DeleteTaskTemplate(db, templateID)
}

// ---- Approval ----

func SetTaskApprovalStatus(db *sql.DB, taskID int, status string) (*TaskV2, error) {
	return dbstore.SetTaskApprovalStatus(db, taskID, status)
}

func emitLeaseLifecycleEvent(db *sql.DB, kind string, lease *Lease, extra map[string]interface{}) error {
	return dbstore.EmitLeaseLifecycleEvent(db, kind, lease, extra)
}

