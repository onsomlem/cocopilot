package server

import (
	"strings"
	"testing"
)

func assertTaskUpdatedAtPresent(t *testing.T, task map[string]interface{}) {
	t.Helper()

	value, ok := task["updated_at"]
	if !ok || value == nil {
		t.Fatal("expected task.updated_at to be present")
	}

	updatedAt, ok := value.(string)
	if !ok || strings.TrimSpace(updatedAt) == "" {
		t.Fatalf("expected task.updated_at to be a non-empty string, got %v", value)
	}
}
