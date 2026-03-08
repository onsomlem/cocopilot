package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// Project Dashboard Handler
// ============================================================================

// v2ProjectDashboardHandler handles GET /api/v2/projects/{id}/dashboard
// Returns a comprehensive project overview: queue stats, active runs,
// active agents, recent repo changes, recent failures, automation actions.
func v2ProjectDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Project ID required", nil)
		return
	}
	projectID := parts[0]

	project, err := GetProject(db, projectID)
	if err != nil || project == nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Project not found", nil)
		return
	}

	// Core projection: efficient read model via db_v2 helper.
	dash, err := GetProjectDashboardData(db, projectID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	// Active agents (agents with recent heartbeat in last 10 min).
	type agentInfo struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		Status        string  `json:"status"`
		LastHeartbeat *string `json:"last_heartbeat,omitempty"`
	}
	var activeAgents []agentInfo
	tenMinAgo := time.Now().UTC().Add(-10 * time.Minute).Format("2006-01-02T15:04:05.000000Z")
	agentRows, err := db.Query(`
		SELECT id, name, status, last_heartbeat_at
		FROM agents WHERE project_id = ? AND (last_heartbeat_at >= ? OR status = 'ONLINE')
		ORDER BY last_heartbeat_at DESC LIMIT 10
	`, projectID, tenMinAgo)
	if err == nil {
		defer agentRows.Close()
		for agentRows.Next() {
			var ai agentInfo
			var lastHB sql.NullString
			if err := agentRows.Scan(&ai.ID, &ai.Name, &ai.Status, &lastHB); err == nil {
				if lastHB.Valid {
					ai.LastHeartbeat = &lastHB.String
				}
				activeAgents = append(activeAgents, ai)
			}
		}
	}
	if activeAgents == nil {
		activeAgents = []agentInfo{}
	}

	// Recommendations come from the dashboard projection (scored by multiple signals).
	resp := map[string]interface{}{
		"project_id": projectID,
		"queue": map[string]interface{}{
			"pending":     dash.TaskCounts.Queued,
			"in_progress": dash.TaskCounts.InProgress,
			"failed":      dash.TaskCounts.Failed,
		},
		"task_counts":         dash.TaskCounts,
		"active_runs":         dash.ActiveRuns,
		"active_agents":       activeAgents,
		"recent_repo_changes": dash.RecentChanges,
		"recent_failures":     dash.RecentFailures,
		"automation_actions":  dash.AutomationEvents,
		"recommended_next":    dash.Recommendations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// Repo Files Handlers
// ============================================================================

// v2ProjectFilesHandler handles GET /api/v2/projects/:id/files
func v2ProjectFilesHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "files" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project files path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	projectID := parts[0]

	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Verify project exists
	if _, err := GetProject(db, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	query := r.URL.Query()
	opts := ListRepoFilesOpts{
		Limit:  100,
		Offset: 0,
	}

	if lang := strings.TrimSpace(query.Get("language")); lang != "" {
		opts.Language = &lang
	}
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
				"limit": rawLimit,
			})
			return
		}
		if parsed > 1000 {
			parsed = 1000
		}
		opts.Limit = parsed
	}
	if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil || parsed < 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "offset must be a non-negative integer", map[string]interface{}{
				"offset": rawOffset,
			})
			return
		}
		opts.Offset = parsed
	}

	files, total, err := ListRepoFiles(db, projectID, opts)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"files":  files,
		"total":  total,
		"limit":  opts.Limit,
		"offset": opts.Offset,
	})
}

// v2ProjectFileDetailHandler handles GET/PUT/DELETE /api/v2/projects/:id/files/{path}
func v2ProjectFileDetailHandler(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	// Find the /files/ separator
	filesIdx := strings.Index(trimmed, "/files/")
	if filesIdx <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project file path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	projectID := trimmed[:filesIdx]
	filePath := trimmed[filesIdx+len("/files/"):]
	if filePath == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "File path is required", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	// Verify project exists
	if _, err := GetProject(db, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		file, err := GetRepoFile(db, projectID, filePath)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "File not found", map[string]interface{}{
					"project_id": projectID,
					"path":       filePath,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
				"path":       filePath,
			})
			return
		}
		writeV2JSON(w, http.StatusOK, file)

	case http.MethodPut:
		var req struct {
			ContentHash  *string                `json:"content_hash"`
			SizeBytes    *int64                 `json:"size_bytes"`
			Language     *string                `json:"language"`
			LastModified *string                `json:"last_modified"`
			Metadata     map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
				"content_type": r.Header.Get("Content-Type"),
			})
			return
		}

		file := RepoFile{
			ProjectID:    projectID,
			Path:         filePath,
			ContentHash:  req.ContentHash,
			SizeBytes:    req.SizeBytes,
			Language:     req.Language,
			LastModified: req.LastModified,
			Metadata:     req.Metadata,
		}

		result, err := UpsertRepoFile(db, file)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
				"path":       filePath,
			})
			return
		}
		writeV2JSON(w, http.StatusOK, result)

	case http.MethodDelete:
		err := DeleteRepoFile(db, projectID, filePath)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "File not found", map[string]interface{}{
					"project_id": projectID,
					"path":       filePath,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
				"path":       filePath,
			})
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodDelete)
	}
}

// v2ProjectNotificationsHandler handles GET /api/v2/projects/:id/notifications
// Returns recent notification-worthy events for the project.
func v2ProjectNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Project ID required", nil)
		return
	}
	projectID := parts[0]

	project, err := GetProject(db, projectID)
	if err != nil || project == nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Project not found", nil)
		return
	}

	// Collect notification-worthy events: task.failed, task.stalled, run.failed,
	// policy.denied, lease.expired, project.idle
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	kind := r.URL.Query().Get("kind")
	since := r.URL.Query().Get("since")

	var events []Event
	if kind != "" {
		events, _, err = ListEvents(db, projectID, kind, since, "", limit, 0)
	} else {
		// Query all notification-worthy kinds
		var allEvents []Event
		for nk := range notifiableEvents {
			evts, _, qErr := ListEvents(db, projectID, nk, since, "", limit, 0)
			if qErr == nil {
				allEvents = append(allEvents, evts...)
			}
		}
		// Sort by created_at descending and cap
		sort.SliceStable(allEvents, func(i, j int) bool {
			return allEvents[i].CreatedAt > allEvents[j].CreatedAt
		})
		if len(allEvents) > limit {
			allEvents = allEvents[:limit]
		}
		events = allEvents
	}
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to list events", nil)
		return
	}
	if events == nil {
		events = []Event{}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  len(events),
	})
}

// v2ProjectFilesScanHandler handles POST /api/v2/projects/:id/files/scan
// Scans the project's workdir and upserts discovered files.
func v2ProjectFilesScanHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "files" || parts[2] != "scan" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project files scan path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	projectID := parts[0]

	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	proj, err := GetProject(db, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	if proj.Workdir == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Project has no workdir configured", map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	var req struct {
		Purge bool `json:"purge"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	scannedFiles, err := ScanProjectFiles(projectID, proj.Workdir)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", fmt.Sprintf("scan failed: %s", err.Error()), map[string]interface{}{
			"project_id": projectID,
			"workdir":    proj.Workdir,
		})
		return
	}

	synced := 0
	for _, f := range scannedFiles {
		if _, err := UpsertRepoFile(db, f); err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", fmt.Sprintf("failed to upsert file %s: %s", f.Path, err.Error()), map[string]interface{}{
				"project_id": projectID,
				"path":       f.Path,
			})
			return
		}
		synced++
	}

	deleted := 0
	if req.Purge {
		scannedPaths := make(map[string]bool, len(scannedFiles))
		for _, f := range scannedFiles {
			scannedPaths[f.Path] = true
		}
		existing, _, err := ListRepoFiles(db, projectID, ListRepoFilesOpts{Limit: 0})
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		for _, ef := range existing {
			if !scannedPaths[ef.Path] {
				if err := DeleteRepoFile(db, projectID, ef.Path); err != nil {
					writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
					return
				}
				deleted++
			}
		}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"scanned": len(scannedFiles),
		"synced":  synced,
		"deleted": deleted,
	})
}

// v2ProjectFilesSyncHandler handles POST /api/v2/projects/:id/files/sync
func v2ProjectFilesSyncHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	// Expected: {projectID}/files/sync
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "files" || parts[2] != "sync" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project files sync path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	projectID := parts[0]

	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Verify project exists
	if _, err := GetProject(db, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	var req struct {
		Files []struct {
			Path         string                 `json:"path"`
			ContentHash  *string                `json:"content_hash"`
			SizeBytes    *int64                 `json:"size_bytes"`
			Language     *string                `json:"language"`
			LastModified *string                `json:"last_modified"`
			Metadata     map[string]interface{} `json:"metadata"`
		} `json:"files"`
		Purge bool `json:"purge"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if req.Files == nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "files array is required", nil)
		return
	}

	// Upsert all files
	synced := 0
	for _, f := range req.Files {
		if f.Path == "" {
			continue
		}
		file := RepoFile{
			ProjectID:    projectID,
			Path:         f.Path,
			ContentHash:  f.ContentHash,
			SizeBytes:    f.SizeBytes,
			Language:     f.Language,
			LastModified: f.LastModified,
			Metadata:     f.Metadata,
		}
		if _, err := UpsertRepoFile(db, file); err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", fmt.Sprintf("failed to upsert file %s: %s", f.Path, err.Error()), map[string]interface{}{
				"project_id": projectID,
				"path":       f.Path,
			})
			return
		}
		synced++
	}

	deleted := 0
	if req.Purge {
		// Build set of synced paths
		syncedPaths := make(map[string]bool, len(req.Files))
		for _, f := range req.Files {
			if f.Path != "" {
				syncedPaths[f.Path] = true
			}
		}

		// List all existing files for the project
		existing, _, err := ListRepoFiles(db, projectID, ListRepoFilesOpts{Limit: 0})
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}

		// Delete files not in the sync list
		for _, ef := range existing {
			if !syncedPaths[ef.Path] {
				if err := DeleteRepoFile(db, projectID, ef.Path); err != nil {
					writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
						"project_id": projectID,
						"path":       ef.Path,
					})
					return
				}
				deleted++
			}
		}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"synced":  synced,
		"deleted": deleted,
	})
}
