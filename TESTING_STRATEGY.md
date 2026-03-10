# Testing Strategy

## Current State (as of this assessment)

- **Server package**: 67.9% coverage across 63 test files
- **Internal packages**: 0% coverage (except `ratelimit` at 83.3%) — tested indirectly through server tests via thin wrappers
- **Total test files**: 63

## Coverage Breakdown by Risk Level

### ✅ Well Tested (70-100% coverage)
| Area | Coverage | Files |
|------|----------|-------|
| Auth middleware | 100% | `auth.go` |
| SSE events | 100% | `sse.go` |
| Scanner | 100% | `scanner.go` + `scanner_e2e_test.go` |
| Routes | ~100% | `routes.go` |
| v2 Task CRUD | ~95% | `handlers_v2_tasks.go` |
| v2 Events | ~100% | `handlers_v2_events.go` |
| Automation engine | ~92% | `automation.go` |
| Policy engine | 100% | `policy_engine.go` |
| DB operations | ~84% | `db_v2.go` (83/99 functions) |
| Migrations | ~88% | `migrations.go` |
| UI pages | ~84% | `ui_pages.go`, `ui_management.go` |
| Rate limiter | 83.3% | `internal/ratelimit` |

### ⚠️ Partially Tested (30-70% coverage)
| Area | Coverage | Gap |
|------|----------|-----|
| v2 Projects | ~85% | Audit, IDE signals, graph tasks, automation stats |
| Finalization | ~83% | `Complete()` function |
| Export/Import | - | Needs verification |
| v1 Handlers | ~90% | `saveHandler` partial |

### ❌ Untested (0% coverage) — Needs Tests
| Area | Functions | Priority |
|------|-----------|----------|
| **Task Templates** | 3 handlers | HIGH — new feature, user-facing |
| **Task Approval** | 2 handlers | HIGH — workflow gate |
| **Notifications** | StallDetection, IdleDetection | MEDIUM — background jobs |
| **Models status checks** | 16 utility functions | MEDIUM — state logic |
| **Config env parsing** | 4 functions | LOW — startup only |
| **CLI** | 2 functions | LOW — hard to test |
| **Worker** | 2 functions | LOW — process lifecycle |

## Testing Layers

### 1. Unit Tests (fastest, most granular)
- **Purpose**: Test pure functions and simple logic in isolation
- **Targets**: Status check functions, JSON helpers, language detection, config parsing
- **Pattern**: `TestFunctionName_Scenario`
- **Run**: `go test -run "TestUnit" ./server/`

### 2. Handler Tests (HTTP contract tests)
- **Purpose**: Test API endpoints with `httptest.NewRequest` / `httptest.NewRecorder`
- **Targets**: All v2 handlers, v1 handlers
- **Pattern**: `Test<Endpoint>_<Scenario>` (e.g., `TestTemplatesCRUD`, `TestApprovalFlow`)
- **Setup**: `setupTestDB(t)` + create required entities
- **Run**: `go test -run "TestV2|TestHandler" ./server/`

### 3. Integration Tests (cross-cutting flows)
- **Purpose**: Test multi-step workflows across handlers
- **Targets**: Task lifecycle (create → claim → run → complete), automation chains, dependency resolution
- **Pattern**: `TestIntegration_<Workflow>`
- **Run**: `go test -run "TestIntegration" ./server/`

### 4. E2E Tests (full system)
- **Purpose**: Test filesystem interactions, scanning, real data flows
- **Targets**: Scanner, file sync, backup/restore
- **Pattern**: `TestE2E_<Feature>`
- **Run**: `go test -run "TestE2E|TestScannerE2E" ./server/`

### 5. Stress / Load Tests
- **Purpose**: Catch race conditions, deadlocks, OOM under load
- **Targets**: Concurrent claims, SSE fan-out, DB contention
- **Pattern**: `Benchmark*` or `TestStability_*`
- **Run**: `go test -race -run "TestStability" ./server/` or `go test -bench . ./server/`

## CI Pipeline Recommendations

```bash
# Stage 1: Build check (fast)
go build -o /dev/null ./cmd/cocopilot

# Stage 2: Unit + handler tests (< 30s)
go test -race -timeout 60s ./server/ ./internal/...

# Stage 3: Coverage gate (block merge if coverage drops)
go test -coverprofile=coverage.out ./server/
go tool cover -func=coverage.out | grep total | awk '{print $3}'
# Fail if < 65%

# Stage 4: Benchmarks (optional, nightly)
go test -bench . -benchtime 5x -timeout 60s ./server/
```

## Regression Test Protocol

When fixing a bug:
1. Write a failing test that reproduces the bug FIRST
2. Fix the bug
3. Verify the test passes
4. The test stays forever as a regression guard

## Test Naming Conventions

```
TestV2Templates_CreateAndList       # Handler test for templates
TestV2TaskApproval_ApproveReject    # Approval workflow
TestUnit_IsTaskTerminal             # Unit test for status check
TestIntegration_TaskLifecycle       # Full lifecycle flow
TestScannerE2E_GitignoreFiltering   # E2E scanner test
```

## Immediate Action Items

1. ✅ Scanner E2E tests (16 tests) — DONE
2. 🔲 Template handler tests (CRUD + instantiate)
3. 🔲 Approval handler tests (approve + reject flows)
4. 🔲 Model status function unit tests
5. 🔲 Project analytics endpoint tests (audit, graph, IDE signals)
6. 🔲 Notification background job tests
7. 🔲 Makefile/CI script for test automation
