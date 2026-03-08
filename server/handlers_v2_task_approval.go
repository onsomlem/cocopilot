package server

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ============================================================================
// Task Approval Handlers
// ============================================================================

func v2TaskApproveHandler(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}
	taskID, err := strconv.Atoi(rawID)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", nil)
		return
	}
	task, err := GetTaskV2(db, taskID)
	if err != nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Task not found", nil)
		return
	}
	if !task.RequiresApproval {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Task does not require approval", nil)
		return
	}
	if task.ApprovalStatus != nil && *task.ApprovalStatus == ApprovalApproved {
		writeV2Error(w, http.StatusConflict, "ALREADY_APPROVED", "Task already approved", nil)
		return
	}
	updated, err := SetTaskApprovalStatus(db, taskID, ApprovalApproved)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}
	CreateEvent(db, task.ProjectID, "task.approved", "task", strconv.Itoa(taskID), map[string]interface{}{
		"task_id": taskID,
	})
	writeV2JSON(w, http.StatusOK, map[string]interface{}{"task": updated})
}

func v2TaskRejectHandler(w http.ResponseWriter, r *http.Request, rawID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}
	taskID, err := strconv.Atoi(rawID)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", nil)
		return
	}
	task, err := GetTaskV2(db, taskID)
	if err != nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Task not found", nil)
		return
	}
	if !task.RequiresApproval {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Task does not require approval", nil)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	updated, err := SetTaskApprovalStatus(db, taskID, ApprovalRejected)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}
	CreateEvent(db, task.ProjectID, "task.rejected", "task", strconv.Itoa(taskID), map[string]interface{}{
		"task_id": taskID,
		"reason":  req.Reason,
	})
	writeV2JSON(w, http.StatusOK, map[string]interface{}{"task": updated})
}
