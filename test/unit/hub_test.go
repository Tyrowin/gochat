// Package unit contains unit tests for individual components of the Blip server.
//
// These tests focus on testing specific functions and methods in isolation,
// using mocks and stubs where necessary to avoid dependencies on external systems.
// Unit tests ensure that each component behaves correctly under various conditions.
package unit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/maltemindedal/blip/internal/server"
)

const shutdownErrorMsg = "Failed to shutdown hub: %v"

// shutdownTimeout is the budget every hub shutdown in this file gets. It is
// generous because a failing shutdown should report a real error, not a race
// against a slow machine.
const shutdownTimeout = 5 * time.Second

// startHub runs a hub's event loop and shuts it down when the test ends. It
// returns once the loop is provably serving requests, so no caller needs to
// sleep before using the hub.
func startHub(t *testing.T) *server.Hub {
	t.Helper()

	hub := server.NewHub()
	hub.Start()

	// ClientCount is answered by the Run goroutine, so a reply proves the loop
	// is up and has processed everything queued before this point.
	hub.ClientCount()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := hub.Shutdown(ctx); err != nil {
			t.Errorf(shutdownErrorMsg, err)
		}
	})

	return hub
}

// shutdownHub shuts a hub down with the standard budget and reports failures.
func shutdownHub(t *testing.T, hub *server.Hub) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return hub.Shutdown(ctx)
}

// TestNewHubExposesItsChannels verifies that NewHub returns a hub whose
// register, unregister, and broadcast channels are all usable.
func TestNewHubExposesItsChannels(t *testing.T) {
	hub := server.NewHub()

	if hub.RegisterChan() == nil {
		t.Error("Register channel is nil")
	}
	if hub.UnregisterChan() == nil {
		t.Error("Unregister channel is nil")
	}
	if hub.BroadcastChan() == nil {
		t.Error("Broadcast channel is nil")
	}
}

// TestHubStartsWithNoClients verifies that a freshly started hub reports an
// empty client set.
func TestHubStartsWithNoClients(t *testing.T) {
	hub := startHub(t)

	if count := hub.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients on a new hub, got %d", count)
	}
}

// TestHubAcceptsBroadcastWithNoClients verifies that broadcasting into an empty
// hub is accepted and leaves the hub running.
func TestHubAcceptsBroadcastWithNoClients(t *testing.T) {
	hub := startHub(t)

	select {
	case hub.BroadcastChan() <- server.BroadcastMessage{Payload: []byte(`{"content":"nobody home"}`)}:
	case <-time.After(time.Second):
		t.Fatal("Broadcast channel did not accept a message")
	}

	// The reply proves the loop finished the broadcast and came back around.
	if count := hub.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after an empty broadcast, got %d", count)
	}
}

// TestHubHandlesConcurrentBroadcasts verifies that many goroutines can publish
// to the broadcast channel at once without deadlocking the event loop.
func TestHubHandlesConcurrentBroadcasts(t *testing.T) {
	hub := startHub(t)

	const senders = 10
	var wg sync.WaitGroup
	wg.Add(senders)

	for range senders {
		go func() {
			defer wg.Done()

			select {
			case hub.BroadcastChan() <- server.BroadcastMessage{Payload: []byte(`{"content":"concurrent"}`)}:
			case <-time.After(2 * time.Second):
				t.Error("Broadcast channel blocked under concurrent senders")
			}
		}()
	}

	wg.Wait()

	if count := hub.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after concurrent broadcasts, got %d", count)
	}
}

// TestHubShutdownStopsTheEventLoop verifies that Shutdown drains the hub and
// leaves it reporting stopped.
func TestHubShutdownStopsTheEventLoop(t *testing.T) {
	hub := server.NewHub()
	hub.Start()
	hub.ClientCount()

	if hub.IsStopped() {
		t.Fatal("Hub reported stopped while still running")
	}

	if err := shutdownHub(t, hub); err != nil {
		t.Fatalf(shutdownErrorMsg, err)
	}

	if !hub.IsStopped() {
		t.Error("Hub did not report stopped after Shutdown returned")
	}
}

// TestHubShutdownBeforeStartIsNoOp verifies that shutting down a hub that never
// ran succeeds instead of blocking on an event loop that does not exist.
func TestHubShutdownBeforeStartIsNoOp(t *testing.T) {
	hub := server.NewHub()

	if err := shutdownHub(t, hub); err != nil {
		t.Errorf("Expected shutdown of an unstarted hub to succeed, got: %v", err)
	}
}

// TestHubShutdownIsIdempotent verifies that concurrent and repeated Shutdown
// calls are safe and all report success.
func TestHubShutdownIsIdempotent(t *testing.T) {
	hub := server.NewHub()
	hub.Start()
	hub.ClientCount()

	const callers = 3
	var wg sync.WaitGroup
	wg.Add(callers)

	errs := make(chan error, callers)
	for range callers {
		go func() {
			defer wg.Done()
			errs <- shutdownHub(t, hub)
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("Concurrent shutdown returned an error: %v", err)
		}
	}

	if err := shutdownHub(t, hub); err != nil {
		t.Errorf("Shutdown after shutdown returned an error: %v", err)
	}
}

// TestHubClientCountAfterShutdown verifies that ClientCount stops blocking once
// the event loop has exited.
func TestHubClientCountAfterShutdown(t *testing.T) {
	hub := server.NewHub()
	hub.Start()
	hub.ClientCount()

	if err := shutdownHub(t, hub); err != nil {
		t.Fatalf(shutdownErrorMsg, err)
	}

	done := make(chan int, 1)
	go func() { done <- hub.ClientCount() }()

	select {
	case count := <-done:
		if count != 0 {
			t.Errorf("Expected 0 clients after shutdown, got %d", count)
		}
	case <-time.After(time.Second):
		t.Error("ClientCount blocked after the hub stopped")
	}
}
