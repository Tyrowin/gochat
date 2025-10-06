package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
	"github.com/Tyrowin/gochat/test/testhelpers"
	"github.com/gorilla/websocket"
)

const (
	testOriginURL = "http://localhost:8080"
)

// TestGracefulShutdown verifies that the server shuts down gracefully
// when the hub receives a shutdown signal
func TestGracefulShutdown(t *testing.T) {
	// Create a new hub for this test
	hub := server.NewHub()

	// Start the hub
	go hub.Run()

	// Give hub time to start
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown
	err := hub.Shutdown(5 * time.Second)
	if err != nil {
		t.Errorf("Hub shutdown failed: %v", err)
	}
}

// TestGracefulShutdownWithClients verifies that active client connections
// are properly closed during graceful shutdown
func TestGracefulShutdownWithClients(t *testing.T) {
	hub, httpServer := setupShutdownTestServer(t, ":18082")

	numClients := 5
	clients := connectTestClients(t, numClients, "ws://localhost:18082/ws")

	performGracefulShutdown(t, httpServer, hub)
	verifyClientsDisconnected(t, clients, numClients)
}

// setupShutdownTestServer creates and starts a test server for shutdown testing
func setupShutdownTestServer(_ *testing.T, port string) (*server.Hub, *http.Server) {
	config := server.NewConfig()
	config.Port = port
	config.AllowedOrigins = []string{testOriginURL, port}
	server.SetConfig(config)

	hub := server.NewHub()
	go hub.Run()

	mux := server.SetupRoutes()
	httpServer := server.CreateServer(config.Port, mux)

	go func() {
		_ = server.StartServer(httpServer)
	}()

	time.Sleep(100 * time.Millisecond)
	return hub, httpServer
}

// connectTestClients creates multiple WebSocket clients without background readers
func connectTestClients(t *testing.T, numClients int, url string) []*websocket.Conn {
	clients := make([]*websocket.Conn, numClients)

	for i := 0; i < numClients; i++ {
		conn, err := testhelpers.ConnectWebSocket(url)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		clients[i] = conn
	}

	time.Sleep(100 * time.Millisecond)
	return clients
}

// performGracefulShutdown initiates and waits for graceful shutdown to complete
func performGracefulShutdown(t *testing.T, httpServer *http.Server, hub *server.Hub) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownComplete := make(chan error, 1)
	go func() {
		if err := server.ShutdownServer(httpServer, 5*time.Second); err != nil {
			shutdownComplete <- err
			return
		}
		if err := hub.Shutdown(5 * time.Second); err != nil {
			shutdownComplete <- err
			return
		}
		shutdownComplete <- nil
	}()

	select {
	case err := <-shutdownComplete:
		if err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	case <-shutdownCtx.Done():
		t.Fatal("Shutdown timeout exceeded")
	}
}

// verifyClientsDisconnected checks that all client connections are closed
func verifyClientsDisconnected(t *testing.T, clients []*websocket.Conn, expectedCount int) {
	closedClients := 0
	for i, conn := range clients {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, _, err := conn.ReadMessage()
		if err != nil {
			closedClients++
		} else {
			t.Errorf("Client %d still connected after shutdown", i)
		}
		conn.Close()
	}

	if closedClients != expectedCount {
		t.Errorf("Expected %d clients to be closed, got %d", expectedCount, closedClients)
	}
}

// TestShutdownWithActiveMessages verifies that messages in flight are handled
// properly during shutdown
func TestShutdownWithActiveMessages(t *testing.T) {
	hub, httpServer := setupMessageTestServer(t)
	client1, client2 := connectMessageTestClients(t)
	defer client1.Close()
	defer client2.Close()

	messagesSent, messagesReceived := runMessageExchange(t, client1, client2)
	shutdownMessageTestServer(t, httpServer, hub)

	// Log results
	t.Logf("Messages sent: %d, Messages received: %d", messagesSent, messagesReceived)

	// Note: During shutdown, some messages may not be delivered
	// The important thing is the shutdown completes gracefully
	if messagesSent == 0 {
		t.Error("Failed to send any messages")
	}
}

// setupMessageTestServer creates and starts a test server for message testing
func setupMessageTestServer(_ *testing.T) (*server.Hub, *http.Server) {
	config := server.NewConfig()
	config.Port = ":18083"
	config.AllowedOrigins = []string{testOriginURL, "http://localhost:18083"}
	server.SetConfig(config)

	hub := server.NewHub()
	go hub.Run()

	mux := server.SetupRoutes()
	httpServer := server.CreateServer(config.Port, mux)

	go func() {
		_ = server.StartServer(httpServer)
	}()

	time.Sleep(100 * time.Millisecond)
	return hub, httpServer
}

// connectMessageTestClients creates two WebSocket clients for message exchange
func connectMessageTestClients(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	client1, err := testhelpers.ConnectWebSocket("ws://localhost:18083/ws")
	if err != nil {
		t.Fatalf("Failed to connect client1: %v", err)
	}

	client2, err := testhelpers.ConnectWebSocket("ws://localhost:18083/ws")
	if err != nil {
		t.Fatalf("Failed to connect client2: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	return client1, client2
}

// runMessageExchange sends messages from client1 and receives on client2
func runMessageExchange(_ *testing.T, client1, client2 *websocket.Conn) (int, int) {
	messagesSent := 0
	messagesReceived := 0
	var receiveMutex sync.Mutex
	stopReceiving := make(chan struct{})

	// Start receiving on client2
	go receiveMessages(client2, &messagesReceived, &receiveMutex, stopReceiving)

	// Send multiple messages
	for i := 0; i < 10; i++ {
		err := testhelpers.SendMessage(client1, "Test message")
		if err == nil {
			messagesSent++
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait a bit for messages to be delivered
	time.Sleep(200 * time.Millisecond)
	close(stopReceiving)

	return messagesSent, messagesReceived
}

// receiveMessages continuously receives messages on a WebSocket connection
func receiveMessages(client *websocket.Conn, messagesReceived *int, mutex *sync.Mutex, stop chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			// Silently recover from panics during shutdown
		}
	}()

	for {
		select {
		case <-stop:
			return
		default:
			client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			_, _, err := client.ReadMessage()
			if err == nil {
				mutex.Lock()
				(*messagesReceived)++
				mutex.Unlock()
			} else {
				// Connection closed or error - stop receiving
				return
			}
		}
	}
}

// shutdownMessageTestServer initiates graceful shutdown of the test server
func shutdownMessageTestServer(t *testing.T, httpServer *http.Server, hub *server.Hub) {
	if err := server.ShutdownServer(httpServer, 3*time.Second); err != nil {
		t.Logf("HTTP server shutdown error (may be expected): %v", err)
	}

	if err := hub.Shutdown(3 * time.Second); err != nil {
		t.Logf("Hub shutdown error (may be expected): %v", err)
	}
}

// TestShutdownTimeout verifies that shutdown respects timeout
func TestShutdownTimeout(t *testing.T) {
	// Create a hub
	hub := server.NewHub()
	go hub.Run()

	// Give hub time to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown with very short timeout
	start := time.Now()
	err := hub.Shutdown(100 * time.Millisecond)
	elapsed := time.Since(start)

	// Should complete quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}

	// May or may not have error depending on timing
	if err != nil {
		t.Logf("Shutdown returned error (may be expected with short timeout): %v", err)
	}
}

// TestConcurrentShutdown verifies that multiple shutdown calls are safe
func TestConcurrentShutdown(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()

	time.Sleep(50 * time.Millisecond)

	// Call shutdown multiple times concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := hub.Shutdown(2 * time.Second)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Collect any errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Shutdown error: %v", err)
	}

	// First call should succeed, others may timeout or error
	t.Logf("Total shutdown errors: %d (expected: at least 2 of 3 to timeout)", errorCount)
}

// TestNoClientsShutdown verifies shutdown works when no clients are connected
func TestNoClientsShutdown(t *testing.T) {
	config := server.NewConfig()
	config.Port = ":18084"
	config.AllowedOrigins = []string{testOriginURL, "http://localhost:18084"}
	server.SetConfig(config)

	hub := server.NewHub()
	go hub.Run()

	// Setup routes AFTER config to ensure origin validation is configured
	mux := server.SetupRoutes()
	httpServer := server.CreateServer(config.Port, mux)

	go func() {
		_ = server.StartServer(httpServer)
	}()

	time.Sleep(100 * time.Millisecond)

	// Shutdown with no clients
	if err := server.ShutdownServer(httpServer, 2*time.Second); err != nil {
		t.Errorf("HTTP server shutdown failed: %v", err)
	}

	if err := hub.Shutdown(2 * time.Second); err != nil {
		t.Errorf("Hub shutdown failed: %v", err)
	}
}
