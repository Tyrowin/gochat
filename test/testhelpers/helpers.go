// Package testhelpers provides common utilities and helper functions for testing the GoChat server.
//
// This package contains reusable test utilities that are shared across unit and integration tests.
// It provides functions for creating test servers, making HTTP requests, and asserting response
// properties to reduce code duplication in test files.
package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// CreateTestServer creates a test HTTP server with the given handler.
// It returns a running httptest.Server that should be closed after use.
func CreateTestServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}

// CreateTestServerWithConfig creates a test server with custom timeout configuration.
// It allows specifying custom read, write, and idle timeouts for testing server behavior
// under different timeout conditions.
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

// AssertStatusCode checks if the HTTP response has the expected status code.
// It fails the test with a descriptive error message if the status codes don't match.
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("Expected status code %d, got %d", expected, resp.StatusCode)
	}
}

// AssertContentType checks if the HTTP response has the expected Content-Type header.
// It fails the test with a descriptive error message if the content types don't match.
func AssertContentType(t *testing.T, resp *http.Response, expected string) {
	t.Helper()
	contentType := resp.Header.Get("Content-Type")
	if contentType != expected {
		t.Errorf("Expected content type %s, got %s", expected, contentType)
	}
}

// CreateHealthHandler creates the standard health check handler for testing purposes.
// It returns an HTTP handler function that responds with a health check message,
// including proper error handling for write operations.
func CreateHealthHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte("GoChat server is running!")); err != nil {
			// In a real application, you might want to log this error
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

// MakeRequest creates and executes an HTTP request, returning the response.
// It includes a 5-second timeout and fails the test if the request cannot be
// created or executed successfully.
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

// ConnectWebSocket creates a WebSocket connection to the specified URL.
// It returns the connection or an error if connection fails.
func ConnectWebSocket(url string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	// Set a proper origin header for testing
	headers := http.Header{}
	headers.Set("Origin", "http://localhost:8080")

	conn, resp, err := dialer.Dial(url, headers)
	if resp != nil {
		_ = resp.Body.Close()
	}
	return conn, err
}

// SendMessage sends a JSON message over the WebSocket connection.
// It marshals the message with a "content" field and sends it as JSON.
func SendMessage(conn *websocket.Conn, content string) error {
	message := map[string]string{"content": content}
	return conn.WriteJSON(message)
}

// ReceiveMessage reads a JSON message from the WebSocket connection.
// It returns the message content or an error if reading fails.
func ReceiveMessage(conn *websocket.Conn) (map[string]interface{}, error) {
	var message map[string]interface{}
	err := conn.ReadJSON(&message)
	return message, err
}

// SendRawMessage sends a raw byte message over the WebSocket connection.
func SendRawMessage(conn *websocket.Conn, messageType int, data []byte) error {
	return conn.WriteMessage(messageType, data)
}

// ReceiveRawMessage reads a raw message from the WebSocket connection.
func ReceiveRawMessage(conn *websocket.Conn) (int, []byte, error) {
	return conn.ReadMessage()
}

// CloseWebSocket gracefully closes a WebSocket connection.
func CloseWebSocket(conn *websocket.Conn) error {
	err := conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}
	return conn.Close()
}

// AssertMessageContent checks if the received message has the expected content.
func AssertMessageContent(t *testing.T, message map[string]interface{}, expectedContent string) {
	t.Helper()

	content, ok := message["content"]
	if !ok {
		t.Error("Message does not contain 'content' field")
		return
	}

	contentStr, ok := content.(string)
	if !ok {
		t.Error("Message content is not a string")
		return
	}

	if contentStr != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, contentStr)
	}
}

// CreateJSONMessage creates a JSON-encoded message with the given content.
func CreateJSONMessage(content string) ([]byte, error) {
	message := map[string]string{"content": content}
	return json.Marshal(message)
}
