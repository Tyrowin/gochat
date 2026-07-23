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
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maltemindedal/blip/internal/server"
)

// Shutdown budget: the HTTP server and the hub each get half of the total.
const (
	shutdownTimeout = 30 * time.Second
	stageTimeout    = shutdownTimeout / 2
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

	config := server.NewConfigFromEnv()
	server.SetConfig(config)
	server.StartHub()

	httpServer := server.CreateServer(config.Port, server.SetupRoutes())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.StartServer(httpServer)
	}()

	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil

	case <-ctx.Done():
		// Stop intercepting signals so a second interrupt terminates immediately
		// instead of waiting out the shutdown budget.
		stop()
		logger.Info("shutdown signal received; draining connections")

		if err := gracefulShutdown(httpServer); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}

		logger.Info("server stopped gracefully")
		return nil
	}
}

// gracefulShutdown stops accepting new connections and then drains the hub,
// giving up once the overall shutdown budget is exhausted.
func gracefulShutdown(httpServer *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	complete := make(chan error, 1)
	go func() {
		// The HTTP server must stop accepting connections before the hub drains,
		// so these run in sequence and their failures are reported together.
		httpErr := server.ShutdownServer(httpServer, stageTimeout)
		hubErr := server.GetHub().Shutdown(stageTimeout)
		complete <- errors.Join(httpErr, hubErr)
	}()

	select {
	case err := <-complete:
		return err
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout exceeded: %w", ctx.Err())
	}
}
