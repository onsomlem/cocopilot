package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// ============================================================================
// Task Template Handlers
// ============================================================================

func v2ProjectTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Project ID required", nil)
		return
	}
	projectID := parts[0]

	switch r.Method {
	case http.MethodGet:
		templates, err := ListTaskTemplates(db, projectID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		if templates == nil {
			templates = []TaskTemplate{}
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{"templates": templates})
	case http.MethodPost:
		var req struct {
			Name            string                 `json:"name"`
			Description     *string                `json:"description"`
			Instructions    string                 `json:"instructions"`
			DefaultType     *string                `json:"default_type"`
			DefaultPriority *int                   `json:"default_priority"`
			DefaultTags     []string               `json:"default_tags"`
			DefaultMetadata map[string]interface{} `json:"default_metadata"`
			DefaultLoopAnchor *string              `json:"default_loop_anchor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
			return
		}
		if req.Name == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Template name is required", nil)
			return
		}
		priority := 50
		if req.DefaultPriority != nil {
			priority = *req.DefaultPriority
		}
		tmpl, err := CreateTaskTemplate(db, projectID, req.Name, req.Description, req.Instructions, req.DefaultType, priority, req.DefaultTags, req.DefaultMetadata, req.DefaultLoopAnchor)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		writeV2JSON(w, http.StatusCreated, map[string]interface{}{"template": tmpl})
	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

func v2ProjectTemplateDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	// Expected: {projectId}/templates/{templateId} or {projectId}/templates/{templateId}/instantiate
	if len(parts) < 3 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Template ID required", nil)
		return
	}
	projectID := parts[0]
	templateID := parts[2]

	if len(parts) == 4 && parts[3] == "instantiate" {
		v2TemplateInstantiateHandler(w, r, projectID, templateID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tmpl, err := GetTaskTemplate(db, templateID)
		if err != nil {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found", nil)
			return
		}
		if tmpl.ProjectID != projectID {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found in project", nil)
			return
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{"template": tmpl})
	case http.MethodPut:
		var req struct {
			Name              *string                `json:"name"`
			Description       *string                `json:"description"`
			Instructions      *string                `json:"instructions"`
			DefaultType       *string                `json:"default_type"`
			DefaultPriority   *int                   `json:"default_priority"`
			DefaultTags       []string               `json:"default_tags"`
			DefaultMetadata   map[string]interface{} `json:"default_metadata"`
			DefaultLoopAnchor *string                `json:"default_loop_anchor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
			return
		}
		tmpl, err := UpdateTaskTemplate(db, templateID, req.Name, req.Description, req.Instructions, req.DefaultType, req.DefaultPriority, req.DefaultTags, req.DefaultMetadata, req.DefaultLoopAnchor)
		if err != nil {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{"template": tmpl})
	case http.MethodDelete:
		if err := DeleteTaskTemplate(db, templateID); err != nil {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{"deleted": true})
	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodDelete)
	}
}

func v2TemplateInstantiateHandler(w http.ResponseWriter, r *http.Request, projectID, templateID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	tmpl, err := GetTaskTemplate(db, templateID)
	if err != nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found", nil)
		return
	}
	if tmpl.ProjectID != projectID {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found in project", nil)
		return
	}

	// Allow overrides from request body
	var overrides struct {
		Title        *string  `json:"title"`
		Instructions *string  `json:"instructions"`
		Priority     *int     `json:"priority"`
		Tags         []string `json:"tags"`
	}
	json.NewDecoder(r.Body).Decode(&overrides)

	instructions := tmpl.Instructions
	if overrides.Instructions != nil {
		instructions = *overrides.Instructions
	}

	title := &tmpl.Name
	if overrides.Title != nil {
		title = overrides.Title
	}

	priority := &tmpl.DefaultPriority
	if overrides.Priority != nil {
		priority = overrides.Priority
	}

	tags := tmpl.DefaultTags
	if overrides.Tags != nil {
		tags = overrides.Tags
	}

	var taskType *TaskType
	if tmpl.DefaultType != nil {
		tt := TaskType(*tmpl.DefaultType)
		taskType = &tt
	}

	task, err := CreateTaskV2WithMeta(db, instructions, projectID, nil, title, taskType, priority, tags, tmpl.DefaultLoopAnchor)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	CreateEvent(db, projectID, "task.created", "task", strconv.Itoa(task.ID), map[string]interface{}{
		"template_id": templateID,
		"title":       title,
	})

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{"task": task, "template_id": templateID})
}
