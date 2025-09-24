// Package server coordinates client registration, message broadcast, and
// connection cleanup for the GoChat WebSocket system via the Hub type.
package server

import (
	"log"
	"sync"
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
}

// NewHub creates and initializes a new Hub instance with all necessary channels
// and client map. The returned Hub is ready to manage WebSocket connections.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
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
	for {
		select {
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

			go client.writePump()
			go client.readPump()

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
			h.mutex.RLock()
			clientCount := len(h.clients)
			clients := make([]*Client, 0, clientCount)
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mutex.RUnlock()

			targetCount := clientCount
			if broadcastMsg.Sender != nil {
				targetCount--
			}
			if targetCount < 0 {
				targetCount = 0
			}

			log.Printf("Broadcasting message to %d clients", targetCount)

			var clientsToRemove []*Client

			for _, client := range clients {
				if broadcastMsg.Sender != nil && client == broadcastMsg.Sender {
					continue
				}
				if !h.safeSend(client, broadcastMsg.Payload) {
					clientsToRemove = append(clientsToRemove, client)
				}
			}

			if len(clientsToRemove) > 0 {
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
		}
	}
}

var hub = NewHub()
