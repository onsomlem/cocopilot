# Event Type Test Coverage Matrix

**Audit Date:** 2026-03-05  
**Status:** 7 of 20 event types have direct test coverage (35%)

## Coverage Summary

| # | Event Type | Status | Test File(s) | Test Function(s) | Notes |
|----|-----------|--------|-------------|-----------------|-------|
| 1 | task.created | ✅ | db_v2_test.go | TestCreateEvent, TestListEventsBasic | Verified in: automation_governance_test, v2_task_complete_test, v2_events_stream_test, v2_events_replay_test, v2_events_list_test |
| 2 | task.updated | ✅ | v2_events_list_test.go | TestV2EventsListSuccess | Verified in: v2_events_stream_test, v2_events_replay_test |
| 3 | task.claimed | ❌ | assignment.go:126 | *No event verification* | Event emitted in ClaimTaskByID() but no test checks for emissions; v2_task_claim_test only verifies status changes |
| 4 | task.completed | ✅ | v2_task_complete_test.go | TestV2TaskCompleteSuccess | Verified in: automation_governance_test |
| 5 | task.failed | ❌ | assignment.go:256 | *No event verification* | Event emitted in FailTask() but no test verifies emission |
| 6 | task.blocked | ❌ | db_v2.go:704 | *No event verification* | Event emitted in SetTaskBlocked() but not tested |
| 7 | run.started | ❌ | db_v2.go:1030 | *No event verification* | Event emitted in CreateRun() but not tested; runs_test.go only checks run creation in DB |
| 8 | run.completed | ⚠️ | v2_project_automation_simulate_test.go:136 | TestSimulateSuccessfulRuleExecution | Event used in automation simulation but not verified as direct emission |
| 9 | run.failed | ❌ | db_v2.go:1105 | *No event verification* | Event emitted in CompleteRunStatus() but not tested |
| 10 | lease.created | ✅ | db_v2_test.go | TestExpiredLeaseCleanup | Verified in: v2_events_list_test.go |
| 11 | lease.renewed | ❌ | main.go:7128 | *No event verification* | Event emitted in v2RenewLeaseHandler but not tested |
| 12 | lease.expired | ✅ | db_v2_test.go | TestExpiredLeaseCleanup | Event verified during lease expiration test |
| 13 | repo.changed | ❌ | main.go:5098 | *No event verification* | Event emitted in v2ProjectRepoChangesHandler but not tested |
| 14 | repo.scanned | ❌ | main.go:4979 | *No event verification* | Event emitted in v2ProjectScanHandler but not tested |
| 15 | context.refreshed | ❌ | main.go:6927 | *No event verification* | Event emitted in v2RefreshContextPackHandler but not tested |
| 16 | memory.created | ❌ | main.go:6778 | *No event verification* | Event emitted in v2ProjectMemoryPutHandler but not tested (v2_memory_put_test.go only checks PUT response) |
| 17 | memory.updated | ❌ | main.go:6780 | *No event verification* | Event emitted in v2ProjectMemoryPutHandler but not tested (v2_memory_put_test.go only checks PUT response) |
| 18 | automation.triggered | ✅ | automation_governance_test.go | TestAutomationDepthGovern (line 291), TestAutomationCircuitBreaker (line 83) | ListEvents verifies emission during automation rule execution |
| 19 | automation.blocked | ✅ | automation_governance_test.go | TestAutomationCircuitBreaker (line 113) | BlockReason: recursion_depth_exceeded, rate_limit, circuit_open |
| 20 | automation.circuit_opened | ❌ | automation.go:718 | *No event verification* | Event emitted when circuit breaker trips but not tested |
| 21 | policy.denied | ❌ | automation.go:562 | *No event verification* | Event emitted in ApplyPolicy but not tested (v2_task_complete_test has TestV2TaskCompletePolicyBlock but doesn't verify event) |
| 22 | project.idle | ❌ | main.go:3832 | *No event verification* | Event emitted in handleProjectIdlePlanner but not tested |
| 23 | project.created | ❌ | main.go:4742 | *No event verification* | Event emitted in v2ProjectCreateHandler but not tested (v2_project_task_create_test checks response, not event) |
| 24 | project.updated | ❌ | main.go:4864 | *No event verification* | Event emitted in v2ProjectUpdateHandler but not tested |
| 25 | project.deleted | ❌ | main.go:4903 | *No event verification* | Event emitted in v2ProjectDeleteHandler but not tested |
| 26 | agent.registered | ❌ | main.go:4467 | *No event verification* | Event emitted in v2RegisterAgentHandler but not tested (agents_test.go:724 only logs registration, doesn't verify event) |
| 27 | agent.deleted | ❌ | main.go:4623 | *No event verification* | Event emitted in v2DeleteAgentHandler but not tested |

## Audit Findings

### ✅ Well-Tested Events (7)
- **task.created**: Multi-file coverage across event streaming, listing, replay, and automation tests
- **task.updated**: Verified in event listing and streaming tests
- **task.completed**: Direct emission verified in completion test
- **lease.created**: Verified during lease test suite
- **lease.expired**: Verified during lease lifecycle test
- **automation.triggered**: Verified via automation governance tests
- **automation.blocked**: Verified via circuit breaker and recursion depth tests

### ⚠️ Partial Coverage (1)
- **run.completed**: Mentioned in automation simulation but not verified as direct event emission from CompleteRunStatus()

### ❌ Untested Events (19)

#### Task Lifecycle Gaps
- **task.claimed**: Emitted but assignment_test.go only checks status changes (line 37)
- **task.failed**: Emitted but no verification exists
- **task.blocked**: Emitted but no verification exists

#### Run Lifecycle Gaps
- **run.started**: Emitted on task claim but runs_test.go doesn't verify event
- **run.failed**: Emitted but not tested

#### Lease Lifecycle Gaps
- **lease.renewed**: Emitted but not tested

#### Repository/Context Gaps
- **repo.changed**: Emitted but not tested
- **repo.scanned**: Emitted but not tested
- **context.refreshed**: Emitted but not tested

#### Memory/State Gaps
- **memory.created**: Emitted but v2_memory_put_test only checks HTTP response
- **memory.updated**: Emitted but v2_memory_put_test only checks HTTP response

#### Automation/Policy Gaps
- **automation.circuit_opened**: Emitted when circuit trip but not tested
- **policy.denied**: Emitted but v2_task_complete_test policy tests don't verify event

#### Project Lifecycle Gaps
- **project.idle**: Emitted but not tested
- **project.created**: Emitted but not tested
- **project.updated**: Emitted but not tested
- **project.deleted**: Emitted but not tested

#### Agent Lifecycle Gaps
- **agent.registered**: Emitted but agents_test only logs registration
- **agent.deleted**: Emitted but not tested

## Testing Patterns

### Current Verification Methods
1. **Direct CreateEvent() + ListEvents()** - Used for task.created, task.updated, task.completed, automation.triggered, automation.blocked
2. **Database scan for event.Kind** - Used in db_v2_test.go TestExpiredLeaseCleanup
3. **HTTP handler testing without event verification** - v2_memory_put_test, v2_project_task_create_test (insufficient pattern)
4. **Automation simulation** - run.completed only via simulate endpoint

### Recommended Test Pattern for Missing Coverage
```go
// Verify event emission during operation
func TestEventEmissionForXXX(t *testing.T) {
    // Setup
    testDB, cleanup := setupV2TestDB(t)
    defer cleanup()
    
    // Operation
    // ... (perform action that emits event)
    
    // Verification
    events, _, err := ListEvents(testDB, projectID, "event.kind", "", "", 10, 0)
    if err != nil {
        t.Fatalf("ListEvents failed: %v", err)
    }
    if len(events) == 0 {
        t.Fatal("expected event.kind event to be emitted")
    }
    if events[0].Kind != "event.kind" {
        t.Fatalf("expected event.kind, got %s", events[0].Kind)
    }
}
```

## Recommendations

### High Priority Test Additions
1. **task.claimed** - Add event verification to v2_task_claim_test.go::TestV2TaskClaimSuccess
2. **memory.created/updated** - Add event verification to v2_memory_put_test.go
3. **project.created/updated/deleted** - Add event verification to v2_project_* tests
4. **agent.registered/deleted** - Add event verification to agents_test.go
5. **task.failed** - Add new test TestFailTask with event verification

### Medium Priority Test Additions
1. **run.started/completed/failed** - Add run lifecycle tests
2. **lease.renewed** - Add lease renewal test
3. **repo.changed/scanned** - Add repo event tests
4. **policy.denied** - Update policy tests to verify events

### Low Priority Test Additions
1. **context.refreshed** - Nice-to-have context pack test
2. **project.idle** - Planner scheduling feature test
3. **automation.circuit_opened** - Add circuit breaker event test

## Test Files With Event Verification Capability
- `automation_governance_test.go` - Best practices for ListEvents() verification
- `db_v2_test.go` - Database-level event scanning patterns
- `v2_events_list_test.go` - Event listing and filtering patterns
- `v2_task_complete_test.go` - Event verification during major operations

---

**Note**: This matrix covers event emission verification only. Some operations may have functional tests that verify side effects without explicitly checking event records (e.g., status changes, database updates). Events provide the audit trail and automation trigger mechanism, so explicit verification is essential.
