// Package integration contains integration tests for the GoChat server.
//
// These tests verify that multiple components work together correctly by testing
// the complete system behavior with real HTTP servers, WebSocket connections,
// and end-to-end functionality. Integration tests ensure that the system works
// as expected when all components are assembled together.
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
	"github.com/gorilla/websocket"
)

// TestWebSocketEndpointIntegration tests the WebSocket endpoint with full server integration.
// It verifies that WebSocket connections can be established, messages can be sent and received,
// and the complete WebSocket functionality works in a real server environment.
func TestWebSocketEndpointIntegration(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	t.Run("Successful WebSocket Connection", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatalf("Failed to connect to WebSocket: %v", err)
		}
		defer func() { _ = conn.Close() }()
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("Expected status %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
		}

		testMessage := "Hello, WebSocket!"
		err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
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

	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	const numClients = 3
	connections := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
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
	testMessage := "Hello from client 0!"
	err = connections[0].WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		t.Fatalf("Failed to send message from client 0: %v", err)
	}

	// Check that all other clients receive the message
	for i := 1; i < numClients; i++ {
		// Set a read deadline
		err = connections[i].SetReadDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			t.Errorf("Failed to set read deadline for client %d: %v", i, err)
			continue
		}

		// Read the broadcasted message
		messageType, message, err := connections[i].ReadMessage()
		if err != nil {
			t.Errorf("Client %d failed to receive broadcasted message: %v", i, err)
			continue
		}

		if messageType != websocket.TextMessage {
			t.Errorf("Client %d: Expected text message, got type %d", i, messageType)
		}

		if string(message) != testMessage {
			t.Errorf("Client %d: Expected message %q, got %q", i, testMessage, string(message))
		}
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
	u, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	t.Run("Connection and Disconnection", func(t *testing.T) {
		// Connect
		conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
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
			conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				t.Fatalf("Failed to connect on iteration %d: %v", i, err)
			}

			// Send a test message
			testMsg := "Test message " + string(rune('A'+i))
			err = conn.WriteMessage(websocket.TextMessage, []byte(testMsg))
			if err != nil {
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
		go func(clientID int) {
			defer func() {
				if r := recover(); r != nil {
					done <- err
				}
			}()

			// Connect
			conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				done <- err
				return
			}
			defer func() { _ = conn.Close() }()
			defer func() { _ = resp.Body.Close() }()

			// Send a message
			message := "Message from client " + string(rune('0'+clientID))
			err = conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				done <- err
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
		}(i)
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
