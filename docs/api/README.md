# API Documentation Index

This directory contains the complete API v2 contract design and supporting documentation for Cocopilot.

## Quick Start

**New to v2?** Start here:
1. Read [v2-summary.md](v2-summary.md) - Executive summary and quick reference
2. Review [openapi-v2.yaml](openapi-v2.yaml) - Complete API specification (VSIX: `Cocopilot: Open OpenAPI Spec`)
3. Check [v2-compatibility.md](v2-compatibility.md) - v1/v2 compatibility strategy

**Ready to implement?**
1. Study [v2-design.md](v2-design.md) - Comprehensive design document
2. Review [../schema/v2-migrations.sql](../schema/v2-migrations.sql) - Database schema
3. Follow [v2-roadmap.md](v2-roadmap.md) - Phased implementation plan

## Documents

| File | Purpose | Audience |
|------|---------|----------|
| **[v2-summary.md](v2-summary.md)** | Executive summary with quick reference | All stakeholders |
| **[openapi-v2.yaml](openapi-v2.yaml)** | OpenAPI 3.0 specification for v2 API | API consumers, developers |
| **[v2-design.md](v2-design.md)** | Comprehensive design document with architecture and rationale | Architects, developers |
| **[v2-compatibility.md](v2-compatibility.md)** | Backward compatibility strategy and migration path | Developers, v1 users |
| **[v2-roadmap.md](v2-roadmap.md)** | Phased implementation plan with timeline | Project managers, developers |
| **[v2-health-version.md](v2-health-version.md)** | Health check and versioning contract | Developers, operators |

## API v2 Features

### Core Enhancements
- **Multi-project support** - Isolated workspaces with independent settings
- **Execution ledger** - Track runs, steps, logs, and artifacts
- **Lease-based claiming** - Agent coordination with expiration and heartbeat
- **Structured completion** - Rich metadata for changes, files, tests, risks
- **Real-time events** - Project-scoped SSE with replay capability (stream/replay endpoints)
- **Project audit log** - Project-scoped event list with filters and pagination
- **Persistent memory** - Knowledge base that accumulates across tasks
- **Context packs** - Automated context generation for tasks
- **Project tree and changes** - Shallow workdir snapshot and change feed per project

### Context Pack Detail
`GET /api/v2/context-packs/{packId}` returns a previously generated context pack by id.

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

### Automation Engine Configuration
- **Environment**: `COCO_AUTOMATION_RULES` is a JSON array of automation rules. When unset or empty, automation is disabled and no follow-up tasks are created.
- **Endpoint**: `GET /api/v2/projects/{projectId}/automation/rules` returns the current server rules (applies to all projects; empty array when disabled).
- **Simulation**: `POST /api/v2/projects/{projectId}/automation/simulate` previews actions and tasks that would be created for a supported event.
- **Rule fields**: `name`, `enabled` (default: true), `trigger` (only `task.completed`), `actions` (non-empty).
- **Action fields**: `type` must be `create_task`; `task.instructions` is required; `task.title`, `task.type`, `task.priority` (non-negative), `task.tags`, `task.parent` are optional.
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

**Replay example:**
`POST /api/v2/projects/{projectId}/automation/replay?since_event_id=evt_auto_1&limit=100`

**Replay response:**
```json
{
  "since_event_id": "evt_auto_1",
  "events_replayed": 3,
  "task_completed_events": 2
}
```

### Policy Rule Catalog
Policies are project-scoped and stored in `policies.rules` via the policies endpoints. Rules are validated and normalized on create/update.

**Rule validation and normalization:**
- `type` is required, trimmed, and lowercased.
- Allowed types: `automation.block`, `completion.block`, `task.create.block`, `task.update.block`, `task.delete.block`.
- `reason` is optional; if present it must be a string.
- Disabled policies (`enabled: false`) are ignored.

**Enforcement behavior:**
- The first matching rule in the first enabled policy blocks the action.
- Blocking returns `FORBIDDEN` with a policy-specific message and `reason` in `error.details`.

**Rule effects:**
- `automation.block`: prevents automation follow-up creation from `COCO_AUTOMATION_RULES` on `task.completed` events.
- `completion.block`: blocks `POST /api/v2/tasks/{taskId}/complete`.
- `task.create.block`: blocks `POST /api/v2/tasks` and `POST /api/v2/projects/{projectId}/tasks`.
- `task.update.block`: blocks `PATCH /api/v2/tasks/{taskId}`.
- `task.delete.block`: blocks `DELETE /api/v2/tasks/{taskId}`.

**Example policy payload:**
```json
{
  "name": "Default Policy",
  "description": "Block sensitive operations",
  "enabled": true,
  "rules": [
    {
      "type": "completion.block",
      "reason": "Completion requires manual review"
    },
    {
      "type": "task.delete.block",
      "reason": "Deletes are disabled for this project"
    }
  ]
}
```

**Example blocked response:**
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Task delete blocked by policy: Deletes are disabled for this project",
    "details": {
      "task_id": 123,
      "project_id": "proj_default",
      "reason": "Deletes are disabled for this project"
    }
  }
}
```

### Project Audit
`GET /api/v2/projects/{projectId}/audit` returns a filtered, paginated list of audit events for a project.

**Auth:** When scoped identities are enabled, requires `events.read` or `audit.read`.

**Filters:**
- `type` - Filter by event kind (for example, `task.created`).
- `task_id` - Filter by task ID.
- `since` - Only return events created at or after this timestamp (RFC3339).
- `limit` - Page size (default 100, max 500).
- `offset` - Page offset (default 0).

**Pagination:** The response includes `total` with the number of events matching the filters.

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

### Feature References
- **Runs**: [openapi-v2.yaml](openapi-v2.yaml) (runs endpoints), [v2-design.md](v2-design.md) (execution ledger)
- **Memory**: [openapi-v2.yaml](openapi-v2.yaml) (memory endpoints), [v2-design.md](v2-design.md) (memory model)
- **Context packs**: [openapi-v2.yaml](openapi-v2.yaml) (context pack endpoints), [v2-design.md](v2-design.md) (context pack flow)
- **Project tree/changes**: [openapi-v2.yaml](openapi-v2.yaml) (project tree/changes endpoints), [v2-design.md](v2-design.md) (repo perception)
- **Tool invocations**: Available via run detail (`tool_invocations` on runs detail responses)

### Backward Compatibility
- All v1 endpoints unchanged at root level
- v1 and v2 share same database (coexistence)
- Status sync between v1 and v2 formats
- v1 output auto-generated from v2 completion

### Implementation Notes (2026-02-11)
- Implemented: tasks endpoints (create/list/detail/update/delete/claim/complete + filters), project task list/create, task dependencies, project tree/changes, runs detail/steps/logs/artifacts, memory put/get, context packs, agents list/detail/delete, leases create/heartbeat/release, events list/stream, project events stream/replay, config, version, auth guardrails, standardized v2 error envelope
- Implemented: project audit list (`GET /api/v2/projects/{projectId}/audit`) with filters and pagination
- Implemented: project policy endpoints (create/list/detail/update/delete)
- Planned: projects CRUD

## API Structure

```
/                           v1 endpoints (legacy, unchanged)
├── GET  /task
├── POST /create
├── POST /save
├── GET  /events
├── POST /set-workdir
└── GET  /api/tasks

/api/v2/                    v2 endpoints (new features)
├── health                  Health check and version info
├── projects                Multi-project management
│   ├── /{projectId}/tasks  Project-scoped tasks (list/create)
│   ├── /{projectId}/events/stream Project-scoped SSE
│   ├── /{projectId}/events/replay Project-scoped event replay
│   ├── /{projectId}/audit  Project-scoped audit list
│   ├── /{projectId}/automation/simulate Automation preview for task.completed
│   ├── /{projectId}/tree   Project workdir snapshot
│   ├── /{projectId}/changes Project change feed
│   ├── /{projectId}/policies Project-scoped policies
│   ├── /{projectId}/policies/{policyId} Policy detail/update/delete
│   ├── /{projectId}/memory Persistent knowledge
│   └── /{projectId}/context-packs
├── context-packs            Context pack detail
│   └── /{packId}
├── tasks                   Enhanced task operations
│   ├── /{taskId}/claim     Lease-based claiming
│   └── /{taskId}/complete  Structured completion
├── runs                    Execution ledger
│   ├── /{runId}/steps      Log execution steps
│   ├── /{runId}/logs       Stream logs
│   └── /{runId}/artifacts  Attach artifacts
└── leases                  Agent coordination
    ├── /{leaseId}/heartbeat
    └── /{leaseId}/release
```

  ## MCP Tool Catalog

  The MCP server in [tools/cocopilot-mcp/](../../tools/cocopilot-mcp/) exposes Cocopilot capabilities as MCP tools backed by HTTP endpoints. Tool inputs map to the matching path params and request bodies for the endpoint.

  ### v2 Tool Mapping

  | MCP tool | HTTP endpoint |
  |----------|---------------|
  | `coco.project.list` | `GET /api/v2/projects` |
  | `coco.project.create` | `POST /api/v2/projects` |
  | `coco.project.update` | `PATCH /api/v2/projects/{projectId}` |
  | `coco.project.get` | `GET /api/v2/projects/{projectId}` |
  | `coco.project.delete` | `DELETE /api/v2/projects/{projectId}` |
  | `coco.config.get` | `GET /api/v2/config` |
  | `coco.version.get` | `GET /api/v2/version` |
  | `coco.health.get` | `GET /api/v2/health` |
  | `coco.agent.list` | `GET /api/v2/agents` |
  | `coco.agent.get` | `GET /api/v2/agents/{agentId}` |
  | `coco.agent.delete` | `DELETE /api/v2/agents/{agentId}` |
  | `coco.project.tasks.list` | `GET /api/v2/projects/{projectId}/tasks` |
  | `coco.project.memory.query` | `GET /api/v2/projects/{projectId}/memory` |
  | `coco.project.audit.list` | `GET /api/v2/projects/{projectId}/audit` |
  | `coco.project.tree` | `GET /api/v2/projects/{projectId}/tree` |
  | `coco.project.changes` | `GET /api/v2/projects/{projectId}/changes` |
  | `coco.project.events.replay` | `GET /api/v2/projects/{projectId}/events/replay` |
  | `coco.policy.list` | `GET /api/v2/projects/{projectId}/policies` |
  | `coco.policy.get` | `GET /api/v2/projects/{projectId}/policies/{policyId}` |
  | `coco.policy.create` | `POST /api/v2/projects/{projectId}/policies` |
  | `coco.policy.update` | `PATCH /api/v2/projects/{projectId}/policies/{policyId}` |
  | `coco.policy.delete` | `DELETE /api/v2/projects/{projectId}/policies/{policyId}` |
  | `coco.context_pack.create` | `POST /api/v2/projects/{projectId}/context-packs` |
  | `coco.task.create` | `POST /api/v2/tasks` |
  | `coco.task.list` | `GET /api/v2/tasks` |
  | `coco.task.complete` | `POST /api/v2/tasks/{taskId}/complete` |
  | `coco.task.update` | `PATCH /api/v2/tasks/{taskId}` |
  | `coco.task.delete` | `DELETE /api/v2/tasks/{taskId}` |
  | `coco.task.dependencies.list` | `GET /api/v2/tasks/{taskId}/dependencies` |
  | `coco.task.dependencies.create` | `POST /api/v2/tasks/{taskId}/dependencies` |
  | `coco.task.dependencies.delete` | `DELETE /api/v2/tasks/{taskId}/dependencies/{dependsOnTaskId}` |
  | `coco.lease.create` | `POST /api/v2/leases` |
  | `coco.lease.heartbeat` | `POST /api/v2/leases/{leaseId}/heartbeat` |
  | `coco.lease.release` | `POST /api/v2/leases/{leaseId}/release` |
  | `coco.events.list` | `GET /api/v2/events` |
  | `coco.run.get` | `GET /api/v2/runs/{runId}` |
  | `coco.run.steps` | `POST /api/v2/runs/{runId}/steps` |
  | `coco.run.logs` | `POST /api/v2/runs/{runId}/logs` |
  | `coco.run.artifacts` | `POST /api/v2/runs/{runId}/artifacts` |

  ### v1 Compatibility Tools

  | MCP tool | HTTP endpoint |
  |----------|---------------|
  | `coco.task.claim` | `GET /task` |
  | `coco.task.save` | `POST /save` (form-encoded `task_id`, `message`) |

  For the full tool list and usage examples, see [tools/cocopilot-mcp/README.md](../../tools/cocopilot-mcp/README.md).

## v2 Error Contract

All current v2 handlers return JSON errors in the same shape:

```json
{
  "error": {
    "code": "METHOD_NOT_ALLOWED",
    "message": "Method not allowed",
    "details": {
      "method": "POST",
      "allowed_methods": ["GET"]
    }
  }
}
```

`details` is optional and carries endpoint-specific context (`task_id`, `agent_id`, path info, etc.).

In `openapi-v2.yaml`, operations that are design targets but not yet live are marked with `x-runtime-status: planned`.
When auth is enabled via runtime config, v2 routes use `X-API-Key` and return `UNAUTHORIZED` in the same envelope format.
When a key is valid but lacks required scope, v2 routes return `FORBIDDEN` with `error.details.required_scope`.

## Database Schema

| Migration | Purpose | Status |
|-----------|---------|--------|
| 0001 | Schema migrations tracking | ✅ Applied |
| 0002 | Tasks v1 baseline | ✅ Applied |
| 0003 | Projects table | ✅ Applied |
| 0004 | Tasks.project_id column | ✅ Applied |
| 0005 | Tasks v2 enhancements | ✅ Applied |
| 0006 | Runs ledger | ✅ Applied |
| 0007 | Leases | ✅ Applied |
| 0008 | Events | ✅ Applied |
| 0009 | Memory | ✅ Applied |
| 0010 | Context packs | ✅ Applied |
| 0011 | Tasks project FK and defaults | ✅ Applied |
| 0012 | Agents | ✅ Applied |
| 0013 | Task dependencies | ✅ Applied |
| 0014 | Events project_id backfill | ✅ Applied |
| 0015 | Event filtering indexes | ✅ Applied |
| 0016 | Task sort indexes | ✅ Applied |
| 0017 | tasks.updated_at backfill | ✅ Applied |
| 0018 | Policy engine foundation | ✅ Applied |

See [../schema/v2-migrations.sql](../schema/v2-migrations.sql) for complete schema.

## Implementation Status

| Phase | Target | Status |
|-------|--------|--------|
| Design | Week 0 | ✅ Complete |
| Phase 1: Foundation | Week 1 | In progress (tasks/agents/events/config/version live; projects CRUD pending) |
| Phase 2: Execution Ledger | Week 2 | Implemented (runs detail/steps/logs/artifacts live) |
| Phase 3: Leases & Claiming | Week 2-3 | ✅ Implemented |
| Phase 4: Structured Completion | Week 3 | ✅ Implemented (endpoint live; follow-on tests pending) |
| Phase 5: Events & Memory | Week 4 | Events and memory implemented |
| Phase 6: Context Packs | Week 4 | ✅ Implemented |
| Phase 7: Polish & Docs | Week 5 | In progress |

Target Launch: **March 14, 2026**

## Key Design Decisions

1. **v1 Stability**: All v1 endpoints must remain unchanged to prevent breaking existing clients
2. **Dual Status**: Maintain both v1 and v2 status fields for smooth transition
3. **Default Project**: Use `proj_default` to make v1 implicitly single-project
4. **Namespace Separation**: v2 under `/api/v2/*` to avoid conflicts
5. **Progressive Enhancement**: v2 capabilities built on top of v1 data model
6. **Feature Flags**: Risky features (leases, tool execution) can be disabled
7. **SQLite First**: Optimize for SQLite, but design for PostgreSQL migration
8. **No Breaking Changes**: v2 can be rolled back without data loss

## Testing Strategy

### Test Coverage Goals
- Unit tests: 80%+ coverage
- Integration tests: All workflows
- Compatibility tests: v1/v2 interoperability
- Performance tests: 1K tasks, 100 agents
- Security tests: SQL injection, path traversal

### Critical Test Scenarios
- ✅ v1 create → v2 read
- ✅ v2 create → v1 read
- ✅ v1 save → v2 completion
- ✅ v2 complete → v1 output
- ✅ Lease conflict handling
- ✅ Lease expiration and reclaim
- ✅ SSE event delivery
- ✅ Status sync accuracy

## Related Documents

### Current System
- [../../server/main.go](../../server/main.go) - Current v1 implementation
- [../../server/migrations.go](../../server/migrations.go) - Migration runner
- [../../migrations/](../../migrations/) - Applied migrations

### Architecture
- [../state/architecture.md](../state/architecture.md) - Current architecture
- [../ai/kb/01-architecture.md](../ai/kb/01-architecture.md) - AI knowledge base

## Tools and Resources

### OpenAPI Tools
- **Swagger Editor**: https://editor.swagger.io/ - Visualize and test API
- **Swagger UI**: Host locally to explore API interactively
- **OpenAPI Generator**: Generate client SDKs for various languages

### Database Tools
- **SQLite Browser**: https://sqlitebrowser.org/ - Explore database
- **DBeaver**: Universal database tool with SQLite support

### Testing Tools
- **curl**: Command-line HTTP client for manual testing
- **Postman**: GUI for API testing and collection management
- **k6**: Load testing tool for performance validation

### Go Libraries
- **github.com/google/uuid**: UUID generation for IDs
- **github.com/oklog/ulid**: ULID generation for event IDs
- **modernc.org/sqlite**: Pure Go SQLite driver

## Contributing

When working on v2 implementation:

1. **Follow the roadmap** - Implement phases in order
2. **Write tests first** - TDD approach for reliability
3. **Update docs** - Keep OpenAPI spec in sync with code
4. **Test compatibility** - Run v1 tests after each v2 change
5. **Review schema** - Ensure migrations are idempotent

## Support

**Questions? Issues?**
- Review the design docs first
- Check the compatibility plan for v1/v2 questions
- Consult the roadmap for implementation guidance
- Refer to OpenAPI spec for endpoint details

---

**Last Updated:** February 11, 2026  
**Design Version:** 2.0.0  
**Status:** Implementation underway (snapshot 2026-02-11)
