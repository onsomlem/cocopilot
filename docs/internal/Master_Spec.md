Cocopilot Master System Specification
Source-of-truth engineering specification for runtime canonicalization, orchestration unification, and system hardening
Date: 2026-03-04   |   Scope: Uploaded codebase audit -> target architecture -> implementation program

1. Executive Summary
The current codebase has enough implemented surface area to become a powerful local orchestration platform, but it is still behaving like a collection of features rather than a single coherent system. The primary defect is not a lack of components. The primary defect is the absence of one canonical runtime model that forces tasks, projects, agents, leases, runs, context, memory, repo intelligence, automation, and clients to move through the same lifecycle.
This specification defines the required runtime model, architectural boundaries, lifecycle invariants, migration strategy, and acceptance criteria to turn the codebase into a coherent agent operating system.
2. Current-State Diagnosis
2.1 High-confidence findings
The backend mixes legacy v1 queue behavior with richer v2 project-scoped behavior, creating split-brain execution paths.
The server entrypoint is oversized and centralizes routing, business logic, UI generation, metrics, project operations, and orchestration behavior.
Task, run, and lease state handling are inconsistent across metrics, policy checks, claim paths, and UI surfaces.
The codebase contains generated runtime artifacts, including SQLite state and built outputs, which weakens reproducibility and obscures the source of truth.
The MCP layer and VSIX layer are not guaranteed to match the backend contract, creating client drift risk.
Repo intelligence, memory, context packs, automation, and events exist, but they do not form a strong closed-loop control system.
2.2 Root problem statement
The system has implemented many of the right primitives, but the primitives are not forced through one canonical assignment lifecycle. As a result, capability grows more slowly than complexity and every new feature increases drift risk.
3. Product Intent
3.1 Target outcome
Transform the codebase into a project-scoped orchestration platform where every worker, extension, and tool uses the same assignment lifecycle: claim -> lease -> run -> context -> execute -> finalize -> learn -> automate.
3.2 Non-goals for this program
Do not add net-new feature surface before runtime consistency is established.
Do not deepen legacy endpoint behavior except as compatibility wrappers.
Do not prioritize UI polish ahead of lifecycle correctness, contract integrity, and observability.
Do not treat repo scanning, memory, or automation as side features; they must become part of the operating loop.
4. Required Design Principles
One runtime model. V2 becomes the canonical execution model.
One assignment lifecycle. All clients receive equivalent task, lease, run, and context semantics.
One state truth. Task, run, and lease states must come from centralized helpers, not scattered raw strings.
One contract truth. Backend, MCP, and VSIX must derive from the same request and response definitions.
Project scope first. Execution, repo access, context, memory, and automation are project-scoped by default.
Events matter. Important lifecycle changes must emit events and update derived state.
Repo intelligence must feed orchestration. Changed files, summaries, and risk signals must affect planning and context.
Learning is operational. Completion and failure data must update memory, recommendations, and future assignments.
5. Canonical Runtime Model
The canonical runtime begins when a client asks for work and ends only after the system has finalized run state, updated task state, emitted lifecycle events, refreshed derived state, and generated learning outputs.
5.1 Canonical flow
Select next eligible task within project scope.
Create lease for the claimant.
Create run for the assignment.
Transition task state to claimed or running using canonical rules.
Assemble project-scoped context.
Return a structured assignment envelope.
Execute task and collect logs, artifacts, and output.
Finalize run as succeeded or failed through one service.
Update task state through canonical completion logic.
Emit lifecycle events.
Extract memory and update repo or project intelligence as needed.
Trigger automation and refresh dashboard read models.
5.2 Assignment envelope
Every client should receive one structured envelope containing task, project, lease, run, context, policy, and completion expectations. No client should have to manually stitch these pieces together.
Envelope Field
Purpose
task
Assigned task payload and status metadata
project
Project identity, workdir, and project settings
lease
Claim duration and renewal data
run
Execution session identifier and status
policy
Constraints, quotas, and allowed operations
context
Parent chain, dependencies, repo changes, memory hits, artifacts, relevant files
instructions
Expected outputs and completion contract
6. Domain Architecture
Project Domain: Owns project identity, workdir, settings, and project-scoped derived state.
Task Domain: Owns task lifecycle, dependency graph, prioritization, and queue eligibility.
Assignment Domain: Owns claim, lease, run creation, completion, failure, and assignment envelopes.
Run Domain: Owns logs, artifacts, step history, summaries, and execution outcomes.
Context Domain: Owns automatic context assembly, relevance ranking, and invalidation.
Memory Domain: Owns extracted operational memory, ranking, retention, and safeguards.
Repo Intelligence Domain: Owns workdir validation, file sync, git change detection, summaries, and file risk signals.
Automation Domain: Owns event-driven reactions, planners, and follow-up work generation.
Policy Domain: Owns quotas, authorization gates, allowed actions, and runtime limits.
Event Domain: Owns event schema, append-only event recording, fanout, and replay into read models.
Read Model Domain: Owns fast projections for dashboard, queue health, agent state, and project health.
7. Core Invariants
A claimed task always has a valid project scope, a valid lease, and a valid run.
A finalized run always updates run state, task state, summary output, and event emission through one service.
All clients receive equivalent assignment semantics regardless of whether they are CLI, MCP, VSIX, or compatibility wrappers.
Dashboard counts, policy checks, and queue views are derived from canonical status helpers, not ad hoc queries.
Repo changes can invalidate stale context and influence follow-up planning.
Memory persistence must respect redaction and sensitive-data rules.
Legacy endpoints may not introduce side effects that differ from canonical v2 behavior.
8. Event Model
The following event families should exist and use normalized names and payload shapes:
task.created, task.updated, task.claimed, task.completed, task.failed, task.blocked
run.started, run.completed, run.failed
lease.created, lease.renewed, lease.expired
repo.changed, repo.scanned
context.invalidated, context.refreshed
memory.created, memory.updated
automation.emitted
policy.denied
project.idle
9. Automated Synergy Model
The missing high-level behavior is automated synergy: the system must observe change, synthesize state, plan next work, execute with assembled context, and learn from results. The following closed-loop model defines that behavior.
9.1 Observe
Queue state, dependency state, active leases, active runs
Repo changes and file summaries
Recent artifacts, logs, summaries, and failures
Policy denials and quota pressure
Memory and historical performance signals
9.2 Understand
Derive project health, blocked work, stale context, hot files, failure concentration, and likely next tasks
9.3 Plan
Generate next-best tasks, reprioritize stale work, escalate failures, and recommend operator interventions
9.4 Execute
Issue one assignment envelope with all required execution context and constraints
9.5 Learn
Write run summaries, extract reusable memory, update risk signals, refresh projections, and emit follow-up automation
10. Immediate Technical Defects To Correct
Oversized entrypoint architecture in main.go
Legacy and v2 split-brain execution paths
Metrics using stale task state assumptions
Lease activity checks that do not match expiry-based lease semantics
Policy checks using stale or mismatched run-state logic
Planner task types that are not consistently recognized
Claim paths that do not produce the same side effects
MCP and VSIX contract drift risk
Generated binaries and SQLite state committed or carried with the source tree
Insufficient CI enforcement for backend correctness and contract integrity
11. Target Package Layout
Recommended backend structure:
cmd/cocopilot/main.go
internal/config
internal/http/legacy
internal/http/v2
internal/projects
internal/tasks
internal/assignments
internal/leases
internal/runs
internal/context
internal/memory
internal/repo
internal/automation
internal/policy
internal/events
internal/readmodels
internal/ui
internal/auth
pkg/contracts
12. Migration Program
Phase 0 - Clean ground truth: Remove generated artifacts, force temp-dir test behavior, add CI, lock dependency installs, add repo-cleanliness checks.
Phase 1 - Canonicalize status and state models: Centralize task, run, and lease helpers; fix metrics, policy queries, and planner task typing.
Phase 2 - Unify assignment lifecycle: Introduce one assignment service for claim, lease, run creation, completion, and failure handling.
Phase 3 - Refactor handlers out of main.go: Move domain logic into services and reduce main.go to startup and route wiring.
Phase 4 - Lock contracts: Create one API contract source of truth and generate shared clients or schemas.
Phase 5 - Build automatic context: Assemble context from tasks, dependencies, repo changes, memory, runs, and artifacts by default.
Phase 6 - Wire repo intelligence into orchestration: Drive context invalidation, recommendations, and planning from repo changes and file summaries.
Phase 7 - Build memory and learning loop: Extract operational memory from run outcomes and repeated patterns.
Phase 8 - Expand event-driven automation: Use normalized event triggers to refresh context, plan work, update read models, and react to failure.
Phase 9 - Build unified operator dashboard: Replace scattered diagnostics with one project health surface.
Phase 10 - Harden security and deployment: Default to localhost, enforce auth on mutations, add workdir guardrails and redaction.
13. Acceptance Criteria
Every active worker path uses the same assignment lifecycle and returns the same envelope shape.
No dashboard or policy logic depends on stale or duplicated status assumptions.
Legacy endpoints, if retained, act as compatibility shims over canonical services.
Context assembly is automatic for all primary execution paths.
Repo changes influence context freshness, recommendations, and follow-up work.
Run completion and failure update memory and emit normalized events.
MCP and VSIX are contract-aligned with backend truth.
Backend CI, contract drift checks, and repo cleanliness checks are enforced.
Local runtime safety is stronger through project scoping, auth, and path guardrails.
14. Risks and Failure Modes
Attempting UI polish before lifecycle unification will consume time without fixing system coherence.
Keeping multiple independent claim paths will preserve hidden state divergence.
Adding more feature surface before contract and state cleanup will increase maintenance cost and reduce trust.
Treating repo intelligence or memory as optional will preserve the current feature-pile behavior.
Leaving security posture soft while exposing workdir and mutation endpoints increases operational risk.
15. Implementation Priority
The first build wave should execute in this order:
Clean repo artifacts and fix temp-dir test behavior
Add backend CI and dependency reproducibility
Centralize task, run, and lease state helpers
Fix metrics and policy queries
Normalize planner task typing
Build canonical assignment service
Unify direct claim and project claim-next on that service
Unify completion and failure on one finalization service
Define contract source of truth and update MCP and VSIX
Build automatic context assembly and repo-driven invalidation
16. Final Directive
The codebase should optimize for coherence over feature count. It already has enough ingredients. The win now comes from forcing tasks, runs, context, repo intelligence, memory, automation, and clients through one runtime model so that each component reinforces the others instead of drifting beside them.
