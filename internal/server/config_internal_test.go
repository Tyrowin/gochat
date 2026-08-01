package server

import (
	"reflect"
	"testing"
	"time"
)

// TestResolveConfigPreservesEveryField pins that resolution is lossless for a
// Config that needs nothing substituted. It compares the whole struct instead of
// naming fields on purpose: enumerating them here would repeat the mistake it is
// meant to catch, so a field added to [Config] and dropped by [resolveConfig]
// fails this test without anyone remembering to extend it.
//
// It catches dropped fields only. [reflect.DeepEqual] compares what a reference
// field points at, not whether both sides point at the same thing, so a field
// that arrives aliased to the caller's rather than copied passes here. That
// hazard is guarded by the comment in resolveConfig, not by this test.
func TestResolveConfigPreservesEveryField(t *testing.T) {
	t.Parallel()

	// Every value is non-default and already normalized, so nothing in the
	// result can differ because it was resolved rather than dropped.
	cfg := Config{
		Port:           ":9443",
		AllowedOrigins: []string{"https://example.com"},
		MaxMessageSize: 4096,
		RateLimit: RateLimitConfig{
			Burst:          42,
			RefillInterval: 3 * time.Second,
		},
	}

	resolved := resolveConfig(&cfg)

	if !reflect.DeepEqual(resolved.Config, cfg) {
		t.Errorf("resolveConfig(%+v) resolved to %+v, want the input carried across unchanged",
			cfg, resolved.Config)
	}
}
