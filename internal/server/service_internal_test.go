package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewConfiguresHTTPServer pins the production HTTP server settings New
// applies. They are not reachable from outside the package, so this is where
// the timeouts and the header limit are asserted.
func TestNewConfiguresHTTPServer(t *testing.T) {
	t.Parallel()

	cfg := NewConfig()
	cfg.Port = ":18090"

	svc := New(cfg)

	if svc.Hub() == nil {
		t.Fatal("New returned a service without a hub")
	}
	if svc.httpServer.Handler == nil {
		t.Fatal("New returned a service without routes")
	}
	if svc.httpServer.Addr != cfg.Port {
		t.Errorf("Expected server addr %s, got %s", cfg.Port, svc.httpServer.Addr)
	}

	timeouts := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadTimeout", svc.httpServer.ReadTimeout, 15 * time.Second},
		{"ReadHeaderTimeout", svc.httpServer.ReadHeaderTimeout, 5 * time.Second},
		{"WriteTimeout", svc.httpServer.WriteTimeout, 15 * time.Second},
		{"IdleTimeout", svc.httpServer.IdleTimeout, 60 * time.Second},
	}
	for _, tt := range timeouts {
		if tt.got != tt.want {
			t.Errorf("Expected %s %v, got %v", tt.name, tt.want, tt.got)
		}
	}

	if svc.httpServer.MaxHeaderBytes != 1<<16 {
		t.Errorf("Expected MaxHeaderBytes %d, got %d", 1<<16, svc.httpServer.MaxHeaderBytes)
	}
}

// TestNewServesTheApplicationRoutes verifies that the handler New builds answers
// the health route, so the service is wired to the real mux rather than an empty
// one.
func TestNewServesTheApplicationRoutes(t *testing.T) {
	t.Parallel()

	svc := New(NewConfig())

	rec := httptest.NewRecorder()
	svc.httpServer.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d from /, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != HealthResponse {
		t.Errorf("Expected body %q, got %q", HealthResponse, rec.Body.String())
	}
}

// TestProductionTimeoutsAllowSlowResponses verifies that a response slower than
// a moment still completes under the timeouts New applies, so the production
// write timeout is not cutting responses off early.
func TestProductionTimeoutsAllowSlowResponses(t *testing.T) {
	t.Parallel()

	// A handler slower than any synchronization delay in this suite. The sleep
	// is the behavior under test, not a wait for something else to happen.
	const handlerDelay = 2 * time.Second

	production := New(NewConfig()).httpServer

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(handlerDelay)
		w.WriteHeader(http.StatusOK)
	})

	testServer := httptest.NewUnstartedServer(mux)
	testServer.Config.ReadTimeout = production.ReadTimeout
	testServer.Config.WriteTimeout = production.WriteTimeout
	testServer.Config.IdleTimeout = production.IdleTimeout
	testServer.Start()
	t.Cleanup(testServer.Close)

	client := testServer.Client()
	client.Timeout = handlerDelay + 5*time.Second

	resp, err := client.Get(testServer.URL + "/slow")
	if err != nil {
		t.Fatalf("Slow request failed under the production timeouts: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}
