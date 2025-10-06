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

const (
	errMsgReadDeadline = "Failed to set read deadline: %v"
	errMsgParseURL     = "Failed to parse test server URL: %v"
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
		t.Fatalf(errMsgReadDeadline, err)
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

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Successful WebSocket Connection", func(t *testing.T) {
		testSuccessfulWebSocketConnection(t, wsURL, testServer.URL)
	})

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		testInvalidHTTPMethod(t, testServer.URL)
	})

	t.Run("GET Without WebSocket Headers", func(t *testing.T) {
		testGETWithoutWebSocketHeaders(t, testServer.URL)
	})
}

// buildWebSocketURL constructs a WebSocket URL from the test server URL
func buildWebSocketURL(t *testing.T, serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf(errMsgParseURL, err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"
	return u.String()
}

// testSuccessfulWebSocketConnection tests establishing a WebSocket connection and sending messages
func testSuccessfulWebSocketConnection(t *testing.T, wsURL, serverURL string) {
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
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
}

// testInvalidHTTPMethod verifies that POST requests to WebSocket endpoint are rejected
func testInvalidHTTPMethod(t *testing.T, serverURL string) {
	resp, err := http.Post(serverURL+"/ws", "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d for POST request, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

// testGETWithoutWebSocketHeaders verifies that GET requests without WebSocket headers are rejected
func testGETWithoutWebSocketHeaders(t *testing.T, serverURL string) {
	resp, err := http.Get(serverURL + "/ws")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d for GET without WebSocket headers, got %d", http.StatusBadRequest, resp.StatusCode)
	}
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

	wsURL := buildWebSocketURL(t, testServer.URL)
	connections := connectMultipleClients(t, wsURL, testServer.URL, 3)

	// Ensure all connections are closed at the end
	defer func() {
		for _, conn := range connections {
			if conn != nil {
				if err := conn.Close(); err != nil {
					t.Logf("Failed to close connection: %v", err)
				}
			}
		}
	}()

	// Give the hub time to register all clients
	time.Sleep(50 * time.Millisecond)

	messageContent := "Hello from client 0!"
	sendMessageFromClient(t, connections[0], messageContent)
	verifyMessageReceivedByOtherClients(t, connections, messageContent, 0)
	expectNoMessage(t, connections[0], 200*time.Millisecond)

	testMalformedMessageIgnored(t, connections)
	closeAllConnections(t, connections)
}

// connectMultipleClients establishes multiple WebSocket connections
func connectMultipleClients(t *testing.T, wsURL, serverURL string, numClients int) []*websocket.Conn {
	connections := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		// Don't defer close here - let the caller handle cleanup
		_ = resp.Body.Close()
		connections[i] = conn
	}
	return connections
}

// sendMessageFromClient sends a message from a specific client
func sendMessageFromClient(t *testing.T, conn *websocket.Conn, content string) {
	if err := conn.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, content)); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
}

// verifyMessageReceivedByOtherClients checks that all clients except sender receive the message
func verifyMessageReceivedByOtherClients(t *testing.T, connections []*websocket.Conn, expectedContent string, senderIndex int) {
	for i := 1; i < len(connections); i++ {
		if i == senderIndex {
			continue
		}
		verifyClientReceivesMessage(t, connections[i], expectedContent, i)
	}
}

// verifyClientReceivesMessage verifies a single client receives the expected message
func verifyClientReceivesMessage(t *testing.T, conn *websocket.Conn, expectedContent string, clientIndex int) {
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Errorf("Failed to set read deadline for client %d: %v", clientIndex, err)
		return
	}

	messageType, message, err := conn.ReadMessage()
	if err != nil {
		t.Errorf("Client %d failed to receive broadcasted message: %v", clientIndex, err)
		return
	}

	if messageType != websocket.TextMessage {
		t.Errorf("Client %d: Expected text message, got type %d", clientIndex, messageType)
	}

	var received server.Message
	if err := json.Unmarshal(message, &received); err != nil {
		t.Errorf("Client %d: Failed to unmarshal message: %v", clientIndex, err)
		return
	}

	if received.Content != expectedContent {
		t.Errorf("Client %d: Expected content %q, got %q", clientIndex, expectedContent, received.Content)
	}
}

// testMalformedMessageIgnored sends malformed JSON and verifies it's ignored by all clients
func testMalformedMessageIgnored(t *testing.T, connections []*websocket.Conn) {
	if err := connections[1].WriteMessage(websocket.TextMessage, []byte("not valid json")); err != nil {
		t.Fatalf("Failed to send malformed message: %v", err)
	}

	for i := 0; i < len(connections); i++ {
		if i == 1 {
			continue
		}
		expectNoMessage(t, connections[i], 150*time.Millisecond)
	}
}

// closeAllConnections gracefully closes all WebSocket connections
func closeAllConnections(t *testing.T, connections []*websocket.Conn) {
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
		t.Fatalf(errMsgParseURL, err)
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
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)

	wsURL := buildWebSocketURL(t, testServer.URL)

	const numConcurrentClients = 10
	done := make(chan error, numConcurrentClients)

	launchConcurrentClients(wsURL, testServer.URL, numConcurrentClients, done)
	waitForConcurrentClients(t, numConcurrentClients, done)
}

// launchConcurrentClients starts multiple WebSocket clients concurrently
func launchConcurrentClients(wsURL, serverURL string, numClients int, done chan error) {
	for i := 0; i < numClients; i++ {
		message := "Message from client " + string(rune('0'+i))
		payload, err := json.Marshal(server.Message{Content: message})
		if err != nil {
			done <- fmt.Errorf("failed to marshal message for client %d: %w", i, err)
			continue
		}

		go runConcurrentClient(i, wsURL, serverURL, payload, done)
	}
}

// runConcurrentClient runs a single concurrent WebSocket client
func runConcurrentClient(clientID int, wsURL, serverURL string, msgPayload []byte, done chan error) {
	defer func() {
		if r := recover(); r != nil {
			done <- fmt.Errorf("client %d panic: %v", clientID, r)
		}
	}()

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		done <- fmt.Errorf("client %d dial: %w", clientID, err)
		return
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = resp.Body.Close() }()

	if err := conn.WriteMessage(websocket.TextMessage, msgPayload); err != nil {
		done <- fmt.Errorf("client %d write: %w", clientID, err)
		return
	}

	readMessagesWithTimeout(conn, 100*time.Millisecond)
	done <- nil
}

// readMessagesWithTimeout reads messages from a connection with a timeout
func readMessagesWithTimeout(conn *websocket.Conn, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, _, err := conn.ReadMessage()
				if err != nil {
					return
				}
			}
		}
	}()

	<-ctx.Done()
}

// waitForConcurrentClients waits for all concurrent clients to complete
func waitForConcurrentClients(t *testing.T, numClients int, done chan error) {
	for i := 0; i < numClients; i++ {
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

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Allowed origin", func(t *testing.T) {
		testAllowedOrigin(t, wsURL, allowedOrigin)
	})

	t.Run("Disallowed origin", func(t *testing.T) {
		testDisallowedOrigin(t, wsURL)
	})
}

// testAllowedOrigin verifies that connections from allowed origins succeed
func testAllowedOrigin(t *testing.T, wsURL, allowedOrigin string) {
	header := http.Header{}
	header.Set("Origin", allowedOrigin)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
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
}

// testDisallowedOrigin verifies that connections from disallowed origins are rejected
func testDisallowedOrigin(t *testing.T, wsURL string) {
	header := http.Header{}
	header.Set("Origin", "http://blocked.test")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
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
		t.Fatalf(errMsgParseURL, err)
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
		t.Fatalf(errMsgReadDeadline, err)
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

	wsURL := buildWebSocketURL(t, testServer.URL)
	sender, senderResp := connectRateLimitClient(t, wsURL, testServer.URL, "sender")
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp := connectRateLimitClient(t, wsURL, testServer.URL, "receiver")
	defer func() { _ = receiver.Close() }()
	defer func() { _ = receiverResp.Body.Close() }()

	sendAndReceiveBurstMessages(t, sender, receiver, rateCfg.Burst)
	testOverLimitMessageRejected(t, sender, receiver)

	receiver, receiverResp = reconnectReceiver(t, wsURL, testServer.URL, receiver, receiverResp)
	testMessageAfterRefill(t, sender, receiver, rateCfg.RefillInterval)
}

// connectRateLimitClient establishes a WebSocket connection for rate limit testing
func connectRateLimitClient(t *testing.T, wsURL, serverURL, clientName string) (*websocket.Conn, *http.Response) {
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf("Failed to connect %s: %v", clientName, err)
	}
	return conn, resp
}

// sendAndReceiveBurstMessages sends and receives messages up to the burst limit
func sendAndReceiveBurstMessages(t *testing.T, sender, receiver *websocket.Conn, burstLimit int) {
	for i := 0; i < burstLimit; i++ {
		content := fmt.Sprintf("msg-%d", i)
		sendAndVerifyMessage(t, sender, receiver, content, i)
	}
}

// sendAndVerifyMessage sends a message from sender and verifies receiver gets it
func sendAndVerifyMessage(t *testing.T, sender, receiver *websocket.Conn, content string, msgNum int) {
	if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, content)); err != nil {
		t.Fatalf("Failed to send message %d: %v", msgNum, err)
	}

	if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf(errMsgReadDeadline, err)
	}

	_, raw, err := receiver.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to receive message %d: %v", msgNum, err)
	}

	var msg server.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("Failed to unmarshal message %d: %v", msgNum, err)
	}

	if msg.Content != content {
		t.Fatalf("Expected content %q, got %q", content, msg.Content)
	}
}

// testOverLimitMessageRejected verifies that messages over the rate limit are rejected
func testOverLimitMessageRejected(t *testing.T, sender, receiver *websocket.Conn) {
	if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "over-limit")); err != nil {
		t.Fatalf("Failed to send over-limit message: %v", err)
	}
	expectNoMessage(t, receiver, 200*time.Millisecond)
}

// reconnectReceiver closes and reconnects the receiver client
func reconnectReceiver(t *testing.T, wsURL, serverURL string, oldReceiver *websocket.Conn, oldResp *http.Response) (*websocket.Conn, *http.Response) {
	_ = oldReceiver.Close()
	_ = oldResp.Body.Close()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf("Failed to reconnect receiver: %v", err)
	}
	return receiver, receiverResp
}

// testMessageAfterRefill verifies that messages can be sent after the rate limit refills
func testMessageAfterRefill(t *testing.T, sender, receiver *websocket.Conn, refillInterval time.Duration) {
	time.Sleep(refillInterval + 100*time.Millisecond)

	if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "after-refill")); err != nil {
		t.Fatalf("Failed to send message after refill: %v", err)
	}

	waitForSpecificMessage(t, receiver, "after-refill", 2*time.Second)
}

// waitForSpecificMessage waits for a specific message content to be received
func waitForSpecificMessage(t *testing.T, receiver *websocket.Conn, expectedContent string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := receiver.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			t.Fatalf(errMsgReadDeadline, err)
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

		if msg.Content == expectedContent {
			return
		}
	}

	t.Fatalf("Expected '%s' message after tokens refilled", expectedContent)
}
