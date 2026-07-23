// Package server constructs and starts the GoChat HTTP service with helpers
// that apply sensible production defaults.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	globalHubMu sync.Mutex
	hub         = NewHub()
)

func ensureGlobalHub() *Hub {
	globalHubMu.Lock()
	defer globalHubMu.Unlock()

	if hub == nil || hub.IsStopped() {
		hub = NewHub()
	}

	return hub
}

// CreateServer creates and configures an HTTP server with the specified port and handler.
// It sets reasonable timeout values for production use.
func CreateServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              port,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
}

// StartHub initializes and starts the global hub in a separate goroutine.
// This should be called before starting the HTTP server.
func StartHub() {
	ensureGlobalHub().Start()
	log().Info("hub started and ready to manage WebSocket connections")
}

// StartServer starts the HTTP server and begins listening for connections.
// It blocks until the server stops and returns an error if it fails to start.
func StartServer(server *http.Server) error {
	if server == nil {
		return errors.New("http server is nil")
	}

	log().Info("server listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

// ShutdownServer gracefully shuts down the HTTP server without interrupting active connections.
// It waits for active connections to close or until the timeout is reached.
func ShutdownServer(server *http.Server, timeout time.Duration) error {
	if server == nil {
		return errors.New("http server is nil")
	}

	log().Info("shutting down HTTP server")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	log().Info("HTTP server shutdown completed")
	return nil
}

// GetHub returns the global hub instance for shutdown coordination.
func GetHub() *Hub {
	return ensureGlobalHub()
}
