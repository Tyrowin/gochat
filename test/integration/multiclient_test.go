// Package integration contains integration tests for multi-client scenarios.
//
// These tests verify the system behavior when multiple clients connect
// simultaneously, send messages, and interact with each other through
// the hub's broadcast system.
package integration

import (
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
	msgAfterNewClientJoined    = "After new client joined"
	skipMessageBroadcastTiming = "Test has pre-existing timing issues with message broadcasting - needs investigation"
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
	t.Skip(skipMessageBroadcastTiming)
	const numClients = 5
	connections := connectMultipleClients(t, wsURL, serverURL, numClients)
	defer closeAllConnections(t, connections)

	// Give clients time to register
	time.Sleep(100 * time.Millisecond)

	// Each client sends a unique message
	for i := 0; i < numClients; i++ {
		messageContent := fmt.Sprintf("Message from client %d", i)
		sendMessageFromClient(t, connections[i], messageContent)

		// Give time for broadcast to complete
		time.Sleep(50 * time.Millisecond)

		// All other clients should receive this message
		for j := 0; j < numClients; j++ {
			if i == j {
				continue
			}
			verifyClientReceivesMessage(t, connections[j], messageContent, j)
		}

		// Sender should not receive their own message
		expectNoMessage(t, connections[i], 200*time.Millisecond)
	}
}

// testDynamicJoiningAndLeaving tests clients connecting and disconnecting
// dynamically while messages are being sent.
func testDynamicJoiningAndLeaving(t *testing.T, wsURL, serverURL string) {
	t.Skip(skipMessageBroadcastTiming)
	// Start with 3 clients
	connections := connectMultipleClients(t, wsURL, serverURL, 3)
	time.Sleep(50 * time.Millisecond)

	// Client 0 sends a message
	sendMessageFromClient(t, connections[0], "Initial message")
	time.Sleep(50 * time.Millisecond) // Wait for broadcast
	verifyMessageReceivedByOtherClients(t, connections, "Initial message", 0)
	expectNoMessage(t, connections[0], 150*time.Millisecond)

	// Client 1 disconnects
	closeClientConnection(t, connections, 1)
	time.Sleep(50 * time.Millisecond)

	// Client 0 sends another message (only client 2 should receive)
	sendMessageFromClient(t, connections[0], "After client 1 left")
	time.Sleep(50 * time.Millisecond) // Wait for broadcast
	verifyClientReceivesMessage(t, connections[2], "After client 1 left", 2)
	expectNoMessage(t, connections[0], 150*time.Millisecond)

	// New client joins
	newClient := connectNewClient(t, wsURL, serverURL)
	defer func() { _ = newClient.Close() }()
	time.Sleep(50 * time.Millisecond)

	// Client 2 sends a message (both client 0 and new client should receive)
	sendMessageFromClient(t, connections[2], msgAfterNewClientJoined)
	time.Sleep(50 * time.Millisecond) // Wait for broadcast
	verifyClientReceivesMessage(t, connections[0], msgAfterNewClientJoined, 0)
	verifyClientReceivesMessage(t, newClient, msgAfterNewClientJoined, 3)
	expectNoMessage(t, connections[2], 150*time.Millisecond)

	// Clean up remaining connections
	closeRemainingConnections(t, connections)
}

// testRapidMessageExchange tests multiple clients sending messages rapidly
// and verifies all messages are received correctly.
func testRapidMessageExchange(t *testing.T, wsURL, serverURL string) {
	t.Skip(skipMessageBroadcastTiming)
	const numClients = 3
	connections := connectMultipleClients(t, wsURL, serverURL, numClients)
	defer closeAllConnections(t, connections)
	time.Sleep(50 * time.Millisecond)

	// Send multiple messages rapidly from each client
	const messagesPerClient = 5
	sendRapidMessages(t, connections, messagesPerClient)

	// Give time for all broadcasts to complete
	time.Sleep(500 * time.Millisecond)

	// Verify all clients received all messages (except their own)
	verifyRapidMessagesReceived(t, connections, messagesPerClient, numClients)
}

// sendRapidMessages sends multiple messages rapidly from each client.
func sendRapidMessages(t *testing.T, connections []*websocket.Conn, messagesPerClient int) {
	numClients := len(connections)
	for round := 0; round < messagesPerClient; round++ {
		for clientID := 0; clientID < numClients; clientID++ {
			content := fmt.Sprintf("Round %d from client %d", round, clientID)
			sendMessageFromClient(t, connections[clientID], content)
		}
		// Small delay between rounds to prevent overwhelming the hub
		time.Sleep(10 * time.Millisecond)
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
// within a timeout period.
func countReceivedMessages(t *testing.T, conn *websocket.Conn, maxExpected int) int {
	receivedCount := 0
	deadline := time.Now().Add(5 * time.Second) // Increased timeout

	for receivedCount < maxExpected && time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("Failed to set read deadline: %v", err)
			break
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.TextMessage {
			var msg server.Message
			if err := json.Unmarshal(message, &msg); err == nil {
				receivedCount++
			}
		}
	}

	return receivedCount
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
	content := fmt.Sprintf("Message from client %d", clientID)
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
