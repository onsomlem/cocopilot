package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// v2ProjectPlanningHandler handles GET /api/v2/projects/:id/planning
// Returns the planning state for a project.
func v2ProjectPlanningHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid planning path", nil)
		return
	}
	projectID := parts[0]

	switch r.Method {
	case http.MethodGet:
		ps, err := GetPlanningState(db, projectID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		if ps == nil {
			writeV2JSON(w, http.StatusOK, map[string]interface{}{
				"planning_state": nil,
				"initialized":   false,
			})
			return
		}

		workstreams, _ := ListWorkstreams(db, projectID, "")
		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"planning_state": ps,
			"workstreams":    workstreams,
			"initialized":   true,
		})

	case http.MethodPost:
		// POST seeds/initializes planning state
		ps, err := SeedPlanningState(db, projectID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		workstreams, _ := ListWorkstreams(db, projectID, "")
		writeV2JSON(w, http.StatusCreated, map[string]interface{}{
			"planning_state": ps,
			"workstreams":    workstreams,
		})

	case http.MethodPut, http.MethodPatch:
		// Update planning state fields
		ps, err := GetOrCreatePlanningState(db, projectID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}

		var req struct {
			PlanningMode  *string  `json:"planning_mode"`
			Goals         []string `json:"goals"`
			ReleaseFocus  *string  `json:"release_focus"`
			MustNotForget []string `json:"must_not_forget"`
			Blockers      []string `json:"blockers"`
			Risks         []string `json:"risks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid JSON body", nil)
			return
		}

		if req.PlanningMode != nil {
			ps.PlanningMode = *req.PlanningMode
		}
		if req.Goals != nil {
			ps.Goals = req.Goals
		}
		if req.ReleaseFocus != nil {
			ps.ReleaseFocus = *req.ReleaseFocus
		}
		if req.MustNotForget != nil {
			ps.MustNotForget = req.MustNotForget
		}
		if req.Blockers != nil {
			ps.Blockers = req.Blockers
		}
		if req.Risks != nil {
			ps.Risks = req.Risks
		}

		if err := UpdatePlanningState(db, ps); err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"planning_state": ps,
		})

	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch)
	}
}

// v2ProjectPlanningCycleHandler handles POST /api/v2/projects/:id/planning/run
// Triggers a planning cycle.
func v2ProjectPlanningCycleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid planning run path", nil)
		return
	}
	projectID := parts[0]

	cfg := DefaultPipelineConfig()

	// Allow overrides from request body
	var req struct {
		MaxTasksPerCycle *int  `json:"max_tasks_per_cycle"`
		EnableAntiDrift  *bool `json:"enable_anti_drift"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.MaxTasksPerCycle != nil {
			cfg.MaxTasksPerCycle = *req.MaxTasksPerCycle
		}
		if req.EnableAntiDrift != nil {
			cfg.EnableAntiDrift = *req.EnableAntiDrift
		}
	}

	result, err := RunPlanningPipeline(db, projectID, cfg)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "PIPELINE_ERROR", err.Error(), nil)
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"result": result,
	})
}

// v2ProjectWorkstreamsHandler handles GET/POST /api/v2/projects/:id/workstreams
func v2ProjectWorkstreamsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid workstreams path", nil)
		return
	}
	projectID := parts[0]

	switch r.Method {
	case http.MethodGet:
		statusFilter := r.URL.Query().Get("status")
		workstreams, err := ListWorkstreams(db, projectID, statusFilter)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		if workstreams == nil {
			workstreams = []Workstream{}
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"workstreams": workstreams,
			"total":       len(workstreams),
		})

	case http.MethodPost:
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Why         string `json:"why"`
			WhatNext    string `json:"what_next"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid JSON body", nil)
			return
		}
		if req.Title == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "title is required", nil)
			return
		}

		ps, err := GetOrCreatePlanningState(db, projectID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}

		ws, err := CreateWorkstream(db, projectID, ps.ID, req.Title, req.Description, req.Why, req.WhatNext)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}

		writeV2JSON(w, http.StatusCreated, map[string]interface{}{
			"workstream": ws,
		})

	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

// v2ProjectWorkstreamDetailHandler handles GET/PUT/DELETE /api/v2/projects/:id/workstreams/:wsid
func v2ProjectWorkstreamDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] == "" || parts[2] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid workstream path", nil)
		return
	}
	projectID := parts[0]
	workstreamID := parts[2]

	switch r.Method {
	case http.MethodGet:
		ws, err := GetWorkstream(db, projectID, workstreamID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		if ws == nil {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Workstream not found", nil)
			return
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"workstream": ws,
		})

	case http.MethodPut, http.MethodPatch:
		ws, err := GetWorkstream(db, projectID, workstreamID)
		if err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		if ws == nil {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Workstream not found", nil)
			return
		}

		var req struct {
			Title           *string  `json:"title"`
			Description     *string  `json:"description"`
			Status          *string  `json:"status"`
			ContinuityScore *float64 `json:"continuity_score"`
			UrgencyScore    *float64 `json:"urgency_score"`
			Why             *string  `json:"why"`
			WhatRemains     *string  `json:"what_remains"`
			WhatNext        *string  `json:"what_next"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid JSON body", nil)
			return
		}

		if req.Title != nil {
			ws.Title = *req.Title
		}
		if req.Description != nil {
			ws.Description = *req.Description
		}
		if req.Status != nil {
			ws.Status = *req.Status
		}
		if req.ContinuityScore != nil {
			ws.ContinuityScore = *req.ContinuityScore
		}
		if req.UrgencyScore != nil {
			ws.UrgencyScore = *req.UrgencyScore
		}
		if req.Why != nil {
			ws.Why = *req.Why
		}
		if req.WhatRemains != nil {
			ws.WhatRemains = *req.WhatRemains
		}
		if req.WhatNext != nil {
			ws.WhatNext = *req.WhatNext
		}

		if err := UpdateWorkstream(db, ws); err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"workstream": ws,
		})

	case http.MethodDelete:
		if err := DeleteWorkstream(db, projectID, workstreamID); err != nil {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"deleted": true,
		})

	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete)
	}
}

// v2ProjectPlanningCyclesHandler handles GET /api/v2/projects/:id/planning/cycles
func v2ProjectPlanningCyclesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid cycles path", nil)
		return
	}
	projectID := parts[0]

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	cycles, err := ListPlanningCycles(db, projectID, limit)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}
	if cycles == nil {
		cycles = []PlanningCycle{}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"cycles": cycles,
		"total":  len(cycles),
	})
}

// v2ProjectDecisionsHandler handles GET /api/v2/projects/:id/planning/decisions
func v2ProjectDecisionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid decisions path", nil)
		return
	}
	projectID := parts[0]

	cycleID := r.URL.Query().Get("cycle_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	decisions, err := ListPlannerDecisions(db, projectID, cycleID, limit)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}
	if decisions == nil {
		decisions = []PlannerDecision{}
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"decisions": decisions,
		"total":     len(decisions),
	})
}

// v2ProjectPlanningQualityHandler handles GET /api/v2/projects/:id/planning/quality
func v2ProjectPlanningQualityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid path", nil)
		return
	}
	projectID := parts[0]

	metrics, err := ComputePlanningQuality(db, projectID)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{"quality": metrics})
}

// v2TaskHandoffHandler handles GET /api/v2/tasks/:id/handoff
func v2TaskHandoffHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid path", nil)
		return
	}
	taskID, err := strconv.Atoi(parts[0])
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid task ID", nil)
		return
	}

	hc, err := BuildHandoffContext(db, taskID)
	if err != nil {
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{"handoff": hc})
}

// v2ProjectSeedPromptsHandler handles POST /api/v2/projects/:id/planning/seed-prompts
func v2ProjectSeedPromptsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid path", nil)
		return
	}
	projectID := parts[0]

	if err := SeedBuiltInPrompts(db, projectID); err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	prompts, _ := ListPromptTemplates(db, projectID, "", true)
	if prompts == nil {
		prompts = []PromptTemplate{}
	}
	writeV2JSON(w, http.StatusOK, map[string]interface{}{"seeded": true, "prompts": prompts})
}
