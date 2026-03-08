package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// readSSELine reads a single non-empty line from the SSE stream, skipping blank lines.
func readSSELine(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("readSSELine: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			return line
		}
	}
}

// readSSEEventOrComment reads the next SSE message: either a comment (": ...") or
// a full event with id/event/data fields. Returns (isComment, comment, event).
func readSSEEventOrComment(t *testing.T, reader *bufio.Reader) (bool, string, Event) {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("readSSEEventOrComment: read error: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		// comment line like ": ready" or ": ping"
		if strings.HasPrefix(line, ":") {
			return true, strings.TrimSpace(strings.TrimPrefix(line, ":")), Event{}
		}
		// Start of an event frame — read event: and data: lines
		var eventKind, dataStr string
		if strings.HasPrefix(line, "id:") {
			// got id line, read event and data
		} else if strings.HasPrefix(line, "event:") {
			eventKind = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataStr = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		// Read remaining lines of this frame
		for eventKind == "" || dataStr == "" {
			nextLine, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("readSSEEventOrComment: read error mid-frame: %v", err)
			}
			nextLine = strings.TrimRight(nextLine, "\r\n")
			if nextLine == "" {
				break // end of frame
			}
			if strings.HasPrefix(nextLine, "event:") {
				eventKind = strings.TrimSpace(strings.TrimPrefix(nextLine, "event:"))
			}
			if strings.HasPrefix(nextLine, "data:") {
				dataStr = strings.TrimSpace(strings.TrimPrefix(nextLine, "data:"))
			}
		}
		if dataStr == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(dataStr), &ev); err != nil {
			t.Fatalf("readSSEEventOrComment: unmarshal: %v", err)
		}
		return false, "", ev
	}
}

// ============================================================================
// Test 1: SSE heartbeat arrives at the configured interval
// ============================================================================

func TestSSE_HeartbeatInterval(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	// Use a very short heartbeat for testing (100ms is below the config min,
	// but v2EventsStreamHandler accepts the duration directly).
	cfg := runtimeConfig{SSEHeartbeatSeconds: 5} // use config; actual test uses handler directly
	registerRoutes(mux, cfg)

	// Instead of going through registerRoutes with constrained config,
	// create a standalone handler with a 100ms heartbeat.
	heartbeatHandler := v2EventsStreamHandler(100*time.Millisecond, 500)

	// Wrap in a mux that also initializes the project.
	testMux := http.NewServeMux()
	testMux.HandleFunc("/api/v2/events/stream", heartbeatHandler)

	server := httptest.NewServer(testMux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v2/events/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	// Read the ": ready" comment first
	firstLine := readSSELine(t, reader)
	if !strings.HasPrefix(firstLine, ":") {
		t.Fatalf("expected ready comment, got %q", firstLine)
	}

	// Now wait for heartbeat pings. We should get at least 2 within 500ms.
	pingCount := 0
	start := time.Now()
	deadline := start.Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && pingCount < 2 {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, ": ping") {
			pingCount++
		}
	}
	if pingCount < 2 {
		t.Errorf("expected at least 2 heartbeat pings in 500ms, got %d", pingCount)
	}
}

// ============================================================================
// Test 2: SSE replay with since (Last-Event-ID equivalent) replays only after
// ============================================================================

func TestSSE_ReplayWithSince(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "sse-replay-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Insert events with known timestamps
	insertTestEvent(t, testDB, "evt_r1", project.ID, "task.created", "task", "1",
		"2026-02-11T10:00:00.000000Z")
	insertTestEvent(t, testDB, "evt_r2", project.ID, "task.updated", "task", "1",
		"2026-02-11T10:01:00.000000Z")
	insertTestEvent(t, testDB, "evt_r3", project.ID, "task.completed", "task", "1",
		"2026-02-11T10:02:00.000000Z")
	insertTestEvent(t, testDB, "evt_r4", project.ID, "task.created", "task", "2",
		"2026-02-11T10:03:00.000000Z")

	// Connect with since=evt_r2's timestamp. The SSE handler uses resolveEventStreamSince
	// which accepts event IDs, resolving to the created_at of that event.
	handler := v2EventsStreamHandler(10*time.Second, 500)

	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use event ID as since value (resolveEventStreamSince supports this)
	url := server.URL + fmt.Sprintf("?project_id=%s&since=evt_r2", project.ID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)

	// Skip ": ready"
	readSSELine(t, reader)

	// ListEvents returns DESC order, then the handler iterates in reverse (i.e. ASC).
	// Events with created_at >= evt_r2's timestamp: evt_r2, evt_r3, evt_r4.
	// After reverse iteration: evt_r2 first, evt_r3, evt_r4.
	var replayedEvents []Event
	for i := 0; i < 3; i++ {
		ev := readSSEEvent(t, reader)
		replayedEvents = append(replayedEvents, ev)
	}

	if len(replayedEvents) != 3 {
		t.Fatalf("expected 3 replayed events, got %d", len(replayedEvents))
	}

	// Verify the events are in chronological order (oldest first due to reverse iteration)
	expectedIDs := []string{"evt_r2", "evt_r3", "evt_r4"}
	for i, id := range expectedIDs {
		if replayedEvents[i].ID != id {
			t.Errorf("event[%d]: expected ID %s, got %s", i, id, replayedEvents[i].ID)
		}
	}

	// evt_r1 should NOT be replayed (it's before evt_r2)
	for _, ev := range replayedEvents {
		if ev.ID == "evt_r1" {
			t.Error("evt_r1 should not be in replayed events (before since point)")
		}
	}
}

// ============================================================================
// Test 3: SSE replay limit is respected
// ============================================================================

func TestSSE_ReplayLimit(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "sse-limit-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Insert 10 events
	for i := 0; i < 10; i++ {
		ts := fmt.Sprintf("2026-02-11T10:%02d:00.000000Z", i)
		insertTestEvent(t, testDB, fmt.Sprintf("evt_lim_%d", i), project.ID,
			"task.updated", "task", "1", ts)
	}

	// Create handler with replayLimitMax=3
	handler := v2EventsStreamHandler(10*time.Second, 3)

	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// since=evt_lim_0 means all 10 events qualify, but limit=3 should cap replay
	url := server.URL + fmt.Sprintf("?project_id=%s&since=evt_lim_0", project.ID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	readSSELine(t, reader) // skip ": ready"

	// Should get at most 3 events (the replay limit)
	var replayedEvents []Event
	for i := 0; i < 3; i++ {
		ev := readSSEEvent(t, reader)
		replayedEvents = append(replayedEvents, ev)
	}

	if len(replayedEvents) != 3 {
		t.Fatalf("expected exactly 3 replayed events (limit), got %d", len(replayedEvents))
	}

	// The handler calls ListEvents with limit=3, which returns DESC, then iterates reverse.
	// ListEvents DESC LIMIT 3 gives the 3 most recent events.
	// After reverse: oldest of the 3 first.
}

// ============================================================================
// Test 4: SSE replay limit from query param capped by server max
// ============================================================================

func TestSSE_ReplayLimitQueryParamCap(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "sse-cap-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	for i := 0; i < 10; i++ {
		ts := fmt.Sprintf("2026-02-11T10:%02d:00.000000Z", i)
		insertTestEvent(t, testDB, fmt.Sprintf("evt_cap_%d", i), project.ID,
			"task.updated", "task", "1", ts)
	}

	// Server max is 5, but client requests limit=100
	handler := v2EventsStreamHandler(10*time.Second, 5)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := server.URL + fmt.Sprintf("?project_id=%s&since=evt_cap_0&limit=100", project.ID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	readSSELine(t, reader) // skip ": ready"

	var replayedEvents []Event
	for i := 0; i < 5; i++ {
		ev := readSSEEvent(t, reader)
		replayedEvents = append(replayedEvents, ev)
	}

	if len(replayedEvents) != 5 {
		t.Fatalf("expected 5 replayed events (server max), got %d", len(replayedEvents))
	}
}

// ============================================================================
// Test 5: Multiple concurrent SSE connections each receive events
// ============================================================================

func TestSSE_MultipleConcurrentConnections(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "sse-multi-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	handler := v2EventsStreamHandler(10*time.Second, 500)
	server := httptest.NewServer(handler)
	defer server.Close()

	const numClients = 3
	type clientResult struct {
		events []Event
		err    error
	}
	results := make([]clientResult, numClients)
	var wg sync.WaitGroup

	// Connect multiple SSE clients in parallel
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			url := server.URL + fmt.Sprintf("?project_id=%s", project.ID)
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results[idx] = clientResult{err: fmt.Errorf("connect: %v", err)}
				return
			}
			defer resp.Body.Close()

			reader := bufio.NewReader(resp.Body)
			// Read ": ready"
			_, _ = reader.ReadString('\n')

			// Read the event that will be published
			ev := readSSEEvent(t, reader)
			results[idx] = clientResult{events: []Event{ev}}
		}(i)
	}

	// Wait a moment for all clients to connect, then publish an event
	time.Sleep(100 * time.Millisecond)
	_, err = CreateEvent(testDB, project.ID, "task.created", "task", "42", map[string]interface{}{"note": "broadcast"})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	// Broadcast via publishV2Event
	publishV2Event(Event{
		ID:        "evt_broadcast",
		ProjectID: project.ID,
		Kind:      "task.created",
		Payload:   map[string]interface{}{"note": "broadcast"},
	})

	wg.Wait()

	// Verify all clients received the event
	for i, res := range results {
		if res.err != nil {
			t.Errorf("client %d error: %v", i, res.err)
			continue
		}
		if len(res.events) == 0 {
			t.Errorf("client %d: expected at least 1 event, got 0", i)
			continue
		}
		if res.events[0].Kind != "task.created" {
			t.Errorf("client %d: expected task.created event, got %s", i, res.events[0].Kind)
		}
	}
}

// ============================================================================
// Test 6: SSE project-scoped stream filters by project
// ============================================================================

func TestSSE_ProjectScopedFilter(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	projA, err := CreateProject(testDB, "sse-proj-A", "/tmp/a", nil)
	if err != nil {
		t.Fatalf("CreateProject A: %v", err)
	}
	projB, err := CreateProject(testDB, "sse-proj-B", "/tmp/b", nil)
	if err != nil {
		t.Fatalf("CreateProject B: %v", err)
	}

	handler := v2EventsStreamHandler(10*time.Second, 500)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Subscribe only to projA events
	url := server.URL + fmt.Sprintf("?project_id=%s", projA.ID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	readSSELine(t, reader) // ": ready"

	// Wait for connection to register
	time.Sleep(50 * time.Millisecond)

	// Publish event to projB (should NOT be received)
	publishV2Event(Event{
		ID: "evt_projB", ProjectID: projB.ID, Kind: "task.created",
	})

	// Publish event to projA (SHOULD be received)
	publishV2Event(Event{
		ID: "evt_projA", ProjectID: projA.ID, Kind: "task.created",
		Payload: map[string]interface{}{"project": "A"},
	})

	ev := readSSEEvent(t, reader)
	if ev.ID != "evt_projA" {
		t.Errorf("expected event from projA, got ID=%s", ev.ID)
	}
}

// ============================================================================
// Test 7: PruneEvents with retention days
// ============================================================================

func TestPruneEvents_RetentionDays(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "prune-days-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -45).Format(leaseTimeFormat) // 45 days ago
	recent := now.AddDate(0, 0, -5).Format(leaseTimeFormat) // 5 days ago

	// Insert old events
	for i := 0; i < 4; i++ {
		insertTestEvent(t, testDB, fmt.Sprintf("evt_old_%d", i), project.ID,
			"task.created", "task", fmt.Sprintf("%d", i), old)
	}
	// Insert recent events
	for i := 0; i < 3; i++ {
		insertTestEvent(t, testDB, fmt.Sprintf("evt_recent_%d", i), project.ID,
			"task.created", "task", fmt.Sprintf("r%d", i), recent)
	}

	// Prune with 30-day retention — should delete the 4 old events
	deleted, err := PruneEvents(testDB, 30, 0)
	if err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}
	if deleted != 4 {
		t.Errorf("expected 4 deleted, got %d", deleted)
	}

	var remaining int
	testDB.QueryRow("SELECT COUNT(*) FROM events").Scan(&remaining)
	if remaining != 3 {
		t.Errorf("expected 3 remaining, got %d", remaining)
	}
}

// ============================================================================
// Test 8: PruneEvents with max rows
// ============================================================================

func TestPruneEvents_MaxRows(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "prune-max-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// Insert 10 events
	for i := 0; i < 10; i++ {
		ts := fmt.Sprintf("2026-02-11T10:%02d:00.000000Z", i)
		insertTestEvent(t, testDB, fmt.Sprintf("evt_max_%d", i), project.ID,
			"task.created", "task", fmt.Sprintf("%d", i), ts)
	}

	// Prune to keep max 4 rows
	deleted, err := PruneEvents(testDB, 0, 4)
	if err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}
	if deleted != 6 {
		t.Errorf("expected 6 deleted, got %d", deleted)
	}

	var remaining int
	testDB.QueryRow("SELECT COUNT(*) FROM events").Scan(&remaining)
	if remaining != 4 {
		t.Errorf("expected 4 remaining, got %d", remaining)
	}

	// Verify the kept events are the 4 most recent (highest created_at)
	rows, err := testDB.Query("SELECT id FROM events ORDER BY created_at DESC")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var keptIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		keptIDs = append(keptIDs, id)
	}
	// Most recent should be evt_max_9, evt_max_8, evt_max_7, evt_max_6
	for i, expected := range []string{"evt_max_9", "evt_max_8", "evt_max_7", "evt_max_6"} {
		if i >= len(keptIDs) || keptIDs[i] != expected {
			got := "<missing>"
			if i < len(keptIDs) {
				got = keptIDs[i]
			}
			t.Errorf("kept[%d]: expected %s, got %s", i, expected, got)
		}
	}
}

// ============================================================================
// Test 9: PruneEvents with both retention days AND max rows
// ============================================================================

func TestPruneEvents_Combined(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "prune-combo-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -60).Format(leaseTimeFormat) // 60 days old
	recent := now.AddDate(0, 0, -2).Format(leaseTimeFormat) // 2 days old

	// 3 old events, 5 recent events = 8 total
	for i := 0; i < 3; i++ {
		insertTestEvent(t, testDB, fmt.Sprintf("evt_combo_old_%d", i), project.ID,
			"task.created", "task", fmt.Sprintf("o%d", i), old)
	}
	for i := 0; i < 5; i++ {
		insertTestEvent(t, testDB, fmt.Sprintf("evt_combo_new_%d", i), project.ID,
			"task.created", "task", fmt.Sprintf("n%d", i), recent)
	}

	// Prune: 30-day retention + max 3 rows.
	// First pass deletes 3 old events, then second pass trims to 3 (deletes 2 recent).
	deleted, err := PruneEvents(testDB, 30, 3)
	if err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}
	if deleted != 5 { // 3 old + 2 excess recent
		t.Errorf("expected 5 deleted, got %d", deleted)
	}

	var remaining int
	testDB.QueryRow("SELECT COUNT(*) FROM events").Scan(&remaining)
	if remaining != 3 {
		t.Errorf("expected 3 remaining, got %d", remaining)
	}
}

// ============================================================================
// Test 10: PruneEvents edge case — zero retention and zero maxRows = no-op
// ============================================================================

func TestPruneEvents_NoOp(t *testing.T) {
	testDB, cleanup := setupV2TestDB(t)
	defer cleanup()

	project, err := CreateProject(testDB, "prune-noop-proj", "/tmp/test", nil)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	insertTestEvent(t, testDB, "evt_noop", project.ID,
		"task.created", "task", "1", "2026-02-11T10:00:00.000000Z")

	deleted, err := PruneEvents(testDB, 0, 0)
	if err != nil {
		t.Fatalf("PruneEvents: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted with 0/0 config, got %d", deleted)
	}

	var remaining int
	testDB.QueryRow("SELECT COUNT(*) FROM events").Scan(&remaining)
	if remaining != 1 {
		t.Errorf("expected 1 remaining, got %d", remaining)
	}
}

// ============================================================================
// Test 11: SSE type filter — only matching event types are delivered
// ============================================================================

func TestSSE_TypeFilter(t *testing.T) {
	_, cleanup := setupV2TestDB(t)
	defer cleanup()

	handler := v2EventsStreamHandler(10*time.Second, 500)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Subscribe only to task.completed events
	url := server.URL + "?type=task.completed"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	readSSELine(t, reader) // ": ready"

	time.Sleep(50 * time.Millisecond)

	// Publish a task.created event (should be filtered out)
	publishV2Event(Event{ID: "evt_created", Kind: "task.created"})

	// Publish a task.completed event (should be delivered)
	publishV2Event(Event{ID: "evt_completed", Kind: "task.completed",
		Payload: map[string]interface{}{"done": true}})

	ev := readSSEEvent(t, reader)
	if ev.ID != "evt_completed" {
		t.Errorf("expected evt_completed, got %s", ev.ID)
	}
	if ev.Kind != "task.completed" {
		t.Errorf("expected task.completed kind, got %s", ev.Kind)
	}
}
