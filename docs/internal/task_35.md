# Task 35 — CreateLeaseTx & Claim-Next Transaction Flow

## 1. `CreateLeaseTx` from `db_v2.go` (line 1354)

```go
func CreateLeaseTx(tx *sql.Tx, taskID int, agentID string, mode string) (*Lease, error) {
	if mode == "" {
		mode = "exclusive"
	}

	expiresAt := time.Now().UTC().Add(15 * time.Minute).Format(leaseTimeFormat)
	now := nowISO()

	lease := &Lease{
		ID:        "lease_" + uuid.New().String(),
		TaskID:    taskID,
		AgentID:   agentID,
		Mode:      mode,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	// Remove stale lease rows for this task so they do not block reclaims.
	if _, err := tx.Exec("DELETE FROM leases WHERE task_id = ? AND expires_at <= ?", taskID, now); err != nil {
		return nil, fmt.Errorf("failed to clean stale leases: %w", err)
	}

	_, err := tx.Exec(`
		INSERT INTO leases (id, task_id, agent_id, mode, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, lease.ID, lease.TaskID, lease.AgentID, lease.Mode, lease.CreatedAt, lease.ExpiresAt)

	if err != nil {
		if isLeaseConflictError(err) {
			return nil, ErrLeaseConflict
		}
		return nil, fmt.Errorf("failed to create lease: %w", err)
	}

	// Emit lease.created event inside the same transaction.
	if err := emitLeaseLifecycleEventTx(tx, "lease.created", lease, nil); err != nil {
		return nil, err
	}

	return lease, nil
}
```

## 2. Core claim transaction in `v2ProjectTasksClaimNextHandler` from `main.go` (lines 3579–3721)

This is the `db.Begin()` → `tx.Commit()` section inside the retry loop:

```go
		// === Atomic claim transaction ===
		tx, txErr := db.Begin()
		if txErr != nil {
			if isSQLiteBusyError(txErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to begin transaction", nil)
			return
		}

		// Step 1: SELECT candidate task inside tx
		now := nowISO()
		taskID, found, qErr := claimNextTaskTx(tx, projectID, now)
		if qErr != nil {
			_ = tx.Rollback()
			if isSQLiteBusyError(qErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query tasks", map[string]interface{}{
				"project_id": projectID,
			})
			return
		}

		if !found {
			_ = tx.Rollback()

			// No task available — attempt idle planner spawn (once)
			if idlePlannerAttempted {
				if idlePlannerSpawned {
					// Spawned idle planner but couldn't claim it — 202
					writeV2JSON(w, http.StatusAccepted, map[string]interface{}{
						"message": "Idle planner task spawned but could not be claimed in this call",
					})
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			idlePlannerAttempted = true

			// Policy gate
			blocked, reason, pErr := isAutomationBlockedByPolicies(db, projectID)
			if pErr != nil {
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Policy check failed", map[string]interface{}{
					"project_id": projectID,
					"error":      pErr.Error(),
				})
				return
			}
			if blocked {
				log.Printf("idle-planner-v2: blocked by policy: %s", reason)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			spawnTx, spawnTxErr := db.Begin()
			if spawnTxErr != nil {
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to begin transaction", nil)
				return
			}
			newTaskID, created, spawnErr := spawnIdlePlannerTx(spawnTx, projectID)
			if spawnErr != nil {
				_ = spawnTx.Rollback()
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Idle planner spawn failed", map[string]interface{}{
					"project_id": projectID,
					"error":      spawnErr.Error(),
				})
				return
			}
			if !created {
				_ = spawnTx.Rollback()
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if cmErr := spawnTx.Commit(); cmErr != nil {
				_ = spawnTx.Rollback()
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to commit idle planner", nil)
				return
			}
			idlePlannerSpawned = true
			log.Printf("idle-planner-v2: created task %d in %s", newTaskID, projectID)
			continue // retry claim to pick up the newly spawned task
		}

		// Step 2: CREATE lease inside tx using tx-aware helper
		lease, leaseErr := CreateLeaseTx(tx, taskID, req.AgentID, req.Mode)
		if leaseErr != nil {
			_ = tx.Rollback()
			if errors.Is(leaseErr, ErrLeaseConflict) || isLeaseConflictError(leaseErr) || isSQLiteBusyError(leaseErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to create lease", map[string]interface{}{
				"task_id":  taskID,
				"agent_id": req.AgentID,
			})
			return
		}

		// Step 3: UPDATE task status to CLAIMED inside tx.
		// claim-next is assignment; the agent should transition to RUNNING
		// when it actually begins work on the task.
		updatedAt := nowISO()
		_, statusErr := tx.Exec(
			"UPDATE tasks SET status = ?, status_v2 = ?, updated_at = ? WHERE id = ?",
			StatusInProgress,
			TaskStatusClaimed,
			updatedAt,
			taskID,
		)
		if statusErr != nil {
			_ = tx.Rollback()
			if isSQLiteBusyError(statusErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to update task status", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}

		// Step 4: CREATE run record inside tx
		runID := "run_" + uuid.New().String()
		runStartedAt := nowISO()
		_, runErr := tx.Exec(`
			INSERT INTO runs (id, task_id, agent_id, status, started_at)
			VALUES (?, ?, ?, ?, ?)
		`, runID, taskID, req.AgentID, RunStatusRunning, runStartedAt)
		if runErr != nil {
			_ = tx.Rollback()
			if isSQLiteBusyError(runErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to create run", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}

		// Step 5: COMMIT — if fails, rollback and retry
		if cmErr := tx.Commit(); cmErr != nil {
			_ = tx.Rollback()
			if isSQLiteBusyError(cmErr) {
				continue
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to commit claim", map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
```
