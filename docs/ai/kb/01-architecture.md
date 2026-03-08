cocopilot/docs/ai/kb/01-architecture.md
# Architecture

## Overview
The system is a web-based task queue server designed to orchestrate LLM agents. It provides a Kanban-style user interface for task management and a v2 HTTP API for agents to interact with tasks. The architecture emphasizes lease-based claiming for safe multi-agent coordination, real-time event streaming with configurable retention, and optional API key guardrails with scoped identities. It is designed to ensure scalability, maintainability, and extensibility while adhering to the project's core principles of correctness, security, and performance.

---

## High-Level Architecture

The system is composed of the following key components:

1. **Web Interface**:
   - A Kanban-style UI accessible via a web browser.
   - Allows users to create, view, update, and delete tasks.
   - Supports drag-and-drop functionality for task management.
   - Provides real-time updates using Server-Sent Events (SSE).

2. **Go Server**:
   - The core backend of the system, implemented in Go.
   - Exposes RESTful API endpoints for tasks, leases, agents, runs, memory, context packs, events, project tree/changes, and config/version introspection.
   - Handles task state transitions (`NOT_PICKED`, `IN_PROGRESS`, `COMPLETE`) and v2 lease-based claiming.
   - Supports project-scoped task creation via `POST /api/v2/projects/{projectId}/tasks`.
   - Supports v2 completion with `next_tasks` to create child tasks for chaining.
   - Streams events via SSE and emits structured lifecycle events for tasks, runs, and leases.
   - Provides project-scoped event streaming and replay via `GET /api/v2/projects/{projectId}/events/stream` and `GET /api/v2/projects/{projectId}/events/replay`.
   - Provides project tree snapshots (shallow workdir listing) and a git status-based changes feed filtered by RFC3339 `since` when supplied.
   - Enforces optional v2 API key guardrails with scoped identities and standard error envelopes.
   - Auth scopes include `v2:read`, `v2:write`, `tasks:read`, `tasks:write`, `leases:write`, `agents:write`, `runs:write`, and `projects:write`.
   - Manages communication between the web interface, agents, and the database.

3. **LLM Agents**:
   - External agents that poll the server for tasks using the v1 `/task` endpoint or v2 list/claim flows.
   - Claim tasks with `/api/v2/tasks/{taskId}/claim` and complete them with `/api/v2/tasks/{taskId}/complete`.
   - Execute task instructions and submit results back to the server, optionally creating child tasks via `next_tasks` on completion.
   - Use `X-API-Key` for v2 requests when API key guardrails are enabled.

4. **SQLite Database**:
   - Stores all task-related data, including task states, instructions, and outputs.
   - Provides persistence for tasks, runs (including steps/logs/artifacts), leases, agents, memory items, context packs, and event history.
   - Supports configurable events retention pruning (by age or max rows).
   - Resides in a file named `tasks.db` in the server's working directory.

---

## Component Interactions

### 1. Web Interface ↔ Go Server
- The web interface communicates with the Go server over HTTP.
- Users interact with the Kanban UI to manage tasks, which triggers API calls to the server.
- The server sends real-time updates to the UI using SSE.
- v2 event streams support `project_id`, `type`, `since`, and `limit` filters for scoped replay and updates.

### 2. LLM Agents ↔ Go Server
- Agents poll the v1 `/task` endpoint or use v2 list/claim flows to retrieve work.
- Claims are lease-backed; agents heartbeat or release leases via v2 endpoints.
- After completing a task, agents submit results via the v1 `/save` endpoint or v2 completion endpoint, which releases any active lease.
- Agents can record run steps/logs/artifacts and store or query project memory and context packs via v2 endpoints.
- Agents can also update task statuses or delete tasks using the respective API endpoints.

### 3. Go Server ↔ SQLite Database
- The server interacts with the SQLite database to store and retrieve task, run, lease, agent, and event data.
- Events are persisted for SSE replay and can be pruned by retention settings.
- All task states and metadata are persisted in the database to ensure data integrity and recovery.

---

## Deployment Architecture

1. **Server**:
   - Runs on `http://127.0.0.1:8080` by default.
   - Can be deployed on any platform supporting Go 1.21 or later.
   - The server is designed to handle concurrent requests efficiently.
   - Exposes `GET /api/v2/version` and `GET /api/v2/config` for versioning and safe runtime configuration snapshots.

2. **Database**:
   - SQLite is used for simplicity and portability.
   - The database file (`tasks.db`) is stored locally in the server's working directory.
   - The database schema is optimized for task, lease, and event operations.

3. **Agents**:
   - Agents are external processes that interact with the server via its API.
   - They can be deployed on separate machines or containers, enabling distributed task execution.

---

## Boundaries and Constraints

1. **Scalability**:
   - The system is designed for small to medium-scale task management.
   - SQLite is used for simplicity, but it may become a bottleneck for high-concurrency scenarios.

2. **Security**:
   - The system must validate all inputs to prevent injection attacks.
   - Optional v2 API key guardrails can protect reads and writes with scoped identities.
   - Sensitive data (e.g., API keys, PII) must not be logged or exposed.

3. **Extensibility**:
   - The architecture allows for future integration with other databases or external systems.
   - The API is designed to be modular and easily extendable.

4. **Performance**:
   - The Go server is optimized for handling concurrent requests.
   - Task polling by agents is designed to minimize server load.

---

## Future Considerations

1. **Scaling the Database**:
   - Consider migrating from SQLite to a more scalable database (e.g., PostgreSQL) for larger deployments.

2. **Authentication and Authorization**:
   - Consider additional hardening (mTLS or OIDC) and richer policy tooling for deployments beyond API key guardrails.

3. **Agent Management**:
   - Add features to monitor and manage connected agents, including health checks and performance metrics.

4. **Task Prioritization**:
   - Introduce task prioritization to allow agents to pick higher-priority tasks first.

5. **Cloud Deployment**:
   - Provide deployment scripts for cloud platforms (e.g., AWS, GCP, Azure) to simplify scaling and hosting.

---

This architecture is designed to provide a robust foundation for the task queue server while allowing for future growth and enhancements.