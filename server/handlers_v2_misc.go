package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// v2StatusHandler handles GET /api/v2/status — combined server status overview
func v2StatusHandler(cfg runtimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeV2MethodNotAllowed(w, r, http.MethodGet)
			return
		}

		uptime := time.Since(serverStartTime)
		schemaVer := getCurrentSchemaVersion()

		var projectCount, activeLeases, activeRuns int
		_ = db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectCount)
		_ = db.QueryRow("SELECT COUNT(*) FROM leases WHERE expires_at > ?", nowISO()).Scan(&activeLeases)
		_ = db.QueryRow("SELECT COUNT(*) FROM runs WHERE status IN ('running','claimed')").Scan(&activeRuns)

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"version":          "1.0.0",
			"schema_version":   schemaVer,
			"db_path":          cfg.DBPath,
			"active_projects":  projectCount,
			"active_leases":    activeLeases,
			"active_runs":      activeRuns,
			"uptime_seconds":   int(uptime.Seconds()),
			"uptime":           uptime.Truncate(time.Second).String(),
			"server_start":     serverStartTime.Format(time.RFC3339),
		})
	}
}

func v2HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	uptime := time.Since(serverStartTime)
	schemaVer := getCurrentSchemaVersion()

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"ok":              true,
		"uptime_seconds":  int(uptime.Seconds()),
		"uptime":          uptime.Truncate(time.Second).String(),
		"schema_version":  schemaVer,
		"migration_count": schemaVer,
		"server_start":    serverStartTime.Format(time.RFC3339),
	})
}

// v2MetricsHandler handles GET /api/v2/metrics
func v2MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	var taskCount, agentCount, eventCount, runCount, projectCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&agentCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&eventCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM runs").Scan(&runCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}

	var queuedCount, runningCount, succeededCount, failedCount, needsReviewCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status_v2 = ?", TaskStatusQueued).Scan(&queuedCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status_v2 IN (?, ?)", TaskStatusClaimed, TaskStatusRunning).Scan(&runningCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status_v2 = ?", TaskStatusSucceeded).Scan(&succeededCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status_v2 = ?", TaskStatusFailed).Scan(&failedCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status_v2 = ?", TaskStatusNeedsReview).Scan(&needsReviewCount); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}

	var activeLeases int
	if err := db.QueryRow("SELECT COUNT(*) FROM leases WHERE expires_at > ?", nowISO()).Scan(&activeLeases); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query metrics", nil)
		return
	}

	var circuitState string
	if automationCircuit != nil {
		if automationCircuit.AllowExecution("__probe__") {
			circuitState = "closed"
		} else {
			circuitState = "open"
		}
	} else {
		circuitState = "unknown"
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"totals": map[string]int{
			"tasks":    taskCount,
			"agents":   agentCount,
			"events":   eventCount,
			"runs":     runCount,
			"projects": projectCount,
		},
		"tasks_by_status": map[string]int{
			"queued":       queuedCount,
			"running":      runningCount,
			"succeeded":    succeededCount,
			"failed":       failedCount,
			"needs_review": needsReviewCount,
		},
		"active_leases":            activeLeases,
		"automation_circuit_state": circuitState,
		"schema_version":           getCurrentSchemaVersion(),
	})
}

// v2BackupHandler handles GET /api/v2/backup - streams SQLite database as download
func v2BackupHandler(cfg runtimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeV2MethodNotAllowed(w, r, http.MethodGet)
			return
		}

		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = "./tasks.db"
		}

		f, err := os.Open(dbPath)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "BACKUP_FAILED", "Failed to open database file", nil)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=cocopilot-backup.db")
		io.Copy(w, f)
	}
}

// v2RestoreHandler handles POST /api/v2/restore - restores database from upload
func v2RestoreHandler(cfg runtimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeV2MethodNotAllowed(w, r, http.MethodPost)
			return
		}

		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = "./tasks.db"
		}

		// Create backup of current DB first
		backupPath := dbPath + ".pre-restore-" + time.Now().UTC().Format("20060102T150405Z")
		srcFile, err := os.Open(dbPath)
		if err == nil {
			dstFile, createErr := os.Create(backupPath)
			if createErr == nil {
				io.Copy(dstFile, srcFile)
				dstFile.Close()
			}
			srcFile.Close()
		}

		// Read upload (limit to 500MB)
		body, err := io.ReadAll(io.LimitReader(r.Body, 500*1024*1024))
		if err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_BODY", "Failed to read upload body", nil)
			return
		}
		if len(body) < 16 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_BODY", "Upload too small to be a valid database", nil)
			return
		}

		// Validate SQLite header
		if string(body[:16]) != "SQLite format 3\x00" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_FORMAT", "Upload is not a valid SQLite database", nil)
			return
		}

		if err := os.WriteFile(dbPath, body, 0600); err != nil {
			writeV2Error(w, http.StatusInternalServerError, "RESTORE_FAILED", "Failed to write database file", nil)
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"restored":    true,
			"backup_path": backupPath,
			"message":     "Database restored. Restart the server to apply changes.",
		})
	}
}

// v2ArtifactCommentsHandler handles GET/POST /api/v2/artifacts/{id}/comments
func v2ArtifactCommentsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract artifact ID from path: /api/v2/artifacts/{id}/comments
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/artifacts/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "comments" || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid artifact endpoint format", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	artifactID := parts[0]

	switch r.Method {
	case http.MethodGet:
		comments, err := ListArtifactComments(db, artifactID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil)
			return
		}
		if comments == nil {
			comments = []ArtifactComment{}
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"comments": comments,
		})
	case http.MethodPost:
		var req struct {
			LineNumber int    `json:"line_number"`
			Body       string `json:"body"`
			Author     string `json:"author"`
			ProjectID  string `json:"project_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil)
			return
		}
		if req.Body == "" {
			writeV2Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "body is required", nil)
			return
		}
		if req.LineNumber < 1 {
			writeV2Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "line_number must be >= 1", nil)
			return
		}
		comment, err := CreateArtifactComment(db, artifactID, req.ProjectID, req.LineNumber, req.Body, req.Author)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "DB_ERROR", err.Error(), nil)
			return
		}
		writeV2JSON(w, http.StatusCreated, map[string]interface{}{
			"comment": comment,
		})
	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

func retentionSnapshot(cfg runtimeConfig) map[string]interface{} {
	interval := resolveEventsPruneInterval(cfg)
	return map[string]interface{}{
		"enabled":          cfg.EventsRetentionDays > 0 || cfg.EventsRetentionMax > 0,
		"interval_seconds": int(interval.Seconds()),
		"max_rows":         cfg.EventsRetentionMax,
		"days":             cfg.EventsRetentionDays,
	}
}

// v2VersionHandler handles GET /api/v2/version
func v2VersionHandler(cfg runtimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeV2MethodNotAllowed(w, r, http.MethodGet)
			return
		}

		schemaVersion := getCurrentSchemaVersion()

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"service": "cocopilot",
			"version": Version,
			"api": map[string]bool{
				"v1": true,
				"v2": true,
			},
			"schema_version": schemaVersion,
			"retention":      retentionSnapshot(cfg),
		})
	}
}

// v2ConfigHandler handles GET /api/v2/config
func v2ConfigHandler(cfg runtimeConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeV2MethodNotAllowed(w, r, http.MethodGet)
			return
		}

		heartbeatSeconds := cfg.SSEHeartbeatSeconds
		if heartbeatSeconds == 0 {
			heartbeatSeconds = defaultSSEHeartbeatSeconds
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"db_path": "[redacted]",
			"auth": map[string]interface{}{
				"required":           cfg.RequireAPIKey,
				"require_reads":      cfg.RequireAPIKeyReads,
				"identity_count":     len(cfg.AuthIdentities),
				"legacy_api_key_set": cfg.APIKey != "",
			},
			"retention": retentionSnapshot(cfg),
			"sse": map[string]interface{}{
				"heartbeat_seconds": heartbeatSeconds,
				"replay_limit_max":  resolveSSEReplayLimitMax(cfg),
			},
		})
	}
}

// v2SeedDemoHandler seeds sample data for new installations.
func v2SeedDemoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Check if data already exists — refuse to seed if tasks already exist.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count); err == nil && count > 0 {
		writeV2Error(w, http.StatusConflict, "ALREADY_SEEDED", "Database already contains tasks; demo seeding skipped.", nil)
		return
	}

	// Ensure a default project exists.
	var projID string
	err := db.QueryRow("SELECT id FROM projects LIMIT 1").Scan(&projID)
	if err != nil {
		p, cerr := CreateProject(db, "Demo Project", ".", nil)
		if cerr != nil {
			writeV2Error(w, http.StatusInternalServerError, "SEED_FAILED", cerr.Error(), nil)
			return
		}
		projID = p.ID
	}

	title := func(s string) *string { return &s }
	prio := func(n int) *int { return &n }
	typ := func(s string) *TaskType { t := TaskType(s); return &t }

	demos := []struct {
		title string
		instr string
		ttype string
		prio  int
		tags  []string
	}{
		{"Set up CI pipeline", "Configure GitHub Actions for lint, test, and build steps.", "feature", 2, []string{"devops", "ci"}},
		{"Write API documentation", "Document all v2 endpoints with request/response examples.", "documentation", 3, []string{"docs", "api"}},
		{"Fix login timeout bug", "Users report session expiry after 5 minutes instead of 30.", "bug", 1, []string{"auth", "urgent"}},
		{"Add dark mode toggle", "Implement a theme switcher in the settings page.", "feature", 4, []string{"ui", "settings"}},
		{"Review security headers", "Audit and add missing security headers (CSP, HSTS, etc).", "chore", 2, []string{"security"}},
	}

	var created []int
	for _, d := range demos {
		t, terr := CreateTaskV2WithMeta(db, d.instr, projID, nil, title(d.title), typ(d.ttype), prio(d.prio), d.tags)
		if terr != nil {
			writeV2Error(w, http.StatusInternalServerError, "SEED_FAILED", terr.Error(), nil)
			return
		}
		created = append(created, t.ID)
	}

	// Create a dependency: "Add dark mode" depends on "Fix login timeout"
	if len(created) >= 4 {
		_, _ = db.Exec("INSERT INTO task_dependencies (task_id, depends_on_task_id) VALUES (?, ?)", created[3], created[2])
	}

	// Record an event
	_, _ = CreateEvent(db, projID, "demo.seeded", "project", projID, map[string]interface{}{
		"tasks_created": len(created),
	})

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"seeded":     true,
		"project_id": projID,
		"tasks":      created,
	})
}
