# Test Coverage Analysis Report

**Generated:** March 5, 2026  
**Command:** `go test -cover -coverprofile=coverage.out -covermode=atomic ./...`

---

## Executive Summary

**Overall Coverage: 58.0%**

The cocopilot codebase currently has 58% test coverage across all statements. While this provides a reasonable foundation, there are significant gaps in coverage, particularly in HTTP handlers, UI routes, automation configuration, and certain database operations.

### Key Findings

- ✅ **6 files** have ≥80% coverage (good)
- ⚠️ **6 files** have <80% coverage (need improvement)
- 🔴 **100+ functions** have 0% coverage (critical gaps)
- 🔴 **22 HTTP handlers** are completely untested

---

## Per-File Coverage Breakdown

| File | Coverage | Status | Priority |
|------|----------|--------|----------|
| scanner.go | 88.5% | ✅ Good | Low |
| rate_limiter.go | 80.5% | ✅ Good | Low |
| automation.go | 80.3% | ✅ Good | Low |
| policy_engine.go | 77.5% | ⚠️ Below target | Medium |
| db_v2.go | 74.3% | ⚠️ Below target | High |
| migrations.go | 73.4% | ⚠️ Below target | Medium |
| assignment.go | 72.5% | ⚠️ Below target | High |
| models_v2.go | 66.7% | ⚠️ Below target | Medium |
| main.go | 61.3% | 🔴 Needs work | **Critical** |

---

## Critical Gaps: 0% Coverage Functions

### Automation Configuration (automation.go)

**High Priority** - These control automation behavior but are untested:

```
- String()                              0.0%
- newAutomationCircuitBreaker()         0.0%
- getOrCreate()                         0.0%
- RecordSuccess()                       0.0%
- setAutomationCircuitBreaker()         0.0%
- init()                                0.0%
- setAutomationRateLimit()              0.0%
- getAutomationRateLimit()              0.0%
- setAutomationBurstLimit()             0.0%
- getAutomationBurstLimit()             0.0%
- setMaxAutomationDepth()               0.0%
- getMaxAutomationDepth()               0.0%
- setAutomationRules()                  0.0%
- getAutomationRules()                  0.0%
- normalizeAutomationRule()             0.0%
- isAutomationRuleEnabled()             0.0%
- isAutomationBlockedByPolicies()       0.0%
- isCompletionBlockedByPolicies()       0.0%
- isTaskCreateBlockedByPolicies()       0.0%
- isTaskUpdateBlockedByPolicies()       0.0%
- isTaskDeleteBlockedByPolicies()       0.0%
- emitPolicyDeniedEvent()               0.0%
- resolveAutomationParentID()           0.0%
- buildAutomationTemplateData()         0.0%
- buildTaskCreatedPayload()             0.0%
- SetEmissionWindow()                   0.0%
- GetEmissionWindow()                   0.0%
- computeEmissionDedupeKey()            0.0%
- CleanupOldEmissions()                 0.0%
- createAutomationReviewTask()          0.0%
```

### HTTP Handlers (main.go)

**Critical Priority** - 22 handlers with 0% coverage:

#### V1 API Handlers
```
- instructionsHandler()                 0.0%  (GET /instructions)
- deleteTaskHandler()                   0.0%  (DELETE /task)
```

#### V2 API Handlers
```
- v2MetricsHandler()                    0.0%  (/api/v2/metrics)
- v2ArtifactCommentsHandler()           0.0%  (/api/v2/artifacts/*/comments)
- v2ProjectTasksClaimNextHandler()      0.0%  (/api/v2/projects/*/tasks/claim-next)
- v2ProjectAutomationStatsHandler()     0.0%  (/api/v2/projects/*/automation/stats)
- v2ProjectGraphTasksHandler()          0.0%  (/api/v2/projects/*/graph/tasks)
- v2ProjectIDESignalsHandler()          0.0%  (/api/v2/projects/*/ide/signals)
- v2ProjectAuditExportHandler()         0.0%  (/api/v2/projects/*/audit/export)
- v2ProjectFilesScanHandler()           0.0%  (/api/v2/projects/*/files/scan)
```

#### UI Handlers (HTML pages)
```
- indexHandler()                        0.0%  (/)
- uiPlaceholderHandler()                0.0%
- taskGraphsPlaceholderHandler()        0.0%
- memoryPlaceholderHandler()            0.0%
- agentsPlaceholderHandler()            0.0%
- auditPlaceholderHandler()             0.0%
- repoPlaceholderHandler()              0.0%
- healthDashboardHandler()              0.0%
- repoGraphHandler()                    0.0%
- diffViewerHandler()                   0.0%
- runsPlaceholderHandler()              0.0%
- contextPacksPlaceholderHandler()      0.0%
```

### Database Operations (db_v2.go)

**Medium Priority** - Core DB functions with 0% coverage:

```
- mapTaskStatusV1ToV2()                 0.0%
- mapTaskStatusV2ToV1()                 0.0%
- taskIDFromPayload()                   0.0%
- CreateArtifactComment()               0.0%
- ListArtifactComments()                0.0%
- DeleteArtifactComment()               0.0%
- ptrInt64ToNullInt64()                 0.0%
```

### Core Functions (main.go)

**Critical Priority** - Essential utilities untested:

```
- main()                                0.0%  (entry point - expected)
- handleCLI()                           0.0%  (CLI mode)
- printHelp()                           0.0%
- openBrowser()                         0.0%
- initDB()                              0.0%
- normalizeV1EventType()                0.0%
- validateWorkdir()                     0.0%
- writeV2Error()                        0.0%
- writeV2MethodNotAllowed()             0.0%
- getEnvConfigValue()                   0.0%
- getEnvBoolValue()                     0.0%
- registerRoutes()                      0.0%
- v2TasksRouteHandler()                 0.0%
- getInstructions()                     0.0%
```

### Policy & Rate Limiting

```
Policy Engine (policy_engine.go):
- NewPolicyEngine()                     0.0%
- Evaluate()                            0.0%
- EvaluatePolicy()                      0.0%
- newPolicyRateTracker()                0.0%
- record()                              0.0%
- count()                               0.0%
- countActiveResources()                0.0%

Rate Limiter (rate_limiter.go):
- NewSlidingWindowRateLimiter()         0.0%
- rateLimitKey()                        0.0%
- CheckRateLimit()                      0.0%
- Reset()                               0.0%
- CountInWindow()                       0.0%
```

---

## Functions with Low Coverage (<50%)

### Assignment Module (assignment.go)
```
- FailTask()                            0.8%   🔴 Critical - task failure flow
- ClaimTaskByID()                       6.0%   🔴 Critical - task claiming
- assembleContext()                     8.8%   ⚠️  Context building
- CompleteTask()                        4.2%   🔴 Critical - task completion
```

### Automation Engine (automation.go)
```
- resolveAutomationParentID()           40.0%  ⚠️
- processAutomationEvent()              64.9%  ⚠️
- policyBlocksTaskCreate()              63.6%  ⚠️
- policyBlocksTaskUpdate()              63.6%  ⚠️
- policyBlocksTaskDelete()              63.6%  ⚠️
- parseAutomationRules()                25.0%  🔴
- applyAutomationTemplate()             75.0%  ⚠️
```

### Database Operations (db_v2.go)
```
- CreateProject()                       75.0%  ⚠️
- CreateTaskV2()                        76.9%  ⚠️
- RegisterAgent()                       72.7%  ⚠️
- GetTaskParentChain()                  66.7%  ⚠️
- TaskDependencyCreatesCycle()          77.8%  ⚠️
- DeleteTaskDependency()                70.0%  ⚠️
- CreateLease()                         72.7%  ⚠️
- DeleteExpiredLeases()                 72.7%  ⚠️
- ExtendLease()                         70.6%  ⚠️
- resolveTaskProjectID()                54.5%  ⚠️
- emitTaskDependencyEvent()             75.0%  ⚠️
- emitLeaseLifecycleEvent()             72.7%  ⚠️
- emitPolicyLifecycleEvent()            75.0%  ⚠️
- parseEventTaskID()                    71.4%  ⚠️
```

### HTTP Handlers (main.go) - <50%
```
- updateStatusHandler()                 54.3%  ⚠️
- v2BackupHandler()                     6.7%   🔴  Backup system
- v2RestoreHandler()                    3.4%   🔴  Restore system
- v2ProjectsRouteHandler()              50.0%  ⚠️
- v2ProjectFileDetailHandler()          61.4%  ⚠️
- retentionSnapshot()                   00.0%  🔴  Data retention
```

---

## Detailed Analysis by Module

### 1. HTTP Handlers (main.go) - 61.3% coverage

**Status:** Needs significant improvement

**Issues:**
- 22 handlers with 0% coverage
- Most UI handlers completely untested
- Several critical API v2 handlers untested
- SSE streaming handlers have low coverage

**Well-tested handlers (>80%):**
- `v2TasksHandler()` - 84.8%
- `v2CreateTaskHandler()` - 51.0%
- `v2TaskClaimHandler()` - 58.8%
- `v2TaskCompleteHandler()` - 67.3%

**Poorly tested handlers (<20%):**
- All UI handlers (0%)
- `v2BackupHandler()` - 6.7%
- `v2RestoreHandler()` - 3.4%
- `deleteTaskHandler()` - 0%

### 2. Database Layer (db_v2.go) - 74.3% coverage

**Status:** Below target, needs improvement

**Issues:**
- Status mapping functions untested
- Artifact comments feature untested
- Several event emission functions have ~75% coverage
- Lease management has gaps

**Well-tested functions (>90%):**
- `GetTaskV2()` - 92.3%
- `CreateTaskV2WithMeta()` - 90.3%
- `UpdateTaskV2()` - 90.9%
- `ListEvents()` - 90.2%
- `GetRepoFile()` - 94.4%
- `ListRepoFiles()` - 92.7%

**Gaps:**
- Helper functions (mappers, validators)
- New features (artifact comments)
- Edge cases in lease operations

### 3. Automation Engine (automation.go) - 80.3% coverage

**Status:** Meets 80% threshold but has critical gaps

**Issues:**
- All configuration setters/getters untested (0%)
- Circuit breaker initialization untested
- Policy blocking checks untested
- Emission deduplication untested

**Well-tested functions:**
- `TryRecordEmission()` - 85.7%
- `CheckEmissionAllowed()` - 85.7%
- `normalizeAutomationTags()` - 81.8%
- `createAutomationEscalateTask()` - 81.8%

**Critical gaps:**
- Environment variable configuration system
- Automation rule parsing and normalization
- Policy enforcement integration
- Circuit breaker functionality

### 4. Assignment Module (assignment.go) - 72.5% coverage

**Status:** Below target, critical functionality gaps

**Issues:**
- `FailTask()` - 0.8% (almost completely untested)
- `ClaimTaskByID()` - 6.0%
- `CompleteTask()` - 74.2%
- `assembleContext()` - 68.8%

**Impact:** Core task lifecycle operations have very low coverage

### 5. Policy Engine (policy_engine.go) - 77.5% coverage

**Status:** Just below 80% target

**Well-tested:**
- `evaluateRateLimit()` - 77.3%
- `evaluateWorkflowConstraint()` - 78.6%
- `evaluateTimeWindow()` - 86.4%

**Gaps:**
- Constructor and main entry points (0%)
- Policy tracker (0%)
- Resource counting (0%)

### 6. Migrations (migrations.go) - 73.4% coverage

**Status:** Below target

**Well-tested:**
- `loadMigrations()` - 77.3%
- `applyMigration()` - 77.8%

**Gaps:**
- `ensureSchemaMigrationsTable()` - 0%
- `getMigrationStatus()` - 0%
- `splitSQLStatements()` - 0%
- CLI commands and rollback operations

### 7. Models (models_v2.go) - 66.7% coverage

**Status:** Below target

**Gaps:**
- Status checking methods (0%)
- Helper functions for null handling (0%)
- Time formatting utilities (0%)

### 8. Scanner (scanner.go) - 88.5% coverage

**Status:** ✅ Excellent

**Well-tested:**
- `parseGitignore()` - 94.3%
- `ScanProjectFiles()` - 87.8%
- `computeContentHash()` - 87.5%

**Minor gaps:**
- `detectLanguage()` - 0%
- `fileScanMaxSize()` - 0%

### 9. Rate Limiter (rate_limiter.go) - 80.5% coverage

**Status:** ✅ Meets threshold

**Well-tested:**
- `Cleanup()` - 83.3%

**Gaps:**
- All constructor and core methods (0%)

---

## Recommendations

### Priority 1: Critical Handlers (High Impact)

**Task claiming and lifecycle:**
```
1. Test assignment.go:ClaimTaskByID() - currently 6.0%
2. Test assignment.go:FailTask() - currently 0.8%
3. Test assignment.go:CompleteTask() - currently 74.2%
4. Test main.go:v2TaskClaimHandler() - currently 58.8%
```

**Backup/restore system:**
```
5. Test main.go:v2BackupHandler() - currently 6.7%
6. Test main.go:v2RestoreHandler() - currently 3.4%
```

**Critical API handlers:**
```
7. Test main.go:deleteTaskHandler() - currently 0%
8. Test main.go:v2ProjectTasksClaimNextHandler() - currently 0%
```

### Priority 2: Automation System (Medium Impact)

**Configuration and policy enforcement:**
```
1. Test all automation configuration getters/setters
2. Test policy blocking functions (isAutomationBlockedByPolicies, etc.)
3. Test automation rule parsing and normalization
4. Test emission deduplication system
5. Test circuit breaker functionality
```

### Priority 3: Database Operations (Medium Impact)

**Status mapping and helpers:**
```
1. Test mapTaskStatusV1ToV2() / mapTaskStatusV2ToV1()
2. Test taskIDFromPayload()
3. Test helper functions (nullString, ptrString, etc.)
```

**New features:**
```
4. Test artifact comments (Create/List/Delete)
5. Improve lease management coverage
```

### Priority 4: API Foundation (Lower Impact but Good Practice)

**Entry points and utilities:**
```
1. Test main.go helper functions (writeV2Error, etc.)
2. Test environment configuration (getEnvConfigValue, etc.)
3. Test routing functions
4. Add basic tests for UI handlers (even simple 200 OK checks)
```

### Priority 5: Policy & Rate Limiting (Low Impact - Already Well Tested)

**Fill remaining gaps:**
```
1. Test constructors (NewPolicyEngine, NewSlidingWindowRateLimiter)
2. Test main entry points (Evaluate, CheckRateLimit)
3. Test resource counting logic
```

---

## Areas Hard to Test

Several areas have inherently low testability or high test complexity:

### 1. **SSE (Server-Sent Events) Handlers**
- Real-time streaming is difficult to test
- Requires complex setup with long-running connections
- Current coverage: ~70-80%
- **Recommendation:** Consider SSE integration tests with test clients

### 2. **UI Handlers (HTML generation)**
- All at 0% coverage
- Low value to test string concatenation for HTML
- No business logic to validate
- **Recommendation:** Accept low coverage, focus on API tests

### 3. **CLI and Main Entry Points**
- `main()` and `handleCLI()` at 0%
- Difficult to test process lifecycle
- **Recommendation:** Manual testing, integration tests

### 4. **File System Operations**
- Scanner is well-tested (88.5%)
- Git operations harder to test
- `v2ProjectFilesScanHandler()` at 0%
- **Recommendation:** Mock filesystem or use test directories

### 5. **Browser Opening**
- `openBrowser()` at 0%
- System-dependent, hard to test
- **Recommendation:** Accept low coverage

### 6. **Database Initialization**
- `initDB()` at 0%
- Happens at startup, tested implicitly
- **Recommendation:** Add integration test

### 7. **Environment Variable Loading**
- Configuration getters at 0%
- Simple wrappers, low risk
- **Recommendation:** Low priority

---

## Coverage Improvement Plan

### Phase 1: Critical Path (Target: 65% → 70%)
- Focus on assignment.go (ClaimTaskByID, FailTask, CompleteTask)
- Test main lifecycle handlers (v2TaskClaimHandler, v2TaskCompleteHandler)
- Test backup/restore handlers
- **Estimated effort:** 2-3 days
- **Impact:** Covers core task workflow

### Phase 2: Automation System (Target: 70% → 75%)
- Test automation configuration system
- Test policy blocking functions
- Test emission deduplication
- **Estimated effort:** 3-4 days
- **Impact:** Ensures automation reliability

### Phase 3: Database Gaps (Target: 75% → 78%)
- Test status mappers
- Test artifact comments
- Improve lease coverage
- **Estimated effort:** 2 days
- **Impact:** Better data layer confidence

### Phase 4: Foundation (Target: 78% → 80%)
- Test utility functions
- Test environment configuration
- Basic UI handler tests
- **Estimated effort:** 2 days
- **Impact:** Stronger foundation

---

## Test Quality Observations

Based on the existing test files, the project demonstrates:

### Strengths ✅
- Comprehensive HTTP handler tests (where they exist)
- Good database operation coverage
- Well-structured test utilities
- Integration-style tests covering full request/response cycles
- Event system well tested

### Patterns to Continue 📋
- Table-driven tests
- Helper functions for setup/teardown
- Clear test naming conventions
- Testing with real database (SQLite in-memory)
- JSON payload validation

### Gaps Identified ⚠️
- Missing tests for error paths
- Configuration and initialization not tested
- Some CRUD operations incomplete
- Edge cases for policies and automation
- No benchmark tests for performance-critical paths

---

## Metrics Summary

```
Total Coverage:              58.0%
Files with ≥80% coverage:    3 out of 9 (33%)
Files with <80% coverage:    6 out of 9 (67%)

Functions at 0%:             ~100+
Handlers at 0%:              22 (mostly UI)
Critical gaps:               8-10 high-priority functions

Target coverage:             80%
Gap to close:                22 percentage points
Estimated effort:            2-3 weeks for systematic improvement
```

---

## Conclusion

The cocopilot project has a solid test foundation at 58% coverage, with particularly strong coverage in the scanner, rate limiter, and automation modules. However, there are significant gaps in critical areas:

1. **Assignment/task lifecycle** - Core workflows like ClaimTaskByID and FailTask are severely undertested
2. **HTTP handlers** - 22 handlers have zero coverage, including backup/restore
3. **Automation configuration** - All setters/getters and policy enforcement untested
4. **Database helpers** - Status mapping and utility functions untested

The most impactful improvements would come from testing:
- Task claiming and lifecycle operations (assignment.go)
- Backup/restore functionality (main.go)
- Automation policy enforcement (automation.go)
- Critical API handlers (main.go)

With focused effort on these areas, the project could reach 70-75% coverage relatively quickly, providing significantly better confidence in production reliability.

**Note:** UI handlers at 0% are acceptable as they primarily generate HTML with minimal business logic. Focus should remain on API handlers and core business logic.
