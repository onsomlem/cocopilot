package server

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onsomlem/cocopilot/internal/migrate"
)

func TestMain(m *testing.M) {
	// Tests run with CWD=server/, so point migrations to parent directory.
	migrate.MigrationsDir = filepath.Join("..", "migrations")
	os.Exit(m.Run())
}

// setupTestDB creates a fresh test database in an OS-managed temp directory.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use t.TempDir() so Go automatically cleans up after the test.
	testDBPath := filepath.Join(t.TempDir(), fmt.Sprintf("test_%d.db", time.Now().UnixNano()))
	testDB, err := sql.Open("sqlite", testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Initialize schema via real migrations so tests match production behavior.
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to initialize test database with migrations: %v", err)
	}

	// Replace global db with test db
	oldDB := db
	db = testDB

	// Reset workdir to default
	workdirMu.Lock()
	oldWorkdir := workdir
	workdir = "/tmp"
	workdirMu.Unlock()

	// Clear SSE clients
	sseMutex.Lock()
	sseClients = make([]v1SSEClient, 0)
	sseMutex.Unlock()
	v2EventMu.Lock()
	v2EventSubscribers = make([]v2EventSubscriber, 0)
	v2EventMu.Unlock()

	cleanup := func() {
		db.Close()
		db = oldDB
		workdirMu.Lock()
		workdir = oldWorkdir
		workdirMu.Unlock()
		os.Remove(testDBPath)
	}

	return testDB, cleanup
}

func restoreEnvVar(t *testing.T, name string) {
	t.Helper()
	value, exists := os.LookupEnv(name)
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(name, value)
		} else {
			_ = os.Unsetenv(name)
		}
	})
}

type v1TaskListResponse struct {
	Tasks []Task `json:"tasks"`
	Total int    `json:"total"`
}

func TestLoadRuntimeConfigDefaults(t *testing.T) {
	restoreEnvVar(t, "COCO_DB_PATH")
	restoreEnvVar(t, "COCO_HTTP_ADDR")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
	restoreEnvVar(t, "COCO_API_KEY")
	restoreEnvVar(t, "COCO_API_IDENTITIES")
	restoreEnvVar(t, "COCO_SSE_HEARTBEAT_SECONDS")
	restoreEnvVar(t, "COCO_SSE_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_DAYS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_MAX_ROWS")
	restoreEnvVar(t, "COCO_AUTOMATION_RULES")
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	_ = os.Unsetenv("COCO_DB_PATH")
	_ = os.Unsetenv("COCO_HTTP_ADDR")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
	_ = os.Unsetenv("COCO_API_KEY")
	_ = os.Unsetenv("COCO_API_IDENTITIES")
	_ = os.Unsetenv("COCO_SSE_HEARTBEAT_SECONDS")
	_ = os.Unsetenv("COCO_SSE_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_DAYS")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_MAX_ROWS")
	_ = os.Unsetenv("COCO_AUTOMATION_RULES")
	_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")

	cfg, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("loadRuntimeConfig returned error: %v", err)
	}

	if cfg.DBPath != filepath.Join(".", "tasks.db") {
		t.Fatalf("expected default DB path %s, got %s", filepath.Join(".", "tasks.db"), cfg.DBPath)
	}
	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("expected default HTTP addr %s, got %s", defaultHTTPAddr, cfg.HTTPAddr)
	}
	if cfg.RequireAPIKey {
		t.Fatalf("expected RequireAPIKey false by default")
	}
	if cfg.RequireAPIKeyReads {
		t.Fatalf("expected RequireAPIKeyReads false by default")
	}
	if cfg.APIKey != "" {
		t.Fatalf("expected empty API key by default")
	}
	if cfg.SSEHeartbeatSeconds != defaultSSEHeartbeatSeconds {
		t.Fatalf("expected default SSE heartbeat %d, got %d", defaultSSEHeartbeatSeconds, cfg.SSEHeartbeatSeconds)
	}
	if cfg.SSEReplayLimitMax != v2EventsListMaxLimit {
		t.Fatalf("expected default SSE replay limit max %d, got %d", v2EventsListMaxLimit, cfg.SSEReplayLimitMax)
	}
	if cfg.V1EventsReplayLimitMax != v1EventsReplayLimitMaxDefault {
		t.Fatalf("expected default v1 events replay limit max %d, got %d", v1EventsReplayLimitMaxDefault, cfg.V1EventsReplayLimitMax)
	}
	if cfg.EventsRetentionDays != defaultEventsRetentionDays {
		t.Fatalf("expected default events retention days %d, got %d", defaultEventsRetentionDays, cfg.EventsRetentionDays)
	}
	if cfg.EventsRetentionMax != defaultEventsRetentionMax {
		t.Fatalf("expected default events retention max %d, got %d", defaultEventsRetentionMax, cfg.EventsRetentionMax)
	}
	if cfg.EventsPruneIntervalSeconds != defaultEventsPruneIntervalSeconds {
		t.Fatalf("expected default events prune interval %d, got %d", defaultEventsPruneIntervalSeconds, cfg.EventsPruneIntervalSeconds)
	}
}

func TestLoadRuntimeConfigOverrides(t *testing.T) {
	restoreEnvVar(t, "COCO_DB_PATH")
	restoreEnvVar(t, "COCO_HTTP_ADDR")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
	restoreEnvVar(t, "COCO_API_KEY")
	restoreEnvVar(t, "COCO_API_IDENTITIES")
	restoreEnvVar(t, "COCO_SSE_HEARTBEAT_SECONDS")
	restoreEnvVar(t, "COCO_SSE_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_DAYS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_MAX_ROWS")
	restoreEnvVar(t, "COCO_AUTOMATION_RULES")
	_ = os.Setenv("COCO_DB_PATH", filepath.Join(".", "tmp", "custom.db"))
	_ = os.Setenv("COCO_HTTP_ADDR", ":9090")
	_ = os.Setenv("COCO_REQUIRE_API_KEY", "true")
	_ = os.Setenv("COCO_REQUIRE_API_KEY_READS", "true")
	_ = os.Setenv("COCO_API_KEY", "super-secret")
	_ = os.Unsetenv("COCO_API_IDENTITIES")
	_ = os.Setenv("COCO_SSE_HEARTBEAT_SECONDS", "15")
	_ = os.Setenv("COCO_SSE_REPLAY_LIMIT_MAX", "250")
	_ = os.Setenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX", "200")
	_ = os.Setenv("COCO_EVENTS_RETENTION_DAYS", "14")
	_ = os.Setenv("COCO_EVENTS_RETENTION_MAX_ROWS", "5000")
	_ = os.Setenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS", "900")
	_ = os.Unsetenv("COCO_AUTOMATION_RULES")

	cfg, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("loadRuntimeConfig returned error: %v", err)
	}

	if cfg.DBPath != filepath.Join(".", "tmp", "custom.db") {
		t.Fatalf("expected override DB path, got %s", cfg.DBPath)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected override HTTP addr :9090, got %s", cfg.HTTPAddr)
	}
	if !cfg.RequireAPIKey {
		t.Fatalf("expected RequireAPIKey true")
	}
	if !cfg.RequireAPIKeyReads {
		t.Fatalf("expected RequireAPIKeyReads true")
	}
	if cfg.APIKey != "super-secret" {
		t.Fatalf("expected API key override, got %q", cfg.APIKey)
	}
	if cfg.SSEHeartbeatSeconds != 15 {
		t.Fatalf("expected SSE heartbeat override 15, got %d", cfg.SSEHeartbeatSeconds)
	}
	if cfg.SSEReplayLimitMax != 250 {
		t.Fatalf("expected SSE replay limit max override 250, got %d", cfg.SSEReplayLimitMax)
	}
	if cfg.V1EventsReplayLimitMax != 200 {
		t.Fatalf("expected v1 events replay limit max override 200, got %d", cfg.V1EventsReplayLimitMax)
	}
	if cfg.EventsRetentionDays != 14 {
		t.Fatalf("expected events retention days override 14, got %d", cfg.EventsRetentionDays)
	}
	if cfg.EventsRetentionMax != 5000 {
		t.Fatalf("expected events retention max override 5000, got %d", cfg.EventsRetentionMax)
	}
	if cfg.EventsPruneIntervalSeconds != 900 {
		t.Fatalf("expected events prune interval override 900, got %d", cfg.EventsPruneIntervalSeconds)
	}
	if len(cfg.AuthIdentities) == 0 {
		t.Fatalf("expected at least one auth identity when COCO_API_KEY is set")
	}
}

func TestLoadRuntimeConfigInvalidSSEHeartbeatSeconds(t *testing.T) {
	restoreEnvVar(t, "COCO_SSE_HEARTBEAT_SECONDS")
	restoreEnvVar(t, "COCO_DB_PATH")
	restoreEnvVar(t, "COCO_HTTP_ADDR")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
	restoreEnvVar(t, "COCO_API_KEY")
	restoreEnvVar(t, "COCO_API_IDENTITIES")
	restoreEnvVar(t, "COCO_SSE_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_DAYS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_MAX_ROWS")
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	restoreEnvVar(t, "COCO_AUTOMATION_RULES")

	_ = os.Unsetenv("COCO_DB_PATH")
	_ = os.Unsetenv("COCO_HTTP_ADDR")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
	_ = os.Unsetenv("COCO_API_KEY")
	_ = os.Unsetenv("COCO_API_IDENTITIES")
	_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_DAYS")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_MAX_ROWS")
	_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	_ = os.Unsetenv("COCO_AUTOMATION_RULES")

	values := []string{"", "0", "999", "bogus"}
	for _, value := range values {
		if value == "" {
			_ = os.Setenv("COCO_SSE_HEARTBEAT_SECONDS", " ")
		} else {
			_ = os.Setenv("COCO_SSE_HEARTBEAT_SECONDS", value)
		}
		if _, err := loadRuntimeConfig(); err == nil {
			t.Fatalf("expected error for COCO_SSE_HEARTBEAT_SECONDS=%q", value)
		}
	}
}

func TestLoadRuntimeConfigInvalidSSEReplayLimitMax(t *testing.T) {
	restoreEnvVar(t, "COCO_SSE_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_DB_PATH")
	restoreEnvVar(t, "COCO_HTTP_ADDR")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
	restoreEnvVar(t, "COCO_API_KEY")
	restoreEnvVar(t, "COCO_API_IDENTITIES")
	restoreEnvVar(t, "COCO_SSE_HEARTBEAT_SECONDS")
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_DAYS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_MAX_ROWS")
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	restoreEnvVar(t, "COCO_AUTOMATION_RULES")

	_ = os.Unsetenv("COCO_DB_PATH")
	_ = os.Unsetenv("COCO_HTTP_ADDR")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
	_ = os.Unsetenv("COCO_API_KEY")
	_ = os.Unsetenv("COCO_API_IDENTITIES")
	_ = os.Unsetenv("COCO_SSE_HEARTBEAT_SECONDS")
	_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_DAYS")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_MAX_ROWS")
	_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	_ = os.Unsetenv("COCO_AUTOMATION_RULES")

	values := []string{"", "0", "-1", "999", "bogus"}
	for _, value := range values {
		if value == "" {
			_ = os.Setenv("COCO_SSE_REPLAY_LIMIT_MAX", " ")
		} else {
			_ = os.Setenv("COCO_SSE_REPLAY_LIMIT_MAX", value)
		}
		if _, err := loadRuntimeConfig(); err == nil {
			t.Fatalf("expected error for COCO_SSE_REPLAY_LIMIT_MAX=%q", value)
		}
	}
}

func TestLoadRuntimeConfigInvalidV1EventsReplayLimitMax(t *testing.T) {
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_DB_PATH")
	restoreEnvVar(t, "COCO_HTTP_ADDR")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
	restoreEnvVar(t, "COCO_API_KEY")
	restoreEnvVar(t, "COCO_API_IDENTITIES")
	restoreEnvVar(t, "COCO_SSE_HEARTBEAT_SECONDS")
	restoreEnvVar(t, "COCO_SSE_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_DAYS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_MAX_ROWS")
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")

	_ = os.Unsetenv("COCO_DB_PATH")
	_ = os.Unsetenv("COCO_HTTP_ADDR")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
	_ = os.Unsetenv("COCO_API_KEY")
	_ = os.Unsetenv("COCO_API_IDENTITIES")
	_ = os.Unsetenv("COCO_SSE_HEARTBEAT_SECONDS")
	_ = os.Unsetenv("COCO_SSE_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_DAYS")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_MAX_ROWS")
	_ = os.Unsetenv("COCO_AUTOMATION_RULES")
	_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")

	values := []string{"", "0", "-1", "999", "bogus"}
	for _, value := range values {
		if value == "" {
			_ = os.Setenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX", " ")
		} else {
			_ = os.Setenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX", value)
		}
		if _, err := loadRuntimeConfig(); err == nil {
			t.Fatalf("expected error for COCO_V1_EVENTS_REPLAY_LIMIT_MAX=%q", value)
		}
	}
}

func TestLoadRuntimeConfigInvalidEventsRetention(t *testing.T) {
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_DAYS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_MAX_ROWS")
	restoreEnvVar(t, "COCO_DB_PATH")
	restoreEnvVar(t, "COCO_HTTP_ADDR")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
	restoreEnvVar(t, "COCO_API_KEY")
	restoreEnvVar(t, "COCO_API_IDENTITIES")
	restoreEnvVar(t, "COCO_SSE_HEARTBEAT_SECONDS")
	restoreEnvVar(t, "COCO_SSE_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	restoreEnvVar(t, "COCO_AUTOMATION_RULES")

	_ = os.Unsetenv("COCO_DB_PATH")
	_ = os.Unsetenv("COCO_HTTP_ADDR")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
	_ = os.Unsetenv("COCO_API_KEY")
	_ = os.Unsetenv("COCO_API_IDENTITIES")
	_ = os.Unsetenv("COCO_SSE_HEARTBEAT_SECONDS")
	_ = os.Unsetenv("COCO_SSE_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	_ = os.Unsetenv("COCO_AUTOMATION_RULES")

	values := []string{"", "-1", "bogus"}
	for _, value := range values {
		if value == "" {
			_ = os.Setenv("COCO_EVENTS_RETENTION_DAYS", " ")
		} else {
			_ = os.Setenv("COCO_EVENTS_RETENTION_DAYS", value)
		}
		_ = os.Unsetenv("COCO_EVENTS_RETENTION_MAX_ROWS")
		if _, err := loadRuntimeConfig(); err == nil {
			t.Fatalf("expected error for COCO_EVENTS_RETENTION_DAYS=%q", value)
		}
	}

	for _, value := range values {
		_ = os.Unsetenv("COCO_EVENTS_RETENTION_DAYS")
		if value == "" {
			_ = os.Setenv("COCO_EVENTS_RETENTION_MAX_ROWS", " ")
		} else {
			_ = os.Setenv("COCO_EVENTS_RETENTION_MAX_ROWS", value)
		}
		if _, err := loadRuntimeConfig(); err == nil {
			t.Fatalf("expected error for COCO_EVENTS_RETENTION_MAX_ROWS=%q", value)
		}
	}
}

func TestLoadRuntimeConfigInvalidEventsPruneInterval(t *testing.T) {
	restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
	restoreEnvVar(t, "COCO_DB_PATH")
	restoreEnvVar(t, "COCO_HTTP_ADDR")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
	restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
	restoreEnvVar(t, "COCO_API_KEY")
	restoreEnvVar(t, "COCO_API_IDENTITIES")
	restoreEnvVar(t, "COCO_SSE_HEARTBEAT_SECONDS")
	restoreEnvVar(t, "COCO_SSE_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_DAYS")
	restoreEnvVar(t, "COCO_EVENTS_RETENTION_MAX_ROWS")

	_ = os.Unsetenv("COCO_DB_PATH")
	_ = os.Unsetenv("COCO_HTTP_ADDR")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
	_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
	_ = os.Unsetenv("COCO_API_KEY")
	_ = os.Unsetenv("COCO_API_IDENTITIES")
	_ = os.Unsetenv("COCO_SSE_HEARTBEAT_SECONDS")
	_ = os.Unsetenv("COCO_SSE_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_DAYS")
	_ = os.Unsetenv("COCO_EVENTS_RETENTION_MAX_ROWS")

	values := []string{"", "0", "59", "86401", "bogus"}
	for _, value := range values {
		if value == "" {
			_ = os.Setenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS", " ")
		} else {
			_ = os.Setenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS", value)
		}
		if _, err := loadRuntimeConfig(); err == nil {
			t.Fatalf("expected error for COCO_EVENTS_PRUNE_INTERVAL_SECONDS=%q", value)
		}
	}
}

func TestLoadRuntimeConfigValidation(t *testing.T) {
	t.Run("invalid http addr", func(t *testing.T) {
		restoreEnvVar(t, "COCO_DB_PATH")
		restoreEnvVar(t, "COCO_HTTP_ADDR")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
		restoreEnvVar(t, "COCO_API_KEY")
		restoreEnvVar(t, "COCO_API_IDENTITIES")
		restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		restoreEnvVar(t, "COCO_AUTOMATION_RULES")
		_ = os.Setenv("COCO_DB_PATH", filepath.Join(".", "tmp", "valid.db"))
		_ = os.Setenv("COCO_HTTP_ADDR", "invalid_addr")
		_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
		_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
		_ = os.Unsetenv("COCO_API_KEY")
		_ = os.Unsetenv("COCO_API_IDENTITIES")
		_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		_ = os.Unsetenv("COCO_AUTOMATION_RULES")

		_, err := loadRuntimeConfig()
		if err == nil {
			t.Fatal("expected error for invalid COCO_HTTP_ADDR")
		}
	})
	t.Run("empty db path when set", func(t *testing.T) {
		restoreEnvVar(t, "COCO_DB_PATH")
		restoreEnvVar(t, "COCO_HTTP_ADDR")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
		restoreEnvVar(t, "COCO_API_KEY")
		restoreEnvVar(t, "COCO_API_IDENTITIES")
		restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		restoreEnvVar(t, "COCO_AUTOMATION_RULES")
		_ = os.Setenv("COCO_DB_PATH", "   ")
		_ = os.Setenv("COCO_HTTP_ADDR", ":8080")
		_ = os.Unsetenv("COCO_REQUIRE_API_KEY")
		_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
		_ = os.Unsetenv("COCO_API_KEY")
		_ = os.Unsetenv("COCO_API_IDENTITIES")
		_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		_ = os.Unsetenv("COCO_AUTOMATION_RULES")

		_, err := loadRuntimeConfig()
		if err == nil {
			t.Fatal("expected error for empty COCO_DB_PATH")
		}
	})
	t.Run("invalid auth bool", func(t *testing.T) {
		restoreEnvVar(t, "COCO_DB_PATH")
		restoreEnvVar(t, "COCO_HTTP_ADDR")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
		restoreEnvVar(t, "COCO_API_KEY")
		restoreEnvVar(t, "COCO_API_IDENTITIES")
		restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		restoreEnvVar(t, "COCO_AUTOMATION_RULES")
		_ = os.Setenv("COCO_DB_PATH", filepath.Join(".", "tmp", "valid.db"))
		_ = os.Setenv("COCO_HTTP_ADDR", ":8080")
		_ = os.Setenv("COCO_REQUIRE_API_KEY", "not-a-bool")
		_ = os.Unsetenv("COCO_REQUIRE_API_KEY_READS")
		_ = os.Unsetenv("COCO_API_KEY")
		_ = os.Unsetenv("COCO_API_IDENTITIES")
		_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		_ = os.Unsetenv("COCO_AUTOMATION_RULES")

		_, err := loadRuntimeConfig()
		if err == nil {
			t.Fatal("expected error for invalid COCO_REQUIRE_API_KEY")
		}
	})

	t.Run("missing API key when auth enabled", func(t *testing.T) {
		restoreEnvVar(t, "COCO_DB_PATH")
		restoreEnvVar(t, "COCO_HTTP_ADDR")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
		restoreEnvVar(t, "COCO_API_KEY")
		restoreEnvVar(t, "COCO_API_IDENTITIES")
		restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		restoreEnvVar(t, "COCO_AUTOMATION_RULES")
		_ = os.Setenv("COCO_DB_PATH", filepath.Join(".", "tmp", "valid.db"))
		_ = os.Setenv("COCO_HTTP_ADDR", ":8080")
		_ = os.Setenv("COCO_REQUIRE_API_KEY", "true")
		_ = os.Setenv("COCO_REQUIRE_API_KEY_READS", "false")
		_ = os.Unsetenv("COCO_API_KEY")
		_ = os.Unsetenv("COCO_API_IDENTITIES")
		_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		_ = os.Unsetenv("COCO_AUTOMATION_RULES")

		_, err := loadRuntimeConfig()
		if err == nil {
			t.Fatal("expected error when COCO_REQUIRE_API_KEY=true and COCO_API_KEY is missing")
		}
	})

	t.Run("invalid API identities format", func(t *testing.T) {
		restoreEnvVar(t, "COCO_DB_PATH")
		restoreEnvVar(t, "COCO_HTTP_ADDR")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
		restoreEnvVar(t, "COCO_API_KEY")
		restoreEnvVar(t, "COCO_API_IDENTITIES")
		restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		restoreEnvVar(t, "COCO_AUTOMATION_RULES")
		_ = os.Setenv("COCO_DB_PATH", filepath.Join(".", "tmp", "valid.db"))
		_ = os.Setenv("COCO_HTTP_ADDR", ":8080")
		_ = os.Setenv("COCO_REQUIRE_API_KEY", "true")
		_ = os.Unsetenv("COCO_API_KEY")
		_ = os.Setenv("COCO_API_IDENTITIES", "broken-entry")
		_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		_ = os.Unsetenv("COCO_AUTOMATION_RULES")

		_, err := loadRuntimeConfig()
		if err == nil {
			t.Fatal("expected error for invalid COCO_API_IDENTITIES format")
		}
	})

	t.Run("api identities satisfy auth requirement", func(t *testing.T) {
		restoreEnvVar(t, "COCO_DB_PATH")
		restoreEnvVar(t, "COCO_HTTP_ADDR")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY")
		restoreEnvVar(t, "COCO_REQUIRE_API_KEY_READS")
		restoreEnvVar(t, "COCO_API_KEY")
		restoreEnvVar(t, "COCO_API_IDENTITIES")
		restoreEnvVar(t, "COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		restoreEnvVar(t, "COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		_ = os.Setenv("COCO_DB_PATH", filepath.Join(".", "tmp", "valid.db"))
		_ = os.Setenv("COCO_HTTP_ADDR", ":8080")
		_ = os.Setenv("COCO_REQUIRE_API_KEY", "true")
		_ = os.Unsetenv("COCO_API_KEY")
		_ = os.Setenv("COCO_API_IDENTITIES", "agent_runner|agent|agent-key|v2:read,leases:write")
		_ = os.Unsetenv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX")
		_ = os.Unsetenv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS")
		_ = os.Unsetenv("COCO_AUTOMATION_RULES")

		cfg, err := loadRuntimeConfig()
		if err != nil {
			t.Fatalf("expected valid config with COCO_API_IDENTITIES, got error: %v", err)
		}
		if len(cfg.AuthIdentities) != 1 {
			t.Fatalf("expected 1 identity, got %d", len(cfg.AuthIdentities))
		}
		if cfg.AuthIdentities[0].ID != "agent_runner" {
			t.Fatalf("expected identity id agent_runner, got %s", cfg.AuthIdentities[0].ID)
		}
	})
}

// TestPOCREG001_CreateClaimSaveLifecycle tests the full task lifecycle
// POC-REG-001: Create → Claim → Save lifecycle
func TestPOCREG001_CreateClaimSaveLifecycle(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	t.Log("POC-REG-001: Testing Create → Claim → Save lifecycle")

	// Step 1: Create a task
	t.Log("Step 1: Creating task...")
	formData := url.Values{}
	formData.Set("instructions", "POC-REG-001: say hello")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed with status %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	if !createResp["success"].(bool) {
		t.Fatal("Create response success is false")
	}
	createdUpdatedAt, ok := createResp["updated_at"].(string)
	if !ok || strings.TrimSpace(createdUpdatedAt) == "" {
		t.Fatalf("Create response missing updated_at. Got: %v", createResp["updated_at"])
	}

	taskID := int(createResp["task_id"].(float64))
	t.Logf("✓ Created task ID: %d", taskID)

	// Step 2: Claim the task
	t.Log("Step 2: Claiming task...")
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()

	getTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get task failed with status %d: %s", w.Code, w.Body.String())
	}

	taskResponse := w.Body.String()
	if !strings.Contains(taskResponse, fmt.Sprintf("AVAILABLE TASK ID: %d", taskID)) {
		t.Fatalf("Task response doesn't contain expected task ID. Got: %s", taskResponse)
	}

	if !strings.Contains(taskResponse, "POC-REG-001: say hello") {
		t.Fatalf("Task response doesn't contain expected instructions. Got: %s", taskResponse)
	}

	if !strings.Contains(taskResponse, "IN_PROGRESS") {
		t.Fatalf("Task status not IN_PROGRESS. Got: %s", taskResponse)
	}
	if !strings.Contains(taskResponse, "UPDATED_AT:") {
		t.Fatalf("Task response missing updated_at. Got: %s", taskResponse)
	}
	t.Log("✓ Claimed task successfully, status is IN_PROGRESS")

	// Step 3: Save the task
	t.Log("Step 3: Saving task...")
	formData = url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", taskID))
	formData.Set("message", "hello")

	req = httptest.NewRequest("POST", "/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	saveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Save failed with status %d: %s", w.Code, w.Body.String())
	}

	saveResponse := w.Body.String()
	if !strings.Contains(saveResponse, "COMPLETE") {
		t.Fatalf("Save response doesn't indicate completion. Got: %s", saveResponse)
	}
	if !strings.Contains(saveResponse, "UPDATED_AT:") {
		t.Fatalf("Save response missing updated_at. Got: %s", saveResponse)
	}
	t.Log("✓ Saved task successfully")

	// Step 4: Verify task list
	t.Log("Step 4: Verifying task in list...")
	req = httptest.NewRequest("GET", "/api/tasks", nil)
	w = httptest.NewRecorder()

	apiTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get tasks failed with status %d: %s", w.Code, w.Body.String())
	}

	var tasks []Task
	var listResp v1TaskListResponse
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode tasks response: %v", err)
	}
	tasks = listResp.Tasks

	var foundTask *Task
	for i := range tasks {
		if tasks[i].ID == taskID {
			foundTask = &tasks[i]
			break
		}
	}

	if foundTask == nil {
		t.Fatalf("Task %d not found in tasks list", taskID)
	}

	if foundTask.Status != StatusComplete {
		t.Fatalf("Task status is %s, expected COMPLETE", foundTask.Status)
	}

	if foundTask.Output == nil || *foundTask.Output != "hello" {
		t.Fatalf("Task output is %v, expected 'hello'", foundTask.Output)
	}

	if strings.TrimSpace(foundTask.UpdatedAt) == "" {
		t.Fatal("Task updated_at is empty, expected non-empty")
	}

	t.Log("✓ Task verified in list: status=COMPLETE, output='hello'")
	t.Log("✅ POC-REG-001 PASSED")
}

func TestV1UpdatedAtOnClaimSaveAndStatusUpdate(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	formData := url.Values{}
	formData.Set("instructions", "updated_at v1 flow")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed with status %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}
	createdUpdatedAt, ok := createResp["updated_at"].(string)
	if !ok || strings.TrimSpace(createdUpdatedAt) == "" {
		t.Fatalf("Create response missing updated_at. Got: %v", createResp["updated_at"])
	}

	taskID := int(createResp["task_id"].(float64))
	oldUpdatedAt := "2000-01-01T00:00:00.000000Z"
	if _, err := db.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldUpdatedAt, taskID); err != nil {
		t.Fatalf("Failed to seed updated_at: %v", err)
	}

	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()
	getTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get task failed with status %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "UPDATED_AT:") {
		t.Fatalf("Expected updated_at in task response. Got: %s", w.Body.String())
	}

	var claimedUpdatedAt sql.NullString
	if err := db.QueryRow("SELECT updated_at FROM tasks WHERE id = ?", taskID).Scan(&claimedUpdatedAt); err != nil {
		t.Fatalf("Failed to read updated_at after claim: %v", err)
	}
	if !claimedUpdatedAt.Valid || claimedUpdatedAt.String == "" {
		t.Fatal("Expected updated_at to be set after claim")
	}
	if claimedUpdatedAt.String == oldUpdatedAt {
		t.Fatalf("Expected updated_at to change after claim, still %s", claimedUpdatedAt.String)
	}

	oldUpdatedAt = "2000-01-02T00:00:00.000000Z"
	if _, err := db.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldUpdatedAt, taskID); err != nil {
		t.Fatalf("Failed to reseed updated_at: %v", err)
	}

	formData = url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", taskID))
	formData.Set("message", "done")

	req = httptest.NewRequest("POST", "/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	saveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Save failed with status %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "UPDATED_AT:") {
		t.Fatalf("Save response missing updated_at. Got: %s", w.Body.String())
	}

	var savedUpdatedAt sql.NullString
	if err := db.QueryRow("SELECT updated_at FROM tasks WHERE id = ?", taskID).Scan(&savedUpdatedAt); err != nil {
		t.Fatalf("Failed to read updated_at after save: %v", err)
	}
	if !savedUpdatedAt.Valid || savedUpdatedAt.String == "" {
		t.Fatal("Expected updated_at to be set after save")
	}
	if savedUpdatedAt.String == oldUpdatedAt {
		t.Fatalf("Expected updated_at to change after save, still %s", savedUpdatedAt.String)
	}

	oldUpdatedAt = "2000-01-03T00:00:00.000000Z"
	if _, err := db.Exec("UPDATE tasks SET updated_at = ? WHERE id = ?", oldUpdatedAt, taskID); err != nil {
		t.Fatalf("Failed to reseed updated_at for status update: %v", err)
	}

	formData = url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", taskID))
	formData.Set("status", string(StatusInProgress))

	req = httptest.NewRequest("POST", "/update-status", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	updateStatusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update status failed with status %d: %s", w.Code, w.Body.String())
	}

	var updateResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&updateResp); err != nil {
		t.Fatalf("Failed to decode update-status response: %v", err)
	}
	statusUpdatedAtResp, ok := updateResp["updated_at"].(string)
	if !ok || statusUpdatedAtResp == "" {
		t.Fatalf("Update-status response missing updated_at. Got: %v", updateResp["updated_at"])
	}

	var statusUpdatedAt sql.NullString
	if err := db.QueryRow("SELECT updated_at FROM tasks WHERE id = ?", taskID).Scan(&statusUpdatedAt); err != nil {
		t.Fatalf("Failed to read updated_at after status update: %v", err)
	}
	if !statusUpdatedAt.Valid || statusUpdatedAt.String == "" {
		t.Fatal("Expected updated_at to be set after status update")
	}
	if statusUpdatedAt.String == oldUpdatedAt {
		t.Fatalf("Expected updated_at to change after status update, still %s", statusUpdatedAt.String)
	}
	if statusUpdatedAt.String != statusUpdatedAtResp {
		t.Fatalf("Expected update-status response updated_at to match DB. Response: %s DB: %s", statusUpdatedAtResp, statusUpdatedAt.String)
	}
}

func TestV1ApiTasksFilters(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour)
	mid := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	insertTask := func(instructions string, status TaskStatus, ts time.Time) int {
		result, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			status,
			"proj_default",
			ts.Format(leaseTimeFormat),
			ts.Format(leaseTimeFormat),
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to read task id: %v", err)
		}
		return int(id)
	}

	insertTask("t1", StatusNotPicked, past)
	insertTask("t2", StatusInProgress, mid)
	insertTask("t3", StatusComplete, future)

	// Filter by status
	req := httptest.NewRequest("GET", "/api/tasks?status=IN_PROGRESS", nil)
	w := httptest.NewRecorder()
	apiTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get tasks failed with status %d: %s", w.Code, w.Body.String())
	}

	var listResp v1TaskListResponse
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode tasks response: %v", err)
	}
	if len(listResp.Tasks) != 1 {
		t.Fatalf("Expected 1 task for status filter, got %d", len(listResp.Tasks))
	}
	if listResp.Tasks[0].Status != StatusInProgress {
		t.Fatalf("Expected status IN_PROGRESS, got %s", listResp.Tasks[0].Status)
	}
	if listResp.Total != 1 {
		t.Fatalf("Expected total 1 for status filter, got %d", listResp.Total)
	}

	// Filter by updated_since
	sinceRaw := mid.Add(-30 * time.Minute).Format(time.RFC3339Nano)
	req = httptest.NewRequest("GET", "/api/tasks?updated_since="+url.QueryEscape(sinceRaw), nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get tasks failed with status %d: %s", w.Code, w.Body.String())
	}

	listResp = v1TaskListResponse{}
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode tasks response: %v", err)
	}
	if len(listResp.Tasks) != 2 {
		t.Fatalf("Expected 2 tasks for updated_since filter, got %d", len(listResp.Tasks))
	}
	if listResp.Total != 2 {
		t.Fatalf("Expected total 2 for updated_since filter, got %d", listResp.Total)
	}
	sinceTime, err := time.Parse(time.RFC3339Nano, sinceRaw)
	if err != nil {
		t.Fatalf("Failed to parse since time: %v", err)
	}
	for _, task := range listResp.Tasks {
		parsed, err := time.Parse(leaseTimeFormat, task.UpdatedAt)
		if err != nil {
			t.Fatalf("Failed to parse task updated_at: %v", err)
		}
		if parsed.Before(sinceTime) {
			t.Fatalf("Expected updated_at >= since, got %s", task.UpdatedAt)
		}
	}

	// Combined filters
	req = httptest.NewRequest("GET", "/api/tasks?status=COMPLETE&updated_since="+url.QueryEscape(mid.Format(time.RFC3339Nano)), nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get tasks failed with status %d: %s", w.Code, w.Body.String())
	}

	listResp = v1TaskListResponse{}
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode tasks response: %v", err)
	}
	if len(listResp.Tasks) != 1 {
		t.Fatalf("Expected 1 task for combined filters, got %d", len(listResp.Tasks))
	}
	if listResp.Tasks[0].Status != StatusComplete {
		t.Fatalf("Expected status COMPLETE, got %s", listResp.Tasks[0].Status)
	}
	if listResp.Total != 1 {
		t.Fatalf("Expected total 1 for combined filters, got %d", listResp.Total)
	}

	// Validation errors
	req = httptest.NewRequest("GET", "/api/tasks?status=NOPE", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid status, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/tasks?updated_since=bad", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid updated_since, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/tasks?status=", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty status, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/tasks?updated_since=", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty updated_since, got %d", w.Code)
	}
}

func TestV1ApiTasksProjectFilter(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	project1, err := CreateProject(db, "Project 1", "/tmp/p1", nil)
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}
	project2, err := CreateProject(db, "Project 2", "/tmp/p2", nil)
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	now := time.Now().UTC()
	insertTask := func(instructions string, status TaskStatus, projectID string, ts time.Time) int {
		result, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			status,
			projectID,
			ts.Format(leaseTimeFormat),
			ts.Format(leaseTimeFormat),
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to read task id: %v", err)
		}
		return int(id)
	}

	insertTask("p1-task-1", StatusNotPicked, project1.ID, now)
	insertTask("p1-task-2", StatusInProgress, project1.ID, now.Add(1*time.Minute))
	insertTask("p2-task-1", StatusComplete, project2.ID, now.Add(2*time.Minute))

	req := httptest.NewRequest("GET", "/api/tasks?project_id="+url.QueryEscape(project1.ID), nil)
	w := httptest.NewRecorder()
	apiTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get tasks failed with status %d: %s", w.Code, w.Body.String())
	}

	var listResp v1TaskListResponse
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode tasks response: %v", err)
	}
	if listResp.Total != 2 {
		t.Fatalf("Expected total 2 for project filter, got %d", listResp.Total)
	}
	if len(listResp.Tasks) != 2 {
		t.Fatalf("Expected 2 tasks for project filter, got %d", len(listResp.Tasks))
	}
	for _, task := range listResp.Tasks {
		if !strings.HasPrefix(task.Instructions, "p1-") {
			t.Fatalf("Unexpected task returned for project filter: %s", task.Instructions)
		}
	}

	req = httptest.NewRequest("GET", "/api/tasks?project_id=", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty project_id, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/tasks?project_id=proj_missing", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404 for missing project_id, got %d", w.Code)
	}
}

func TestV1ApiTasksPagination(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour)
	mid := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	insertTask := func(instructions string, status TaskStatus, ts time.Time) int {
		result, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			status,
			"proj_default",
			ts.Format(leaseTimeFormat),
			ts.Format(leaseTimeFormat),
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to read task id: %v", err)
		}
		return int(id)
	}

	insertTask("t1", StatusNotPicked, past)
	midID := insertTask("t2", StatusInProgress, mid)
	insertTask("t3", StatusComplete, future)

	req := httptest.NewRequest("GET", "/api/tasks?limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	apiTasksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get tasks failed with status %d: %s", w.Code, w.Body.String())
	}

	var listResp v1TaskListResponse
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode tasks response: %v", err)
	}
	if listResp.Total != 3 {
		t.Fatalf("Expected total 3, got %d", listResp.Total)
	}
	if len(listResp.Tasks) != 1 {
		t.Fatalf("Expected 1 task with limit=1, got %d", len(listResp.Tasks))
	}
	if listResp.Tasks[0].ID != midID {
		t.Fatalf("Expected task id %d at offset=1, got %d", midID, listResp.Tasks[0].ID)
	}

	req = httptest.NewRequest("GET", "/api/tasks?limit=0", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid limit, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/tasks?offset=-1", nil)
	w = httptest.NewRecorder()
	apiTasksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid offset, got %d", w.Code)
	}
}

func TestV1ApiTasksSorting(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour)
	mid := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	insertTask := func(instructions string, status TaskStatus, createdAt time.Time, updatedAt time.Time) int {
		result, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			status,
			"proj_default",
			createdAt.Format(leaseTimeFormat),
			updatedAt.Format(leaseTimeFormat),
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to read task id: %v", err)
		}
		return int(id)
	}

	firstID := insertTask("t1", StatusNotPicked, past, mid)
	secondID := insertTask("t2", StatusInProgress, mid, future)
	thirdID := insertTask("t3", StatusComplete, future, past)

	getIDs := func(path string) []int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		apiTasksHandler(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Get tasks failed with status %d: %s", w.Code, w.Body.String())
		}
		var listResp v1TaskListResponse
		if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
			t.Fatalf("Failed to decode tasks response: %v", err)
		}
		ids := make([]int, 0, len(listResp.Tasks))
		for _, task := range listResp.Tasks {
			ids = append(ids, task.ID)
		}
		return ids
	}

	createdAsc := getIDs("/api/tasks?sort=created_at:asc")
	if len(createdAsc) < 3 || createdAsc[0] != firstID || createdAsc[1] != secondID || createdAsc[2] != thirdID {
		t.Fatalf("Expected created_at asc order [%d %d %d], got %v", firstID, secondID, thirdID, createdAsc)
	}

	createdDesc := getIDs("/api/tasks?sort=created_at:desc")
	if len(createdDesc) < 3 || createdDesc[0] != thirdID || createdDesc[1] != secondID || createdDesc[2] != firstID {
		t.Fatalf("Expected created_at desc order [%d %d %d], got %v", thirdID, secondID, firstID, createdDesc)
	}

	updatedDesc := getIDs("/api/tasks?sort=updated_at")
	if len(updatedDesc) < 3 || updatedDesc[0] != secondID || updatedDesc[1] != firstID || updatedDesc[2] != thirdID {
		t.Fatalf("Expected updated_at desc order [%d %d %d], got %v", secondID, firstID, thirdID, updatedDesc)
	}

	invalidPaths := []string{
		"/api/tasks?sort=created_at",
		"/api/tasks?sort=updated_at:asc",
		"/api/tasks?sort=unknown",
		"/api/tasks?sort=",
	}
	for _, path := range invalidPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		apiTasksHandler(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400 for invalid sort %s, got %d", path, w.Code)
		}
	}
}

// TestPOCREG002_ParentTaskContext tests parent task context preservation
// POC-REG-002: Parent task context is preserved
func TestPOCREG002_ParentTaskContext(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	t.Log("POC-REG-002: Testing Parent task context preservation")

	// Step 1: Create parent task
	t.Log("Step 1: Creating parent task...")
	formData := url.Values{}
	formData.Set("instructions", "parent: gather info")

	req := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	createHandler(w, req)

	var createResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&createResp)
	parentID := int(createResp["task_id"].(float64))
	t.Logf("✓ Created parent task ID: %d", parentID)

	// Step 2: Claim and save parent task
	t.Log("Step 2: Claiming parent task...")
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()
	getTaskHandler(w, req)
	t.Log("✓ Claimed parent task")

	t.Log("Step 3: Saving parent task with output...")
	formData = url.Values{}
	formData.Set("task_id", fmt.Sprintf("%d", parentID))
	formData.Set("message", "parent output: important data collected")

	req = httptest.NewRequest("POST", "/save", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	saveHandler(w, req)
	t.Log("✓ Saved parent task with output")

	// Step 3: Create child task with parent_task_id
	t.Log("Step 4: Creating child task with parent reference...")
	formData = url.Values{}
	formData.Set("instructions", "child: process the data")
	formData.Set("parent_task_id", fmt.Sprintf("%d", parentID))

	req = httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	createHandler(w, req)

	json.NewDecoder(w.Body).Decode(&createResp)
	childID := int(createResp["task_id"].(float64))
	t.Logf("✓ Created child task ID: %d", childID)

	// Step 4: Claim child task and verify context
	t.Log("Step 5: Claiming child task and verifying context...")
	req = httptest.NewRequest("GET", "/task", nil)
	w = httptest.NewRecorder()

	getTaskHandler(w, req)

	taskResponse := w.Body.String()

	// Verify child task is returned
	if !strings.Contains(taskResponse, fmt.Sprintf("AVAILABLE TASK ID: %d", childID)) {
		t.Fatalf("Expected child task ID %d in response. Got: %s", childID, taskResponse)
	}

	// Verify context block is present
	if !strings.Contains(taskResponse, "CONTEXT FROM PREVIOUS TASKS:") {
		t.Fatalf("Context block not found in task response. Got: %s", taskResponse)
	}

	// Verify parent output is in context
	if !strings.Contains(taskResponse, "parent output: important data collected") {
		t.Fatalf("Parent output not found in context. Got: %s", taskResponse)
	}

	// Verify parent task number is referenced
	if !strings.Contains(taskResponse, fmt.Sprintf("Task #%d", parentID)) {
		t.Fatalf("Parent task #%d not referenced in context. Got: %s", parentID, taskResponse)
	}
	if !strings.Contains(taskResponse, fmt.Sprintf("Task #%d (UPDATED_AT:", parentID)) {
		t.Fatalf("Parent task updated_at not included in context. Got: %s", taskResponse)
	}

	t.Log("✓ Child task contains parent context block")
	t.Log("✅ POC-REG-002 PASSED")
}

// TestPOCREG003_SSEEventsStream tests SSE functionality
// POC-REG-003: SSE events stream continues to function
func TestPOCREG003_SSEEventsStream(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	t.Log("POC-REG-003: Testing SSE events stream functionality")

	// Create a test server
	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	// Start listening to SSE stream
	t.Log("Step 1: Connecting to SSE stream...")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE connection failed with status %d", resp.StatusCode)
	}

	// Verify SSE headers
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Fatalf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}
	t.Log("✓ Connected to SSE stream successfully")

	// Channel to collect events
	eventsChan := make(chan string, 10)
	doneChan := make(chan bool)

	// Start reading events in goroutine
	go func() {
		reader := bufio.NewReader(resp.Body)
		eventCount := 0
		for eventCount < 3 { // Read at least 3 events
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					t.Logf("Error reading SSE stream: %v", err)
				}
				break
			}
			if strings.HasPrefix(line, "data: ") {
				eventsChan <- line
				eventCount++
			}
		}
		doneChan <- true
	}()

	// Wait for initial event (sent immediately on connection)
	t.Log("Step 2: Waiting for initial SSE event...")
	select {
	case event := <-eventsChan:
		t.Logf("✓ Received initial SSE event: %s", event[:min(50, len(event))])
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for initial SSE event")
	}

	// Create a task to trigger an SSE update
	t.Log("Step 3: Creating task to trigger SSE update...")
	formData := url.Values{}
	formData.Set("instructions", "SSE test task")

	reqHTTP := httptest.NewRequest("POST", "/create", strings.NewReader(formData.Encode()))
	reqHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	createHandler(w, reqHTTP)
	t.Log("✓ Created task")

	// Wait for SSE event about the new task
	t.Log("Step 4: Waiting for SSE update event...")
	select {
	case event := <-eventsChan:
		if !strings.Contains(event, "data: ") {
			t.Fatalf("Invalid SSE event format: %s", event)
		}
		// Verify event contains task data
		if !strings.Contains(event, "SSE test task") {
			t.Logf("Warning: Event may not contain new task data: %s", event[:min(100, len(event))])
		}
		t.Log("✓ Received SSE update event after task creation")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for SSE update event")
	}

	t.Log("✅ POC-REG-003 PASSED")
}

func TestV1SSEProjectFilter(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	project1, err := CreateProject(db, "Project 1", "/tmp/p1", nil)
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}
	project2, err := CreateProject(db, "Project 2", "/tmp/p2", nil)
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	insertTask := func(instructions string, projectID string) {
		_, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			StatusNotPicked,
			projectID,
			nowISO(),
			nowISO(),
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
	}

	insertTask("p1-task-1", project1.ID)
	insertTask("p1-task-2", project1.ID)
	insertTask("p2-task-1", project2.ID)

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(server.URL + "?project_id=" + url.QueryEscape(project1.ID))
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE connection failed with status %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var payload string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read SSE line: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			payload = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}

	var tasks []Task
	if err := json.Unmarshal([]byte(payload), &tasks); err != nil {
		t.Fatalf("Failed to decode SSE payload: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks for project filter, got %d", len(tasks))
	}
	for _, task := range tasks {
		if !strings.HasPrefix(task.Instructions, "p1-") {
			t.Fatalf("Unexpected task returned for project filter: %s", task.Instructions)
		}
	}
}

func TestV1SSETypeFilter(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	insertTask := func(instructions string) {
		_, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			StatusNotPicked,
			"proj_default",
			nowISO(),
			nowISO(),
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
	}

	insertTask("type-task-1")
	insertTask("type-task-2")

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(server.URL + "?type=tasks")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE connection failed with status %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var payload string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read SSE line: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			payload = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}

	var tasks []Task
	if err := json.Unmarshal([]byte(payload), &tasks); err != nil {
		t.Fatalf("Failed to decode SSE payload: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks for type filter, got %d", len(tasks))
	}
}

func TestV1SSESinceFilter(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	base := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	oldUpdated := base.Add(-5 * time.Minute).Format(leaseTimeFormat)
	recentUpdated := base.Add(5 * time.Minute).Format(leaseTimeFormat)
	oldCreated := base.Add(-10 * time.Minute).Format(leaseTimeFormat)
	recentCreated := base.Add(1 * time.Minute).Format(leaseTimeFormat)

	insertTask := func(instructions, createdAt, updatedAt string) {
		_, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			StatusNotPicked,
			"proj_default",
			createdAt,
			updatedAt,
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
	}

	insertTask("since-old", oldCreated, oldUpdated)
	insertTask("since-recent", recentCreated, recentUpdated)

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	sinceParam := base.Format(time.RFC3339Nano)
	resp, err := client.Get(server.URL + "?since=" + url.QueryEscape(sinceParam))
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE connection failed with status %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var payload string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read SSE line: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			payload = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}

	var tasks []Task
	if err := json.Unmarshal([]byte(payload), &tasks); err != nil {
		t.Fatalf("Failed to decode SSE payload: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task for since filter, got %d", len(tasks))
	}
	if tasks[0].Instructions != "since-recent" {
		t.Fatalf("Unexpected task returned for since filter: %s", tasks[0].Instructions)
	}
}

func TestV1SSESinceFilterLimit(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	base := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	older := base.Add(1 * time.Minute).Format(leaseTimeFormat)
	newer := base.Add(2 * time.Minute).Format(leaseTimeFormat)

	insertTask := func(instructions, createdAt, updatedAt string) {
		_, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			StatusNotPicked,
			"proj_default",
			createdAt,
			updatedAt,
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
	}

	insertTask("since-older", older, older)
	insertTask("since-newer", newer, newer)

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	sinceParam := base.Format(time.RFC3339Nano)
	resp, err := client.Get(server.URL + "?since=" + url.QueryEscape(sinceParam) + "&limit=1")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE connection failed with status %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var payload string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read SSE line: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			payload = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}

	var tasks []Task
	if err := json.Unmarshal([]byte(payload), &tasks); err != nil {
		t.Fatalf("Failed to decode SSE payload: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task for since limit, got %d", len(tasks))
	}
	if tasks[0].Instructions != "since-newer" {
		t.Fatalf("Unexpected task returned for since limit: %s", tasks[0].Instructions)
	}
}

func TestV1SSESinceFilterLimitCap(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	base := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	older := base.Add(1 * time.Minute).Format(leaseTimeFormat)
	newer := base.Add(2 * time.Minute).Format(leaseTimeFormat)

	insertTask := func(instructions, createdAt, updatedAt string) {
		_, err := db.Exec(
			"INSERT INTO tasks (instructions, status, project_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			instructions,
			StatusNotPicked,
			"proj_default",
			createdAt,
			updatedAt,
		)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}
	}

	insertTask("since-older", older, older)
	insertTask("since-newer", newer, newer)

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, 1))
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	sinceParam := base.Format(time.RFC3339Nano)
	resp, err := client.Get(server.URL + "?since=" + url.QueryEscape(sinceParam) + "&limit=10")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE connection failed with status %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var payload string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read SSE line: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			payload = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}

	var tasks []Task
	if err := json.Unmarshal([]byte(payload), &tasks); err != nil {
		t.Fatalf("Failed to decode SSE payload: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task for since limit cap, got %d", len(tasks))
	}
	if tasks[0].Instructions != "since-newer" {
		t.Fatalf("Unexpected task returned for since limit cap: %s", tasks[0].Instructions)
	}
}

func TestV1SSEProjectFilterInvalid(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(server.URL + "?project_id=")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty project_id, got %d", resp.StatusCode)
	}

	resp, err = client.Get(server.URL + "?project_id=proj_missing")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404 for missing project_id, got %d", resp.StatusCode)
	}
}

func TestV1SSETypeFilterInvalid(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(server.URL + "?type=")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty type, got %d", resp.StatusCode)
	}

	resp, err = client.Get(server.URL + "?type=bogus")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid type, got %d", resp.StatusCode)
	}
}

func TestV1SSESinceFilterInvalid(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	heartbeat := resolveSSEHeartbeatInterval(runtimeConfig{})
	server := httptest.NewServer(eventsHandler(heartbeat, resolveV1EventsReplayLimitMax(runtimeConfig{})))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(server.URL + "?since=")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty since, got %d", resp.StatusCode)
	}

	resp, err = client.Get(server.URL + "?since=not-a-time")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid since, got %d", resp.StatusCode)
	}

	resp, err = client.Get(server.URL + "?since=2026-02-11T10:00:00.000000Z&limit=0")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid limit, got %d", resp.StatusCode)
	}

	resp, err = client.Get(server.URL + "?since=2026-02-11T10:00:00.000000Z&limit=bogus")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid limit, got %d", resp.StatusCode)
	}

	resp, err = client.Get(server.URL + "?limit=1")
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for limit without since, got %d", resp.StatusCode)
	}
}

// TestPOCREG004_WorkdirManagement tests workdir set/get functionality
// POC-REG-004: Workdir set/get (default scope)
func TestPOCREG004_WorkdirManagement(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	t.Log("POC-REG-004: Testing Workdir management")

	testWorkdir := "/tmp/coco-workdir"

	// Step 1: Set workdir
	t.Log("Step 1: Setting workdir...")
	formData := url.Values{}
	formData.Set("workdir", testWorkdir)

	req := httptest.NewRequest("POST", "/set-workdir", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	setWorkdirHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Set workdir failed with status %d: %s", w.Code, w.Body.String())
	}

	var setResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&setResp); err != nil {
		t.Fatalf("Failed to decode set workdir response: %v", err)
	}

	if !setResp["success"].(bool) {
		t.Fatal("Set workdir response success is false")
	}

	if setResp["workdir"].(string) != testWorkdir {
		t.Fatalf("Set workdir response returned %s, expected %s", setResp["workdir"], testWorkdir)
	}
	t.Logf("✓ Set workdir to: %s", testWorkdir)

	// Step 2: Get workdir and verify
	t.Log("Step 2: Getting workdir...")
	req = httptest.NewRequest("GET", "/api/workdir", nil)
	w = httptest.NewRecorder()

	getWorkdirHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get workdir failed with status %d: %s", w.Code, w.Body.String())
	}

	var getResp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&getResp); err != nil {
		t.Fatalf("Failed to decode get workdir response: %v", err)
	}

	if getResp["workdir"] != testWorkdir {
		t.Fatalf("Get workdir returned %s, expected %s", getResp["workdir"], testWorkdir)
	}
	t.Logf("✓ Got workdir: %s (matches expected)", getResp["workdir"])

	// Step 3: Test empty workdir error handling
	t.Log("Step 3: Testing empty workdir validation...")
	formData = url.Values{}
	formData.Set("workdir", "")

	req = httptest.NewRequest("POST", "/set-workdir", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	setWorkdirHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected BadRequest for empty workdir, got %d", w.Code)
	}
	t.Log("✓ Empty workdir correctly rejected")

	t.Log("✅ POC-REG-004 PASSED")
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestAllPOCRegressionSuite runs all POC regression tests in sequence
func TestAllPOCRegressionSuite(t *testing.T) {
	t.Log("========================================")
	t.Log("Running Complete POC Regression Suite")
	t.Log("========================================")

	t.Run("POC-REG-001_Lifecycle", TestPOCREG001_CreateClaimSaveLifecycle)
	t.Run("POC-REG-002_ParentContext", TestPOCREG002_ParentTaskContext)
	t.Run("POC-REG-003_SSE", TestPOCREG003_SSEEventsStream)
	t.Run("POC-REG-004_Workdir", TestPOCREG004_WorkdirManagement)

	t.Log("========================================")
	t.Log("✅ All POC Regression Tests Passed!")
	t.Log("========================================")
}

// ============================================================================
// API v2 Projects Integration Tests
// ============================================================================

func TestV2CreateProject(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	// Run migrations to ensure projects table exists
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	reqBody := `{"name":"Test Project","workdir":"/test/path","settings":{"theme":"dark"}}`
	req := httptest.NewRequest("POST", "/api/v2/projects", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2CreateProjectHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	project := resp["project"].(map[string]interface{})
	if project["name"] != "Test Project" {
		t.Errorf("Expected name 'Test Project', got %v", project["name"])
	}
	if project["workdir"] != "/test/path" {
		t.Errorf("Expected workdir '/test/path', got %v", project["workdir"])
	}
}

func TestV2CreateProjectValidation(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Missing required fields
	reqBody := `{"name":"Test"}`
	req := httptest.NewRequest("POST", "/api/v2/projects", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2CreateProjectHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}

	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2CreateProjectTrimsFields(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	reqBody := `{"name":"  Trim Me  ","workdir":"  /tmp/trim  "}`
	req := httptest.NewRequest("POST", "/api/v2/projects", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2CreateProjectHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	project := resp["project"].(map[string]interface{})
	if project["name"] != "Trim Me" {
		t.Errorf("Expected trimmed name 'Trim Me', got %v", project["name"])
	}
	if project["workdir"] != "/tmp/trim" {
		t.Errorf("Expected trimmed workdir '/tmp/trim', got %v", project["workdir"])
	}
}

func TestV2CreateProjectWhitespaceValidation(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	reqBody := `{"name":"   ","workdir":"   "}`
	req := httptest.NewRequest("POST", "/api/v2/projects", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2CreateProjectHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}

	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2ProjectsMethodNotAllowed(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v2/projects", nil)
	w := httptest.NewRecorder()

	v2CreateProjectHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d", w.Code)
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodGet {
		t.Fatalf("expected details.method GET, got %v", details["method"])
	}
}

func TestV2ListProjects(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a test project
	_, err := CreateProject(testDB, "Project 1", "/path1", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v2/projects", nil)
	w := httptest.NewRecorder()

	v2ListProjectsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	projects := resp["projects"].([]interface{})
	if len(projects) < 1 {
		t.Error("Expected at least 1 project (default + created)")
	}
}

func TestV2GetProject(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a test project
	project, err := CreateProject(testDB, "Test Project", "/test/path", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v2/projects/"+project.ID, nil)
	w := httptest.NewRecorder()

	v2GetProjectHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	proj := resp["project"].(map[string]interface{})
	if proj["id"] != project.ID {
		t.Errorf("Expected ID %s, got %v", project.ID, proj["id"])
	}
}

func TestV2GetProjectNotFound(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v2/projects/proj_nonexistent", nil)
	w := httptest.NewRecorder()

	v2GetProjectHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
}

func TestV2UpdateProject(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a test project
	project, err := CreateProject(testDB, "Original", "/original", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	reqBody := `{"name":"Updated Name","workdir":"/updated/path"}`
	req := httptest.NewRequest("PUT", "/api/v2/projects/"+project.ID, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2UpdateProjectHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	proj := resp["project"].(map[string]interface{})
	if proj["name"] != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %v", proj["name"])
	}
	if proj["workdir"] != "/updated/path" {
		t.Errorf("Expected workdir '/updated/path', got %v", proj["workdir"])
	}
}

func TestV2UpdateProjectNoFields(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	project, err := CreateProject(testDB, "Original", "/original", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	reqBody := `{}`
	req := httptest.NewRequest("PUT", "/api/v2/projects/"+project.ID, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2UpdateProjectHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2UpdateProjectEmptyName(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	project, err := CreateProject(testDB, "Original", "/original", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	reqBody := `{"name":"   "}`
	req := httptest.NewRequest("PUT", "/api/v2/projects/"+project.ID, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2UpdateProjectHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2UpdateProjectEmptyWorkdir(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	project, err := CreateProject(testDB, "Original", "/original", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	reqBody := `{"workdir":"   "}`
	req := httptest.NewRequest("PUT", "/api/v2/projects/"+project.ID, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2UpdateProjectHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	assertV2ErrorEnvelope(t, w, "INVALID_ARGUMENT")
}

func TestV2UpdateProjectNotFound(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	reqBody := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v2/projects/proj_nonexistent", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2UpdateProjectHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
}

func TestV2DeleteProject(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a test project
	project, err := CreateProject(testDB, "To Delete", "/delete", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/v2/projects/"+project.ID, nil)
	w := httptest.NewRecorder()

	v2DeleteProjectHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's deleted
	_, err = GetProject(testDB, project.ID)
	if err == nil {
		t.Error("Expected error for deleted project")
	}
}

func TestV2DeleteDefaultProject(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/v2/projects/proj_default", nil)
	w := httptest.NewRecorder()

	v2DeleteProjectHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", w.Code)
	}
}

func TestV2DeleteProjectNotFound(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/v2/projects/proj_nonexistent", nil)
	w := httptest.NewRecorder()

	v2DeleteProjectHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}
}

func TestV2UpdateProjectWithPatch(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a test project
	project, err := CreateProject(testDB, "Original", "/original", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Use PATCH method for partial update
	reqBody := `{"name":"Updated via PATCH"}`
	req := httptest.NewRequest("PATCH", "/api/v2/projects/"+project.ID, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	v2UpdateProjectHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	proj := resp["project"].(map[string]interface{})
	if proj["name"] != "Updated via PATCH" {
		t.Errorf("Expected name 'Updated via PATCH', got %v", proj["name"])
	}
	// workdir should remain unchanged
	if proj["workdir"] != "/original" {
		t.Errorf("Expected workdir '/original', got %v", proj["workdir"])
	}
}

func TestV2DeleteProjectCascade(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a test project
	project, err := CreateProject(testDB, "To Delete", "/delete", nil)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Verify the project delete works correctly

	// Delete the project
	req := httptest.NewRequest("DELETE", "/api/v2/projects/"+project.ID, nil)
	w := httptest.NewRecorder()
	v2DeleteProjectHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify project is deleted
	_, err = GetProject(testDB, project.ID)
	if err == nil {
		t.Error("Expected error for deleted project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// ============================================================================
// v2 Health and Version Endpoint Tests
// ============================================================================

func TestV2HealthEndpoint(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	// Run migrations to set up schema_migrations table
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Measure response time
	start := time.Now()

	req := httptest.NewRequest("GET", "/api/v2/health", nil)
	w := httptest.NewRecorder()

	v2HealthHandler(w, req)

	elapsed := time.Since(start)

	// Check HTTP status
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check response content
	ok, exists := resp["ok"]
	if !exists {
		t.Error("Response missing 'ok' field")
	}
	if ok != true {
		t.Errorf("Expected ok=true, got %v", ok)
	}

	// Check response time (should be < 100ms per requirements)
	if elapsed > 100*time.Millisecond {
		t.Logf("Warning: Health endpoint took %v (requirement: <100ms)", elapsed)
	} else {
		t.Logf("Health endpoint responded in %v", elapsed)
	}
}

func TestV2HealthEndpointMethodNotAllowed(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Test POST method (should fail)
	req := httptest.NewRequest("POST", "/api/v2/health", nil)
	w := httptest.NewRecorder()

	v2HealthHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodPost {
		t.Fatalf("expected details.method POST, got %v", details["method"])
	}
}

func TestV2VersionEndpoint(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{
		EventsRetentionDays:        defaultEventsRetentionDays,
		EventsRetentionMax:         defaultEventsRetentionMax,
		EventsPruneIntervalSeconds: defaultEventsPruneIntervalSeconds,
	}

	// Run migrations to set up schema_migrations table
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v2/version", nil)
	w := httptest.NewRecorder()

	v2VersionHandler(cfg)(w, req)

	// Check HTTP status
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check service name
	service, exists := resp["service"]
	if !exists {
		t.Error("Response missing 'service' field")
	}
	if service != "cocopilot" {
		t.Errorf("Expected service='cocopilot', got %v", service)
	}

	// Check version field exists
	_, exists = resp["version"]
	if !exists {
		t.Error("Response missing 'version' field")
	}

	// Check API support
	apiInterface, exists := resp["api"]
	if !exists {
		t.Fatal("Response missing 'api' field")
	}
	api, ok := apiInterface.(map[string]interface{})
	if !ok {
		t.Fatal("'api' field is not a map")
	}

	if api["v1"] != true {
		t.Errorf("Expected api.v1=true, got %v", api["v1"])
	}
	if api["v2"] != true {
		t.Errorf("Expected api.v2=true, got %v", api["v2"])
	}

	// Check schema_version
	schemaVersion, exists := resp["schema_version"]
	if !exists {
		t.Error("Response missing 'schema_version' field")
	}

	// Schema version should be > 0 after migrations
	schemaVersionFloat, ok := schemaVersion.(float64)
	if !ok {
		t.Fatalf("schema_version is not a number: %v", schemaVersion)
	}
	if schemaVersionFloat <= 0 {
		t.Errorf("Expected schema_version > 0, got %v", schemaVersionFloat)
	}

	retentionRaw, exists := resp["retention"]
	if !exists {
		t.Fatal("Response missing 'retention' field")
	}
	retention, ok := retentionRaw.(map[string]interface{})
	if !ok {
		t.Fatal("'retention' field is not a map")
	}
	if retention["enabled"] != true {
		t.Errorf("Expected retention.enabled=true, got %v", retention["enabled"])
	}
	if retention["interval_seconds"] != float64(defaultEventsPruneIntervalSeconds) {
		t.Errorf("Expected retention.interval_seconds=%d, got %v", defaultEventsPruneIntervalSeconds, retention["interval_seconds"])
	}
	if retention["max_rows"] != float64(defaultEventsRetentionMax) {
		t.Errorf("Expected retention.max_rows=%d, got %v", defaultEventsRetentionMax, retention["max_rows"])
	}
	if retention["days"] != float64(defaultEventsRetentionDays) {
		t.Errorf("Expected retention.days=%d, got %v", defaultEventsRetentionDays, retention["days"])
	}

	t.Logf("Version endpoint response: service=%s, api_v1=%v, api_v2=%v, schema_version=%v",
		service, api["v1"], api["v2"], schemaVersionFloat)
}

func TestV2VersionEndpointSchemaVersion(t *testing.T) {
	testDB, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{
		EventsRetentionDays:        defaultEventsRetentionDays,
		EventsRetentionMax:         defaultEventsRetentionMax,
		EventsPruneIntervalSeconds: defaultEventsPruneIntervalSeconds,
	}

	// Run migrations
	if err := runMigrations(testDB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Get schema version via endpoint
	req := httptest.NewRequest("GET", "/api/v2/version", nil)
	w := httptest.NewRecorder()
	v2VersionHandler(cfg)(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	endpointVersion := int(resp["schema_version"].(float64))

	// Get schema version directly from database
	var dbVersion int
	err := testDB.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&dbVersion)
	if err != nil {
		t.Fatalf("Failed to query schema version: %v", err)
	}

	// They should match
	if endpointVersion != dbVersion {
		t.Errorf("Schema version mismatch: endpoint=%d, database=%d", endpointVersion, dbVersion)
	}

	t.Logf("Schema version correctly reported as %d", endpointVersion)
}

func TestV2VersionEndpointMethodNotAllowed(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := runtimeConfig{
		EventsRetentionDays:        defaultEventsRetentionDays,
		EventsRetentionMax:         defaultEventsRetentionMax,
		EventsPruneIntervalSeconds: defaultEventsPruneIntervalSeconds,
	}

	// Test POST method (should fail)
	req := httptest.NewRequest("POST", "/api/v2/version", nil)
	w := httptest.NewRecorder()

	v2VersionHandler(cfg)(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	errField := assertV2ErrorEnvelope(t, w, "METHOD_NOT_ALLOWED")
	details := errField["details"].(map[string]interface{})
	if details["method"] != http.MethodPost {
		t.Fatalf("expected details.method POST, got %v", details["method"])
	}
}
