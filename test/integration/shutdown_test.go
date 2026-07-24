package integration

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maltemindedal/blip/internal/server"
	"github.com/maltemindedal/blip/test/testhelpers"
)

const (
	testOriginURL = "http://localhost:8080"

	// shutdownBudget is the deadline every shutdown in this package gets unless
	// it is deliberately testing a short one.
	shutdownBudget = 5 * time.Second
)

// TestGracefulShutdown verifies that the server shuts down gracefully
// when the hub receives a shutdown signal
func TestGracefulShutdown(t *testing.T) {
	hub := startHub(t)

	if err := hub.Shutdown(shutdownContext(t, shutdownBudget)); err != nil {
		t.Errorf("Hub shutdown failed: %v", err)
	}

	if !hub.IsStopped() {
		t.Error("Hub did not report stopped after shutdown")
	}
}

// TestGracefulShutdownWithClients verifies that active client connections
// are properly closed during graceful shutdown
func TestGracefulShutdownWithClients(t *testing.T) {
	hub, httpServer := setupShutdownTestServer(t, ":18082")

	numClients := 5
	clients := connectTestClients(t, hub, numClients, "ws://localhost:18082/ws")

	performGracefulShutdown(t, httpServer, hub)
	verifyClientsDisconnected(t, clients, numClients)
}

// setupShutdownTestServer creates and starts a test server for shutdown testing
func setupShutdownTestServer(t *testing.T, port string) (*server.Hub, *http.Server) {
	t.Helper()
	t.Cleanup(func() {
		server.SetConfig(nil)
	})

	config := server.NewConfig()
	config.Port = port
	config.AllowedOrigins = []string{testOriginURL, "http://localhost" + port}
	// Rate limiting is exercised in security_test.go; here it would only throttle
	// the messages a shutdown test needs in flight.
	config.RateLimit = server.RateLimitConfig{Burst: 1000, RefillInterval: time.Second}
	server.SetConfig(config)

	hub := startHub(t)

	mux := server.SetupRoutesWithHub(hub)
	httpServer := server.CreateServer(config.Port, mux)

	go func() {
		_ = server.StartServer(httpServer)
	}()

	testhelpers.WaitForServer(t, "http://localhost"+port+"/", shutdownBudget)
	return hub, httpServer
}

// connectTestClients creates multiple WebSocket clients and returns once the hub
// has registered every one of them.
func connectTestClients(t *testing.T, hub *server.Hub, numClients int, url string) []*websocket.Conn {
	t.Helper()

	clients := make([]*websocket.Conn, numClients)

	for i := range numClients {
		conn, err := testhelpers.ConnectWebSocket(url)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		clients[i] = conn
	}

	testhelpers.WaitFor(t, shutdownBudget, "every client to register", func() bool {
		return hub.ClientCount() == numClients
	})

	return clients
}

// performGracefulShutdown initiates and waits for graceful shutdown to complete
func performGracefulShutdown(t *testing.T, httpServer *http.Server, hub *server.Hub) {
	t.Helper()

	if err := server.ShutdownServer(shutdownContext(t, shutdownBudget), httpServer); err != nil {
		t.Errorf("HTTP server shutdown failed: %v", err)
	}

	if err := hub.Shutdown(shutdownContext(t, shutdownBudget)); err != nil {
		t.Errorf("Hub shutdown failed: %v", err)
	}
}

// verifyClientsDisconnected checks that all client connections are closed
func verifyClientsDisconnected(t *testing.T, clients []*websocket.Conn, expectedCount int) {
	t.Helper()

	closedClients := 0
	for i, conn := range clients {
		if err := conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("Failed to set read deadline for client %d: %v", i, err)
		}
		_, _, err := conn.ReadMessage()
		if err != nil {
			closedClients++
		} else {
			t.Errorf("Client %d still connected after shutdown", i)
		}
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close client %d: %v", i, err)
		}
	}

	if closedClients != expectedCount {
		t.Errorf("Expected %d clients to be closed, got %d", expectedCount, closedClients)
	}
}

// TestShutdownWithActiveMessages verifies that messages in flight are delivered
// before shutdown tears the hub down.
func TestShutdownWithActiveMessages(t *testing.T) {
	hub, httpServer := setupShutdownTestServer(t, ":18083")
	clients := connectTestClients(t, hub, 2, "ws://localhost:18083/ws")
	sender, receiver := clients[0], clients[1]

	const messageCount = 10
	for i := range messageCount {
		if err := testhelpers.SendMessage(sender, "Test message"); err != nil {
			t.Fatalf("Failed to send message %d: %v", i, err)
		}
	}

	received := countReceivedMessages(t, receiver, messageCount)
	if received != messageCount {
		t.Errorf("Expected %d messages before shutdown, got %d", messageCount, received)
	}

	performGracefulShutdown(t, httpServer, hub)
}

// TestShutdownTimeout verifies that shutdown returns promptly rather than
// blocking for its whole budget when there is nothing left to drain.
func TestShutdownTimeout(t *testing.T) {
	hub := startHub(t)

	// A budget this short is only met if Shutdown returns as soon as the event
	// loop and the pumps are done, rather than waiting out a timer.
	if err := hub.Shutdown(shutdownContext(t, 100*time.Millisecond)); err != nil {
		t.Errorf("Expected an idle hub to shut down within its budget, got: %v", err)
	}
}

// TestConcurrentShutdown verifies that multiple shutdown calls are safe and all
// report success.
func TestConcurrentShutdown(t *testing.T) {
	hub := startHub(t)

	const callers = 3
	var wg sync.WaitGroup
	wg.Add(callers)

	errs := make(chan error, callers)
	for range callers {
		go func() {
			defer wg.Done()
			errs <- hub.Shutdown(shutdownContext(t, shutdownBudget))
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("Concurrent shutdown returned an error: %v", err)
		}
	}
}

// TestNoClientsShutdown verifies shutdown works when no clients are connected
func TestNoClientsShutdown(t *testing.T) {
	hub, httpServer := setupShutdownTestServer(t, ":18084")

	if count := hub.ClientCount(); count != 0 {
		t.Fatalf("Expected no clients, got %d", count)
	}

	performGracefulShutdown(t, httpServer, hub)
}
