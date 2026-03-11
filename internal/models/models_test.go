package models

import (
	"database/sql"
	"testing"
	"time"
)

func TestTaskStatusV2_IsTerminal(t *testing.T) {
	terminal := []TaskStatusV2{TaskStatusSucceeded, TaskStatusFailed, TaskStatusCancelled}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminal := []TaskStatusV2{TaskStatusQueued, TaskStatusClaimed, TaskStatusRunning, TaskStatusNeedsReview}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestTaskStatusV2_IsActive(t *testing.T) {
	active := []TaskStatusV2{TaskStatusClaimed, TaskStatusRunning}
	for _, s := range active {
		if !s.IsActive() {
			t.Errorf("%s should be active", s)
		}
	}
	inactive := []TaskStatusV2{TaskStatusQueued, TaskStatusSucceeded, TaskStatusFailed}
	for _, s := range inactive {
		if s.IsActive() {
			t.Errorf("%s should not be active", s)
		}
	}
}

func TestRunStatus_IsTerminal(t *testing.T) {
	terminal := []RunStatus{RunStatusSucceeded, RunStatusFailed, RunStatusCancelled}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	if RunStatusRunning.IsTerminal() {
		t.Error("RUNNING should not be terminal")
	}
}

func TestBucketForStatus(t *testing.T) {
	tests := []struct {
		status TaskStatusV2
		bucket string
	}{
		{TaskStatusQueued, "queued"},
		{TaskStatusClaimed, "active"},
		{TaskStatusRunning, "active"},
		{TaskStatusSucceeded, "success"},
		{TaskStatusFailed, "failed"},
		{TaskStatusNeedsReview, "review"},
		{TaskStatusCancelled, "cancelled"},
		{TaskStatusV2("BOGUS"), "unknown"},
	}
	for _, tc := range tests {
		got := BucketForStatus(tc.status)
		if got != tc.bucket {
			t.Errorf("BucketForStatus(%s) = %s, want %s", tc.status, got, tc.bucket)
		}
	}
}

func TestNullString_PtrString_Roundtrip(t *testing.T) {
	s := "hello"
	ns := NullString(&s)
	if !ns.Valid || ns.String != "hello" {
		t.Fatalf("NullString(&s) = %+v", ns)
	}
	ptr := PtrString(ns)
	if ptr == nil || *ptr != "hello" {
		t.Fatal("roundtrip failed")
	}
}

func TestNullString_Nil(t *testing.T) {
	ns := NullString(nil)
	if ns.Valid {
		t.Fatal("NullString(nil) should not be valid")
	}
	ptr := PtrString(ns)
	if ptr != nil {
		t.Fatal("PtrString of invalid should be nil")
	}
}

func TestPtrInt64(t *testing.T) {
	ni := sql.NullInt64{Int64: 42, Valid: true}
	p := PtrInt64(ni)
	if p == nil || *p != 42 {
		t.Fatal("PtrInt64 failed for valid")
	}
	p2 := PtrInt64(sql.NullInt64{Valid: false})
	if p2 != nil {
		t.Fatal("PtrInt64 should be nil for invalid")
	}
}

func TestMarshalJSON(t *testing.T) {
	s, err := MarshalJSON(map[string]int{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if s != `{"a":1}` {
		t.Fatalf("unexpected JSON: %s", s)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	var m map[string]int
	if err := UnmarshalJSON(`{"x":5}`, &m); err != nil {
		t.Fatal(err)
	}
	if m["x"] != 5 {
		t.Fatalf("unexpected result: %v", m)
	}
}

func TestNowISO(t *testing.T) {
	now := NowISO()
	parsed, err := time.Parse(LeaseTimeFormat, now)
	if err != nil {
		t.Fatalf("NowISO() returned unparseable time %q: %v", now, err)
	}
	if time.Since(parsed) > 2*time.Second {
		t.Fatalf("NowISO() too far from current time: %s", now)
	}
}

func TestIsLeaseActiveAt(t *testing.T) {
	future := time.Now().UTC().Add(10 * time.Minute).Format(LeaseTimeFormat)
	past := time.Now().UTC().Add(-10 * time.Minute).Format(LeaseTimeFormat)
	now := NowISO()

	if !IsLeaseActiveAt(future, now) {
		t.Error("future lease should be active")
	}
	if IsLeaseActiveAt(past, now) {
		t.Error("past lease should not be active")
	}
}

func TestTaskStatusBuckets(t *testing.T) {
	buckets := TaskStatusBuckets()
	if len(buckets) == 0 {
		t.Fatal("TaskStatusBuckets returned empty map")
	}
	if _, ok := buckets["active"]; !ok {
		t.Fatal("missing 'active' bucket")
	}
	if _, ok := buckets["terminal"]; !ok {
		t.Fatal("missing 'terminal' bucket")
	}
}

func TestRunStatusBuckets(t *testing.T) {
	buckets := RunStatusBuckets()
	if len(buckets) == 0 {
		t.Fatal("RunStatusBuckets returned empty map")
	}
	if _, ok := buckets["active"]; !ok {
		t.Fatal("missing 'active' bucket")
	}
}
