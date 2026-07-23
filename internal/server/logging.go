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

var (
	activeLogger atomic.Pointer[slog.Logger]
	debugOn      atomic.Bool
)

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
	debugOn.Store(logger.Enabled(context.Background(), slog.LevelDebug))
}

// log returns the active logger, falling back to the slog default before the
// package finishes initializing.
func log() *slog.Logger {
	if logger := activeLogger.Load(); logger != nil {
		return logger
	}

	return slog.Default()
}

// debugEnabled reports whether debug records are worth building. It is checked
// on per-message hot paths to avoid formatting work that would be discarded.
func debugEnabled() bool {
	return debugOn.Load()
}
