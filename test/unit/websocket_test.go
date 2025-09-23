package unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tyrowin/gochat/internal/server"
)

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

// TestWebSocketUpgraderConfiguration tests that the upgrader is properly configured
// Note: This is more of an integration test, but we'll keep it simple
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

func TestStartHub(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("StartHub panicked: %v", r)
		}
	}()

	server.StartHub()
}
