# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [2.0.0] - 2026-03-08

### Added
- **API v2**: Full JSON API with structured error responses, projects, tasks, runs, leases, agents, events, memory, policies, context packs, and automation.
- **Lease-based task claiming**: Prevents double-claiming with automatic expiry.
- **Automation rules engine**: Event-driven task creation with rate limiting, circuit breaker, and depth limits.
- **Policy enforcement**: Per-project policies that restrict agent actions.
- **Memory system**: Persistent key-value memory scoped to projects for learning across tasks.
- **Context packs**: Bundled context (files, instructions) attached to tasks for agent consumption.
- **Run summaries**: Structured completion data extracted from task outputs and persisted to `run_summaries` table.
- **SSE streaming (v2)**: Real-time event streaming with heartbeat, replay, and filtering.
- **CORS middleware**: `withCORS()` echoes Origin, handles preflight OPTIONS, sets standard headers.
- **Request logging middleware**: `withRequestLog()` logs method+path for non-static requests.
- **Agent registration**: Named agent tracking with heartbeat and capabilities.
- **Task dependencies**: Block tasks until their dependencies reach terminal state.
- **Backup/restore**: Download and upload consistent SQLite snapshots via API.
- **Health dashboard**: `/health` endpoint with system metrics and diagnostics.
- **Benchmarks**: Load tests for task creation, claim throughput, concurrent contention, and list performance.
- **Security documentation**: Threat model, security guide, and vulnerability reporting policy.
- **Release automation**: GitHub Actions workflow for tag-triggered cross-platform builds with checksums.

### Changed
- **Package structure**: Refactored from `package main` to `package server` with thin `cmd/cocopilot/main.go` entry point.
- **Internal packages**: Extracted 10 `internal/` packages (models, dbstore, config, httputil, migrate, policy, ratelimit, scanner, worker, notifications) for cleaner separation of concerns.
- **Default bind address**: Changed from `0.0.0.0:8080` to `127.0.0.1:8080` (localhost only) for security.
- **Build path**: `go build ./cmd/cocopilot` (was `go build .`).

### Fixed
- Dashboard metric queries use canonical `TaskStatusV2` constants.
- Workdir validation blocks path traversal and dangerous directories.
- Secret scanning prevents committing private keys.

## [1.0.0] - 2026-02-15

### Added
- Initial v1 API: form-encoded task queue with poll/create/save/delete.
- Kanban-style web UI for task management.
- SQLite database with auto-applying migrations.
- SSE streaming for real-time UI updates.
- MCP server for VS Code integration (`tools/cocopilot-mcp`).
- VS Code extension scaffold (`tools/cocopilot-vsix`).
