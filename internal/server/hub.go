// Package server coordinates client registration, message broadcast, and
// connection cleanup for the Blip WebSocket system via the Hub type.
package server

import (
	"context"
	"fmt"
	"sync"
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

	// countReq carries a reply channel into the Run goroutine so callers can
	// read the client count without touching the map themselves.
	countReq chan chan int

	wg           sync.WaitGroup
	shutdown     chan struct{}
	shutdownOnce sync.Once
	done         chan struct{}
	stateMu      sync.Mutex
	started      bool
}

// NewHub creates and initializes a new Hub instance with all necessary channels
// and client map. The returned Hub is ready to manage WebSocket connections.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		countReq:   make(chan chan int),
		shutdown:   make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// RegisterChan returns the channel used for registering new clients to the hub.
// This channel is write-only from the caller's perspective.
func (h *Hub) RegisterChan() chan<- *Client {
	return h.register
}

// UnregisterChan returns the channel used for unregistering clients from the hub.
// This channel is write-only from the caller's perspective.
func (h *Hub) UnregisterChan() chan<- *Client {
	return h.unregister
}

// BroadcastChan returns the channel used for broadcasting messages to all clients.
// This channel is write-only from the caller's perspective.
func (h *Hub) BroadcastChan() chan<- BroadcastMessage {
	return h.broadcast
}

// ClientCount reports how many clients are currently registered. The count is
// answered by the Run goroutine, which owns the map, so it is consistent with
// every registration and broadcast the hub has already processed. It returns 0
// once the hub has stopped.
func (h *Hub) ClientCount() int {
	reply := make(chan int, 1)

	select {
	case h.countReq <- reply:
		return <-reply
	case <-h.done:
		return 0
	}
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

		case reply := <-h.countReq:
			reply <- len(h.clients)
		}
	}
}

// addClient registers a client and starts its read and write pumps.
func (h *Hub) addClient(client *Client) {
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

// Shutdown initiates graceful shutdown of the hub and waits for all goroutines
// to complete. It returns after all client connections are closed and goroutines
// have finished, or when ctx is done — whichever comes first. Both stages share
// the one deadline carried by ctx.
func (h *Hub) Shutdown(ctx context.Context) error {
	if !h.hasStarted() {
		return nil
	}

	log().Info("initiating hub shutdown")

	h.shutdownOnce.Do(func() {
		close(h.shutdown)
	})

	if err := awaitStage(ctx, h.done, "event loop"); err != nil {
		return err
	}

	clientsDone := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(clientsDone)
	}()

	if err := awaitStage(ctx, clientsDone, "client"); err != nil {
		return err
	}

	log().Info("hub shutdown completed")
	return nil
}

// awaitStage blocks until done closes or ctx is cancelled, naming the stage in
// the log line and the returned error.
func awaitStage(ctx context.Context, done <-chan struct{}, stage string) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		log().Error("hub shutdown timed out", "stage", stage)
		return fmt.Errorf("hub %s shutdown timed out: %w", stage, ctx.Err())
	}
}
