// Package server wires HTTP handlers into a ServeMux for the GoChat
// application via routing helpers.
package server

import "net/http"

// SetupRoutes configures and returns an HTTP ServeMux with all application routes.
// It sets up handlers for health check, WebSocket endpoint, and test page.
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HealthHandler)
	mux.HandleFunc("/ws", WebSocketHandler)
	mux.HandleFunc("/test", TestPageHandler)
	return mux
}
