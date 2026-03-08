# Task 33 — CreateLease Signature & V2 Task Status Constants

## 1. `CreateLease` Function (db_v2.go:1351)

### Signature

```go
func CreateLease(db *sql.DB, taskID int, agentID string, mode string) (*Lease, error)
```

### `expires_at` Logic

```go
expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(leaseTimeFormat)
```

- Defaults `mode` to `"exclusive"` if empty.
- Computes `expires_at` as **now + 15 minutes** in UTC, formatted via `leaseTimeFormat`.
- Before inserting, it deletes stale leases for the same task where `expires_at <= now`.
- Inserts the lease row into the `leases` table with the computed `expires_at`.
- Emits a `lease.created` lifecycle event; rolls back (deletes the lease) if the event emission fails.

## 2. V2 Task Status Constants (models_v2.go:37-46)

```go
type TaskStatusV2 string

const (
    TaskStatusQueued      TaskStatusV2 = "QUEUED"
    TaskStatusClaimed     TaskStatusV2 = "CLAIMED"
    TaskStatusRunning     TaskStatusV2 = "RUNNING"
    TaskStatusSucceeded   TaskStatusV2 = "SUCCEEDED"
    TaskStatusFailed      TaskStatusV2 = "FAILED"
    TaskStatusNeedsReview TaskStatusV2 = "NEEDS_REVIEW"
    TaskStatusCancelled   TaskStatusV2 = "CANCELLED"
)
```

Seven statuses covering the full task lifecycle: queued → claimed → running → succeeded/failed/needs_review/cancelled.
