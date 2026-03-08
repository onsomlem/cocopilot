package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// API v2 Agent Handlers
// ============================================================================

func v2RegisterAgentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeV2MethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		Name         string                 `json:"name"`
		Capabilities []string               `json:"capabilities"`
		Metadata     map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body", map[string]interface{}{
			"content_type": r.Header.Get("Content-Type"),
		})
		return
	}

	if req.Name == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Agent name is required", nil)
		return
	}

	agent, err := RegisterAgent(db, req.Name, req.Capabilities, req.Metadata)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"agent_name": req.Name})
		return
	}

	CreateEvent(db, DefaultProjectID, "agent.registered", "agent", agent.ID, map[string]interface{}{
		"name": agent.Name,
	})

	writeV2JSON(w, http.StatusCreated, map[string]interface{}{
		"agent": agent,
	})
}

func v2ListAgentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	query := r.URL.Query()
	statusFilter := ""
	if rawValues, ok := query["status"]; ok {
		rawStatus := ""
		if len(rawValues) > 0 {
			rawStatus = strings.TrimSpace(rawValues[0])
		}
		if rawStatus == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "status cannot be empty", map[string]interface{}{
				"status": rawStatus,
			})
			return
		}
		statusFilter = strings.ToLower(rawStatus)
		if statusFilter != "active" && statusFilter != "stale" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "status must be active or stale", map[string]interface{}{
				"status": rawStatus,
			})
			return
		}
	}

	since := ""
	if rawValues, ok := query["since"]; ok {
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
		since = parsed.UTC().Format(leaseTimeFormat)
	}

	limit := v2AgentListDefaultLimit
	offset := 0
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
				"limit": rawLimit,
			})
			return
		}
		if parsed > v2AgentListMaxLimit {
			limit = v2AgentListMaxLimit
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

	sortField := "created_at"
	sortDirection := "asc"
	if rawSort := strings.TrimSpace(query.Get("sort")); rawSort != "" {
		switch rawSort {
		case "created_at":
			sortField = "created_at"
			sortDirection = "asc"
		case "last_seen:asc":
			sortField = "last_seen"
			sortDirection = "asc"
		case "last_seen:desc":
			sortField = "last_seen"
			sortDirection = "desc"
		default:
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "sort must be created_at, last_seen:asc, or last_seen:desc", map[string]interface{}{
				"sort": rawSort,
			})
			return
		}
	}

	agents, total, err := ListAgents(db, statusFilter, since, limit, offset, sortField, sortDirection)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"total":  total,
	})
}

func v2GetAgentHandler(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	agent, err := GetAgent(db, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Agent not found", map[string]interface{}{"agent_id": agentID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"agent_id": agentID})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"agent": agent,
	})
}

func v2DeleteAgentHandler(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodDelete {
		writeV2MethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	agent, err := DeleteAgent(db, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Agent not found", map[string]interface{}{"agent_id": agentID})
			return
		}
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"agent_id": agentID})
		return
	}

	CreateEvent(db, DefaultProjectID, "agent.deleted", "agent", agentID, map[string]interface{}{
		"name": agent.Name,
	})

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"agent": agent,
	})
}

func v2AgentActionHandler(w http.ResponseWriter, r *http.Request) {
	// Extract agent ID from path: /api/v2/agents/:id/action
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/agents/")
	if path == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid agent endpoint format", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	parts := strings.Split(path, "/")
	agentID := parts[0]
	if agentID == "" {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid agent endpoint format", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	if len(parts) == 1 || (len(parts) == 2 && parts[1] == "") {
		switch r.Method {
		case http.MethodGet:
			v2GetAgentHandler(w, r, agentID)
		case http.MethodDelete:
			v2DeleteAgentHandler(w, r, agentID)
		default:
			writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodDelete)
		}
		return
	}
	if len(parts) != 2 {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid agent endpoint format", map[string]interface{}{
			"path": r.URL.Path,
		})
		return
	}

	action := parts[1]

	switch action {
	case "heartbeat":
		if r.Method != http.MethodPost {
			writeV2MethodNotAllowed(w, r, http.MethodPost)
			return
		}

		err := UpdateAgentHeartbeat(db, agentID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Agent not found", map[string]interface{}{"agent_id": agentID})
				return
			}
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{"agent_id": agentID})
			return
		}

		writeV2JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Heartbeat updated",
		})
	case "capabilities":
		switch r.Method {
		case http.MethodGet:
			agent, err := GetAgent(db, agentID)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Agent not found", map[string]interface{}{"agent_id": agentID})
					return
				}
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
				return
			}
			caps := agent.Capabilities
			if caps == nil {
				caps = []string{}
			}
			writeV2JSON(w, http.StatusOK, map[string]interface{}{
				"agent_id":     agentID,
				"capabilities": caps,
			})
		case http.MethodPut:
			var req struct {
				Capabilities []string `json:"capabilities"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeV2Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
				return
			}
			agent, err := UpdateAgentCapabilities(db, agentID, req.Capabilities)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Agent not found", map[string]interface{}{"agent_id": agentID})
					return
				}
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
				return
			}
			writeV2JSON(w, http.StatusOK, map[string]interface{}{
				"agent": agent,
			})
		default:
			writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPut)
		}
	default:
		writeV2Error(w, http.StatusNotFound, "NOT_FOUND", "Unknown agent action: "+action, map[string]interface{}{
			"agent_id": agentID,
			"action":   action,
		})
	}
}
