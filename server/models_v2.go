// API v2 Data Models — thin wrapper over internal/models.
package server

import (
	"database/sql"

	"github.com/onsomlem/cocopilot/internal/models"
)

// Type aliases for backward compatibility.
type Project = models.Project
type TreeNode = models.TreeNode
type FileChange = models.FileChange
type TaskStatus = models.TaskStatus
type TaskStatusV2 = models.TaskStatusV2
type TaskType = models.TaskType
type TaskV2 = models.TaskV2
type TaskDependency = models.TaskDependency
type RunStatus = models.RunStatus
type Run = models.Run
type RunDetail = models.RunDetail
type StepStatus = models.StepStatus
type RunStep = models.RunStep
type RunLog = models.RunLog
type Artifact = models.Artifact
type ToolInvocation = models.ToolInvocation
type Lease = models.Lease
type AgentStatus = models.AgentStatus
type Agent = models.Agent
type Event = models.Event
type Memory = models.Memory
type PolicyRule = models.PolicyRule
type Policy = models.Policy
type ContextPack = models.ContextPack
type RunSummary = models.RunSummary
type TaskTemplate = models.TaskTemplate
type RepoFile = models.RepoFile
type ListRepoFilesOpts = models.ListRepoFilesOpts
type ArtifactComment = models.ArtifactComment
type DashboardData = models.DashboardData
type DashboardTaskCounts = models.DashboardTaskCounts
type DashboardRun = models.DashboardRun
type DashboardRepoChange = models.DashboardRepoChange
type DashboardFailure = models.DashboardFailure
type DashboardAutoEvent = models.DashboardAutoEvent
type DashboardRecommendation = models.DashboardRecommendation
type DashboardStalledTask = models.DashboardStalledTask

// Constants re-exported.
const (
	StatusNotPicked  = models.StatusNotPicked
	StatusInProgress = models.StatusInProgress
	StatusComplete   = models.StatusComplete
	StatusFailed     = models.StatusFailed

	TaskStatusQueued      = models.TaskStatusQueued
	TaskStatusClaimed     = models.TaskStatusClaimed
	TaskStatusRunning     = models.TaskStatusRunning
	TaskStatusSucceeded   = models.TaskStatusSucceeded
	TaskStatusFailed      = models.TaskStatusFailed
	TaskStatusNeedsReview = models.TaskStatusNeedsReview
	TaskStatusCancelled   = models.TaskStatusCancelled

	TaskTypeAnalyze  = models.TaskTypeAnalyze
	TaskTypeModify   = models.TaskTypeModify
	TaskTypeTest     = models.TaskTypeTest
	TaskTypeReview   = models.TaskTypeReview
	TaskTypeDoc      = models.TaskTypeDoc
	TaskTypeRelease  = models.TaskTypeRelease
	TaskTypeRollback = models.TaskTypeRollback
	TaskTypePlan     = models.TaskTypePlan

	RunStatusRunning   = models.RunStatusRunning
	RunStatusSucceeded = models.RunStatusSucceeded
	RunStatusFailed    = models.RunStatusFailed
	RunStatusCancelled = models.RunStatusCancelled

	StepStatusStarted   = models.StepStatusStarted
	StepStatusSucceeded = models.StepStatusSucceeded
	StepStatusFailed    = models.StepStatusFailed

	AgentStatusOnline  = models.AgentStatusOnline
	AgentStatusOffline = models.AgentStatusOffline
	AgentStatusBusy    = models.AgentStatusBusy
	AgentStatusIdle    = models.AgentStatusIdle

	ApprovalPending  = models.ApprovalPending
	ApprovalApproved = models.ApprovalApproved
	ApprovalRejected = models.ApprovalRejected
)

// Function wrappers for backward compatibility.
func marshalJSON(v interface{}) (string, error)      { return models.MarshalJSON(v) }
func unmarshalJSON(s string, v interface{}) error     { return models.UnmarshalJSON(s, v) }
func nullString(s *string) sql.NullString             { return models.NullString(s) }
func ptrString(ns sql.NullString) *string             { return models.PtrString(ns) }
func ptrInt64(ni sql.NullInt64) *int64                { return models.PtrInt64(ni) }
func nowISO() string                                  { return models.NowISO() }
func TaskStatusBuckets() map[string][]TaskStatusV2    { return models.TaskStatusBuckets() }
func RunStatusBuckets() map[string][]RunStatus        { return models.RunStatusBuckets() }
func IsTaskQueued(status TaskStatusV2) bool            { return models.IsTaskQueued(status) }
func IsTaskActive(status TaskStatusV2) bool            { return models.IsTaskActive(status) }
func IsTaskTerminal(status TaskStatusV2) bool          { return models.IsTaskTerminal(status) }
func BucketForStatus(status TaskStatusV2) string       { return models.BucketForStatus(status) }
func IsRunActive(status RunStatus) bool                { return models.IsRunActive(status) }
func IsRunTerminal(status RunStatus) bool              { return models.IsRunTerminal(status) }
func IsLeaseActiveAt(expiresAt string, at string) bool { return models.IsLeaseActiveAt(expiresAt, at) }
func IsLeaseActive(expiresAt string) bool              { return models.IsLeaseActive(expiresAt) }
func TaskStatusIsSuccessful(status TaskStatusV2) bool  { return models.TaskStatusIsSuccessful(status) }
func TaskStatusIsFailed(status TaskStatusV2) bool      { return models.TaskStatusIsFailed(status) }
func RunStatusIsSuccessful(status RunStatus) bool      { return models.RunStatusIsSuccessful(status) }
func RunStatusIsFailed(status RunStatus) bool          { return models.RunStatusIsFailed(status) }
