# API v2 Backward Compatibility Plan

## Executive Summary

This document outlines the strategy for maintaining full backward compatibility between Cocopilot API v1 and v2. The goal is to ensure existing v1 clients continue working without any changes while enabling new v2 clients to leverage enhanced features.

**Status:** Implementation Snapshot (2026-02-12)  
**Date:** February 12, 2026  
**Priority:** HIGH - Critical for adoption

## Compatibility Principles

### Core Guarantees

1. **v1 Endpoints Immutable**: All v1 endpoints remain unchanged in behavior, response format, and URL paths
2. **v1 Data Readable**: v2 additions to schema do not break v1 queries
3. **Cross-Version Status Visibility**: v2 writes both status fields; v1 writes status only; `status_v2` defaults to `QUEUED` on inserts; v2 reads map missing `status_v2` from `status`
4. **Default Project**: v1 operates implicitly on `proj_default` project
5. **Status Mapping**: deterministic mapping rules exist; v2 writes both fields while v1 writes `status` only

**Implementation Note:** v1 updates do not modify `status_v2` when it is already set (including the schema default), so v2 reads will prefer the existing `status_v2` value even if it no longer matches `status` until a v2 write updates it.

## Endpoint Compatibility Matrix

| v1 Endpoint             | Status      | v2 Equivalent                                    | Notes                          |
|-------------------------|-------------|--------------------------------------------------|--------------------------------|
| `GET /task`             | UNCHANGED   | `POST /api/v2/tasks/{id}/claim`                 | v1 auto-claims on GET          |
| `POST /create`          | UNCHANGED   | `POST /api/v2/tasks`                            | v1 creates in proj_default     |
| `POST /save`            | UNCHANGED   | `POST /api/v2/tasks/{id}/complete`              | v1 saves plain text output     |
| `POST /update-status`   | UNCHANGED   | `PATCH /api/v2/tasks/{id}`                      | v1 updates status only         |
| `GET /events` (SSE)     | UNCHANGED   | `GET /api/v2/events/stream`                     | v1 broadcasts all projects     |
| `POST /set-workdir`     | UNCHANGED   | `PATCH /api/v2/projects/proj_default`           | v1 only updates in-memory workdir |
| `GET /api/tasks`        | UNCHANGED   | `GET /api/v2/tasks`                             | v1 returns all projects mixed  |
| `POST /delete`          | UNCHANGED   | `DELETE /api/v2/tasks/{id}`                     | v1 deletes by task_id          |

### v2-Only Endpoints (No v1 Parity)

The following v2 endpoints introduce new capabilities and do not have v1 equivalents. They are intentionally v2-only and should not be considered part of the v1 compatibility surface:

- Task claim (`POST /api/v2/tasks/{id}/claim`)
- Runs detail/sub-resources (`GET /api/v2/runs/{id}`, `POST /api/v2/runs/{id}/steps`, `POST /api/v2/runs/{id}/logs`, `POST /api/v2/runs/{id}/artifacts`)
- Memory endpoints (for example, `GET /api/v2/memory/{key}`, `PUT /api/v2/memory/{key}`)
- Context packs endpoints (for example, `GET /api/v2/context-packs`, `POST /api/v2/context-packs`)
- Policy endpoints (for example, `GET /api/v2/projects/{projectId}/policies`, `POST /api/v2/projects/{projectId}/policies`, `GET /api/v2/projects/{projectId}/policies/{policyId}`, `PATCH /api/v2/projects/{projectId}/policies/{policyId}`, `DELETE /api/v2/projects/{projectId}/policies/{policyId}`)

**Policy Endpoints (v2-only mapping):**
- v1 parity: none. Policies are a v2-only project-scoped feature.
- Auth scope expectations: when scoped auth is enabled, policy reads require `v2:read` (or broader), and create/update/delete require `v2:write` (or broader).
- Behavior notes: policy rules are validated and normalized on create/update; disabled policies are ignored; missing projects or policy IDs return v2 error envelopes with 404.

## Data Model Compatibility

### Tasks Table Schema

```sql
-- v1 columns (original)
id              INTEGER PRIMARY KEY AUTOINCREMENT
instructions    TEXT NOT NULL
status          TEXT NOT NULL DEFAULT 'NOT_PICKED'
output          TEXT
parent_task_id  INTEGER
created_at      TEXT NOT NULL

-- v2 additions (nullable, with defaults)
project_id      TEXT                                -- backfilled to 'proj_default'
title           TEXT                                -- optional in v2
type            TEXT DEFAULT 'MODIFY'               -- defaults to MODIFY
priority        INTEGER DEFAULT 0                   -- defaults to 0
status_v2       TEXT DEFAULT 'QUEUED'               -- v2 writes keep in sync; v1 writes do not update
tags_json       TEXT                                -- JSON array, optional
updated_at      TEXT                                -- tracks last modification
```

**Compatibility Strategy:**
- v1 queries ignore new columns (they're nullable or have defaults)
- v2 queries read both old and new columns
- All v1 tasks have `project_id = 'proj_default'` after migration 0004
- `updated_at` is backfilled to `created_at`, set on create, and bumped on claims, v1 `POST /update-status`, v2 task updates, and completion (v1 `POST /save` or v2 complete)

### Status Synchronization

v2 endpoints write both `status` and `status_v2`. v1 endpoints write `status` only. When `status_v2` is empty, v2 reads map from `status`.

**Note:** `status_v2` defaults to `QUEUED` on inserts, so mapping only applies when rows truly lack `status_v2` (legacy data). v1 updates do not change `status_v2`, which can leave v2 reads showing a stale `status_v2` until a v2 write updates it.

#### v1 → v2 Status Mapping (Reads)

When reading a task created/updated via v1 (when `status_v2` is empty):

| v1 Status (`status`)  | v2 Status (`status_v2`)     |
|-----------------------|-----------------------------|
| `NOT_PICKED`          | `QUEUED`                    |
| `IN_PROGRESS`         | `RUNNING`                   |
| `COMPLETE`            | `SUCCEEDED`                 |

#### v2 → v1 Status Mapping (Reads)

When reading a task created/updated via v2:

| v2 Status (`status_v2`)  | v1 Status (`status`)       |
|--------------------------|----------------------------|
| `QUEUED`                 | `NOT_PICKED`               |
| `CLAIMED`                | `IN_PROGRESS`              |
| `RUNNING`                | `IN_PROGRESS`              |
| `SUCCEEDED`              | `COMPLETE`                 |
| `FAILED`                 | `COMPLETE`                 |
| `NEEDS_REVIEW`           | `COMPLETE`                 |
| `CANCELLED`              | `COMPLETE`                 |

**Current Implementation:**
- v2 handlers set both `status` and `status_v2` (claim, update, complete)
- v1 handlers set `status` only; `status_v2` is usually present due to the schema default
- v2 reads fall back to mapping only when `status_v2` is null or empty

## Endpoint Behavior Preservation

### GET /task

**v1 Behavior (Must Preserve):**
1. Find oldest `NOT_PICKED` task (ORDER BY id ASC)
2. Mark it `IN_PROGRESS`
3. Return instructions + parent context block (includes `UPDATED_AT` metadata)
4. If no `NOT_PICKED`, return existing `IN_PROGRESS` task
5. If none, return "No tasks available"

**Current Implementation Notes:**
- GET /task acquires an exclusive lease before returning a task
- The claim updates `status` to `IN_PROGRESS` and bumps `updated_at` (no `status_v2` write)
- A run record is created for the claim when possible
- Response includes `UPDATED_AT` in the text payload

### POST /create

**v1 Behavior (Must Preserve):**
- Body (form): `instructions=...`, optional `parent_task_id`, optional `project_id`
- Creates task with status `NOT_PICKED`
- Returns JSON with `task_id` and `updated_at`

**Current Implementation Notes:**
- Defaults `project_id` to `proj_default` when absent
- Writes `status` and `updated_at`; `status_v2` defaults to `QUEUED` via schema
- v2 reads only derive `status_v2` from `status` when `status_v2` is null or empty

### POST /save

**v1 Behavior (Must Preserve):**
- Body (form): `task_id=<id>&message=<output>`
- Updates task status to `COMPLETE`
- Stores output in `tasks.output`
- Returns a text response that includes `UPDATED_AT`

**Current Implementation Notes:**
- Updates `status` and `updated_at` only; `status_v2` is unchanged for v1 saves
- Completes the latest running run (if present) and releases any active lease

### POST /update-status

**v1 Behavior (Must Preserve):**
- Body (form): `task_id=<id>&status=<status>`
- Updates task `status` to one of `NOT_PICKED`, `IN_PROGRESS`, `COMPLETE`
- Returns JSON with `updated_at`

**Current Implementation Notes:**
- Updates `status` and `updated_at` only; `status_v2` is unchanged for v1 status updates

### GET /events (SSE)

**v1 Behavior (Must Preserve):**
- Returns SSE stream of tasks (optional `project_id` filter and `type=tasks` filter to scope updates)
- Supports `since` replay with optional `limit` (requires `since`; capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`)
- Sends full task list on connect
- Sends updates on task changes
- Format: `event: tasks` + `data: [...]`, plus keep-alive ping comments

**Current Implementation Notes:**
- `limit` requires `since` and is capped by `COCO_V1_EVENTS_REPLAY_LIMIT_MAX`
- Initial payload is filtered by `project_id` and `since` when supplied
- Heartbeat comments (`: ping`) keep connections alive

### POST /set-workdir

**v1 Behavior (Must Preserve):**
- Body (form): `workdir=/home/user/project`
- Updates in-memory workdir only
- Returns JSON `{ "success": true, "workdir": "..." }`

**v2 Implications:**
- v2 clients set workdir per project via PATCH/PUT /api/v2/projects/{id}
- v1 workdir is in-memory only and does not update `projects.workdir`

### GET /api/tasks

**v1 Behavior (Must Preserve):**
- Returns JSON object: `{ "tasks": [...], "total": <int> }`
- Ordered by created_at DESC by default
- Supports `status`, `updated_since`, `project_id` filters
- Supports `sort` (`created_at:asc`, `created_at:desc`, `updated_at`)
- Supports `limit`/`offset` pagination
- Includes v1 fields: id, instructions, status, output, parent_task_id, created_at, updated_at

**v2 Implications:**
- v1 response remains v1-only; v2 fields are available through v2 list endpoints

## Error Envelope and Auth (v2)

- All v2 4xx/5xx responses use the JSON error envelope (`{"error":{"code","message","details"}}`)
- Unauthorized v2 requests return `UNAUTHORIZED`; insufficient scope returns `FORBIDDEN` with `required_scope`
- v1 endpoints keep legacy error formats (plain text or small JSON objects)

## Testing Strategy

### Compatibility Test Suite

#### Test 1: v1 Create → v2 Read
```go
func TestV1CreateV2Read(t *testing.T) {
    // Create task via v1 endpoint
    resp := postJSON("/create", map[string]interface{}{
        "instructions": "Test task",
    })
    taskID := resp["task_id"]
    
    // Read via v2 endpoint
    v2Resp := getJSON(fmt.Sprintf("/api/v2/tasks/%d", taskID))
    
    // Assert v2 fields have expected defaults
    assert.Equal(t, "proj_default", v2Resp["task"]["project_id"])
    assert.Equal(t, "MODIFY", v2Resp["task"]["type"])
    assert.Equal(t, 0, v2Resp["task"]["priority"])
    assert.Equal(t, "QUEUED", v2Resp["task"]["status_v2"])
}
```

#### Test 2: v2 Create → v1 Read
```go
func TestV2CreateV1Read(t *testing.T) {
    // Create task via v2 endpoint
    resp := postJSON("/api/v2/projects/proj_default/tasks", map[string]interface{}{
        "instructions": "Test task",
    })
    taskID := resp["task"]["id"]
    
    // Read via v1 endpoint (GET /task)
    v1Resp := getText("/task")
    
    // Assert task is returned and marked IN_PROGRESS
    assert.Contains(t, v1Resp, "Test task")
    
    // Check v1 status is updated; status_v2 remains at the v2-created value
    task := getTaskFromDB(taskID)
    assert.Equal(t, "IN_PROGRESS", task.Status)
    assert.Equal(t, "QUEUED", task.StatusV2)
}
```

#### Test 3: v1 Save → v2 Read Completion
```go
func TestV1SaveV2ReadCompletion(t *testing.T) {
    // Create task
    taskID := createTask("Test task")
    
    // Save via v1 endpoint
    postForm("/save", map[string]string{
        "task_id": strconv.Itoa(taskID),
        "message": "Task completed",
    })
    
    // Read via v2 endpoint
    v2Resp := getJSON(fmt.Sprintf("/api/v2/tasks/%d", taskID))
    
    // Assert v1 status is COMPLETE and status_v2 remains unchanged
    assert.Equal(t, "COMPLETE", v2Resp["task"]["status"])
    assert.Equal(t, "QUEUED", v2Resp["task"]["status_v2"])
    assert.Equal(t, "Task completed", v2Resp["task"]["output"])
}
```

#### Test 4: v2 Complete → v1 Read Output
```go
func TestV2CompleteV1ReadOutput(t *testing.T) {
    // Create and claim task via v2
    taskID := createTaskV2("Test task")
    postJSON(fmt.Sprintf("/api/v2/tasks/%d/claim", taskID), map[string]string{
        "agent_id": "test_agent",
    })
    
    // Complete via v2 endpoint with required result fields and a v1 output message
    postJSON(fmt.Sprintf("/api/v2/tasks/%d/complete", taskID), map[string]interface{}{
        "message": "Implemented feature X",
        "result": map[string]interface{}{
            "summary": "Implemented feature X",
            "changes_made": []string{"Added file Y", "Modified file Z"},
            "files_touched": []string{"src/y.go", "src/z.go"},
        },
    })
    
    // Read via v1 endpoint
    task := getTaskFromDB(taskID)
    
    // Assert v1 output is populated with message/output
    assert.Contains(t, task.Output, "Implemented feature X")
    assert.Equal(t, "COMPLETE", task.Status)
}
```

#### Test 5: v1 SSE Receives v2 Events
```go
func TestV1SSEReceivesV2Events(t *testing.T) {
    // Connect to v1 SSE
    sseClient := connectSSE("/events")
    
    // Create task via v2
    taskID := createTaskV2("New task")
    
    // Assert v1 SSE receives update
    event := sseClient.waitForEvent(5 * time.Second)
    assert.NotNil(t, event)
    
    // Parse event data (should contain task with v2 fields)
    var tasks []Task
    json.Unmarshal([]byte(event.Data), &tasks)
    
    found := false
    for _, t := range tasks {
        if t.ID == taskID {
            found = true
            // v1 clients see v2 fields but can ignore them
            assert.Equal(t, "proj_default", t.ProjectID)
            break
        }
    }
    assert.True(t, found)
}
```

## Migration Path for Clients

### Phase 1: Preparation (Week 0)
- Announce v2 availability
- Publish v2 documentation and OpenAPI spec
- Ensure v1 tests pass with v2 schema
- Deploy v2 endpoints alongside v1

### Phase 2: Parallel Operation (Months 1-3)
- Both v1 and v2 endpoints active
- New clients adopt v2 for new features
- Existing v1 clients continue unchanged
- Monitor usage metrics (v1 vs v2 traffic)

### Phase 3: Gradual Migration (Months 4-6)
- Identify high-value v1 clients
- Provide migration guides and support
- Offer incentives (e.g., access to v2-only features)
- Continue v1 support with security/bug fixes only

### Phase 4: v1 Deprecation (Month 7+)
- Announce v1 deprecation timeline (e.g., 6 months notice)
- Set v1 endpoints to return deprecation warnings
- Continue full functionality until deprecation date
- Final migration of remaining v1 clients

### Phase 5: v1 Sunset (Year 2+)
- Remove v1 endpoint handlers (routes return 410 Gone)
- Keep v1 data compatibility (old tasks still readable)
- Archive v1 documentation as "legacy"

## Rollback Plan

If v2 causes issues, rollback is straightforward:

### Rollback Procedure
1. **Stop deployment**: Halt v2 endpoint rollout
2. **Database**: v2 schema additions are backward compatible (nullable columns)
3. **Code**: Revert to version without v2 handlers
4. **Data**: v1 data is unchanged, v2 data (runs, leases) can be ignored
5. **SSE**: v1 SSE continues working with original broadcast logic

### Data Preservation
- Tasks created via v2 remain in database
- v1 can read them (project_id, type, etc. are optional)
- Runs, leases, events tables are unused by v1 (no impact)

## Monitoring and Observability

### Metrics to Track

- **v1 Endpoint Usage**: Requests/sec to /task, /create, /save
- **v2 Endpoint Usage**: Requests/sec to /api/v2/*
- **Status Sync Errors**: Mismatches between status and status_v2
- **SSE Connections**: v1 vs v2 SSE client counts
- **Dual-Client Tasks**: Tasks accessed by both v1 and v2 clients

### Alerts

- **Status Desync**: Alert if status != expected mapping of status_v2
- **v1 Errors**: Spike in v1 endpoint 5xx errors after v2 deployment
- **Migration Progress**: Weekly report on v1 → v2 traffic ratio

## Conclusion

Backward compatibility is achieved through:

1. **Endpoint Separation**: v1 at root, v2 at /api/v2/*
2. **Schema Addition**: New columns don't break v1 queries
3. **Status Synchronization**: deterministic mapping exists; v2 writes both fields, v1 writes `status` only
4. **Default Project**: v1 operates on proj_default
5. **Output Bridge**: v2 completion writes v1 output only when `output` or `message` is provided; result summaries are not auto-synthesized

This plan ensures zero disruption to existing clients while enabling new capabilities for v2 adopters.

---

**Next Steps:**
1. Decide if v1 writes should update or clear `status_v2` for tighter v1/v2 sync
2. Write/extend compatibility test suite
3. Deploy v2 alongside v1
4. Monitor metrics and validate compatibility

**Related Documents:**
- [v2 Design Document](v2-design.md)
- [OpenAPI Specification](openapi-v2.yaml)
- [Database Schema](../schema/v2-migrations.sql)
