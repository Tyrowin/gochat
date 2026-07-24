// Package server provides configuration helpers that define runtime defaults,
// validation, and rate-limiting parameters for the Blip service.
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

	cfg.MaxMessageSize = positiveIntFromEnv("MAX_MESSAGE_SIZE", cfg.MaxMessageSize)
	cfg.RateLimit.Burst = positiveIntFromEnv("RATE_LIMIT_BURST", cfg.RateLimit.Burst)

	refillSeconds := positiveIntFromEnv("RATE_LIMIT_REFILL_INTERVAL",
		int(cfg.RateLimit.RefillInterval/time.Second))
	cfg.RateLimit.RefillInterval = time.Duration(refillSeconds) * time.Second

	return &cfg
}

func parseOrigins(origins string) []string {
	parts := strings.Split(origins, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// positiveIntFromEnv reads name and returns it as an integer greater than zero.
// An unset variable, a value that does not parse, and a value that is zero or
// negative all yield fallback; the latter two are logged first.
func positiveIntFromEnv[T int | int64](name string, fallback T) T {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		log().Warn("invalid "+name+"; using default", "value", raw, "default", fallback)
		return fallback
	}

	return T(parsed)
}
