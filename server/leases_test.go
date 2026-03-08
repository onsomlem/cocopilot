package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupLeaseTestDB(t *testing.T) *sql.DB {
	// Use a temporary file instead of :memory: for better concurrency support
	dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_%d.db", time.Now().UnixNano()))
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable WAL mode and busy timeout to match production settings and
	// avoid "cannot start a transaction within a transaction" under concurrency.
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA busy_timeout=5000")

	// Clean up the test database file when the test completes
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	// Run all migrations to set up tables
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a test task
	_, err = db.Exec(`
		INSERT INTO tasks (id, title, instructions, status, created_at, updated_at)
		VALUES (1, 'Test Task', 'Test instructions', 'NOT_PICKED', '2024-01-01T00:00:00.000Z', '2024-01-01T00:00:00.000Z')
	`)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	return db
}

func TestLeaseCreation(t *testing.T) {
	db := setupLeaseTestDB(t)
	defer db.Close()

	// Test creating a lease
	lease, err := CreateLease(db, 1, "test-agent", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}

	if lease == nil {
		t.Fatal("Expected lease to be created, got nil")
	}

	if lease.TaskID != 1 {
		t.Errorf("Expected task ID 1, got %d", lease.TaskID)
	}

	if lease.AgentID != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got %s", lease.AgentID)
	}

	if lease.Mode != "exclusive" {
		t.Errorf("Expected mode 'exclusive', got %s", lease.Mode)
	}

	// Verify lease was saved in database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM leases WHERE task_id = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query lease count: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 lease in database, got %d", count)
	}
}

func TestLeaseCreationDuplicate(t *testing.T) {
	db := setupLeaseTestDB(t)
	defer db.Close()

	// Create first lease
	_, err := CreateLease(db, 1, "agent1", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create first lease: %v", err)
	}

	// Try to create second lease for same task - should fail
	_, err = CreateLease(db, 1, "agent2", "exclusive")
	if err == nil {
		t.Fatal("Expected error when creating duplicate lease, got nil")
	}
}

func TestGetLeaseByID(t *testing.T) {
	db := setupLeaseTestDB(t)
	defer db.Close()

	// Create a lease
	originalLease, err := CreateLease(db, 1, "test-agent", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}

	// Get lease by ID
	retrievedLease, err := GetLeaseByID(db, originalLease.ID)
	if err != nil {
		t.Fatalf("Failed to get lease by ID: %v", err)
	}

	if retrievedLease == nil {
		t.Fatal("Expected lease to be found, got nil")
	}

	if retrievedLease.ID != originalLease.ID {
		t.Errorf("Expected lease ID %s, got %s", originalLease.ID, retrievedLease.ID)
	}

	if retrievedLease.TaskID != originalLease.TaskID {
		t.Errorf("Expected task ID %d, got %d", originalLease.TaskID, retrievedLease.TaskID)
	}
}

func TestGetLeaseByIDNotFound(t *testing.T) {
	db := setupLeaseTestDB(t)
	defer db.Close()

	lease, err := GetLeaseByID(db, "non-existent-lease")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if lease != nil {
		t.Errorf("Expected nil lease for non-existent ID, got %+v", lease)
	}
}

func TestExtendLease(t *testing.T) {
	db := setupLeaseTestDB(t)
	defer db.Close()

	// Create a lease
	lease, err := CreateLease(db, 1, "test-agent", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}

	// Parse original expiration time
	originalExpiresAt, err := time.Parse("2006-01-02T15:04:05.999999Z", lease.ExpiresAt)
	if err != nil {
		t.Fatalf("Failed to parse original expiration time: %v", err)
	}

	// Extend lease by 30 minutes
	err = ExtendLease(db, lease.ID, 30*time.Minute)
	if err != nil {
		t.Fatalf("Failed to extend lease: %v", err)
	}

	// Get updated lease
	updatedLease, err := GetLeaseByID(db, lease.ID)
	if err != nil {
		t.Fatalf("Failed to get updated lease: %v", err)
	}

	// Parse new expiration time
	newExpiresAt, err := time.Parse("2006-01-02T15:04:05.999999Z", updatedLease.ExpiresAt)
	if err != nil {
		t.Fatalf("Failed to parse new expiration time: %v", err)
	}

	// New expiration should be significantly later than original (at least 20 minutes)
	if newExpiresAt.Before(originalExpiresAt.Add(20 * time.Minute)) {
		t.Errorf("Expected lease to be extended significantly. Original: %v, New: %v",
			originalExpiresAt, newExpiresAt)
	}
}

func TestLeaseExpiredDeletion(t *testing.T) {
	db := setupLeaseTestDB(t)
	defer db.Close()

	// Create tasks for testing
	_, err := db.Exec(`
		INSERT INTO tasks (id, title, instructions, status, created_at, updated_at)
		VALUES (2, 'Test Task 2', 'Test instructions 2', 'NOT_PICKED', '2024-01-01T00:00:00.000Z', '2024-01-01T00:00:00.000Z')
	`)
	if err != nil {
		t.Fatalf("Failed to create test task 2: %v", err)
	}

	// Create an expired lease (manually insert with past expiration)
	expiredTime := time.Now().UTC().Add(-1 * time.Hour).Format("2006-01-02T15:04:05.999999Z")
	_, err = db.Exec(`
		INSERT INTO leases (id, task_id, agent_id, mode, created_at, expires_at)
		VALUES ('expired-lease', 1, 'agent1', 'exclusive', ?, ?)
	`, nowISO(), expiredTime)
	if err != nil {
		t.Fatalf("Failed to create expired lease: %v", err)
	}

	// Create a valid lease
	_, err = CreateLease(db, 2, "agent2", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create valid lease: %v", err)
	}

	// Delete expired leases
	count, err := DeleteExpiredLeases(db)
	if err != nil {
		t.Fatalf("Failed to delete expired leases: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 expired lease deleted, got %d", count)
	}

	// Verify only valid lease remains
	var remaining int
	err = db.QueryRow("SELECT COUNT(*) FROM leases").Scan(&remaining)
	if err != nil {
		t.Fatalf("Failed to count remaining leases: %v", err)
	}

	if remaining != 1 {
		t.Errorf("Expected 1 remaining lease, got %d", remaining)
	}
}

func TestLeaseAPICreateLease(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	// Set the global db for the handlers
	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	req := map[string]interface{}{
		"task_id":  1,
		"agent_id": "test-agent",
		"mode":     "exclusive",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/v2/leases", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2CreateLeaseHandler(w, httpReq)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response Lease
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TaskID != 1 {
		t.Errorf("Expected task ID 1, got %d", response.TaskID)
	}

	if response.AgentID != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got %s", response.AgentID)
	}
}

func TestLeaseAPICreateDuplicateLease(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	// Set the global db for the handlers
	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	// Create first lease
	req1 := map[string]interface{}{
		"task_id":  1,
		"agent_id": "agent1",
		"mode":     "exclusive",
	}

	body1, _ := json.Marshal(req1)
	httpReq1 := httptest.NewRequest("POST", "/api/v2/leases", bytes.NewReader(body1))
	httpReq1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()

	v2CreateLeaseHandler(w1, httpReq1)

	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200 for first lease, got %d", w1.Code)
	}

	// Try to create duplicate lease
	req2 := map[string]interface{}{
		"task_id":  1,
		"agent_id": "agent2",
		"mode":     "exclusive",
	}

	body2, _ := json.Marshal(req2)
	httpReq2 := httptest.NewRequest("POST", "/api/v2/leases", bytes.NewReader(body2))
	httpReq2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	v2CreateLeaseHandler(w2, httpReq2)

	if w2.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate lease, got %d", w2.Code)
	}

	assertV2ErrorEnvelope(t, w2, "CONFLICT")
}

func TestLeaseAPICreateValidation(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	body := bytes.NewReader([]byte(`{"task_id":0,"agent_id":""}`))
	httpReq := httptest.NewRequest("POST", "/api/v2/leases", body)
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2CreateLeaseHandler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d body=%s", w.Code, w.Body.String())
	}

	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestLeaseHandlerMethodNotAllowed(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	httpReq := httptest.NewRequest("GET", "/api/v2/leases", nil)
	w := httptest.NewRecorder()

	v2LeaseHandler(w, httpReq)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d body=%s", w.Code, w.Body.String())
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodGet {
		t.Fatalf("expected details.method GET, got %v", details["method"])
	}
}

func TestLeaseAPIHeartbeat(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	// Set the global db for the handlers
	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	// Create a lease
	lease, err := CreateLease(testDB, 1, "test-agent", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}

	// Test heartbeat
	httpReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/leases/%s/heartbeat", lease.ID), nil)
	w := httptest.NewRecorder()

	v2LeaseHeartbeatHandler(w, httpReq, lease.ID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response Lease
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.ID != lease.ID {
		t.Errorf("Expected lease ID %s, got %s", lease.ID, response.ID)
	}
}

func TestLeaseAPIRelease(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	lease, err := CreateLease(testDB, 1, "test-agent", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}

	reqBody, _ := json.Marshal(map[string]interface{}{"reason": "manual_release"})
	httpReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/leases/%s/release", lease.ID), bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	v2LeaseActionHandler(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d body=%s", w.Code, w.Body.String())
	}

	releasedLease, err := GetLeaseByID(testDB, lease.ID)
	if err != nil {
		t.Fatalf("Failed to query released lease: %v", err)
	}
	if releasedLease != nil {
		t.Fatal("Expected lease to be removed after release")
	}

	events, err := GetEventsByProjectID(testDB, "proj_default", 20)
	if err != nil {
		t.Fatalf("Failed to query events: %v", err)
	}

	found := false
	for _, event := range events {
		if event.Kind == "lease.released" && event.EntityID == lease.ID {
			found = true
			if event.Payload["reason"] != "manual_release" {
				t.Fatalf("Expected release reason manual_release, got %v", event.Payload["reason"])
			}
		}
	}
	if !found {
		t.Fatal("Expected lease.released event")
	}
}

func TestLeaseAPIReleaseNotFound(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	httpReq := httptest.NewRequest("POST", "/api/v2/leases/lease_missing/release", nil)
	w := httptest.NewRecorder()
	v2LeaseActionHandler(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLeaseAPIReleaseIsIdempotent(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	lease, err := CreateLease(testDB, 1, "agent-idempotent", "exclusive")
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}

	firstReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/leases/%s/release", lease.ID), nil)
	firstW := httptest.NewRecorder()
	v2LeaseActionHandler(firstW, firstReq)
	if firstW.Code != http.StatusOK {
		t.Fatalf("Expected first release status 200, got %d", firstW.Code)
	}

	secondReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/leases/%s/release", lease.ID), nil)
	secondW := httptest.NewRecorder()
	v2LeaseActionHandler(secondW, secondReq)
	if secondW.Code != http.StatusNotFound {
		t.Fatalf("Expected second release status 404, got %d", secondW.Code)
	}

	leaseAfter, err := GetLeaseByID(testDB, lease.ID)
	if err != nil {
		t.Fatalf("Failed to query lease after repeated release: %v", err)
	}
	if leaseAfter != nil {
		t.Fatal("Expected no lease after repeated release attempts")
	}

	events, err := GetEventsByProjectID(testDB, "proj_default", 50)
	if err != nil {
		t.Fatalf("Failed to query events: %v", err)
	}
	releaseCount := 0
	for _, event := range events {
		if event.Kind == "lease.released" && event.EntityID == lease.ID {
			releaseCount++
		}
	}
	if releaseCount != 1 {
		t.Fatalf("Expected one lease.released event, got %d", releaseCount)
	}
}

func TestConcurrentTaskClaiming(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	// Instead of full concurrency test (which is limited by SQLite),
	// test that our constraint works for sequential duplicate attempts

	// First attempt should succeed
	lease1, err1 := CreateLease(testDB, 1, "agent1", "exclusive")
	if err1 != nil {
		t.Fatalf("First lease creation failed: %v", err1)
	}
	if lease1 == nil {
		t.Fatal("First lease should not be nil")
	}

	// Second attempt should fail due to unique constraint
	lease2, err2 := CreateLease(testDB, 1, "agent2", "exclusive")
	if err2 == nil {
		t.Fatal("Second lease creation should have failed")
	}
	if lease2 != nil {
		t.Error("Second lease should be nil")
	}

	// Verify error is constraint-related
	if !errors.Is(err2, ErrLeaseConflict) && !isLeaseConflictError(err2) {
		t.Errorf("Expected lease conflict error, got: %v", err2)
	}

	// Verify only one lease exists in database
	var leaseCount int
	err := testDB.QueryRow("SELECT COUNT(*) FROM leases WHERE task_id = 1").Scan(&leaseCount)
	if err != nil {
		t.Fatalf("Failed to query lease count: %v", err)
	}

	if leaseCount != 1 {
		t.Errorf("Expected 1 lease in database, got %d", leaseCount)
	}

	// Test that after deleting the first lease, a new one can be created
	err = DeleteLease(testDB, lease1.ID)
	if err != nil {
		t.Fatalf("Failed to delete first lease: %v", err)
	}

	// Now a new lease should succeed
	lease3, err3 := CreateLease(testDB, 1, "agent3", "exclusive")
	if err3 != nil {
		t.Fatalf("Third lease creation failed after deletion: %v", err3)
	}
	if lease3 == nil {
		t.Fatal("Third lease should not be nil")
	}

	// Verify exactly one lease exists again
	err = testDB.QueryRow("SELECT COUNT(*) FROM leases WHERE task_id = 1").Scan(&leaseCount)
	if err != nil {
		t.Fatalf("Failed to query final lease count: %v", err)
	}

	if leaseCount != 1 {
		t.Errorf("Expected 1 final lease in database, got %d", leaseCount)
	}
}

func TestGetTaskHandlerConcurrentClaimSingleWinner(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	type result struct {
		status int
		body   string
	}

	const workers = 12
	results := make([]result, 0, workers)
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/task", nil)
			req.Header.Set("X-Agent-ID", fmt.Sprintf("agent-%02d", i))
			w := httptest.NewRecorder()
			getTaskHandler(w, req)

			mu.Lock()
			results = append(results, result{status: w.Code, body: w.Body.String()})
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	winners := 0
	noTasks := 0
	conflicts := 0
	idlePlannerClaimed := 0
	for _, r := range results {
		switch {
		case r.status == http.StatusOK && strings.Contains(r.body, "AVAILABLE TASK ID: 1"):
			winners++
		case r.status == http.StatusOK && strings.Contains(r.body, "AVAILABLE TASK ID:"):
			// An agent claimed the idle planner task spawned when the queue
			// appeared empty — this is valid concurrent behaviour.
			idlePlannerClaimed++
		case r.status == http.StatusOK && strings.Contains(r.body, "No tasks available."):
			noTasks++
		case r.status == http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected response status/body: %d %q", r.status, r.body)
		}
	}

	if winners != 1 {
		t.Fatalf("expected exactly 1 winner for task 1, got %d", winners)
	}
	if winners+noTasks+conflicts+idlePlannerClaimed != workers {
		t.Fatalf("missing worker results: winners=%d noTasks=%d conflicts=%d idlePlanner=%d workers=%d", winners, noTasks, conflicts, idlePlannerClaimed, workers)
	}

	var status string
	if err := testDB.QueryRow("SELECT status FROM tasks WHERE id = 1").Scan(&status); err != nil {
		t.Fatalf("failed to read task status: %v", err)
	}
	if status != string(StatusInProgress) {
		t.Fatalf("expected task status IN_PROGRESS after claim, got %s", status)
	}

	var leaseCount int
	if err := testDB.QueryRow("SELECT COUNT(*) FROM leases WHERE task_id = 1 AND expires_at > ?", nowISO()).Scan(&leaseCount); err != nil {
		t.Fatalf("failed to query lease count: %v", err)
	}
	if leaseCount != 1 {
		t.Fatalf("expected exactly one active lease, got %d", leaseCount)
	}
}

func TestGetTaskHandlerReclaimsInProgressTaskWithLease(t *testing.T) {
	testDB := setupLeaseTestDB(t)
	defer testDB.Close()

	_, err := testDB.Exec("UPDATE tasks SET status = ? WHERE id = 1", StatusInProgress)
	if err != nil {
		t.Fatalf("failed to set task to IN_PROGRESS: %v", err)
	}

	originalDB := db
	db = testDB
	defer func() { db = originalDB }()

	req := httptest.NewRequest(http.MethodGet, "/task", nil)
	req.Header.Set("X-Agent-ID", "agent-reclaimer")
	w := httptest.NewRecorder()
	getTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "AVAILABLE TASK ID: 1") {
		t.Fatalf("expected reclaimed task in response, got %s", w.Body.String())
	}

	lease, err := GetLeaseByTaskID(testDB, 1)
	if err != nil {
		t.Fatalf("failed to query active lease: %v", err)
	}
	if lease == nil {
		t.Fatal("expected active lease for reclaimed in-progress task")
	}
	if lease.AgentID != "agent-reclaimer" {
		t.Fatalf("expected reclaiming agent lease, got %s", lease.AgentID)
	}
}
