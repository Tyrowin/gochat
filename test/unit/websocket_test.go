// Package unit contains unit tests for individual components of the GoChat server.
//
// These tests focus on testing specific functions and methods in isolation,
// using mocks and stubs where necessary to avoid dependencies on external systems.
// Unit tests ensure that each component behaves correctly under various conditions.
package unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tyrowin/gochat/internal/server"
)

// TestWebSocketHandlerMethodValidation tests the WebSocket handler's HTTP method validation.
// It verifies that the handler correctly rejects non-GET requests with the appropriate
// status code and error message, as WebSocket upgrades require GET requests.
func TestWebSocketHandlerMethodValidation(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "POST request should be rejected",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed. WebSocket endpoint only accepts GET requests.",
		},
		{
			name:           "PUT request should be rejected",
			method:         "PUT",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed. WebSocket endpoint only accepts GET requests.",
		},
		{
			name:           "DELETE request should be rejected",
			method:         "DELETE",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed. WebSocket endpoint only accepts GET requests.",
		},
		{
			name:           "PATCH request should be rejected",
			method:         "PATCH",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed. WebSocket endpoint only accepts GET requests.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/ws", nil)
			w := httptest.NewRecorder()

			server.WebSocketHandler(w, req)

			resp := w.Result()
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body := w.Body.String()
			if strings.TrimSpace(body) != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, strings.TrimSpace(body))
			}
		})
	}
}

// TestWebSocketHandlerGETWithoutUpgrade tests the WebSocket handler's behavior with GET requests
// that don't include proper WebSocket upgrade headers. It verifies that such requests
// are rejected with a Bad Request status code.
func TestWebSocketHandlerGETWithoutUpgrade(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	server.WebSocketHandler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d for invalid WebSocket upgrade, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// TestWebSocketHandlerContentType tests that the WebSocket handler sets the correct
// Content-Type header when rejecting invalid requests. It verifies that error responses
// include the appropriate content type for the error message.
func TestWebSocketHandlerContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "/ws", nil)
	w := httptest.NewRecorder()

	server.WebSocketHandler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to contain 'text/plain', got %q", contentType)
	}
}

// TestWebSocketUpgraderConfiguration tests that the upgrader is properly configured.
// It verifies that requests with proper WebSocket headers are handled appropriately,
// either succeeding with a protocol switch or failing with an appropriate error.
func TestWebSocketUpgraderConfiguration(t *testing.T) {
	// Create a GET request with proper WebSocket headers
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	w := httptest.NewRecorder()

	server.WebSocketHandler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSwitchingProtocols && resp.StatusCode < 400 {
		t.Errorf("Expected either status 101 or an error status (>=400), got %d", resp.StatusCode)
	}
}

// TestWebSocketHandlerWithValidHeaders tests the WebSocket handler with valid WebSocket headers.
// It verifies that requests with proper WebSocket upgrade headers are not rejected
// with a Method Not Allowed status, ensuring the handler recognizes valid WebSocket requests.
func TestWebSocketHandlerWithValidHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)

	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "x3JJHMbDL1EzLkh9GBhXDw==")
	req.Header.Set("Origin", "http://localhost:8080")

	w := httptest.NewRecorder()

	server.WebSocketHandler(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		t.Error("Valid WebSocket request should not return Method Not Allowed")
	}
}

// TestStartHub tests that the StartHub function executes without panicking.
// It verifies that the hub can be started successfully and runs in the background
// without encountering runtime errors during initialization.
func TestStartHub(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("StartHub panicked: %v", r)
		}
	}()

	server.StartHub()
}
