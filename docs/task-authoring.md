# Task Authoring Guide

## Task Structure

Every task has these key fields:

| Field | Required | Description |
|-------|----------|-------------|
| `instructions` | Yes | What the agent should do (be specific) |
| `title` | No | Short summary for dashboard display |
| `type` | No | `ANALYZE`, `MODIFY`, `TEST`, `REVIEW`, `DOC`, `RELEASE`, `ROLLBACK` (affects completion contract) |
| `priority` | No | 0-100 (higher = claimed first, default 50) |
| `tags` | No | Labels for filtering (`["bugfix", "urgent"]`) |

## Writing Good Instructions

**Be specific and actionable:**
```
BAD:  "Fix the bug"
GOOD: "Fix the null pointer exception in UserService.getProfile() 
       when the user has no avatar set. Add a nil check before 
       accessing user.Avatar.URL. Add a unit test for this case."
```

**Include acceptance criteria:**
```
"Refactor the authentication middleware to use JWT tokens.

Acceptance criteria:
1. Replace session-based auth with JWT
2. Token expiry set to 24 hours
3. Refresh token endpoint at /api/auth/refresh
4. All existing tests pass
5. Add 3 new tests for JWT validation"
```

## Task Types

| Type | Use When | Expected Outputs |
|------|----------|-----------------|
| `MODIFY` | Changing code | `changes_made`, `files_touched` |
| `ANALYZE` | Research/investigation | `findings`, `risks`, `next_tasks` |
| `TEST` | Writing/running tests | `tests_run`, `changes_made` |
| `REVIEW` | Code review | `findings`, `risks` |
| `DOC` | Documentation updates | `changes_made`, `files_touched` |
| `RELEASE` | Release preparation | `changes_made`, `summary` |
| `ROLLBACK` | Reverting changes | `changes_made`, `files_touched` |

The completion contract (returned on claim) tells the agent exactly what fields to include in the result.

## Priority Guidelines

| Priority | When to Use |
|----------|-------------|
| 90-100 | Critical production issues |
| 70-89 | Important features, blocking work |
| 50-69 | Normal priority (default) |
| 30-49 | Nice to have, low urgency |
| 0-29 | Background tasks, cleanup |

## Using Tags

Tags help with filtering and automation:
- `["urgent"]` — high-priority items
- `["bugfix"]` — bug fixes
- `["auto"]` — created by automation rules
- `["context-refresh"]` — auto-generated context tasks

## Task Dependencies

Link related tasks:
```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/tasks/5/dependencies \
  -H "Content-Type: application/json" \
  -d '{"depends_on_task_id": 3, "dependency_type": "blocks"}'
```

Dependencies are included in the claim context so agents understand ordering.

## Task Templates

Create reusable templates for common task patterns:
```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Bug Fix",
    "instructions": "Fix the reported bug. Include: root cause, fix, tests.",
    "default_type": "MODIFY",
    "default_priority": 70,
    "default_tags": ["bugfix"]
  }'
```

Instantiate from a template:
```bash
curl -s -X POST http://127.0.0.1:8080/api/v2/projects/proj_default/templates/{id}/instantiate \
  -H "Content-Type: application/json" \
  -d '{"title": "Fix login timeout"}'
```
