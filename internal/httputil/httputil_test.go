package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteV2JSON(t *testing.T) {
	w := httptest.NewRecorder()
	WriteV2JSON(w, http.StatusOK, map[string]string{"hello": "world"})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["hello"] != "world" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestWriteV2Error(t *testing.T) {
	w := httptest.NewRecorder()
	WriteV2Error(w, http.StatusNotFound, "not_found", "task not found", nil)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var env V2ErrorEnvelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "not_found" {
		t.Fatalf("unexpected error code: %s", env.Error.Code)
	}
}

func TestClientIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if ip := ClientIP(r); ip != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", ip)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	if ip := ClientIP(r); ip != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %s", ip)
	}
}

func TestClientIP_SingleForwarded(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "9.8.7.6")
	if ip := ClientIP(r); ip != "9.8.7.6" {
		t.Fatalf("expected 9.8.7.6, got %s", ip)
	}
}

func TestValidateWorkdir_Valid(t *testing.T) {
	if err := ValidateWorkdir("/home/user/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkdir_Empty(t *testing.T) {
	if err := ValidateWorkdir(""); err == nil {
		t.Fatal("expected error for empty workdir")
	}
}

func TestValidateWorkdir_Relative(t *testing.T) {
	if err := ValidateWorkdir("relative/path"); err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestValidateWorkdir_SystemDir(t *testing.T) {
	for _, d := range []string{"/", "/etc", "/proc/self", "/dev", "/root"} {
		if err := ValidateWorkdir(d); err == nil {
			t.Fatalf("expected error for system dir %q", d)
		}
	}
}

func TestValidateWorkdir_NullByte(t *testing.T) {
	if err := ValidateWorkdir("/home/user\x00evil"); err == nil {
		t.Fatal("expected error for null byte")
	}
}

func TestWriteV2MethodNotAllowed(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/x", nil)
	WriteV2MethodNotAllowed(w, r, "GET", "POST")
	if w.Code != 405 {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
