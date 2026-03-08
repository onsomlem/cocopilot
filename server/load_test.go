package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func setupBenchDB(b *testing.B) (*sql.DB, func()) {
	b.Helper()
	dir := b.TempDir()
	dbPath := filepath.Join(dir, fmt.Sprintf("bench_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	if err := runMigrations(testDB); err != nil {
		b.Fatalf("migrations: %v", err)
	}
	oldDB := db
	db = testDB
	cleanup := func() {
		db.Close()
		db = oldDB
		os.Remove(dbPath)
	}
	return testDB, cleanup
}

// BenchmarkTaskCreate measures throughput for task creation.
func BenchmarkTaskCreate(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "bench-proj", "/tmp", nil)
	if err != nil {
		b.Fatalf("create project: %v", err)
	}

	body := fmt.Sprintf(`{"project_id":"%s","title":"bench task","instructions":"do it"}`, proj.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			b.Fatalf("create failed: %d %s", w.Code, w.Body.String())
		}
	}
}

// BenchmarkTaskClaim measures claim throughput for sequential agents.
func BenchmarkTaskClaim(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "claim-bench-proj", "/tmp", nil)
	if err != nil {
		b.Fatalf("create project: %v", err)
	}

	// Pre-create tasks
	taskIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		body := fmt.Sprintf(`{"project_id":"%s","title":"claim bench %d","instructions":"do it"}`, proj.ID, i)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			b.Fatalf("create failed: %d", w.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		taskIDs[i] = resp["id"].(string)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		claimBody := `{"agent_id":"bench-agent","mode":"code"}`
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%s/claim", taskIDs[i]), strings.NewReader(claimBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("claim failed: %d %s", w.Code, w.Body.String())
		}
	}
}

// BenchmarkConcurrentClaims measures concurrent claim contention.
func BenchmarkConcurrentClaims(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "concurrent-bench-proj", "/tmp", nil)
	if err != nil {
		b.Fatalf("create project: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := fmt.Sprintf(`{"project_id":"%s","title":"concurrent claim %d","instructions":"do it"}`, proj.ID, i)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		taskID := resp["id"].(string)

		// Race 4 agents to claim it
		var wg sync.WaitGroup
		wins := 0
		var mu sync.Mutex
		for a := 0; a < 4; a++ {
			wg.Add(1)
			go func(agentIdx int) {
				defer wg.Done()
				claimBody := fmt.Sprintf(`{"agent_id":"agent-%d","mode":"code"}`, agentIdx)
				cr := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/tasks/%s/claim", taskID), strings.NewReader(claimBody))
				cr.Header.Set("Content-Type", "application/json")
				cw := httptest.NewRecorder()
				mux.ServeHTTP(cw, cr)
				if cw.Code == http.StatusOK {
					mu.Lock()
					wins++
					mu.Unlock()
				}
			}(a)
		}
		wg.Wait()

		if wins == 0 {
			b.Fatal("no agent won the claim")
		}
	}
}

// BenchmarkTaskList measures list endpoint throughput with many tasks.
func BenchmarkTaskList(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	mux := http.NewServeMux()
	registerRoutes(mux, runtimeConfig{})

	proj, err := CreateProject(db, "list-bench-proj", "/tmp", nil)
	if err != nil {
		b.Fatalf("create project: %v", err)
	}

	// Pre-populate 100 tasks
	for i := 0; i < 100; i++ {
		body := fmt.Sprintf(`{"project_id":"%s","title":"list bench %d","instructions":"do it"}`, proj.ID, i)
		req := httptest.NewRequest(http.MethodPost, "/api/v2/tasks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/tasks?limit=50", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("list failed: %d", w.Code)
		}
	}
}
