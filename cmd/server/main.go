/*
Blip is a real-time WebSocket-based chat server.

The server provides WebSocket endpoints for real-time communication
and includes a built-in test page for development and testing.

Usage:

	blip

The server will start on port 8080 by default and provide the following endpoints:

  - / - Health check endpoint
  - /ws - WebSocket endpoint for chat connections
  - /test - HTML test page for WebSocket functionality
*/
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/maltemindedal/blip/internal/server"
)

// Build information, injected at link time with -ldflags. See the build
// targets in the Makefile and Dockerfile.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := server.NewLogger(server.LogLevelFromEnv())
	slog.SetDefault(logger)
	server.SetLogger(logger)

	logger.Info("starting Blip server",
		"version", Version, "commit", Commit, "build_time", BuildTime)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Stop intercepting signals as soon as the first one arrives, so a second
	// interrupt terminates immediately instead of waiting out the shutdown
	// budget.
	go func() {
		<-ctx.Done()
		stop()
	}()

	return server.New(server.NewConfigFromEnv()).Run(ctx)
}
