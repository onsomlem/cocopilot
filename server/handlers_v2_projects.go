package server

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// API v2 Project Handlers
// ============================================================================

func v2CreateProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		Name     string                 `json:"name"`
		Workdir  string                 `json:"workdir"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	// Validate required fields
	name := strings.TrimSpace(req.Name)
	workdir := strings.TrimSpace(req.Workdir)
	if name == "" || workdir == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "name and workdir are required", nil)
		return
	}
	if err := validateWorkdir(workdir); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
		return
	}
	workdir = filepath.Clean(workdir)

	project, err := CreateProject(db, name, workdir, req.Settings)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"project_name": req.Name})
		return
	}

	CreateEvent(db, project.ID, "project.created", "project", project.ID, map[string]interface{}{
		"name": project.Name, "workdir": project.Workdir,
	})

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"project": project,
	})
}

func v2ListProjectsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	projects, err := ListProjects(db)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"projects": projects,
	})
}

func v2GetProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Extract project ID from URL path
	projectID := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	if projectID == "" || strings.Contains(projectID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project ID", map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	project, err := GetProject(db, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{"project_id": projectID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"project_id": projectID})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"project": project,
	})
}

func v2UpdateProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeV2MethodNotAllowed(w, r, http.MethodPut, http.MethodPatch)
		return
	}

	// Extract project ID from URL path
	projectID := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	if projectID == "" || strings.Contains(projectID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project ID", map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	var req struct {
		Name     *string                `json:"name"`
		Workdir  *string                `json:"workdir"`
		Settings map[string]interface{} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if req.Name == nil && req.Workdir == nil && req.Settings == nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "At least one field is required", nil)
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "name cannot be empty", nil)
			return
		}
		req.Name = &trimmed
	}

	if req.Workdir != nil {
		trimmed := strings.TrimSpace(*req.Workdir)
		if trimmed == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir cannot be empty", nil)
			return
		}
		if err := validateWorkdir(trimmed); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), nil)
			return
		}
		cleaned := filepath.Clean(trimmed)
		req.Workdir = &cleaned
	}

	project, err := UpdateProject(db, projectID, req.Name, req.Workdir, req.Settings)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{"project_id": projectID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"project_id": projectID})
		return
	}

	CreateEvent(db, projectID, "project.updated", "project", projectID, map[string]interface{}{
		"name": project.Name, "workdir": project.Workdir,
	})

	// Audit event: record which fields changed and the requester's IP.
	changedFields := []string{}
	if req.Name != nil {
		changedFields = append(changedFields, "name")
	}
	if req.Workdir != nil {
		changedFields = append(changedFields, "workdir")
	}
	if req.Settings != nil {
		changedFields = append(changedFields, "settings")
	}
	CreateEvent(db, projectID, "audit.project.updated", "project", projectID, map[string]interface{}{
		"changed_fields": changedFields,
		"changed_by_ip":  clientIP(r),
	})

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"project": project,
	})
}

func v2DeleteProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeV2MethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	// Extract project ID from URL path
	projectID := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	if projectID == "" || strings.Contains(projectID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project ID", map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	err := DeleteProject(db, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{"project_id": projectID})
			return
		}
		if strings.Contains(err.Error(), "cannot delete default") {
			writeV2Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), map[string]interface{}{"project_id": projectID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"project_id": projectID})
		return
	}

	// Emit event before responding (project is already deleted, use projectID as best-effort)
	CreateEvent(db, projectID, "project.deleted", "project", projectID, nil)

	w.WriteHeader(http.StatusNoContent)
}

func v2ProjectTreeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "tree" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project tree path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
	project, err := GetProject(db, projectID)
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

	workdir := strings.TrimSpace(project.Workdir)
	if workdir == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir is required", map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	cleaned := filepath.Clean(workdir)
	info, err := os.Stat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir not found", map[string]interface{}{
				"project_id": projectID,
				"workdir":    cleaned,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}
	if !info.IsDir() {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir must be a directory", map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}

	tree, err := buildProjectTreeSnapshot(cleaned)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}

	CreateEvent(db, projectID, "repo.scanned", "project", projectID, map[string]interface{}{
		"workdir": cleaned,
	})

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"tree": tree,
	})
}

func v2ProjectChangesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "changes" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project changes path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
	project, err := GetProject(db, projectID)
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

	workdir := strings.TrimSpace(project.Workdir)
	if workdir == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir is required", map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	cleaned := filepath.Clean(workdir)
	info, err := os.Stat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir not found", map[string]interface{}{
				"project_id": projectID,
				"workdir":    cleaned,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}
	if !info.IsDir() {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir must be a directory", map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}

	var since *time.Time
	if rawValues, ok := r.URL.Query()["since"]; ok {
		rawSince := ""
		if len(rawValues) > 0 {
			rawSince = strings.TrimSpace(rawValues[0])
		}
		if rawSince == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since cannot be empty", map[string]interface{}{
				"since": rawSince,
			})
			return
		}
		parsed, err := time.Parse(time.RFC3339Nano, rawSince)
		if err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since must be RFC3339", map[string]interface{}{
				"since": rawSince,
			})
			return
		}
		parsed = parsed.UTC()
		since = &parsed
	}

	isRepo, err := isGitRepo(cleaned)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}
	if !isRepo {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "workdir must be a git repository", map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}

	changes, err := gitStatusChanges(cleaned, since)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"workdir":    cleaned,
		})
		return
	}

	CreateEvent(db, projectID, "repo.changed", "project", projectID, map[string]interface{}{
		"change_count": len(changes),
	})

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"changes": changes,
	})
}

func v2ProjectAutomationRulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "automation" || parts[2] != "rules" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project automation rules path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
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

	rules := getAutomationRules()
	if rules == nil {
		rules = []automationRule{}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
	})
}

type v2AutomationSimulateEvent struct {
	Kind     string                 `json:"kind"`
	EntityID string                 `json:"entity_id"`
	Payload  map[string]interface{} `json:"payload"`
}

type v2AutomationSimulateRequest struct {
	Event *v2AutomationSimulateEvent `json:"event"`
}

type v2AutomationSimulatedTask struct {
	ProjectID    string   `json:"project_id"`
	ParentTaskID *int     `json:"parent_task_id,omitempty"`
	Title        *string  `json:"title,omitempty"`
	Instructions string   `json:"instructions"`
	Type         TaskType `json:"type"`
	Priority     int      `json:"priority"`
	Tags         []string `json:"tags,omitempty"`
}

type v2AutomationSimulatedAction struct {
	RuleName string                    `json:"rule_name"`
	Type     string                    `json:"type"`
	Task     v2AutomationSimulatedTask `json:"task"`
}

func v2ProjectAutomationSimulateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "automation" || parts[2] != "simulate" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project automation simulate path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}
	if strings.TrimSpace(string(bodyBytes)) == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "event is required", nil)
		return
	}

	var req v2AutomationSimulateRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}
	if req.Event == nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "event is required", nil)
		return
	}

	kind := strings.ToLower(strings.TrimSpace(req.Event.Kind))
	if kind == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "event.kind is required", map[string]interface{}{
			"kind": req.Event.Kind,
		})
		return
	}
	if kind != "task.completed" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Unsupported event kind", map[string]interface{}{
			"kind": req.Event.Kind,
		})
		return
	}

	taskID, ok := parseEventTaskID(req.Event.EntityID)
	if !ok {
		if fromPayload, ok := taskIDFromPayload(req.Event.Payload); ok {
			taskID = fromPayload
		} else {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "event.entity_id or event.payload.task_id is required", map[string]interface{}{
				"entity_id": req.Event.EntityID,
			})
			return
		}
	}

	task, err := GetTaskV2(db, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"task_id": taskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": taskID,
		})
		return
	}
	if task.ProjectID != projectID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "task does not belong to project", map[string]interface{}{
			"task_id":    taskID,
			"project_id": projectID,
		})
		return
	}

	blocked, reason, err := isAutomationBlockedByPolicies(db, projectID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}
	if blocked {
		message := "Automation blocked by policy"
		if reason != "" {
			message = fmt.Sprintf("Automation blocked by policy: %s", reason)
		}
		writeV2Error(w, http.StatusForbidden, "FORBIDDEN", message, map[string]interface{}{
			"project_id": projectID,
			"reason":     reason,
		})
		return
	}

	rules := getAutomationRules()
	if len(rules) == 0 {
		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"actions":                     []v2AutomationSimulatedAction{},
			"tasks_that_would_be_created": []v2AutomationSimulatedTask{},
		})
		return
	}

	event := Event{
		ID:         "evt_simulate",
		ProjectID:  projectID,
		Kind:       kind,
		EntityType: "task",
		EntityID:   fmt.Sprintf("%d", taskID),
		Payload:    req.Event.Payload,
	}
	templateData := buildAutomationTemplateData(event, task)

	actions := make([]v2AutomationSimulatedAction, 0)
	tasks := make([]v2AutomationSimulatedTask, 0)

	for _, rule := range rules {
		if !isAutomationRuleEnabled(rule) || rule.Trigger != "task.completed" {
			continue
		}
		for _, action := range rule.Actions {
			if action.Type != "create_task" {
				continue
			}

			instructions := strings.TrimSpace(applyAutomationTemplate(action.Task.Instructions, templateData))
			if instructions == "" {
				continue
			}

			var title *string
			if action.Task.Title != nil {
				renderedTitle := strings.TrimSpace(applyAutomationTemplate(*action.Task.Title, templateData))
				if renderedTitle != "" {
					title = &renderedTitle
				}
			}

			resolvedType := TaskTypeModify
			if action.Task.Type != nil {
				trimmedType := strings.ToUpper(strings.TrimSpace(*action.Task.Type))
				if trimmedType != "" {
					resolvedType = TaskType(trimmedType)
				}
			}

			resolvedPriority := 50
			if action.Task.Priority != nil {
				resolvedPriority = *action.Task.Priority
			}

			parentID := resolveAutomationParentID(action.Task.Parent, taskID)
			tags := normalizeAutomationTags(action.Task.Tags)

			simTask := v2AutomationSimulatedTask{
				ProjectID:    projectID,
				ParentTaskID: parentID,
				Title:        title,
				Instructions: instructions,
				Type:         resolvedType,
				Priority:     resolvedPriority,
				Tags:         tags,
			}

			actions = append(actions, v2AutomationSimulatedAction{
				RuleName: rule.Name,
				Type:     action.Type,
				Task:     simTask,
			})
			tasks = append(tasks, simTask)
		}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"actions":                     actions,
		"tasks_that_would_be_created": tasks,
	})
}

func v2ProjectAutomationReplayHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "automation" || parts[2] != "replay" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project automation replay path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
	if rawProjectID := strings.TrimSpace(r.URL.Query().Get("project_id")); rawProjectID != "" && rawProjectID != projectID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id must match path", map[string]interface{}{
			"project_id":      rawProjectID,
			"path_project_id": projectID,
		})
		return
	}

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

	sinceEventID := strings.TrimSpace(r.URL.Query().Get("since_event_id"))
	if sinceEventID == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since_event_id is required", map[string]interface{}{
			"since_event_id": sinceEventID,
		})
		return
	}

	limit := v2EventsListDefaultLimit
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
				"limit": rawLimit,
			})
			return
		}
		if parsed > v2EventsListMaxLimit {
			limit = v2EventsListMaxLimit
		} else {
			limit = parsed
		}
	}

	anchorProjectID, createdAt, err := GetEventReplayAnchor(db, sinceEventID)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since_event_id not found", map[string]interface{}{
			"since_event_id": sinceEventID,
		})
		return
	}
	if anchorProjectID != projectID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since_event_id must belong to project", map[string]interface{}{
			"since_event_id": sinceEventID,
			"project_id":     projectID,
		})
		return
	}

	results, _, err := ListEvents(db, projectID, "", createdAt, "", limit, 0)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id":     projectID,
			"since_event_id": sinceEventID,
		})
		return
	}

	replay := make([]Event, 0, len(results))
	for i := len(results) - 1; i >= 0; i-- {
		replay = append(replay, results[i])
	}

	taskCompletedCount := 0
	for _, event := range replay {
		if event.Kind == "task.completed" {
			taskCompletedCount++
			processAutomationEvent(db, event)
		}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"since_event_id":        sinceEventID,
		"events_replayed":       len(replay),
		"task_completed_events": taskCompletedCount,
	})
}

func v2ProjectAutomationStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project automation stats path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}
	projectID := parts[0]

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

	stats := map[string]interface{}{
		"project_id": projectID,
	}

	var triggered int
	if err := db.QueryRow("SELECT COUNT(*) FROM events WHERE project_id = ? AND kind = 'automation.triggered'", projectID).Scan(&triggered); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query automation stats", nil)
		return
	}
	stats["triggered_count"] = triggered

	var blocked int
	if err := db.QueryRow("SELECT COUNT(*) FROM events WHERE project_id = ? AND kind = 'automation.blocked'", projectID).Scan(&blocked); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query automation stats", nil)
		return
	}
	stats["blocked_count"] = blocked

	var circuitOpened int
	if err := db.QueryRow("SELECT COUNT(*) FROM events WHERE project_id = ? AND kind = 'automation.circuit_opened'", projectID).Scan(&circuitOpened); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Failed to query automation stats", nil)
		return
	}
	stats["circuit_opened_count"] = circuitOpened

	stats["total_events"] = triggered + blocked + circuitOpened

	if automationCircuit != nil {
		circuitStates := make(map[string]string)
		for _, rule := range getAutomationRules() {
			state := automationCircuit.GetState(rule.Name)
			switch state {
			case circuitClosed:
				circuitStates[rule.Name] = "closed"
			case circuitOpen:
				circuitStates[rule.Name] = "open"
			case circuitHalfOpen:
				circuitStates[rule.Name] = "half_open"
			}
		}
		stats["circuit_states"] = circuitStates
	}

	writeV2JSON(w, http.StatusOK, stats)
}

// v2ProjectGraphTasksHandler returns task DAG as nodes and edges for visualization.
// GET /api/v2/projects/{projectId}/graphs/tasks
func v2ProjectGraphTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project graph path", nil)
		return
	}
	projectID := parts[0]

	if _, err := GetProject(db, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{"project_id": projectID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	limit := 200
	if qs := r.URL.Query().Get("limit"); qs != "" {
		if parsed, err := strconv.Atoi(qs); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	tasks, _, err := ListTasksV2(db, projectID, "", "", "", "", "", limit, 0, "created_at", "asc")
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "failed to list tasks", nil)
		return
	}

	type graphNode struct {
		ID       int          `json:"id"`
		Title    *string      `json:"title,omitempty"`
		Status   TaskStatusV2 `json:"status"`
		Type     TaskType     `json:"type,omitempty"`
		ParentID *int         `json:"parent_task_id,omitempty"`
	}
	type graphEdge struct {
		From int    `json:"from"`
		To   int    `json:"to"`
		Type string `json:"type"`
	}

	nodes := make([]graphNode, 0, len(tasks))
	edges := make([]graphEdge, 0)
	taskIDs := make(map[int]bool, len(tasks))

	for _, t := range tasks {
		taskIDs[t.ID] = true
		nodes = append(nodes, graphNode{
			ID:       t.ID,
			Title:    t.Title,
			Status:   t.StatusV2,
			Type:     t.Type,
			ParentID: t.ParentTaskID,
		})
		if t.ParentTaskID != nil {
			edges = append(edges, graphEdge{From: *t.ParentTaskID, To: t.ID, Type: "parent_child"})
		}
	}

	// Add dependency edges
	for _, t := range tasks {
		deps, err := ListTaskDependencies(db, t.ID)
		if err != nil {
			continue
		}
		for _, dep := range deps {
			edges = append(edges, graphEdge{From: dep.DependsOnTaskID, To: dep.TaskID, Type: "depends_on"})
		}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"project_id": projectID,
		"nodes":      nodes,
		"edges":      edges,
	})
}

// v2ProjectIDESignalsHandler captures IDE signals from extensions.
// POST /api/v2/projects/{projectId}/ide-signals
func v2ProjectIDESignalsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project path", nil)
		return
	}
	projectID := parts[0]

	if _, err := GetProject(db, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{"project_id": projectID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	var payload struct {
		Kind string      `json:"kind"`
		Data interface{} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid JSON body", nil)
		return
	}
	if payload.Kind == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "kind is required", nil)
		return
	}

	eventPayload := map[string]interface{}{
		"kind": payload.Kind,
		"data": payload.Data,
	}
	event, err := CreateEvent(db, projectID, payload.Kind, "ide_signal", projectID, eventPayload)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "failed to record IDE signal", nil)
		return
	}

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"event_id":   event.ID,
		"project_id": projectID,
		"kind":       payload.Kind,
		"created_at": event.CreatedAt,
	})
}

// v2ProjectAuditExportHandler exports project audit events as CSV or JSON.
// GET /api/v2/projects/{projectId}/audit/export?format=csv|json
func v2ProjectAuditExportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project path", nil)
		return
	}
	projectID := parts[0]

	if _, err := GetProject(db, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{"project_id": projectID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "format must be json or csv", nil)
		return
	}

	limit := 10000
	if qs := r.URL.Query().Get("limit"); qs != "" {
		if parsed, err := strconv.Atoi(qs); err == nil && parsed > 0 && parsed <= 100000 {
			limit = parsed
		}
	}

	typeFilter := r.URL.Query().Get("type")

	events, _, err := ListEvents(db, projectID, typeFilter, "", "", limit, 0)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "failed to list events", nil)
		return
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=audit_"+projectID+".csv")
		w.WriteHeader(http.StatusOK)
		csvWriter := csv.NewWriter(w)
		csvWriter.Write([]string{"id", "kind", "entity_type", "entity_id", "project_id", "created_at", "payload"})
		for _, ev := range events {
			payloadBytes, _ := json.Marshal(ev.Payload)
			csvWriter.Write([]string{ev.ID, ev.Kind, ev.EntityType, ev.EntityID, ev.ProjectID, ev.CreatedAt, string(payloadBytes)})
		}
		csvWriter.Flush()
		return
	}

	// JSON export
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=audit_"+projectID+".json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id": projectID,
		"total":      len(events),
		"events":     events,
	})
}

func isGitRepo(workdir string) (bool, error) {
	cmd := exec.Command("git", "-C", workdir, "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

func gitStatusChanges(workdir string, since *time.Time) ([]FileChange, error) {
	cmd := exec.Command("git", "-C", workdir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	entries := strings.Split(string(output), "\x00")
	changes := make([]FileChange, 0, len(entries))
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		if len(entry) < 3 {
			continue
		}
		status := entry[:2]
		pathPart := strings.TrimSpace(entry[2:])
		if pathPart == "" {
			continue
		}
		if strings.Contains(pathPart, " -> ") {
			parts := strings.Split(pathPart, " -> ")
			pathPart = parts[len(parts)-1]
		}
		kind := mapGitStatusToChangeKind(status)
		if kind == "" {
			continue
		}

		tsTime := now
		fsPath := filepath.Join(workdir, filepath.FromSlash(pathPart))
		if info, err := os.Stat(fsPath); err == nil {
			tsTime = info.ModTime().UTC()
		}
		if since != nil && tsTime.Before(*since) {
			continue
		}

		changes = append(changes, FileChange{
			Path: filepath.ToSlash(pathPart),
			Kind: kind,
			Ts:   tsTime.Format(leaseTimeFormat),
		})
	}

	return changes, nil
}

func mapGitStatusToChangeKind(status string) string {
	switch {
	case status == "??":
		return "added"
	case status == "!!":
		return ""
	case strings.Contains(status, "D"):
		return "deleted"
	case strings.Contains(status, "A"):
		return "added"
	case strings.Contains(status, "M"):
		return "modified"
	case strings.Contains(status, "R"), strings.Contains(status, "C"):
		return "modified"
	case strings.Contains(status, "U"):
		return "modified"
	default:
		return "modified"
	}
}

func buildProjectTreeSnapshot(root string) (TreeNode, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return TreeNode{}, err
	}

	children := make([]TreeNode, 0, len(entries))
	for _, entry := range entries {
		name := filepath.ToSlash(entry.Name())
		if entry.IsDir() {
			children = append(children, TreeNode{
				Path:     name,
				Kind:     "dir",
				Children: []TreeNode{},
			})
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return TreeNode{}, err
		}
		size := info.Size()
		children = append(children, TreeNode{
			Path: name,
			Kind: "file",
			Size: &size,
		})
	}

	return TreeNode{
		Path:     ".",
		Kind:     "dir",
		Children: children,
	}, nil
}

// v2AuditHandler returns audit.* events across all projects with filters.
// GET /api/v2/audit?type=audit.policy.changed&project_id=...&limit=100&offset=0
func v2AuditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	query := r.URL.Query()

	// Default to audit.* events only. If type is specified, use it directly.
	eventType := strings.TrimSpace(query.Get("type"))
	projectID := strings.TrimSpace(query.Get("project_id"))
	since := strings.TrimSpace(query.Get("since"))

	limit := 100
	if v := query.Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	offset := 0
	if v := query.Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Query audit events: use LIKE for prefix matching if no specific type given.
	events, total, err := listAuditEvents(db, projectID, eventType, since, limit, offset)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// listAuditEvents queries events table with LIKE prefix matching for audit events.
func listAuditEvents(database *sql.DB, projectID, eventType, since string, limit, offset int) ([]Event, int, error) {
	conditions := make([]string, 0, 4)
	args := make([]interface{}, 0, 4)

	if eventType != "" {
		// Exact match if a specific type is given.
		conditions = append(conditions, "kind = ?")
		args = append(args, eventType)
	} else {
		// Default: all audit.* events.
		conditions = append(conditions, "kind LIKE ?")
		args = append(args, "audit.%")
	}
	if projectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}
	if since != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, since)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := database.QueryRow("SELECT COUNT(*) FROM events"+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}

	queryArgs := append(args, limit, offset)
	rows, err := database.Query(
		"SELECT id, project_id, kind, entity_type, entity_id, created_at, payload_json FROM events"+whereClause+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		var payloadJSON sql.NullString
		if err := rows.Scan(&event.ID, &event.ProjectID, &event.Kind, &event.EntityType, &event.EntityID, &event.CreatedAt, &payloadJSON); err != nil {
			return nil, 0, fmt.Errorf("scan audit event: %w", err)
		}
		if payloadJSON.Valid && payloadJSON.String != "" {
			if jsonErr := unmarshalJSON(payloadJSON.String, &event.Payload); jsonErr != nil {
				return nil, 0, fmt.Errorf("unmarshal audit payload: %w", jsonErr)
			}
		}
		events = append(events, event)
	}
	return events, total, nil
}

func v2ProjectAuditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "audit" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project audit path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
	if rawProjectID := strings.TrimSpace(r.URL.Query().Get("project_id")); rawProjectID != "" && rawProjectID != projectID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id must match path", map[string]interface{}{
			"project_id":      rawProjectID,
			"path_project_id": projectID,
		})
		return
	}

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
	query.Set("project_id", projectID)
	r.URL.RawQuery = query.Encode()

	v2ListEventsHandler(w, r)
}

func v2ProjectEventsStreamHandler(heartbeatInterval time.Duration, replayLimitMax int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeV2MethodNotAllowed(w, r, http.MethodGet)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
		parts := strings.Split(path, "/")
		if len(parts) != 3 || parts[0] == "" || parts[1] != "events" || parts[2] != "stream" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project events stream path", map[string]interface{}{
				"path": r.URL.Path,
			})
			return
		}

		projectID := parts[0]
		if rawProjectID := strings.TrimSpace(r.URL.Query().Get("project_id")); rawProjectID != "" && rawProjectID != projectID {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id must match path", map[string]interface{}{
				"project_id":      rawProjectID,
				"path_project_id": projectID,
			})
			return
		}

		query := r.URL.Query()
		query.Set("project_id", projectID)
		r.URL.RawQuery = query.Encode()

		v2EventsStreamHandler(heartbeatInterval, replayLimitMax)(w, r)
	}
}

func v2ProjectEventsReplayHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "events" || parts[2] != "replay" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project events replay path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
	if rawProjectID := strings.TrimSpace(r.URL.Query().Get("project_id")); rawProjectID != "" && rawProjectID != projectID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id must match path", map[string]interface{}{
			"project_id":      rawProjectID,
			"path_project_id": projectID,
		})
		return
	}

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

	sinceID := strings.TrimSpace(r.URL.Query().Get("since_id"))
	if sinceID == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since_id is required", map[string]interface{}{
			"since_id": sinceID,
		})
		return
	}

	limit := v2EventsListDefaultLimit
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
				"limit": rawLimit,
			})
			return
		}
		if parsed > v2EventsListMaxLimit {
			limit = v2EventsListMaxLimit
		} else {
			limit = parsed
		}
	}

	anchorProjectID, createdAt, err := GetEventReplayAnchor(db, sinceID)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since_id not found", map[string]interface{}{
			"since_id": sinceID,
		})
		return
	}
	if anchorProjectID != projectID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since_id must belong to project", map[string]interface{}{
			"since_id":   sinceID,
			"project_id": projectID,
		})
		return
	}

	results, _, err := ListEvents(db, projectID, "", createdAt, "", limit, 0)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"since_id":   sinceID,
		})
		return
	}

	replay := make([]Event, 0, len(results))
	for i := len(results) - 1; i >= 0; i-- {
		replay = append(replay, results[i])
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"events": replay,
	})
}

func v2ListProjectTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Extract project ID from URL path (/api/v2/projects/:id/tasks)
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "tasks" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project ID or path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]

	statusColumn, statusValue, err := resolveV2TaskListStatusFilter(r.URL.Query().Get("status"))
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid status filter", map[string]interface{}{
			"status": r.URL.Query().Get("status"),
		})
		return
	}

	query := r.URL.Query()
	typeFilter, err := resolveV2TaskListTypeFilter(query)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), map[string]interface{}{
			"type": query.Get("type"),
		})
		return
	}

	tagFilter, err := resolveV2TaskListTagFilter(query)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), map[string]interface{}{
			"tag": query.Get("tag"),
		})
		return
	}

	queryFilter, err := resolveV2TaskListQueryFilter(query)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error(), map[string]interface{}{
			"q": query.Get("q"),
		})
		return
	}

	limit := v2TaskListDefaultLimit
	offset := 0
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
				"limit": rawLimit,
			})
			return
		}
		if parsed > v2TaskListMaxLimit {
			limit = v2TaskListMaxLimit
		} else {
			limit = parsed
		}
	}
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil || parsed < 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "offset must be a non-negative integer", map[string]interface{}{
				"offset": rawOffset,
			})
			return
		}
		offset = parsed
	}

	sortField, sortDirection, err := resolveV2TaskListSort(r.URL.Query().Get("sort"))
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "sort must be created_at:asc, created_at:desc, updated_at:asc, or updated_at:desc", map[string]interface{}{
			"sort": r.URL.Query().Get("sort"),
		})
		return
	}

	// Verify project exists
	_, err = GetProject(db, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Project not found: %s", projectID), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"project_id": projectID})
		return
	}

	// Get tasks for this project
	tasks, total, err := ListTasksV2(db, projectID, statusColumn, statusValue, typeFilter, tagFilter, queryFilter, limit, offset, sortField, sortDirection)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"project_id": projectID})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"tasks": tasks,
		"total": total,
	})
}

func v2ProjectTasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		v2ListProjectTasksHandler(w, r)
		return
	case http.MethodPost:
		// continue
	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
		return
	}

	// Extract project ID from URL path (/api/v2/projects/:id/tasks)
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "tasks" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project ID or path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]

	var req struct {
		Instructions     string `json:"instructions"`
		ParentTaskID     *int   `json:"parent_task_id"`
		ProjectID        string `json:"project_id"`
		RequiresApproval bool   `json:"requires_approval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	instructions := strings.TrimSpace(req.Instructions)
	if instructions == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "instructions is required", map[string]interface{}{
			"instructions": req.Instructions,
		})
		return
	}

	if rawProjectID := strings.TrimSpace(req.ProjectID); rawProjectID != "" && rawProjectID != projectID {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id must match path", map[string]interface{}{
			"project_id": rawProjectID,
			"path":       projectID,
		})
		return
	}

	if req.ParentTaskID != nil && *req.ParentTaskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id must be a positive integer", map[string]interface{}{
			"parent_task_id": *req.ParentTaskID,
		})
		return
	}

	project, err := GetProject(db, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Project not found: %s", projectID), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
		})
		return
	}

	if req.ParentTaskID != nil {
		parentTask, err := GetTaskV2(db, *req.ParentTaskID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent_task_id not found", map[string]interface{}{
					"parent_task_id": *req.ParentTaskID,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
			})
			return
		}
		if parentTask.ProjectID != project.ID {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "parent task must be in the same project", map[string]interface{}{
				"parent_task_id": *req.ParentTaskID,
				"project_id":     project.ID,
			})
			return
		}
	}

	blocked, reason, err := isTaskCreateBlockedByPolicies(db, project.ID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": project.ID,
		})
		return
	}
	if blocked {
		message := "Task creation blocked by policy"
		if reason != "" {
			message = fmt.Sprintf("Task creation blocked by policy: %s", reason)
		}
		writeV2Error(w, http.StatusForbidden, "FORBIDDEN", message, map[string]interface{}{
			"project_id": project.ID,
			"reason":     reason,
		})
		return
	}

	task, err := CreateTaskV2(db, instructions, project.ID, req.ParentTaskID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": project.ID,
		})
		return
	}

	if req.RequiresApproval {
		db.Exec(`UPDATE tasks SET requires_approval = 1, approval_status = ? WHERE id = ?`, ApprovalPending, task.ID)
		task.RequiresApproval = true
		s := ApprovalPending
		task.ApprovalStatus = &s
	}

	go broadcastUpdate(v1EventTypeTasks)

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"task": task,
	})
}

func v2ProjectPoliciesHandler(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from URL path (/api/v2/projects/:id/policies)
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "policies" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project policies path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]

	switch r.Method {
	case http.MethodGet:
		_, err := GetProject(db, projectID)
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

		query := r.URL.Query()
		var enabledFilter *bool
		if rawEnabled := strings.TrimSpace(query.Get("enabled")); rawEnabled != "" {
			enabled, err := strconv.ParseBool(rawEnabled)
			if err != nil {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "enabled must be true or false", map[string]interface{}{
					"enabled": rawEnabled,
				})
				return
			}
			enabledFilter = &enabled
		}

		sortField, sortDirection, err := resolveV2PolicyListSort(query.Get("sort"))
		if err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "sort must be created_at:asc, created_at:desc, name:asc, or name:desc", map[string]interface{}{
				"sort": query.Get("sort"),
			})
			return
		}

		limit := v2PolicyListDefaultLimit
		offset := 0
		if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed <= 0 {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
					"limit": rawLimit,
				})
				return
			}
			if parsed > v2PolicyListMaxLimit {
				limit = v2PolicyListMaxLimit
			} else {
				limit = parsed
			}
		}
		if rawOffset := strings.TrimSpace(query.Get("offset")); rawOffset != "" {
			parsed, err := strconv.Atoi(rawOffset)
			if err != nil || parsed < 0 {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "offset must be a non-negative integer", map[string]interface{}{
					"offset": rawOffset,
				})
				return
			}
			offset = parsed
		}

		policies, total, err := ListPoliciesByProject(db, projectID, enabledFilter, limit, offset, sortField, sortDirection)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"policies": policies,
			"total":    total,
		})
		return
	case http.MethodPost:
		// continue
	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
		return
	}

	var req struct {
		Name        string       `json:"name"`
		Description *string      `json:"description"`
		Rules       []PolicyRule `json:"rules"`
		Enabled     *bool        `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
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

	_, err := GetProject(db, projectID)
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

	rules := req.Rules
	if rules == nil {
		rules = []PolicyRule{}
	}
	if err := validatePolicyRules(rules); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid policy rules", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	var description *string
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		if trimmed != "" {
			description = &trimmed
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	policy, err := CreatePolicy(db, projectID, name, description, rules, enabled)
	if err != nil {
		if errors.Is(err, ErrInvalidPolicyRules) {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid policy rules", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"name":       name,
		})
		return
	}

	CreateEvent(db, projectID, "audit.policy.changed", "policy", policy.ID, map[string]interface{}{
		"action":        "created",
		"policy_id":     policy.ID,
		"policy_name":   policy.Name,
		"changed_by_ip": clientIP(r),
	})

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"policy": policy,
	})
}

func v2ProjectPolicyDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")

	// Handle /enable and /disable sub-actions (4 parts: projectID/policies/policyID/action)
	if len(parts) == 4 && parts[0] != "" && parts[1] == "policies" && parts[2] != "" {
		action := parts[3]
		if action == "enable" || action == "disable" {
			projectID := parts[0]
			policyID := parts[2]

			if r.Method != http.MethodPost {
				writeV2MethodNotAllowed(w, r, http.MethodPost)
				return
			}

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

			enabled := action == "enable"
			policy, err := UpdatePolicy(db, projectID, policyID, nil, nil, nil, &enabled)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
						"project_id": projectID,
						"policy_id":  policyID,
					})
					return
				}
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
					"project_id": projectID,
					"policy_id":  policyID,
				})
				return
			}

			writeV2JSON(w, http.StatusOK, map[string]interface{}{
				"policy": policy,
			})

			CreateEvent(db, projectID, "audit.policy.changed", "policy", policyID, map[string]interface{}{
				"action":        action,
				"policy_id":     policyID,
				"changed_by_ip": clientIP(r),
			})
			return
		}
	}

	if len(parts) != 3 || parts[0] == "" || parts[1] != "policies" || parts[2] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project policy path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]
	policyID := parts[2]

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
		policy, err := GetPolicy(db, projectID, policyID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
					"project_id": projectID,
					"policy_id":  policyID,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
				"policy_id":  policyID,
			})
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"policy": policy,
		})
		return
	case http.MethodPatch:
		// continue
	case http.MethodDelete:
		err := DeletePolicy(db, projectID, policyID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
					"project_id": projectID,
					"policy_id":  policyID,
				})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
				"policy_id":  policyID,
			})
			return
		}

		CreateEvent(db, projectID, "audit.policy.changed", "policy", policyID, map[string]interface{}{
			"action":        "deleted",
			"policy_id":     policyID,
			"changed_by_ip": clientIP(r),
		})
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPatch, http.MethodDelete)
		return
	}

	var req struct {
		Name        *string      `json:"name"`
		Description *string      `json:"description"`
		Rules       []PolicyRule `json:"rules"`
		Enabled     *bool        `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if req.Name == nil && req.Description == nil && req.Rules == nil && req.Enabled == nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "At least one field is required", nil)
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "name cannot be empty", nil)
			return
		}
		req.Name = &trimmed
	}
	if req.Rules != nil {
		if err := validatePolicyRules(req.Rules); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid policy rules", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
	}

	policy, err := UpdatePolicy(db, projectID, policyID, req.Name, req.Description, req.Rules, req.Enabled)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"project_id": projectID,
				"policy_id":  policyID,
			})
			return
		}
		if errors.Is(err, ErrInvalidPolicyRules) {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid policy rules", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"policy_id":  policyID,
		})
		return
	}

	CreateEvent(db, projectID, "audit.policy.changed", "policy", policyID, map[string]interface{}{
		"action":        "updated",
		"policy_id":     policyID,
		"policy_name":   policy.Name,
		"changed_by_ip": clientIP(r),
	})

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"policy": policy,
	})
}

func v2ProjectMemoryHandler(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from URL path (/api/v2/projects/:id/memory)
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "memory" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project memory path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]

	if r.Method == http.MethodGet {
		scope := strings.TrimSpace(r.URL.Query().Get("scope"))
		key := strings.TrimSpace(r.URL.Query().Get("key"))
		queryFilter := strings.TrimSpace(r.URL.Query().Get("q"))

		_, err := GetProject(db, projectID)
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

		items, err := QueryMemories(db, projectID, scope, key, queryFilter)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
				"project_id": projectID,
			})
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"items": items,
		})
		return
	}

	if r.Method != http.MethodPut {
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPut)
		return
	}

	var req struct {
		Scope      string                 `json:"scope"`
		Key        string                 `json:"key"`
		Value      map[string]interface{} `json:"value"`
		SourceRefs []string               `json:"source_refs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	scope := strings.TrimSpace(req.Scope)
	key := strings.TrimSpace(req.Key)
	if scope == "" || key == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "scope and key are required", map[string]interface{}{
			"scope": req.Scope,
			"key":   req.Key,
		})
		return
	}
	if req.Value == nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "value is required", nil)
		return
	}

	_, err := GetProject(db, projectID)
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

	// Check if memory already exists to distinguish created vs updated.
	existingMem, _ := GetMemory(db, projectID, scope, key)

	memory, err := CreateMemory(db, projectID, scope, key, req.Value, req.SourceRefs)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"scope":      scope,
			"key":        key,
		})
		return
	}

	eventType := "memory.created"
	if existingMem != nil {
		eventType = "memory.updated"
	}
	CreateEvent(db, projectID, eventType, "memory", memory.ID, map[string]interface{}{
		"scope": scope,
		"key":   key,
	})

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"item": memory,
	})
}

func v2ProjectContextPacksHandler(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from URL path (/api/v2/projects/:id/context-packs)
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "context-packs" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid project context packs path", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	projectID := parts[0]

	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		TaskID              int    `json:"task_id"`
		Query               string `json:"query"`
		IncludeFileMetadata bool   `json:"include_file_metadata"`
		Budget              struct {
			MaxFiles    int `json:"max_files"`
			MaxBytes    int `json:"max_bytes"`
			MaxSnippets int `json:"max_snippets"`
		} `json:"budget"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if req.TaskID <= 0 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "task_id is required", map[string]interface{}{
			"task_id": req.TaskID,
		})
		return
	}

	_, err := GetProject(db, projectID)
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

	task, err := GetTaskV2(db, req.TaskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"task_id": req.TaskID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"task_id": req.TaskID,
		})
		return
	}
	if task.ProjectID != projectID {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Task not found in project", map[string]interface{}{
			"task_id":    req.TaskID,
			"project_id": projectID,
		})
		return
	}

	contents := map[string]interface{}{
		"task": map[string]interface{}{
			"id":           task.ID,
			"title":        task.Title,
			"instructions": task.Instructions,
			"type":         task.Type,
			"status_v2":    task.StatusV2,
		},
	}
	if strings.TrimSpace(req.Query) != "" {
		contents["query"] = strings.TrimSpace(req.Query)
	}
	if req.Budget.MaxFiles > 0 || req.Budget.MaxBytes > 0 || req.Budget.MaxSnippets > 0 {
		contents["budget"] = map[string]interface{}{
			"max_files":    req.Budget.MaxFiles,
			"max_bytes":    req.Budget.MaxBytes,
			"max_snippets": req.Budget.MaxSnippets,
		}
	}

	// Include repo file metadata if requested
	if req.IncludeFileMetadata {
		limit := 1000
		if req.Budget.MaxFiles > 0 {
			limit = req.Budget.MaxFiles
		}
		repoFiles, total, err := ListRepoFiles(db, projectID, ListRepoFilesOpts{Limit: limit})
		if err == nil && total > 0 {
			filesSummary := make([]map[string]interface{}, 0, len(repoFiles))
			for _, f := range repoFiles {
				entry := map[string]interface{}{"path": f.Path}
				if f.Language != nil {
					entry["language"] = *f.Language
				}
				if f.SizeBytes != nil {
					entry["size_bytes"] = *f.SizeBytes
				}
				filesSummary = append(filesSummary, entry)
			}
			contents["repo_files"] = map[string]interface{}{
				"total": total,
				"files": filesSummary,
			}
		}
	}

	summary := fmt.Sprintf("Context pack for task %d", req.TaskID)

	pack, err := CreateContextPack(db, projectID, req.TaskID, summary, contents)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"task_id":    req.TaskID,
		})
		return
	}

	CreateEvent(db, projectID, "context.refreshed", "context_pack", pack.ID, map[string]interface{}{
		"task_id": req.TaskID,
		"pack_id": pack.ID,
	})

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"context_pack": pack,
	})
}

func v2ContextPackDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	packID := strings.TrimPrefix(r.URL.Path, "/api/v2/context-packs/")
	if packID == "" || strings.Contains(packID, "/") {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid context pack ID", map[string]interface{}{
			"pack_id": packID,
		})
		return
	}

	pack, err := GetContextPackByID(db, packID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), map[string]interface{}{
				"pack_id": packID,
			})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"pack_id": packID,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"context_pack": pack,
	})
}

// v2LeaseHandler handles POST /api/v2/leases
func v2LeaseHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		v2CreateLeaseHandler(w, r)
	default:
		writeV2MethodNotAllowed(w, r, http.MethodPost)
	}
}

// v2CreateLeaseHandler handles POST /api/v2/leases
func v2CreateLeaseHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID  int    `json:"task_id"`
		AgentID string `json:"agent_id"`
		Mode    string `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if req.TaskID == 0 || req.AgentID == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "task_id and agent_id are required", map[string]interface{}{
			"task_id":  req.TaskID,
			"agent_id": req.AgentID,
		})
		return
	}

	if req.Mode == "" {
		req.Mode = "exclusive"
	}

	lease, err := CreateLease(db, req.TaskID, req.AgentID, req.Mode)
	if err != nil {
		if errors.Is(err, ErrLeaseConflict) || isLeaseConflictError(err) {
			writeV2Error(w, http.StatusConflict, "CONFLICT", "Task is already leased by another agent", map[string]interface{}{
				"task_id":  req.TaskID,
				"agent_id": req.AgentID,
			})
			return
		}
		if isSQLiteBusyError(err) {
			writeV2Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database is busy, retry claim", map[string]interface{}{
				"task_id":  req.TaskID,
				"agent_id": req.AgentID,
			})
			return
		}
		log.Printf("Error creating lease: %v", err)
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Internal server error", map[string]interface{}{
			"task_id":  req.TaskID,
			"agent_id": req.AgentID,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, lease)
}

// v2LeaseActionHandler handles actions on specific leases
func v2LeaseActionHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/leases/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Lease ID required", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	leaseID := parts[0]

	if len(parts) == 2 {
		switch parts[1] {
		case "heartbeat":
			if r.Method == http.MethodPost {
				v2LeaseHeartbeatHandler(w, r, leaseID)
			} else {
				writeV2MethodNotAllowed(w, r, http.MethodPost)
			}
			return
		case "release":
			if r.Method == http.MethodPost {
				v2LeaseReleaseHandler(w, r, leaseID)
			} else {
				writeV2MethodNotAllowed(w, r, http.MethodPost)
			}
			return
		}
	}

	writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Not found", map[string]interface{}{
		"path":     r.URL.Path,
		"lease_id": leaseID,
	})
}

// v2LeaseHeartbeatHandler handles POST /api/v2/leases/:id/heartbeat
func v2LeaseHeartbeatHandler(w http.ResponseWriter, r *http.Request, leaseID string) {
	lease, err := GetLeaseByID(db, leaseID)
	if err != nil {
		log.Printf("Error getting lease: %v", err)
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Internal server error", map[string]interface{}{
			"lease_id": leaseID,
		})
		return
	}

	if lease == nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Lease not found", map[string]interface{}{
			"lease_id": leaseID,
		})
		return
	}

	// Check if lease has expired using canonical IsLeaseActive() helper.
	if !IsLeaseActive(lease.ExpiresAt) {
		writeV2Error(w, http.StatusGone, "GONE", "Lease has expired", map[string]interface{}{
			"lease_id": leaseID,
		})
		return
	}

	// Extend lease by 15 minutes from now
	err = ExtendLease(db, leaseID, 15*time.Minute)
	if err != nil {
		log.Printf("Error extending lease: %v", err)
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Internal server error", map[string]interface{}{
			"lease_id": leaseID,
		})
		return
	}

	// Get updated lease
	lease, err = GetLeaseByID(db, leaseID)
	if err != nil {
		log.Printf("Error getting updated lease: %v", err)
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Internal server error", map[string]interface{}{
			"lease_id": leaseID,
		})
		return
	}

	emitLeaseLifecycleEvent(db, "lease.renewed", lease, nil)

	writeV2JSON(w, http.StatusOK, lease)
}

// v2LeaseReleaseHandler handles POST /api/v2/leases/:id/release
func v2LeaseReleaseHandler(w http.ResponseWriter, r *http.Request, leaseID string) {
	var req struct {
		Reason string `json:"reason"`
	}

	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON", map[string]interface{}{
				"lease_id": leaseID,
			})
			return
		}
	}

	released, _, err := ReleaseLease(db, leaseID, req.Reason)
	if err != nil {
		log.Printf("Error releasing lease: %v", err)
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "Internal server error", map[string]interface{}{
			"lease_id": leaseID,
		})
		return
	}

	if !released {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Lease not found", map[string]interface{}{
			"lease_id": leaseID,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"lease_id": leaseID,
		"released": true,
	})
}
