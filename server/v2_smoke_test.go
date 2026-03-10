package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- UI Page Handlers (all GET, return HTML, no DB needed) ----------

func TestSmokeUI_IndexHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	indexHandler(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("indexHandler returned %d, expected 302 redirect", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %s", loc)
	}
}

func TestSmokeUI_IndexHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nosuch", nil)
	w := httptest.NewRecorder()
	indexHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_DashboardHandler(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	dashboardHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboardHandler returned %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
}

func TestSmokeUI_DashboardHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dashboard/extra", nil)
	w := httptest.NewRecorder()
	dashboardHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_HealthDashboardHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthDashboardHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("healthDashboardHandler returned %d", w.Code)
	}
}

func TestSmokeUI_HealthDashboard_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health/extra", nil)
	w := httptest.NewRecorder()
	healthDashboardHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_AgentsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/agents", nil)
	w := httptest.NewRecorder()
	agentsPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("agentsPlaceholderHandler returned %d", w.Code)
	}
}

func TestSmokeUI_AgentsHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/agents/extra", nil)
	w := httptest.NewRecorder()
	agentsPlaceholderHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_MemoryHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/memory", nil)
	w := httptest.NewRecorder()
	memoryPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("memoryPlaceholderHandler returned %d", w.Code)
	}
}

func TestSmokeUI_MemoryHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/memory/extra", nil)
	w := httptest.NewRecorder()
	memoryPlaceholderHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_AuditHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	w := httptest.NewRecorder()
	auditPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("auditPlaceholderHandler returned %d", w.Code)
	}
}

func TestSmokeUI_AuditHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/audit/extra", nil)
	w := httptest.NewRecorder()
	auditPlaceholderHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_RepoHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/repo", nil)
	w := httptest.NewRecorder()
	repoPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("repoPlaceholderHandler returned %d", w.Code)
	}
}

func TestSmokeUI_RepoHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/repo/extra", nil)
	w := httptest.NewRecorder()
	repoPlaceholderHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_ProjectsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	w := httptest.NewRecorder()
	projectsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("projectsHandler returned %d", w.Code)
	}
}

func TestSmokeUI_ProjectsHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/projects/extra", nil)
	w := httptest.NewRecorder()
	projectsHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_PoliciesHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	policiesHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("policiesHandler returned %d", w.Code)
	}
}

func TestSmokeUI_PoliciesHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/policies/extra", nil)
	w := httptest.NewRecorder()
	policiesHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_SettingsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	settingsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("settingsHandler returned %d", w.Code)
	}
}

func TestSmokeUI_SettingsHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/settings/extra", nil)
	w := httptest.NewRecorder()
	settingsHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_DependenciesHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dependencies", nil)
	w := httptest.NewRecorder()
	dependenciesHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dependenciesHandler returned %d", w.Code)
	}
}

func TestSmokeUI_DependenciesHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dependencies/extra", nil)
	w := httptest.NewRecorder()
	dependenciesHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_EventsBrowserHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/events-browser", nil)
	w := httptest.NewRecorder()
	eventsBrowserHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("eventsBrowserHandler returned %d", w.Code)
	}
}

func TestSmokeUI_EventsBrowserHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/events-browser/extra", nil)
	w := httptest.NewRecorder()
	eventsBrowserHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_TaskGraphsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/graphs/tasks", nil)
	w := httptest.NewRecorder()
	taskGraphsPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("taskGraphsPlaceholderHandler returned %d", w.Code)
	}
}

func TestSmokeUI_TaskGraphsHandler_WrongPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/graphs/tasks/extra", nil)
	w := httptest.NewRecorder()
	taskGraphsPlaceholderHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_RunsHandler_RootPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/runs/", nil)
	w := httptest.NewRecorder()
	runsPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("runsPlaceholderHandler returned %d", w.Code)
	}
}

func TestSmokeUI_ContextPacksHandler_Root(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/context-packs/", nil)
	w := httptest.NewRecorder()
	contextPacksPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("contextPacksPlaceholderHandler returned %d", w.Code)
	}
}

func TestSmokeUI_ContextPacksHandler_Detail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/context-packs/some-pack-id", nil)
	w := httptest.NewRecorder()
	contextPacksPlaceholderHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("contextPacksPlaceholderHandler detail returned %d", w.Code)
	}
}

func TestSmokeUI_ContextPacksHandler_SubPath404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/context-packs/pack-id/extra", nil)
	w := httptest.NewRecorder()
	contextPacksPlaceholderHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeUI_RepoGraphHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/graphs/repo", nil)
	w := httptest.NewRecorder()
	repoGraphHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("repoGraphHandler returned %d", w.Code)
	}
}

func TestSmokeUI_DiffViewerHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/diffs/some-id", nil)
	w := httptest.NewRecorder()
	diffViewerHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("diffViewerHandler returned %d", w.Code)
	}
}

// ---------- V1 Handlers ----------

func TestSmokeV1_InstructionsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/instructions", nil)
	w := httptest.NewRecorder()
	instructionsHandler(w, req)
	// instructionsHandler writes 303 status
	if w.Code != http.StatusSeeOther {
		t.Fatalf("instructionsHandler returned %d, want 303", w.Code)
	}
}

func TestSmokeV1_InstructionsDetailedHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/instructions-detailed", nil)
	w := httptest.NewRecorder()
	instructionsDetailedHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("instructionsDetailedHandler returned %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
}

func TestSmokeV1_DeleteTaskHandler_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/delete", nil)
	w := httptest.NewRecorder()
	deleteTaskHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestSmokeV1_DeleteTaskHandler_BadID(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/delete", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	deleteTaskHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSmokeV1_GetAllTasksJSON(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	tasks, err := getAllTasksJSON()
	if err != nil {
		t.Fatalf("getAllTasksJSON failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

// ---------- V2 API Handlers (JSON endpoints requiring DB) ----------

func TestSmokeV2_StatusHandler(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	handler := v2StatusHandler(runtimeConfig{})
	req := httptest.NewRequest(http.MethodGet, "/api/v2/status", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("v2StatusHandler returned %d", w.Code)
	}
}

func TestSmokeV2_StatusHandler_WrongMethod(t *testing.T) {
	handler := v2StatusHandler(runtimeConfig{})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/status", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestSmokeV2_ArtifactCommentsHandler_InvalidPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/artifacts/", nil)
	w := httptest.NewRecorder()
	v2ArtifactCommentsHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSmokeV2_ArtifactCommentsHandler_GET(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/artifacts/test-artifact/comments", nil)
	w := httptest.NewRecorder()
	v2ArtifactCommentsHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSmokeV2_ProjectDashboardHandler_MissingProject(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v2/projects/nonexistent/dashboard", nil)
	w := httptest.NewRecorder()
	v2ProjectDashboardHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSmokeV2_ProjectDashboardHandler_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/test/dashboard", nil)
	w := httptest.NewRecorder()
	v2ProjectDashboardHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestSmokeV2_ProjectNotificationsHandler_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects/test/notifications", nil)
	w := httptest.NewRecorder()
	v2ProjectNotificationsHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestSmokeV2_ProjectFilesScanHandler_InvalidPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/projects//files/scan", nil)
	w := httptest.NewRecorder()
	v2ProjectFilesScanHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------- V1 Handlers with DB ----------

func TestSmokeV1_DeleteTaskHandler_NonexistentTask(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	body := "task_id=9999"
	req := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	deleteTaskHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ---------- Middleware ----------

func TestSmokeMiddleware_WithCORS(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := withCORS(inner)

	// Regular request with Origin header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("missing CORS origin header")
	}

	// Preflight OPTIONS request
	req2 := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req2.Header.Set("Origin", "http://localhost:3000")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS expected 204, got %d", w2.Code)
	}
}

func TestSmokeMiddleware_WithRequestLog(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := withRequestLog(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !called {
		t.Fatal("inner handler was not called")
	}

	// Static path should also pass through
	called = false
	req2 := httptest.NewRequest(http.MethodGet, "/static/foo.js", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if !called {
		t.Fatal("inner handler was not called for static path")
	}
}

// ---------- RegisterRoutes Smoke ----------

func TestSmokeRegisterRoutes(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	cfg := runtimeConfig{}
	registerRoutes(mux, cfg)

	// Validate a sampling of registered routes
	paths := []struct {
		method string
		path   string
		expect int // minimum acceptable (not 404)
	}{
		{http.MethodGet, "/", http.StatusOK},
		{http.MethodGet, "/dashboard", http.StatusOK},
		{http.MethodGet, "/agents", http.StatusOK},
		{http.MethodGet, "/health", http.StatusOK},
		{http.MethodGet, "/api/v2/health", http.StatusOK},
		{http.MethodGet, "/api/v2/version", http.StatusOK},
	}

	for _, tc := range paths {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s returned 404 — route not registered", tc.method, tc.path)
		}
	}
}

// TestSmokePublicNavSuite asserts every public navigation route returns a
// non-404 status, covering all UI pages and key API endpoints.
func TestSmokePublicNavSuite(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	cfg := runtimeConfig{}
	registerRoutes(mux, cfg)

	routes := []struct {
		method string
		path   string
	}{
		// UI pages (primary nav)
		{http.MethodGet, "/"},
		{http.MethodGet, "/dashboard"},
		{http.MethodGet, "/board"},
		{http.MethodGet, "/runs"},
		{http.MethodGet, "/agents"},
		{http.MethodGet, "/repo"},
		{http.MethodGet, "/events-browser"},
		// UI pages (secondary nav)
		{http.MethodGet, "/dependencies"},
		{http.MethodGet, "/graphs/tasks"},
		{http.MethodGet, "/memory"},
		{http.MethodGet, "/health"},
		{http.MethodGet, "/audit"},
		{http.MethodGet, "/context-packs"},
		{http.MethodGet, "/graphs/repo"},
		// Management pages
		{http.MethodGet, "/projects"},
		{http.MethodGet, "/policies"},
		{http.MethodGet, "/settings"},
		// API v2 read endpoints
		{http.MethodGet, "/api/v2/health"},
		{http.MethodGet, "/api/v2/version"},
		{http.MethodGet, "/api/v2/metrics"},
		{http.MethodGet, "/api/v2/status"},
		{http.MethodGet, "/api/v2/projects"},
		{http.MethodGet, "/api/v2/tasks"},
		{http.MethodGet, "/api/v2/agents"},
		{http.MethodGet, "/api/v2/events"},
		{http.MethodGet, "/api/v2/runs"},
		// v1 endpoints
		{http.MethodGet, "/task"},
		{http.MethodGet, "/instructions"},
		{http.MethodGet, "/api/tasks"},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code == http.StatusNotFound {
				t.Errorf("%s %s returned 404 — route not registered", tc.method, tc.path)
			}
		})
	}
}
