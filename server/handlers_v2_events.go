package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/onsomlem/cocopilot/internal/config"
)

// ============================================================================
// API v2 Event Handlers
// ============================================================================

const (
	v2EventsListDefaultLimit = config.V2EventsListDefaultLimit
	v2EventsListMaxLimit     = config.V2EventsListMaxLimit
)

func normalizeEventSince(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format(leaseTimeFormat), nil
}

func resolveEventStreamSince(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed.UTC().Format(leaseTimeFormat), nil
	}
	createdAt, err := GetEventCreatedAtByID(db, raw)
	if err != nil {
		return "", err
	}
	return createdAt, nil
}

func v2ListEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeV2MethodNotAllowed(w, r, http.MethodGet)
		return
	}

	query := r.URL.Query()
	kind := strings.TrimSpace(query.Get("type"))
	taskIDRaw := strings.TrimSpace(query.Get("task_id"))
	projectID := ""
	if rawValues, ok := query["project_id"]; ok {
		rawProject := ""
		if len(rawValues) > 0 {
			rawProject = strings.TrimSpace(rawValues[0])
		}
		if rawProject == "" {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id cannot be empty", map[string]interface{}{
				"project_id": rawProject,
			})
			return
		}
		projectID = rawProject
		if projectID != "" {
			if _, err := GetProject(db, projectID); err != nil {
				if strings.Contains(err.Error(), "not found") {
					writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id not found", map[string]interface{}{
						"project_id": projectID,
					})
					return
				}
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
					"project_id": projectID,
				})
				return
			}
		}
	}

	limit := v2EventsListDefaultLimit
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
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

	offset := 0
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

	sinceRaw := strings.TrimSpace(query.Get("since"))
	since, err := normalizeEventSince(sinceRaw)
	if err != nil {
		writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since must be RFC3339", map[string]interface{}{
			"since": sinceRaw,
		})
		return
	}

	taskID := ""
	if taskIDRaw != "" {
		parsed, err := strconv.Atoi(taskIDRaw)
		if err != nil || parsed <= 0 {
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "task_id must be a positive integer", map[string]interface{}{
				"task_id": taskIDRaw,
			})
			return
		}
		taskID = strconv.Itoa(parsed)
	}

	events, total, err := ListEvents(db, projectID, kind, since, taskID, limit, offset)
	if err != nil {
		writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
			"project_id": projectID,
			"type":       kind,
			"since":      sinceRaw,
			"limit":      limit,
			"offset":     offset,
		})
		return
	}

	writeV2JSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
	})
}

func v2EventsStreamHandler(heartbeatInterval time.Duration, replayLimitMax int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeV2MethodNotAllowed(w, r, http.MethodGet)
			return
		}

		query := r.URL.Query()
		kind := strings.TrimSpace(query.Get("type"))
		sinceRaw := strings.TrimSpace(query.Get("since"))
		projectID := ""
		if rawValues, ok := query["project_id"]; ok {
			rawProject := ""
			if len(rawValues) > 0 {
				rawProject = strings.TrimSpace(rawValues[0])
			}
			if rawProject == "" {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id cannot be empty", map[string]interface{}{
					"project_id": rawProject,
				})
				return
			}
			projectID = rawProject
			if projectID != "" {
				if _, err := GetProject(db, projectID); err != nil {
					if strings.Contains(err.Error(), "not found") {
						writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "project_id not found", map[string]interface{}{
							"project_id": projectID,
						})
						return
					}
					writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
						"project_id": projectID,
					})
					return
				}
			}
		}

		replayLimit := replayLimitMax
		if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed <= 0 {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "limit must be a positive integer", map[string]interface{}{
					"limit": rawLimit,
				})
				return
			}
			if parsed > replayLimitMax {
				replayLimit = replayLimitMax
			} else {
				replayLimit = parsed
			}
		}

		since, err := resolveEventStreamSince(sinceRaw)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since must be RFC3339 or event id", map[string]interface{}{
					"since": sinceRaw,
				})
				return
			}
			writeV2Error(w, http.StatusBadRequest, "INVALID_ARGUMENT", "since must be RFC3339 or event id", map[string]interface{}{
				"since": sinceRaw,
			})
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeV2Error(w, http.StatusInternalServerError, "INTERNAL", "SSE not supported", nil)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		fmt.Fprintf(w, ": ready\n\n")
		flusher.Flush()

		if since != "" {
			replay, _, err := ListEvents(db, projectID, kind, since, "", replayLimit, 0)
			if err != nil {
				writeV2Error(w, http.StatusInternalServerError, "INTERNAL", err.Error(), map[string]interface{}{
					"project_id": projectID,
					"type":       kind,
					"since":      sinceRaw,
				})
				return
			}
			for i := len(replay) - 1; i >= 0; i-- {
				select {
				case <-r.Context().Done():
					return
				default:
				}
				data, err := json.Marshal(replay[i])
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "id: %s\n", replay[i].ID)
				fmt.Fprintf(w, "event: %s\n", replay[i].Kind)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}

		clientChan := make(chan Event, 10)
		subscriber := v2EventSubscriber{
			ch:        clientChan,
			projectID: projectID,
			kind:      kind,
		}
		v2EventMu.Lock()
		v2EventSubscribers = append(v2EventSubscribers, subscriber)
		v2EventMu.Unlock()

		defer func() {
			v2EventMu.Lock()
			for i, entry := range v2EventSubscribers {
				if entry.ch == clientChan {
					v2EventSubscribers = append(v2EventSubscribers[:i], v2EventSubscribers[i+1:]...)
					break
				}
			}
			v2EventMu.Unlock()
			close(clientChan)
		}()

		heartbeat := time.NewTicker(heartbeatInterval)
		defer heartbeat.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-clientChan:
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "id: %s\n", event.ID)
				fmt.Fprintf(w, "event: %s\n", event.Kind)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-heartbeat.C:
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	}
}
