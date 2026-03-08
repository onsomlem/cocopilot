// Package config provides runtime configuration loading and validation.
package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultSSEHeartbeatSeconds        = 30
	MinSSEHeartbeatSeconds            = 5
	MaxSSEHeartbeatSeconds            = 300
	DefaultDBPath                     = "./tasks.db"
	DefaultHTTPAddr                   = "127.0.0.1:8080"
	DefaultEventsRetentionDays        = 30
	DefaultEventsRetentionMax         = 0
	DefaultEventsPruneIntervalSeconds = 3600
	MinEventsPruneIntervalSeconds     = 60
	MaxEventsPruneIntervalSeconds     = 86400
	V1EventTypeTasks                  = "tasks"

	V1TasksListDefaultLimit       = 100
	V1TasksListMaxLimit           = 500
	V1EventsReplayLimitMaxDefault = 500
	DefaultProjectID              = "proj_default"

	V2EventsListDefaultLimit = 100
	V2EventsListMaxLimit     = 500
)

// RuntimeConfig holds all server configuration parsed from env vars.
type RuntimeConfig struct {
	SSEHeartbeatSeconds        int
	SSEReplayLimitMax          int
	V1EventsReplayLimitMax     int
	DBPath                     string
	HTTPAddr                   string
	RequireAPIKey              bool
	RequireAPIKeyReads         bool
	APIKey                     string
	AuthIdentities             []AuthIdentity
	EventsRetentionDays        int
	EventsRetentionMax         int
	EventsPruneIntervalSeconds int
	AutomationRules            []AutomationRule
	MaxAutomationDepth         int
	AutomationRateLimit        int           // per hour, default 100
	AutomationBurstLimit       int           // per minute, default 10
	AutomationCircuitMaxFail   int           // consecutive failures before circuit opens, default 5
	AutomationCircuitCooldown  time.Duration // cooldown before half-open, default 5m
	NoBrowser                  bool          // skip auto-opening browser on start
}

// AuthIdentity represents an API identity for authentication.
type AuthIdentity struct {
	ID     string
	Type   string
	APIKey string
	Scopes map[string]struct{}
}

// AutomationRule represents a user-defined automation trigger/action pair.
type AutomationRule struct {
	Name    string             `json:"name"`
	Enabled *bool              `json:"enabled"`
	Trigger string             `json:"trigger"`
	Actions []AutomationAction `json:"actions"`
}

// AutomationAction represents an action executed by an automation rule.
type AutomationAction struct {
	Type string             `json:"type"`
	Task AutomationTaskSpec `json:"task"`
}

// AutomationTaskSpec defines the task to create when an automation fires.
type AutomationTaskSpec struct {
	Title        *string  `json:"title"`
	Instructions string   `json:"instructions"`
	Type         *string  `json:"type"`
	Priority     *int     `json:"priority"`
	Tags         []string `json:"tags"`
	Parent       *string  `json:"parent"`
}

// GetEnvConfigValue reads a string env var with a fallback default.
func GetEnvConfigValue(name, fallback string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists {
		return fallback, nil
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is set but empty", name)
	}
	return trimmed, nil
}

// GetEnvBoolValue reads a boolean env var with a fallback default.
func GetEnvBoolValue(name string, fallback bool) (bool, error) {
	value, exists := os.LookupEnv(name)
	if !exists {
		return fallback, nil
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s: expected true/false", name)
	}
}

// ParseScopeSet parses a comma-separated scope string into a set.
func ParseScopeSet(raw string) (map[string]struct{}, error) {
	scopes := make(map[string]struct{})
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope == "" {
			continue
		}
		scopes[scope] = struct{}{}
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("at least one scope is required")
	}
	return scopes, nil
}

// ParseAuthIdentities parses the COCO_API_IDENTITIES env var format.
func ParseAuthIdentities(raw string) ([]AuthIdentity, error) {
	entries := strings.Split(raw, ";")
	identities := make([]AuthIdentity, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		fields := strings.Split(entry, "|")
		if len(fields) != 4 {
			return nil, fmt.Errorf("invalid COCO_API_IDENTITIES entry %q: expected id|type|api_key|scope1,scope2", entry)
		}

		id := strings.TrimSpace(fields[0])
		identityType := strings.ToLower(strings.TrimSpace(fields[1]))
		apiKey := strings.TrimSpace(fields[2])
		scopes, err := ParseScopeSet(fields[3])
		if err != nil {
			return nil, fmt.Errorf("invalid scopes for identity %q: %w", id, err)
		}

		if id == "" {
			return nil, fmt.Errorf("identity id cannot be empty")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("api key cannot be empty for identity %q", id)
		}
		switch identityType {
		case "agent", "user", "service":
		default:
			return nil, fmt.Errorf("invalid identity type %q for identity %q", identityType, id)
		}

		identities = append(identities, AuthIdentity{
			ID:     id,
			Type:   identityType,
			APIKey: apiKey,
			Scopes: scopes,
		})
	}

	return identities, nil
}

// NormalizeV1EventType validates and normalizes a v1 event type string.
func NormalizeV1EventType(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case V1EventTypeTasks:
		return V1EventTypeTasks, true
	default:
		return "", false
	}
}

// LoadRuntimeConfig reads all COCO_* env vars and returns a RuntimeConfig.
// The parseRules callback is used to parse and validate the
// COCO_AUTOMATION_RULES JSON string (kept external to avoid circular deps
// with the automation package logic).
func LoadRuntimeConfig(parseRules func(string) ([]AutomationRule, error)) (RuntimeConfig, error) {
	dbPath, err := GetEnvConfigValue("COCO_DB_PATH", filepath.Join(".", "tasks.db"))
	if err != nil {
		return RuntimeConfig{}, err
	}

	httpAddr, err := GetEnvConfigValue("COCO_HTTP_ADDR", DefaultHTTPAddr)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if _, err := net.ResolveTCPAddr("tcp", httpAddr); err != nil {
		return RuntimeConfig{}, fmt.Errorf("invalid COCO_HTTP_ADDR: %w", err)
	}

	requireAPIKey, err := GetEnvBoolValue("COCO_REQUIRE_API_KEY", false)
	if err != nil {
		return RuntimeConfig{}, err
	}
	requireAPIKeyReads, err := GetEnvBoolValue("COCO_REQUIRE_API_KEY_READS", false)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if requireAPIKeyReads {
		requireAPIKey = true
	}

	apiKey := strings.TrimSpace(os.Getenv("COCO_API_KEY"))
	identitiesRaw := strings.TrimSpace(os.Getenv("COCO_API_IDENTITIES"))
	authIdentities := make([]AuthIdentity, 0)
	if identitiesRaw != "" {
		parsed, err := ParseAuthIdentities(identitiesRaw)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_API_IDENTITIES: %w", err)
		}
		authIdentities = append(authIdentities, parsed...)
	}

	if apiKey != "" {
		// Backward-compatible single-key mode gets full scope access.
		authIdentities = append(authIdentities, AuthIdentity{
			ID:     "legacy_default",
			Type:   "service",
			APIKey: apiKey,
			Scopes: map[string]struct{}{"*": {}},
		})
	}

	if requireAPIKey && len(authIdentities) == 0 {
		return RuntimeConfig{}, fmt.Errorf("COCO_API_KEY or COCO_API_IDENTITIES is required when COCO_REQUIRE_API_KEY=true")
	}

	sseHeartbeatSeconds := DefaultSSEHeartbeatSeconds
	if raw, exists := os.LookupEnv("COCO_SSE_HEARTBEAT_SECONDS"); exists {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return RuntimeConfig{}, fmt.Errorf("COCO_SSE_HEARTBEAT_SECONDS is set but empty")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_SSE_HEARTBEAT_SECONDS: expected integer")
		}
		if parsed < MinSSEHeartbeatSeconds || parsed > MaxSSEHeartbeatSeconds {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_SSE_HEARTBEAT_SECONDS: must be between %d and %d", MinSSEHeartbeatSeconds, MaxSSEHeartbeatSeconds)
		}
		sseHeartbeatSeconds = parsed
	}

	sseReplayLimitMax := V2EventsListMaxLimit
	if raw, exists := os.LookupEnv("COCO_SSE_REPLAY_LIMIT_MAX"); exists {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return RuntimeConfig{}, fmt.Errorf("COCO_SSE_REPLAY_LIMIT_MAX is set but empty")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_SSE_REPLAY_LIMIT_MAX: expected integer")
		}
		if parsed < 1 || parsed > V2EventsListMaxLimit {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_SSE_REPLAY_LIMIT_MAX: must be between %d and %d", 1, V2EventsListMaxLimit)
		}
		sseReplayLimitMax = parsed
	}

	v1EventsReplayLimitMax := V1EventsReplayLimitMaxDefault
	if raw, exists := os.LookupEnv("COCO_V1_EVENTS_REPLAY_LIMIT_MAX"); exists {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return RuntimeConfig{}, fmt.Errorf("COCO_V1_EVENTS_REPLAY_LIMIT_MAX is set but empty")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_V1_EVENTS_REPLAY_LIMIT_MAX: expected integer")
		}
		if parsed < 1 || parsed > V1EventsReplayLimitMaxDefault {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_V1_EVENTS_REPLAY_LIMIT_MAX: must be between %d and %d", 1, V1EventsReplayLimitMaxDefault)
		}
		v1EventsReplayLimitMax = parsed
	}

	eventsRetentionDays := DefaultEventsRetentionDays
	if raw, exists := os.LookupEnv("COCO_EVENTS_RETENTION_DAYS"); exists {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return RuntimeConfig{}, fmt.Errorf("COCO_EVENTS_RETENTION_DAYS is set but empty")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil || parsed < 0 {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_EVENTS_RETENTION_DAYS: expected non-negative integer")
		}
		eventsRetentionDays = parsed
	}

	eventsRetentionMax := DefaultEventsRetentionMax
	if raw, exists := os.LookupEnv("COCO_EVENTS_RETENTION_MAX_ROWS"); exists {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return RuntimeConfig{}, fmt.Errorf("COCO_EVENTS_RETENTION_MAX_ROWS is set but empty")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil || parsed < 0 {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_EVENTS_RETENTION_MAX_ROWS: expected non-negative integer")
		}
		eventsRetentionMax = parsed
	}

	eventsPruneIntervalSeconds := DefaultEventsPruneIntervalSeconds
	if raw, exists := os.LookupEnv("COCO_EVENTS_PRUNE_INTERVAL_SECONDS"); exists {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return RuntimeConfig{}, fmt.Errorf("COCO_EVENTS_PRUNE_INTERVAL_SECONDS is set but empty")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_EVENTS_PRUNE_INTERVAL_SECONDS: expected integer")
		}
		if parsed < MinEventsPruneIntervalSeconds || parsed > MaxEventsPruneIntervalSeconds {
			return RuntimeConfig{}, fmt.Errorf("invalid COCO_EVENTS_PRUNE_INTERVAL_SECONDS: must be between %d and %d", MinEventsPruneIntervalSeconds, MaxEventsPruneIntervalSeconds)
		}
		eventsPruneIntervalSeconds = parsed
	}

	var automationRules []AutomationRule
	if parseRules != nil {
		automationRules, err = parseRules(strings.TrimSpace(os.Getenv("COCO_AUTOMATION_RULES")))
		if err != nil {
			return RuntimeConfig{}, err
		}
	}

	maxAutomationDepth := 5
	if raw := os.Getenv("COCO_MAX_AUTOMATION_DEPTH"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			maxAutomationDepth = parsed
		}
	}

	automationRateLimit := 100
	if raw := os.Getenv("COCO_AUTOMATION_RATE_LIMIT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			automationRateLimit = parsed
		}
	}

	automationBurstLimit := 10
	if raw := os.Getenv("COCO_AUTOMATION_BURST_LIMIT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			automationBurstLimit = parsed
		}
	}

	automationCircuitMaxFail := 5
	if raw := os.Getenv("COCO_AUTOMATION_CIRCUIT_MAX_FAILURES"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			automationCircuitMaxFail = parsed
		}
	}

	automationCircuitCooldown := 5 * time.Minute
	if raw := os.Getenv("COCO_AUTOMATION_CIRCUIT_COOLDOWN"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			automationCircuitCooldown = parsed
		}
	}

	noBrowser, _ := GetEnvBoolValue("COCO_NO_BROWSER", false)
	// Also check CLI flags
	for _, arg := range os.Args[1:] {
		if arg == "--no-browser" {
			noBrowser = true
		}
	}

	return RuntimeConfig{
		DBPath:                     dbPath,
		HTTPAddr:                   httpAddr,
		RequireAPIKey:              requireAPIKey,
		RequireAPIKeyReads:         requireAPIKeyReads,
		APIKey:                     apiKey,
		AuthIdentities:             authIdentities,
		SSEHeartbeatSeconds:        sseHeartbeatSeconds,
		SSEReplayLimitMax:          sseReplayLimitMax,
		V1EventsReplayLimitMax:     v1EventsReplayLimitMax,
		EventsRetentionDays:        eventsRetentionDays,
		EventsRetentionMax:         eventsRetentionMax,
		EventsPruneIntervalSeconds: eventsPruneIntervalSeconds,
		AutomationRules:            automationRules,
		MaxAutomationDepth:         maxAutomationDepth,
		AutomationRateLimit:        automationRateLimit,
		AutomationBurstLimit:       automationBurstLimit,
		AutomationCircuitMaxFail:   automationCircuitMaxFail,
		AutomationCircuitCooldown:  automationCircuitCooldown,
		NoBrowser:                  noBrowser,
	}, nil
}

// ResolveSSEHeartbeatInterval returns the SSE heartbeat as a time.Duration.
func ResolveSSEHeartbeatInterval(cfg RuntimeConfig) time.Duration {
	seconds := cfg.SSEHeartbeatSeconds
	if seconds == 0 {
		seconds = DefaultSSEHeartbeatSeconds
	}
	return time.Duration(seconds) * time.Second
}

// ResolveV1EventsReplayLimitMax returns the v1 events replay limit.
func ResolveV1EventsReplayLimitMax(cfg RuntimeConfig) int {
	if cfg.V1EventsReplayLimitMax > 0 {
		return cfg.V1EventsReplayLimitMax
	}
	return V1EventsReplayLimitMaxDefault
}

// ResolveSSEReplayLimitMax returns the SSE replay limit.
func ResolveSSEReplayLimitMax(cfg RuntimeConfig) int {
	if cfg.SSEReplayLimitMax > 0 {
		return cfg.SSEReplayLimitMax
	}
	return V2EventsListMaxLimit
}

// ResolveEventsPruneInterval returns the events prune interval as a time.Duration.
func ResolveEventsPruneInterval(cfg RuntimeConfig) time.Duration {
	seconds := cfg.EventsPruneIntervalSeconds
	if seconds == 0 {
		seconds = DefaultEventsPruneIntervalSeconds
	}
	return time.Duration(seconds) * time.Second
}
