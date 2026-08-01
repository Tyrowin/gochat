// Package integration contains integration tests for the Blip server.
//
// These tests verify that multiple components work together correctly by testing
// the complete system behavior with real HTTP servers, WebSocket connections,
// and end-to-end functionality. Integration tests ensure that the system works
// as expected when all components are assembled together.
//
// This file holds the plumbing the other files in the package share: starting a
// server backed by its own hub, dialing clients and waiting for the hub to
// register them, and the small assertions used throughout.
package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maltemindedal/blip/internal/server"
	"github.com/maltemindedal/blip/test/testhelpers"
)

const (
	errMsgReadDeadline = "Failed to set read deadline: %v"
	errMsgParseURL     = "Failed to parse test server URL: %v"

	// registerWait is how long a test will wait for the hub to catch up with
	// connections it has already established.
	registerWait = 5 * time.Second
)

// shutdownContext returns a context carrying the given budget, cancelled when
// the test ends.
func shutdownContext(t *testing.T, budget time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), budget)
	t.Cleanup(cancel)
	return ctx
}

// startHub runs a hub's event loop and returns once it is provably serving
// requests, so no caller needs to sleep before using it.
func startHub(t *testing.T) *server.Hub {
	t.Helper()

	hub := server.NewHub()
	hub.Start()

	// ClientCount is answered by the Run goroutine, so a reply proves the loop
	// is up.
	hub.ClientCount()

	return hub
}

// testService is a real [server.Service] running on a real port, driven exactly
// the way main drives it: New, then Run under a context the test cancels.
type testService struct {
	*server.Service

	port string
	stop context.CancelFunc
	done chan error

	once   sync.Once
	runErr error
}

// startService runs a service on port and returns once it is accepting
// requests. It is stopped when the test ends if the test did not stop it itself.
func startService(t *testing.T, port string) *testService {
	t.Helper()

	cfg := server.NewConfig()
	cfg.Port = port
	cfg.AllowedOrigins = []string{testOriginURL, "http://localhost" + port}
	// Rate limiting is exercised in security_test.go; here it would only throttle
	// the messages a lifecycle test needs in flight.
	cfg.RateLimit = server.RateLimitConfig{Burst: 1000, RefillInterval: time.Second}

	// New publishes the config globally, so restore the defaults afterwards.
	svc := server.New(cfg)
	t.Cleanup(func() { server.SetConfig(nil) })

	ctx, cancel := context.WithCancel(context.Background())
	svcTest := &testService{Service: svc, port: port, stop: cancel, done: make(chan error, 1)}

	go func() { svcTest.done <- svc.Run(ctx) }()
	t.Cleanup(func() { _ = svcTest.shutdown(t) })

	testhelpers.WaitForServer(t, svcTest.baseURL()+"/", shutdownBudget)
	return svcTest
}

func (s *testService) baseURL() string { return "http://localhost" + s.port }

func (s *testService) wsURL() string { return "ws://localhost" + s.port + "/ws" }

// shutdown cancels the service's context and returns what Run returned. Calling
// it more than once — which the test cleanup does — replays the first result.
func (s *testService) shutdown(t *testing.T) error {
	t.Helper()

	s.once.Do(func() {
		s.stop()
		select {
		case s.runErr = <-s.done:
		case <-time.After(shutdownBudget):
			t.Error("Run did not return within the shutdown budget")
		}
	})

	return s.runErr
}

// newTestServer starts an HTTP server backed by a hub of its own, so the client
// counts a test observes belong to that test alone. Both are torn down when the
// test ends. Use [startService] instead when the lifecycle itself is under test.
func newTestServer(t *testing.T) (*httptest.Server, *server.Hub) {
	t.Helper()

	hub := startHub(t)
	httpServer := testhelpers.CreateTestServer(t, server.SetupRoutesWithHub(hub))

	t.Cleanup(func() {
		if err := hub.Shutdown(shutdownContext(t, shutdownBudget)); err != nil {
			t.Errorf("Failed to shut down the test hub: %v", err)
		}
	})

	return httpServer, hub
}

// buildWebSocketURL constructs a WebSocket URL from the test server URL
func buildWebSocketURL(t *testing.T, serverURL string) string {
	t.Helper()

	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf(errMsgParseURL, err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"
	return u.String()
}

// dial connects one client and returns once the hub has registered it.
func dial(t *testing.T, hub *server.Hub, wsURL, origin string) *websocket.Conn {
	t.Helper()

	return dialClients(t, hub, wsURL, origin, 1)[0]
}

// dialPair connects the sender/receiver pair that message-delivery tests need,
// returning once the hub has registered both.
func dialPair(t *testing.T, hub *server.Hub, wsURL, origin string) (sender, receiver *websocket.Conn) {
	t.Helper()

	conns := dialClients(t, hub, wsURL, origin, 2)
	return conns[0], conns[1]
}

// dialClients connects n clients and blocks until the hub has registered all of
// them, so a caller never races the registration its assertions depend on.
func dialClients(t *testing.T, hub *server.Hub, wsURL, origin string, n int) []*websocket.Conn {
	t.Helper()

	before := hub.ClientCount()

	conns := make([]*websocket.Conn, n)
	for i := range conns {
		conns[i] = testhelpers.Dial(t, wsURL, origin)
	}

	testhelpers.WaitFor(t, registerWait, "the hub to register every client", func() bool {
		return hub.ClientCount() >= before+n
	})

	return conns
}

// waitForUnregister blocks until the hub's client count drops to want.
func waitForUnregister(t *testing.T, hub *server.Hub, want int) {
	t.Helper()

	testhelpers.WaitFor(t, registerWait, "the hub to unregister a client", func() bool {
		return hub.ClientCount() == want
	})
}

func mustMarshalMessage(t *testing.T, content string) []byte {
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
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return
	}
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return
	}
	t.Fatalf("Unexpected error while waiting for absence of message: %v", err)
}

func configureServerForTest(t *testing.T, baseURL string, customize func(cfg *server.Config)) {
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
