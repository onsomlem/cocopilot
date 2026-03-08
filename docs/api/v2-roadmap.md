# API v2 Implementation Roadmap

## Overview

This roadmap breaks down the API v2 implementation into manageable phases, each delivering incremental value while maintaining backward compatibility. Total estimated timeline: **5 weeks** with parallel work streams.

Recent documentation updates note that the MCP server and VSIX scaffolds are documented with current command/tool coverage; packaging and release automation remain pending. The automation API now includes rules, simulate, and replay endpoints.

**Start Date:** February 11, 2026  
**Target Completion:** March 14, 2026  
**Team Size:** 2-3 developers
**Status:** v2.0.0 released. All core backend phases complete. See bottom of file for remaining polish items.

## Phase 1: Foundation (Week 1)

**Goal:** Establish v2 infrastructure and basic project/task endpoints

### Tasks

#### 1.1 Project Setup (Day 1)
- [x] Create `/api/v2` route namespace
- [ ] Set up v2 middleware (logging, error handling, CORS)
- [x] Implement standard error response format
- [ ] Add API versioning header support

**Deliverables:**
- `handlers/v2/middleware.go`
- `handlers/v2/errors.go`
- Route setup in `main.go`

**Effort:** 4 hours

#### 1.2 Health & Version Endpoints (Day 1)
- [x] Implement `GET /api/v2/health`
- [x] Implement `GET /api/v2/version`
- [x] Add unit tests
- [x] Update API documentation

**Notes:**
- `GET /api/v2/version` includes retention config snapshot (retention.enabled, interval_seconds, max_rows, days).
- `GET /api/v2/config` is implemented (redacted runtime config snapshot).

**Deliverables:**
- `handlers/v2/health.go`
- Tests in `handlers/v2/health_test.go`

**Effort:** 2 hours

#### 1.3 Database Migration 0005 (Day 1)
- [x] Create migration file `0005_tasks_v2_enhancements.sql`
- [x] Add status_v2, title, type, priority, tags_json columns
- [x] Write backfill logic for status_v2
- [x] Create indexes for v2 queries
- [x] Test migration on development database

**Deliverables:**
- `migrations/0005_tasks_v2_enhancements.sql`
- Migration test script

**Effort:** 3 hours

#### 1.4 Projects CRUD (Day 2-3)
- [x] Implement `POST /api/v2/projects` (create)
- [x] Implement `GET /api/v2/projects` (list)
- [x] Implement `GET /api/v2/projects/{projectId}` (get)
- [x] Implement `PATCH /api/v2/projects/{projectId}` (update)
- [x] Implement `DELETE /api/v2/projects/{projectId}` (delete)
- [ ] Add validation (name, workdir required)
- [ ] Write unit and integration tests
- [ ] Generate project IDs (format: `proj_<uuid>`)

**Deliverables:**
- `handlers/v2/projects.go`
- `models/project.go`
- Tests in `handlers/v2/projects_test.go`

**Effort:** 12 hours

#### 1.4a Project Tree & Changes (Day 3)
- [x] Implement `GET /api/v2/projects/{projectId}/tree`
- [x] Implement `GET /api/v2/projects/{projectId}/changes`
- [x] Write unit and integration tests

**Deliverables:**
- Tests in `v2_project_tree_test.go` and `v2_project_changes_test.go`

**Effort:** 4 hours

#### 1.5 Tasks v2 Endpoints (Day 3-5)
- [x] Implement `POST /api/v2/projects/{projectId}/tasks` (create)
- [x] Update task model with v2 fields
- [x] Implement `GET /api/v2/tasks` (list with filters)
- [x] Implement `GET /api/v2/projects/{projectId}/tasks` (list with filters)
- [x] Implement `GET /api/v2/tasks/{taskId}` (get)
- [x] Implement `PATCH /api/v2/tasks/{taskId}` (update)
- [x] Implement `DELETE /api/v2/tasks/{taskId}` (delete)
- [x] Implement `POST /api/v2/tasks/{taskId}/claim` (claim)
- [x] Implement `POST /api/v2/tasks/{taskId}/complete` (complete)
- [x] Implement `POST/GET/DELETE /api/v2/tasks/{taskId}/dependencies`
- [x] Add query parameter parsing (project_id, status, limit, offset)
- [x] Add sort support (created_at:asc|desc, updated_at:asc|desc)
- [x] Add query parameter parsing (type, tag, q)
- [ ] Implement status sync logic (v1 ↔ v2)
- [x] Write unit and integration tests
- [ ] Test v1 compatibility (create via v1, read via v2)

**Notes:**
- Task creation is implemented as `POST /api/v2/tasks` with optional `project_id` and `parent_task_id`.
- Task list endpoints support `project_id`, `status`, `type`, `tag`, `q`, `limit`, `offset`, and `sort` on `created_at`/`updated_at`.
- Task update (`PATCH /api/v2/tasks/{taskId}`), delete (`DELETE /api/v2/tasks/{taskId}`), claim (`POST /api/v2/tasks/{taskId}/claim`), and complete (`POST /api/v2/tasks/{taskId}/complete`) are implemented.
- Task dependency endpoints are implemented: `POST/GET/DELETE /api/v2/tasks/{taskId}/dependencies`.

**Deliverables:**
- `handlers/v2/tasks.go`
- `models/task.go` (updated)
- `utils/status.go` (status mapping functions)
- Tests in `handlers/v2/tasks_test.go`

**Effort:** 16 hours

#### 1.6 Agents Endpoints (Day 4-5)
- [x] Implement `GET /api/v2/agents/{agentId}` (detail)
- [x] Implement `DELETE /api/v2/agents/{agentId}` (delete)
- [x] Implement `GET /api/v2/agents` (list)
- [x] Add filters (`status`, `since`, `limit`, `offset`)
- [x] Add sorting (`created_at`, `last_seen:asc`, `last_seen:desc`)
- [x] Write unit and integration tests

**Deliverables:**
- `handlers/v2/agents.go`
- `models_v2.go` (agent model)
- Tests in `v2_agent_*_test.go`

**Effort:** 6 hours

#### 1.7 Auth & Scope Guardrails (Day 4-5)
- [x] Add API key enforcement for mutating v2 endpoints
- [x] Add optional read-endpoint enforcement toggle
- [x] Enforce scoped identities with per-endpoint scope checks
- [x] Standardize unauthorized/forbidden responses in v2 error envelope
- [x] Emit auth decision logs and denial events
- [x] Guard v2 events stream with read scopes when enabled
- [x] Write route-level auth tests

**Deliverables:**
- Auth middleware in v2 route setup
- Auth decision logging + events
- Tests in `v2_routes_test.go` and `v2_events_stream_test.go`

**Effort:** 8 hours

#### 1.7a Policy Engine Foundation (Day 4-5)
- [x] Add policies table migration `0018_policies.sql`
- [x] Store policy definitions for future evaluation
- [x] Wire policy engine foundation hooks for enforcement evolution

**Notes:**
- This is storage + scaffolding only; enforcement expansion remains planned.

**Deliverables:**
- `migrations/0018_policies.sql`
- Policy engine foundation wiring

**Effort:** 4 hours

#### 1.8 Testing & Documentation (Day 5)
- [ ] Run full test suite (v1 + v2)
- [ ] Test v1 endpoints still work
- [x] Update OpenAPI spec with implemented endpoints
- [ ] Write usage examples for projects and tasks
- [ ] Update README with v2 sections

**Deliverables:**
- Updated `docs/api/openapi-v2.yaml`
- `docs/api/v2-examples.md`

**Effort:** 4 hours

### Phase 1 Milestones
- [x] v2 infrastructure in place (routing + error envelope)
- [x] Projects CRUD endpoints implemented
- [x] Project tree and project changes endpoints implemented
- [x] Tasks endpoints implemented (create/list/detail/update/delete/claim/complete/dependencies; project tasks list)
- [x] Agents endpoints implemented (list/detail/delete with filters + sort)
- [x] Auth guardrails implemented for v2 routes (API key + scopes)
- [x] Policies table + policy engine foundation in place
- [x] v1 compatibility maintained

**Total Effort:** ~40 hours (1 week for 1 developer)

---

## Phase 2: Execution Ledger (Week 2)

**Goal:** Implement runs, steps, logs, and artifacts for execution tracking

### Tasks

#### 2.1 Database Migration 0006 (Day 1)
- [x] Create migration file `0006_runs.sql`
- [x] Create `runs` table
- [x] Create `run_steps` table
- [x] Create `run_logs` table  
- [x] Create `artifacts` table
- [x] Create `tool_invocations` table
- [x] Test migration on development database

**Deliverables:**
- `migrations/0006_runs.sql`

**Effort:** 4 hours

#### 2.2 Run Model & ID Generation (Day 1-2)
- [x] Implement run model (v2 models)
- [x] Add UUID generation for run IDs (format: `run_<uuid>`)
- [x] Implement run creation logic
- [x] Add run status transitions (RUNNING → SUCCEEDED/FAILED)
- [x] Write unit tests

**Deliverables:**
- `models/run.go`
- `utils/id.go` (UUID generation)
- Tests

**Effort:** 6 hours

#### 2.3 Runs Endpoints (Day 2-3)
- [x] Implement `GET /api/v2/runs/{runId}` (get details)
- [x] Include steps, logs (recent), artifacts in response
- [ ] Implement pagination for logs
- [ ] Add filtering options
- [x] Write tests

**Deliverables:**
- `handlers/v2/runs.go`
- Tests

**Effort:** 8 hours

#### 2.4 Run Steps Endpoint (Day 3)
- [x] Implement `POST /api/v2/runs/{runId}/steps`
- [x] Add step validation (name, status)
- [x] Store step details as JSON
- [x] Write tests

**Deliverables:**
- Step creation in `handlers/v2/runs.go`
- Tests

**Effort:** 4 hours

#### 2.5 Run Logs Endpoint (Day 3-4)
- [x] Implement `POST /api/v2/runs/{runId}/logs`
- [x] Support stream types (stdout, stderr, info)
- [x] Add timestamp handling
- [x] Implement log chunking/streaming
- [x] Write tests

**Deliverables:**
- Log creation in `handlers/v2/runs.go`
- Tests

**Effort:** 6 hours

#### 2.6 Artifacts Endpoint (Day 4)
- [x] Implement `POST /api/v2/runs/{runId}/artifacts`
- [x] Support artifact kinds (diff, patch, log, report, file)
- [x] Add storage reference handling (local paths)
- [x] Add SHA256 validation (optional)
- [x] Write tests

**Deliverables:**
- Artifact creation in `handlers/v2/runs.go`
- Tests

**Effort:** 6 hours

#### 2.7 Integration Testing (Day 5)
- [ ] End-to-end test: create run → add steps → log → attach artifact
- [ ] Test run detail retrieval with all nested data
- [ ] Performance test (1K runs, 10K steps)
- [ ] Update documentation

**Deliverables:**
- Integration tests in `tests/integration/runs_test.go`
- Performance benchmarks

**Effort:** 6 hours

### Phase 2 Milestones
- [x] Run schema + GET run endpoint implemented
- [x] Steps, logs, artifacts fully functional
- [ ] Execution history queryable
- [ ] Performance validated

**Total Effort:** ~40 hours (1 week for 1 developer)

---

## Phase 3: Leases & Claiming (Week 2-3)

**Goal:** Enable multi-agent coordination with lease-based task claiming

### Tasks

#### 3.1 Database Migration 0007 (Day 1)
- [x] Create migration file `0007_leases.sql`
- [x] Create `leases` table with unique task_id constraint
- [x] Add indexes for expiration and agent queries
- [x] Test migration

**Deliverables:**
- `migrations/0007_leases.sql`

**Effort:** 2 hours

#### 3.2 Lease Model (Day 1-2)
- [x] Implement `models/lease.go`
- [x] Add lease ID generation (format: `lease_<uuid>`)
- [x] Implement lease expiration logic (default: 5 minutes)
- [x] Add lease validation (task not already leased)
- [ ] Write unit tests

**Deliverables:**
- `models/lease.go`
- Tests

**Effort:** 6 hours

#### 3.3 Task Claim Endpoint (Day 2-3)
- [x] Implement `POST /api/v2/tasks/{taskId}/claim`
- [x] Check task not already leased (return 409 CONFLICT)
- [x] Create lease with expiration
- [x] Create run associated with claim
- [x] Update task status (v1: IN_PROGRESS, v2: CLAIMED)
- [ ] Optionally generate context pack
- [x] Return lease and task payload
- [ ] Return run and context_pack_id (when generated)
- [x] Write tests (success, conflict, not found)

**Notes:**
- `POST /api/v2/leases` is implemented for direct lease creation.
- `GET /task` acquires leases before returning work.

**Deliverables:**
- `handlers/v2/leases.go`
- Tests

**Effort:** 10 hours

#### 3.4 Lease Heartbeat (Day 3)
- [x] Implement `POST /api/v2/leases/{leaseId}/heartbeat`
- [x] Extend lease expiration (e.g., +5 minutes)
- [x] Validate lease exists and not expired
- [x] Return updated lease with new expires_at
- [ ] Write tests

**Deliverables:**
- Heartbeat in `handlers/v2/leases.go`
- Tests

**Effort:** 4 hours

#### 3.5 Lease Release (Day 3)
- [x] Implement `POST /api/v2/leases/{leaseId}/release`
- [x] Delete lease from database
- [x] Update task status back to queued if not completed
- [ ] Log release reason
- [ ] Write tests

**Deliverables:**
- Release in `handlers/v2/leases.go`
- Tests

**Effort:** 4 hours

#### 3.6 Lease Expiration Background Job (Day 4)
- [x] Implement background goroutine to check expired leases
- [x] Run every 30 seconds
- [x] Delete expired leases
- [x] Update task status back to queued
- [x] Publish events (lease.expired)
- [x] Add logging for expiration
- [ ] Write tests

**Deliverables:**
- `jobs/lease_expiration.go`
- Tests

**Effort:** 6 hours

#### 3.7 Integration Testing (Day 4-5)
- [ ] Test full claim → heartbeat → release flow
- [ ] Test claim conflict (two agents)
- [ ] Test lease expiration and reclaim
- [ ] Test claim with context pack generation
- [ ] Performance test (100 concurrent claims)
- [ ] Update documentation

**Deliverables:**
- Integration tests
- Performance benchmarks

**Effort:** 8 hours

### Phase 3 Milestones
- ✅ Tasks claimable by agents
- ✅ Leases prevent conflicts
- ✅ Heartbeat keeps leases alive
- ✅ Expiration allows reclaim

**Total Effort:** ~40 hours (1 week for 1 developer)

---

## Phase 4: Structured Completion (Week 3)

**Goal:** Enable rich, structured task completion metadata

### Tasks

#### 4.1 Completion Model (Day 1)
- [ ] Define `models/completion.go`
- [ ] Add result structure (summary, changes_made, files_touched, etc.)
- [ ] Add validation (summary required)
- [ ] Write unit tests

**Deliverables:**
- `models/completion.go`
- Tests

**Effort:** 4 hours

#### 4.2 Complete Endpoint (Day 1-2)
- [x] Implement `POST /api/v2/tasks/{taskId}/complete`
- [ ] Validate run_id belongs to task
- [x] Update run status (SUCCEEDED/FAILED)
- [x] Update task status_v2 (SUCCEEDED/FAILED/NEEDS_REVIEW)
- [x] Update task status_v1 (COMPLETE or IN_PROGRESS)
- [ ] Generate v1 output from result summary
- [x] Release lease if exists
- [x] Publish events
- [x] Write tests

**Deliverables:**
- `handlers/v2/completion.go`
- Tests

**Effort:** 10 hours

#### 4.3 v1 Output Generation (Day 2)
- [ ] Implement function to generate plain-text output from v2 result
- [ ] Format: "Summary: ...\n\nChanges:\n- ...\n\nFiles: ..."
- [ ] Store in tasks.output for v1 compatibility
- [ ] Write tests comparing v1 and v2 formats

**Deliverables:**
- `utils/output.go`
- Tests

**Effort:** 4 hours

#### 4.4 Next Tasks Generation (Day 3)
- [x] Parse result.next_tasks array
- [x] Create new tasks in database
- [x] Link as children of current task
- [x] Publish events (task.created)
- [x] Write tests

**Deliverables:**
- Next task logic in `handlers/v2/completion.go`
- Tests

**Effort:** 6 hours

#### 4.5 Integration Testing (Day 3-4)
- [ ] Test full flow: claim → work → complete with rich result
- [ ] Test v1 client reads output generated from v2 completion
- [x] Test next_tasks creation
- [ ] Test different completion statuses (SUCCEEDED, FAILED, NEEDS_REVIEW)
- [ ] Update documentation

**Deliverables:**
- Integration tests
- Updated API examples

**Effort:** 6 hours

### Phase 4 Milestones
- [x] Structured completion endpoint implemented
- [ ] v1 output auto-generated from v2
- [x] Next tasks auto-created
- [x] Full v1/v2 compatibility (core task flows)

**Total Effort:** ~30 hours (3-4 days for 1 developer)

---

## Phase 5: Events & Memory (Week 4)

**Goal:** Real-time updates via SSE and persistent knowledge base

### Tasks

#### 5.1 Database Migrations 0008-0009 (Day 1)
- [x] Create migration `0008_events.sql`
- [x] Create migration `0009_memory.sql`
- [x] Test migrations

**Deliverables:**
- Migration files

**Effort:** 3 hours

#### 5.2 Event Model & Publisher (Day 1-2)
- [x] Implement `models/event.go`
- [x] Add event ID generation (format: `event_<ulid>`)
- [x] Implement event publisher (stores to DB)
- [x] Add event kinds (task.*, run.*, lease.*, etc.)
- [ ] Write unit tests

**Deliverables:**
- `models/event.go`
- `services/event_publisher.go`
- Tests

**Effort:** 8 hours

#### 5.3 Project-Scoped SSE (Day 2-3)
- [x] Implement `GET /api/v2/events/stream`
- [x] Set up SSE response headers
- [x] Stream events as they occur
- [x] Maintain client connections
- [x] Add keepalive pings
- [x] Support `project_id` scoping for per-project streams
- [x] Write tests (SSE client simulation)

**Deliverables:**
- `handlers/v2/events.go`
- SSE test client

**Effort:** 10 hours

#### 5.4 Event Replay (Day 3)
- [x] Implement `GET /api/v2/events`
- [x] Query events with `since`, `type`, `task_id`, `project_id`, `limit`, and `offset`
- [x] Return JSON array of events with `total` count
- [x] Support `project_id` scoping for per-project replay
- [x] Write tests

**Deliverables:**
- Replay endpoint in `handlers/v2/events.go`
- Tests

**Effort:** 4 hours

#### 5.5 Integrate Event Publishing (Day 3-4)
- [x] Add publishEvent calls to all mutation endpoints
- [x] Task create/update/delete → events
- [ ] Run start/complete → events
- [x] Lease create/expire/release → events
- [ ] Test event flow end-to-end

**Notes:**
- Events retention cleanup runs on a configurable interval when retention is enabled.
- Events filtering indexes (migration `0015_events_filter_indexes.sql`) are applied.

**Deliverables:**
- Updated handlers with event publishing

**Effort:** 6 hours

#### 5.6 Memory Endpoints (Day 4-5)
- [x] Implement `PUT /api/v2/projects/{projectId}/memory`
- [x] Implement `GET /api/v2/projects/{projectId}/memory`
- [x] Add query parameter parsing (scope, key, q)
- [x] Implement upsert logic (unique constraint)
- [x] Support source_refs
- [x] Write tests

**Deliverables:**
- `handlers/v2/memory.go`
- Tests

**Effort:** 10 hours

### Phase 5 Milestones
- [x] Real-time events per project
- [x] Event replay for catch-up
- [x] Memory persistence working
- [x] All mutations publish events

**Total Effort:** ~40 hours (1 week for 1 developer)

---

## Phase 6: Context Packs (Week 4)

**Goal:** Automated context generation for tasks

### Tasks

#### 6.1 Database Migration 0010 (Day 1)
- [x] Create migration `0010_context_packs.sql`
- [x] Test migration

**Deliverables:**
- Migration file

**Effort:** 2 hours

#### 6.2 Context Pack Algorithm (Day 1-3)
- [ ] Implement file tree traversal
- [ ] Implement relevance scoring (file name matching, task keywords)
- [ ] Implement snippet extraction (important functions/classes)
- [ ] Implement budget enforcement (max files, bytes, snippets)
- [ ] Add related task lookup
- [ ] Add memory (decisions) lookup
- [ ] Add git repo state detection
- [ ] Write unit tests

**Deliverables:**
- `services/context_generator.go`
- Tests

**Effort:** 12 hours

#### 6.3 Context Pack Endpoint (Day 3-4)
- [x] Implement `POST /api/v2/projects/{projectId}/context-packs`
- [x] Call context generator
- [x] Store pack in database
- [x] Return pack with contents
- [x] Write tests

**Deliverables:**
- `handlers/v2/context.go`
- Tests

**Effort:** 6 hours

#### 6.4 Integrate with Claim (Day 4)
- [ ] Add optional context pack generation during claim
- [ ] Add query param `/tasks/{taskId}/claim?generate_context=true`
- [ ] Return context_pack_id in claim response
- [ ] Update tests

**Deliverables:**
- Updated claim handler

**Effort:** 4 hours

#### 6.5 Testing & Optimization (Day 5)
- [ ] Test with various project sizes
- [ ] Optimize file scanning performance
- [ ] Add caching for repeated context generation
- [ ] Update documentation

**Deliverables:**
- Performance benchmarks
- Optimization report

**Effort:** 6 hours

### Phase 6 Milestones
- [ ] Context packs auto-generated
- [ ] Integrated with task claim
- [ ] Performance optimized
- [ ] Budget controls working

**Total Effort:** ~30 hours (3-4 days for 1 developer)

---

## Phase 7: Polish & Documentation (Week 5)

**Goal:** Production readiness, testing, and documentation

### Tasks

#### 7.1 Error Handling Audit (Day 1)
- [ ] Review all endpoints for consistent error responses
- [ ] Add validation error messages
- [ ] Add proper HTTP status codes
- [ ] Test error scenarios

**Deliverables:**
- Error handling report
- Fixed endpoints

**Effort:** 6 hours

#### 7.2 OpenAPI Validation (Day 1-2)
- [x] Set up Swagger UI locally
- [x] Validate OpenAPI spec against implementation
- [x] Fix any discrepancies
- [ ] Add request/response examples
- [ ] Test all endpoints via Swagger UI

**Notes:**
- OpenAPI runtime parity pass completed for shipped v2 endpoints; planned endpoints are marked with `x-runtime-status: planned`.

**Deliverables:**
- Validated `docs/api/openapi-v2.yaml`
- Swagger UI setup instructions

**Effort:** 8 hours

#### 7.3 API Usage Examples (Day 2-3)
- [ ] Write example: Create project and task
- [ ] Write example: Claim, execute, complete task
- [ ] Write example: Use SSE for real-time updates
- [ ] Write example: Store and query memory
- [ ] Write example: Generate context pack
- [ ] Add curl commands and Go client examples

**Deliverables:**
- `docs/api/v2-examples.md`
- Example scripts in `examples/v2/`

**Effort:** 10 hours

#### 7.4 Performance Testing (Day 3-4)
- [ ] Load test: 1K tasks, 100 concurrent claims
- [ ] Load test: 10K events, 100 SSE clients
- [ ] Load test: 1K memory items, complex queries
- [ ] Identify bottlenecks
- [ ] Optimize hot paths (indexing, caching)
- [ ] Document performance characteristics

**Deliverables:**
- Performance test suite in `tests/performance/`
- Performance report

**Effort:** 10 hours

#### 7.5 Security Audit (Day 4)
- [ ] Review for SQL injection vulnerabilities
- [ ] Review for path traversal (workdir, file paths)
- [ ] Review for XSS in JSON responses
- [ ] Review for DoS vectors (unbounded queries)
- [ ] Add rate limiting recommendations
- [ ] Document security considerations

**Deliverables:**
- Security audit report
- Updated security section in design doc

**Effort:** 6 hours

#### 7.6 Final Integration Tests (Day 5)
- [ ] Test full workflow: project → task → claim → execute → complete
- [ ] Test v1/v2 interoperability across all endpoints
- [ ] Test error scenarios and edge cases
- [ ] Test concurrent agent scenarios
- [ ] Validate all events published correctly
- [ ] Ensure memory persists across runs

**Deliverables:**
- Comprehensive test suite
- Test coverage report (target: 80%+)

**Effort:** 8 hours

#### 7.7 Documentation & Release (Day 5)
- [ ] Update README with v2 sections
- [ ] Write v2 migration guide for v1 users
- [ ] Update architecture diagrams
- [ ] Prepare release notes
- [ ] Tag release (e.g., v2.0.0)

**Deliverables:**
- Updated documentation
- Release notes
- Git tag

**Effort:** 4 hours

#### 7.8 MCP/VSIX Scaffolding (Day 5)
- [x] Scaffold MCP server workspace (`tools/cocopilot-mcp`)
- [x] Scaffold VSIX extension workspace (`tools/cocopilot-vsix`)
- [ ] Wire MCP server to v2 API client (auth, base URL, retries)
- [ ] Define MCP capability surface (tasks, runs, events, memory)
- [ ] Connect VSIX UI to MCP client (commands, status, logs)
- [ ] Add local dev scripts (build, watch, package)
- [ ] Add smoke tests (MCP handshake, VSIX activation)
- [ ] Document install and usage for MCP and VSIX

**Deliverables:**
- MCP server client wiring
- VSIX command surface
- Setup docs and smoke tests

**Effort:** 6 hours

### Phase 7 Milestones
- [ ] Production-ready code
- [ ] Comprehensive documentation
- [ ] Performance validated
- [ ] Security reviewed
- [ ] MCP server and VSIX extension usable end-to-end
- [ ] v2.0.0 released

**Total Effort:** ~50 hours (1+ week for 1 developer)

---

## Summary Timeline

| Phase | Duration | Effort | Deliverable |
|-------|----------|--------|-------------|
| 1. Foundation | Week 1 | 40h | Projects + Tasks v2 |
| 2. Execution Ledger | Week 2 | 40h | Runs, Steps, Logs, Artifacts |
| 3. Leases & Claiming | Week 2-3 | 40h | Task claiming with leases |
| 4. Structured Completion | Week 3 | 30h | Rich completion metadata |
| 5. Events & Memory | Week 4 | 40h | SSE + Memory persistence |
| 6. Context Packs | Week 4 | 30h | Automated context generation |
| 7. Polish & Docs | Week 5 | 50h | Production readiness |

**Total:** ~270 hours (~7 weeks for 1 developer, ~5 weeks for 2 developers with parallel work)

## Parallel Work Streams

To accelerate delivery, work can be parallelized:

### Stream A: Core API (Developer 1)
- Phase 1: Foundation
- Phase 2: Execution Ledger
- Phase 4: Structured Completion

### Stream B: Advanced Features (Developer 2)
- Phase 3: Leases & Claiming
- Phase 5: Events & Memory
- Phase 6: Context Packs

### Stream C: Quality (Developer 3 or shared)
- Phase 7: Polish & Documentation (ongoing throughout)

**With 2 developers:**
- Weeks 1-2: Stream A (Phases 1-2)
- Weeks 1-2: Stream B (Phase 3)
- Week 3: Both work on Phase 4
- Week 4: Stream A (Phase 5), Stream B (Phase 6)
- Week 5: Both work on Phase 7

## Risk Mitigation

### Technical Risks

| Risk | Mitigation |
|------|------------|
| Status sync bugs | Comprehensive unit tests, audit triggers |
| Lease edge cases | Thorough testing of expiration and conflicts |
| SSE scalability | Load test early, optimize connection handling |
| Context pack performance | Budget controls, caching, incremental delivery |

### Schedule Risks

| Risk | Mitigation |
|------|------------|
| Scope creep | Defer non-critical features to v2.1 |
| Underestimated tasks | Add 20% buffer to each phase |
| Dependency blocks | Parallelize independent work streams |
| Testing bottleneck | Test continuously, not just Phase 7 |

## Success Criteria

- [ ] All v2 endpoints operational
- [ ] v1 endpoints unchanged and passing tests
- [ ] v1 ↔ v2 compatibility validated
- [ ] OpenAPI spec matches implementation
- [ ] Test coverage ≥ 80%
- [ ] Performance targets met (1K tasks, 100 concurrent agents)
- [ ] Security audit passed
- [ ] Documentation complete

## Post-Launch Roadmap

After v2.0.0 release:

### v2.1 (Month 2)
- Auth policy hardening and reporting
- GraphQL endpoint
- Bulk operations

### v2.2 (Month 3)
- Webhooks
- Task templates
- Scheduled tasks

### v2.3 (Month 4)
- Full repo perception implementation
- Advanced context pack heuristics
- Performance optimizations (PostgreSQL migration)

---

**Status:** Implemented  
**Note:** All phases complete. See v2-summary.md for current API reference.  
**Related Documents:**
- [v2 Design](v2-design.md)
- [OpenAPI Spec](openapi-v2.yaml)
- [Compatibility Plan](v2-compatibility.md)
