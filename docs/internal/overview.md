# Project Overview

**Last Updated**: 2026-02-13

## What This Project Is

Cocopilot is a web-based task orchestration system for coordinating LLM agents. It provides:

1. A Kanban-style web UI for human operators
2. REST APIs for agent workflows
3. Real-time synchronization via SSE
4. Durable storage via Go + SQLite

## Primary Goal

Evolve from a PoC task queue into a durable, context-aware project brain that supports autonomous and observable agent execution.

## Current State

- Plan completion estimate: ~60% (backend is mature; UI expansion, automation governance, and packaging are incomplete)
- Core task lifecycle: Create -> Claim -> Complete
- Parent-child context inheritance
- Project-aware task data model
- Run ledger and execution tracking with v2 runs sub-resources
- Agent registration and heartbeat
- Lease-based safe claiming with expiration cleanup and lifecycle events
- Real-time UI updates through SSE
- v2 API surface largely implemented (tasks, runs sub-resources, agents, leases, events, memory, context packs, config) with standard error envelopes
- Project tree and project changes endpoints available in v2
- Project-scoped events stream and replay endpoints available in v2
- Automation API includes rules, simulate, and replay endpoints
- Optional v2 API-key auth with scoped identities, scoped enforcement, and auth denial events
- Events retention pruning (configurable interval) and filter indexes in place
- Latest `go test -v ./...` run passes after env isolation fixes
- MCP server scaffolding for IDE/agent integration (tools/cocopilot-mcp), with tools wired for tasks/projects/events/runs/leases/agents/config/memory/policies/context packs
- VSIX extension scaffolding for VS Code integration (tools/cocopilot-vsix), with MCP config + start/stop commands and OpenAPI docs shortcuts
- MCP/VSIX release checklists available for packaging and publishing

## Key References

- [COMPLETION_SUMMARY.md](../../COMPLETION_SUMMARY.md)
- [TEST_REGRESSION.md](../../TEST_REGRESSION.md)

## Target State

- Full execution ledger and auditability
- Event-driven automation engine
- Durable memory and context packs
- Repository awareness and change tracking
- Rich mission control UI and IDE integration

## Success Metrics

- v1 endpoint stability maintained
- Agent actions traceable and reproducible
- Multi-agent coordination remains safe under contention
- Developer workflow remains simple and low-overhead
