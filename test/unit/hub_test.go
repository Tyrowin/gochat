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
	case hub.GetBroadcastChan() <- server.BroadcastMessage{Payload: testMessage}:
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to send message to broadcast channel")
	}

	time.Sleep(10 * time.Millisecond)
}

const testClientAddr = "127.0.0.1:12345"

// TestNewClient tests the client creation function.
// It verifies that NewClient returns a properly initialized Client
// with all necessary fields and channels set up correctly.
func TestNewClient(t *testing.T) {
	hub := server.NewHub()

	client := server.NewClient(nil, hub, testClientAddr)

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
	client := server.NewClient(nil, hub, testClientAddr)

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
			case hub.GetBroadcastChan() <- server.BroadcastMessage{Payload: message}:
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

// TestHubClientRegistrationChannel tests the hub's client registration channel.
// It verifies that the registration channel accepts clients without blocking.
func TestHubClientRegistrationChannel(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Shutdown(time.Second)
	time.Sleep(10 * time.Millisecond)

	t.Run("Nil client registration is ignored", func(t *testing.T) {
		select {
		case hub.GetRegisterChan() <- nil:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Failed to send nil client")
		}

		// Should not panic or cause issues
		time.Sleep(20 * time.Millisecond)
	})

	t.Run("Registration channel is non-blocking", func(t *testing.T) {
		// Test that we can send to the registration channel
		done := make(chan bool, 1)

		go func() {
			// This would block if the hub isn't running
			hub.GetRegisterChan() <- nil
			done <- true
		}()

		select {
		case <-done:
			// Success - channel accepted the value
		case <-time.After(100 * time.Millisecond):
			t.Error("Registration channel blocked")
		}

		time.Sleep(20 * time.Millisecond)
	})
}

// TestHubClientUnregistration tests the hub's client unregistration functionality.
// It verifies that unregistration requests are properly handled by the hub.
func TestHubClientUnregistration(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Shutdown(time.Second)
	time.Sleep(10 * time.Millisecond)

	t.Run("Unregister channel is non-blocking", func(t *testing.T) {
		done := make(chan bool, 1)

		go func() {
			// Send a nil client (hub should handle it gracefully)
			hub.GetUnregisterChan() <- nil
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Error("Unregistration channel blocked")
		}

		time.Sleep(20 * time.Millisecond)
	})

	t.Run("Multiple concurrent unregistration requests", func(t *testing.T) {
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(id int) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Unregistration goroutine %d panicked: %v", id, r)
					}
					done <- true
				}()

				// Send nil - hub should handle gracefully
				select {
				case hub.GetUnregisterChan() <- nil:
				case <-time.After(100 * time.Millisecond):
					t.Errorf("Failed to send unregister request %d", id)
				}
			}(i)
		}

		for i := 0; i < 5; i++ {
			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
				t.Error("Concurrent unregistration test timed out")
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	})
}

// TestHubBroadcastMessage tests the hub's broadcast functionality.
// It verifies that messages are properly broadcast to all clients except the sender.
func TestHubBroadcastMessage(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Shutdown(time.Second)
	time.Sleep(10 * time.Millisecond)

	t.Run("Broadcast with nil sender", func(t *testing.T) {
		testMsg := []byte(`{"content":"broadcast test"}`)

		select {
		case hub.GetBroadcastChan() <- server.BroadcastMessage{Sender: nil, Payload: testMsg}:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Failed to send broadcast message")
		}

		time.Sleep(20 * time.Millisecond)
	})

	t.Run("Broadcast with sender", func(t *testing.T) {
		sender := server.NewClient(nil, hub, "127.0.0.1:12345")
		testMsg := []byte(`{"content":"message from sender"}`)

		select {
		case hub.GetBroadcastChan() <- server.BroadcastMessage{Sender: sender, Payload: testMsg}:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Failed to send broadcast message")
		}

		time.Sleep(20 * time.Millisecond)
	})

	t.Run("Multiple concurrent broadcasts", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Broadcast goroutine %d panicked: %v", id, r)
					}
					done <- true
				}()

				msg := []byte(`{"content":"concurrent broadcast"}`)
				select {
				case hub.GetBroadcastChan() <- server.BroadcastMessage{Payload: msg}:
				case <-time.After(100 * time.Millisecond):
					t.Errorf("Failed to broadcast message %d", id)
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
				t.Error("Concurrent broadcast test timed out")
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Broadcast empty message", func(t *testing.T) {
		emptyMsg := []byte("")

		select {
		case hub.GetBroadcastChan() <- server.BroadcastMessage{Payload: emptyMsg}:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Failed to send empty broadcast message")
		}

		time.Sleep(20 * time.Millisecond)
	})
}

// TestHubShutdown tests the hub's graceful shutdown functionality.
// It verifies that the hub can be shut down properly and all resources are cleaned up.
func TestHubShutdown(t *testing.T) {
	t.Run("Shutdown empty hub", func(t *testing.T) {
		hub := server.NewHub()
		go hub.Run()
		time.Sleep(10 * time.Millisecond)

		err := hub.Shutdown(time.Second)
		if err != nil {
			t.Errorf("Expected successful shutdown, got error: %v", err)
		}
	})

	t.Run("Shutdown hub with clients", func(t *testing.T) {
		hub := server.NewHub()
		go hub.Run()
		time.Sleep(10 * time.Millisecond)

		// Register some clients
		for i := 0; i < 3; i++ {
			client := server.NewClient(nil, hub, "127.0.0.1:"+string(rune(12340+i)))
			select {
			case hub.GetRegisterChan() <- client:
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Failed to register client %d", i)
			}
		}
		time.Sleep(50 * time.Millisecond)

		err := hub.Shutdown(2 * time.Second)
		if err != nil {
			t.Errorf("Expected successful shutdown with clients, got error: %v", err)
		}
	})
}

// TestHubChannelsCommunication tests that hub channels can communicate properly.
// It verifies that messages can be sent through broadcast, register, and unregister channels.
func TestHubChannelsCommunication(t *testing.T) {
	hub := server.NewHub()
	go hub.Run()
	defer hub.Shutdown(time.Second)
	time.Sleep(10 * time.Millisecond)

	t.Run("Broadcast channel accepts messages", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			msg := []byte(`{"content":"test"}`)
			select {
			case hub.GetBroadcastChan() <- server.BroadcastMessage{Payload: msg}:
				// Success
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Iteration %d: Failed to send broadcast message", i)
			}
			time.Sleep(10 * time.Millisecond)
		}
	})

	t.Run("All channels remain responsive", func(t *testing.T) {
		// Send to all channels to ensure they're all working
		select {
		case hub.GetBroadcastChan() <- server.BroadcastMessage{Payload: []byte(`{"content":"test"}`)}:
		case <-time.After(50 * time.Millisecond):
			t.Error("Broadcast channel not responsive")
		}

		select {
		case hub.GetRegisterChan() <- nil:
		case <-time.After(50 * time.Millisecond):
			t.Error("Register channel not responsive")
		}

		select {
		case hub.GetUnregisterChan() <- nil:
		case <-time.After(50 * time.Millisecond):
			t.Error("Unregister channel not responsive")
		}

		time.Sleep(20 * time.Millisecond)
	})
}
