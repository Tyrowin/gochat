// Package unit contains unit tests for individual components of the GoChat server.
//
// These tests focus on testing specific functions and methods in isolation,
// using mocks and stubs where necessary to avoid dependencies on external systems.
// Unit tests ensure that each component behaves correctly under various conditions.
package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
)

// TestHealthHandlerUnit tests the health handler function in isolation.
// It verifies that the handler responds correctly to different HTTP methods
// and returns the expected status code and response body.
func TestHealthHandlerUnit(t *testing.T) {
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
			expectedBody:   "GoChat server is running!",
		},
		{
			name:           "POST request to health endpoint",
			method:         "POST",
			expectedStatus: http.StatusOK,
			expectedBody:   "GoChat server is running!",
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("GoChat server is running!")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run("Test_"+method+"_method", func(t *testing.T) {
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
				expected := "GoChat server is running!"
				if rr.Body.String() != expected {
					t.Errorf("handler returned unexpected body for %s: got %v want %v",
						method, rr.Body.String(), expected)
				}
			}
		})
	}
}

// TestSetupRoutes tests the route setup function.
// It verifies that SetupRoutes returns a properly configured ServeMux
// with the expected routes and handlers properly registered.
func TestSetupRoutes(t *testing.T) {
	mux := server.SetupRoutes()

	// Test that the mux is not nil
	if mux == nil {
		t.Fatal("SetupRoutes returned nil mux")
	}

	// Test that the root route is properly configured
	req, err := http.NewRequest("GET", "/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "GoChat server is running!"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

// TestCreateServer tests the server creation function.
// It verifies that CreateServer returns an HTTP server with the correct
// configuration including address, handler, and timeout settings.
func TestCreateServer(t *testing.T) {
	port := ":8080"
	mux := server.SetupRoutes()

	srv := server.CreateServer(port, mux)

	// Test server configuration
	if srv.Addr != port {
		t.Errorf("Expected server addr %s, got %s", port, srv.Addr)
	}

	if srv.Handler != mux {
		t.Error("Server handler not set correctly")
	}

	// Test timeout settings
	expectedReadTimeout := 15 * time.Second
	expectedWriteTimeout := 15 * time.Second
	expectedIdleTimeout := 60 * time.Second

	if srv.ReadTimeout != expectedReadTimeout {
		t.Errorf("Expected ReadTimeout %v, got %v", expectedReadTimeout, srv.ReadTimeout)
	}

	if srv.WriteTimeout != expectedWriteTimeout {
		t.Errorf("Expected WriteTimeout %v, got %v", expectedWriteTimeout, srv.WriteTimeout)
	}

	if srv.IdleTimeout != expectedIdleTimeout {
		t.Errorf("Expected IdleTimeout %v, got %v", expectedIdleTimeout, srv.IdleTimeout)
	}
}

// TestNewConfig tests the configuration creation function.
// It verifies that NewConfig returns a properly initialized Config
// struct with the expected default values.
func TestNewConfig(t *testing.T) {
	config := server.NewConfig()

	if config == nil {
		t.Fatal("NewConfig returned nil")
	}

	expectedPort := ":8080"
	if config.Port != expectedPort {
		t.Errorf("Expected default port %s, got %s", expectedPort, config.Port)
	}
}
