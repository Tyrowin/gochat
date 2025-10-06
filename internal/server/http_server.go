// Package server constructs and starts the GoChat HTTP service with helpers
// that apply sensible production defaults.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// CreateServer creates and configures an HTTP server with the specified port and handler.
// It sets reasonable timeout values for production use.
func CreateServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// StartHub initializes and starts the global hub in a separate goroutine.
// This should be called before starting the HTTP server.
func StartHub() {
	go hub.Run()
	log.Println("Hub started and ready to manage WebSocket connections")
}

// StartServer starts the HTTP server and begins listening for connections.
// It returns an error if the server fails to start.
func StartServer(server *http.Server) error {
	fmt.Printf("Server listening on port %s\n", server.Addr)
	return server.ListenAndServe()
}

// ShutdownServer gracefully shuts down the HTTP server without interrupting active connections.
// It waits for active connections to close or until the timeout is reached.
func ShutdownServer(server *http.Server, timeout time.Duration) error {
	log.Println("Shutting down HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
		return err
	}

	log.Println("HTTP server shutdown completed")
	return nil
}

// GetHub returns the global hub instance for shutdown coordination
func GetHub() *Hub {
	return hub
}
