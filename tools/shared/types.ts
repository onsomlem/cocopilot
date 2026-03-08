/**
 * Shared TypeScript types for Cocopilot v2 API.
 * Generated from Go structs in models_v2.go and assignment.go.
 * Used by both cocopilot-mcp and cocopilot-vsix.
 */

// ── Status Enums ─────────────────────────────────────────────────

export type TaskStatusV2 =
  | "QUEUED"
  | "CLAIMED"
  | "RUNNING"
  | "SUCCEEDED"
  | "FAILED"
  | "NEEDS_REVIEW"
  | "CANCELLED";

export type TaskType =
  | "ANALYZE"
  | "MODIFY"
  | "TEST"
  | "REVIEW"
  | "DOC"
  | "RELEASE"
  | "ROLLBACK"
  | "PLAN";

export type RunStatus = "RUNNING" | "SUCCEEDED" | "FAILED" | "CANCELLED";

export type StepStatus = "STARTED" | "SUCCEEDED" | "FAILED";

export type AgentStatus = "ONLINE" | "OFFLINE" | "BUSY" | "IDLE";

// v1 compat status
export type TaskStatusV1 = "AVAILABLE" | "CLAIMED" | "COMPLETE" | "FAILED";

// ── Core Entities ────────────────────────────────────────────────

export interface Project {
  id: string;
  name: string;
  workdir: string;
  settings?: Record<string, unknown>;
  created_at: string;
}

export interface TaskV2 {
  id: number;
  project_id: string;
  title?: string;
  instructions: string;
  type: TaskType;
  priority: number;
  tags?: string[];
  status_v1: TaskStatusV1;
  status_v2: TaskStatusV2;
  parent_task_id?: number;
  output?: string;
  created_at: string;
  updated_at?: string;
  automation_depth: number;
}

export interface TaskDependency {
  task_id: number;
  depends_on_task_id: number;
}

export interface Run {
  id: string;
  task_id: number;
  agent_id: string;
  status: RunStatus;
  started_at: string;
  finished_at?: string;
  error?: string;
}

export interface RunStep {
  id: string;
  run_id: string;
  name: string;
  status: StepStatus;
  details?: Record<string, unknown>;
  created_at: string;
}

export interface RunLog {
  id: number;
  run_id: string;
  stream: string;
  chunk: string;
  ts: string;
}

export interface Artifact {
  id: string;
  run_id: string;
  kind: string;
  storage_ref: string;
  sha256?: string;
  size?: number;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface ToolInvocation {
  id: string;
  run_id: string;
  tool_name: string;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  started_at: string;
  finished_at?: string;
}

export interface RunDetail extends Run {
  steps?: RunStep[];
  logs?: RunLog[];
  artifacts?: Artifact[];
  tool_invocations?: ToolInvocation[];
}

export interface RunSummary {
  id: string;
  status: string;
  started_at: string;
  finished_at?: string;
  summary?: string;
  error_message?: string;
  files_touched?: string[];
}

export interface Lease {
  id: string;
  task_id: number;
  agent_id: string;
  mode: string;
  created_at: string;
  expires_at: string;
}

export interface Agent {
  id: string;
  name: string;
  capabilities?: string[];
  metadata?: Record<string, unknown>;
  status: AgentStatus;
  last_seen?: string;
  registered_at: string;
}

export interface Event {
  id: string;
  project_id: string;
  kind: string;
  entity_type: string;
  entity_id: string;
  created_at: string;
  payload?: Record<string, unknown>;
}

export interface Memory {
  id: string;
  project_id: string;
  scope: string;
  key: string;
  value: Record<string, unknown>;
  source_refs?: string[];
  created_at: string;
  updated_at: string;
}

export interface Policy {
  id: string;
  project_id: string;
  name: string;
  description?: string;
  rules: Record<string, unknown>[];
  enabled: boolean;
  created_at: string;
}

export interface ContextPack {
  id: string;
  project_id: string;
  task_id: number;
  summary: string;
  contents: Record<string, unknown>;
  created_at: string;
  stale?: boolean;
}

export interface RepoFile {
  id: string;
  project_id: string;
  path: string;
  content_hash?: string;
  size_bytes?: number;
  language?: string;
  last_modified?: string;
  created_at: string;
  updated_at: string;
  metadata?: Record<string, unknown>;
}

// ── Assignment / Claim Envelope ──────────────────────────────────

export interface CompletionContract {
  required_fields?: string[];
  expected_outputs?: string[];
  notes?: string;
}

export interface TaskContext {
  context_pack?: ContextPack;
  memories?: Memory[];
  policies?: Policy[];
  dependencies?: TaskDependency[];
  recent_run_summaries?: RunSummary[];
  repo_files?: string[];
}

export interface AssignmentEnvelope {
  task: TaskV2;
  lease: Lease;
  run: Run;
  project?: Project;
  completion_contract?: CompletionContract;
  context?: TaskContext;
}

// ── Tree / Changes ───────────────────────────────────────────────

export interface TreeNode {
  path: string;
  kind: string;
  size?: number;
  children?: TreeNode[];
}

export interface FileChange {
  path: string;
  kind: string;
  sha256?: string;
  ts: string;
}

// ── API Error Response ───────────────────────────────────────────

export interface V2Error {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}

// ── List Responses ───────────────────────────────────────────────

export interface ListProjectsResponse {
  projects: Project[];
}

export interface ListTasksResponse {
  tasks: TaskV2[];
  total?: number;
}

export interface ListEventsResponse {
  events: Event[];
}

export interface ListAgentsResponse {
  agents: Agent[];
}

export interface ListMemoriesResponse {
  memories: Memory[];
}

export interface ListPoliciesResponse {
  policies: Policy[];
}

// ── Structured Finalization ──────────────────────────────────────

export interface CompleteTaskPayload {
  output?: string;
  summary?: {
    what_changed?: string;
    files_touched?: string[];
    tests_passed?: boolean;
    notes?: string;
  };
  result?: Record<string, unknown>;
}

export interface FailTaskPayload {
  error?: string;
  output?: string;
  result?: Record<string, unknown>;
}

// ── Dashboard Projection ─────────────────────────────────────────

export interface DashboardTaskCounts {
  queued: number;
  in_progress: number;
  completed: number;
  failed: number;
}

export interface DashboardRun {
  run_id: string;
  task_title: string;
  status: string;
  started_at: string;
}

export interface DashboardRepoChange {
  path: string;
  updated_at: string;
}

export interface DashboardFailure {
  task_id: number;
  title?: string;
  error?: string;
  updated_at: string;
}

export interface DashboardAutoEvent {
  kind: string;
  created_at: string;
}

export interface DashboardRecommendation {
  task_id: number;
  title?: string;
  priority: number;
  type?: string;
  why: string;
}

export interface DashboardStalledTask {
  task_id: number;
  title?: string;
  status: string;
  agent_id?: string;
  claimed_at: string;
  reason: string;
}

export interface DashboardData {
  project_id: string;
  task_counts: DashboardTaskCounts;
  active_runs: DashboardRun[];
  recent_changes: DashboardRepoChange[];
  recent_failures: DashboardFailure[];
  automation_events: DashboardAutoEvent[];
  recommendations: DashboardRecommendation[];
  stalled_tasks: DashboardStalledTask[];
}
