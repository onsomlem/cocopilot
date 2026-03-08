cocopilot/docs/ai/kb/00-overview.md
# Overview

## Goal
The primary goal of this project is to develop a robust, web-based task queue server designed to orchestrate and manage tasks for LLM (Large Language Model) agents. The system provides a Kanban-style user interface for task management and a simple HTTP API for agents to interact with tasks, enabling seamless collaboration between human users and AI agents.

## Non-Goals
- The system is not intended to provide advanced AI capabilities or perform tasks beyond task orchestration and management.
- It is not designed to replace existing project management tools but to complement them by focusing on LLM agent workflows.
- The project does not aim to support databases other than SQLite.

## System Intent
The system is intended to:
1. Provide a user-friendly Kanban-style interface for task creation, management, and tracking.
2. Enable LLM agents to interact with tasks via a simple and efficient HTTP API.
3. Support real-time updates and task chaining through parent-child relationships.
4. Track execution attempts with a runs ledger and sub-resources (steps, logs, artifacts).
5. Support v2 completion with `next_tasks` for child task creation and chaining.
6. Provide project-scoped memory and context packs for durable task context.
7. Provide project-scoped events stream and replay endpoints (`GET /api/v2/projects/{projectId}/events/stream`, `GET /api/v2/projects/{projectId}/events/replay`).
8. Provide project tree snapshots and a changes feed per project for repo awareness.
9. Ensure reliability, security, and maintainability while adhering to the project's engineering standards and rules.
10. Track MCP server scaffold progress, including implemented tool coverage and supported commands.
11. Track VSIX extension integration progress, including tool coverage and command wiring.

Current completion estimate: v2.0.0 released. All core features implemented.

## Documentation Index
- [../../api/README.md](../../api/README.md) - API documentation index
- [../../state/architecture.md](../../state/architecture.md) - architecture notes and diagrams
- [../../../CHANGELOG.md](../../../CHANGELOG.md) - version history and release notes
- VSIX shortcut: `Cocopilot: Open OpenAPI Spec` (`cocopilot.openOpenApiSpec`) opens `docs/api/openapi-v2.yaml`.

This Knowledge Base (KB) serves as the authoritative source of truth for the project's goals, architecture, decisions, constraints, risks, and future directions. All changes to the project's behavior, architecture, or assumptions must be reflected in the KB to ensure alignment with the project's objectives.