# POC Regression Test Suite

## Overview
This test suite ensures the existing proof-of-concept (v1) functionality remains stable as the system evolves. It covers the core task lifecycle, parent-child relationships, real-time updates, and workdir management.
For overall project status and completion context, see [COMPLETION_SUMMARY.md](COMPLETION_SUMMARY.md).
For the broader plan and milestones, see [ROADMAP.md](ROADMAP.md).

## Test Cases

### POC-REG-001: Create → Claim → Save Lifecycle
Tests the complete task lifecycle:
1. Create a task via `POST /create`
2. Claim the task via `GET /task` (status changes to IN_PROGRESS)
3. Save task output via `POST /save` (status changes to COMPLETE)
4. Verify task appears correctly in `GET /api/tasks`

**Status**: ✅ PASSING

### POC-REG-002: Parent Task Context Preservation
Tests parent-child task relationships:
1. Create and complete a parent task
2. Create a child task with `parent_task_id` reference
3. Claim child task and verify it contains parent context
4. Ensure parent output is included in the context block

**Status**: ✅ PASSING

### POC-REG-003: SSE Events Stream Functionality
Tests Server-Sent Events (SSE) real-time updates:
1. Connect to SSE stream at `GET /events`
2. Verify proper SSE headers and initial event
3. Create a task and verify SSE update is received
4. Ensure events contain task data

**Status**: ✅ PASSING

### POC-REG-004: Workdir Management
Tests working directory set/get functionality:
1. Set workdir via `POST /set-workdir`
2. Get workdir via `GET /api/workdir`
3. Verify workdir matches what was set
4. Test error handling for invalid input

**Status**: PASSING

### V1-REG-005: updated_at Response Parity
Tests updated_at propagation across v1 endpoints:
1. Create a task via `POST /create` and verify JSON response includes non-empty `updated_at`
2. Claim the task via `GET /task` and verify text response includes `UPDATED_AT:` metadata
3. Save the task via `POST /save` and verify JSON response includes non-empty `updated_at`
4. Update status via `POST /update-status` and verify JSON response includes non-empty `updated_at`
5. List tasks via `GET /api/tasks` and verify each task includes `updated_at`

**Status**: PASSING

### V1-REG-006: Events Stream Replay and Limits
Tests v1 SSE replay behavior for `GET /events`:
1. Create multiple tasks so events exist
2. Connect with `since` (RFC3339) and verify replayed events are streamed before live events
3. Add `limit` with `since` and verify replay count is capped
4. Verify `limit` without `since` returns validation error
5. Verify replay size is capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`

**Status**: PASSING

### V2-REG-007: Tasks CRUD + updated_at
Tests v2 task lifecycle and updated_at propagation:
1. Create a task via `POST /api/v2/tasks` and verify `updated_at` is present
2. Claim via `POST /api/v2/tasks/:id/claim` and verify `updated_at` changes
3. Update via `PATCH /api/v2/tasks/:id` and verify `updated_at` changes
4. Complete via `POST /api/v2/tasks/:id/complete` and verify `updated_at` changes
5. Delete via `DELETE /api/v2/tasks/:id` and verify subsequent GET returns NOT_FOUND
6. List via `GET /api/v2/tasks` and verify `updated_at` for each task

**Status**: PASSING

### V2-REG-008: Events List + Stream Replay
Tests v2 events list and SSE stream behavior:
1. List via `GET /api/v2/events` and validate pagination + filters (`project_id`, `task_id`, `type`, `since`)
2. Connect to `GET /api/v2/events/stream` with `project_id` and verify SSE headers
3. Connect with `since` (RFC3339 or event id) and verify replay occurs before live events
4. Verify `limit` caps replay size and is bounded by `COCO_SSE_REPLAY_LIMIT_MAX`
5. Validate error handling for invalid `since`, `limit`, and missing `project_id`

**Status**: PASSING

### V2-REG-009: Agents List/Detail/Delete
Tests v2 agent inventory behavior:
1. Register agent via `POST /api/v2/agents` and capture `agent_id`
2. List via `GET /api/v2/agents` with paging/sorting filters
3. Fetch via `GET /api/v2/agents/:id` and validate fields
4. Delete via `DELETE /api/v2/agents/:id` and verify NOT_FOUND on fetch
5. Verify method-not-allowed errors on unsupported verbs

**Status**: PASSING

### V2-REG-010: Leases Create/Heartbeat/Release
Tests v2 leases behavior:
1. Create lease via `POST /api/v2/leases` and capture `lease_id`
2. Heartbeat via `POST /api/v2/leases/:id/heartbeat` and verify renewal fields
3. Release via `POST /api/v2/leases/:id/release` and verify lease is closed
4. Verify invalid lease ids return NOT_FOUND
5. Verify method-not-allowed errors on unsupported verbs

**Status**: PASSING

### V2-REG-011: Config + Version Endpoints
Tests v2 config/version health contract:
1. `GET /api/v2/config` returns redacted runtime config snapshot (auth, retention, sse)
2. `GET /api/v2/version` returns service version and schema version
3. Validate retention snapshot fields (`enabled`, `interval_seconds`, `max_rows`, `days`)
4. Verify method-not-allowed errors on unsupported verbs

**Status**: PASSING

### V2-REG-012: Runs Sub-Resources
Tests v2 runs sub-resources for tasks:
1. Create a task via `POST /api/v2/tasks`
2. Create a run via `POST /api/v2/tasks/:id/runs` and verify required fields in response
3. List via `GET /api/v2/tasks/:id/runs` and verify pagination and ordering
4. Fetch via `GET /api/v2/tasks/:id/runs/:run_id` and validate run details
5. Verify NOT_FOUND for unknown task/run ids and method-not-allowed errors

**Status**: PASSING

### V2-REG-013: Memory Endpoints
Tests v2 memory get/put contract:
1. Put memory via `PUT /api/v2/memory` with `project_id`, `task_id`, `key`, and `value`
2. Get memory via `GET /api/v2/memory` with matching selectors and verify stored value
3. Verify idempotent overwrite for the same `project_id`, `task_id`, and `key`
4. Validate error handling for missing selectors, invalid payloads, and oversized values
5. Verify method-not-allowed errors on unsupported verbs

**Status**: PASSING

### V2-REG-014: Context Packs Endpoints
Tests v2 context packs list/detail behavior:
1. List via `GET /api/v2/context-packs` with filters (`project_id`, `task_id`, `type`)
2. Fetch via `GET /api/v2/context-packs/:id` and validate payload structure
3. Verify pagination for list and consistent ordering by `created_at`
4. Validate NOT_FOUND for unknown ids and method-not-allowed errors

**Status**: PASSING

### V2-REG-015: Completion next_tasks Child Creation + Events
Tests v2 completion behavior when `next_tasks` is provided:
1. Create a parent task via `POST /api/v2/tasks` with a known `project_id`
2. Complete via `POST /api/v2/tasks/:id/complete` and include `next_tasks` with at least two child task payloads
3. Validate `result` requires `summary`, `changes_made`, `files_touched`, `commands_run`, `tests_run`, `risks`, and `next_tasks` (each child requires `title`, `instructions`, `type`, `priority`)
4. Verify response includes created child task ids and correct `parent_task_id` links
5. Fetch each child via `GET /api/v2/tasks/:child_id` and validate `project_id` inheritance and initial status
6. List events via `GET /api/v2/events` filtered by `task_id` and `project_id` and verify completion + child creation events
7. Stream events via `GET /api/v2/events/stream` with `since` (event id or RFC3339) and verify replay includes the same events

**Status**: PASSING

### V2-REG-016: Project-Scoped Task Create (Success + Validation + Not Found)
Tests v2 project-scoped task creation behavior:
1. Create a project via `POST /api/v2/projects` and capture `project_id`
2. Create a task via `POST /api/v2/projects/:project_id/tasks` with required fields and verify `project_id` is set
3. Validate the created task via `GET /api/v2/tasks/:id` and ensure it matches the project
4. Submit an invalid payload (missing required fields) and verify validation error
5. Submit to an unknown `project_id` and verify NOT_FOUND

**Status**: PASSING

### V2-REG-017: Project-Scoped Events Stream + Replay
Tests project-scoped event streaming and replay endpoints:
1. Create a project via `POST /api/v2/projects` and capture `project_id`
2. Create events for this project and a different project
3. Connect to `GET /api/v2/projects/:project_id/events/stream` and verify SSE headers + event payloads only include the target project
4. Verify `type` filter (for example `task.created`) is respected when streaming
5. Replay via `GET /api/v2/projects/:project_id/events/replay?since_id=<event_id>` and verify events are returned in order
6. Validate errors for missing `since_id`, invalid `since_id`, and unknown `project_id`

**Status**: PASSING

### V2-REG-018: Project Tree + Changes Endpoints
Tests project tree snapshot and changes feed behavior:
1. Create a project with a temporary workdir and add a file + subdirectory
2. Fetch `GET /api/v2/projects/:project_id/tree` and verify root node `.` with expected child entries and metadata
3. Verify errors for missing project and missing workdir
4. Fetch `GET /api/v2/projects/:project_id/changes` and verify response contains git status-based change entries
5. Validate `since` (RFC3339) handling and verify invalid `since` returns INVALID_ARGUMENT
6. Verify method-not-allowed errors for non-GET requests on both endpoints

**Status**: PASSING

### V2-REG-019: Automation Rules Edge Cases
Tests automation rule followups for edge cases:
1. Configure a non-matching trigger and verify no followup tasks are created
2. Use templates with unknown fields and verify known tokens render while unknown tokens remain literal
3. Configure multiple followup actions and verify all followup tasks are created

**Status**: PASSING

## Running the Tests

### Prerequisites
- Go toolchain (1.16 or later)
- SQLite support (using modernc.org/sqlite)

### Run All Tests
```bash
cd "C:\Users\weli\Downloads\Work\cocopilot"
go test -v
```

### Run Specific Test
```bash
# Run a specific test case
go test -v -run TestPOCREG001_CreateClaimSaveLifecycle

# Run all regression tests as a suite
go test -v -run TestAllPOCRegressionSuite
```

### Run Tests with Short Output
```bash
go test
```

## Test Environment

### Isolated Test Database
Tests automatically create isolated test databases in `./tmp/test_<timestamp>.db` and clean them up after completion. This ensures:
- No interference with production `tasks.db`
- Each test run starts with a clean state
- Parallel test execution is safe

### Test Server
Tests use `httptest.NewRecorder()` and `httptest.NewServer()` for in-memory testing without requiring a running server instance.

## Continuous Integration

These tests should be integrated into CI/CD pipelines as a gate for all PRs:

```yaml
# Example GitHub Actions workflow
- name: Run POC Regression Tests
  run: go test -v
```

**Expected behavior**: All tests must pass before merging any PR.

## Test Results

Latest run results:
```
PASS: TestPOCREG001_CreateClaimSaveLifecycle (0.02s)
PASS: TestPOCREG002_ParentTaskContext (0.02s)
PASS: TestPOCREG003_SSEEventsStream (0.01s)
PASS: TestPOCREG004_WorkdirManagement (0.01s)
PASS: TestAllPOCRegressionSuite (0.06s)

Total: 0.228s
```

### 2026-02-13
- Command: `go test ./...`
- Outcome: FAIL
- Failure summary:
  - FAIL theinf-loop (26.700s)

### 2026-02-13 (follow-up)
- Command: `go test ./... -v`
- Outcome: PASS
- Notes: All packages passed.

### 2026-02-13 (task 748)
- Change: Clear `COCO_AUTOMATION_RULES` in runtime config tests to avoid ambient env contamination.
- Command: `go test -v ./...`
- Outcome: PASS
- Notes: All packages passed.

### 2026-02-12
- Command: `go test ./...`
- Outcome: FAIL
- Failure summary:
  - db_v2_test.go:600:2: expected declaration, found policies
  - FAIL theinf-loop [setup failed]

### 2026-02-12 (follow-up)
- Change: Moved policy persistence assertions into `TestCreatePolicyDefaults` to fix parse error in db_v2_test.go.
- Command: `go test ./...`
- Outcome: FAIL
- Failure summary:
  - v2_task_create_test.go:12:2: declared and not used: testDB
  - FAIL theinf-loop [build failed]

### 2026-02-12 (follow-up 2)
- Change: Removed unused `testDB` binding in `TestV2TaskCreateSuccess`.
- Command: `go test ./...`
- Outcome: PASS
- Notes: All packages passed.

### 2026-02-12 (follow-up 3)
- Command: `go test ./...`
- Outcome: FAIL
- Failure summary:
  - FAIL theinf-loop (27.095s)

### 2026-02-12 (follow-up 4)
- Command: `go test ./...`
- Outcome: FAIL
- Failure summary:
  - FAIL theinf-loop (26.700s)

### 2026-02-12 (release checks)
- Command: `npm run release:vsix`
- Outcome: FAIL
- Failure summary: `npm` not recognized (npm not installed or not on PATH).
- Command: `npm run release:check`
- Outcome: FAIL
- Failure summary: `npm` not recognized (npm not installed or not on PATH).

## Troubleshooting

### Test Failures
If a test fails:
1. Check the verbose output (`-v` flag) for detailed error messages
2. Verify no other process is using the test database
3. Ensure all dependencies are installed (`go mod tidy`)
4. Check that the database schema in `main.go` matches test expectations

### Database Issues
If you see database-related errors:
```bash
# Clean up test databases
rm -rf ./tmp/test_*.db
```

### SSE Connection Errors
The SSE test may show a "use of closed network connection" error at the end - this is expected behavior when the test completes and closes the connection.

## Code Coverage

To run tests with coverage:
```bash
go test -v -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Next Steps

After successful implementation of the migration system (NEXT-002), verify that:
1. All regression tests still pass with the new migration system
2. Existing `tasks.db` files upgrade smoothly
3. No regression in task lifecycle or context preservation
