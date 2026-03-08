# Troubleshooting Guide

## Common Issues

### Tasks Stuck in "claimed" State

**Symptom:** Tasks remain in "claimed" status indefinitely.

**Cause:** The agent that claimed the task crashed or disconnected without completing it.

**Fix:** Check for stuck tasks via the API:
```bash
# List claimed tasks
curl -s "http://127.0.0.1:8080/api/v2/tasks?status=CLAIMED"

# Force-unclaim a stuck task (sets it back to QUEUED)
curl -s -X PATCH "http://127.0.0.1:8080/api/v2/tasks/{id}" \
  -H "Content-Type: application/json" \
  -d '{"status": "QUEUED"}'
```

The server also runs automatic stuck-task detection. Tasks claimed for more than 30 minutes without progress are automatically requeued.

### Lease Expired

**Symptom:** Agent gets a 409 Conflict when trying to complete a task.

**Cause:** The lease expired before the agent submitted its result.

**Fix:**
1. The task is released back to the queue automatically
2. The agent should re-claim the task if it still wants to work on it
3. To prevent this, ensure your agent completes tasks within the lease duration (default: 10 minutes)

### Database Locked

**Symptom:** `database is locked` errors in server logs.

**Cause:** SQLite doesn't support high-concurrency writes. Another process may have the DB open.

**Fix:**
1. Ensure only one server instance accesses the DB file at a time
2. If the error persists, stop the server and check for stale lock files
3. If the DB is corrupted, restore from backup:
   ```bash
   # List backups
   curl -s "http://127.0.0.1:8080/api/v2/backup" -o backup.json
   
   # Restore (this replaces the current DB)
   curl -s -X POST "http://127.0.0.1:8080/api/v2/restore" \
     -H "Content-Type: application/json" \
     -d @backup.json
   ```

### Database Recovery

If the database is corrupted beyond repair:

```bash
# Stop the server
# Delete the database file
rm tasks.db

# Restart — migrations will recreate the schema
go run .
```

All data will be lost. Use the backup/restore API regularly to prevent data loss.

### SSE Connection Drops

**Symptom:** Real-time event stream disconnects frequently.

**Cause:** Reverse proxies or load balancers may timeout idle connections.

**Fix:**
- The server sends heartbeat pings every 30 seconds
- Configure your proxy to allow long-lived connections:
  ```nginx
  # nginx
  proxy_read_timeout 3600s;
  proxy_send_timeout 3600s;
  ```
- Clients should implement automatic reconnection with `Last-Event-ID` for replay

### No Tasks Available

**Symptom:** Agent polls but never receives tasks.

**Cause:** Tasks may be in a different project, or all tasks are already claimed.

**Fix:**
1. Check which projects have pending tasks:
   ```bash
   curl -s "http://127.0.0.1:8080/api/v2/tasks?status=QUEUED"
   ```
2. Ensure the agent is polling the correct project
3. Check that tasks exist and are not all completed or claimed

### API Key Authentication Errors

**Symptom:** 401 Unauthorized on all v2 mutation endpoints.

**Cause:** API key auth is enabled but the key is missing or wrong.

**Fix:**
1. Check if auth is required: `echo $COCO_REQUIRE_API_KEY`
2. Include the key in requests:
   ```bash
   curl -s -H "Authorization: Bearer YOUR_KEY" \
     "http://127.0.0.1:8080/api/v2/tasks"
   ```

### Automation Rules Not Firing

**Symptom:** `task.completed` events don't trigger automation.

**Cause:** Rules may not match the event, or rate limits may be exceeded.

**Fix:**
1. Check the automation rules:
   ```bash
   curl -s "http://127.0.0.1:8080/api/v2/projects/{id}/automation/rules"
   ```
2. Simulate a rule execution:
   ```bash
   curl -s -X POST "http://127.0.0.1:8080/api/v2/projects/{id}/automation/simulate" \
     -H "Content-Type: application/json" \
     -d '{"event_type": "task.completed", "payload": {"task_id": 1}}'
   ```
3. Check rate limit status — the server allows 100 automations/hour and 10/minute by default
