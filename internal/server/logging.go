// Package server centralizes structured logging so the HTTP, hub, and client
// layers share a single configurable [log/slog] logger.
package server

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
)

var activeLogger atomic.Pointer[slog.Logger]

func init() {
	SetLogger(NewLogger(LogLevelFromEnv()))
}

// NewLogger builds the structured text logger used by the server. Output is
// written to stderr so stdout stays free for application data.
func NewLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

// LogLevelFromEnv reads LOG_LEVEL (debug, info, warn, error) and falls back to
// info when the variable is unset or unparsable.
func LogLevelFromEnv() slog.Level {
	raw := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if raw == "" {
		return slog.LevelInfo
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return slog.LevelInfo
	}

	return level
}

// SetLogger installs the logger used across the package. Passing nil restores
// the default info-level logger.
func SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = NewLogger(slog.LevelInfo)
	}

	activeLogger.Store(logger)
}

// log returns the active logger, falling back to the slog default before the
// package finishes initializing.
func log() *slog.Logger {
	if logger := activeLogger.Load(); logger != nil {
		return logger
	}

	return slog.Default()
}

// debugEnabled reports whether debug records are worth building. Per-message
// hot paths check it before doing work that only exists to be logged — such as
// converting a payload to a string — which slog itself cannot avoid, because
// arguments are evaluated before it sees them.
//
// It asks the handler every time rather than caching the answer, so a level
// that changes at runtime (a [slog.LevelVar]) takes effect immediately.
func debugEnabled() bool {
	return log().Enabled(context.Background(), slog.LevelDebug)
}
