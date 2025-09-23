// Package unit contains unit tests for individual components of the GoChat server.
//
// These tests focus on testing specific functions and methods in isolation,
// using mocks and stubs where necessary to avoid dependencies on external systems.
// Unit tests ensure that each component behaves correctly under various conditions.
package unit

import (
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
)

// TestNewHub tests the hub creation function.
// It verifies that NewHub returns a properly initialized Hub
// with all necessary channels and data structures.
func TestNewHub(t *testing.T) {
	hub := server.NewHub()

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	select {
	case hub.GetRegisterChan() <- nil:
	case <-time.After(10 * time.Millisecond):
	}
}

// TestHubChannels tests that all hub channels are properly initialized.
// It verifies that the register, unregister, and broadcast channels
// are not nil and accessible through their getter methods.
func TestHubChannels(t *testing.T) {
	hub := server.NewHub()

	regChan := hub.GetRegisterChan()
	unregChan := hub.GetUnregisterChan()
	broadcastChan := hub.GetBroadcastChan()

	if regChan == nil {
		t.Error("Register channel is nil")
	}
	if unregChan == nil {
		t.Error("Unregister channel is nil")
	}
	if broadcastChan == nil {
		t.Error("Broadcast channel is nil")
	}
}

// TestHubRunStartsWithoutPanic tests that the hub's Run method starts without panicking.
// It verifies that the hub can be started in a goroutine and runs successfully
// for a short period without encountering runtime errors.
func TestHubRunStartsWithoutPanic(t *testing.T) {
	hub := server.NewHub()

	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Hub.Run() panicked: %v", r)
			}
			done <- true
		}()
		go hub.Run()
		time.Sleep(10 * time.Millisecond)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("Hub.Run() test timed out")
	}
}

// TestHubBroadcastChannel tests the hub's broadcast channel functionality.
// It verifies that messages can be sent to the broadcast channel
// without blocking when the hub is running.
func TestHubBroadcastChannel(t *testing.T) {
	hub := server.NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	testMessage := []byte("test broadcast")

	select {
	case hub.GetBroadcastChan() <- testMessage:
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to send message to broadcast channel")
	}

	time.Sleep(10 * time.Millisecond)
}

// TestNewClient tests the client creation function.
// It verifies that NewClient returns a properly initialized Client
// with all necessary fields and channels set up correctly.
func TestNewClient(t *testing.T) {
	hub := server.NewHub()

	client := server.NewClient(nil, hub, "127.0.0.1:12345")

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	sendChan := client.GetSendChan()
	if sendChan == nil {
		t.Error("Client send channel is nil")
	}
}

// TestClientSendChannel tests the client's send channel functionality.
// It verifies that the client's send channel is properly initialized
// and accessible through the GetSendChan method.
func TestClientSendChannel(t *testing.T) {
	hub := server.NewHub()
	client := server.NewClient(nil, hub, "127.0.0.1:12345")

	sendChan := client.GetSendChan()

	select {
	case <-sendChan:
		t.Error("Expected empty send channel but received a message")
	case <-time.After(10 * time.Millisecond):
	}
}

// TestConcurrentHubOperations tests that the hub handles concurrent operations safely.
// It verifies that multiple goroutines can send messages to the broadcast channel
// simultaneously without causing race conditions or panics.
func TestConcurrentHubOperations(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()

			message := []byte("concurrent message")
			select {
			case hub.GetBroadcastChan() <- message:
			case <-time.After(100 * time.Millisecond):
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Error("Concurrent operations test timed out")
			return
		}
	}
}
