// Package server provides configuration helpers that define runtime defaults,
// validation, and rate-limiting parameters for the GoChat service.
package server

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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

var (
	configMu        sync.RWMutex
	activeConfig    Config
	allowedOrigins  map[string]struct{}
	allowAllOrigins bool
)

func init() {
	SetConfig(nil)
}

func defaultConfig() Config {
	return Config{
		Port: ":8080",
		AllowedOrigins: []string{
			"http://localhost:8080",
		},
		MaxMessageSize: 512,
		RateLimit: RateLimitConfig{
			Burst:          5,
			RefillInterval: time.Second,
		},
	}
}

func sanitizeConfig(cfg Config) Config {
	if cfg.Port == "" {
		cfg.Port = ":8080"
	}

	if cfg.MaxMessageSize <= 0 {
		cfg.MaxMessageSize = 512
	}

	if cfg.RateLimit.Burst <= 0 {
		cfg.RateLimit.Burst = 5
	}

	if cfg.RateLimit.RefillInterval <= 0 {
		cfg.RateLimit.RefillInterval = time.Second
	}

	normalizedOrigins, allowAll := normalizeOrigins(cfg.AllowedOrigins)
	cfg.AllowedOrigins = normalizedOrigins

	configMu.Lock()
	defer configMu.Unlock()

	activeConfig = cfg
	allowAllOrigins = allowAll
	allowedOrigins = make(map[string]struct{}, len(normalizedOrigins))
	for _, origin := range normalizedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	return cfg
}

// SetConfig applies the provided configuration. Passing nil resets to defaults.
func SetConfig(cfg *Config) {
	if cfg == nil {
		defaultCfg := defaultConfig()
		sanitizeConfig(defaultCfg)
		return
	}

	sanitized := Config{
		Port:           cfg.Port,
		AllowedOrigins: append([]string(nil), cfg.AllowedOrigins...),
		MaxMessageSize: cfg.MaxMessageSize,
		RateLimit: RateLimitConfig{
			Burst:          cfg.RateLimit.Burst,
			RefillInterval: cfg.RateLimit.RefillInterval,
		},
	}
	sanitizeConfig(sanitized)
}

func currentConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()

	cfg := activeConfig
	cfg.AllowedOrigins = append([]string(nil), cfg.AllowedOrigins...)
	return cfg
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

	// Load SERVER_PORT
	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Port = port
	}

	// Load ALLOWED_ORIGINS
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = parseOrigins(origins)
	}

	// Load MAX_MESSAGE_SIZE
	if maxSize := os.Getenv("MAX_MESSAGE_SIZE"); maxSize != "" {
		cfg.MaxMessageSize = parseMaxMessageSize(maxSize, cfg.MaxMessageSize)
	}

	// Load RATE_LIMIT_BURST
	if burst := os.Getenv("RATE_LIMIT_BURST"); burst != "" {
		cfg.RateLimit.Burst = parseIntValue(burst, cfg.RateLimit.Burst)
	}

	// Load RATE_LIMIT_REFILL_INTERVAL
	if interval := os.Getenv("RATE_LIMIT_REFILL_INTERVAL"); interval != "" {
		cfg.RateLimit.RefillInterval = parseRefillInterval(interval, cfg.RateLimit.RefillInterval)
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

func parseMaxMessageSize(value string, defaultValue int64) int64 {
	if size, err := strconv.ParseInt(value, 10, 64); err == nil && size > 0 {
		return size
	}
	return defaultValue
}

func parseIntValue(value string, defaultValue int) int {
	if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
		return parsed
	}
	return defaultValue
}

func parseRefillInterval(value string, defaultValue time.Duration) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultValue
}
