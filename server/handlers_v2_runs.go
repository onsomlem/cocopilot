package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// ============================================================================
// API v2 Run Handlers
// ============================================================================

// v2ListRunsHandler handles GET /api/v2/runs — lists recent runs.
func v2ListRunsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	runs, err := ListRecentRuns(db, limit)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}
	if runs == nil {
		runs = []Run{}
	}
	writeV2JSON(w, http.StatusOK, map[string]interface{}{"runs": runs, "total": len(runs)})
}

func v2RunsRouteHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/runs/")
	if path == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid run endpoint", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	parts := strings.Split(path, "/")
	runID := strings.TrimSpace(parts[0])
	if runID == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid run ID", map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	if len(parts) == 1 || (len(parts) == 2 && parts[1] == "") {
		if r.Method != http.MethodGet {
			writeV2MethodNotAllowed(w, r, http.MethodGet)
			return
		}
		v2GetRunHandler(w, r)
		return
	}

	if len(parts) == 2 && parts[1] == "steps" {
		v2CreateRunStepHandler(w, r, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		v2CreateRunLogHandler(w, r, runID)
		return
	}

	if len(parts) == 2 && parts[1] == "artifacts" {
		v2CreateRunArtifactHandler(w, r, runID)
		return
	}

	writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Not found", map[string]interface{}{
		"path": r.URL.Path,
	})
}

func v2GetRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Extract run ID from URL path (/api/v2/runs/:id)
	runID := strings.TrimPrefix(r.URL.Path, "/api/v2/runs/")
	if runID == "" || strings.Contains(runID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid run ID", map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	run, err := GetRun(db, runID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{"run_id": runID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"run_id": runID})
		return
	}

	steps, err := GetRunSteps(db, runID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"run_id": runID})
		return
	}

	logs, err := GetRunLogs(db, runID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"run_id": runID})
		return
	}

	artifacts, err := GetArtifactsByRunID(db, runID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"run_id": runID})
		return
	}

	toolInvocations, err := GetToolInvocationsByRunID(db, runID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"run_id": runID})
		return
	}

	detail := RunDetail{
		Run:             *run,
		Steps:           steps,
		Logs:            logs,
		Artifacts:       artifacts,
		ToolInvocations: toolInvocations,
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"run": detail,
	})
}

func v2CreateRunStepHandler(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if strings.TrimSpace(runID) == "" || strings.Contains(runID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid run ID", map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	var req struct {
		Name    string                 `json:"name"`
		Status  string                 `json:"status"`
		Details map[string]interface{} `json:"details"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "name is required", map[string]interface{}{
			"name": req.Name,
		})
		return
	}

	statusRaw := strings.TrimSpace(req.Status)
	if statusRaw == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "status is required", map[string]interface{}{
			"status": req.Status,
		})
		return
	}

	var status StepStatus
	switch statusRaw {
	case string(StepStatusStarted), string(StepStatusSucceeded), string(StepStatusFailed):
		status = StepStatus(statusRaw)
	default:
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid step status", map[string]interface{}{
			"status": statusRaw,
		})
		return
	}

	if _, err := GetRun(db, runID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"run_id": runID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	step, err := CreateRunStep(db, runID, name, status, req.Details)
	if err != nil {
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry run step", map[string]interface{}{
				"run_id": runID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"step": step,
	})
}

func v2CreateRunLogHandler(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if strings.TrimSpace(runID) == "" || strings.Contains(runID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid run ID", map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	var req struct {
		Stream string `json:"stream"`
		Chunk  string `json:"chunk"`
		Ts     string `json:"ts"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	stream := strings.TrimSpace(req.Stream)
	if stream == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "stream is required", map[string]interface{}{
			"stream": req.Stream,
		})
		return
	}

	switch stream {
	case "stdout", "stderr", "info":
	default:
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid log stream", map[string]interface{}{
			"stream": stream,
		})
		return
	}

	chunk := req.Chunk
	if strings.TrimSpace(chunk) == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "chunk is required", map[string]interface{}{
			"chunk": req.Chunk,
		})
		return
	}

	if _, err := GetRun(db, runID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"run_id": runID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	if err := CreateRunLog(db, runID, stream, chunk); err != nil {
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry run log", map[string]interface{}{
				"run_id": runID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func v2CreateRunArtifactHandler(w http.ResponseWriter, r *http.Request, runID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	if strings.TrimSpace(runID) == "" || strings.Contains(runID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid run ID", map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	var req struct {
		Kind       string                 `json:"kind"`
		StorageRef string                 `json:"storage_ref"`
		Sha256     *string                `json:"sha256"`
		Size       *int64                 `json:"size"`
		Metadata   map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "kind is required", map[string]interface{}{
			"kind": req.Kind,
		})
		return
	}

	switch kind {
	case "diff", "patch", "log", "report", "file":
	default:
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid artifact kind", map[string]interface{}{
			"kind": kind,
		})
		return
	}

	storageRef := strings.TrimSpace(req.StorageRef)
	if storageRef == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "storage_ref is required", map[string]interface{}{
			"storage_ref": req.StorageRef,
		})
		return
	}

	if req.Size == nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "size is required", map[string]interface{}{
			"size": req.Size,
		})
		return
	}

	if *req.Size < 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "size must be non-negative", map[string]interface{}{
			"size": *req.Size,
		})
		return
	}

	if req.Sha256 != nil {
		trimmed := strings.TrimSpace(*req.Sha256)
		if trimmed == "" {
			req.Sha256 = nil
		} else {
			req.Sha256 = &trimmed
		}
	}

	if _, err := GetRun(db, runID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"run_id": runID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	artifact, err := CreateArtifact(db, runID, kind, storageRef, req.Sha256, req.Size, req.Metadata)
	if err != nil {
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry artifact", map[string]interface{}{
				"run_id": runID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"run_id": runID,
		})
		return
	}

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"artifact": artifact,
	})
}
