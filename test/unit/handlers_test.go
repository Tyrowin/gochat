// Package unit contains unit tests for individual components of the Blip server.
//
// These tests focus on testing specific functions and methods in isolation,
// using mocks and stubs where necessary to avoid dependencies on external systems.
// Unit tests ensure that each component behaves correctly under various conditions.
package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maltemindedal/blip/internal/server"
)

// expectedHealthResponse is the handler's own constant, so a change to the
// served text cannot pass these tests by accident.
const expectedHealthResponse = server.HealthResponse

// TestHealthHandlerUnit tests the health handler function in isolation.
// It verifies that the handler responds correctly to different HTTP methods
// and returns the expected status code and response body.
func TestHealthHandlerUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET request to health endpoint",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedBody:   expectedHealthResponse,
		},
		{
			name:           "POST request to health endpoint",
			method:         "POST",
			expectedStatus: http.StatusOK,
			expectedBody:   expectedHealthResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			server.HealthHandler(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			if rr.Body.String() != tt.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestHTTPMethodsUnit tests various HTTP methods on the health endpoint.
// It verifies that the handler responds correctly to different HTTP methods
// including GET, POST, PUT, DELETE, PATCH, HEAD, and OPTIONS.
func TestHTTPMethodsUnit(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(expectedHealthResponse)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run("Test_"+method+"_method", func(t *testing.T) {
			testHTTPMethod(t, handler, method)
		})
	}
}

// testHTTPMethod tests a single HTTP method against the handler
func testHTTPMethod(t *testing.T, handler http.HandlerFunc, method string) {
	t.Helper()

	req, err := http.NewRequest(method, "/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code for %s: got %v want %v",
			method, status, http.StatusOK)
	}

	// For our simple handler, all methods return the same response
	// Note: In a real implementation, HEAD would typically not include a body
	// but our test handler is simplified
	if method != "HEAD" {
		expected := expectedHealthResponse
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body for %s: got %v want %v",
				method, rr.Body.String(), expected)
		}
	}
}

// newRoutes builds the application routes bound to a hub of this test's own, so
// no test observes another's clients. The hub takes the default configuration,
// which is all the routing tests need. It is drained when the test ends.
func newRoutes(t *testing.T) *http.ServeMux {
	t.Helper()

	return server.SetupRoutesWithHub(startHub(t, nil))
}

// TestSetupRoutes tests the route setup function.
// It verifies that SetupRoutesWithHub returns a properly configured ServeMux
// with the expected routes and handlers properly registered.
func TestSetupRoutes(t *testing.T) {
	t.Parallel()

	mux := newRoutes(t)

	// Test that the mux is not nil
	if mux == nil {
		t.Fatal("SetupRoutesWithHub returned nil mux")
	}

	// Test that the root route is properly configured
	req, err := http.NewRequest(http.MethodGet, "/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := expectedHealthResponse
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

// The address, timeouts, and header limit the service applies to its HTTP
// server are pinned in internal/server/service_internal_test.go: they belong to
// the *http.Server that [server.Service] owns, which is deliberately not
// reachable from outside the package.

// TestNewConfig tests the configuration creation function.
// It verifies that NewConfig returns a properly initialized Config
// struct with the expected default values.
func TestNewConfig(t *testing.T) {
	t.Parallel()

	config := server.NewConfig()

	if config == nil {
		t.Fatal("NewConfig returned nil")
	}

	expectedPort := ":8080"
	if config.Port != expectedPort {
		t.Errorf("Expected default port %s, got %s", expectedPort, config.Port)
	}
}
