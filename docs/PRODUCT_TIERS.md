# Product Tiers

This document defines the supported product boundary. Features in Tier 1 must
work every time. Tier 2 features are supported but receive less testing scrutiny.

## Tier 1 — Must Work Every Time

These paths are covered by the golden-path test and `make gate`. A regression
here is a ship blocker.

| Feature | Test coverage |
|---------|--------------|
| Single-binary local server | `go build`, golden path |
| Dashboard / Kanban board | `TestSmokeUI*` (32 tests) |
| Projects and tasks CRUD | `TestV2Task*`, `TestGoldenPath*` |
| Agent registration and claiming | `TestClaimTask*`, `TestAgentListing*` |
| Runs, steps, logs, artifacts | `TestGetRun*`, golden path |
| Events and SSE streaming | `TestSSE*`, `TestEventVerify*` |
| Memory read/write | `TestV2Memory*`, golden path |
| Context packs | `TestV2ContextPack*` |
| Task dependencies | `TestV2TaskDependencies*`, golden path |
| Release packaging | `make verify-release`, `make verify-source` |
| Policy engine | `TestPolicy*` (36 tests) |

## Tier 2 — Supported but Secondary

These features work but have lighter test coverage. Regressions here are
polish issues, not ship blockers.

| Feature | Notes |
|---------|-------|
| Docker deployment | `Dockerfile` exists, manual testing |
| MCP integration | `tools/cocopilot-mcp/`, separate Node.js package |
| VS Code extension | `tools/cocopilot-vsix/`, scaffold stage |
| Advanced automation rules | `TestAutomationGovernance*` covers basics |
| Templates | `TestV2Templates*` (7 tests) |
| Task approval workflows | `TestV2TaskApproval*` (6 tests) |
| Repo file scanning | `TestScannerE2E*` (16 tests) |
| V1 legacy API | `TestSmokeV1*` (6 tests), maintenance mode |

## How to Use This

- **Before shipping**: All Tier 1 features must pass `make gate`.
- **Tier 2 regressions**: Fix in the next cycle, not a release blocker.
- **Promoting to Tier 1**: Add golden-path coverage and move the entry up.
- **New features**: Start in Tier 2 until they have full test coverage.
