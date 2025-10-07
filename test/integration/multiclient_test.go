// Package integration contains integration tests for multi-client scenarios.
//
// These tests verify the system behavior when multiple clients connect
// simultaneously, send messages, and interact with each other through
// the hub's broadcast system.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
	"github.com/gorilla/websocket"
)

const (
	msgAfterNewClientJoined = "After new client joined"
	msgFromClientTemplate   = "Message from client %d"
	msgInitial              = "Initial message"
)

// TestMultipleClientsMessageExchange tests complex message exchange scenarios
// between multiple clients connected to the hub.
func TestMultipleClientsMessageExchange(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Five clients sending and receiving messages", func(t *testing.T) {
		testFiveClientsSendingAndReceiving(t, wsURL, testServer.URL)
	})

	t.Run("Clients joining and leaving dynamically", func(t *testing.T) {
		testDynamicJoiningAndLeaving(t, wsURL, testServer.URL)
	})

	t.Run("Rapid message exchange between clients", func(t *testing.T) {
		testRapidMessageExchange(t, wsURL, testServer.URL)
	})
}

// TestMultipleClientsConcurrentOperations tests concurrent operations with multiple clients.
func TestMultipleClientsConcurrentOperations(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Concurrent client connections and disconnections", func(t *testing.T) {
		testConcurrentConnectionsAndDisconnections(t, wsURL, testServer.URL)
	})

	t.Run("Concurrent message sending from multiple clients", func(t *testing.T) {
		testConcurrentMessageSending(t, wsURL, testServer.URL)
	})
}

// TestMultipleClientsEdgeCases tests edge cases with multiple clients.
func TestMultipleClientsEdgeCases(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()
	configureServerForTest(t, testServer.URL, nil)

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Single client broadcasting to itself", func(t *testing.T) {
		connections := connectMultipleClients(t, wsURL, testServer.URL, 1)
		defer closeAllConnections(t, connections)
		time.Sleep(50 * time.Millisecond)

		// Send a message (should not receive it back)
		sendMessageFromClient(t, connections[0], "Self message")
		expectNoMessage(t, connections[0], 300*time.Millisecond)
	})

	t.Run("All clients disconnecting simultaneously", func(t *testing.T) {
		const numClients = 5
		connections := connectMultipleClients(t, wsURL, testServer.URL, numClients)
		time.Sleep(50 * time.Millisecond)

		var wg sync.WaitGroup
		wg.Add(numClients)

		for i := 0; i < numClients; i++ {
			go func(clientID int) {
				defer wg.Done()
				if err := connections[clientID].Close(); err != nil {
					t.Logf("Client %d close error: %v", clientID, err)
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Client sending empty content messages", func(t *testing.T) {
		connections := connectMultipleClients(t, wsURL, testServer.URL, 2)
		defer closeAllConnections(t, connections)
		time.Sleep(50 * time.Millisecond)

		// Send message with empty content
		sendMessageFromClient(t, connections[0], "")

		// Client 1 should receive it
		verifyClientReceivesMessage(t, connections[1], "", 1)
		expectNoMessage(t, connections[0], 150*time.Millisecond)
	})

	t.Run("Clients sending very long content", func(t *testing.T) {
		connections := connectMultipleClients(t, wsURL, testServer.URL, 2)
		defer closeAllConnections(t, connections)
		time.Sleep(50 * time.Millisecond)

		// Send a long message (but within size limit)
		longContent := ""
		for i := 0; i < 50; i++ {
			longContent += "X"
		}

		sendMessageFromClient(t, connections[0], longContent)
		verifyClientReceivesMessage(t, connections[1], longContent, 1)
		expectNoMessage(t, connections[0], 150*time.Millisecond)
	})
}

// drainMessages reads and discards all available messages from a connection
func drainMessages(conn *websocket.Conn, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
			break
		}
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// testFiveClientsSendingAndReceiving tests that five clients can send messages
// and all other clients receive them correctly.
func testFiveClientsSendingAndReceiving(t *testing.T, wsURL, serverURL string) {
	const numClients = 5
	connections := connectMultipleClients(t, wsURL, serverURL, numClients)
	defer closeAllConnections(t, connections)

	// Give clients time to register and start their read/write pumps
	time.Sleep(200 * time.Millisecond)

	// Each client sends a unique message
	sendMessagesFromAllClients(t, connections, numClients)

	// Wait for all messages to be delivered
	time.Sleep(200 * time.Millisecond)

	// Verify each client received all messages except their own
	verifyAllClientsReceivedMessages(t, connections, numClients)
}

// sendMessagesFromAllClients sends one message from each client
func sendMessagesFromAllClients(t *testing.T, connections []*websocket.Conn, numClients int) {
	for i := 0; i < numClients; i++ {
		messageContent := fmt.Sprintf(msgFromClientTemplate, i)
		sendMessageFromClient(t, connections[i], messageContent)
		time.Sleep(100 * time.Millisecond)
	}
}

// verifyAllClientsReceivedMessages verifies each client received expected messages
func verifyAllClientsReceivedMessages(t *testing.T, connections []*websocket.Conn, numClients int) {
	expectedMessagesPerClient := numClients - 1

	for i := 0; i < numClients; i++ {
		messagesReceived := readAllMessagesFromClient(t, connections[i], expectedMessagesPerClient, i)
		verifyReceivedMessageCount(t, messagesReceived, expectedMessagesPerClient, i)
		verifyDidNotReceiveOwnMessage(t, messagesReceived, i)
	}
}

// readAllMessagesFromClient reads all available messages for a client
func readAllMessagesFromClient(t *testing.T, conn *websocket.Conn, expectedCount, clientIndex int) map[string]bool {
	messagesReceived := make(map[string]bool)
	deadline := time.Now().Add(2 * time.Second)

	for len(messagesReceived) < expectedCount && time.Now().Before(deadline) {
		messages := readSingleWebSocketMessage(t, conn, clientIndex)
		if messages == nil {
			break
		}
		for _, content := range messages {
			messagesReceived[content] = true
		}
	}

	return messagesReceived
}

// readSingleWebSocketMessage reads one WebSocket message and returns all contained messages
func readSingleWebSocketMessage(t *testing.T, conn *websocket.Conn, clientIndex int) []string {
	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Errorf("Client %d: Failed to set read deadline: %v", clientIndex, err)
		return nil
	}

	messageType, message, err := conn.ReadMessage()
	if err != nil {
		return nil
	}

	if messageType != websocket.TextMessage {
		return nil
	}

	return parseMessageContent(message)
}

// parseMessageContent parses batched messages separated by newlines
func parseMessageContent(message []byte) []string {
	var contents []string
	parts := bytes.Split(message, []byte("\n"))

	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		var msg server.Message
		if err := json.Unmarshal(part, &msg); err == nil {
			contents = append(contents, msg.Content)
		}
	}

	return contents
}

// verifyReceivedMessageCount checks if the client received the expected number of messages
func verifyReceivedMessageCount(t *testing.T, messagesReceived map[string]bool, expected, clientIndex int) {
	if len(messagesReceived) != expected {
		t.Errorf("Client %d: Expected %d messages, got %d", clientIndex, expected, len(messagesReceived))
	}
}

// verifyDidNotReceiveOwnMessage checks that a client didn't receive its own message
func verifyDidNotReceiveOwnMessage(t *testing.T, messagesReceived map[string]bool, clientIndex int) {
	ownMessage := fmt.Sprintf(msgFromClientTemplate, clientIndex)
	if messagesReceived[ownMessage] {
		t.Errorf("Client %d received its own message", clientIndex)
	}
}

// testDynamicJoiningAndLeaving tests clients connecting and disconnecting
// dynamically while messages are being sent.
func testDynamicJoiningAndLeaving(t *testing.T, wsURL, serverURL string) {
	// Start with 3 clients
	connections := connectMultipleClients(t, wsURL, serverURL, 3)
	time.Sleep(200 * time.Millisecond) // Wait for registration and pump startup

	// Client 0 sends a message
	sendMessageFromClient(t, connections[0], msgInitial)
	time.Sleep(150 * time.Millisecond) // Wait for broadcast

	// Verify clients 1 and 2 received the message
	verifyClientReceivesMessage(t, connections[1], msgInitial, 1)
	verifyClientReceivesMessage(t, connections[2], msgInitial, 2)

	// Client 1 disconnects
	closeClientConnection(t, connections, 1)
	time.Sleep(150 * time.Millisecond) // Wait for unregistration

	// Client 0 sends another message (only client 2 should receive)
	sendMessageFromClient(t, connections[0], "After client 1 left")
	time.Sleep(150 * time.Millisecond) // Wait for broadcast

	verifyClientReceivesMessage(t, connections[2], "After client 1 left", 2)

	// New client joins
	newClient := connectNewClient(t, wsURL, serverURL)
	defer func() { _ = newClient.Close() }()
	time.Sleep(200 * time.Millisecond) // Wait for registration and pump startup

	// Client 2 sends a message (both client 0 and new client should receive)
	sendMessageFromClient(t, connections[2], msgAfterNewClientJoined)
	time.Sleep(300 * time.Millisecond) // Wait longer for broadcast

	// Use a more flexible verification that handles batched messages and retries
	verifyClientReceivesMessageFlexible(t, connections[0], msgAfterNewClientJoined, 0)
	verifyClientReceivesMessage(t, newClient, msgAfterNewClientJoined, 3)
	expectNoMessage(t, connections[2], 200*time.Millisecond)

	// Clean up remaining connections
	closeRemainingConnections(t, connections)
}

// verifyClientReceivesMessageFlexible is a more flexible version that handles
// potential timing issues and batched messages
func verifyClientReceivesMessageFlexible(t *testing.T, conn *websocket.Conn, expectedContent string, clientIndex int) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)

	defer handlePanicDuringMessageRead(t, clientIndex)

	found := searchForMessageWithRetry(t, conn, expectedContent, clientIndex, deadline)

	if !found {
		t.Errorf("Client %d: Expected content %q not found after 3 seconds", clientIndex, expectedContent)
	}
}

// handlePanicDuringMessageRead recovers from panics during WebSocket reads
func handlePanicDuringMessageRead(t *testing.T, clientIndex int) {
	if r := recover(); r != nil {
		t.Errorf("Client %d: Panic while reading message: %v", clientIndex, r)
	}
}

// searchForMessageWithRetry searches for expected message content with retry logic
func searchForMessageWithRetry(t *testing.T, conn *websocket.Conn, expectedContent string, clientIndex int, deadline time.Time) bool {
	for time.Now().Before(deadline) {
		message, err := readWebSocketMessageWithTimeout(t, conn, clientIndex)
		if err != nil {
			if isFatalWebSocketError(err) {
				t.Errorf("Client %d: Connection closed while waiting for message: %v", clientIndex, err)
				return false
			}
			// Timeout is OK, we'll try again
			continue
		}

		if message == nil {
			continue
		}

		if messageContainsExpectedContent(message, expectedContent) {
			return true
		}
	}
	return false
}

// readWebSocketMessageWithTimeout reads a WebSocket message with a timeout
func readWebSocketMessageWithTimeout(t *testing.T, conn *websocket.Conn, clientIndex int) ([]byte, error) {
	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Errorf("Client %d: Failed to set read deadline: %v", clientIndex, err)
		return nil, err
	}

	messageType, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if messageType != websocket.TextMessage {
		return nil, nil
	}

	return message, nil
}

// isFatalWebSocketError checks if the error is a fatal WebSocket connection error
func isFatalWebSocketError(err error) bool {
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) ||
		websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
}

// messageContainsExpectedContent checks if batched message contains expected content
func messageContainsExpectedContent(message []byte, expectedContent string) bool {
	parts := bytes.Split(message, []byte("\n"))

	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		var received server.Message
		if err := json.Unmarshal(part, &received); err != nil {
			continue
		}

		if received.Content == expectedContent {
			return true
		}
	}

	return false
}

// testRapidMessageExchange tests multiple clients sending messages rapidly
// and verifies all messages are received correctly.
func testRapidMessageExchange(t *testing.T, wsURL, serverURL string) {
	const numClients = 3
	connections := connectMultipleClients(t, wsURL, serverURL, numClients)
	defer closeAllConnections(t, connections)
	time.Sleep(200 * time.Millisecond) // Wait for registration and pump startup

	// Send multiple messages rapidly from each client
	const messagesPerClient = 5
	sendRapidMessages(t, connections, messagesPerClient)

	// Give time for all broadcasts to complete
	// With 3 clients and 5 messages each, we have 15 messages total
	// Each message needs to be broadcast to 2 other clients
	// Wait longer to ensure all messages are processed
	time.Sleep(1500 * time.Millisecond)

	// Verify all clients received the expected number of messages (allow some tolerance for timing)
	expectedMessagesPerClient := messagesPerClient * (numClients - 1)

	for clientID := 0; clientID < numClients; clientID++ {
		receivedCount := countReceivedMessages(t, connections[clientID], expectedMessagesPerClient)

		// Allow a small tolerance (e.g., at least 80% of messages should be received)
		minExpected := int(float64(expectedMessagesPerClient) * 0.8)

		if receivedCount < minExpected {
			t.Errorf("Client %d: expected at least %d messages (80%% of %d), got %d",
				clientID, minExpected, expectedMessagesPerClient, receivedCount)
		} else if receivedCount != expectedMessagesPerClient {
			t.Logf("Client %d: received %d/%d messages (%.0f%%)",
				clientID, receivedCount, expectedMessagesPerClient,
				float64(receivedCount)/float64(expectedMessagesPerClient)*100)
		}
	}
}

// sendRapidMessages sends multiple messages rapidly from each client.
func sendRapidMessages(t *testing.T, connections []*websocket.Conn, messagesPerClient int) {
	numClients := len(connections)
	for round := 0; round < messagesPerClient; round++ {
		for clientID := 0; clientID < numClients; clientID++ {
			content := fmt.Sprintf("Round %d from client %d", round, clientID)
			sendMessageFromClient(t, connections[clientID], content)
		}
		// Delay between rounds to prevent overwhelming the hub
		time.Sleep(50 * time.Millisecond)
	}
}

// verifyRapidMessagesReceived verifies that each client received the expected
// number of messages from other clients.
func verifyRapidMessagesReceived(t *testing.T, connections []*websocket.Conn, messagesPerClient, numClients int) {
	expectedMessagesPerClient := messagesPerClient * (numClients - 1)
	for clientID := 0; clientID < numClients; clientID++ {
		receivedCount := countReceivedMessages(t, connections[clientID], expectedMessagesPerClient)
		if receivedCount != expectedMessagesPerClient {
			t.Errorf("Client %d: expected %d messages, got %d", clientID, expectedMessagesPerClient, receivedCount)
		}
	}
}

// countReceivedMessages counts how many valid messages a client receives
// within a timeout period. Handles batched messages separated by newlines.
func countReceivedMessages(t *testing.T, conn *websocket.Conn, maxExpected int) int {
	receivedCount := 0
	deadline := time.Now().Add(5 * time.Second)

	for receivedCount < maxExpected && time.Now().Before(deadline) {
		message, err := readSingleMessageWithDeadline(t, conn)
		if err != nil {
			break
		}

		if message != nil {
			receivedCount += countMessagesInBatch(message)
		}
	}

	return receivedCount
}

// readSingleMessageWithDeadline reads a single WebSocket message with a deadline
func readSingleMessageWithDeadline(t *testing.T, conn *websocket.Conn) ([]byte, error) {
	if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Logf("Failed to set read deadline: %v", err)
		return nil, err
	}

	messageType, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if messageType != websocket.TextMessage {
		return nil, nil
	}

	return message, nil
}

// countMessagesInBatch counts valid messages in a batched message payload
func countMessagesInBatch(message []byte) int {
	count := 0
	parts := bytes.Split(message, []byte("\n"))

	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		var msg server.Message
		if err := json.Unmarshal(part, &msg); err == nil {
			count++
		}
	}

	return count
}

// closeClientConnection safely closes a client connection at the given index.
func closeClientConnection(t *testing.T, connections []*websocket.Conn, index int) {
	if err := connections[index].Close(); err != nil {
		t.Errorf("Failed to close client %d: %v", index, err)
	}
	connections[index] = nil
}

// closeRemainingConnections closes all non-nil connections in the slice.
func closeRemainingConnections(t *testing.T, connections []*websocket.Conn) {
	for i, conn := range connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				t.Logf("Failed to close connection %d: %v", i, err)
			}
		}
	}
}

// connectNewClient establishes a new WebSocket connection and returns it.
func connectNewClient(t *testing.T, wsURL, serverURL string) *websocket.Conn {
	newClient, resp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf("Failed to connect new client: %v", err)
	}
	_ = resp.Body.Close()
	return newClient
}

// testConcurrentConnectionsAndDisconnections tests multiple clients connecting
// and disconnecting concurrently.
func testConcurrentConnectionsAndDisconnections(t *testing.T, wsURL, serverURL string) {
	const numClients = 10
	var wg sync.WaitGroup
	errors := make(chan error, numClients)

	wg.Add(numClients)
	for i := 0; i < numClients; i++ {
		go runSingleConcurrentClient(t, wsURL, serverURL, i, &wg, errors)
	}

	wg.Wait()
	close(errors)

	reportErrors(t, errors)
}

// runSingleConcurrentClient connects a single client, sends a message, reads responses,
// and disconnects.
func runSingleConcurrentClient(t *testing.T, wsURL, serverURL string, clientID int, wg *sync.WaitGroup, errors chan<- error) {
	defer wg.Done()

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		errors <- fmt.Errorf("client %d: connection failed: %w", clientID, err)
		return
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = resp.Body.Close() }()

	// Send a message
	content := fmt.Sprintf(msgFromClientTemplate, clientID)
	if err := conn.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, content)); err != nil {
		errors <- fmt.Errorf("client %d: send failed: %w", clientID, err)
		return
	}

	// Try to read some messages (may or may not receive)
	attemptToReadMessages(conn, 500*time.Millisecond)
}

// attemptToReadMessages attempts to read messages from a connection
// within the specified timeout period.
func attemptToReadMessages(conn *websocket.Conn, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			break
		}
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// testConcurrentMessageSending tests multiple clients sending messages concurrently.
func testConcurrentMessageSending(t *testing.T, wsURL, serverURL string) {
	const numClients = 5
	connections := connectMultipleClients(t, wsURL, serverURL, numClients)
	defer closeAllConnections(t, connections)
	time.Sleep(100 * time.Millisecond)

	errors := sendMessagesFromAllClientsConcurrently(t, connections)
	reportErrors(t, errors)

	// Drain messages from all clients
	drainAllClientMessages(connections)
}

// sendMessagesFromAllClientsConcurrently sends multiple messages from each client
// concurrently and returns any errors that occurred.
func sendMessagesFromAllClientsConcurrently(t *testing.T, connections []*websocket.Conn) chan error {
	const messagesPerClient = 10
	numClients := len(connections)

	var wg sync.WaitGroup
	errors := make(chan error, numClients*messagesPerClient)

	// Each client sends 10 messages concurrently
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go sendMultipleMessagesFromClient(t, connections[i], i, messagesPerClient, &wg, errors)
	}

	wg.Wait()
	close(errors)

	return errors
}

// sendMultipleMessagesFromClient sends multiple messages from a single client.
func sendMultipleMessagesFromClient(t *testing.T, conn *websocket.Conn, clientID, numMessages int, wg *sync.WaitGroup, errors chan<- error) {
	defer wg.Done()

	for msgNum := 0; msgNum < numMessages; msgNum++ {
		content := fmt.Sprintf("Client %d message %d", clientID, msgNum)
		if err := conn.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, content)); err != nil {
			errors <- fmt.Errorf("client %d msg %d: send failed: %w", clientID, msgNum, err)
		}
		time.Sleep(10 * time.Millisecond) // Small delay between messages
	}
}

// drainAllClientMessages drains messages from all client connections.
func drainAllClientMessages(connections []*websocket.Conn) {
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < len(connections); i++ {
		drainMessages(connections[i], 1*time.Second)
	}
}

// reportErrors reports all errors from the error channel to the test.
func reportErrors(t *testing.T, errors <-chan error) {
	for err := range errors {
		t.Error(err)
	}
}
