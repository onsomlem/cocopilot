package config

import (
	"os"
	"testing"
)

func TestGetEnvConfigValue_Default(t *testing.T) {
	v, err := GetEnvConfigValue("COCO_TEST_NONEXISTENT_9999", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if v != "fallback" {
		t.Fatalf("expected fallback, got %q", v)
	}
}

func TestGetEnvConfigValue_Set(t *testing.T) {
	os.Setenv("COCO_TEST_SET", "hello")
	defer os.Unsetenv("COCO_TEST_SET")
	v, err := GetEnvConfigValue("COCO_TEST_SET", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if v != "hello" {
		t.Fatalf("expected hello, got %q", v)
	}
}

func TestGetEnvConfigValue_Empty(t *testing.T) {
	os.Setenv("COCO_TEST_EMPTY", "   ")
	defer os.Unsetenv("COCO_TEST_EMPTY")
	_, err := GetEnvConfigValue("COCO_TEST_EMPTY", "x")
	if err == nil {
		t.Fatal("expected error for empty env var")
	}
}

func TestGetEnvBoolValue_Truthy(t *testing.T) {
	for _, tv := range []string{"1", "true", "yes", "on"} {
		os.Setenv("COCO_TEST_BOOL", tv)
		v, err := GetEnvBoolValue("COCO_TEST_BOOL", false)
		if err != nil {
			t.Fatalf("input %q: %v", tv, err)
		}
		if !v {
			t.Fatalf("input %q: expected true", tv)
		}
	}
	os.Unsetenv("COCO_TEST_BOOL")
}

func TestGetEnvBoolValue_Invalid(t *testing.T) {
	os.Setenv("COCO_TEST_BOOL", "maybe")
	defer os.Unsetenv("COCO_TEST_BOOL")
	_, err := GetEnvBoolValue("COCO_TEST_BOOL", false)
	if err == nil {
		t.Fatal("expected error for invalid bool")
	}
}

func TestParseScopeSet_Valid(t *testing.T) {
	scopes, err := ParseScopeSet("read, write ,admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 3 {
		t.Fatalf("expected 3 scopes, got %d", len(scopes))
	}
}

func TestParseScopeSet_Empty(t *testing.T) {
	_, err := ParseScopeSet("")
	if err == nil {
		t.Fatal("expected error for empty scope set")
	}
}

func TestParseAuthIdentities_Valid(t *testing.T) {
	ids, err := ParseAuthIdentities("bot1|agent|secret1|read,write; svc|service|key2|*")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(ids))
	}
	if ids[0].ID != "bot1" {
		t.Fatalf("unexpected first identity: %+v", ids[0])
	}
}

func TestParseAuthIdentities_InvalidFormat(t *testing.T) {
	_, err := ParseAuthIdentities("bad-entry")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestNormalizeV1EventType(t *testing.T) {
	if v, ok := NormalizeV1EventType("tasks"); !ok || v != "tasks" {
		t.Fatal("expected tasks, true")
	}
	if _, ok := NormalizeV1EventType("unknown"); ok {
		t.Fatal("expected false for unknown")
	}
}
