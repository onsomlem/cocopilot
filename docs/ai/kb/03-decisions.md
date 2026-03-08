cocopilot/docs/ai/kb/03-decisions.md
```

```cocopilot/docs/ai/kb/03-decisions.md
# Decision Log (03-decisions.md)

This document serves as the decision log for the project. All architectural, strategic, and engineering decisions are recorded here to ensure traceability, transparency, and alignment with the project's goals and constraints.

Each decision is documented with the following structure:
- **Date**: The date the decision was made.
- **Status**: Locked (cannot be changed) or Revisitable (can be revisited under specific circumstances).
- **Context**: The situation or problem that led to the decision.
- **Decision**: The decision made.
- **Tradeoffs**: The pros and cons of the decision.
- **Consequences**: The expected outcomes or impacts of the decision.

---

## Decision Records

### Decision 1: Establish Knowledge Base (KB)
- **Date**: [Insert Date]
- **Status**: Locked
- **Context**: The project requires a centralized, structured, and authoritative source of truth for all decisions, architecture, constraints, and risks.
- **Decision**: Create a Knowledge Base (KB) in `docs/ai/kb/` with the following structure:
  - `00-overview.md`
  - `01-architecture.md`
  - `02-invariants.md`
  - `03-decisions.md`
  - `04-constraints.md`
  - `05-risks.md`
  - `06-open-questions.md`
  - `07-future.md`
- **Tradeoffs**:
  - **Pros**:
    - Ensures all decisions are well-documented and traceable.
    - Facilitates collaboration and alignment across the team.
    - Provides a single source of truth for the project's long-term memory.
  - **Cons**:
    - Requires ongoing maintenance and discipline to keep the KB up to date.
- **Consequences**:
  - The KB will serve as the foundation for all future decisions and planning.
  - Any changes to the project's behavior, architecture, or assumptions must be reflected in the KB.

---

### Decision 2: Prioritize Correctness Over Other Factors
- **Date**: [Insert Date]
- **Status**: Locked
- **Context**: The project requires a strict prioritization of correctness to ensure reliability and trustworthiness of the system.
- **Decision**: Adopt the following priority order for all decisions:
  1. Correctness
  2. Security
  3. Maintainability
  4. Performance
  5. Style
- **Tradeoffs**:
  - **Pros**:
    - Guarantees a robust and reliable system.
    - Reduces the risk of critical failures.
  - **Cons**:
    - May result in slower development cycles.
    - Performance optimizations may be delayed in favor of correctness.
- **Consequences**:
  - All decisions must align with this priority order.
  - Any deviation from this order must be explicitly justified and documented.

---

### Decision 3: Dual-Voice Workflow
- **Date**: [Insert Date]
- **Status**: Locked
- **Context**: To ensure structured and efficient task execution, a dual-phase workflow is required.
- **Decision**: Implement a dual-voice workflow with the following phases:
  - **Phase A (Planner)**: Focus on problem restatement, constraints, options, chosen path, verification hooks, and pre-mortem analysis.
  - **Phase B (Executor)**: Implement the chosen plan with minimal, reviewable changes and proper testing.
- **Tradeoffs**:
  - **Pros**:
    - Encourages thorough planning and reduces errors during implementation.
    - Ensures alignment with project goals and constraints.
  - **Cons**:
    - May increase the time required for task completion.
- **Consequences**:
  - All non-trivial tasks must follow the dual-voice workflow.
  - Tasks that skip the planning phase are considered invalid.

---

### Decision 4: Use SQLite for Task Storage
- **Date**: [Insert Date]
- **Status**: Revisitable
- **Context**: The project requires a lightweight and easy-to-use database for storing tasks.
- **Decision**: Use SQLite as the database for task storage.
- **Tradeoffs**:
  - **Pros**:
    - Simple to set up and use.
    - No need for a separate database server.
    - Well-suited for small to medium-sized projects.
  - **Cons**:
    - Limited scalability for large-scale systems.
    - May require migration to a more robust database in the future.
- **Consequences**:
  - The current implementation is optimized for small to medium workloads.
  - Future scaling may require transitioning to a different database solution.

---

### Decision 5: API Design for Task Management
- **Date**: [Insert Date]
- **Status**: Revisitable
- **Context**: The project requires a simple and intuitive API for managing tasks and interacting with LLM agents.
- **Decision**: Keep v1 endpoints stable for compatibility and implement v2 endpoints for richer workflows.
  - **v1 (compat)**:
    - `GET /task`: Retrieve the next available task.
    - `POST /create`: Create a new task.
    - `POST /save`: Save task output and mark it as complete.
    - `POST /update-status`: Update the status of a task.
    - `POST /delete`: Delete a task.
    - `GET /api/tasks`: Retrieve all tasks in JSON format.
    - `GET /api/workdir`: Get the current working directory.
    - `POST /set-workdir`: Set the working directory.
    - `GET /events`: Provide real-time task updates for the web UI (supports `project_id`, `type=tasks`, `since`, `limit`).
  - **v2 (implemented)**:
    - Tasks: `POST /api/v2/tasks`, `GET /api/v2/tasks`, `GET /api/v2/tasks/{taskId}`, `PATCH /api/v2/tasks/{taskId}`, `DELETE /api/v2/tasks/{taskId}`.
    - Project tasks: `POST /api/v2/projects/{projectId}/tasks`, `GET /api/v2/projects/{projectId}/tasks`.
    - Project tree/changes: `GET /api/v2/projects/{projectId}/tree`, `GET /api/v2/projects/{projectId}/changes` (optional `since` in RFC3339; git status-based change feed).
    - Claim/complete: `POST /api/v2/tasks/{taskId}/claim`, `POST /api/v2/tasks/{taskId}/complete` (supports `next_tasks` for child task creation).
    - Dependencies: `POST /api/v2/tasks/{taskId}/dependencies`, `GET /api/v2/tasks/{taskId}/dependencies`, `DELETE /api/v2/tasks/{taskId}/dependencies/{dependsOnTaskId}`.
    - Runs: `GET /api/v2/runs/{runId}`, `POST /api/v2/runs/{runId}/steps`, `POST /api/v2/runs/{runId}/logs`, `POST /api/v2/runs/{runId}/artifacts`.
    - Memory: `GET /api/v2/projects/{projectId}/memory`, `PUT /api/v2/projects/{projectId}/memory`.
    - Context packs: `POST /api/v2/projects/{projectId}/context-packs`.
    - Leases: `POST /api/v2/leases`, `POST /api/v2/leases/{leaseId}/heartbeat`, `POST /api/v2/leases/{leaseId}/release`.
    - Agents: `GET /api/v2/agents`, `GET /api/v2/agents/{agentId}`, `DELETE /api/v2/agents/{agentId}`.
    - Events: `GET /api/v2/events` (filters `project_id`, `type`, `since`, `task_id`, `limit`, `offset`), `GET /api/v2/events/stream` (filters `project_id`, `type`, `since`, `limit`).
    - Project events: `GET /api/v2/projects/{projectId}/events/stream`, `GET /api/v2/projects/{projectId}/events/replay`.
    - Runtime: `GET /api/v2/config`, `GET /api/v2/version`.
  - **Auth**:
    - v2 endpoints honor `X-API-Key` guardrails with scoped identities (`v2:read`, `v2:write`, `tasks:read`, `tasks:write`, `leases:write`, `agents:write`, `runs:write`, `projects:write`).
- **Tradeoffs**:
  - **Pros**:
    - Provides a clear and consistent interface for agents and users.
    - Simplifies integration with external systems.
  - **Cons**:
    - Limited flexibility for complex workflows.
- **Consequences**:
  - The API design is optimized for simplicity and ease of use.
  - Future enhancements may require extending the API.

---

### Notes
- All decisions marked as "Revisitable" may be updated if new constraints or requirements arise. Any changes must be documented in this file with proper justification.
- Locked decisions are final and cannot be changed unless explicitly revised in the KB.

### Next Steps
- Regularly review and update this document as new decisions are made or existing ones are revisited.