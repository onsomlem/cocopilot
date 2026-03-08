package server

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func requiresAPIKeyForRequest(cfg runtimeConfig, r *http.Request) bool {
	if !cfg.RequireAPIKey {
		return false
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v2/") {
		return false
	}
	if cfg.RequireAPIKeyReads {
		return true
	}
	return r.Method != http.MethodGet
}

func isV2TaskClaimRequest(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v2/tasks/") {
		return false
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v2/tasks/")
	parts := strings.Split(trimmed, "/")
	return len(parts) == 2 && parts[1] == "claim"
}

func isV2ProjectTasksRequest(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/v2/projects/") {
		return false
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(trimmed, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] == "tasks"
}

func isV2ProjectAuditRequest(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/v2/projects/") {
		return false
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(trimmed, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] == "audit"
}

func isV2ProjectPoliciesRequest(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/v2/projects/") {
		return false
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	parts := strings.Split(trimmed, "/")
	return len(parts) >= 2 && parts[0] != "" && parts[1] == "policies"
}

func requiredScopeForRequest(r *http.Request) string {
	if r.Method == http.MethodGet {
		if isV2ProjectPoliciesRequest(r) {
			return "policy.read"
		}
		if isV2ProjectAuditRequest(r) {
			return "audit.read"
		}
		if strings.HasPrefix(r.URL.Path, "/api/v2/tasks") {
			return "tasks:read"
		}
		return "v2:read"
	}

	switch {
	case isV2ProjectPoliciesRequest(r):
		return "policy.write"
	case strings.HasPrefix(r.URL.Path, "/api/v2/tasks"):
		return "tasks:write"
	case strings.HasPrefix(r.URL.Path, "/api/v2/projects"):
		return "projects:write"
	case strings.HasPrefix(r.URL.Path, "/api/v2/agents"):
		return "agents:write"
	case strings.HasPrefix(r.URL.Path, "/api/v2/leases"):
		return "leases:write"
	case strings.HasPrefix(r.URL.Path, "/api/v2/runs"):
		return "runs:write"
	default:
		return "v2:write"
	}
}

func allowedScopesForRequest(r *http.Request) []string {
	if isV2TaskClaimRequest(r) {
		return []string{"tasks:write", "leases:write"}
	}
	if r.Method == http.MethodGet && isV2ProjectAuditRequest(r) {
		return []string{"events.read", "audit.read"}
	}
	if r.Method == http.MethodGet && isV2ProjectTasksRequest(r) {
		return []string{"v2:read", "tasks:read"}
	}
	return nil
}

func authEventProjectID(r *http.Request) string {
	if db == nil {
		return DefaultProjectID
	}
	if strings.HasPrefix(r.URL.Path, "/api/v2/projects/") {
		trimmed := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) > 0 && parts[0] != "" {
			var exists int
			err := db.QueryRow("SELECT COUNT(*) FROM projects WHERE id = ?", parts[0]).Scan(&exists)
			if err == nil && exists > 0 {
				return parts[0]
			}
		}
	}
	return DefaultProjectID
}

func logAuthDecision(decision string, identity *authIdentity, requiredScope string, r *http.Request, reason string) {
	entry := map[string]interface{}{
		"event":          "auth_decision",
		"decision":       decision,
		"required_scope": requiredScope,
		"endpoint":       r.URL.Path,
		"method":         r.Method,
	}
	if reason != "" {
		entry["reason"] = reason
	}
	if identity != nil {
		entry["identity_id"] = identity.ID
		entry["identity_type"] = identity.Type
	} else {
		entry["identity_id"] = "unknown"
		entry["identity_type"] = "unknown"
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("auth_decision decision=%s method=%s path=%s: %v", decision, r.Method, r.URL.Path, err)
		return
	}
	log.Print(string(payload))
}

func emitAuthDeniedEvent(identity *authIdentity, requiredScope string, r *http.Request, reason string) {
	if db == nil {
		return
	}

	payload := map[string]interface{}{
		"reason":         reason,
		"required_scope": requiredScope,
		"endpoint":       r.URL.Path,
		"method":         r.Method,
	}
	if identity != nil {
		payload["identity_id"] = identity.ID
		payload["identity_type"] = identity.Type
	}

	projectID := authEventProjectID(r)
	entityID := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	if _, err := CreateEvent(db, projectID, "auth.denied", "auth", entityID, payload); err != nil {
		log.Printf("Warning: failed to emit auth.denied event: %v", err)
	}
}

func identityHasScope(identity authIdentity, requiredScope string) bool {
	if requiredScope == "" {
		return true
	}

	if _, ok := identity.Scopes["*"]; ok {
		return true
	}
	if _, ok := identity.Scopes[requiredScope]; ok {
		return true
	}

	scopeParts := strings.SplitN(requiredScope, ":", 2)
	if len(scopeParts) == 2 {
		if _, ok := identity.Scopes[scopeParts[0]+":*"]; ok {
			return true
		}
	}

	dotParts := strings.SplitN(requiredScope, ".", 2)
	if len(dotParts) == 2 {
		if _, ok := identity.Scopes[dotParts[0]+".*"]; ok {
			return true
		}
	}

	if strings.HasSuffix(requiredScope, ":write") {
		if _, ok := identity.Scopes["v2:write"]; ok {
			return true
		}
	}
	if strings.HasSuffix(requiredScope, ":read") || strings.HasSuffix(requiredScope, ".read") {
		if _, ok := identity.Scopes["v2:read"]; ok {
			return true
		}
	}
	if strings.HasSuffix(requiredScope, ".write") {
		if _, ok := identity.Scopes["v2:write"]; ok {
			return true
		}
	}

	return false
}

func resolveAuthIdentity(cfg runtimeConfig, providedAPIKey string) (*authIdentity, bool) {
	for i := range cfg.AuthIdentities {
		identity := &cfg.AuthIdentities[i]
		if subtle.ConstantTimeCompare([]byte(providedAPIKey), []byte(identity.APIKey)) == 1 {
			return identity, true
		}
	}
	if cfg.APIKey != "" && subtle.ConstantTimeCompare([]byte(providedAPIKey), []byte(cfg.APIKey)) == 1 {
		return &authIdentity{
			ID:     "legacy_default",
			Type:   "service",
			APIKey: cfg.APIKey,
			Scopes: map[string]struct{}{"*": {}},
		}, true
	}
	return nil, false
}

func authorizeV2Request(cfg runtimeConfig, w http.ResponseWriter, r *http.Request) bool {
	requiredScope := requiredScopeForRequest(r)
	allowedScopes := allowedScopesForRequest(r)
	if len(allowedScopes) == 0 {
		allowedScopes = []string{requiredScope}
	} else {
		requiredScope = strings.Join(allowedScopes, " or ")
	}
	if !requiresAPIKeyForRequest(cfg, r) {
		logAuthDecision("allow", nil, requiredScope, r, "auth_not_required")
		return true
	}

	provided := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if provided == "" {
		logAuthDecision("deny", nil, requiredScope, r, "missing_api_key")
		emitAuthDeniedEvent(nil, requiredScope, r, "missing_api_key")
		writeV2Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing API key", map[string]interface{}{
			"header": "X-API-Key",
		})
		return false
	}

	identity, ok := resolveAuthIdentity(cfg, provided)
	if !ok {
		logAuthDecision("deny", nil, requiredScope, r, "invalid_api_key")
		emitAuthDeniedEvent(nil, requiredScope, r, "invalid_api_key")
		writeV2Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid API key", map[string]interface{}{
			"header": "X-API-Key",
		})
		return false
	}
	authorized := false
	for _, scope := range allowedScopes {
		if identityHasScope(*identity, scope) {
			authorized = true
			break
		}
	}
	if !authorized {
		logAuthDecision("deny", identity, requiredScope, r, "insufficient_scope")
		emitAuthDeniedEvent(identity, requiredScope, r, "insufficient_scope")
		writeV2Error(w, http.StatusForbidden, "FORBIDDEN", "Insufficient scope", map[string]interface{}{
			"identity_id":    identity.ID,
			"identity_type":  identity.Type,
			"required_scope": requiredScope,
		})
		return false
	}

	logAuthDecision("allow", identity, requiredScope, r, "authorized")

	return true
}

func withV2Auth(cfg runtimeConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorizeV2Request(cfg, w, r) {
			return
		}
		next(w, r)
	}
}

// withV1MutationAuth protects v1 mutation endpoints when API key auth is enabled.
// Unlike v2 auth, v1 endpoints return plain text errors to match existing v1 conventions.
func withV1MutationAuth(cfg runtimeConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.RequireAPIKey {
			next(w, r)
			return
		}
		provided := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if provided == "" {
			http.Error(w, "Unauthorized: missing API key", http.StatusUnauthorized)
			return
		}
		if _, ok := resolveAuthIdentity(cfg, provided); !ok {
			http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ---------------------------------------------------------------------------
// Policy enforcement middleware (B1.3)
// ---------------------------------------------------------------------------

// policyEnforcementMiddleware evaluates project policies before allowing
// mutation requests (POST/PUT/PATCH/DELETE) to proceed.
func policyEnforcementMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only enforce on mutation methods
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		// Determine project ID from the request
		projectID := extractProjectID(r)
		if projectID == "" {
			next(w, r) // Can't enforce without project context
			return
		}

		// Load enabled policies for the project
		enabledOnly := true
		policies, _, err := ListPoliciesByProject(db, projectID, &enabledOnly, 100, 0, "created_at", "asc")
		if err != nil || len(policies) == 0 {
			next(w, r)
			return
		}

		// Build policy context
		ctx := PolicyContext{
			ProjectID:    projectID,
			AgentID:      extractAgentID(r),
			Action:       determinePolicyAction(r),
			ResourceType: determineResourceType(r),
			Timestamp:    time.Now().UTC(),
		}

		// Evaluate
		if policyEngine == nil {
			next(w, r)
			return
		}
		allowed, violations := policyEngine.Evaluate(ctx, policies)
		if !allowed {
			details := map[string]interface{}{
				"violations": violations,
			}
			writeV2Error(w, http.StatusForbidden, "POLICY_VIOLATION",
				"Request denied by policy", details)
			return
		}

		next(w, r)
	}
}

// extractProjectID extracts a project ID from the request. It checks the
// query parameter "project_id" first, then attempts to parse it from the URL
// path for project-scoped endpoints.
func extractProjectID(r *http.Request) string {
	// 1. Query parameter
	if pid := r.URL.Query().Get("project_id"); pid != "" {
		return pid
	}

	// 2. URL path for /api/v2/projects/{id}/... endpoints
	if strings.HasPrefix(r.URL.Path, "/api/v2/projects/") {
		seg := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
		parts := strings.SplitN(seg, "/", 2)
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	return ""
}

// extractAgentID extracts an agent identifier from the request.
func extractAgentID(r *http.Request) string {
	if aid := r.Header.Get("X-Agent-ID"); aid != "" {
		return aid
	}
	return r.URL.Query().Get("agent_id")
}

// determinePolicyAction maps the HTTP method and URL path to a policy action
// string such as "create_task", "claim_task", "create_run", etc.
func determinePolicyAction(r *http.Request) string {
	path := r.URL.Path

	// Tasks
	if strings.HasPrefix(path, "/api/v2/tasks") {
		if strings.HasSuffix(path, "/claim") {
			return "claim_task"
		}
		if strings.HasSuffix(path, "/complete") {
			return "complete_task"
		}
		if strings.Contains(path, "/dependencies") {
			switch r.Method {
			case http.MethodPost:
				return "add_dependency"
			case http.MethodDelete:
				return "remove_dependency"
			}
			return "manage_dependency"
		}
		switch r.Method {
		case http.MethodPost:
			return "create_task"
		case http.MethodPatch:
			return "update_task"
		case http.MethodDelete:
			return "delete_task"
		}
		return "modify_task"
	}

	// Runs
	if strings.HasPrefix(path, "/api/v2/runs") {
		if strings.HasSuffix(path, "/steps") {
			return "create_run_step"
		}
		if strings.HasSuffix(path, "/logs") {
			return "create_run_log"
		}
		if strings.HasSuffix(path, "/artifacts") {
			return "create_artifact"
		}
		switch r.Method {
		case http.MethodPost:
			return "create_run"
		case http.MethodPatch:
			return "update_run"
		}
		return "modify_run"
	}

	// Projects
	if strings.HasPrefix(path, "/api/v2/projects") {
		if strings.Contains(path, "/policies") {
			switch r.Method {
			case http.MethodPost:
				return "create_policy"
			case http.MethodPatch:
				return "update_policy"
			case http.MethodDelete:
				return "delete_policy"
			}
			return "manage_policy"
		}
		switch r.Method {
		case http.MethodPost:
			return "create_project"
		case http.MethodPatch:
			return "update_project"
		case http.MethodDelete:
			return "delete_project"
		}
		return "modify_project"
	}

	return "unknown"
}

// determineResourceType maps the URL path to a resource type.
func determineResourceType(r *http.Request) string {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v2/tasks") {
		return "task"
	}
	if strings.HasPrefix(path, "/api/v2/runs") {
		return "run"
	}
	if strings.HasPrefix(path, "/api/v2/projects") {
		if strings.Contains(path, "/policies") {
			return "policy"
		}
		return "project"
	}
	if strings.HasPrefix(path, "/api/v2/agents") {
		return "agent"
	}
	if strings.HasPrefix(path, "/api/v2/leases") {
		return "lease"
	}
	return "unknown"
}
