# Task ID 161: Remediation Plan & Implementation Status

**Date**: March 5, 2026  
**Status**: ANALYSIS COMPLETE, IMPLEMENTATION STARTING

## Current State Assessment

### What's Already Implemented ✅

1. **AssignmentEnvelope** exists in `assignment.go` with required fields:
   - `Task`, `Lease`, `Run`, `Context`

2. **Core Assignment Functions** implemented:
   - `ClaimTaskByID()` - canonical claim path with atomic lease + run creation
   - `CompleteTask()` - canonical completion with status update + lease release
   - `assembleContext()` - automatic context assembly on claim
   - `TaskContext` structure includes ContextPack, Memories, Policies, Dependencies

3. **v2 Handlers Integration**:
   - `v2TaskClaimHandler` uses `ClaimTaskByID()`
   - `v2TaskCompleteHandler` implements completion logic
   - `v2ProjectTasksClaimNextHandler` uses `claimNextTaskTx()` 

4. **v1 Integration** (Partial):
   - `getTaskHandler` uses `ClaimTaskByID()` - wrapper functioning

5. **Database Schema**:
   - Tasks, Runs, Leases, Events all exist with foreign keys
   - Status enums defined in models_v2.go

### Gaps Remaining (Blocking Compliance)

#### Critical (Phase 2 - Canonical Services)
1. **FinalizationService NOT centralized**
   - `CompleteTask()` exists but NOT consistently used across all completion paths
   - v1 `/save` endpoint has separate completion logic
   - Failure/cancellation paths have scattered implementations
   - Need single source of truth for ALL state transitions on completion

2. **AssignmentService Inconsistency**
   - v1 and v2 can still take different code paths despite both using `ClaimTaskByID()`
   - v1 `/task` endpoint returns plain text; v2 returns JSON
   - Context assembly happens in `ClaimTaskByID()` but response formatting differs
   - Need unified envelope contract at HTTP boundary

3. **v1 Endpoint Testing**
   - No proof that v1 and v2 produce identical state for same operations
   - v1 uses text response format; doesn't expose full envelope to caller
   - Need regression tests proving wrapper equivalence

#### High (Phase 1 - State Canonicalization)
4. **Status Helpers Not Centralized**
   - TaskStatusV2 and RunStatus enums exist but scattered
   - Policy/metrics counting likely hardcoded against string values
   - `ActiveLease` definition not centralized (need `expires_at > now` helper)
   - Need single source of truth functions

5. **Memory Extraction NOT Automated**
   - `CompleteTask()` doesn't extract structured memory from run summary
   - No memory ranking/redaction pipeline
   - Missing integration with context assembly feedback loop

6. **Repo Intelligence NOT Driving Context/Memory**
   - Repo scanning exists but doesn't auto-trigger context invalidation
   - No follow-up task proposal on repo changes
   - Context packs not auto-refreshed on project changes

#### Medium (Phase 4 - Synergy)
7. **Automation Pipeline Incomplete**
   - Current triggers: task.started, task.completed, lease.expired
   - Missing: repo.changed, run.failed, context.invalidated, policy.denied
   - No automation workers for context refresh, repo summarization, projection updates

8. **Contract Drift Risk**
   - No OpenAPI source of truth
   - MCP tool schemas hand-maintained
   - VSIX docs potentially stale
   - No CI drift checks

#### Operational
9. **Hygiene/Packaging** (Phase 0)
   - Runtime DB artifacts may not be properly .gitignored
   - CI 'clean tree' check needed
   - Test cleanup may not be deterministic

---

## Prioritized Remediation Plan (48-72 hours)

### PRIORITY 1: Canonical Finalization Service (High Impact)
**Impact**: CRITICAL | **Effort**: 6-8 hours | **Blocks**: Everything else

**What**: Centralize ALL completion/failure/cancellation into single `FinalizationService`

**Acceptance**:
- All task completion routes (v1 `/save`, v2 `POST /task/:id/complete`, automation) call single service
- Service atomically: updates run/task status, releases lease, writes summary, extracts memory, emits events
- v1 and v2 completions produce identical DB state
- Regression test: `TestCompletionEquivalence` proves v1 and v2 same-state

**Implementation Steps**:
1. Create `finalization.go` with:
   ```go
   func FailTask(db *sql.DB, taskID int, errMsg string, ...) (*TaskV2, error)
   func CancelTask(db *sql.DB, taskID int, ...) (*TaskV2, error)
   // CompleteTask already exists but may need refactoring
   ```

2. Refactor `CompleteTask()` to extract structured summary
   - Input: output string or `completionResult` struct
   - Output: structured `RunSummary` with changes/files/risks saved to DB

3. Update all completion callers to use centralized service:
   - `v2TaskCompleteHandler` → calls service
   - v1 `/save` endpoint → convert to wrapper calling service
   - Automation completion hooks → use service

4. Add regression tests:
   - Test v1 `/save` vs v2 `/complete` produce same state
   - Test structured summary extracted and stored

---

### PRIORITY 2: Status Canonicalization Helpers (Medium Impact)
**Impact**: HIGH | **Effort**: 3-4 hours | **Blocks**: Observability fixes

**What**: Create centralized status helpers and eliminate hardcoded string checks

**Implementation**:
1. In `models_v2.go`, add helper file or package-level functions:
   ```go
   func IsTaskActive(status TaskStatusV2) bool
   func IsRunActive(status RunStatus) bool
   func IsLeaseActive(now string, expiresAt *string) bool
   func TaskStatusBuckets() map[string][]TaskStatusV2  // for metrics
   func RunStatusBuckets() map[string][]RunStatus
   ```

2. Refactor policy/metrics queries to use helpers instead of hardcoded SQL filters

3. Update lease queries to use `IsLeaseActive()` helper consistently

4. Add tests for helper functions and metrics accuracy

---

### PRIORITY 3: Memory Extraction Pipeline (Medium Impact)
**Impact**: HIGH | **Effort**: 4-6 hours | **Enables**: Learning loop

**What**: Automatically extract + rank memory from run completions and feed context engine

**Implementation**:
1. Add to `CompleteTask()` or separate `ExtractMemory()` function:
   - Parse `RunSummary` (changes, files, commands, tests, risks)
   - Create/update `Memory` records with:
     - scope: changes/files/patterns identified
     - confidence: based on test success/failure
     - tags: auto-derived from payload
     - recency bonus: creation timestamp

2. Update context assembly (`assembleContext()`) to rank memory hits:
   - Recent memories (< 1 week) score higher
   - High-confidence memories score higher
   - Project-scoped memories score higher
   - Include top-N (5-10) ranked memories in context response

3. Add memory redaction:
   - Scan memory.content for common secret patterns
   - Flag/hide sensitive memories from non-admin users

4. Tests:
   - Test memory extraction from completion payload
   - Test ranking/scoring logic
   - Test memory appears in future claim contexts

---

### PRIORITY 4: Automated Synergy Events & Workers (High Impact)
**Impact**: CRITICAL | **Effort**: 8-10 hours | **Enables**: Self-coordinating system

**What**: Expand event triggers and add missing automation workers

**Implementation**:
1. Emit new event types from core lifecycle:
   - `context.invalidated` - when repo changes detected
   - `run.failed` - when run transitions to FAILED
   - `lease.expired` - when lease timeout occurs
   - `policy.denied` - when policy blocks action

2. Add automation workers:
   - **ContextRefresher**: on `context.invalidated`, refresh context packs for related tasks
   - **RepoSummarizer**: on `repo.changed`, scan/sync repo, update context
   - **FailureSummarizer**: on `run.failed`, create follow-up analysis task
   - **ProjectionUpdater**: on task completion, check for unlocked dependencies, propose next tasks

3. Route all workers through automation engine with rate limits + dedup

4. Tests:
   - Event flow: repo.changed → context.invalidated → context refresher runs
   - Memory loop: task completion → memory extracted → appears in next claim
   - Dependency unlock: parent completion → child proposed task

---

### PRIORITY 5: Contract Integrity (Medium Impact)
**Impact**: HIGH | **Effort**: 12+ hours (iterative)

**What**: Define contract source of truth and eliminate drift risk

**Implementation**:
1. Create OpenAPI 3.1 spec in `docs/api/openapi.yaml`:
   - Define all v2 endpoints
   - Define AssignmentEnvelope, TaskContext, ErrorResponse
   - Define all entity schemas (Task, Run, Lease, Event, Memory, Policy)
   - Define automation rule format, etc.

2. Generate TypeScript client:
   - Use `openapi-typescript` or similar
   - Publish to npm as `@cocopilot/client`

3. Migrate VSIX:
   - Replace hand-maintained service.ts with generated client
   - Update all handlers to use generated types

4. Generate MCP tool manifest:
   - Pull schema from OpenAPI spec
   - Generate JSON for client consumption

5. Add CI check:
   - Run `openapi-generator` on PR
   - Fail if generated client differs from tracked version
   - Fail if MCP manifest differs

---

### PRIORITY 6: Repo Intelligence Pipeline (High Impact)
**Impact**: CRITICAL | **Effort**: 10-12 hours | **Enables**: Proactive planning

**What**: Auto-trigger context/memory/task creation on repo changes

**Current**: Repo scanning exists (`scanner.go`), context packs persist

**Implementation**:
1. On repo.changed event:
   - Re-scan project workdir
   - Compare with previous scan
   - Emit `repo.changed` with diff
   - Trigger `context.invalidated` for tasks in modified scope

2. On context.invalidated:
   - Update context pack with new repo state
   - Increment version
   - Broadcast to active agents

3. On significant changes:
   - Propose follow-up tasks (e.g., "review changes in X", "test modified Y")
   - Use automation rules to decide if auto-created or queued manual

4. Integration with memory:
   - Repo diffs inform memory extraction
   - High-impact files trigger memory recording

---

## Implementation Order (48-72 Hour Sprint)

### Hours 0-8: Priority 1 - Finalization Service
- [ ] Create `finalization.go` with FailTask/CancelTask
- [ ] Refactor CompleteTask to extract RunSummary
- [ ] Update v1 `/save` to use centralized service
- [ ] Add regression tests
- [ ] Validate DB state equivalence v1/v2

### Hours 8-12: Priority 2 - Status Helpers
- [ ] Create status helper functions
- [ ] Refactor policy/metrics to use helpers
- [ ] Update lease queries
- [ ] Add unit tests for helpers

### Hours 12-18: Priority 3 - Memory Extraction
- [ ] Implement ExtractMemory in completion path
- [ ] Update assembleContext to rank/score memories
- [ ] Add memory redaction logic
- [ ] Test extraction + ranking

### Hours 18-28: Priority 4 - Automation Workers
- [ ] Add event types to CreateEvent
- [ ] Implement 4 workers (context, repo, failure, projection)
- [ ] Wire into automation engine
- [ ] Add e2e tests for event chains

### Hours 28-40: Priority 5 - Contract (iterative)
- [ ] Write OpenAPI spec
- [ ] Generate TS client
- [ ] Update VSIX
- [ ] Add CI check (high effort)

### Hours 40-52: Priority 6 - Repo Pipeline
- [ ] Implement repo change detection
- [ ] Wire context invalidation
- [ ] Add follow-up task proposal
- [ ] Integrate with memory

---

## Acceptance Criteria (Proof Points)

### For Phase 1 Complete (Canonical Services)
- [ ] All 5 completion paths call `FinalizationService`
- [ ] v1 `/save` and v2 `/complete` produce identical DB state for same task
- [ ] `TestCompletionEquivalence` passes with 10+ scenarios
- [ ] No task leaks from incomplete finalization

### For Phase 2 Complete (Status Canonicalization)
- [ ] All policy/metrics queries use helper functions
- [ ] No hardcoded status string checks remain in main.go
- [ ] Lease `IsActive` check is single source of truth
- [ ] Metrics match DB reality ±0 (verified by tests)

### For Synergy Complete (Pipelines)
- [ ] repo.changed → context.invalidated → context refresh (e2e test)
- [ ] task.completed → memory extracted → appears in future claim
- [ ] automation triggers + workers coordinate without manual stitching
- [ ] Dashboard shows accurate queue, active runs, failures, recommendations

---

## Key Files to Modify

Primary:
- `finalization.go` (NEW - ~200 lines)
- `assignment.go` (REFACTOR - CompleteTask, add Memory extraction)
- `models_v2.go` (EXTEND - status helpers, new event types)
- `main.go` (REFACTOR - route completions to service, update v1 `/save`)
- `automation.go` (EXTEND - new triggers + workers)
- `db_v2.go` (EXTEND - RunSummary queries)

Secondary:
- `scanner.go` (REFACTOR - repo change detection)
- Policy + metrics queries (REFACTOR - use helpers)
- Test files (NEW - comprehensive regression tests)

External:
- `docs/api/openapi.yaml` (NEW)
- VSIX `service.ts` (REFACTOR - use generated client)
- `.gitignore` + CI (ENFORCE - hygiene)

---

## Risk & Mitigation

| Risk | Likelihood | Mitigation |
|------|------------|-----------|
| Finalization refactor breaks v1 compat | LOW | Comprehensive regression tests; dual-run proof points |
| Status helper changes break metrics | MEDIUM | Test metrics against DB ground truth before/after |
| Automation workers create infinite loops | MEDIUM | Rate limit + dedup engine already in place; extend with max-depth |
| Contract drift continues in VSIX | MEDIUM | Generated client reduces manual sync; CI gates future drift |
| Repo pipeline creates runaway task creation | MEDIUM | Automation policy gate + manual approval required at first |

---

## Success Criteria

**System achieves canonical-runtime + automated-synergy compliance when**:
1. ✅ Single claim path returns AssignmentEnvelope (v1, v2, all claim types)
2. ✅ Single completion path finalizes all state atomically
3. ✅ Context auto-assembled + ranked at claim time
4. ✅ Memory extracted, scored, and fed back to context engine
5. ✅ Repo changes auto-invalidate context and propose tasks
6. ✅ Automation workers coordinate without manual intervention
7. ✅ Dashboard reflects one truth view (zero staleness)
8. ✅ All contracts de-drifted via generated clients + CI gates
9. ✅ Compliance proven by 20+ acceptance tests

---

## Next Immediate Actions (Start Now)

1. **Create `finalization.go`** with FailTask/CancelTask/structured summary
2. **Refactor `CompleteTask()`** to extract RunSummary
3. **Update v1 `/save`** to call centralized service
4. **Write `TestCompletionEquivalence`** to lock in contract
5. **Add status helpers** to models_v2.go
