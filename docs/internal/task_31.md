# Task 31 — claim-next dispatch & handler skeleton

## 1. Dispatch block in `v2ProjectRouteHandler` (main.go ~L9070)

```go
		// claim-next must be checked before /tasks suffix match
		if strings.HasSuffix(r.URL.Path, "/tasks/claim-next") {
			v2ProjectTasksClaimNextHandler(w, r)
			return
		}
```

## 2. `v2ProjectTasksClaimNextHandler` function (main.go L3514–L3700)

```go
// v2ProjectTasksClaimNextHandler handles POST /api/v2/projects/:id/tasks/claim-next
// It finds the next available task in the project, claims it, and returns it.
// If no task is available, it attempts idle planner emission + creation + one retry.
// Otherwise returns 204 No Content.
func v2ProjectTasksClaimNextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Extract project ID from URL path (/api/v2/projects/:id/tasks/claim-next)
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project ID", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	projectID := parts[0]

	var req struct {
		AgentID string `json:"agent_id"`
		Mode    string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
	if strings.TrimSpace(req.AgentID) == "" { ... }
	if req.Mode == "" { req.Mode = "exclusive" }

	idlePlannerAttempted := false
	const maxAttempts = 20
	for attempt := 0; attempt < maxAttempts; attempt++ {
		now := nowISO()

		// 1. Find next available task (read-only tx via claimNextTaskTx)
		// 2. If !found → idle planner spawn (once), then retry or 204
		// 3. Claim: CreateLease → UPDATE tasks SET status → CreateRun
		// 4. broadcastUpdate, fetch task, return JSON {lease, task}
	}

	// Exhausted attempts → 409 CONFLICT
	writeV2Error(w, http.StatusConflict, "CONFLICT", "Could not claim a task after multiple attempts", ...)
}
```

### Main flow summary

1. **Method guard** — only POST allowed
2. **Extract project ID** from URL path
3. **Parse JSON body** — requires `agent_id`, defaults `mode` to `"exclusive"`
4. **Retry loop** (up to 20 attempts, handles SQLite busy / lease conflicts):
   - Query next claimable task via `claimNextTaskTx`
   - If none found → try idle planner spawn once, then 204
   - `CreateLease` → update task status to `claimed` → `CreateRun`
   - Broadcast SSE update, return `{lease, task}` as JSON 200
5. **Fallback** — 409 Conflict after exhausting retries
