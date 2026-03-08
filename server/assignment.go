// Canonical assignment service: unified claim, completion, and failure paths.
// All handlers (v1, v2, claim-by-id, claim-next) should go through these
// functions so that every claim creates a lease + run and every completion
// emits events + releases lease + updates run.
package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
        "sort"
        "strings"
        "time"

        "github.com/google/uuid"
)

// AssignmentService is the single canonical service for task claiming.
// All claim paths (v1 /task, v2 claim-by-id, v2 claim-next) must route through
// this service so that lease creation, run creation, status transitions, event
// emission, and context assembly happen consistently.
type AssignmentService struct {
	DB *sql.DB
}

// ClaimByID claims a specific task for an agent. It creates a lease, run,
// transitions the task to CLAIMED, emits events, and assembles context.
func (svc *AssignmentService) ClaimByID(taskID int, agentID, mode string) (*AssignmentEnvelope, error) {
	return ClaimTaskByID(svc.DB, taskID, agentID, mode)
}

// ClaimNext finds the next claimable task in a project and claims it.
// It uses a read-only transaction to find the task, then delegates to ClaimByID.
func (svc *AssignmentService) ClaimNext(projectID, agentID, mode string) (*AssignmentEnvelope, error) {
	now := nowISO()
	findTx, err := svc.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin find tx: %w", err)
	}
	foundID, found, qErr := claimNextTaskTx(findTx, projectID, now)
	_ = findTx.Rollback() // read-only
	if qErr != nil {
		return nil, fmt.Errorf("find next task: %w", qErr)
	}
	if !found {
		return nil, nil
	}
	return svc.ClaimByID(foundID, agentID, mode)
}

// AssembleContext gathers ranked context for a task assignment.
func (svc *AssignmentService) AssembleContext(task *TaskV2) *TaskContext {
	return assembleContext(svc.DB, task)
}

// CompletionContract describes what a claimed task expects as outputs.
// It is derived from the task type and allows agents to understand
// the format and required fields before starting work.
type CompletionContract struct {
	RequiredFields  []string `json:"required_fields,omitempty"`
	ExpectedOutputs []string `json:"expected_outputs,omitempty"`
	Notes           string   `json:"notes,omitempty"`
}

// AssignmentEnvelope is the canonical return type for a successful claim.
// Context is automatically assembled so callers get everything needed.
type AssignmentEnvelope struct {
	Task               *TaskV2             `json:"task"`
	Lease              *Lease              `json:"lease"`
	Run                *Run                `json:"run"`
	Project            *Project            `json:"project,omitempty"`
	PolicySnapshot     []Policy            `json:"policy_snapshot,omitempty"`
	CompletionContract *CompletionContract `json:"completion_contract,omitempty"`
	Context            *TaskContext        `json:"context,omitempty"`
}

// TaskContext holds automatically-assembled context for a task assignment.
// ArtifactSummary is a lightweight representation of a run artifact for context.
type ArtifactSummary struct {
	RunID      string `json:"run_id"`
	Kind       string `json:"kind"`
	StorageRef string `json:"storage_ref"`
	Size       *int64 `json:"size,omitempty"`
}

// ContextQuality summarises how well-populated the assembled context is.
// Score ranges from 0.0 (empty) to 1.0 (fully populated across all categories).
type ContextQuality struct {
	Score           float64          `json:"score"`
	CategoryCounts  map[string]int   `json:"category_counts"`
	BudgetUsedBytes int              `json:"budget_used_bytes"`
	BudgetMaxBytes  int              `json:"budget_max_bytes"`
}

type TaskContext struct {
	ContextPack        *ContextPack      `json:"context_pack,omitempty"`
	Memories           []Memory          `json:"memories,omitempty"`
	Policies           []Policy          `json:"policies,omitempty"`
	Dependencies       []TaskDependency  `json:"dependencies,omitempty"`
	RecentRunSummaries []RunSummary      `json:"recent_run_summaries,omitempty"`
	RepoFiles          []string          `json:"repo_files,omitempty"`
	ArtifactSummaries  []ArtifactSummary `json:"artifact_summaries,omitempty"`
	Quality            *ContextQuality   `json:"quality,omitempty"`
}

// assembleContext gathers context pack, memories, policies, dependencies,
// recent runs, and repo file state for a claimed task. Errors are logged
// but not fatal — partial context is better than no assignment.
//
// Hard caps per category enforce bounded context:
//   memories: 10, recent runs: 5, repo files: 20, policies: 5, dependencies: 10
// Total serialized payload is capped at 64KB; lowest-priority items are trimmed.
func assembleContext(database *sql.DB, task *TaskV2) *TaskContext {
	const (
		maxMemories     = 10
		maxRecentRuns   = 5
		maxRepoFiles    = 20
		maxPolicies     = 5
		maxDependencies = 10
		maxPayloadBytes = 64 * 1024 // 64KB total budget
	)

	ctx := &TaskContext{}

	// Latest context pack for this task.
	if pack, err := GetContextPackByTaskID(database, task.ID); err != nil {
		log.Printf("Warning: context assembly - context pack: %v", err)
	} else if pack != nil {
		ctx.ContextPack = pack
	}

	// Filtered project memories: prefer memories relevant to task type and
	// title keywords, plus recent failure memories (max 10 total).
	ctx.Memories = assembleFilteredMemories(database, task)
	if len(ctx.Memories) > maxMemories {
		ctx.Memories = ctx.Memories[:maxMemories]
	}

	// Active policies for the project, ranked by type (constraint > policy > others).
	enabled := true
	if policies, _, err := ListPoliciesByProject(database, task.ProjectID, &enabled, 100, 0, "created_at", "desc"); err != nil {
		log.Printf("Warning: context assembly - policies: %v", err)
	} else {
		// Rank: constraint/security policies first, then by recency.
		sort.SliceStable(policies, func(i, j int) bool {
			iScore := policyPriorityScore(policies[i])
			jScore := policyPriorityScore(policies[j])
			return iScore > jScore
		})
		if len(policies) > maxPolicies {
			policies = policies[:maxPolicies]
		}
		ctx.Policies = policies
	}

	// Task dependencies (capped).
	if deps, err := ListTaskDependencies(database, task.ID); err != nil {
		log.Printf("Warning: context assembly - dependencies: %v", err)
	} else {
		if len(deps) > maxDependencies {
			deps = deps[:maxDependencies]
		}
		ctx.Dependencies = deps
	}

	// Recent runs for this task as ranked lightweight summaries.
	// Ranking: failed/errored runs score higher (more informative for agents),
	// then by recency. Capped at maxRecentRuns.
        if runs, err := GetRunsByTaskID(database, task.ID); err != nil {
                log.Printf("Warning: context assembly - recent runs: %v", err)
        } else {
                summaries := make([]RunSummary, 0, len(runs))
                for _, r := range runs {
                        s := RunSummary{
                                ID:        r.ID,
                                Status:    string(r.Status),
                                StartedAt: r.StartedAt,
                        }
                        if r.FinishedAt != nil {
                                s.FinishedAt = *r.FinishedAt
                        }
                        if r.Error != nil {
                                s.ErrorMessage = *r.Error
                        }
                        // Populate FilesTouched from artifacts for this run.
                        if artifacts, aerr := GetArtifactsByRunID(database, r.ID); aerr == nil {
                                for _, a := range artifacts {
                                        if a.StorageRef != "" {
                                                s.FilesTouched = append(s.FilesTouched, a.StorageRef)
                                        }
                                }
                        }
                        summaries = append(summaries, s)
                }
                // Rank runs: failed/errored first (most informative), then by
                // start time descending. This ensures agents see relevant failures.
                sort.SliceStable(summaries, func(i, j int) bool {
                        si := runSummaryScore(summaries[i])
                        sj := runSummaryScore(summaries[j])
                        if si != sj {
                                return si > sj
                        }
                        return summaries[i].StartedAt > summaries[j].StartedAt
                })
                if len(summaries) > maxRecentRuns {
                        summaries = summaries[:maxRecentRuns]
                }
                ctx.RecentRunSummaries = summaries
        }

        // Repo files: keyword-ranked and capped to keep context focused.
        if repoFiles, err := ListRecentRepoFiles(database, task.ProjectID, 100); err != nil {
                log.Printf("Warning: context assembly - repo files: %v", err)
        } else {
                ctx.RepoFiles = rankRepoFilesByRelevance(repoFiles, task, maxRepoFiles)
        }

        // Artifact summaries from recent runs (lightweight, no full content).
        const maxArtifacts = 10
        for _, rs := range ctx.RecentRunSummaries {
                if len(ctx.ArtifactSummaries) >= maxArtifacts {
                        break
                }
                if arts, aerr := GetArtifactsByRunID(database, rs.ID); aerr == nil {
                        for _, a := range arts {
                                if len(ctx.ArtifactSummaries) >= maxArtifacts {
                                        break
                                }
                                ctx.ArtifactSummaries = append(ctx.ArtifactSummaries, ArtifactSummary{
                                        RunID:      a.RunID,
                                        Kind:       a.Kind,
                                        StorageRef: a.StorageRef,
                                        Size:       a.Size,
                                })
                        }
                }
        }

        // Enforce total payload budget: if serialized context exceeds maxPayloadBytes,
        // trim lowest-priority items (repo files first, then artifacts, then runs, then memories).
        enforceContextBudget(ctx, maxPayloadBytes)

        // Compute context quality score.
        ctx.Quality = computeContextQuality(ctx, maxPayloadBytes)

        return ctx
}

func assembleFilteredMemories(database *sql.DB, task *TaskV2) []Memory {
	seen := make(map[string]bool)
	var allMems []Memory

	addMemories := func(mems []Memory) {
		for _, m := range mems {
			if !seen[m.ID] {
				seen[m.ID] = true
				allMems = append(allMems, m)
			}
		}
	}

	// Gather memories from all relevant queries.
	if task.Type != "" {
		if mems, err := QueryMemories(database, task.ProjectID, string(task.Type), "", ""); err == nil {
			addMemories(mems)
		}
	}
	if mems, err := QueryMemories(database, task.ProjectID, "failure_pattern", "", ""); err == nil {
		addMemories(mems)
	}
	if mems, err := QueryMemories(database, task.ProjectID, "constraint", "", ""); err == nil {
		addMemories(mems)
	}
	if mems, err := QueryMemories(database, task.ProjectID, "policy", "", ""); err == nil {
		addMemories(mems)
	}
	if mems, err := QueryMemories(database, task.ProjectID, "convention", "", ""); err == nil {
		addMemories(mems)
	}
	titleKeyword := extractTitleKeyword(task.Title)
	if titleKeyword != "" {
		if mems, err := QueryMemories(database, task.ProjectID, "", "", titleKeyword); err == nil {
			addMemories(mems)
		}
	}
	// Fallback: generic memories.
	if mems, err := QueryMemories(database, task.ProjectID, "", "", ""); err == nil {
		addMemories(mems)
	}

	// Determine if the task has had prior failures.
	hasPriorFailures := false
	for _, m := range allMems {
		if m.Scope == "failure_pattern" {
			hasPriorFailures = true
			break
		}
	}

	// Mandatory memories (always included regardless of budget).
	var mandatory []Memory
	var candidates []Memory
	for _, m := range allMems {
		if m.Scope == "constraint" || m.Scope == "policy" {
			mandatory = append(mandatory, m)
		} else {
			candidates = append(candidates, m)
		}
	}

	// Score candidates by: relevance (type/keywords/files), recency,
	// confidence (from memory value), and success (fix_pattern/convention rank higher).
	titleKw := extractTitleKeyword(task.Title)
	type scored struct {
		mem   Memory
		score int
	}
	var scoredList []scored
	// Dedupe by content hash: skip memories with identical content.
	seenHashes := make(map[string]bool)

	for _, m := range candidates {
		// Content hash dedup: skip if we've already seen this content.
		if h, ok := m.Value["content_hash"].(string); ok && h != "" {
			if seenHashes[h] {
				continue
			}
			seenHashes[h] = true
		}

		s := 0
		// +3 if key contains a word from task title (relevance).
		if titleKw != "" && strings.Contains(strings.ToLower(m.Key), titleKw) {
			s += 3
		}
		// +2 if failure_pattern and task had prior failures (relevance).
		if m.Scope == "failure_pattern" && hasPriorFailures {
			s += 2
		}
		// +1 if created within last 7 days (recency).
		if m.CreatedAt != "" && m.CreatedAt > sevenDaysOffsetISO() {
			s += 1
		}
		// +2 if convention (success-type boost).
		if m.Scope == "convention" {
			s += 2
		}
		// +2 if fix_pattern (proven fixes are valuable).
		if m.Scope == "fix_pattern" {
			s += 2
		}
		// Confidence boost: memories with stored confidence score.
		if conf, ok := m.Value["confidence"].(float64); ok {
			if conf >= 0.8 {
				s += 3
			} else if conf >= 0.6 {
				s += 2
			} else if conf >= 0.4 {
				s += 1
			}
		}
		// File relevance: boost if memory files overlap with task type files.
		if files, ok := m.Value["files"].([]interface{}); ok && len(files) > 0 {
			s += 1
		}
		scoredList = append(scoredList, scored{m, s})
	}

	// Sort by score descending.
	sort.SliceStable(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	const memBudget = 10
	const maxByteBudget = 32 * 1024 // 32KB

	result := make([]Memory, 0, memBudget)
	result = append(result, mandatory...)

	totalBytes := 0
	for _, m := range mandatory {
		if b, err := json.Marshal(m); err == nil {
			totalBytes += len(b)
		}
	}

	for _, s := range scoredList {
		if len(result) >= memBudget {
			break
		}
		b, err := json.Marshal(s.mem)
		if err == nil && totalBytes+len(b) > maxByteBudget {
			continue
		}
		result = append(result, s.mem)
		if err == nil {
			totalBytes += len(b)
		}
	}

	return result
}

// enforceContextBudget checks total serialized size and trims lowest-priority
// categories until the payload fits within budget. Trim order: repo files,
// then artifacts, then recent runs, then memories (keeping mandatory constraint/policy memories).
func enforceContextBudget(ctx *TaskContext, maxBytes int) {
	for {
		b, err := json.Marshal(ctx)
		if err != nil || len(b) <= maxBytes {
			return
		}

		// Trim repo files first (lowest priority).
		if len(ctx.RepoFiles) > 1 {
			ctx.RepoFiles = ctx.RepoFiles[:len(ctx.RepoFiles)/2]
			continue
		} else if len(ctx.RepoFiles) == 1 {
			ctx.RepoFiles = nil
			continue
		}

		// Then trim artifact summaries.
		if len(ctx.ArtifactSummaries) > 1 {
			ctx.ArtifactSummaries = ctx.ArtifactSummaries[:len(ctx.ArtifactSummaries)/2]
			continue
		} else if len(ctx.ArtifactSummaries) == 1 {
			ctx.ArtifactSummaries = nil
			continue
		}

		// Then trim recent runs.
		if len(ctx.RecentRunSummaries) > 1 {
			ctx.RecentRunSummaries = ctx.RecentRunSummaries[:len(ctx.RecentRunSummaries)/2]
			continue
		} else if len(ctx.RecentRunSummaries) == 1 {
			ctx.RecentRunSummaries = nil
			continue
		}

		// Then trim memories (keeping at least mandatory ones).
		if len(ctx.Memories) > 2 {
			ctx.Memories = ctx.Memories[:len(ctx.Memories)/2]
			continue
		}

		// Nothing left to trim.
		return
	}
}

// computeContextQuality scores the assembled context across all categories.
// Each populated category contributes a weighted fraction to the total score.
func computeContextQuality(ctx *TaskContext, maxBytes int) *ContextQuality {
	// Category weights (total = 1.0).
	const (
		wContextPack = 0.15
		wMemories    = 0.25
		wPolicies    = 0.10
		wDeps        = 0.10
		wRuns        = 0.15
		wRepoFiles   = 0.15
		wArtifacts   = 0.10
	)

	score := 0.0
	counts := map[string]int{}

	if ctx.ContextPack != nil {
		score += wContextPack
		counts["context_pack"] = 1
	}
	if n := len(ctx.Memories); n > 0 {
		score += wMemories * clamp(float64(n)/3.0, 1.0)
		counts["memories"] = n
	}
	if n := len(ctx.Policies); n > 0 {
		score += wPolicies * clamp(float64(n)/2.0, 1.0)
		counts["policies"] = n
	}
	if n := len(ctx.Dependencies); n > 0 {
		score += wDeps
		counts["dependencies"] = n
	}
	if n := len(ctx.RecentRunSummaries); n > 0 {
		score += wRuns * clamp(float64(n)/2.0, 1.0)
		counts["recent_runs"] = n
	}
	if n := len(ctx.RepoFiles); n > 0 {
		score += wRepoFiles * clamp(float64(n)/5.0, 1.0)
		counts["repo_files"] = n
	}
	if n := len(ctx.ArtifactSummaries); n > 0 {
		score += wArtifacts * clamp(float64(n)/2.0, 1.0)
		counts["artifacts"] = n
	}

	usedBytes := 0
	if b, err := json.Marshal(ctx); err == nil {
		usedBytes = len(b)
	}

	return &ContextQuality{
		Score:           score,
		CategoryCounts:  counts,
		BudgetUsedBytes: usedBytes,
		BudgetMaxBytes:  maxBytes,
	}
}

// clamp returns min(v, max).
func clamp(v, max float64) float64 {
	if v > max {
		return max
	}
	return v
}

// policyPriorityScore ranks policies for context inclusion.
// Constraint/security policies rank highest, then general policies.
func policyPriorityScore(p Policy) int {
	score := 0
	nameLower := strings.ToLower(p.Name)
	if strings.Contains(nameLower, "constraint") || strings.Contains(nameLower, "security") {
		score += 3
	}
	if strings.Contains(nameLower, "required") || strings.Contains(nameLower, "mandatory") {
		score += 2
	}
	// Boost policies with more rules (more substance).
	if len(p.Rules) > 0 {
		score += 1
	}
	return score
}

// runSummaryScore ranks run summaries for context inclusion.
// Failed/errored runs are most informative for agents (+3),
// completed runs with files touched are moderately useful (+1).
func runSummaryScore(s RunSummary) int {
	score := 0
	status := strings.ToUpper(s.Status)
	if status == "FAILED" || status == "ERROR" {
		score += 3
	}
	if s.ErrorMessage != "" {
		score += 1
	}
	if len(s.FilesTouched) > 0 {
		score += 1
	}
	return score
}

// sevenDaysOffsetISO returns the ISO timestamp for 7 days ago (for budget scoring).
// rankRepoFilesByRelevance returns at most cap file paths from repoFiles,
// ranked so that files whose path contains a keyword from the task title or type
// appear first; ties preserve recency (ListRecentRepoFiles already orders by
// last_modified DESC). This keeps injected context focused rather than noisy.
func rankRepoFilesByRelevance(files []RepoFile, task *TaskV2, cap int) []string {
        // Build a small keyword set from the task title and type.
        keywords := map[string]bool{}
        if task.Title != nil {
                for _, w := range strings.Fields(strings.ToLower(*task.Title)) {
                        kw := strings.Trim(w, ".,;:!?()[]{}\"'")
                        if len(kw) > 3 {
                                keywords[kw] = true
                        }
                }
        }
        keywords[strings.ToLower(string(task.Type))] = true

        type scored struct {
                path  string
                score int
        }
        ss := make([]scored, 0, len(files))
        for _, f := range files {
                s := 0
                lower := strings.ToLower(f.Path)
                for kw := range keywords {
                        if strings.Contains(lower, kw) {
                                s++
                        }
                }
                ss = append(ss, scored{path: f.Path, score: s})
        }
        // Stable sort: high score first (recency already baked in by input order).
        sort.SliceStable(ss, func(i, j int) bool { return ss[i].score > ss[j].score })
        if cap > len(ss) {
                cap = len(ss)
        }
        paths := make([]string, 0, cap)
        for _, s := range ss[:cap] {
                paths = append(paths, s.path)
        }
        return paths
}

func sevenDaysOffsetISO() string {
	t := time.Now().UTC().AddDate(0, 0, -7)
	return t.Format("2006-01-02T15:04:05.000000Z")
}

// extractTitleKeyword returns the first significant word from a task title,
// for use as a memory search keyword.
func extractTitleKeyword(title *string) string {
	if title == nil || *title == "" {
		return ""
	}
	// Skip common stop words and use the first meaningful token.
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "with": true, "by": true, "from": true,
	}
	words := strings.Fields(strings.ToLower(*title))
	for _, w := range words {
		// Strip punctuation
		clean := strings.Trim(w, ".,;:!?()[]{}\"'")
		if len(clean) > 3 && !stopWords[clean] {
			return clean
		}
	}
	if len(words) > 0 {
		return strings.Trim(words[0], ".,;:!?()[]{}\"'")
	}
	return ""
}

// buildCompletionContract returns a completion contract for a task based on its type.
// The contract tells the agent what fields and outputs are expected.
func buildCompletionContract(task *TaskV2) *CompletionContract {
	switch task.Type {
	case TaskTypeModify:
		return &CompletionContract{
			RequiredFields:  []string{"summary"},
			ExpectedOutputs: []string{"changes_made", "files_touched"},
			Notes:           "Describe changes made and list all modified files.",
		}
	case TaskTypeAnalyze:
		return &CompletionContract{
			RequiredFields:  []string{"summary"},
			ExpectedOutputs: []string{"findings", "risks", "next_tasks"},
			Notes:           "Provide analysis findings and propose follow-up tasks if needed.",
		}
	case TaskTypeTest:
		return &CompletionContract{
			RequiredFields:  []string{"summary"},
			ExpectedOutputs: []string{"tests_run", "changes_made"},
			Notes:           "List tests executed and any code changes required to make them pass.",
		}
	case TaskTypeReview:
		return &CompletionContract{
			RequiredFields:  []string{"summary"},
			ExpectedOutputs: []string{"findings", "risks"},
			Notes:           "Summarise review findings and highlight any risks discovered.",
		}
	default:
		return &CompletionContract{
			RequiredFields:  []string{"summary"},
			ExpectedOutputs: []string{"changes_made"},
			Notes:           "Provide a summary of work completed.",
		}
	}
}
// task status, and creates a run — all inside a single transaction.
func ClaimTaskByID(database *sql.DB, taskID int, agentID, mode string) (*AssignmentEnvelope, error) {
	if mode == "" {
		mode = "exclusive"
	}

	tx, err := database.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Create lease (tx-aware, handles conflict detection).
	lease, err := CreateLeaseTx(tx, taskID, agentID, mode)
	if err != nil {
		return nil, fmt.Errorf("lease: %w", err)
	}

	// 2. Update task status to CLAIMED.
	now := nowISO()
	_, err = tx.Exec(
		"UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ?",
		StatusInProgress, TaskStatusClaimed, now, taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("status update: %w", err)
	}

	// 3. Create run for this assignment.
	run := &Run{
		ID:        "run_" + uuid.New().String(),
		TaskID:    taskID,
		AgentID:   agentID,
		Status:    RunStatusRunning,
		StartedAt: nowISO(),
	}
	_, err = tx.Exec(
		"INSERT INTO runs (id, task_id, agent_id, status, started_at) VALUES (?, ?, ?, ?, ?)",
		run.ID, run.TaskID, run.AgentID, run.Status, run.StartedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Fetch full task after commit.
	task, err := GetTaskV2(database, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch task: %w", err)
	}

	// Emit task.claimed event.
	CreateEvent(database, task.ProjectID, "task.claimed", "task", fmt.Sprintf("%d", taskID), map[string]interface{}{
		"task_id":  taskID,
		"agent_id": agentID,
		"lease_id": lease.ID,
		"run_id":   run.ID,
	})

	// Automatic context assembly (AC4).
	ctx := assembleContext(database, task)

	// Fetch project info for envelope.
	project, err := GetProject(database, task.ProjectID)
	if err != nil {
		log.Printf("Warning: failed to fetch project %s for envelope: %v", task.ProjectID, err)
	}

	// Build completion contract based on task type.
	contract := buildCompletionContract(task)

	// Fetch effective policy snapshot (enabled policies for this project).
	var policySnapshot []Policy
	enabled := true
	if policies, _, pErr := ListPoliciesByProject(database, task.ProjectID, &enabled, 10, 0, "created_at", "desc"); pErr == nil {
		policySnapshot = policies
	} else {
		log.Printf("Warning: failed to fetch policy snapshot for task %d: %v", taskID, pErr)
	}

	return &AssignmentEnvelope{Task: task, Lease: lease, Run: run, Project: project, PolicySnapshot: policySnapshot, CompletionContract: contract, Context: ctx}, nil
}

// CompleteTask is the canonical completion path. It:
//  1. Updates task status to COMPLETE/SUCCEEDED
//  2. Updates the latest run to SUCCEEDED
//  3. Releases the lease
//  4. Emits a task.completed event (triggers automation)
//  5. Broadcasts SSE
//
// Returns the updated task.
func CompleteTask(database *sql.DB, taskID int, output *string) (*TaskV2, error) {
	now := nowISO()

	// 1. Update task status (TOCTOU-safe: AND status != COMPLETE).
	result, err := database.Exec(
		"UPDATE tasks SET output = ?, status = ?, status_v2 = ?, updated_at = ? WHERE id = ? AND status != ?",
		nullString(output), StatusComplete, TaskStatusSucceeded, now, taskID, StatusComplete,
	)
	if err != nil {
		return nil, fmt.Errorf("task status update: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("task %d already completed", taskID)
	}

	// 2. Update latest run.
	latestRun, err := GetLatestRunByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get run for task %d: %v", taskID, err)
	} else if latestRun != nil {
		if err := UpdateRunStatus(database, latestRun.ID, RunStatusSucceeded, nil); err != nil {
			log.Printf("Warning: failed to update run %s: %v", latestRun.ID, err)
		}
	}

	// 3. Release lease.
	lease, err := GetLeaseByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get lease for task %d: %v", taskID, err)
	} else if lease != nil {
		if _, _, err := ReleaseLease(database, lease.ID, "task_completed"); err != nil {
			log.Printf("Warning: failed to release lease for task %d: %v", taskID, err)
		}
	}

	// 4. Emit task.completed event.
	task, err := GetTaskV2(database, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch task: %w", err)
	}

	payload := map[string]interface{}{
		"task_id":   taskID,
		"status_v1": string(StatusComplete),
		"status_v2": string(TaskStatusSucceeded),
	}
	if output != nil {
		payload["output"] = *output
	}
	if _, err := CreateEvent(database, task.ProjectID, "task.completed", "task", fmt.Sprintf("%d", taskID), payload); err != nil {
		log.Printf("Warning: failed to emit task.completed event for task %d: %v", taskID, err)
	}

	// 5. SSE broadcast.
	go broadcastUpdate(v1EventTypeTasks)

	return task, nil
}

// FailTask is the canonical failure path. It:
//  1. Updates task status to FAILED
//  2. Updates the latest run to FAILED
//  3. Releases the lease
//  4. Emits a task.failed event
//  5. Broadcasts SSE
func FailTask(database *sql.DB, taskID int, errMsg string) (*TaskV2, error) {
	now := nowISO()

	_, err := database.Exec(
		"UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ? AND status != ? AND status != ?",
		StatusFailed, TaskStatusFailed, now, taskID, StatusComplete, StatusFailed,
	)
	if err != nil {
		return nil, fmt.Errorf("task status update: %w", err)
	}

	latestRun, err := GetLatestRunByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get run for task %d: %v", taskID, err)
	} else if latestRun != nil {
		if err := UpdateRunStatus(database, latestRun.ID, RunStatusFailed, &errMsg); err != nil {
			log.Printf("Warning: failed to update run %s: %v", latestRun.ID, err)
		}
	}

	lease, err := GetLeaseByTaskID(database, taskID)
	if err != nil {
		log.Printf("Warning: failed to get lease for task %d: %v", taskID, err)
	} else if lease != nil {
		if _, _, err := ReleaseLease(database, lease.ID, "task_failed"); err != nil {
			log.Printf("Warning: failed to release lease for task %d: %v", taskID, err)
		}
	}

	task, err := GetTaskV2(database, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch task: %w", err)
	}

	payload := map[string]interface{}{
		"task_id":   taskID,
		"status_v1": string(StatusFailed),
		"status_v2": string(TaskStatusFailed),
		"error":     errMsg,
	}
	if _, err := CreateEvent(database, task.ProjectID, "task.failed", "task", fmt.Sprintf("%d", taskID), payload); err != nil {
		log.Printf("Warning: failed to emit task.failed event for task %d: %v", taskID, err)
	}

	go broadcastUpdate(v1EventTypeTasks)

	return task, nil
}
