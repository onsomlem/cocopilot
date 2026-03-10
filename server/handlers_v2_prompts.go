package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// v2ProjectPromptsHandler handles GET/POST on /api/v2/projects/{id}/prompts
func v2ProjectPromptsHandler(w http.ResponseWriter, r *http.Request, projectID string) {
	switch r.Method {
	case http.MethodGet:
		role := r.URL.Query().Get("role")
		activeOnly := r.URL.Query().Get("active_only") == "true"
		templates, err := ListPromptTemplates(db, projectID, role, activeOnly)
		if err != nil {
			writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to list prompts", nil)
			return
		}
		if templates == nil {
			templates = []PromptTemplate{}
		}
		writeV2JSON(w, 200, map[string]interface{}{"prompts": templates})

	case http.MethodPost:
		var body struct {
			Role         string  `json:"role"`
			Name         string  `json:"name"`
			Description  *string `json:"description"`
			SystemPrompt string  `json:"system_prompt"`
			UserTemplate string  `json:"user_template"`
			OutputSchema *string `json:"output_schema"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeV2Error(w, 400, "INVALID_JSON", "Invalid request body", nil)
			return
		}
		if body.Role == "" || body.Name == "" {
			writeV2Error(w, 400, "VALIDATION_ERROR", "role and name are required", nil)
			return
		}

		pt, err := CreatePromptTemplate(db, projectID, body.Role, body.Name, body.SystemPrompt, body.UserTemplate, body.Description, body.OutputSchema)
		if err != nil {
			writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to create prompt template", nil)
			return
		}
		writeV2JSON(w, 201, map[string]interface{}{"prompt": pt})

	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

// v2ProjectPromptDetailHandler handles GET/PUT/DELETE on /api/v2/projects/{id}/prompts/{promptID}
func v2ProjectPromptDetailHandler(w http.ResponseWriter, r *http.Request, projectID, promptID string) {
	switch r.Method {
	case http.MethodGet:
		pt, err := GetPromptTemplate(db, promptID)
		if err == sql.ErrNoRows {
			writeV2Error(w, 404, "NOT_FOUND", "Prompt template not found", nil)
			return
		}
		if err != nil {
			writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to get prompt template", nil)
			return
		}
		writeV2JSON(w, 200, map[string]interface{}{"prompt": pt})

	case http.MethodPut, http.MethodPatch:
		pt, err := GetPromptTemplate(db, promptID)
		if err == sql.ErrNoRows {
			writeV2Error(w, 404, "NOT_FOUND", "Prompt template not found", nil)
			return
		}
		if err != nil {
			writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to get prompt template", nil)
			return
		}

		var body struct {
			Name         *string `json:"name"`
			Description  *string `json:"description"`
			SystemPrompt *string `json:"system_prompt"`
			UserTemplate *string `json:"user_template"`
			OutputSchema *string `json:"output_schema"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeV2Error(w, 400, "INVALID_JSON", "Invalid request body", nil)
			return
		}

		if body.Name != nil {
			pt.Name = *body.Name
		}
		if body.Description != nil {
			pt.Description = body.Description
		}
		if body.SystemPrompt != nil {
			pt.SystemPrompt = *body.SystemPrompt
		}
		if body.UserTemplate != nil {
			pt.UserTemplate = *body.UserTemplate
		}
		if body.OutputSchema != nil {
			pt.OutputSchema = body.OutputSchema
		}

		if err := UpdatePromptTemplate(db, pt); err != nil {
			writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to update prompt template", nil)
			return
		}

		updated, _ := GetPromptTemplate(db, promptID)
		writeV2JSON(w, 200, map[string]interface{}{"prompt": updated})

	case http.MethodDelete:
		if err := DeletePromptTemplate(db, promptID); err != nil {
			writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to delete prompt template", nil)
			return
		}
		writeV2JSON(w, 200, map[string]interface{}{"deleted": true})

	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete)
	}
}

// v2ProjectPromptActivateHandler handles POST on /api/v2/projects/{id}/prompts/activate
func v2ProjectPromptActivateHandler(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var body struct {
		Role    string `json:"role"`
		Version int    `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeV2Error(w, 400, "INVALID_JSON", "Invalid request body", nil)
		return
	}
	if body.Role == "" || body.Version <= 0 {
		writeV2Error(w, 400, "VALIDATION_ERROR", "role and version (>0) are required", nil)
		return
	}

	if err := ActivatePromptVersion(db, projectID, body.Role, body.Version); err != nil {
		writeV2Error(w, 404, "NOT_FOUND", err.Error(), nil)
		return
	}

	pt, _ := GetActivePromptByRole(db, projectID, body.Role)
	writeV2JSON(w, 200, map[string]interface{}{"prompt": pt})
}

// v2ProjectPromptTestRunHandler handles POST on /api/v2/projects/{id}/prompts/test-run
// It renders a prompt template with sample context and returns the rendered output.
func v2ProjectPromptTestRunHandler(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var body struct {
		Role    string                 `json:"role"`
		Version *int                   `json:"version"`
		Context map[string]interface{} `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeV2Error(w, 400, "INVALID_JSON", "Invalid request body", nil)
		return
	}
	if body.Role == "" {
		writeV2Error(w, 400, "VALIDATION_ERROR", "role is required", nil)
		return
	}

	// Get the template
	var pt *PromptTemplate
	var err error
	if body.Version != nil {
		// Find specific version
		templates, lErr := ListPromptTemplates(db, projectID, body.Role, false)
		if lErr != nil {
			writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to list templates", nil)
			return
		}
		for i := range templates {
			if templates[i].Version == *body.Version {
				pt = &templates[i]
				break
			}
		}
		if pt == nil {
			writeV2Error(w, 404, "NOT_FOUND", "Version not found for role", nil)
			return
		}
	} else {
		pt, err = GetActivePromptByRole(db, projectID, body.Role)
		if err != nil {
			writeV2Error(w, 404, "NOT_FOUND", "No active template for role", nil)
			return
		}
	}

	// Simple template rendering: replace {{key}} with context values
	rendered := renderTemplate(pt.UserTemplate, body.Context)

	writeV2JSON(w, 200, map[string]interface{}{
		"role":            pt.Role,
		"version":         pt.Version,
		"system_prompt":   pt.SystemPrompt,
		"rendered_user":   rendered,
		"template_id":     pt.ID,
	})
}

// renderTemplate performs simple {{key}} substitution.
func renderTemplate(tmpl string, ctx map[string]interface{}) string {
	if ctx == nil {
		return tmpl
	}
	result := tmpl
	for key, val := range ctx {
		placeholder := "{{" + key + "}}"
		var valStr string
		switch v := val.(type) {
		case string:
			valStr = v
		case float64:
			valStr = strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			valStr = strconv.FormatBool(v)
		default:
			b, _ := json.Marshal(v)
			valStr = string(b)
		}
		result = strings.ReplaceAll(result, placeholder, valStr)
	}
	return result
}

// v2ProjectPromptVersionsHandler handles GET on /api/v2/projects/{id}/prompts/role/{role}/versions
func v2ProjectPromptVersionsHandler(w http.ResponseWriter, r *http.Request, projectID, role string) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	templates, err := ListPromptTemplates(db, projectID, role, false)
	if err != nil {
		writeV2Error(w, 500, "INTERNAL_ERROR", "Failed to list versions", nil)
		return
	}
	if templates == nil {
		templates = []PromptTemplate{}
	}
	writeV2JSON(w, 200, map[string]interface{}{"role": role, "versions": templates})
}
