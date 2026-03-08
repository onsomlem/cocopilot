cocopilot\docs\ai\kb\06-open-questions.md
```

```cocopilot/docs/ai/kb/06-open-questions.md
# Open Questions

This document captures unresolved questions and areas requiring further exploration or clarification. These questions should be revisited periodically to ensure progress and alignment with the project's goals.

---

## Resolved (2026-02-11)
- Runs sub-resources (steps, logs, artifacts) are implemented; no open questions remain here.
- Project memory endpoints are implemented; no open questions remain here.
- Context packs are implemented; no open questions remain here.

## Resolved (2026-03-08)
- **Task Prioritization**: Tasks have a `priority` field (0-100). Claim-next returns the highest-priority OPEN task. Implemented.
- **Agent Behavior**: Lease-based task claiming with configurable expiry handles agent failures/disconnects. Stalled task detection is implemented in `internal/notifications/`.
- **Database Management**: Full migration system implemented (18 migrations in `migrations/`). Auto-applies on startup. See `MIGRATIONS.md`.
- **Error Handling and Logging**: Structured v2 error responses (`writeV2Error`), request logging middleware, event system for audit trail. Implemented.
- **Integration with External Systems**: Webhook notifications implemented in `internal/notifications/`. MCP server for VS Code in `tools/cocopilot-mcp/`.
- **Testing and Validation**: Comprehensive test suite (30+ test files), race-condition-free, benchmarks in `load_test.go`. All passing.
- **Task Dependencies**: Implemented with dependency graph support and blocked-task detection.

---

## Open

### Security
- What is the desired rotation and revocation workflow for v2 API keys and scoped identities?
- Should we add stronger auth (mTLS or OIDC) beyond the current API key guardrails?

### Scalability
- What are the performance limits of the current SQLite architecture at high concurrency?
- Should horizontal scaling be considered for future deployments?

### UI Enhancements
- Should users be able to customize the Kanban board columns or layout?

### Future Features
- Analytics and reporting capabilities
- Advanced workflow orchestration beyond current automation rules