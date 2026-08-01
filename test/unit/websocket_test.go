// Package unit contains unit tests for individual components of the Blip server.
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

	"github.com/maltemindedal/blip/internal/server"
)

const (
	errMethodNotAllowed = "Method not allowed. WebSocket endpoint only accepts GET requests."
)

// serveWebSocket drives req through the application routes bound to a hub of
// this test's own. The upgrade handler is unexported — the service owns it — so
// the mux is how these tests reach it.
func serveWebSocket(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	newRoutes(t).ServeHTTP(w, req)
	return w
}

// TestWebSocketHandlerMethodValidation tests the WebSocket handler's HTTP method validation.
// It verifies that the handler correctly rejects non-GET requests with the appropriate
// status code and error message, as WebSocket upgrades require GET requests.
func TestWebSocketHandlerMethodValidation(t *testing.T) {
	t.Parallel()

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
			expectedBody:   errMethodNotAllowed,
		},
		{
			name:           "PUT request should be rejected",
			method:         "PUT",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   errMethodNotAllowed,
		},
		{
			name:           "DELETE request should be rejected",
			method:         "DELETE",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   errMethodNotAllowed,
		},
		{
			name:           "PATCH request should be rejected",
			method:         "PATCH",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   errMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serveWebSocket(t, httptest.NewRequest(tt.method, "/ws", nil))

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
	t.Parallel()

	w := serveWebSocket(t, httptest.NewRequest(http.MethodGet, "/ws", nil))

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
	t.Parallel()

	w := serveWebSocket(t, httptest.NewRequest(http.MethodPost, "/ws", nil))

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
	t.Parallel()

	// Create a GET request with proper WebSocket headers
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	w := serveWebSocket(t, req)

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
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "x3JJHMbDL1EzLkh9GBhXDw==")
	req.Header.Set("Origin", "http://localhost:8080")

	w := serveWebSocket(t, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		t.Error("Valid WebSocket request should not return Method Not Allowed")
	}
}

// routesAllowing builds the application routes bound to a hub of this test's
// own that allows exactly one origin.
func routesAllowing(t *testing.T, origin string) *http.ServeMux {
	t.Helper()

	cfg := server.NewConfig()
	cfg.AllowedOrigins = []string{origin}

	return server.SetupRoutesWithHub(startHub(t, cfg))
}

// upgradeStatus drives a WebSocket handshake from origin through routes and
// reports the status. A recorder cannot be hijacked, so an accepted handshake
// still fails — later than the origin check does, and with a different status.
func upgradeStatus(t *testing.T, routes *http.ServeMux, origin string) int {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "x3JJHMbDL1EzLkh9GBhXDw==")
	req.Header.Set("Origin", origin)

	w := httptest.NewRecorder()
	routes.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode
}

// TestHubsCarryTheirOwnOriginPolicy pins what the configuration seam is for: the
// allow-list belongs to the hub, so two hubs in one process can disagree about
// the same origin. Each hub must accept its own and reject the other's.
func TestHubsCarryTheirOwnOriginPolicy(t *testing.T) {
	t.Parallel()

	const (
		originA = "http://a.test"
		originB = "http://b.test"
	)

	routesA := routesAllowing(t, originA)
	routesB := routesAllowing(t, originB)

	cases := []struct {
		name    string
		routes  *http.ServeMux
		origin  string
		blocked bool
	}{
		{"hub A allows its own origin", routesA, originA, false},
		{"hub A blocks hub B's origin", routesA, originB, true},
		{"hub B allows its own origin", routesB, originB, false},
		{"hub B blocks hub A's origin", routesB, originA, true},
	}

	for _, tt := range cases {
		status := upgradeStatus(t, tt.routes, tt.origin)

		if blocked := status == http.StatusForbidden; blocked != tt.blocked {
			t.Errorf("%s: expected blocked=%v for origin %s, got status %d",
				tt.name, tt.blocked, tt.origin, status)
		}
	}
}

// TestNewServiceBuildsAServerWithAHub replaces the old StartHub smoke test: the
// startup path is now server.New, which must assemble a service around a hub
// without panicking. Running it is covered in test/integration.
func TestNewServiceBuildsAServerWithAHub(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("server.New panicked: %v", r)
		}
	}()

	svc := server.New(server.NewConfig())

	if svc.Hub() == nil {
		t.Fatal("server.New returned a service without a hub")
	}
}
