cocopilot/docs/ai/kb/02-invariants.md
```

```cocopilot/docs/ai/kb/02-invariants.md
# 02 - Invariants

This document outlines the core invariants of the project. These are non-negotiable rules that must not be violated under any circumstances unless explicitly revised in the Knowledge Base (KB). Any changes to these invariants must follow the process outlined in the `.rules` file.

---

## Global Invariants
1. **Correctness First**: All implementations must prioritize correctness above all else. Incorrect behavior is unacceptable, even if it improves performance or reduces complexity.
2. **Security by Design**: The system must adhere to the principles of least privilege, safe defaults, and input validation. No sensitive data (e.g., secrets, PII) should be logged or exposed.
3. **Deterministic Behavior**: The system must produce consistent and predictable results for the same inputs, regardless of external factors.
4. **No Silent Failures**: All errors must be logged with sufficient context to enable debugging. Silent failures are not allowed.
5. **Atomic KB Entries**: Each KB entry must represent a single fact, decision, or invariant. No duplication of information is allowed.

---

## Module/Directory-Specific Invariants
1. **Task Management**:
   - Tasks must have one of the following states: `NOT_PICKED`, `IN_PROGRESS`, or `COMPLETE`.
   - Tasks must transition between states in a valid order: `NOT_PICKED` → `IN_PROGRESS` → `COMPLETE`.
   - Parent-child task relationships must preserve context integrity. Child tasks must inherit relevant context from their parent tasks.
   - v2 task completion with `result.next_tasks` must validate each entry (non-empty `instructions`, optional `title`, `type`, `priority`, `tags`) and create child tasks in the same project with `parent_task_id` set; each child emits a `task.created` event and the completion response may include `next_tasks` when available.

2. **API Endpoints**:
   - All API endpoints must validate inputs and return appropriate HTTP status codes.
   - API responses must include clear and actionable error messages for invalid requests.
   - Task-related endpoints must ensure data consistency in the SQLite database.
   - v2 events endpoints must validate and apply filters (`project_id`, `type`, `since`, `task_id`, `limit`, `offset`) and enforce replay caps.
   - Project-scoped event streaming and replay endpoints remain available at `GET /api/v2/projects/{projectId}/events/stream` and `GET /api/v2/projects/{projectId}/events/replay`.
   - v2 runs sub-resource endpoints must validate run existence before accepting steps, logs, or artifacts.
   - v2 project memory reads must honor `scope`, `key`, and `q` filters and remain project-scoped.
   - v2 context packs must be created only for valid project/task pairs and remain project-scoped.

3. **Database**:
   - The SQLite database schema must not be altered without an explicit update to the KB.
   - All database operations must be atomic to prevent data corruption.

4. **Web Interface**:
   - The Kanban UI must reflect the real-time state of tasks in the database.
   - Drag-and-drop operations must be synchronized with the backend to ensure consistency.

---

## Security Invariants
1. **Authentication and Authorization**:
   - v2 endpoints must enforce API key guardrails when enabled (`COCO_REQUIRE_API_KEY` and `COCO_REQUIRE_API_KEY_READS`).
   - v2 scope checks must honor configured identity scopes (`v2:read`, `v2:write`, `tasks:read`, `tasks:write`, `leases:write`, `agents:write`, `runs:write`, `projects:write`).
   - Sensitive operations (e.g., task deletion, status updates) must be restricted to authorized users.

2. **Data Protection**:
   - No sensitive data (e.g., API keys, passwords) should be stored in plaintext.
   - Logs must not contain sensitive information.

3. **Network Security**:
   - All communication between the server, agents, and web interface must use secure protocols (e.g., HTTPS).

---

## Revision Process
1. Any proposed changes to these invariants must be documented in the KB under `03-decisions.md`.
2. The proposed change must include:
   - The reason for the change.
   - The tradeoffs involved.
   - The potential risks and mitigations.
3. Changes to invariants must be approved before implementation.

---

End of Document