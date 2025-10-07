/*
GoChat is a real-time WebSocket-based chat server.

The server provides WebSocket endpoints for real-time communication
and includes a built-in test page for development and testing.

Usage:

	gochat

The server will start on port 8080 by default and provide the following endpoints:

  - / - Health check endpoint
  - /ws - WebSocket endpoint for chat connections
  - /test - HTML test page for WebSocket functionality
*/
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
)

func main() {
	fmt.Println("Starting GoChat server...")

	config := server.NewConfigFromEnv()
	server.SetConfig(config)
	server.StartHub()
	mux := server.SetupRoutes()
	httpServer := server.CreateServer(config.Port, mux)

	// Channel to listen for errors coming from the HTTP server
	serverErrors := make(chan error, 1)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", config.Port)
		serverErrors <- server.StartServer(httpServer)
	}()

	// Channel to listen for OS interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Received shutdown signal: %v", sig)

		// Initiate graceful shutdown
		if err := gracefulShutdown(httpServer); err != nil {
			log.Fatalf("Graceful shutdown failed: %v", err)
		}

		log.Println("Server stopped gracefully")
	}
}

// gracefulShutdown performs orderly shutdown of the server components
func gracefulShutdown(httpServer *http.Server) error {
	// Define shutdown timeout
	const shutdownTimeout = 30 * time.Second

	// Create a context with timeout for the entire shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Channel to track shutdown completion
	shutdownComplete := make(chan error, 1)

	go func() {
		// Step 1: Stop accepting new HTTP connections
		log.Println("Step 1: Stopping HTTP server...")
		if err := server.ShutdownServer(httpServer, 15*time.Second); err != nil {
			shutdownComplete <- fmt.Errorf("HTTP server shutdown error: %w", err)
			return
		}

		// Step 2: Shutdown the hub (closes all WebSocket connections)
		log.Println("Step 2: Shutting down WebSocket hub...")
		hub := server.GetHub()
		if err := hub.Shutdown(15 * time.Second); err != nil {
			shutdownComplete <- fmt.Errorf("hub shutdown error: %w", err)
			return
		}

		shutdownComplete <- nil
	}()

	// Wait for shutdown to complete or timeout
	select {
	case err := <-shutdownComplete:
		return err
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout exceeded")
	}
}
