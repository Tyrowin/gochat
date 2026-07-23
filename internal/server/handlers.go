// Package server exposes HTTP handlers, including WebSocket upgrades, health
// checks, and the built-in test page.
package server

import (
	_ "embed"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

//go:embed testpage.html
var testPageHTML []byte

var (
	healthResponse = []byte("GoChat server is running!")

	// testPageLength is precomputed so the handler does no per-request work
	// beyond writing the embedded bytes.
	testPageLength = strconv.Itoa(len(testPageHTML))
	healthLength   = strconv.Itoa(len(healthResponse))
)

// writeBufferPool lets gorilla/websocket share write buffers across
// connections instead of retaining one per client, which keeps memory flat as
// the connection count grows.
var writeBufferPool = &sync.Pool{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	WriteBufferPool: writeBufferPool,
	CheckOrigin:     checkOrigin,
}

func webSocketHandlerForHub(h *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed. WebSocket endpoint only accepts GET requests.", http.StatusMethodNotAllowed)
			return
		}

		if r.Context().Err() != nil {
			log().Debug("websocket request cancelled before upgrade")
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log().Warn("websocket upgrade failed", "remote_addr", r.RemoteAddr, "error", err)
			return
		}

		client := NewClient(conn, h, r.RemoteAddr)

		select {
		case h.register <- client:
		case <-h.shutdown:
			log().Info("rejected websocket client; hub is shutting down", "remote_addr", r.RemoteAddr)
			client.closeConnection()
		case <-r.Context().Done():
			client.closeConnection()
		}
	}
}

// WebSocketHandler handles WebSocket upgrade requests and manages client connections.
// It validates that the request uses the GET method, upgrades the HTTP connection
// to WebSocket, creates a new Client instance, and starts the client's read/write pumps.
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	webSocketHandlerForHub(GetHub()).ServeHTTP(w, r)
}

// HealthHandler provides a simple health check endpoint that returns server status.
// It responds with a plain text message indicating the server is running.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", healthLength)

	if _, err := w.Write(healthResponse); err != nil {
		log().Debug("error writing health response", "error", err)
	}
}

// TestPageHandler serves an HTML page for exercising the WebSocket endpoint.
// It provides a simple web interface to connect to the WebSocket endpoint,
// send messages, and view real-time chat communication.
func TestPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/html; charset=utf-8")
	header.Set("Content-Length", testPageLength)

	if _, err := w.Write(testPageHTML); err != nil {
		log().Debug("error writing test page response", "error", err)
	}
}
