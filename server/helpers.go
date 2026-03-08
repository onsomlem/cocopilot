package server

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/onsomlem/cocopilot/internal/httputil"
)

// ============================================================================
// First-Run Helpers

// ensureDefaultProject creates the default project if no projects exist.
func ensureDefaultProject(database *sql.DB) {
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM projects").Scan(&count); err != nil {
		log.Printf("Warning: could not check projects table: %v", err)
		return
	}
	if count == 0 {
		now := nowISO()
		_, err := database.Exec(
			"INSERT INTO projects (id, name, workdir, settings_json, created_at, updated_at) VALUES (?, ?, '', NULL, ?, ?)",
			DefaultProjectID, "Default", now, now,
		)
		if err != nil {
			log.Printf("Warning: could not create default project: %v", err)
		} else {
			log.Printf("First run detected — created default project %q", DefaultProjectID)
		}
	}
}

// ============================================================================
// API v2 Core Handlers — thin wrappers delegating to internal/httputil
// ============================================================================

// Type aliases so existing code using v2Error / v2ErrorEnvelope keeps compiling.
type v2Error = httputil.V2Error
type v2ErrorEnvelope = httputil.V2ErrorEnvelope

func writeV2JSON(w http.ResponseWriter, status int, payload interface{}) {
	httputil.WriteV2JSON(w, status, payload)
}

func writeV2Error(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	httputil.WriteV2Error(w, status, code, message, details)
}

func writeV2MethodNotAllowed(w http.ResponseWriter, r *http.Request, allowed ...string) {
	httputil.WriteV2MethodNotAllowed(w, r, allowed...)
}

func clientIP(r *http.Request) string {
	return httputil.ClientIP(r)
}

func validateWorkdir(path string) error {
	return httputil.ValidateWorkdir(path)
}

// getCurrentSchemaVersion returns the current schema version from the database
func getCurrentSchemaVersion() int {
	var version int
	err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err == sql.ErrNoRows || err != nil {
		return 0
	}
	return version
}
