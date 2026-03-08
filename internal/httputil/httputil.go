package httputil

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
)

type V2Error struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type V2ErrorEnvelope struct {
	Error V2Error `json:"error"`
}

func WriteV2JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Warning: failed to encode v2 JSON response: %v", err)
	}
}

func WriteV2Error(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	WriteV2JSON(w, status, V2ErrorEnvelope{
		Error: V2Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func WriteV2MethodNotAllowed(w http.ResponseWriter, r *http.Request, allowed ...string) {
	details := map[string]interface{}{
		"method":          r.Method,
		"allowed_methods": allowed,
	}
	WriteV2Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", details)
}

// ClientIP extracts the best-available client IP from an HTTP request.
// Checks X-Forwarded-For first, falls back to RemoteAddr.
func ClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if idx := strings.IndexByte(fwd, ','); idx != -1 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// ValidateWorkdir checks that a workdir path is safe to use.
// It rejects empty paths, relative paths, paths containing null bytes,
// and known dangerous system directories.
func ValidateWorkdir(path string) error {
	if path == "" {
		return fmt.Errorf("workdir cannot be empty")
	}
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("workdir contains invalid characters")
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		return fmt.Errorf("workdir must be an absolute path")
	}
	// Block dangerous system directories
	dangerousPrefixes := []string{"/proc", "/sys", "/dev", "/boot", "/sbin"}
	dangerousExact := []string{"/", "/etc", "/usr", "/bin", "/var", "/tmp", "/root",
		"/lib", "/lib64", "/opt", "/run"}
	for _, d := range dangerousExact {
		if cleaned == d {
			return fmt.Errorf("workdir cannot be a system directory: %s", cleaned)
		}
	}
	for _, prefix := range dangerousPrefixes {
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
			return fmt.Errorf("workdir cannot be under system directory: %s", prefix)
		}
	}
	return nil
}
