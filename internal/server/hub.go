// Package server coordinates client registration, message broadcast, and
// connection cleanup for the GoChat WebSocket system via the Hub type.
package server

import (
	"context"
	"log"
	"sync"
	"time"
)

// Hub manages all WebSocket client connections and handles message broadcasting.
// It maintains client registration/unregistration and ensures thread-safe operations
// through mutex protection.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan BroadcastMessage
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewHub creates and initializes a new Hub instance with all necessary channels
// and client map. The returned Hub is ready to manage WebSocket connections.
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}
}

// GetRegisterChan returns the channel used for registering new clients to the hub.
// This channel is write-only from the caller's perspective.
func (h *Hub) GetRegisterChan() chan<- *Client {
	return h.register
}

// GetUnregisterChan returns the channel used for unregistering clients from the hub.
// This channel is write-only from the caller's perspective.
func (h *Hub) GetUnregisterChan() chan<- *Client {
	return h.unregister
}

// GetBroadcastChan returns the channel used for broadcasting messages to all clients.
// This channel is write-only from the caller's perspective.
func (h *Hub) GetBroadcastChan() chan<- BroadcastMessage {
	return h.broadcast
}

func (h *Hub) safeSend(client *Client, message []byte) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in safeSend: %v", r)
		}
	}()

	// Hold the lock during the entire send operation to prevent race conditions
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Check if client is still registered and not closed
	_, exists := h.clients[client]
	if !exists || client.closed {
		return false
	}

	// Try to send the message (channel might be closed, so we need to recover from panic)
	select {
	case client.send <- message:
		return true
	default:
		return false
	}
}

// Run starts the hub's main event loop, handling client registration, unregistration,
// and message broadcasting. This method should be called in a separate goroutine
// as it runs indefinitely.
func (h *Hub) Run() {
	defer close(h.done)

	for {
		select {
		case <-h.ctx.Done():
			h.shutdownClients()
			return

		case client := <-h.register:
			if client == nil {
				log.Printf("Received nil client registration; skipping")
				continue
			}

			h.mutex.Lock()
			client.closed = false
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mutex.Unlock()
			log.Printf("Client registered from %s. Total clients: %d", client.addr, clientCount)

			h.wg.Add(2)
			go func() {
				defer h.wg.Done()
				client.writePump()
			}()
			go func() {
				defer h.wg.Done()
				client.readPump()
			}()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.closed = true
				clientCount := len(h.clients)
				h.mutex.Unlock()
				// Close the channel after releasing the lock
				close(client.send)
				log.Printf("Client unregistered from %s. Total clients: %d", client.addr, clientCount)
			} else {
				h.mutex.Unlock()
			}

		case broadcastMsg := <-h.broadcast:
			h.handleBroadcast(broadcastMsg)
		}
	}
}

var hub = NewHub()

// handleBroadcast processes a broadcast message and sends it to all clients except the sender
func (h *Hub) handleBroadcast(broadcastMsg BroadcastMessage) {
	clients := h.getClientSnapshot()
	targetCount := h.calculateTargetCount(len(clients), broadcastMsg.Sender)

	log.Printf("Broadcasting message to %d clients", targetCount)

	clientsToRemove := h.broadcastToClients(clients, broadcastMsg)
	h.removeFailedClients(clientsToRemove)
}

// getClientSnapshot returns a thread-safe snapshot of all current clients
func (h *Hub) getClientSnapshot() []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

// calculateTargetCount determines how many clients will receive the broadcast
func (h *Hub) calculateTargetCount(clientCount int, sender *Client) int {
	targetCount := clientCount
	if sender != nil {
		targetCount--
	}
	if targetCount < 0 {
		targetCount = 0
	}
	return targetCount
}

// broadcastToClients sends the message to all clients except the sender and returns failed clients
func (h *Hub) broadcastToClients(clients []*Client, broadcastMsg BroadcastMessage) []*Client {
	var clientsToRemove []*Client

	for _, client := range clients {
		if broadcastMsg.Sender != nil && client == broadcastMsg.Sender {
			continue
		}
		if !h.safeSend(client, broadcastMsg.Payload) {
			clientsToRemove = append(clientsToRemove, client)
		}
	}

	return clientsToRemove
}

// removeFailedClients removes clients that failed to receive messages and closes their channels
func (h *Hub) removeFailedClients(clientsToRemove []*Client) {
	if len(clientsToRemove) == 0 {
		return
	}

	h.mutex.Lock()
	var channelsToClose []chan []byte
	for _, client := range clientsToRemove {
		if _, exists := h.clients[client]; exists {
			delete(h.clients, client)
			client.closed = true
			channelsToClose = append(channelsToClose, client.send)
			log.Printf("Client from %s removed due to full send buffer", client.addr)
		}
	}
	h.mutex.Unlock()

	// Close channels after releasing the lock
	for _, ch := range channelsToClose {
		close(ch)
	}
}

// shutdownClients gracefully closes all active client connections
func (h *Hub) shutdownClients() {
	log.Println("Shutting down all client connections...")

	h.mutex.Lock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mutex.Unlock()

	// Close all client connections
	for _, client := range clients {
		if client.conn != nil {
			if err := client.conn.Close(); err != nil {
				if !isExpectedCloseError(err) {
					log.Printf("Error closing client connection from %s: %v", client.addr, err)
				}
			}
		}
	}

	log.Printf("Closed %d client connections", len(clients))
}

// Shutdown initiates graceful shutdown of the hub and waits for all goroutines to complete.
// It returns after all client connections are closed and goroutines have finished,
// or when the timeout is reached.
func (h *Hub) Shutdown(timeout time.Duration) error {
	log.Println("Initiating hub shutdown...")

	// Signal shutdown
	h.cancel()

	// Wait for Run() to complete
	<-h.done

	// Wait for all client goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Hub shutdown completed successfully")
		return nil
	case <-time.After(timeout):
		log.Println("Hub shutdown timeout reached, some goroutines may still be running")
		return context.DeadlineExceeded
	}
}
