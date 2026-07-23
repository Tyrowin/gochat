// Package server provides configuration helpers that define runtime defaults,
// validation, and rate-limiting parameters for the GoChat service.
package server

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Default configuration values applied when a setting is unset or invalid.
const (
	defaultPort            = ":8080"
	defaultMaxMessageSize  = 512
	defaultRateLimitBurst  = 5
	defaultRateLimitRefill = time.Second
	defaultAllowedOrigin   = "http://localhost:8080"
)

// RateLimitConfig defines the parameters for per-connection message rate limiting.
type RateLimitConfig struct {
	Burst          int
	RefillInterval time.Duration
}

// Config holds the server configuration settings including security controls.
type Config struct {
	Port           string
	AllowedOrigins []string
	MaxMessageSize int64
	RateLimit      RateLimitConfig
}

// configSnapshot is an immutable, fully resolved view of the configuration.
// It is published atomically so hot paths (origin checks, client construction)
// can read it without locking.
type configSnapshot struct {
	cfg      Config
	origins  map[string]struct{}
	allowAll bool
}

var activeConfig atomic.Pointer[configSnapshot]

func init() {
	SetConfig(nil)
}

func defaultConfig() Config {
	return Config{
		Port:           defaultPort,
		AllowedOrigins: []string{defaultAllowedOrigin},
		MaxMessageSize: defaultMaxMessageSize,
		RateLimit: RateLimitConfig{
			Burst:          defaultRateLimitBurst,
			RefillInterval: defaultRateLimitRefill,
		},
	}
}

// newConfigSnapshot substitutes defaults for invalid values and resolves the
// origin allow-list into a lookup set, returning the snapshot to publish.
func newConfigSnapshot(cfg Config) *configSnapshot {
	switch {
	case cfg.Port == "":
		cfg.Port = defaultPort
	case !strings.Contains(cfg.Port, ":"):
		cfg.Port = ":" + cfg.Port
	}

	if cfg.MaxMessageSize <= 0 {
		cfg.MaxMessageSize = defaultMaxMessageSize
	}

	if cfg.RateLimit.Burst <= 0 {
		cfg.RateLimit.Burst = defaultRateLimitBurst
	}

	if cfg.RateLimit.RefillInterval <= 0 {
		cfg.RateLimit.RefillInterval = defaultRateLimitRefill
	}

	normalizedOrigins, allowAll := normalizeOrigins(cfg.AllowedOrigins)
	cfg.AllowedOrigins = normalizedOrigins

	origins := make(map[string]struct{}, len(normalizedOrigins))
	for _, origin := range normalizedOrigins {
		origins[origin] = struct{}{}
	}

	return &configSnapshot{cfg: cfg, origins: origins, allowAll: allowAll}
}

// SetConfig applies the provided configuration. Passing nil resets to defaults.
func SetConfig(cfg *Config) {
	if cfg == nil {
		activeConfig.Store(newConfigSnapshot(defaultConfig()))
		return
	}

	activeConfig.Store(newConfigSnapshot(Config{
		Port:           cfg.Port,
		AllowedOrigins: slices.Clone(cfg.AllowedOrigins),
		MaxMessageSize: cfg.MaxMessageSize,
		RateLimit:      cfg.RateLimit,
	}))
}

// currentSnapshot returns the active resolved configuration without copying.
// Callers must treat the result as read-only.
func currentSnapshot() *configSnapshot {
	if snap := activeConfig.Load(); snap != nil {
		return snap
	}

	return newConfigSnapshot(defaultConfig())
}

// NewConfig creates a Config instance populated with default values for all settings.
func NewConfig() *Config {
	cfg := defaultConfig()
	return &cfg
}

// NewConfigFromEnv creates a Config instance from environment variables.
// Falls back to default values if environment variables are not set.
func NewConfigFromEnv() *Config {
	cfg := defaultConfig()

	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Port = port
	}

	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = parseOrigins(origins)
	}

	if maxSize := os.Getenv("MAX_MESSAGE_SIZE"); maxSize != "" {
		if parsedSize, ok := parseMaxMessageSize(maxSize); ok {
			cfg.MaxMessageSize = parsedSize
		} else {
			log().Warn("invalid MAX_MESSAGE_SIZE; using default",
				"value", maxSize, "default", cfg.MaxMessageSize)
		}
	}

	if burst := os.Getenv("RATE_LIMIT_BURST"); burst != "" {
		if parsedBurst, ok := parseIntValue(burst); ok {
			cfg.RateLimit.Burst = parsedBurst
		} else {
			log().Warn("invalid RATE_LIMIT_BURST; using default",
				"value", burst, "default", cfg.RateLimit.Burst)
		}
	}

	if interval := os.Getenv("RATE_LIMIT_REFILL_INTERVAL"); interval != "" {
		if parsedInterval, ok := parseRefillInterval(interval); ok {
			cfg.RateLimit.RefillInterval = parsedInterval
		} else {
			log().Warn("invalid RATE_LIMIT_REFILL_INTERVAL; using default",
				"value", interval, "default", cfg.RateLimit.RefillInterval)
		}
	}

	return &cfg
}

func parseOrigins(origins string) []string {
	parts := strings.Split(origins, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseMaxMessageSize(value string) (int64, bool) {
	size, err := strconv.ParseInt(value, 10, 64)
	if err != nil || size <= 0 {
		return 0, false
	}

	return size, true
}

func parseIntValue(value string) (int, bool) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, false
	}

	return parsed, true
}

func parseRefillInterval(value string) (time.Duration, bool) {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0, false
	}

	return time.Duration(seconds) * time.Second, true
}
