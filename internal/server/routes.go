// Package server wires HTTP handlers into a ServeMux for the Blip
// application via routing helpers.
package server

import "net/http"

// SetupRoutes configures and returns an HTTP ServeMux with all application routes.
// It sets up handlers for health check, WebSocket endpoint, and test page.
func SetupRoutes() *http.ServeMux {
	return SetupRoutesWithHub(GlobalHub())
}

// SetupRoutesWithHub configures and returns an HTTP ServeMux bound to the provided hub.
// It is primarily useful in tests that need route handlers to exercise a specific hub.
func SetupRoutesWithHub(h *Hub) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HealthHandler)
	mux.HandleFunc("/ws", webSocketHandlerForHub(h))
	mux.HandleFunc("/test", TestPageHandler)
	return mux
}
