package server

import (
	"database/sql"
	"testing"
	"time"
)

func TestUnit_IsTaskQueued(t *testing.T) {
	if !IsTaskQueued(TaskStatusQueued) {
		t.Error("QUEUED should be queued")
	}
	if IsTaskQueued(TaskStatusClaimed) {
		t.Error("CLAIMED should not be queued")
	}
	if IsTaskQueued(TaskStatusSucceeded) {
		t.Error("SUCCEEDED should not be queued")
	}
}

func TestUnit_IsTaskActive(t *testing.T) {
	if !IsTaskActive(TaskStatusClaimed) {
		t.Error("CLAIMED should be active")
	}
	if !IsTaskActive(TaskStatusRunning) {
		t.Error("RUNNING should be active")
	}
	if IsTaskActive(TaskStatusQueued) {
		t.Error("QUEUED should not be active")
	}
	if IsTaskActive(TaskStatusSucceeded) {
		t.Error("SUCCEEDED should not be active")
	}
}

func TestUnit_IsTaskTerminal(t *testing.T) {
	terminalStatuses := []TaskStatusV2{TaskStatusSucceeded, TaskStatusFailed, TaskStatusCancelled}
	for _, s := range terminalStatuses {
		if !IsTaskTerminal(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminalStatuses := []TaskStatusV2{TaskStatusQueued, TaskStatusClaimed, TaskStatusRunning}
	for _, s := range nonTerminalStatuses {
		if IsTaskTerminal(s) {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestUnit_BucketForStatus(t *testing.T) {
	cases := map[TaskStatusV2]string{
		TaskStatusQueued:    "queued",
		TaskStatusClaimed:   "active",
		TaskStatusRunning:   "active",
		TaskStatusSucceeded: "success",
		TaskStatusFailed:    "failed",
		TaskStatusCancelled: "cancelled",
	}
	for status, expected := range cases {
		got := BucketForStatus(status)
		if got != expected {
			t.Errorf("BucketForStatus(%s) = %s, want %s", status, got, expected)
		}
	}
}

func TestUnit_TaskStatusIsSuccessful(t *testing.T) {
	if !TaskStatusIsSuccessful(TaskStatusSucceeded) {
		t.Error("SUCCEEDED should be successful")
	}
	if TaskStatusIsSuccessful(TaskStatusFailed) {
		t.Error("FAILED should not be successful")
	}
}

func TestUnit_TaskStatusIsFailed(t *testing.T) {
	if !TaskStatusIsFailed(TaskStatusFailed) {
		t.Error("FAILED should be failed")
	}
	if TaskStatusIsFailed(TaskStatusSucceeded) {
		t.Error("SUCCEEDED should not be failed")
	}
}

func TestUnit_RunStatusIsSuccessful(t *testing.T) {
	if !RunStatusIsSuccessful("SUCCEEDED") {
		t.Error("SUCCEEDED should be successful")
	}
	if RunStatusIsSuccessful("FAILED") {
		t.Error("FAILED should not be successful")
	}
}

func TestUnit_RunStatusIsFailed(t *testing.T) {
	if !RunStatusIsFailed("FAILED") {
		t.Error("FAILED should be failed")
	}
	if RunStatusIsFailed("SUCCEEDED") {
		t.Error("SUCCEEDED should not be failed")
	}
}

func TestUnit_IsRunActive(t *testing.T) {
	if !IsRunActive("RUNNING") {
		t.Error("RUNNING should be active")
	}
	if IsRunActive("SUCCEEDED") {
		t.Error("SUCCEEDED should not be active")
	}
}

func TestUnit_IsRunTerminal(t *testing.T) {
	if !IsRunTerminal("SUCCEEDED") {
		t.Error("SUCCEEDED should be terminal")
	}
	if !IsRunTerminal("FAILED") {
		t.Error("FAILED should be terminal")
	}
	if !IsRunTerminal("CANCELLED") {
		t.Error("CANCELLED should be terminal")
	}
	if IsRunTerminal("RUNNING") {
		t.Error("RUNNING should not be terminal")
	}
}

func TestUnit_IsLeaseActiveAt(t *testing.T) {
	future := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339Nano)
	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)
	now := nowISO()

	if !IsLeaseActiveAt(future, now) {
		t.Error("lease expiring in future should be active at now")
	}
	if IsLeaseActiveAt(past, now) {
		t.Error("lease expired in past should not be active at now")
	}
}

func TestUnit_IsLeaseActive(t *testing.T) {
	future := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339Nano)
	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)

	if !IsLeaseActive(future) {
		t.Error("lease with future expiry should be active")
	}
	if IsLeaseActive(past) {
		t.Error("lease with past expiry should not be active")
	}
}

func TestUnit_TaskStatusBuckets(t *testing.T) {
	buckets := TaskStatusBuckets()
	if len(buckets) == 0 {
		t.Fatal("TaskStatusBuckets should return non-empty map")
	}
	// Should have standard buckets
	for _, key := range []string{"queued", "active", "success", "failed"} {
		if _, ok := buckets[key]; !ok {
			t.Errorf("missing bucket: %s", key)
		}
	}
}

func TestUnit_RunStatusBuckets(t *testing.T) {
	buckets := RunStatusBuckets()
	if len(buckets) == 0 {
		t.Fatal("RunStatusBuckets should return non-empty map")
	}
}

func TestUnit_MarshalUnmarshalJSON(t *testing.T) {
	data := map[string]interface{}{"key": "value", "count": float64(42)}
	b, err := marshalJSON(data)
	if err != nil {
		t.Fatalf("marshalJSON: %v", err)
	}

	var result map[string]interface{}
	err = unmarshalJSON(b, &result)
	if err != nil {
		t.Fatalf("unmarshalJSON: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key='value', got %v", result["key"])
	}
}

func TestUnit_NullStringPtrString(t *testing.T) {
	hello := "hello"
	ns := nullString(&hello)
	if !ns.Valid || ns.String != "hello" {
		t.Errorf("nullString(&'hello') should be valid with value 'hello'")
	}

	ns2 := nullString(nil)
	if ns2.Valid {
		t.Errorf("nullString(nil) should not be valid")
	}

	validNS := sql.NullString{String: "world", Valid: true}
	ps := ptrString(validNS)
	if ps == nil || *ps != "world" {
		t.Errorf("ptrString should convert valid NullString to *string")
	}

	invalidNS := sql.NullString{Valid: false}
	ps2 := ptrString(invalidNS)
	if ps2 != nil {
		t.Errorf("ptrString for invalid NullString should return nil")
	}
}
