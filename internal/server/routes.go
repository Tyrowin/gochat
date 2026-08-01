// Package server wires HTTP handlers into a ServeMux for the Blip
// application via routing helpers.
package server

import "net/http"

// SetupRoutesWithHub configures and returns an HTTP ServeMux bound to the provided hub.
// [New] wires the service's own hub through it; tests use it to exercise the
// routes against a hub of their own.
func SetupRoutesWithHub(h *Hub) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HealthHandler)
	mux.HandleFunc("/ws", webSocketHandlerForHub(h))
	mux.HandleFunc("/test", TestPageHandler)
	return mux
}
