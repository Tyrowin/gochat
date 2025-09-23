// Package testhelpers provides common utilities for testing the nexus-chat-server
package testhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// CreateTestServer creates a test HTTP server with the given handler
func CreateTestServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}

// CreateTestServerWithConfig creates a test server with custom configuration
func CreateTestServerWithConfig(
	handler http.Handler,
	readTimeout, writeTimeout, idleTimeout time.Duration,
) *httptest.Server {
	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	testServer := httptest.NewUnstartedServer(handler)
	testServer.Config = server
	testServer.Start()
	return testServer
}

// AssertStatusCode checks if the response has the expected status code
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("Expected status code %d, got %d", expected, resp.StatusCode)
	}
}

// AssertContentType checks if the response has the expected content type
func AssertContentType(t *testing.T, resp *http.Response, expected string) {
	t.Helper()
	contentType := resp.Header.Get("Content-Type")
	if contentType != expected {
		t.Errorf("Expected content type %s, got %s", expected, contentType)
	}
}

// CreateHealthHandler creates the standard health check handler
func CreateHealthHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte("GoChat server is running!")); err != nil {
			// In a real application, you might want to log this error
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

// MakeRequest creates and executes an HTTP request, returning the response
func MakeRequest(t *testing.T, method, url string) *http.Response {
	t.Helper()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(method, url, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	return resp
}
