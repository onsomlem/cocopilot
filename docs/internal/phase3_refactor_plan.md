# Phase 3 Package Refactor - Implementation Plan

**Date**: 2026-03-05  
**Status**: Planning  
**Task ID**: 156  
**Scope**: Refactor main.go handlers into internal packages (cmd/internal/pkg layout)

## Executive Summary

This document provides a detailed implementation plan for Phase 3 of the Master Spec: refactoring the monolithic main.go file (~10,927 LOC) into a structured package layout. The goal is to improve maintainability, testability, and architectural clarity without breaking existing functionality.

### Key Metrics
- **Current main.go size**: 10,927 lines of code
- **Already separated**: ~5,626 LOC in domain files (models_v2.go, db_v2.go, automation.go, assignment.go, policy_engine.go, rate_limiter.go, scanner.go, migrations.go)
- **Remaining to refactor**: ~5,301 LOC (handlers, UI, routing, helpers)
- **Target package count**: 15 internal packages + 1 pkg package
- **Estimated migration effort**: 8-12 days (assuming 1-2 packages per day)
- **Test files requiring updates**: ~40 test files

### Guiding Principles
1. **No runtime behavior changes** - All refactoring must preserve existing API contracts
2. **Incremental migration** - Packages can be extracted one at a time
3. **Test coverage maintained** - All existing tests must pass after each migration
4. **Backward compatibility** - v1 and v2 APIs remain unchanged
5. **Clear ownership** - Each package has a single, well-defined responsibility

---

## 1. Current State Analysis

### 1.1 Main.go Structure Breakdown

| Section | Lines (est) | Description |
|---------|-------------|-------------|
| Config & Setup | ~500 | Runtime config, env parsing, DB init, globals |
| V1 Legacy Handlers | ~600 | /task, /save, /create, /update-status, /events endpoints |
| V2 Core Handlers | ~2500 | /api/v2/tasks, events, runs, agents, leases |
| V2 Project Handlers | ~1200 | /api/v2/projects/* endpoints |
| V2 Specialized Handlers | ~600 | Automation, policies, memory, context packs, repo files |
| UI Rendering | ~800 | Kanban board, health dashboard, placeholder UIs |
| Routing & Middleware | ~200 | Route setup, mux configuration, auth wrappers |
| Helper Functions | ~500 | JSON helpers, validation, normalization, error formatting |
| Transaction Helpers | ~200 | claimNextTaskTx, spawnIdlePlannerTx |
| Type Definitions | ~300 | Request/response structs, enumerations |
| SSE Broadcasting | ~100 | v1/v2 event subscriber management |
| Context Assembly | ~200 | Already in assignment.go, referenced from handlers |
| Main & CLI | ~200 | Entrypoint, CLI handling, server startup |

### 1.2 Already Separated (✅ Completed)

These files already exist and contain well-defined domain logic:

| File | LOC | Purpose |
|------|-----|---------|
| `models_v2.go` | 344 | Data models: Project, TaskV2, Run, Lease, Agent, Event, Memory, Policy |
| `db_v2.go` | 2927 | CRUD operations for all v2 entities |
| `automation.go` | 972 | Automation rules engine, event processing, emission dedupe |
| `assignment.go` | 263 | Canonical assignment lifecycle: ClaimTaskByID, CompleteTask, FailTask, assembleContext |
| `policy_engine.go` | 376 | Policy evaluation, quota enforcement |
| `rate_limiter.go` | 143 | Rate limiting and circuit breaker |
| `scanner.go` | 290 | Repo file scanning, git integration |
| `migrations.go` | 311 | Schema migration system |

**Total**: 5,626 LOC already separated ✅

### 1.3 Remaining in main.go (🔴 To Migrate)

| Category | LOC | Status |
|----------|-----|--------|
| HTTP Handlers | ~4500 | 🔴 All in main.go |
| UI Rendering | ~800 | 🔴 All in main.go |
| Routing/Middleware | ~200 | 🔴 All in main.go |
| Helpers & Utils | ~500 | 🔴 All in main.go |
| Config & Init | ~500 | 🔴 All in main.go |

**Total remaining**: ~5,301 LOC to refactor

---

## 2. Target Package Layout

Based on Master Spec Section 11, the target structure is:

```
cmd/cocopilot/main.go              # Entrypoint only (~100-150 LOC)
internal/
  config/                          # Runtime configuration
    config.go                      # Config struct, env parsing
    validation.go                  # Workdir, auth validation
  http/
    legacy/                        # V1 compatibility layer
      tasks.go                     # GET /task, POST /save, POST /create
      events.go                    # GET /events (v1 SSE)
      workdir.go                   # GET/POST /api/workdir
    v2/                            # V2 API handlers
      tasks.go                     # Task CRUD, claim, complete
      events.go                    # Event list, stream
      runs.go                      # Run operations
      agents.go                    # Agent registration, heartbeat
      projects.go                  # Project CRUD
      leases.go                    # Lease create, renew, release
      middleware.go                # Auth, error wrapping
      routing.go                   # Route registration
  projects/                        # Project domain operations
    service.go                     # Project business logic
    tree.go                        # Project tree snapshot
    changes.go                     # Git change detection
  tasks/                           # Task domain operations
    service.go                     # Task business logic
    dependencies.go                # Dependency graph
    queue.go                       # Task queue eligibility logic
  assignments/                     # Assignment lifecycle (already in assignment.go)
    assignment.go                  # ClaimTaskByID, CompleteTask, FailTask
    context.go                     # Context assembly
  leases/                          # Lease management (some in db_v2.go)
    service.go                     # Lease CRUD, expiry checks
  runs/                            # Run execution tracking (some in db_v2.go)
    service.go                     # Run CRUD, steps, logs, artifacts
  context/                         # Context assembly & invalidation
    assembly.go                    # assembleContext (from assignment.go)
    invalidation.go                # Context freshness logic
  memory/                          # Memory persistence
    service.go                     # Memory CRUD
    handlers.go                    # HTTP handlers
  repo/                            # Repo intelligence (scanner.go)
    scanner.go                     # File scanning, git operations
    files.go                       # Repo file DB operations
  automation/                      # Automation engine (automation.go)
    engine.go                      # Rule processing
    templates.go                   # Template rendering
    emission.go                    # Dedupe logic
  policy/                          # Policy engine (policy_engine.go)
    engine.go                      # Policy evaluation
    middleware.go                  # Policy enforcement
  events/                          # Event recording & streaming
    service.go                     # Event CRUD
    broadcast.go                   # SSE fanout (v1/v2)
  readmodels/                      # Fast projections
    metrics.go                     # Dashboard metrics
    health.go                      # Project health
  ui/                              # UI generation
    kanban.go                      # Kanban board rendering
    dashboard.go                   # Health dashboard
    placeholders.go                # Placeholder UIs
  auth/                            # Authentication
    auth.go                        # API key validation, identity resolution
pkg/
  contracts/                       # Shared API contracts (future)
    requests.go                    # Request DTOs
    responses.go                   # Response DTOs
```

---

## 3. Detailed Migration Plan by Package

### 3.1 Package: `internal/config` (Priority: P0 - Foundation)

**Purpose**: Centralize runtime configuration, environment variable parsing, and validation.

**Current Location**: Lines 147-484 in main.go

**Functions to Move**:
- `loadRuntimeConfig()` (~208 LOC)
- `getEnvConfigValue()` (~13 LOC)
- `getEnvBoolValue()` (~16 LOC)
- `parseScopeSet()` (~16 LOC)
- `parseAuthIdentities()` (~45 LOC)
- `validateWorkdir()` (~25 LOC)
- `resolveSSEHeartbeatInterval()` (~6 LOC)
- `resolveSSEReplayLimitMax()` (~7 LOC)
- `resolveEventsPruneInterval()` (~8 LOC)

**Types to Move**:
- `runtimeConfig` struct
- `authIdentity` struct
- Constants (defaultSSEHeartbeatSeconds, etc.)

**Estimated LOC**: ~350

**Dependencies**: None

**Test Files to Update**:
- `v2_config_test.go` (config endpoint tests)
- New file: `config_test.go` (unit tests for env parsing)

**Migration Steps**:
1. Create `internal/config/config.go`
2. Move structs and constants
3. Move validation functions
4. Move env parsing functions
5. Export necessary symbols (LoadConfig, ValidateWorkdir, etc.)
6. Update main.go to import `internal/config`
7. Run tests and verify

**Breaking Changes**: None (internal refactor only)

---

### 3.2 Package: `internal/auth` (Priority: P0 - Foundation)

**Purpose**: Authentication middleware, API key validation, identity resolution.

**Current Location**: Lines 10100-10120 in main.go (middleware), scattered auth checks

**Functions to Move**:
- `withV2MutationAuth()` (middleware wrapper)
- `withV1MutationAuth()` (middleware wrapper)
- `resolveAuthIdentity()` (identity lookup)
- `checkScopes()` (scope validation)

**Estimated LOC**: ~150

**Dependencies**: `internal/config`

**Test Files to Update**:
- All v2 mutation tests (create/update/delete tests check auth)
- New file: `auth_test.go`

**Migration Steps**:
1. Create `internal/auth/auth.go`
2. Move middleware wrappers
3. Extract identity resolution logic
4. Update main.go to use `internal/auth`
5. Verify all auth-protected endpoints still work

**Breaking Changes**: None

---

### 3.3 Package: `internal/http/v2` (Priority: P1 - Core Refactor)

**Purpose**: V2 API handlers, error formatting, routing setup.

**Current Location**: Lines 1483-7170 in main.go

**Files to Create**:

#### `internal/http/v2/errors.go` (~100 LOC)
- `v2Error` struct
- `v2ErrorEnvelope` struct
- `writeV2Error()`
- `writeV2JSON()`
- `writeV2MethodNotAllowed()`

#### `internal/http/v2/tasks.go` (~800 LOC)
- `v2ListTasksHandler()`
- `v2CreateTaskHandler()`
- `v2GetTaskDetailHandler()`
- `v2UpdateTaskHandler()`
- `v2DeleteTaskHandler()`
- `v2TaskClaimHandler()`
- `v2TaskCompleteHandler()`
- `v2TaskDependenciesHandler()`
- `v2TaskDependencyDetailHandler()`
- `v2TaskDetailRouteHandler()` (router)

#### `internal/http/v2/events.go` (~400 LOC)
- `v2ListEventsHandler()`
- `v2EventsStreamHandler()`
- Helper: `normalizeEventSince()`
- Helper: `resolveEventStreamSince()`

#### `internal/http/v2/runs.go` (~400 LOC)
- `v2RunsRouteHandler()`
- `v2GetRunHandler()`
- `v2CreateRunStepHandler()`
- `v2CreateRunLogHandler()`
- `v2CreateRunArtifactHandler()`

#### `internal/http/v2/agents.go` (~350 LOC)
- `v2RegisterAgentHandler()`
- `v2ListAgentsHandler()`
- `v2GetAgentHandler()`
- `v2DeleteAgentHandler()`
- `v2AgentActionHandler()` (heartbeat, etc.)
- `v2AgentsRouteHandler()` (router)

#### `internal/http/v2/projects.go` (~600 LOC)
- `v2CreateProjectHandler()`
- `v2ListProjectsHandler()`
- `v2GetProjectHandler()`
- `v2UpdateProjectHandler()`
- `v2DeleteProjectHandler()`
- `v2ProjectRouteHandler()` (router)

#### `internal/http/v2/leases.go` (~250 LOC)
- `v2LeaseHandler()`
- `v2CreateLeaseHandler()`
- `v2LeaseActionHandler()` (router)
- `v2LeaseHeartbeatHandler()`
- `v2LeaseReleaseHandler()`

#### `internal/http/v2/health.go` (~200 LOC)
- `v2HealthHandler()`
- `v2MetricsHandler()`
- `v2VersionHandler()`
- `v2ConfigHandler()`
- Helper: `getCurrentSchemaVersion()`
- Helper: `retentionSnapshot()`

#### `internal/http/v2/backup.go` (~150 LOC)
- `v2BackupHandler()`
- `v2RestoreHandler()`

#### `internal/http/v2/artifacts.go` (~100 LOC)
- `v2ArtifactCommentsHandler()`

#### `internal/http/v2/routing.go` (~200 LOC)
- `RegisterV2Routes(mux, cfg)` - central route registration
- `v2ProjectsRouteHandler()` - project subrouter
- `v2TasksRouteHandler()` - task subrouter

**Total Estimated LOC**: ~3,550

**Dependencies**:
- `internal/config`
- `internal/auth`
- `db_v2.go` (database operations)
- `assignment.go` (ClaimTaskByID, CompleteTask)
- `models_v2.go` (data structures)

**Test Files to Update** (28 files):
- `v2_task_*.go` (9 files)
- `v2_events_*.go` (3 files)
- `v2_agent_*.go` (2 files)
- `v2_project_*.go` (6 files)
- `v2_context_*.go` (2 files)
- `v2_memory_*.go` (2 files)
- `v2_policies_*.go` (2 files)
- `v2_repo_files_*.go` (1 file)
- `v2_config_test.go` (1 file)

**Migration Steps**:
1. Create `internal/http/v2/` directory
2. Start with `errors.go` (foundation)
3. Move handlers one file at a time (tasks, events, runs, agents, etc.)
4. Update import paths in test files after each handler migration
5. Verify all v2 API tests pass after each step
6. Create `routing.go` last to centralize route registration

**Breaking Changes**: None (all internal, handlers preserve signatures)

---

### 3.4 Package: `internal/http/legacy` (Priority: P1 - V1 Compat)

**Purpose**: V1 API compatibility layer for legacy endpoints.

**Current Location**: Lines 754-1460 in main.go

**Files to Create**:

#### `internal/http/legacy/tasks.go` (~400 LOC)
- `apiTasksHandler()` (deprecated v1 task list)
- `getTaskHandler()` (GET /task - claim next)
- `saveHandler()` (POST /save - complete task)
- `updateStatusHandler()` (POST /update-status)
- `createHandler()` (POST /create)
- `deleteTaskHandler()` (POST /delete)
- Helper: `parseTaskStatusFilter()`
- Helper: `resolveV1TaskListSort()`

#### `internal/http/legacy/events.go` (~300 LOC)
- `eventsHandler()` (SSE stream for v1)
- Helper: `normalizeV1EventType()`

#### `internal/http/legacy/workdir.go` (~100 LOC)
- `getWorkdirHandler()`
- `setWorkdirHandler()`
- `instructionsHandler()`

**Total Estimated LOC**: ~800

**Dependencies**:
- `internal/config`
- `internal/auth`
- `db_v2.go`
- `assignment.go` (ClaimTaskByID, CompleteTask)

**Test Files to Update**:
- `main_test.go` (v1 endpoints)
- New file: `legacy_test.go`

**Migration Steps**:
1. Create `internal/http/legacy/` directory
2. Move v1 handlers one file at a time
3. Update imports in main.go
4. Verify v1 API tests pass

**Breaking Changes**: None (v1 API preserved exactly)

---

### 3.5 Package: `internal/projects` (Priority: P2 - Domain Logic)

**Purpose**: Project-specific business logic, tree snapshot, git change detection.

**Current Location**: Lines 4908-5107, 5773-5856 in main.go

**Files to Create**:

#### `internal/projects/tree.go` (~250 LOC)
- `buildProjectTreeSnapshot()` (recursive directory walk)
- `TreeNode` type (if not in models)

#### `internal/projects/changes.go` (~150 LOC)
- `gitStatusChanges()` (git diff wrapper)
- `isGitRepo()` (git repo validation)

#### `internal/projects/service.go` (~100 LOC)
- Business logic wrappers around db_v2 operations
- Validation helpers

**Total Estimated LOC**: ~500

**Dependencies**:
- `db_v2.go` (GetProject, etc.)
- `models_v2.go`

**Test Files to Update**:
- `v2_project_tree_test.go`
- `v2_project_changes_test.go`

**Migration Steps**:
1. Create `internal/projects/` directory
2. Move tree snapshot logic
3. Move git change detection
4. Update handler imports
5. Verify project-related tests pass

**Breaking Changes**: None

---

### 3.6 Package: `internal/automation` (Priority: P2 - Already Exists)

**Purpose**: Consolidate automation logic (already in automation.go).

**Current Location**: automation.go (~972 LOC) + handlers in main.go (~400 LOC)

**Files to Create**:

#### `internal/automation/handlers.go` (~400 LOC)
- Move from main.go:
  - `v2ProjectAutomationRulesHandler()`
  - `v2ProjectAutomationSimulateHandler()`
  - `v2ProjectAutomationReplayHandler()`
  - `v2ProjectAutomationStatsHandler()`

#### `internal/automation/engine.go` (existing automation.go)
- Rename/move automation.go to internal/automation/engine.go

**Total Estimated LOC**: ~1,372 (existing + handlers)

**Dependencies**:
- `db_v2.go`
- `policy_engine.go`
- `rate_limiter.go`

**Test Files to Update**:
- `v2_project_automation_*.go` (3 files)
- `automation_governance_test.go`

**Migration Steps**:
1. Create `internal/automation/` directory
2. Move automation.go → engine.go
3. Move handlers from main.go → handlers.go
4. Update imports
5. Verify automation tests pass

**Breaking Changes**: None

---

### 3.7 Package: `internal/policy` (Priority: P3 - Already Exists)

**Purpose**: Consolidate policy logic (already in policy_engine.go).

**Current Location**: policy_engine.go (~376 LOC) + handlers in main.go (~300 LOC)

**Files to Create**:

#### `internal/policy/handlers.go` (~300 LOC)
- Move from main.go:
  - `v2ProjectPoliciesHandler()`
  - `v2ProjectPolicyDetailHandler()`

#### `internal/policy/engine.go` (existing policy_engine.go)
- Rename/move policy_engine.go

**Total Estimated LOC**: ~676

**Dependencies**:
- `db_v2.go`

**Test Files to Update**:
- `v2_policies_test.go`
- `v2_policy_enable_disable_test.go`
- `policy_enforcement_test.go`
- `policy_engine_test.go`
- `policy_middleware_test.go`

**Migration Steps**:
1. Create `internal/policy/` directory
2. Move policy_engine.go → engine.go
3. Move handlers from main.go
4. Update imports
5. Verify policy tests pass

**Breaking Changes**: None

---

### 3.8 Package: `internal/memory` (Priority: P3 - Specialized)

**Purpose**: Memory persistence handlers.

**Current Location**: Lines 6672-6791 in main.go

**Files to Create**:

#### `internal/memory/handlers.go` (~120 LOC)
- `v2ProjectMemoryHandler()` (GET/POST/PUT/DELETE memory)

**Total Estimated LOC**: ~120

**Dependencies**:
- `db_v2.go` (Memory CRUD operations)

**Test Files to Update**:
- `v2_memory_get_test.go`
- `v2_memory_put_test.go`

**Migration Steps**:
1. Create `internal/memory/` directory
2. Move handlers
3. Update imports
4. Verify memory tests pass

**Breaking Changes**: None

---

### 3.9 Package: `internal/repo` (Priority: P3 - Already Exists)

**Purpose**: Repo intelligence (already in scanner.go).

**Current Location**: scanner.go (~290 LOC) + handlers in main.go (~400 LOC)

**Files to Create**:

#### `internal/repo/handlers.go` (~400 LOC)
- Move from main.go:
  - `v2ProjectFilesHandler()`
  - `v2ProjectFileDetailHandler()`
  - `v2ProjectFilesScanHandler()`
  - `v2ProjectFilesSyncHandler()`

#### `internal/repo/scanner.go` (existing scanner.go)
- Rename/move scanner.go

**Total Estimated LOC**: ~690

**Dependencies**:
- `db_v2.go`

**Test Files to Update**:
- `v2_repo_files_test.go`
- `repo_files_db_test.go`
- `scanner_test.go`

**Migration Steps**:
1. Create `internal/repo/` directory
2. Move scanner.go
3. Move handlers from main.go
4. Update imports
5. Verify repo tests pass

**Breaking Changes**: None

---

### 3.10 Package: `internal/events` (Priority: P2 - Core Infrastructure)

**Purpose**: Event recording, SSE broadcasting (v1 and v2).

**Current Location**: Lines 527-560, 2283-2698 in main.go

**Files to Create**:

#### `internal/events/broadcast.go` (~150 LOC)
- `broadcastUpdate()` (v1 SSE)
- `publishV2Event()` (v2 SSE)
- `v1SSEClient` struct
- `v2EventSubscriber` struct
- SSE subscriber management

#### `internal/events/handlers.go` (moved from v2/)
- Event list and stream handlers (already part of v2 refactor)

**Total Estimated LOC**: ~150 (broadcast only, handlers in v2/)

**Dependencies**:
- `db_v2.go`

**Test Files to Update**:
- `v2_events_*.go` (already covered in v2 refactor)

**Migration Steps**:
1. Create `internal/events/` directory
2. Move SSE broadcast logic
3. Update imports in legacy/v2 handlers
4. Verify event streaming tests pass

**Breaking Changes**: None

---

### 3.11 Package: `internal/ui` (Priority: P4 - Low Priority)

**Purpose**: UI rendering (Kanban board, dashboards, placeholders).

**Current Location**: Lines 7171-8227 in main.go

**Files to Create**:

#### `internal/ui/kanban.go` (~300 LOC)
- `indexHandler()` (main Kanban board)
- HTML generation helpers

#### `internal/ui/dashboard.go` (~300 LOC)
- `healthDashboardHandler()`
- `repoGraphHandler()`
- `diffViewerHandler()`

#### `internal/ui/placeholders.go` (~200 LOC)
- `uiPlaceholderHandler()`
- `taskGraphsPlaceholderHandler()`
- `memoryPlaceholderHandler()`
- `agentsPlaceholderHandler()`
- `auditPlaceholderHandler()`
- `repoPlaceholderHandler()`
- `runsPlaceholderHandler()`
- `contextPacksPlaceholderHandler()`

**Total Estimated LOC**: ~800

**Dependencies**:
- `db_v2.go`
- `models_v2.go`

**Test Files to Update**:
- None (UI is not currently tested)

**Migration Steps**:
1. Create `internal/ui/` directory
2. Move UI handlers
3. Update routing in main.go
4. Manual browser testing

**Breaking Changes**: None

---

### 3.12 Package: `cmd/cocopilot` (Priority: P0 - Entrypoint)

**Purpose**: Minimal entrypoint, CLI handling, server startup.

**Current Location**: Lines 10306-10927 in main.go

**Files**:

#### `cmd/cocopilot/main.go` (~150 LOC)
- `main()` function
- `handleCLI()` for CLI commands
- Server startup logic
- Route registration (calls routing packages)
- Global var initialization (db, policyEngine)

**Total Estimated LOC**: ~150

**Dependencies**: ALL internal packages

**Migration Steps**:
1. Create `cmd/cocopilot/` directory
2. Move minimal startup logic
3. Import all internal packages
4. Register routes via routing packages
5. Verify server starts and all tests pass

**Breaking Changes**: None

---

## 4. Migration Sequence & Timeline

The migration should proceed in dependency order to minimize breakage:

| Phase | Packages | Days | LOC | Dependencies |
|-------|----------|------|-----|--------------|
| **Phase A: Foundation** | config, auth | 1-2 | 500 | None |
| **Phase B: Core HTTP** | http/v2, http/legacy | 3-4 | 4,350 | config, auth |
| **Phase C: Domain Logic** | projects, events | 2 | 650 | config, db_v2 |
| **Phase D: Specialized** | automation, policy, memory, repo | 2-3 | 2,658 | db_v2, policy_engine, automation |
| **Phase E: UI** | ui | 1 | 800 | db_v2 |
| **Phase F: Entrypoint** | cmd/cocopilot | 1 | 150 | ALL |

**Total Estimated Time**: 10-13 days

### Detailed Week-by-Week Plan

#### Week 1: Foundation & Core HTTP (Days 1-5)
- Day 1: `internal/config` + `internal/auth`
- Day 2: `internal/http/v2/errors.go` + `internal/http/v2/tasks.go`
- Day 3: `internal/http/v2/events.go` + `internal/http/v2/runs.go`
- Day 4: `internal/http/v2/agents.go` + `internal/http/v2/projects.go`
- Day 5: `internal/http/v2/leases.go` + `internal/http/v2/health.go` + `internal/http/v2/routing.go`

#### Week 2: Legacy, Domain, Specialized (Days 6-10)
- Day 6: `internal/http/legacy/` (all v1 handlers)
- Day 7: `internal/projects/` + `internal/events/`
- Day 8: `internal/automation/` + `internal/policy/`
- Day 9: `internal/memory/` + `internal/repo/`
- Day 10: `internal/ui/` + `cmd/cocopilot/main.go`

#### Week 3: Cleanup & Verification (Days 11-13)
- Day 11: Run full test suite, fix failing tests
- Day 12: Manual testing of all endpoints
- Day 13: Documentation updates, final review

---

## 5. Line of Code (LOC) Estimates by Package

| Package | LOC | Status | Priority |
|---------|-----|--------|----------|
| `internal/config` | 350 | New | P0 |
| `internal/auth` | 150 | New | P0 |
| `internal/http/v2/errors.go` | 100 | New | P1 |
| `internal/http/v2/tasks.go` | 800 | New | P1 |
| `internal/http/v2/events.go` | 400 | New | P1 |
| `internal/http/v2/runs.go` | 400 | New | P1 |
| `internal/http/v2/agents.go` | 350 | New | P1 |
| `internal/http/v2/projects.go` | 600 | New | P1 |
| `internal/http/v2/leases.go` | 250 | New | P1 |
| `internal/http/v2/health.go` | 200 | New | P1 |
| `internal/http/v2/backup.go` | 150 | New | P1 |
| `internal/http/v2/artifacts.go` | 100 | New | P1 |
| `internal/http/v2/routing.go` | 200 | New | P1 |
| `internal/http/legacy/tasks.go` | 400 | New | P1 |
| `internal/http/legacy/events.go` | 300 | New | P1 |
| `internal/http/legacy/workdir.go` | 100 | New | P1 |
| `internal/projects/tree.go` | 250 | New | P2 |
| `internal/projects/changes.go` | 150 | New | P2 |
| `internal/projects/service.go` | 100 | New | P2 |
| `internal/automation/handlers.go` | 400 | New | P2 |
| `internal/policy/handlers.go` | 300 | New | P3 |
| `internal/memory/handlers.go` | 120 | New | P3 |
| `internal/repo/handlers.go` | 400 | New | P3 |
| `internal/events/broadcast.go` | 150 | New | P2 |
| `internal/ui/kanban.go` | 300 | New | P4 |
| `internal/ui/dashboard.go` | 300 | New | P4 |
| `internal/ui/placeholders.go` | 200 | New | P4 |
| `cmd/cocopilot/main.go` | 150 | New | P0 |
| **TOTAL NEW CODE** | **6,970** | | |

**Note**: Some code will remain in root-level files:
- `models_v2.go` (344 LOC) - Keep as is (used by all packages)
- `db_v2.go` (2,927 LOC) - Keep as is (database layer)
- `migrations.go` (311 LOC) - Keep as is (schema management)
- `assignment.go` (263 LOC) - Move to `internal/assignments/` eventually
- `automation.go` (972 LOC) - Move to `internal/automation/engine.go`
- `policy_engine.go` (376 LOC) - Move to `internal/policy/engine.go`
- `rate_limiter.go` (143 LOC) - Move to `internal/policy/rate_limiter.go`
- `scanner.go` (290 LOC) - Move to `internal/repo/scanner.go`

**Final main.go**: ~0 LOC (remove entirely, replace with cmd/cocopilot/main.go)

---

## 6. Testing Strategy

### 6.1 Test Classification

| Test Type | Count | Approach |
|-----------|-------|----------|
| Unit Tests (existing) | ~40 files | Update imports, preserve assertions |
| Integration Tests | 0 | None currently (opportunity to add) |
| Manual Smoke Tests | N/A | Required after each phase |

### 6.2 Test Files Requiring Updates

#### V2 API Tests (28 files)
- `v2_task_claim_test.go` - Import `internal/http/v2`
- `v2_task_complete_test.go` - Import `internal/http/v2`
- `v2_task_create_test.go` - Import `internal/http/v2`
- `v2_task_delete_test.go` - Import `internal/http/v2`
- `v2_task_dependencies_test.go` - Import `internal/http/v2`
- `v2_task_detail_test.go` - Import `internal/http/v2`
- `v2_task_update_test.go` - Import `internal/http/v2`
- `v2_tasks_list_test.go` - Import `internal/http/v2`
- `v2_task_response_helpers_test.go` - Import `internal/http/v2`
- `v2_events_list_test.go` - Import `internal/http/v2`
- `v2_events_stream_test.go` - Import `internal/http/v2`
- `v2_events_replay_test.go` - Import `internal/http/v2`
- `v2_agent_delete_test.go` - Import `internal/http/v2`
- `v2_agent_detail_test.go` - Import `internal/http/v2`
- `v2_project_automation_replay_test.go` - Import `internal/automation`
- `v2_project_automation_rules_test.go` - Import `internal/automation`
- `v2_project_automation_simulate_test.go` - Import `internal/automation`
- `v2_project_changes_test.go` - Import `internal/projects`
- `v2_project_tree_test.go` - Import `internal/projects`
- `v2_project_task_create_test.go` - Import `internal/http/v2`
- `v2_project_tasks_list_test.go` - Import `internal/http/v2`
- `v2_project_audit_test.go` - Import `internal/http/v2`
- `v2_policies_test.go` - Import `internal/policy`
- `v2_policy_enable_disable_test.go` - Import `internal/policy`
- `v2_memory_get_test.go` - Import `internal/memory`
- `v2_memory_put_test.go` - Import `internal/memory`
- `v2_repo_files_test.go` - Import `internal/repo`
- `v2_config_test.go` - Import `internal/http/v2`

#### V1 / Legacy Tests (5 files)
- `main_test.go` - Import `internal/http/legacy`
- `v2_routes_test.go` - Import both `internal/http/v2` and `internal/http/legacy`

#### Domain Logic Tests (7 files)
- `automation_governance_test.go` - Import `internal/automation`
- `policy_enforcement_test.go` - Import `internal/policy`
- `policy_engine_test.go` - Import `internal/policy`
- `policy_middleware_test.go` - Import `internal/policy`
- `repo_files_db_test.go` - Import `internal/repo`
- `scanner_test.go` - Import `internal/repo`
- `rate_limiter_test.go` - Import `internal/policy`

**Total**: ~40 test files requiring import updates

### 6.3 Test Update Process (Per Package Migration)

1. **Before Migration**: Run full test suite, confirm all pass
2. **During Migration**: 
   - Move code to new package
   - Update imports in affected test files
   - Run affected tests: `go test -v -run <TestPattern>`
   - Fix import paths and package references
3. **After Migration**: 
   - Run full test suite: `go test ./...`
   - Confirm no regressions
   - Commit changes atomically (code + tests together)

### 6.4 Continuous Verification

After each package migration:
```bash
# Run all tests
go test -v ./...

# Check test coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### 6.5 Manual Testing Checklist

After completing all migrations, perform manual smoke tests:

#### V1 API Endpoints
- [ ] `GET /task` - Claim next task
- [ ] `POST /save` - Complete task
- [ ] `POST /create` - Create task
- [ ] `POST /update-status` - Update task status
- [ ] `GET /events` - SSE stream (v1)

#### V2 API Endpoints (Sample)
- [ ] `GET /api/v2/health` - Health check
- [ ] `GET /api/v2/metrics` - Metrics snapshot
- [ ] `GET /api/v2/tasks` - List tasks
- [ ] `POST /api/v2/tasks` - Create task
- [ ] `POST /api/v2/projects/{id}/tasks/claim-next` - Claim next task
- [ ] `POST /api/v2/tasks/{id}/complete` - Complete task
- [ ] `GET /api/v2/events/stream` - SSE stream (v2)

#### UI Endpoints
- [ ] `GET /` - Kanban board renders
- [ ] `GET /health` - Health dashboard renders
- [ ] `GET /agents` - Agents placeholder renders

---

## 7. Breaking Changes & Compatibility Concerns

### 7.1 No Breaking Changes Expected

This is a pure refactor. All of the following remain unchanged:
- ✅ HTTP endpoint paths
- ✅ Request/response JSON schemas
- ✅ Authentication mechanisms
- ✅ Database schema
- ✅ Environment variables
- ✅ CLI flags and commands
- ✅ SSE event formats
- ✅ Error response structures

### 7.2 Internal API Changes

**Package visibility changes**:
- Many functions currently exported from `main` will become internal
- Test files will need to import from internal packages
- MCP server (`tools/cocopilot-mcp`) may need to be updated if it imports internal functions (unlikely)

**Global variable handling**:
```go
// Current (main.go)
var db *sql.DB
var policyEngine *PolicyEngine

// After refactor (cmd/cocopilot/main.go)
var db *sql.DB  // Still global, passed to handlers via dependency injection or closure
```

### 7.3 Potential Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Test failures due to import changes | High | Update imports incrementally, run tests after each migration |
| Handler signature mismatches | Low | Keep http.HandlerFunc signatures unchanged |
| SSE broadcast breaks | Medium | Test event streaming thoroughly, preserve subscriber management |
| UI rendering breaks | Low | Manual browser testing after UI migration |
| Config parsing changes behavior | Low | Comprehensive config validation tests |
| Race conditions introduced | Low | Run tests with `-race` flag |

### 7.4 Rollback Strategy

If a package migration causes critical issues:
1. **Revert the specific package**: Use git to revert the commit for that package
2. **Keep other packages**: Other successfully migrated packages can remain
3. **Incremental approach**: Each package is atomic, so rollback is granular

---

## 8. Dependencies Between Packages

### 8.1 Dependency Graph

```
cmd/cocopilot/main.go
  ├─ internal/config (no deps)
  ├─ internal/auth (depends: config)
  ├─ internal/http/v2 (depends: config, auth, db_v2, assignment, models_v2)
  ├─ internal/http/legacy (depends: config, auth, db_v2, assignment)
  ├─ internal/projects (depends: db_v2, models_v2)
  ├─ internal/events (depends: db_v2, models_v2)
  ├─ internal/automation (depends: db_v2, policy, rate_limiter)
  ├─ internal/policy (depends: db_v2)
  ├─ internal/memory (depends: db_v2)
  ├─ internal/repo (depends: db_v2)
  └─ internal/ui (depends: db_v2, models_v2)

Shared (root level):
  - models_v2.go (no deps)
  - db_v2.go (depends: models_v2, migrations)
  - migrations.go (no deps)
```

### 8.2 Circular Dependency Prevention

**Design Rule**: Handlers depend on db_v2, but db_v2 must NOT depend on handlers.

**Allowed**:
```
internal/http/v2/tasks.go → db_v2.go (GetTaskV2)
internal/http/v2/tasks.go → assignment.go (ClaimTaskByID)
```

**Forbidden**:
```
db_v2.go → internal/http/v2/tasks.go  ❌
```

---

## 9. Post-Migration Improvements

After Phase 3 completion, the following improvements become possible:

### 9.1 Contract Generation (Master Spec Phase 4)
```
pkg/contracts/
  api.go          # Shared request/response DTOs
  validation.go   # Shared validation logic
```

This enables:
- Generating client SDKs (Go, Python, TypeScript)
- OpenAPI/Swagger documentation
- Contract testing

### 9.2 Service Layer Extraction

Move business logic from handlers to service layers:
```
internal/tasks/service.go
  ListTasks(filters TaskFilters) ([]TaskV2, error)
  CreateTask(req CreateTaskRequest) (*TaskV2, error)
  
internal/http/v2/tasks.go (handler)
  v2ListTasksHandler() {
    filters := parseFilters(r)
    tasks, err := taskService.ListTasks(filters)
    writeV2JSON(w, tasks)
  }
```

Benefits:
- Handlers become thin HTTP adapters
- Business logic is testable without HTTP mocking
- Easier to add gRPC or GraphQL endpoints later

### 9.3 Dependency Injection

Replace global variables with explicit dependency injection:
```go
// cmd/cocopilot/main.go
func main() {
  cfg := config.Load()
  db := initDB(cfg.DBPath)
  authMiddleware := auth.NewMiddleware(cfg)
  
  taskService := tasks.NewService(db)
  taskHandler := v2.NewTaskHandler(taskService, authMiddleware)
  
  mux.HandleFunc("/api/v2/tasks", taskHandler.List)
}
```

Benefits:
- Easier unit testing (mock dependencies)
- Clearer component boundaries
- Better lifecycle management

### 9.4 Integration Testing

Add end-to-end tests:
```go
// integration_test.go
func TestTaskLifecycle(t *testing.T) {
  server := testserver.New(t)
  defer server.Close()
  
  // Create task
  resp := server.POST("/api/v2/tasks", CreateTaskRequest{...})
  assert.Equal(t, 201, resp.StatusCode)
  
  // Claim task
  resp = server.POST("/api/v2/projects/proj_default/tasks/claim-next", ...)
  assert.Equal(t, 200, resp.StatusCode)
  
  // Complete task
  resp = server.POST("/api/v2/tasks/1/complete", ...)
  assert.Equal(t, 200, resp.StatusCode)
}
```

---

## 10. Success Criteria

Phase 3 is complete when:

### Technical Criteria
- [ ] main.go removed entirely, replaced with cmd/cocopilot/main.go (~150 LOC)
- [ ] All 15 internal packages created and populated
- [ ] All existing tests pass (`go test ./...` returns 0 failures)
- [ ] Test coverage remains >= current level (~70%)
- [ ] No race conditions detected (`go test -race ./...`)
- [ ] Server starts successfully and responds to all endpoints

### Functional Criteria
- [ ] All v1 API endpoints work identically to pre-refactor
- [ ] All v2 API endpoints work identically to pre-refactor
- [ ] SSE streaming (v1 and v2) functions correctly
- [ ] UI pages (Kanban, health dashboard) render correctly
- [ ] Authentication and authorization function correctly
- [ ] Database migrations still apply correctly
- [ ] Policy enforcement still blocks/allows correctly
- [ ] Automation rules still trigger correctly

### Quality Criteria
- [ ] No "god packages" (no package >1000 LOC except db_v2.go)
- [ ] Each package has a single, clear responsibility
- [ ] Package dependencies form a DAG (no cycles)
- [ ] All exported functions have godoc comments
- [ ] README.md updated with new architecture diagram

### Documentation Criteria
- [ ] `/docs/state/architecture.md` updated to reflect new package structure
- [ ] `AGENTS.md` updated with new import paths
- [ ] API docs (if any) remain accurate

---

## 11. Risks & Open Questions

### 11.1 Risks

1. **Test breakage cascade**: Updating 40 test files is error-prone
   - **Mitigation**: Incremental approach, test after each package
   
2. **Performance regression**: More packages = more imports = slightly slower compilation
   - **Mitigation**: Measure build times before/after, profile if needed
   
3. **SSE broadcast regression**: Global state for SSE subscribers is tricky
   - **Mitigation**: Extract broadcast logic early, test thoroughly
   
4. **UI rendering breaks**: HTML generation code is fragile
   - **Mitigation**: Manual testing, consider adding UI snapshot tests

5. **Merge conflicts**: Long-running refactor branch
   - **Mitigation**: Rebase frequently, merge incremental packages to main

### 11.2 Open Questions

1. **Should db_v2.go be further split?** 
   - Currently 2,927 LOC, violates "no god packages" rule
   - Candidate splits: `internal/db/{projects,tasks,runs,leases,agents,events,memory}.go`
   - **Recommendation**: Defer to Phase 4 or later

2. **Should assignment.go move to internal/?**
   - Currently 263 LOC, good size
   - **Recommendation**: Yes, move to `internal/assignments/assignment.go` in Phase C

3. **Should models_v2.go move to pkg/?**
   - Currently 344 LOC, domain models
   - **Recommendation**: Yes, move to `pkg/models/` or `internal/models/` in Phase 4 (contract lock)

4. **Should we add service layers now or later?**
   - Service layers (business logic) vs handlers (HTTP adapters)
   - **Recommendation**: Defer to post-Phase 3 improvement

5. **How to handle global variables (db, policyEngine)?**
   - **Option A**: Keep globals, pass by closure
   - **Option B**: Dependency injection
   - **Recommendation**: Option A for Phase 3, Option B for future improvement

---

## 12. Next Steps

### Immediate Actions (Before Starting Implementation)

1. **Review this plan** with team/stakeholders
2. **Create feature branch**: `git checkout -b phase3-package-refactor`
3. **Set up tracking**: Create a migration checklist (Markdown or issue tracker)
4. **Baseline metrics**: Run tests, record coverage and build time

### Implementation Kickoff

1. **Day 1 Morning**: Start with `internal/config` (smallest, no deps)
2. **After each package**: 
   - Run `go test ./...`
   - Commit with message: `refactor: migrate {package} from main.go`
3. **End of Week 1**: Review progress, adjust timeline if needed
4. **End of Week 2**: Code review, pair programming for tricky sections
5. **Week 3**: Final testing, documentation updates, merge to main

---

## 13. Appendix: File Inventory

### Current Root-Level Files (Before Refactor)

| File | LOC | Keep/Move | Destination |
|------|-----|-----------|-------------|
| `main.go` | 10,927 | Move | Split into internal/* and cmd/cocopilot/main.go |
| `models_v2.go` | 344 | Keep | (shared models, used everywhere) |
| `db_v2.go` | 2,927 | Keep | (database layer, consider splitting later) |
| `automation.go` | 972 | Move | internal/automation/engine.go |
| `assignment.go` | 263 | Move | internal/assignments/assignment.go |
| `policy_engine.go` | 376 | Move | internal/policy/engine.go |
| `rate_limiter.go` | 143 | Move | internal/policy/rate_limiter.go |
| `scanner.go` | 290 | Move | internal/repo/scanner.go |
| `migrations.go` | 311 | Keep | (schema management, foundational) |

### Target Structure (After Refactor)

```
cocopilot/
├─ cmd/
│  └─ cocopilot/
│     └─ main.go                     # ~150 LOC (entrypoint)
├─ internal/
│  ├─ config/
│  │  ├─ config.go                   # ~350 LOC
│  │  └─ config_test.go
│  ├─ auth/
│  │  ├─ auth.go                     # ~150 LOC
│  │  └─ auth_test.go
│  ├─ http/
│  │  ├─ legacy/
│  │  │  ├─ tasks.go                 # ~400 LOC
│  │  │  ├─ events.go                # ~300 LOC
│  │  │  ├─ workdir.go               # ~100 LOC
│  │  │  └─ legacy_test.go
│  │  └─ v2/
│  │     ├─ errors.go                # ~100 LOC
│  │     ├─ tasks.go                 # ~800 LOC
│  │     ├─ events.go                # ~400 LOC
│  │     ├─ runs.go                  # ~400 LOC
│  │     ├─ agents.go                # ~350 LOC
│  │     ├─ projects.go              # ~600 LOC
│  │     ├─ leases.go                # ~250 LOC
│  │     ├─ health.go                # ~200 LOC
│  │     ├─ backup.go                # ~150 LOC
│  │     ├─ artifacts.go             # ~100 LOC
│  │     └─ routing.go               # ~200 LOC
│  ├─ projects/
│  │  ├─ service.go                  # ~100 LOC
│  │  ├─ tree.go                     # ~250 LOC
│  │  └─ changes.go                  # ~150 LOC
│  ├─ tasks/
│  │  └─ service.go                  # (future)
│  ├─ assignments/
│  │  └─ assignment.go               # ~263 LOC (from assignment.go)
│  ├─ leases/
│  │  └─ service.go                  # (future)
│  ├─ runs/
│  │  └─ service.go                  # (future)
│  ├─ context/
│  │  └─ assembly.go                 # (future)
│  ├─ memory/
│  │  └─ handlers.go                 # ~120 LOC
│  ├─ repo/
│  │  ├─ scanner.go                  # ~290 LOC (from scanner.go)
│  │  └─ handlers.go                 # ~400 LOC
│  ├─ automation/
│  │  ├─ engine.go                   # ~972 LOC (from automation.go)
│  │  └─ handlers.go                 # ~400 LOC
│  ├─ policy/
│  │  ├─ engine.go                   # ~376 LOC (from policy_engine.go)
│  │  ├─ rate_limiter.go             # ~143 LOC (from rate_limiter.go)
│  │  └─ handlers.go                 # ~300 LOC
│  ├─ events/
│  │  └─ broadcast.go                # ~150 LOC
│  ├─ readmodels/
│  │  └─ metrics.go                  # (future)
│  └─ ui/
│     ├─ kanban.go                   # ~300 LOC
│     ├─ dashboard.go                # ~300 LOC
│     └─ placeholders.go             # ~200 LOC
├─ pkg/
│  └─ contracts/                     # (future - Phase 4)
├─ models_v2.go                      # 344 LOC (keep)
├─ db_v2.go                          # 2,927 LOC (keep)
├─ migrations.go                     # 311 LOC (keep)
├─ go.mod
├─ go.sum
├─ migrations/
│  └─ *.sql
├─ static/
│  └─ (embedded assets)
└─ *_test.go                         # 40 test files (update imports)
```

---

## Summary

This plan provides a comprehensive roadmap for refactoring main.go (~10,927 LOC) into a structured cmd/internal/pkg layout with 15 internal packages. The migration is designed to be:

- **Incremental**: Packages can be extracted one at a time
- **Safe**: No breaking changes to API contracts or database schema
- **Testable**: All existing tests updated and passing after each step
- **Maintainable**: Clear package boundaries and dependency graph

**Estimated Effort**: 10-13 days (2-3 weeks)  
**Risk Level**: Medium (test updates are tedious but straightforward)  
**Benefit**: Significantly improved code organization, maintainability, and testability

The refactor aligns with Master Spec Phase 3 objectives and sets the foundation for Phase 4 (contract locking) and beyond.

---

**End of Plan**

*Generated*: 2026-03-05  
*Author*: Assistant (Task 156)  
*Status*: Ready for Review & Implementation
