// Package integration contains integration tests for the GoChat server.
//
// These tests verify that multiple components work together correctly by testing
// the complete system behavior with real HTTP servers, WebSocket connections,
// and end-to-end functionality. Integration tests ensure that the system works
// as expected when all components are assembled together.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
	"github.com/gorilla/websocket"
)

func mustMarshalMessage(t *testing.T, content string) []byte {
	if t == nil {
		panic("testing.T is required")
	}
	t.Helper()
	payload, err := json.Marshal(server.Message{Content: content})
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}
	return payload
}

func expectNoMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) {
	t.Helper()
	if conn == nil {
		t.Fatalf("nil connection provided to expectNoMessage")
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatalf("Expected no message, but received one")
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return
	}
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return
	}
	t.Fatalf("Unexpected error while waiting for absence of message: %v", err)
}

func configureServerForTest(t *testing.T, baseURL string, customize func(cfg *server.Config)) {
	if t == nil {
		panic("testing.T is required")
	}
	t.Helper()
	cfg := server.NewConfig()
	cfg.AllowedOrigins = append([]string{baseURL}, cfg.AllowedOrigins...)
	if customize != nil {
		customize(cfg)
	}
	server.SetConfig(cfg)
	t.Cleanup(func() {
		server.SetConfig(nil)
	})
}

func newOriginHeader(origin string) http.Header {
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	return header
}

// TestWebSocketEndpointIntegration tests the WebSocket endpoint with full server integration.
// It verifies that WebSocket connections can be established, messages can be sent and received,
// and the complete WebSocket functionality works in a real server environment.
func TestWebSocketEndpointIntegration(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)
	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	t.Run("Successful WebSocket Connection", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer func() { _ = conn.Close() }()
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("Expected status %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
		}

		testMessage := "Hello, WebSocket!"
		err = conn.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, testMessage))
		if err != nil {
			t.Errorf("Failed to send message: %v", err)
		}

		err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			t.Errorf("Failed to send close message: %v", err)
		}
	})

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		resp, err := http.Post(testServer.URL+"/ws", "text/plain", strings.NewReader("test"))
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d for POST request, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
		}
	})

	t.Run("GET Without WebSocket Headers", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/ws")
		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d for GET without WebSocket headers, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
}

// TestWebSocketMessageBroadcasting tests the WebSocket message broadcasting functionality.
// It verifies that messages sent by one client are properly broadcasted to all other
// connected clients through the hub system.
func TestWebSocketMessageBroadcasting(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)

	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	const numClients = 3
	connections := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		defer func(c *websocket.Conn) { _ = c.Close() }(conn)
		defer func() { _ = resp.Body.Close() }()
		connections[i] = conn
	}

	// Give the hub time to register all clients
	time.Sleep(50 * time.Millisecond)

	// Send a message from the first client
	messageContent := "Hello from client 0!"
	if err := connections[0].WriteMessage(websocket.TextMessage, mustMarshalMessage(t, messageContent)); err != nil {
		t.Fatalf("Failed to send message from client 0: %v", err)
	}

	// Check that all other clients receive the message
	for i := 1; i < numClients; i++ {
		if err := connections[i].SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Errorf("Failed to set read deadline for client %d: %v", i, err)
			continue
		}

		messageType, message, err := connections[i].ReadMessage()
		if err != nil {
			t.Errorf("Client %d failed to receive broadcasted message: %v", i, err)
			continue
		}

		if messageType != websocket.TextMessage {
			t.Errorf("Client %d: Expected text message, got type %d", i, messageType)
		}

		var received server.Message
		if err := json.Unmarshal(message, &received); err != nil {
			t.Errorf("Client %d: Failed to unmarshal message: %v", i, err)
			continue
		}

		if received.Content != messageContent {
			t.Errorf("Client %d: Expected content %q, got %q", i, messageContent, received.Content)
		}
	}

	// Ensure the sender does not receive its own message
	expectNoMessage(t, connections[0], 200*time.Millisecond)

	// Send malformed JSON from another client and ensure it is ignored
	if err := connections[1].WriteMessage(websocket.TextMessage, []byte("not valid json")); err != nil {
		t.Fatalf("Failed to send malformed message: %v", err)
	}

	for i := 0; i < numClients; i++ {
		if i == 1 {
			continue
		}
		expectNoMessage(t, connections[i], 150*time.Millisecond)
	}

	// Close all connections gracefully
	for i, conn := range connections {
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			t.Errorf("Failed to send close message for client %d: %v", i, err)
		}
	}
}

// TestWebSocketConnectionLifecycle tests the complete lifecycle of WebSocket connections.
// It verifies that connections can be established, used for communication, and properly
// closed, including testing multiple sequential connections.
func TestWebSocketConnectionLifecycle(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)
	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	t.Run("Connection and Disconnection", func(t *testing.T) {
		// Connect
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		// Test that connection is active
		err = conn.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			t.Errorf("Failed to send ping: %v", err)
		}

		// Close connection
		err = conn.Close()
		if err != nil {
			t.Errorf("Failed to close connection: %v", err)
		}
	})

	t.Run("Multiple Sequential Connections", func(t *testing.T) {
		// Connect and disconnect multiple times
		for i := 0; i < 3; i++ {
			conn, resp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
			if err != nil {
				t.Fatalf("Failed to connect on iteration %d: %v", i, err)
			}

			// Send a test message
			testMsg := "Test message " + string(rune('A'+i))
			if err := conn.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, testMsg)); err != nil {
				t.Errorf("Failed to send message on iteration %d: %v", i, err)
			}

			// Close connection
			_ = conn.Close()
			_ = resp.Body.Close()

			// Brief pause between connections
			time.Sleep(10 * time.Millisecond)
		}
	})
}

// TestWebSocketConcurrentConnections tests concurrent WebSocket connections.
// It verifies that multiple clients can connect simultaneously and exchange messages
// without causing race conditions or system instability.
func TestWebSocketConcurrentConnections(t *testing.T) {
	// Start the hub
	server.StartHub()

	// Create a test server
	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)

	// Convert HTTP URL to WebSocket URL
	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	const numConcurrentClients = 10
	done := make(chan error, numConcurrentClients)

	// Start multiple clients concurrently
	for i := 0; i < numConcurrentClients; i++ {
		message := "Message from client " + string(rune('0'+i))
		payload, err := json.Marshal(server.Message{Content: message})
		if err != nil {
			t.Fatalf("Failed to marshal message for client %d: %v", i, err)
		}

		go func(clientID int, msgPayload []byte) {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("client %d panic: %v", clientID, r)
				}
			}()

			// Connect
			conn, resp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
			if err != nil {
				done <- fmt.Errorf("client %d dial: %w", clientID, err)
				return
			}
			defer func() { _ = conn.Close() }()
			defer func() { _ = resp.Body.Close() }()

			// Send a message
			if err := conn.WriteMessage(websocket.TextMessage, msgPayload); err != nil {
				done <- fmt.Errorf("client %d write: %w", clientID, err)
				return
			}

			// Try to read any broadcasted messages for a short time
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					default:
						_, _, err := conn.ReadMessage()
						if err != nil {
							// Connection might be closed, which is normal
							return
						}
						// Successfully read a message
					}
				}
			}()

			<-ctx.Done()
			done <- nil
		}(i, payload)
	}

	// Wait for all clients to complete
	for i := 0; i < numConcurrentClients; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Client %d failed: %v", i, err)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Client %d timed out", i)
		}
	}
}

func TestWebSocketOriginValidation(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	allowedOrigin := "http://allowed.test"
	configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{testServer.URL, allowedOrigin}
	})

	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	t.Run("Allowed origin", func(t *testing.T) {
		header := http.Header{}
		header.Set("Origin", allowedOrigin)
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), header)
		if err != nil {
			t.Fatalf("Expected allowed origin to succeed: %v", err)
		}
		t.Cleanup(func() {
			_ = conn.Close()
			if resp != nil {
				_ = resp.Body.Close()
			}
		})
		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("Expected status %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
		}
	})

	t.Run("Disallowed origin", func(t *testing.T) {
		header := http.Header{}
		header.Set("Origin", "http://blocked.test")
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), header)
		if err == nil {
			_ = conn.Close()
			if resp != nil {
				_ = resp.Body.Close()
			}
			t.Fatalf("Expected disallowed origin to fail")
		}
		if resp == nil {
			t.Fatalf("Expected HTTP response for disallowed origin")
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected status %d for disallowed origin, got %d", http.StatusForbidden, resp.StatusCode)
		}
	})
}

func TestWebSocketMessageSizeLimit(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	const limit int64 = 64
	configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
		cfg.MaxMessageSize = limit
	})

	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	sender, senderResp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
	if err != nil {
		t.Fatalf("Failed to connect sender: %v", err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
	if err != nil {
		t.Fatalf("Failed to connect receiver: %v", err)
	}
	defer func() { _ = receiver.Close() }()
	defer func() { _ = receiverResp.Body.Close() }()

	oversizedContent := strings.Repeat("A", int(limit)+10)
	oversizedPayload := mustMarshalMessage(t, oversizedContent)
	if int64(len(oversizedPayload)) <= limit {
		t.Fatalf("Test payload is not oversized: %d bytes", len(oversizedPayload))
	}

	if err := sender.WriteMessage(websocket.TextMessage, oversizedPayload); err != nil && !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
		t.Fatalf("Unexpected error writing oversized message: %v", err)
	}

	expectNoMessage(t, receiver, 200*time.Millisecond)

	if err := sender.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}
	if _, _, readErr := sender.ReadMessage(); readErr == nil {
		t.Fatalf("Expected connection closure after oversized message")
	}
}

func TestWebSocketRateLimiting(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	rateCfg := server.RateLimitConfig{Burst: 2, RefillInterval: 500 * time.Millisecond}
	configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
		cfg.RateLimit = rateCfg
	})

	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	sender, senderResp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
	if err != nil {
		t.Fatalf("Failed to connect sender: %v", err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
	if err != nil {
		t.Fatalf("Failed to connect receiver: %v", err)
	}
	defer func() { _ = receiver.Close() }()
	defer func() { _ = receiverResp.Body.Close() }()

	for i := 0; i < rateCfg.Burst; i++ {
		content := fmt.Sprintf("msg-%d", i)
		if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, content)); err != nil {
			t.Fatalf("Failed to send message %d: %v", i, err)
		}
		if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf("Failed to set read deadline: %v", err)
		}
		_, raw, err := receiver.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to receive message %d: %v", i, err)
		}
		var msg server.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("Failed to unmarshal message %d: %v", i, err)
		}
		if msg.Content != content {
			t.Fatalf("Expected content %q, got %q", content, msg.Content)
		}
	}

	if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "over-limit")); err != nil {
		t.Fatalf("Failed to send over-limit message: %v", err)
	}
	expectNoMessage(t, receiver, 200*time.Millisecond)
	_ = receiver.Close()
	_ = receiverResp.Body.Close()
	receiver, receiverResp, err = websocket.DefaultDialer.Dial(u.String(), newOriginHeader(testServer.URL))
	if err != nil {
		t.Fatalf("Failed to reconnect receiver: %v", err)
	}

	time.Sleep(rateCfg.RefillInterval + 100*time.Millisecond)

	if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "after-refill")); err != nil {
		t.Fatalf("Failed to send message after refill: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	received := false
	for time.Now().Before(deadline) {
		if err := receiver.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			t.Fatalf("Failed to set read deadline: %v", err)
		}
		_, raw, err := receiver.ReadMessage()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			t.Fatalf("Failed to receive message after refill: %v", err)
		}
		var msg server.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("Failed to unmarshal message after refill: %v", err)
		}
		if msg.Content == "after-refill" {
			received = true
			break
		}
	}
	if !received {
		t.Fatalf("Expected 'after-refill' message after tokens refilled")
	}
}
