// Package server provides configuration helpers that define runtime defaults,
// validation, and rate-limiting parameters for the GoChat service.
package server

import (
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
