# Task ID 161: Gap Analysis & Remediation Plan (Update #2)

**Source**: uploaded codebase "cocopilot 3.zip"  
**Date**: March 05, 2026  
**Focus**: Canonical-runtime + automated-synergy criteria compliance  

## 1. The Criteria We're Holding the System To

This is the contract. If the system meets these items, the platform behaves like a real agent OS instead of a feature pile.

### 1.1 Canonical runtime loop (non-negotiable)

- Any worker (VSIX, MCP, CLI, autonomous agent) claims work through **ONE canonical flow**
- Claim always creates:
  - lease + run
  - transitions task state
  - emits lifecycle events
  - returns single AssignmentEnvelope
- AssignmentEnvelope always includes:
  - task + project + lease + run
  - policy snapshot + assembled context
  - completion contract
- Completion/failure always finalizes:
  - run status, task status, lease release
  - structured run summary
  - memory extraction, events, automation triggers
- Repo intelligence, context, memory, and automation are **not optional** side tables—they actively drive what the agent sees and what the system proposes next
- Operator dashboard reflects **one truth view** (queue, agents, runs, repo changes, failures, automation actions, recommendations)

### 1.2 Safety + governance baseline

- Legacy endpoints must not bypass auth/policy enforcement when enabled
- Workdir/repo access must be constrained (allowlisted roots, path validation, audit trail)
- Secrets must not leak into memory/logs/artifacts (redaction/retention rules)
- System must be reproducible (no runtime DBs/binaries in repo; deterministic tests; CI gates)

---

## 2. What Still Needs Work — Directly Against the Criteria

This is the remaining gaps blocking compliance with Section 1.

### 2.1 Canonical flow is not yet enforced end-to-end

**Current state**:
- Multiple claim/completion paths still exist (v1 behavior is being upgraded rather than strictly wrapping v2)
- V1 and v2 can still drift unless they share a single internal AssignmentService + FinalizationService
- Not all clients are guaranteed to consume the same AssignmentEnvelope contract (VSIX docs still describe v1 flows)
- Run/lease/task transitions are not yet guaranteed identical regardless of entry path (project claim-next vs task-claim-by-id vs v1 /task)

**Required work**:
1. Implement AssignmentService as the only place that can:
   - choose task
   - create lease
   - create run
   - transition task state
   - emit events
   - assemble context
   - return envelope
2. Implement FinalizationService as the only place that can:
   - complete/fail run
   - release lease
   - transition task
   - write summary
   - extract memory
   - emit events
   - trigger automation
3. Convert v1 endpoints into strict wrappers that call v2 services (no business logic divergence)
4. Update VSIX and MCP to use the v2 assignment envelope contract (v2-first)

### 2.2 Automated synergy loop is still incomplete

**Current state**:
- Repo intelligence isn't automatically driving context/memory/planning
- Repo scanning and repo_files persistence exist, but the platform does not enforce a default pipeline where repo changes update context and produce follow-up actions
- Context is not guaranteed to be assembled automatically on claim
- Context packs exist, but context assembly is not yet mandatory as part of assignment
- Missing: ranked selection of repo changes, repo files/snippets, memory hits, recent run summaries, dependency neighborhood, artifact refs
- Memory is not yet a learning loop
- Memory endpoints exist, but automatic extraction from run summaries (success/failure), ranking, and redaction are not yet enforced as part of the lifecycle
- Automation is not yet the nervous system
- Automation exists, but triggers/consumers need expansion so the system coordinates itself without manual stitching

**Required work**:
1. **Repo pipeline**: repo.changed → scan/sync → summarize → context invalidation → memory update → follow-up task proposals
2. **Context pipeline**: assemble automatically at claim-time and persist for traceability (optional but recommended)
3. **Memory pipeline**: extract from run summaries; rank (recency/confidence/scope/success correlation); redact secrets; feed into context ranking
4. **Automation pipeline**: add triggers for repo.changed, run.failed, lease.expired, policy.denied, dependency.unblocked, context.invalidated; add workers (context refresher, repo summarizer, failure summarizer, projection updater)

### 2.3 Contract drift risk remains (server vs MCP vs VSIX)

**Current state**:
- Without generated contracts and drift CI, MCP/VSIX can silently diverge from server expectations
- Hand-maintained tool schemas are fragile and already a known failure mode

**Required work**:
1. Define one contract source of truth and generate OpenAPI + typed client/schemas
2. Generate MCP tool manifest from the same contract artifacts
3. Migrate VSIX to use generated client or shared schema models
4. Add CI checks that fail on drift

### 2.4 Observability must be proven aligned with the real state model

**Current state**:
- Metrics/policy/dashboards must be validated against the current task/run/lease enums and lifecycle rules
- Any stale SQL or string statuses will create false operator confidence

**Required work**:
1. Centralize enum definitions and status buckets (task + run)
2. Define active lease purely by `expires_at > now`
3. Make metrics/policy counting call helper functions rather than hardcoded SQL filters

### 2.5 Hygiene, reproducibility, and operational posture

**Current state**:
- Runtime DBs/WAL/SHM and platform junk should not ship as source of truth
- Bundled .git directory should never be in release zips
- Tests must not leave DB artifacts behind; CI must gate cleanliness

**Required work**:
1. Clean packaging; strengthen .gitignore; enforce via CI
2. Force tests to use temp dirs; deterministic cleanup
3. Default bind to localhost; require explicit opt-in for external bind; ensure legacy endpoints cannot bypass auth when enabled

---

## 3. The Minimal Set of Changes to Become Compliant

If you only do one thing, do this. This is the shortest path to compliance with Section 1.

### 3.1 Build the canonical AssignmentEnvelope path

1. Implement internal AssignmentService (single source of truth for claim)
2. Implement internal FinalizationService (single source of truth for complete/fail)
3. Return AssignmentEnvelope for every claim (v2 claim-next, v2 claim-by-id, v1 /task wrapper)
4. Make VSIX and MCP consume AssignmentEnvelope (stop relying on legacy semantics)

### 3.2 Force the synergy pipelines into the default lifecycle

1. On claim: auto-assemble context (deps + parents + repo changes + memory hits + recent runs + artifacts + policies)
2. On completion: write structured run summary, extract memory, update repo file metadata if needed, emit events
3. On repo changed: run repo pipeline and invalidate stale contexts; propose tasks
4. Automation consumes events and triggers derived workers

### 3.3 Lock contracts + kill drift

1. Generate OpenAPI + clients
2. Generate MCP manifest
3. Migrate VSIX
4. CI drift checks

---

## 4. Phased Implementation Plan (what to build, in order)

### Phase 0 — Clean source of truth + CI (stop the bleeding)
- Remove runtime DB/WAL/SHM, .git, __MACOSX, .DS_Store, coverage outputs from tracked source and release artifacts
- Update .gitignore and add CI 'clean tree' check
- Force tests to use temp dirs and always cleanup
- Add strong Go CI lane (build/test/vet/race/migration smoke)

### Phase 1 — Canonical state helpers
- Centralize task/run status enums and bucket mapping
- Define active lease via `expires_at > now`
- Update metrics and policy counting to use helpers
- **Impact**: Medium | **Feasibility**: High | **Duration**: 2-4 hours

### Phase 2 — Canonical assignment + finalization services
- Implement AssignmentService and route all claim paths through it
- Implement FinalizationService and route all complete/fail paths through it
- Normalize lifecycle events emitted for claim/start/complete/fail/lease-expired
- Demote v1 endpoints to strict wrappers
- **Impact**: Critical | **Feasibility**: Medium | **Duration**: 8-12 hours

### Phase 3 — Contract integrity
- Define contract source of truth and generate OpenAPI + typed clients
- Generate MCP tool schemas from contract artifacts
- Migrate VSIX core flows to v2 AssignmentEnvelope
- Add CI contract drift checks
- **Impact**: High | **Feasibility**: Medium | **Duration**: 8-12 hours

### Phase 4 — Automated synergy
- Repo pipeline: detect changes → scan/sync → summarize → emit events → invalidate context → propose tasks
- Context engine: rank items; budget by size; persist context packs for traceability
- Memory extraction: from run summaries (success/failure), rank, redact secrets, feed back into context engine
- Automation workers: context refresh, repo summarizer, failure summarizer, projection updater; broaden trigger set
- **Impact**: Critical | **Feasibility**: Low-Medium | **Duration**: 16-24 hours

---

## 5. Acceptance Tests (how we prove compliance)

### 5.1 Lifecycle invariants
- **Claim invariant**: claim always results in exactly one active lease + one active run, created atomically, with task status transition consistent
- **Finalize invariant**: completion/failure always updates run + task, releases lease, writes structured summary, emits events, and triggers automation hooks
- **Envelope invariant**: every claim returns AssignmentEnvelope with required fields and ranked context items
- **Wrapper invariant**: v1 endpoints produce identical state transitions to v2 (same services)

### 5.2 System behavior checks
- Dashboard matches DB truth: queue counts, active runs, active leases, failure counts
- Repo change triggers: context invalidation + follow-up task proposal appears without manual steps
- Memory loop works: finishing a task produces new memory candidates and they appear in future assignment context ranking
- Auth posture: when enabled, mutating endpoints require proper identity/scope; legacy endpoints cannot bypass

---

## 6. Immediate Next Actions (48-hour build order)

1. Clean packaging + repo artifacts; enforce via CI
2. Implement state helpers (enums, active lease/run) and fix metrics/policy counting
3. Implement AssignmentService + FinalizationService; route v1/v2 through them
4. Define AssignmentEnvelope and update clients to consume it (start with VSIX)
5. Wire minimal synergy: on claim include repo changes + memory hits + recent runs; on complete write summary + extract memory
