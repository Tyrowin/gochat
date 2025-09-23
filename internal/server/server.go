// Package server implements the HTTP server functionality for the GoChat server.
package server

import (
	"fmt"
	"net/http"
	"time"
)

// HealthHandler handles the health check endpoint
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "GoChat server is running!")
}

// SetupRoutes configures all HTTP routes for the server
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HealthHandler)
	return mux
}

// CreateServer creates and configures the HTTP server with security settings
func CreateServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Config holds server configuration
type Config struct {
	Port string
}

// NewConfig creates a new server configuration with defaults
func NewConfig() *Config {
	return &Config{
		Port: ":8080",
	}
}

// StartServer starts the HTTP server and blocks until it exits
func StartServer(server *http.Server) error {
	fmt.Printf("Server listening on port %s\n", server.Addr)
	return server.ListenAndServe()
}
