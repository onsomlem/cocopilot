package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func assertV2ErrorEnvelope(t *testing.T, w *httptest.ResponseRecorder, expectedCode string) map[string]interface{} {
	t.Helper()

	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v; body=%s", err, w.Body.String())
	}

	errField, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object in response, got %T (%v)", resp["error"], resp["error"])
	}

	if errField["code"] != expectedCode {
		t.Fatalf("expected error code %q, got %v", expectedCode, errField["code"])
	}

	message, ok := errField["message"].(string)
	if !ok || strings.TrimSpace(message) == "" {
		t.Fatalf("expected non-empty error.message, got %v", errField["message"])
	}

	if details, ok := errField["details"]; ok {
		if _, detailsOK := details.(map[string]interface{}); !detailsOK {
			t.Fatalf("expected error.details to be an object when present, got %T", details)
		}
	}

	return errField
}
