package server

import (
	"reflect"
	"testing"
	"time"
)

// TestResolveConfigPreservesEveryField pins that resolution is lossless for a
// Config that needs nothing substituted.
//
// Two halves, because the comparison alone is not enough. assertEveryFieldIsSet
// fails if the literal below leaves any field of [Config] at its zero value, so
// a field added to the struct stops this test until someone gives it a value
// here. Only then does the [reflect.DeepEqual] comparison mean anything: with
// every field set to something non-zero, a field [resolveConfig] drops comes
// back zero and the comparison catches it. Without the first half the second is
// vacuous — an unset field is zero on both sides and compares equal, which is
// the same enumerate-every-field trap this test exists to guard against, merely
// moved from resolveConfig into the test.
//
// Neither half catches aliasing. DeepEqual compares what a reference field
// points at, not whether both sides point at the same thing, so a field that
// arrives sharing the caller's backing array rather than copied passes here.
// That hazard is guarded by the comment in resolveConfig.
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

	assertEveryFieldIsSet(t, reflect.ValueOf(cfg), "Config")

	resolved := resolveConfig(&cfg)

	if !reflect.DeepEqual(resolved.Config, cfg) {
		t.Errorf("resolveConfig(%+v) resolved to %+v, want the input carried across unchanged",
			cfg, resolved.Config)
	}
}

// assertEveryFieldIsSet fails for each field of v left at its zero value. It
// recurses into nested structs declared in this package — RateLimitConfig
// today — so a field added one level down is caught too, and stops at types
// from elsewhere rather than picking over their internals.
func assertEveryFieldIsSet(t *testing.T, v reflect.Value, path string) {
	t.Helper()

	structType := v.Type()

	for i := range structType.NumField() {
		field := structType.Field(i)
		value := v.Field(i)
		name := path + "." + field.Name

		if value.Kind() == reflect.Struct && field.Type.PkgPath() == structType.PkgPath() {
			assertEveryFieldIsSet(t, value, name)
			continue
		}

		if value.IsZero() {
			t.Errorf("%s is left at its zero value by this test; give it a non-zero value so "+
				"the comparison can tell a field resolveConfig dropped from one never set", name)
		}
	}
}
