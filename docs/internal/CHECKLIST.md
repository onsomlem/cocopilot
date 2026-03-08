# Cocopilot Implementation Checklist

> Progress tracker for completing the roadmap. Check items as they are completed.

## Current State Summary

**Implemented:**
- ✅ Core schema (migrations 0001-0019)
- ✅ Full v1 and v2 API handlers
- ✅ Automation engine (`task.completed` trigger)
- ✅ Automation emission dedupe + throttle
- ✅ Policy persistence (CRUD) and runtime enforcement
- ✅ UI pages (Kanban, agents, audit, memory, runs, graphs, context-packs)
- ✅ MCP/VSIX packaging

**Remaining:**
- ✅ All items complete!

---

## B1: Policy Runtime Enforcement

**Goal:** Enforce stored policies at API boundaries (rate limits, workflow constraints)

### B1.1: Policy Evaluation Engine
- [x] Create `policy_engine.go` with core evaluation logic
- [x] Define `PolicyContext` struct containing: `project_id`, `agent_id`, `action`, `resource_type`, `resource_id`, `timestamp`
- [x] Implement `EvaluatePolicy(ctx PolicyContext, policies []Policy) (allowed bool, violations []string)`
- [x] Support policy types:
  - [x] `rate_limit`: Track request counts per agent/project in sliding window
  - [x] `workflow_constraint`: Validate task state transitions
  - [x] `resource_quota`: Limit concurrent tasks/runs per project
  - [x] `time_window`: Restrict operations to allowed hours

### B1.2: Rate Limiting Infrastructure
- [x] Add in-memory rate limiter (token bucket or sliding window)
- [x] Create `rate_limiter.go` with `RateLimiter` interface
- [x] Implement `CheckRateLimit(projectID, agentID string, limit int, window time.Duration) bool`
- [x] Add migration `0019_rate_limit_state.sql` for persistent rate limit state (optional)

### B1.3: Middleware Integration
- [x] Add `policyEnforcementMiddleware` in `main.go`
- [x] Hook middleware into v2 API routes:
  - [x] `POST /api/v2/tasks` - check task creation limits
  - [x] `POST /api/v2/tasks/{id}/claim` - check claim permissions
  - [x] `POST /api/v2/runs` - check run creation limits
- [x] Return structured error `{"error": {"code": "POLICY_VIOLATION", ...}}`

### B1.4: Policy CRUD Enhancements
- [x] Add `enabled` field to policies table (migration `0019_policies_enabled.sql`)
- [x] Update `GET /api/v2/projects/{id}/policies` to filter by enabled
- [x] Add `POST /api/v2/projects/{id}/policies/{policy_id}/enable`
- [x] Add `POST /api/v2/projects/{id}/policies/{policy_id}/disable`

### B1.5: Testing
- [x] Unit tests for `EvaluatePolicy` with various policy types
- [x] Integration tests for middleware blocking requests
- [x] Test policy enable/disable behavior
- [x] Test rate limit reset after window expires

---

## B2: Automation Governance

**Goal:** Add safety controls to prevent runaway automation

### B2.1: Recursion Depth Tracking
- [x] Add `automation_depth` column to tasks table (migration `0020_automation_depth.sql`)
- [x] Modify `executeAutomationAction` in `automation.go` to:
  - [x] Read parent task's automation_depth
  - [x] Increment depth for child tasks
  - [x] Reject if depth exceeds configurable max (default: 5)
- [x] Add `COCO_MAX_AUTOMATION_DEPTH` env var

### B2.2: Rate Limiting for Automation
- [x] Track automation executions per project in sliding window
- [x] Add `COCO_AUTOMATION_RATE_LIMIT` env var (default: 100/hour)
- [x] Add `COCO_AUTOMATION_BURST_LIMIT` env var (default: 10/minute)
- [x] Log warning when approaching limits

### B2.3: Circuit Breaker
- [x] Implement circuit breaker pattern for automation
- [x] Track failure rate per automation rule
- [x] Open circuit after N consecutive failures (configurable)
- [x] Add `automation_circuit_state` to in-memory state
- [x] Auto-reset after cooldown period

### B2.4: Audit Trail
- [x] Emit `automation.triggered` event with rule details
- [x] Emit `automation.blocked` event when governance rejects
- [x] Emit `automation.circuit_opened` event
- [x] Add `GET /api/v2/projects/{id}/automation/stats` endpoint

### B2.5: Testing
- [x] Test recursion depth limit enforcement
- [x] Test rate limiting blocks excessive automation
- [x] Test circuit breaker opens after failures
- [x] Test audit events are emitted correctly

### B2.6: Emission Dedupe + Throttle
- [x] Add migration `0019_automation_emissions.sql` with dedupe table
- [x] Implement `computeEmissionDedupeKey(projectID, kind, windowSeconds)` using SHA256
- [x] Implement `TryRecordEmission(db, projectID, kind, taskID)` - atomic insert-or-ignore
- [x] Implement `CheckEmissionAllowed(db, projectID, kind)` - read-only check
- [x] Implement `CleanupOldEmissions(db, maxAgeSeconds)` - garbage collection
- [x] Add configurable window via `SetEmissionWindow(seconds)` (default: 5 min)
- [x] Wire into v1 `/task` endpoint when no rows (idle planner trigger)
- [x] Wire into v2 claim-next endpoint when no runnable task
- [x] Add tests for dedupe behavior across time windows

---

## B3: `repo_files` Feature (File Metadata Persistence)

**Goal:** Persist file metadata for richer context generation

### B3.1: Database Schema
- [x] Create `migrations/0021_repo_files.sql`:
  ```sql
  CREATE TABLE repo_files (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    path TEXT NOT NULL,
    content_hash TEXT,
    size_bytes INTEGER,
    language TEXT,
    last_modified TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    metadata_json TEXT,
    UNIQUE(project_id, path)
  );
  ```
- [x] Add indexes for project_path and language

### B3.2: Models
- [x] Add `RepoFile` struct to `models_v2.go`:
  - [x] Fields: `ID`, `ProjectID`, `Path`, `ContentHash`, `SizeBytes`, `Language`, `LastModified`, `CreatedAt`, `UpdatedAt`, `Metadata`

### B3.3: Database Operations
- [x] Add to `db_v2.go`:
  - [x] `UpsertRepoFile(file RepoFile) error`
  - [x] `GetRepoFile(projectID, path string) (*RepoFile, error)`
  - [x] `ListRepoFiles(projectID string, opts ListRepoFilesOpts) ([]RepoFile, error)`
  - [x] `DeleteRepoFile(projectID, path string) error`
  - [x] `DeleteRepoFilesByProject(projectID string) error`

### B3.4: API Endpoints
- [x] `GET /api/v2/projects/{id}/files` - list files with pagination/filtering
- [x] `GET /api/v2/projects/{id}/files/{path}` - get single file metadata
- [x] `PUT /api/v2/projects/{id}/files/{path}` - upsert file metadata
- [x] `DELETE /api/v2/projects/{id}/files/{path}` - remove file
- [x] `POST /api/v2/projects/{id}/files/sync` - bulk sync from filesystem

### B3.5: File System Scanner
- [x] Create `scanner.go` with `ScanProjectFiles(projectID, workdir string) ([]RepoFile, error)`
- [x] Respect `.gitignore` patterns
- [x] Detect language from file extension
- [x] Compute content hash (SHA256 of first 64KB)
- [x] Add `COCO_FILE_SCAN_MAX_SIZE` env var (default: 1MB)

### B3.6: Context Pack Integration
- [x] Modify context pack builder to optionally include repo_files metadata
- [x] Add `include_file_metadata` option to context pack creation

### B3.7: Testing
- [x] Test CRUD operations for repo_files
- [x] Test file scanner respects gitignore
- [x] Test language detection
- [x] Test context pack includes file metadata

---

## B4: MCP/VSIX Packaging

**Goal:** Package tools for VS Code marketplace distribution

### B4.1: MCP Server Completion
- [x] Review `tools/cocopilot-mcp/src/index.ts` for completeness
- [x] Ensure all required MCP tools are implemented:
  - [x] `coco.task.create`
  - [x] `coco.task.list`
  - [x] `coco.task.get`
  - [x] `coco.task.claim`
  - [x] `coco.task.complete`
  - [x] `coco.context_pack.build`
  - [x] `coco.memory.get`
  - [x] `coco.memory.set`
- [x] Add MCP resources for read-only data access
- [x] Add MCP prompts for common workflows

### B4.2: MCP Build Configuration
- [x] Update `tools/cocopilot-mcp/package.json`:
  - [x] Set correct `name`, `version`, `description`
  - [x] Add `bin` entry for CLI
  - [x] Add `files` array for npm publish
- [x] Add `tools/cocopilot-mcp/tsconfig.json` if missing
- [x] Add build script: `npm run build` → `tsc`
- [x] Add `tools/cocopilot-mcp/.npmignore`

### B4.3: VSIX Extension
- [x] Review `tools/cocopilot-vsix/package.json`
- [x] Implement extension activation in `extension.ts`:
  - [x] Auto-register MCP server on activation
  - [x] Handle MCP server lifecycle
- [x] Add configuration options:
  - [x] `cocopilot.serverUrl` - Cocopilot server URL
  - [x] `cocopilot.apiKey` - Optional API key
  - [x] `cocopilot.autoStart` - Auto-start MCP server
- [x] Add commands:
  - [x] `cocopilot.startServer`
  - [x] `cocopilot.stopServer`
  - [x] `cocopilot.showTasks`

### B4.4: VSIX Build Configuration
- [x] Add `tools/cocopilot-vsix/tsconfig.json`
- [x] Add `tools/cocopilot-vsix/.vscodeignore`
- [x] Add build script: `npm run build` → `tsc`
- [x] Add package script: `npm run package` → `vsce package`
- [x] Test extension loads in VS Code

### B4.5: Documentation
- [x] Create `tools/cocopilot-mcp/README.md` with:
  - [x] Installation instructions
  - [x] Configuration options
  - [x] Available tools list
  - [x] Example usage
- [x] Create `tools/cocopilot-vsix/README.md` with:
  - [x] Installation from VSIX
  - [x] Configuration settings
  - [x] Feature overview
- [x] Add `CHANGELOG.md` to both packages

### B4.6: CI/CD (Optional)
- [x] Add GitHub Actions workflow for MCP npm publish
- [x] Add GitHub Actions workflow for VSIX build/release
- [x] Add version bump scripts

### B4.7: Testing
- [x] Manual test MCP server with Claude Desktop
- [x] Manual test VSIX in VS Code
- [x] Test MCP ↔ Cocopilot server communication

---

## Recommended Execution Order

1. **B1: Policy Enforcement** - foundational for governance
2. **B2: Automation Governance** - builds on policies
3. **B4: MCP/VSIX Packaging** - enables distribution
4. **B3: `repo_files`** - optional enhancement

---

## Progress Summary

| Section | Total | Complete | Remaining |
|---------|-------|----------|-----------|
| B1: Policy Enforcement | 23 | 23 | 0 |
| B2: Automation Governance | 26 | 26 | 0 |
| B3: repo_files Feature | 22 | 22 | 0 |
| B4: MCP/VSIX Packaging | 30 | 30 | 0 |
| **Total** | **101** | **101** | **0** |
