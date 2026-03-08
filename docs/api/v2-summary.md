# API v2 Contract Design - Executive Summary

**Date:** February 12, 2026  
**Task ID:** 196  
**Status:** Implementation Underway (snapshot 2026-02-12)

## Overview

This document provides a quick reference to the complete API v2 contract design for Cocopilot. The v2 API introduces multi-project support, execution ledger tracking, agent coordination via leases, structured completion metadata, and real-time events while maintaining full backward compatibility with v1.

## Implementation Snapshot (2026-02-12)

- Plan completion estimate: ~60% (backend v2 largely done; UI expansion, governance tooling, and packaging remain incomplete).
- Migrations `0001`-`0018` applied from `migrations/` (including events filter indexes, task sort indexes, `tasks.updated_at`, and policies storage).
- v2 endpoints implemented: projects create/list/detail/update/delete, tasks create/list/detail/update/delete, task claim/complete, project task list/create, task dependencies, runs detail/steps/logs/artifacts, memory put/get, context packs, events list/stream, leases create/heartbeat/release, agents list/detail/delete, policies create/list/detail/update/delete, health, config, version.
- v2 task list supports filters (`project_id`, `status`, `type`, `tag`, `q`), `limit`/`offset` pagination with `total`, and `created_at`/`updated_at` sorting.
- v2 project task list supports filters (`status`, `type`, `tag`, `q`), `limit`/`offset` pagination with `total`, and `created_at`/`updated_at` sorting.
- v2 events list supports `type`, `since`, `task_id`, `project_id`, `limit`, and `offset` filters with `total`.
- v2 events stream supports `project_id` scoping, optional `type` filter, `since` replay with `limit`, and configurable heartbeat and replay caps.
- v2 project audit list supports `type`, `task_id`, `since`, `limit`, and `offset` filters with `total`.
- v2 policies list supports `enabled` filter, `limit`/`offset` pagination with `total`, and sorting by `created_at` or `name`.
- Events retention cleanup is configurable and logs prune outcomes when enabled.
- v2 project tree returns a shallow workdir snapshot; v2 project changes supports `since` (RFC3339) and returns git status-based changes.
- Standard v2 error envelope is enforced across current v2 handlers; OpenAPI parity pass done for shipped endpoints.
- Auth foundation in place (API keys with optional read protection, scoped identities, and audit logging).
- `GET /api/v2/version` includes retention config snapshot; `GET /api/v2/config` returns redacted runtime config.
- v1 endpoints now return `updated_at` and support expanded filtering/sorting on `GET /api/tasks` and `GET /events`.
- task.completed rules are configurable; follow-up task creation from `result.next_tasks` honors enablement, limits, and allowlists.
- Automation engine rules are configured via `COCO_AUTOMATION_RULES`; when unset or empty, no automation tasks are emitted.
- MCP server and VSIX scaffolds are documented with updated command/tool coverage; packaging and release automation remain pending.

## Key Documents

| Document | Description | Location |
|----------|-------------|----------|
| **OpenAPI Specification** | Complete API contract with all endpoints, schemas, and examples | [openapi-v2.yaml](openapi-v2.yaml) |
| **Design Document** | Comprehensive design with architecture, domain model, and rationale | [v2-design.md](v2-design.md) |
| **Compatibility Plan** | Strategy for maintaining v1/v2 compatibility and migration path | [v2-compatibility.md](v2-compatibility.md) |
| **Database Schema** | SQL migrations for all v2 tables and indexes | [../schema/v2-migrations.sql](../schema/v2-migrations.sql) |
| **Implementation Roadmap** | Phased delivery plan with timeline and tasks | [v2-roadmap.md](v2-roadmap.md) |

## What's New in v2

### 1. Multi-Project Support
- Multiple isolated workspaces within one instance
- Project-scoped tasks, events, and memory
- v1 compatibility via default `proj_default` project

**Key Endpoints:**
- `GET /api/v2/projects/{projectId}/tasks` - List tasks for a project (implemented)
- `POST /api/v2/projects/{projectId}/tasks` - Create task in a project (implemented)
- `POST /api/v2/projects` - Create project (implemented)
- `GET /api/v2/projects` - List projects (implemented)
- `GET /api/v2/projects/{projectId}` - Get project details (implemented)
- `PATCH /api/v2/projects/{projectId}` - Update project (implemented)
- `DELETE /api/v2/projects/{projectId}` - Delete project (implemented)

### 2. Enhanced Tasks
- Task types: ANALYZE, MODIFY, TEST, REVIEW, DOC, RELEASE, ROLLBACK
- Priority ordering
- Tag-based categorization
- Dual status tracking (v1 and v2)
- Full-text search
- Pagination and sorting with `total` counts

**Key Endpoints:**
- `POST /api/v2/tasks` - Create task (implemented)
- `GET /api/v2/tasks` - List/filter tasks (implemented)
- `GET /api/v2/projects/{projectId}/tasks` - List/filter tasks for a project (implemented)
- `POST /api/v2/projects/{projectId}/tasks` - Create task scoped to a project (implemented)
- `GET /api/v2/tasks/{taskId}` - Get task details (implemented)
- `PATCH /api/v2/tasks/{taskId}` - Update task fields (implemented)
- `DELETE /api/v2/tasks/{taskId}` - Delete task (implemented)
- `POST /api/v2/tasks/{taskId}/dependencies` - Add dependency (implemented)
- `GET /api/v2/tasks/{taskId}/dependencies` - List dependencies (implemented)
- `DELETE /api/v2/tasks/{taskId}/dependencies/{dependsOnTaskId}` - Remove dependency (implemented)

### 3. Execution Ledger
- Track every execution attempt (runs)
- Log steps, stdout/stderr, and artifacts
- Attach diffs, patches, test results
- Audit trail for debugging and analysis
- `tool_invocations` are available only in run detail responses

**Status:** Migration is applied; run endpoints are implemented.

**Key Endpoints:**
- `GET /api/v2/runs/{runId}` - Get run details (implemented)
- `POST /api/v2/runs/{runId}/steps` - Log execution step (implemented)
- `POST /api/v2/runs/{runId}/logs` - Stream logs (implemented)
- `POST /api/v2/runs/{runId}/artifacts` - Attach artifacts (implemented)

### 4. Lease-Based Claiming
- Exclusive task claiming for agent coordination
- Time-bound leases with expiration
- Heartbeat mechanism to keep leases alive
- Prevents multiple agents from conflicting

**Key Endpoints:**
- `POST /api/v2/tasks/{taskId}/claim` - Claim task (implemented)
- `POST /api/v2/leases` - Create lease (implemented)
- `POST /api/v2/leases/{leaseId}/heartbeat` - Renew lease (implemented)
- `POST /api/v2/leases/{leaseId}/release` - Release lease (implemented)

### 5. Structured Completion
- Rich metadata: summary, changes, files, commands, tests, risks
- Auto-suggest next tasks
- v1 output auto-generated from v2 summary
- Follow-up creation from `result.next_tasks` is governed by configurable task.completed rules (enable/disable, limits, allowlists, dependency behavior)

#### Automation Rules Configuration
- **Environment**: `COCO_AUTOMATION_RULES` is a JSON array of rules. Empty or unset disables automation.
- **Endpoint**: `GET /api/v2/projects/{projectId}/automation/rules` returns the current server rules (applies to all projects; empty array when disabled).
- **Simulation**: `POST /api/v2/projects/{projectId}/automation/simulate` previews actions and tasks that would be created for a supported event.
- **Replay**: `POST /api/v2/projects/{projectId}/automation/replay?since_event_id=...` re-runs automation for `task.completed` events since the anchor (inclusive). Optional `limit` caps the replay window.
- **Rule fields**: `name`, `enabled` (default: true), `trigger` (only `task.completed`), `actions` (non-empty).
- **Action fields**: `type` must be `create_task`; `task.instructions` is required; optional `task.title`, `task.type`, `task.priority` (non-negative), `task.tags`, `task.parent`.
- **Parent behavior**: `task.parent` defaults to `completed` (child of the completed task). Use `none` to create a root task.
- **Templates**: `task.title` and `task.instructions` support `${event_id}`, `${event_kind}`, `${project_id}`, `${task_id}`, `${task_instructions}`, `${task_output}`, `${task_status_v1}`, `${task_status_v2}`, `${task_title}`.
- **Defaults**: Rules only run for `task.completed` events, create tasks in the same project, and emit a `task.created` event. Invalid rules fail startup. Automation rules are not exposed in `GET /api/v2/config`.

**Simulation example:**
```json
{
  "event": {
    "kind": "task.completed",
    "entity_id": "123",
    "payload": {
      "task_id": 123
    }
  }
}
```

**Simulation response:**
```json
{
  "actions": [
    {
      "rule_name": "Followup",
      "type": "create_task",
      "task": {
        "project_id": "proj_default",
        "parent_task_id": 123,
        "title": "Review Parent task",
        "instructions": "Follow up 123",
        "type": "REVIEW",
        "priority": 5,
        "tags": ["auto", "review"]
      }
    }
  ],
  "tasks_that_would_be_created": [
    {
      "project_id": "proj_default",
      "parent_task_id": 123,
      "title": "Review Parent task",
      "instructions": "Follow up 123",
      "type": "REVIEW",
      "priority": 5,
      "tags": ["auto", "review"]
    }
  ]
}
```

#### Policy Rule Catalog (Current)
- Rules are stored in project policies (`policies.rules`) and validated on create/update.
- `type` is required and normalized to lowercase; `reason` is optional string.
- Enabled policies only; first matching rule blocks the action and returns `FORBIDDEN`.

**Rule types and effects:**
- `automation.block`: blocks automation follow-up creation from `COCO_AUTOMATION_RULES`.
- `completion.block`: blocks `POST /api/v2/tasks/{taskId}/complete`.
- `task.create.block`: blocks `POST /api/v2/tasks` and `POST /api/v2/projects/{projectId}/tasks`.
- `task.update.block`: blocks `PATCH /api/v2/tasks/{taskId}`.
- `task.delete.block`: blocks `DELETE /api/v2/tasks/{taskId}`.

**Example policy rule:**
```json
{
  "type": "task.create.block",
  "reason": "Task creation requires approval"
}
```

**Key Endpoint:**
- `POST /api/v2/tasks/{taskId}/complete` - Complete with metadata (implemented)

### 6. Real-Time Events (SSE)
- Event types: task, run, lease, memory, repo changes
- Project scoping via `project_id` filter
- Stream supports `since` replay with `limit`
- Heartbeat interval and replay limit caps are configurable
- Project-specific stream/replay use `project_id` on the shared endpoints

**Key Endpoints:**
- `GET /api/v2/events` - List events (implemented)
- `GET /api/v2/events/stream` - Subscribe to events (SSE, implemented)

### 7. Persistent Memory
- Store project-level knowledge
- Scoped: GLOBAL, MODULE, FILE, TASK, RUN
- Queryable by scope, key, or full-text search
- Source references track knowledge provenance

**Key Endpoints:**
- `PUT /api/v2/projects/{projectId}/memory` - Store memory (implemented)
- `GET /api/v2/projects/{projectId}/memory` - Query memory (implemented)

### 8. Context Packs
- Automated context generation for tasks
- Budget-controlled (max files, bytes, snippets)
- Includes file snippets, related tasks, decisions, repo state

**Key Endpoints:**
- `POST /api/v2/projects/{projectId}/context-packs` - Generate context (implemented)
- `GET /api/v2/context-packs/{packId}` - Get context pack detail (implemented)

**Example response:**
```json
{
  "context_pack": {
    "id": "pack_384",
    "task_id": 384,
    "project_id": "proj_default",
    "summary": "Relevant migration and API docs for memory and context packs.",
    "contents": {
      "files": [
        {
          "path": "migrations/0010_context_packs.sql",
          "snippets": [
            {
              "start": 1,
              "end": 120,
              "text": "CREATE TABLE context_packs (...);"
            }
          ]
        }
      ],
      "related_tasks": [381, 382],
      "decisions": ["Store request examples under endpoints"],
      "repo_state": {
        "git_head": "abc1234",
        "dirty": true
      }
    }
  }
}
```

### 9. Project Tree and Changes
- Shallow workdir snapshot for quick repo awareness
- Change summary endpoint with optional `since` filter
- Change summary uses git status to report working tree modifications

**Key Endpoints:**
- `GET /api/v2/projects/{projectId}/tree` - Project tree snapshot (implemented)
- `GET /api/v2/projects/{projectId}/changes` - File changes since `since` (implemented)

### 10. Agents
- Track agent activity and last_seen timestamps
- Support cleanup and lifecycle management

**Key Endpoints:**
- `GET /api/v2/agents` - List agents with filters (implemented)
- `GET /api/v2/agents/{agentId}` - Agent detail (implemented)
- `DELETE /api/v2/agents/{agentId}` - Delete agent (implemented)

### 11. Policies
- Project-scoped policy collections with validated rule catalogs
- List supports `enabled` filter, `limit`/`offset` pagination with `total`, and sorting by `created_at` or `name`

**Key Endpoints:**
- `GET /api/v2/projects/{projectId}/policies` - List policies (implemented)
- `POST /api/v2/projects/{projectId}/policies` - Create policy (implemented)
- `GET /api/v2/projects/{projectId}/policies/{policyId}` - Policy detail (implemented)
- `PATCH /api/v2/projects/{projectId}/policies/{policyId}` - Update policy (implemented)
- `DELETE /api/v2/projects/{projectId}/policies/{policyId}` - Delete policy (implemented)

### 12. Config and Version
- Expose runtime configuration snapshot (redacted)
- Report API and retention capability details

**Key Endpoints:**
- `GET /api/v2/config` - Runtime config snapshot (implemented)
- `GET /api/v2/version` - Version and retention snapshot (implemented)

### 13. Project Audit
- Project-scoped audit event list derived from the events ledger
- Filterable by event kind and task ID, with RFC3339 time windowing
- Pagination uses `limit`/`offset` and returns a `total` count

**Key Endpoint:**
- `GET /api/v2/projects/{projectId}/audit` - List audit events for a project (implemented)

**Filters:** `type`, `task_id`, `since`, `limit`, `offset`

**Example response:**
```json
{
  "events": [
    {
      "id": "evt_a1",
      "project_id": "proj_123",
      "kind": "task.created",
      "entity_type": "task",
      "entity_id": "501",
      "created_at": "2026-02-11T10:02:00Z",
      "payload": {
        "title": "Add docs example"
      }
    }
  ],
  "total": 1
}
```

## Database Schema Additions

| Migration | Tables Added | Purpose |
|-----------|--------------|---------|
| 0005 | (columns) | Add v2 fields to tasks: title, type, priority, status_v2, tags_json |
| 0006 | runs, run_steps, run_logs, artifacts, tool_invocations | Execution ledger |
| 0007 | leases | Task claiming and agent coordination |
| 0008 | events | Real-time event storage and replay |
| 0009 | memory | Persistent knowledge base |
| 0010 | context_packs | Automated context generation |
| 0011 | (indexes) | Task/project foreign key and project defaults |
| 0013 | task_dependencies | Task dependency graph |
| 0014 | (backfill) | Events `project_id` backfill |
| 0015 | (indexes) | Event filtering indexes |
| 0016 | (indexes) | Task sort indexes |
| 0017 | (columns/indexes) | `tasks.updated_at` + backfill |

**Total New Tables:** 9  
**Total New Columns:** 6 (in tasks table)

## Backward Compatibility Summary

### v1 Endpoints (Unchanged)
- `GET /task` - Returns next task, marks IN_PROGRESS, includes `updated_at`
- `POST /create` - Creates task with instructions, includes `updated_at`
- `POST /save` - Saves task output, includes `updated_at`
- `POST /update-status` - Updates task status, includes `updated_at`
- `POST /delete` - Deletes task
- `GET /api/tasks` - Lists all tasks with filters, sorting, and `total` count
- `GET /events` - SSE for tasks with `project_id`, `type=tasks`, `since`, and replay `limit`
- `GET /api/workdir` - Gets default project workdir
- `POST /set-workdir` - Updates default project workdir
- `GET /instructions` - Agent setup instructions
- `GET /` - Kanban UI

### Compatibility Strategy
- All v1 tasks belong to `proj_default`
- Status sync: v1 `status` ↔ v2 `status_v2`
- v1 `output` populated from v2 completion summary
- v1 clients can ignore v2 fields in JSON responses

### Data Coexistence
- v1 and v2 share same `tasks` table
- v2 columns are nullable with defaults
- v1 queries work unchanged (ignore new columns)
- v2 queries read both old and new columns

## Implementation Timeline

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| 1. Foundation | Week 1 | Projects CRUD, Tasks v2 endpoints |
| 2. Execution Ledger | Week 2 | Runs, steps, logs, artifacts |
| 3. Leases & Claiming | Week 2-3 | Task claiming with leases |
| 4. Structured Completion | Week 3 | Rich completion metadata |
| 5. Events & Memory | Week 4 | SSE + Memory persistence |
| 6. Context Packs | Week 4 | Automated context generation |
| 7. Polish & Docs | Week 5 | Production readiness |

**Total Estimated Effort:** 270 hours (~5 weeks with 2 developers)

## Quick Start for Developers

### 1. Review the Design
```bash
# Read the comprehensive design document
cat docs/api/v2-design.md
```

### 2. Understand the Schema
```bash
# Review database migrations
cat docs/schema/v2-migrations.sql
```

### 3. Check Compatibility Plan
```bash
# Understand v1/v2 coexistence
cat docs/api/v2-compatibility.md
```

### 4. Follow the Roadmap
```bash
# See phased implementation plan
cat docs/api/v2-roadmap.md
```

### 5. Explore the API
```bash
# Open OpenAPI spec in Swagger UI
# Visit: https://editor.swagger.io/
# Upload: docs/api/openapi-v2.yaml
```

## Example Workflows

### Workflow 1: Create Project and Task (v2)
```bash
# Create project
curl -X POST http://127.0.0.1:8080/api/v2/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Project",
    "workdir": "/home/user/my-project"
  }'

# Create task
curl -X POST http://127.0.0.1:8080/api/v2/projects/proj_abc123/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "instructions": "Implement login feature",
    "type": "MODIFY",
    "priority": 5,
    "tags": ["backend", "auth"]
  }'
```

### Workflow 2: Claim, Execute, Complete (v2)
```bash
# Claim task
curl -X POST http://127.0.0.1:8080/api/v2/tasks/123/claim \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "agent_xyz"}'

# Log step
curl -X POST http://127.0.0.1:8080/api/v2/runs/run_abc123/steps \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Analyze codebase",
    "status": "SUCCEEDED"
  }'

# Complete task
curl -X POST http://127.0.0.1:8080/api/v2/tasks/123/complete \
  -H "Content-Type: application/json" \
  -d '{
    "run_id": "run_abc123",
    "status": "SUCCEEDED",
    "result": {
      "summary": "Implemented login with JWT",
      "changes_made": ["Added POST /api/login"],
      "files_touched": ["src/auth/login.go"],
      "commands_run": ["go test ./..."],
      "tests_run": ["go test ./..."],
      "risks": ["None"],
      "next_tasks": [
        {
          "title": "Document auth flow",
          "instructions": "Update the auth API docs with JWT details",
          "type": "DOC",
          "priority": 7,
          "tags": ["docs", "auth"]
        }
      ]
    }
  }'
```

### Workflow 2a: Run Detail with Steps, Logs, Artifacts (v2)
```bash
# Log step
curl -X POST http://127.0.0.1:8080/api/v2/runs/run_abc123/steps \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Run unit tests",
    "status": "SUCCEEDED"
  }'

# Stream log line
curl -X POST http://127.0.0.1:8080/api/v2/runs/run_abc123/logs \
  -H "Content-Type: application/json" \
  -d '{
    "level": "INFO",
    "message": "go test ./..."
  }'

# Attach artifact
curl -X POST http://127.0.0.1:8080/api/v2/runs/run_abc123/artifacts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "unit-test-report",
    "type": "text/plain",
    "content": "ok\n"
  }'

# Fetch run detail (includes steps, logs, artifacts)
curl http://127.0.0.1:8080/api/v2/runs/run_abc123
```

### Workflow 3: Memory, Context Pack, and Run Logs (v2)
```bash
# Store memory
curl -X PUT http://127.0.0.1:8080/api/v2/projects/proj_abc123/memory \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "MODULE",
    "key": "auth.jwt",
    "value": "JWT auth uses HS256 with 1h expiry",
    "source": "docs/auth.md"
  }'

# Generate a context pack
curl -X POST http://127.0.0.1:8080/api/v2/projects/proj_abc123/context-packs \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": 123,
    "max_files": 8,
    "max_bytes": 60000
  }'

# Stream run logs
curl -X POST http://127.0.0.1:8080/api/v2/runs/run_abc123/logs \
  -H "Content-Type: application/json" \
  -d '{
    "level": "INFO",
    "message": "Tests started"
  }'
```

### Workflow 3a: Project Tree and Changes (v2)
```bash
# Fetch shallow project tree
curl http://127.0.0.1:8080/api/v2/projects/proj_abc123/tree

# Fetch project changes (optionally since a timestamp)
curl "http://127.0.0.1:8080/api/v2/projects/proj_abc123/changes?since=2026-02-10T12:00:00Z"
```

### Workflow 4: v1 Client (Unchanged)
```bash
# Get next task (v1)
curl http://127.0.0.1:8080/task

# Create task (v1)
curl -X POST http://127.0.0.1:8080/create \
  -H "Content-Type: application/json" \
  -d '{"instructions": "Fix bug in login"}'

# Response includes updated_at
# {"success": true, "task_id": 123, "updated_at": "2026-02-11T12:34:56.000000Z"}

# Save task (v1)
curl -X POST http://127.0.0.1:8080/save \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "task_id=123&message=Bug fixed"

# Response includes updated_at
# Task 123 saved and marked as COMPLETE.
# UPDATED_AT: 2026-02-11T12:34:56.000000Z
```

## API Design Highlights

### Consistent Error Format
All v2 errors use:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Task not found",
    "details": {"task_id": 123}
  }
}
```

### Standard HTTP Status Codes
- `200` - Success
- `201` - Created
- `204` - No Content (delete, release)
- `400` - Invalid Argument
- `404` - Not Found
- `409` - Conflict (lease already exists)
- `500` - Internal Error

### ID Formats
- Projects: `proj_<uuid>` (e.g., `proj_abc123...`)
- Tasks: `<integer>` (v1 compatible)
- Runs: `run_<uuid>` (e.g., `run_xyz789...`)
- Leases: `lease_<uuid>` (e.g., `lease_mno456...`)
- Events: `event_<ulid>` (e.g., `event_01ARZ3NDEKTSV4RRFFQ69G5FAV`)

### Timestamps
- Format: ISO8601 (e.g., `2026-02-11T10:30:00.000Z`)
- Always UTC
- Generated server-side

## Testing Strategy

### Unit Tests
- Endpoint handlers with mock DB
- Status mapping logic
- Lease expiration logic
- Context pack generation

### Integration Tests
- Full workflows: create → claim → execute → complete
- v1/v2 interoperability
- Lease conflicts and expiration
- SSE event delivery

### Performance Tests
- 1K tasks, 100 concurrent claims
- 10K events, 100 SSE clients
- 1K memory items, complex queries

### Compatibility Tests
- v1 endpoints work unchanged
- v1 tasks appear in v2 queries
- v2 completions populate v1 output

## Security Considerations

### Current (POC)
- No authentication (local development only)
- Input validation on all endpoints
- SQL injection prevention via parameterized queries
- File path validation within workdir boundaries

### Future (Production)
- JWT-based authentication
- Role-based access control (RBAC)
- Project-level permissions
- Agent identity verification
- Rate limiting
- HTTPS required

### Auth Scopes (v2)
- `policy.read` - Read policy collections and policy details.
- `policy.write` - Create, update, or delete policies and policy rules.
- `audit.read` - Read project audit events (list endpoint).

## Performance Characteristics

### Expected Capabilities
- SQLite: 100K tasks, 1M events
- SSE: ~1K concurrent connections per project
- Task listing: <100ms for filtered queries
- Event replay: <200ms for 1K events
- Context pack: <2s for typical project

### Scalability Path
For higher scale:
- PostgreSQL migration
- Redis for event broker
- S3 for artifact storage
- Read replicas for queries

## Success Metrics

### Technical
- [ ] All v2 endpoints operational
- [ ] v1 endpoints passing all tests
- [ ] Test coverage ≥ 80%
- [ ] OpenAPI spec validated
- [ ] Performance targets met

### Business
- [ ] Multi-project support enables new use cases
- [ ] Execution tracking improves debugging
- [ ] Structured completion enables automation
- [ ] Real-time events improve UX

## Next Steps

1. **Approve Design** - Review and sign off on design documents
2. **Set Up Environment** - Prepare development environment with test database
3. **Begin Phase 1** - Implement foundation (projects, tasks v2)
4. **Continuous Testing** - Test v1 compatibility throughout development
5. **Iterate** - Deliver value incrementally, phase by phase

## Contact and Resources

**Related Documents:**
- Current v1 Implementation: [server/main.go](../../server/main.go)
- Existing Migrations: [migrations/](../../migrations/)

**Tools:**
- **OpenAPI Editor:** https://editor.swagger.io/
- **SQLite Browser:** https://sqlitebrowser.org/
- **SSE Testing:** curl with `-N --no-buffer` flags

---

**Design Status:** ✅ Complete  
**Implementation Status:** In Progress (snapshot 2026-02-11)  
**Target Launch:** March 14, 2026 (5 weeks)

