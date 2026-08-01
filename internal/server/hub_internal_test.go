package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestMain silences server logging so benchmark output stays readable.
func TestMain(m *testing.M) {
	SetLogger(slog.New(slog.DiscardHandler))
	os.Exit(m.Run())
}

func newOriginRequest(tb testing.TB, origin string) *http.Request {
	tb.Helper()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", origin)
	return req
}

// newBenchHub builds a hub with n clients wired directly into the client map,
// so the benchmark measures fan-out rather than pump scheduling.
func newBenchHub(tb testing.TB, n int) (*Hub, []*Client) {
	tb.Helper()

	h := NewHub(nil)
	clients := make([]*Client, n)

	for i := range clients {
		c := &Client{
			send: make(chan []byte, sendBufferSz),
			hub:  h,
			addr: "bench-" + strconv.Itoa(i),
		}
		clients[i] = c
		h.clients[c] = struct{}{}
	}

	return h, clients
}

func BenchmarkHubBroadcast(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(strconv.Itoa(n)+"clients", func(b *testing.B) {
			h, clients := newBenchHub(b, n)
			msg := BroadcastMessage{
				Sender:  clients[0],
				Payload: []byte(`{"content":"hello everyone"}`),
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				h.handleBroadcast(msg)

				// Drain inline. A benchmark loop outruns any consumer
				// goroutine, and the hub evicts clients whose buffers fill,
				// which would shrink the fan-out mid-measurement.
				for _, c := range clients[1:] {
					<-c.send
				}
			}

			if len(h.clients) != n {
				b.Fatalf("hub dropped clients during the benchmark: %d of %d remain", len(h.clients), n)
			}
		})
	}
}

// TestHubRejectsClientWorkAfterShutdown pins the shutdown race that Register and
// Unregister now own: once the run loop has exited, neither may block on a
// channel it will never read again. Both take a *Client, which cannot be built
// without a socket from outside the package, so the test lives here.
func TestHubRejectsClientWorkAfterShutdown(t *testing.T) {
	t.Parallel()

	h := NewHub(nil)
	h.Start()
	h.ClientCount()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := h.Shutdown(ctx); err != nil {
		t.Fatalf("failed to shut the hub down: %v", err)
	}

	client := &Client{hub: h, addr: "shutdown-race"}

	registered := make(chan bool, 1)
	go func() { registered <- h.Register(t.Context(), client) }()

	select {
	case accepted := <-registered:
		if accepted {
			t.Error("Register accepted a client on a stopped hub")
		}
	case <-time.After(time.Second):
		t.Fatal("Register blocked on a stopped hub")
	}

	unregistered := make(chan struct{})
	go func() {
		h.Unregister(client)
		close(unregistered)
	}()

	select {
	case <-unregistered:
	case <-time.After(time.Second):
		t.Fatal("Unregister blocked on a stopped hub")
	}
}

// TestZeroValueRateLimiterAllows pins the zero value as unlimited. A Client
// assembled without NewClient must not be silently throttled to nothing.
func TestZeroValueRateLimiterAllows(t *testing.T) {
	t.Parallel()

	c := &Client{}
	for i := range 100 {
		if !c.rateLimiter.allow() {
			t.Fatalf("zero-value limiter denied message %d", i)
		}
	}
}

// TestRateLimiterThrottlesAtCapacity checks the configured limiter still
// throttles, so the zero-value escape hatch has not disabled the real path.
func TestRateLimiterThrottlesAtCapacity(t *testing.T) {
	t.Parallel()

	rl := newRateLimiter(3, time.Hour)
	for i := range 3 {
		if !rl.allow() {
			t.Fatalf("burst token %d was denied", i)
		}
	}

	if rl.allow() {
		t.Error("limiter allowed a message past its burst")
	}
}

func BenchmarkRateLimiterAllow(b *testing.B) {
	rl := newRateLimiter(1_000_000, time.Second)

	b.ReportAllocs()
	for b.Loop() {
		rl.allow()
	}
}

func BenchmarkOriginCheck(b *testing.B) {
	cfg := resolveConfig(&Config{AllowedOrigins: []string{"http://localhost:8080", "https://example.com"}})

	req := newOriginRequest(b, "https://example.com")

	b.ReportAllocs()
	for b.Loop() {
		if !cfg.isOriginAllowed(req) {
			b.Fatal("expected origin to be allowed")
		}
	}
}
