package unit

import (
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
)

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

func TestHubRunStartsWithoutPanic(t *testing.T) {
	hub := server.NewHub()

	// Start the hub in a goroutine and ensure it doesn't panic
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Hub.Run() panicked: %v", r)
			}
			done <- true
		}()
		// Run for a short time
		go hub.Run()
		time.Sleep(10 * time.Millisecond)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("Hub.Run() test timed out")
	}
}

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

func TestNewClient(t *testing.T) {
	hub := server.NewHub()

	client := server.NewClient(nil, hub, "127.0.0.1:12345")

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Test that we can get the send channel
	sendChan := client.GetSendChan()
	if sendChan == nil {
		t.Error("Client send channel is nil")
	}
}

// TestClientSendChannel tests the client's send channel functionality
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
