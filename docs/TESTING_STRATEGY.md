# Testing Strategy

## Overview

Cocopilot has **659 tests** across 71 test files in the `server/` package.
All tests use an in-memory SQLite database and the standard `net/http/httptest`
recorder, so they run fast (< 30 s on most hardware) with zero external
dependencies.

## Test Pyramid

```
          ┌──────────────┐
          │  Golden Path │  ← 3 tests: full HTTP lifecycle + dependency chain
          │  (E2E-like)  │
         ┌┴──────────────┴┐
         │  Integration   │  ← ~22 tests: multi-handler lifecycle flows
        ┌┴────────────────┴┐
        │   Contract /     │  ← ~46 smoke tests: every route returns expected
        │   Smoke          │    status codes and shapes
       ┌┴──────────────────┴┐
       │  Unit / Focused    │  ← ~588 tests: single function or handler
       └────────────────────┘
```

### Layer Breakdown

| Layer | Test prefix / file | Count | What it covers |
|-------|-------------------|-------|---------------|
| Golden path | `TestGoldenPath_*` | 3 | Full operator journey through HTTP API: project → task → agent → claim → run steps/logs → complete → verify events/memory/UI |
| E2E | `TestScannerE2E*` | 16 | File scanner across languages and edge cases |
| Integration | `TestIntegration*` | 8 | Multi-step lifecycle flows (create → claim → complete) |
| Integration | `TestFailTask*` | 18 | Failure/retry paths |
| Smoke - UI | `TestSmokeUI*` | 32 | Every Kanban / admin page renders without error |
| Smoke - V2 | `TestSmokeV2*` | 8 | V2 API route shape validation |
| Smoke - V1 | `TestSmokeV1*` | 6 | Legacy V1 route validation |
| Policy | `TestPolicyEvaluate*`, `TestPolicyEnforcement*`, `TestPolicyMiddleware*`, `TestDeterminePolicy*` | 36 | Policy engine: evaluation, enforcement, middleware, action determination |
| CORS | `TestCORS_*` | 3 | Preflight, normal, no-origin |
| SSE | `TestSSE*` | 7 | Server-sent events reliability |
| Unit | `TestUnit*` | 16 | Pure functions (helpers, parsers, formatters) |
| DB store | `TestCreate*`, `TestGet*`, `TestDelete*`, `TestList*` | ~30 | Individual DB operations |
| Automation | `TestAutomationGovernance*`, `TestEmissionDedupe*` | ~9 | Automation rules + deduplication |
| Leases | `TestLeaseA*`, `TestGetLease*`, `TestExtendLease*`, `TestDeleteLease*` | ~12 | Lease lifecycle |
| Runs | `TestGetRun*`, `TestCompleteRun*`, `TestDeleteRun*` | ~8 | Run CRUD |
| Agents | `TestAgentListing*`, `TestAgentDies*`, `TestAgentEnd*` | ~9 | Agent registration and status |
| Task ops | `TestClaimTask*`, `TestCompleteTask*`, `TestBulkTask*`, `TestConcurrent*` | ~14 | Claim, complete, bulk, concurrency |
| Config | `TestLoadRuntime*` | 8 | Environment / config parsing |
| Prompt runner | `TestPromptRunner*` | 8 | Planning prompt generation |
| Loop anchor | `TestLoopAnchor*` | 4 | Loop-anchor prompts |
| Templates | `TestV2Templates*` | 7 | Template CRUD, instantiation |
| Approval | `TestV2TaskApproval*` | 6 | Task approval and rejection flows |

## Test Database Setup

Three setup functions create isolated in-memory databases:

| Function | Purpose | Resets |
|----------|---------|--------|
| `setupV2TestDB(t)` | Standard test DB with migrations applied | Global `db` variable |
| `setupLifecycleTestDB(t)` | Same + clears SSE subscriber maps | Global `db` + `sseClients` + `v2EventSubscribers` |
| `setupStabilityTestDB(t)` | Same as lifecycle (used by torture tests) | Same as lifecycle |

All return `(*sql.DB, func())` — the cleanup function restores the previous `db`.

### Handler Test Pattern

```go
func TestSomething(t *testing.T) {
    _, cleanup := setupV2TestDB(t)
    defer cleanup()

    mux := http.NewServeMux()
    registerRoutes(mux, runtimeConfig{})

    // Create request
    body := `{"title":"test task","type":"BUILD"}`
    req := httptest.NewRequest("POST", "/api/v2/tasks", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rr := httptest.NewRecorder()
    mux.ServeHTTP(rr, req)

    // Assert
    if rr.Code != http.StatusCreated {
        t.Fatalf("got %d, want 201", rr.Code)
    }
}
```

## Makefile Targets

| Target | Command | Purpose |
|--------|---------|---------|
| `make test` | `go test -race -timeout 180s ./...` | All tests with race detector |
| `make test-unit` | `go test -run "TestUnit" ./server/` | Unit tests only |
| `make test-smoke` | `go test -run "TestSmoke" ./server/` | Smoke tests only |
| `make test-contract` | `go test -run "TestContract" ./server/` | API contract tests |
| `make test-integration` | `go test -run "TestIntegration" ./server/` | Integration lifecycle tests |
| `make test-e2e` | `go test -run "TestE2E\|TestScannerE2E" ./server/` | End-to-end tests |
| `make test-coverage` | Runs tests + prints coverage % | Coverage report |
| `make test-ci` | Build + test + coverage ≥ 65% gate | CI pipeline |
| `make gate` | verify-repo → verify-source → lint → build → test -race → release + verify-release | **Hard release gate** |
| `make verify-source` | Fails on .db, .zip, binaries, .DS_Store, coverage.out in tree | Source tree hygiene |
| `make bench` | `go test -bench . -benchtime 5x -timeout 60s ./server/` | Benchmarks |

## Naming Conventions

Tests follow a prefix scheme so they can be selected with `-run`:

- `TestUnit*` — pure function tests, no DB
- `TestSmoke*` — route reachability (UI pages, V1, V2)
- `TestIntegration*` — multi-step flows
- `TestGoldenPath_*` — full operator journey
- `TestScannerE2E*` — scanner end-to-end
- `TestPolicy*` — policy engine tests
- `TestCORS_*` — CORS middleware

Handler-specific tests use the entity name: `TestClaimTask*`, `TestFailTask*`,
`TestGetRun*`, `TestCreateEvent*`, etc.

## Coverage

Run `make test-coverage` to generate a report. The CI gate requires ≥ 65%
coverage. Current coverage can be checked with:

```bash
go test -coverprofile=coverage.out -timeout 180s ./server/
go tool cover -func=coverage.out | tail -1
```

## Adding New Tests

1. Choose the right prefix based on scope (Unit, Smoke, Integration, etc.).
2. Use `setupV2TestDB(t)` unless you need SSE (then use `setupLifecycleTestDB`).
3. Create a local `mux` with `registerRoutes(mux, runtimeConfig{})`.
4. Use `httptest.NewRequest` + `httptest.NewRecorder` for HTTP tests.
5. Place the test in an existing file if it fits, or create a new `*_test.go` file.
6. Run `make gate` before committing to ensure the full release gate passes.
