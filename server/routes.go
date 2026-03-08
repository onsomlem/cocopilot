package server

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// withCORS wraps a handler to add CORS headers for API access from browser clients.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withRequestLog wraps a handler to log each request.
func withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/favicon.ico" {
			next.ServeHTTP(w, r)
			return
		}
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func registerRoutes(mux *http.ServeMux, cfg runtimeConfig) {
	setAutomationRules(cfg.AutomationRules)
	maxDepth := cfg.MaxAutomationDepth
	if maxDepth <= 0 {
		maxDepth = 5
	}
	setMaxAutomationDepth(maxDepth)

	automationRate := cfg.AutomationRateLimit
	if automationRate <= 0 {
		automationRate = 100
	}
	setAutomationRateLimit(automationRate)

	automationBurst := cfg.AutomationBurstLimit
	if automationBurst <= 0 {
		automationBurst = 10
	}
	setAutomationBurstLimit(automationBurst)

	circuitMaxFail := cfg.AutomationCircuitMaxFail
	if circuitMaxFail <= 0 {
		circuitMaxFail = 5
	}
	circuitCooldown := cfg.AutomationCircuitCooldown
	if circuitCooldown <= 0 {
		circuitCooldown = 5 * time.Minute
	}
	setAutomationCircuitBreaker(newAutomationCircuitBreaker(circuitMaxFail, circuitCooldown))

	heartbeatInterval := resolveSSEHeartbeatInterval(cfg)
	sseReplayLimitMax := resolveSSEReplayLimitMax(cfg)
	v1EventsReplayLimitMax := resolveV1EventsReplayLimitMax(cfg)

	// Serve embedded static files (Alpine.js, etc.) for self-contained operation
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/projects", projectsHandler)
	mux.HandleFunc("/agents", agentsPlaceholderHandler)
	mux.HandleFunc("/runs", runsPlaceholderHandler)
	mux.HandleFunc("/runs/", runsPlaceholderHandler)
	mux.HandleFunc("/memory", memoryPlaceholderHandler)
	mux.HandleFunc("/policies", policiesHandler)
	mux.HandleFunc("/dependencies", dependenciesHandler)
	mux.HandleFunc("/context-packs", contextPacksPlaceholderHandler)
	mux.HandleFunc("/context-packs/", contextPacksPlaceholderHandler)
	mux.HandleFunc("/events-browser", eventsBrowserHandler)
	mux.HandleFunc("/graphs/tasks", taskGraphsPlaceholderHandler)
	mux.HandleFunc("/audit", auditPlaceholderHandler)
	mux.HandleFunc("/repo", repoPlaceholderHandler)
	mux.HandleFunc("/diffs/", diffViewerHandler)
	mux.HandleFunc("/settings", settingsHandler)
	mux.HandleFunc("/health", healthDashboardHandler)
	mux.HandleFunc("/graphs/repo", repoGraphHandler)
	mux.HandleFunc("/api/tasks", apiTasksHandler)
	mux.HandleFunc("/events", eventsHandler(heartbeatInterval, v1EventsReplayLimitMax))
	mux.HandleFunc("/task", withV1MutationAuth(cfg, getTaskHandler))
	mux.HandleFunc("/save", withV1MutationAuth(cfg, saveHandler))
	mux.HandleFunc("/update-status", withV1MutationAuth(cfg, updateStatusHandler))
	mux.HandleFunc("/create", withV1MutationAuth(cfg, createHandler))
	mux.HandleFunc("/instructions", instructionsHandler)
	mux.HandleFunc("/api/workdir", getWorkdirHandler)
	mux.HandleFunc("/set-workdir", withV1MutationAuth(cfg, setWorkdirHandler))
	mux.HandleFunc("/delete", withV1MutationAuth(cfg, deleteTaskHandler))

	// API v2 routes
	mux.HandleFunc("/api/v2/health", withV2Auth(cfg, v2HealthHandler))
	mux.HandleFunc("/api/v2/status", withV2Auth(cfg, v2StatusHandler(cfg)))
	mux.HandleFunc("/api/v2/metrics", withV2Auth(cfg, v2MetricsHandler))
	mux.HandleFunc("/api/v2/backup", withV2Auth(cfg, v2BackupHandler(cfg)))
	mux.HandleFunc("/api/v2/restore", withV2Auth(cfg, v2RestoreHandler(cfg)))
	mux.HandleFunc("/api/v2/version", withV2Auth(cfg, v2VersionHandler(cfg)))
	mux.HandleFunc("/api/v2/config", withV2Auth(cfg, v2ConfigHandler(cfg)))
	mux.HandleFunc("/api/v2/projects", withV2Auth(cfg, v2ProjectsRouteHandler))
	mux.HandleFunc("/api/v2/projects/", withV2Auth(cfg, v2ProjectRouteHandler(heartbeatInterval, sseReplayLimitMax)))
	mux.HandleFunc("/api/v2/tasks", withV2Auth(cfg, policyEnforcementMiddleware(v2TasksRouteHandler)))
	mux.HandleFunc("/api/v2/tasks/", withV2Auth(cfg, policyEnforcementMiddleware(v2TaskDetailRouteHandler)))
	mux.HandleFunc("/api/v2/runs/", withV2Auth(cfg, policyEnforcementMiddleware(v2RunsRouteHandler)))
	mux.HandleFunc("/api/v2/events", withV2Auth(cfg, v2ListEventsHandler))
	mux.HandleFunc("/api/v2/events/stream", withV2Auth(cfg, v2EventsStreamHandler(heartbeatInterval, sseReplayLimitMax)))
	mux.HandleFunc("/api/v2/audit", withV2Auth(cfg, v2AuditHandler))
	mux.HandleFunc("/api/v2/agents", withV2Auth(cfg, v2AgentsRouteHandler))
	mux.HandleFunc("/api/v2/agents/", withV2Auth(cfg, v2AgentActionHandler))
	mux.HandleFunc("/api/v2/leases", withV2Auth(cfg, v2LeaseHandler))
	mux.HandleFunc("/api/v2/leases/", withV2Auth(cfg, v2LeaseActionHandler))
	mux.HandleFunc("/api/v2/context-packs/", withV2Auth(cfg, v2ContextPackDetailHandler))
	mux.HandleFunc("/api/v2/artifacts/", withV2Auth(cfg, v2ArtifactCommentsHandler))
	mux.HandleFunc("/api/v2/projects/import", withV2Auth(cfg, v2ProjectImportHandler))
}

func v2ProjectsRouteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		v2CreateProjectHandler(w, r)
	case http.MethodGet:
		v2ListProjectsHandler(w, r)
	default:
		writeV2MethodNotAllowed(w, r, http.MethodPost, http.MethodGet)
	}
}

func v2TasksRouteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		v2ListTasksHandler(w, r)
	case http.MethodPost:
		v2CreateTaskHandler(w, r)
	default:
		writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

func v2ProjectRouteHandler(heartbeatInterval time.Duration, replayLimitMax int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/events/replay") {
			v2ProjectEventsReplayHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/events/stream") {
			v2ProjectEventsStreamHandler(heartbeatInterval, replayLimitMax)(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/dashboard") {
			v2ProjectDashboardHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/audit") {
			v2ProjectAuditHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/automation/rules") {
			v2ProjectAutomationRulesHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/automation/simulate") {
			v2ProjectAutomationSimulateHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/automation/replay") {
			v2ProjectAutomationReplayHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/automation/stats") {
			v2ProjectAutomationStatsHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/graphs/tasks") {
			v2ProjectGraphTasksHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/ide-signals") {
			v2ProjectIDESignalsHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/audit/export") {
			v2ProjectAuditExportHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/tree") {
			v2ProjectTreeHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/changes") {
			v2ProjectChangesHandler(w, r)
			return
		}
		// claim-next must be checked before /tasks suffix match
		if strings.HasSuffix(r.URL.Path, "/tasks/claim-next") {
			v2ProjectTasksClaimNextHandler(w, r)
			return
		}
		// Check if this is a project tasks request
		if strings.HasSuffix(r.URL.Path, "/tasks") {
			v2ProjectTasksHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/policies") {
			v2ProjectPoliciesHandler(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "/policies/") {
			v2ProjectPolicyDetailHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/memory") {
			v2ProjectMemoryHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/context-packs") {
			v2ProjectContextPacksHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/notifications") {
			v2ProjectNotificationsHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/files/sync") {
			v2ProjectFilesSyncHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/files/scan") {
			v2ProjectFilesScanHandler(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "/files/") {
			v2ProjectFileDetailHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/files") {
			v2ProjectFilesHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/export") {
			v2ProjectExportHandler(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/templates") {
			v2ProjectTemplatesHandler(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "/templates/") {
			v2ProjectTemplateDetailHandler(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			v2GetProjectHandler(w, r)
		case http.MethodPut, http.MethodPatch:
			v2UpdateProjectHandler(w, r)
		case http.MethodDelete:
			v2DeleteProjectHandler(w, r)
		default:
			writeV2MethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete)
		}
	}
}

func v2AgentsRouteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		v2RegisterAgentHandler(w, r)
	case http.MethodGet:
		v2ListAgentsHandler(w, r)
	default:
		writeV2MethodNotAllowed(w, r, http.MethodPost, http.MethodGet)
	}
}
