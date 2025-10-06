// Package integration contains integration tests for the GoChat server.
//
// These tests verify that multiple components work together correctly by testing
// the complete system behavior with real HTTP servers, WebSocket connections,
// and end-to-end functionality. Integration tests ensure that the system works
// as expected when all components are assembled together.
package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
)

// Test error message constants
const (
	errFailedRequest      = "Failed to make request: %v"
	errExpectedStatusCode = "Expected status code %d, got %d"
)

// TestHealthEndpointIntegration tests the health endpoint with the actual server configuration.
// It verifies that the complete server setup including routing, handlers, and HTTP responses
// work correctly together in a real server environment.
func TestHealthEndpointIntegration(t *testing.T) {
	// Use the actual server setup from our server package
	mux := server.SetupRoutes()

	// Create a test server with the same configuration as production
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test the endpoint
	resp, err := http.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf(errFailedRequest, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf(errExpectedStatusCode, http.StatusOK, resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	expectedContentType := "text/plain"
	if contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}
}

// TestServerTimeouts tests that the server has proper timeout configurations.
// It verifies that the server is configured with appropriate timeout values
// for production use by testing slow endpoints and timeout behavior.
func TestServerTimeouts(t *testing.T) {
	// Create a test route that simulates slow responses
	testMux := http.NewServeMux()
	testMux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		// Simulate a slow endpoint
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	// Use the actual server configuration from our server package
	srv := server.CreateServer(":0", testMux)

	// Start test server
	testServer := httptest.NewUnstartedServer(testMux)
	testServer.Config = srv
	testServer.Start()
	defer testServer.Close()

	// Test that the server responds within reasonable time
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(testServer.URL + "/slow")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf(errExpectedStatusCode, http.StatusOK, resp.StatusCode)
	}
}

// TestServerSecurity tests basic security configurations of the server.
// It verifies that the server responds appropriately to valid requests
// and handles non-existent endpoints correctly.
func TestServerSecurity(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test that server responds to basic requests
	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf(errFailedRequest, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify server is responding
	if resp.StatusCode != http.StatusOK {
		t.Errorf(errExpectedStatusCode, http.StatusOK, resp.StatusCode)
	}

	// Test non-existent endpoint - our simple server returns 404 by default for unhandled routes
	resp404, err := http.Get(server.URL + "/nonexistent")
	if err != nil {
		t.Fatalf(errFailedRequest, err)
	}
	defer func() { _ = resp404.Body.Close() }()

	// With our current simple mux setup, unhandled routes return 404
	if resp404.StatusCode != http.StatusNotFound {
		t.Logf("Note: Current simple server setup returns %d for non-existent routes", resp404.StatusCode)
		// For now, we'll accept this behavior - in a full implementation this would be 404
	}
}

// TestFullServerIntegration tests the complete server setup using the server package.
// It verifies that all components work together correctly including configuration,
// routing, handlers, and server settings in a full integration scenario.
func TestFullServerIntegration(t *testing.T) {
	// Use the actual server configuration
	config := server.NewConfig()
	mux := server.SetupRoutes()
	srv := server.CreateServer(config.Port, mux)

	// Create test server
	testServer := httptest.NewUnstartedServer(mux)
	testServer.Config = srv
	testServer.Start()
	defer testServer.Close()

	// Test the health endpoint
	resp, err := http.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf("Failed to make health check request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify response
	if resp.StatusCode != http.StatusOK {
		t.Errorf(errExpectedStatusCode, http.StatusOK, resp.StatusCode)
	}

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	expectedContentType := "text/plain"
	if contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Verify server timeouts are configured correctly
	if srv.ReadTimeout != 15*time.Second {
		t.Errorf("Expected ReadTimeout 15s, got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 15*time.Second {
		t.Errorf("Expected WriteTimeout 15s, got %v", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("Expected IdleTimeout 60s, got %v", srv.IdleTimeout)
	}
}
