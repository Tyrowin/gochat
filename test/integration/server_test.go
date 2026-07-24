package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/maltemindedal/blip/internal/server"
	"github.com/maltemindedal/blip/test/testhelpers"
)

// TestHealthEndpointIntegration tests the health endpoint with the actual server configuration.
// It verifies that the complete server setup including routing, handlers, and HTTP responses
// work correctly together in a real server environment.
func TestHealthEndpointIntegration(t *testing.T) {
	testServer, _ := newTestServer(t)

	resp := testhelpers.MakeRequest(t, http.MethodGet, testServer.URL+"/")

	testhelpers.AssertStatusCode(t, resp, http.StatusOK)
	testhelpers.AssertContentType(t, resp, "text/plain")
	testhelpers.AssertBody(t, resp, server.HealthResponse)
}

// TestServerTimeouts tests that the server has proper timeout configurations.
// It verifies that a response slower than a moment still completes, so the
// production write timeout is not cutting responses off early.
func TestServerTimeouts(t *testing.T) {
	// A handler slower than any synchronization delay in this suite. The sleep
	// is the behavior under test, not a wait for something else to happen.
	const handlerDelay = 2 * time.Second

	testMux := http.NewServeMux()
	testMux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(handlerDelay)
		w.WriteHeader(http.StatusOK)
	})

	// Use the production timeouts, which must be generous enough for this.
	production := server.CreateServer(":0", testMux)
	testServer := testhelpers.CreateTestServerWithTimeouts(t, testMux, testhelpers.ServerTimeouts{
		Read:  production.ReadTimeout,
		Write: production.WriteTimeout,
		Idle:  production.IdleTimeout,
	})

	resp := testhelpers.MakeRequest(t, http.MethodGet, testServer.URL+"/slow")
	testhelpers.AssertStatusCode(t, resp, http.StatusOK)
}

// TestUnmatchedPathsServeHealth verifies the documented catch-all behavior: `/`
// is registered as the ServeMux fallback, so every unmatched path returns the
// health response rather than a 404.
func TestUnmatchedPathsServeHealth(t *testing.T) {
	testServer, _ := newTestServer(t)

	resp := testhelpers.MakeRequest(t, http.MethodGet, testServer.URL+"/nonexistent")

	testhelpers.AssertStatusCode(t, resp, http.StatusOK)
	testhelpers.AssertBody(t, resp, server.HealthResponse)
}

// TestFullServerIntegration tests the complete server setup using the server package.
// It verifies that all components work together correctly including configuration,
// routing, handlers, and server settings in a full integration scenario.
func TestFullServerIntegration(t *testing.T) {
	config := server.NewConfig()
	hub := startHub(t)
	t.Cleanup(func() {
		if err := hub.Shutdown(shutdownContext(t, shutdownBudget)); err != nil {
			t.Errorf("Failed to shut down the test hub: %v", err)
		}
	})

	srv := server.CreateServer(config.Port, server.SetupRoutesWithHub(hub))
	testServer := testhelpers.CreateTestServerWithTimeouts(t, srv.Handler, testhelpers.ServerTimeouts{
		Read:  srv.ReadTimeout,
		Write: srv.WriteTimeout,
		Idle:  srv.IdleTimeout,
	})

	resp := testhelpers.MakeRequest(t, http.MethodGet, testServer.URL+"/")
	testhelpers.AssertStatusCode(t, resp, http.StatusOK)
	testhelpers.AssertContentType(t, resp, "text/plain")

	// Verify server timeouts are configured correctly
	if srv.ReadTimeout != 15*time.Second {
		t.Errorf("Expected ReadTimeout 15s, got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 15*time.Second {
		t.Errorf("Expected WriteTimeout 15s, got %v", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("Expected IdleTimeout 60s, got %v", srv.IdleTimeout)
	}
}
