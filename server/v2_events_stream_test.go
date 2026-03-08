package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func readSSEEvent(t *testing.T, reader *bufio.Reader) Event {
	t.Helper()

	var eventLine string
	var dataLine string

	for eventLine == "" || dataLine == "" {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("failed to read SSE line: %v", readErr)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventLine = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
		if strings.HasPrefix(line, "data:") {
			dataLine = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}

	var event Event
	if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
		t.Fatalf("failed to unmarshal event payload: %v", err)
	}
	if eventLine != "" && event.Kind != eventLine {
		t.Fatalf("expected event kind %q, got %q", eventLine, event.Kind)
	}
	return event
}

func TestV2EventsStreamHeadersAndFormat(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := server.URL + "/api/v2/events/stream?type=task.created&project_id=proj_default"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected Content-Type text/event-stream, got %q", contentType)
	}
	if cacheControl := resp.Header.Get("Cache-Control"); !strings.Contains(cacheControl, "no-cache") {
		t.Fatalf("expected Cache-Control no-cache, got %q", cacheControl)
	}

	otherProject, err := CreateProject(testDB, "Other", "/tmp/other", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	go func() {
		time.Sleep(25 * time.Millisecond)
		_, _ = CreateEvent(testDB, otherProject.ID, "task.created", "task", "101", nil)
		_, _ = CreateEvent(testDB, "proj_default", "task.updated", "task", "101", nil)
		_, _ = CreateEvent(testDB, "proj_default", "task.created", "task", "101", map[string]interface{}{"note": "hello"})
	}()

	reader := bufio.NewReader(resp.Body)
	var eventLine string
	var dataLine string

	for eventLine == "" || dataLine == "" {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("failed to read SSE line: %v", readErr)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventLine = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
		if strings.HasPrefix(line, "data:") {
			dataLine = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}

	if eventLine != "task.created" {
		t.Fatalf("expected event task.created, got %q", eventLine)
	}

	var event Event
	if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
		t.Fatalf("failed to unmarshal event payload: %v", err)
	}
	if event.Kind != "task.created" {
		t.Fatalf("expected event kind task.created, got %q", event.Kind)
	}
	if event.ProjectID != "proj_default" {
		t.Fatalf("expected project_id proj_default, got %q", event.ProjectID)
	}
}

func TestV2ProjectEventsStreamPath(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := server.URL + "/api/v2/projects/proj_default/events/stream?type=task.created"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	otherProject, err := CreateProject(testDB, "Other", "/tmp/other", nil)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	go func() {
		time.Sleep(25 * time.Millisecond)
		_, _ = CreateEvent(testDB, otherProject.ID, "task.created", "task", "201", nil)
		_, _ = CreateEvent(testDB, "proj_default", "task.created", "task", "201", map[string]interface{}{"note": "hello"})
	}()

	reader := bufio.NewReader(resp.Body)
	event := readSSEEvent(t, reader)
	if event.Kind != "task.created" {
		t.Fatalf("expected event kind task.created, got %q", event.Kind)
	}
	if event.ProjectID != "proj_default" {
		t.Fatalf("expected project_id proj_default, got %q", event.ProjectID)
	}
}

func TestV2EventsStreamAuthEnforcement(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	cfg := runtimeConfig{
		RequireAPIKey:      true,
		RequireAPIKeyReads: true,
		AuthIdentities: []authIdentity{
			{
				ID:     "reader",
				Type:   "service",
				APIKey: "reader-key",
				Scopes: map[string]struct{}{"v2:read": {}},
			},
			{
				ID:     "writer",
				Type:   "service",
				APIKey: "writer-key",
				Scopes: map[string]struct{}{"v2:write": {}},
			},
		},
	}

	mux := http.NewServeMux()
	registerRoutes(mux, cfg)

	t.Run("unauthorized_without_key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/events/stream", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 without key, got %d body=%s", w.Code, w.Body.String())
		}
		assertV2ErrorEnvelope(t, w, "UNAUTHORIZED")
	})

	t.Run("forbidden_without_scope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/events/stream", nil)
		req.Header.Set("X-API-Key", "writer-key")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 without scope, got %d body=%s", w.Code, w.Body.String())
		}
		errField := assertV2ErrorEnvelope(t, w, "FORBIDDEN")
		details := errField["details"].(map[string]interface{})
		if details["required_scope"] != "v2:read" {
			t.Fatalf("expected required_scope v2:read, got %v", details["required_scope"])
		}
	})

	t.Run("authorized_with_read_scope", func(t *testing.T) {
		server := httptest.NewServer(mux)
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v2/events/stream", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("X-API-Key", "reader-key")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to connect to stream: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 with read scope, got %d", resp.StatusCode)
		}
		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/event-stream") {
			t.Fatalf("expected Content-Type text/event-stream, got %q", contentType)
		}
	})
}

func TestV2EventsStreamSinceReplay(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	insertTestEvent(t, testDB, "evt_old", "proj_default", "task.created", "task", "201", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_replay", "proj_default", "task.updated", "task", "201", "2026-02-11T10:01:00.000000Z")

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := server.URL + "/api/v2/events/stream?project_id=proj_default&since=evt_replay"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	created := make(chan *Event, 1)
	go func() {
		time.Sleep(25 * time.Millisecond)
		event, _ := CreateEvent(testDB, "proj_default", "task.created", "task", "201", map[string]interface{}{"note": "live"})
		created <- event
	}()

	reader := bufio.NewReader(resp.Body)
	replayEvent := readSSEEvent(t, reader)
	if replayEvent.ID != "evt_replay" {
		t.Fatalf("expected replay event evt_replay, got %q", replayEvent.ID)
	}

	streamEvent := readSSEEvent(t, reader)
	liveEvent := <-created
	if liveEvent == nil {
		t.Fatalf("expected live event")
	}
	if streamEvent.ID != liveEvent.ID {
		t.Fatalf("expected stream event %q, got %q", liveEvent.ID, streamEvent.ID)
	}
}

func TestV2EventsStreamSinceReplayLimit(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	insertTestEvent(t, testDB, "evt_1", "proj_default", "task.created", "task", "301", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_2", "proj_default", "task.updated", "task", "301", "2026-02-11T10:01:00.000000Z")
	insertTestEvent(t, testDB, "evt_3", "proj_default", "task.completed", "task", "301", "2026-02-11T10:02:00.000000Z")

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := server.URL + "/api/v2/events/stream?project_id=proj_default&since=2026-02-11T10:00:00.000000Z&limit=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	replayEvent := readSSEEvent(t, bufio.NewReader(resp.Body))
	if replayEvent.ID != "evt_3" {
		t.Fatalf("expected replay event evt_3, got %q", replayEvent.ID)
	}
}

func TestV2EventsStreamReplayLimitMaxOverride(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	cfg := runtimeConfig{SSEReplayLimitMax: 2}

	mux := http.NewServeMux()
	registerRoutes(mux, cfg)

	insertTestEvent(t, testDB, "evt_1", "proj_default", "task.created", "task", "401", "2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_2", "proj_default", "task.updated", "task", "401", "2026-02-11T10:01:00.000000Z")
	insertTestEvent(t, testDB, "evt_3", "proj_default", "task.completed", "task", "401", "2026-02-11T10:02:00.000000Z")

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := server.URL + "/api/v2/events/stream?project_id=proj_default&since=2026-02-11T10:00:00.000000Z&limit=10"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	firstReplay := readSSEEvent(t, reader)
	if firstReplay.ID != "evt_2" {
		t.Fatalf("expected replay event evt_2, got %q", firstReplay.ID)
	}
	secondReplay := readSSEEvent(t, reader)
	if secondReplay.ID != "evt_3" {
		t.Fatalf("expected replay event evt_3, got %q", secondReplay.ID)
	}
}

func TestV2EventsStreamInvalidSince(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/events/stream?since=not-a-time", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
	}
	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2EventsStreamInvalidLimit(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	tests := []string{
		"/api/v2/events/stream?since=2026-02-11T10:00:00.000000Z&limit=0",
		"/api/v2/events/stream?since=2026-02-11T10:00:00.000000Z&limit=-1",
		"/api/v2/events/stream?since=2026-02-11T10:00:00.000000Z&limit=bogus",
	}

	for _, path := range tests {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", w.Code, w.Body.String())
		}
		assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
	}
}
