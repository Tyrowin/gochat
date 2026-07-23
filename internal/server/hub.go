// Package server coordinates client registration, message broadcast, and
// connection cleanup for the Blip WebSocket system via the Hub type.
package server

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Hub manages all WebSocket client connections and handles message broadcasting.
//
// The clients map is owned exclusively by the [Hub.Run] goroutine: registration,
// unregistration, broadcast fan-out, and shutdown all happen there. That single
// ownership removes lock traffic from the broadcast path entirely, so every
// mutation of clients must be reached through one of the hub's channels.
type Hub struct {
	clients    map[*Client]struct{}
	broadcast  chan BroadcastMessage
	register   chan *Client
	unregister chan *Client

	// failed is scratch space reused across broadcasts to avoid allocating a
	// slice per message. Only Run touches it.
	failed []*Client

	wg         sync.WaitGroup
	shutdown   chan struct{}
	shutdownMu sync.Once
	done       chan struct{}
	stateMu    sync.Mutex
	started    bool
}

// NewHub creates and initializes a new Hub instance with all necessary channels
// and client map. The returned Hub is ready to manage WebSocket connections.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		shutdown:   make(chan struct{}),
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

// Start launches the hub event loop in a goroutine if it is not already running.
func (h *Hub) Start() {
	go h.Run()
}

// IsStopped reports whether the hub event loop has exited.
func (h *Hub) IsStopped() bool {
	select {
	case <-h.done:
		return true
	default:
		return false
	}
}

func (h *Hub) markStarted() bool {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	if h.started {
		return false
	}

	h.started = true
	return true
}

func (h *Hub) hasStarted() bool {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	return h.started
}

// Run starts the hub's main event loop, handling client registration,
// unregistration, and message broadcasting. It runs until shutdown is signalled
// and should be called in its own goroutine.
func (h *Hub) Run() {
	if !h.markStarted() {
		return
	}

	defer close(h.done)

	for {
		select {
		case <-h.shutdown:
			h.shutdownClients()
			return

		case client := <-h.register:
			h.addClient(client)

		case client := <-h.unregister:
			h.removeClient(client)

		case broadcastMsg := <-h.broadcast:
			h.handleBroadcast(broadcastMsg)
		}
	}
}

// addClient registers a client and starts its read and write pumps.
func (h *Hub) addClient(client *Client) {
	if client == nil {
		log().Warn("received nil client registration; skipping")
		return
	}

	h.clients[client] = struct{}{}
	log().Info("client registered", "addr", client.addr, "total_clients", len(h.clients))

	h.wg.Add(2)
	go func() {
		defer h.wg.Done()
		client.writePump()
	}()
	go func() {
		defer h.wg.Done()
		client.readPump()
	}()
}

// removeClient unregisters a client and closes its send channel. It is a no-op
// for clients that are already gone, which makes double unregistration safe.
func (h *Hub) removeClient(client *Client) {
	if client == nil {
		log().Warn("received nil client unregistration; skipping")
		return
	}

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	close(client.send)
	log().Info("client unregistered", "addr", client.addr, "total_clients", len(h.clients))
}

// handleBroadcast fans a message out to every client except the sender.
//
// Delivery is non-blocking: a client whose buffer is full is dropped rather
// than allowed to stall the hub for everyone else.
func (h *Hub) handleBroadcast(broadcastMsg BroadcastMessage) {
	h.failed = h.failed[:0]
	delivered := 0

	for client := range h.clients {
		if client == broadcastMsg.Sender {
			continue
		}

		select {
		case client.send <- broadcastMsg.Payload:
			delivered++
		default:
			h.failed = append(h.failed, client)
		}
	}

	dropped := len(h.failed)
	for _, client := range h.failed {
		log().Warn("dropping client with a full send buffer", "addr", client.addr)
		h.removeClient(client)
	}
	clear(h.failed)

	if debugEnabled() {
		log().Debug("broadcast complete", "delivered", delivered, "dropped", dropped)
	}
}

// shutdownClients closes every active client connection, which unblocks the
// read pumps so they can exit.
func (h *Hub) shutdownClients() {
	for client := range h.clients {
		client.closeConnection()
	}

	log().Info("closed client connections", "count", len(h.clients))
	clear(h.clients)
}

// Shutdown initiates graceful shutdown of the hub and waits for all goroutines to complete.
// It returns after all client connections are closed and goroutines have finished,
// or when the timeout is reached.
func (h *Hub) Shutdown(timeout time.Duration) error {
	if !h.hasStarted() {
		return nil
	}

	log().Info("initiating hub shutdown")

	h.shutdownMu.Do(func() {
		close(h.shutdown)
	})

	runLoopTimer := time.NewTimer(timeout)
	defer runLoopTimer.Stop()

	select {
	case <-h.done:
	case <-runLoopTimer.C:
		log().Error("hub shutdown timed out waiting for the event loop")
		return fmt.Errorf("hub event loop shutdown timed out: %w", context.DeadlineExceeded)
	}

	clientsDone := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(clientsDone)
	}()

	clientTimer := time.NewTimer(timeout)
	defer clientTimer.Stop()

	select {
	case <-clientsDone:
		log().Info("hub shutdown completed")
		return nil
	case <-clientTimer.C:
		log().Error("hub shutdown timed out; some client goroutines may still be running")
		return fmt.Errorf("hub client shutdown timed out: %w", context.DeadlineExceeded)
	}
}
