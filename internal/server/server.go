// Package server implements the core HTTP and WebSocket server functionality for GoChat.
//
// This package provides a complete real-time chat server implementation using
// WebSockets for bidirectional communication. The server uses a Hub pattern
// to manage client connections and broadcast messages to all connected clients.
//
// # Key Components
//
// The server consists of three main components:
//
//   - Hub: Manages WebSocket client connections and message broadcasting
//   - Client: Represents individual WebSocket connections with read/write pumps
//   - HTTP handlers: Provide WebSocket endpoints and health checks
//
// # Concurrency Safety
//
// The Hub type is safe for concurrent use by multiple goroutines. All client
// map operations are protected by a mutex to prevent race conditions during
// concurrent client registration, unregistration, and message broadcasting.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// isExpectedCloseError checks if an error is expected during connection closure
func isExpectedCloseError(err error) bool {
	if err == nil {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "websocket: close sent") ||
		strings.Contains(errStr, "broken pipe")
}

// Client represents a WebSocket client connection in the chat system.
// It manages the connection state, message sending channel, hub reference,
// and client address information.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	addr   string
	closed bool
}

// Message represents the V1 JSON message format exchanged between clients.
type Message struct {
	Content string `json:"content"`
}

// BroadcastMessage encapsulates a message being broadcast by the hub,
// including the originating client so it can be excluded from delivery.
type BroadcastMessage struct {
	Sender  *Client
	Payload []byte
}

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

// NewClient creates a new Client instance with the provided WebSocket connection,
// hub reference, and client address. The client's send channel is buffered
// to handle message queuing.
func NewClient(conn *websocket.Conn, hub *Hub, addr string) *Client {
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    hub,
		addr:   addr,
		closed: false,
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

// GetSendChan returns the client's send channel for reading outgoing messages.
// This channel is read-only from the caller's perspective.
func (c *Client) GetSendChan() <-chan []byte {
	return c.send
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

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		if err := c.conn.Close(); err != nil {
			if !isExpectedCloseError(err) {
				log.Printf("Error closing connection in readPump: %v", err)
			}
		}
	}()

	if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("Error setting read deadline in pong handler: %v", err)
		}
		return nil
	})

	for {
		_, rawMessage, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error from %s: %v", c.addr, err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			log.Printf("Invalid message from %s: %v", c.addr, err)
			continue
		}

		normalizedMessage, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error normalizing message from %s: %v", c.addr, err)
			continue
		}

		log.Printf("Received message from %s: %s", c.addr, string(normalizedMessage))
		c.hub.broadcast <- BroadcastMessage{Sender: c, Payload: normalizedMessage}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.closeConnection()
	}()

	for c.processWriteEvent(ticker) {
	}
}

// processWriteEvent waits for the next write event and returns false when the
// pump should stop processing.
func (c *Client) processWriteEvent(ticker *time.Ticker) bool {
	select {
	case message, ok := <-c.send:
		return c.handleMessage(message, ok)
	case <-ticker.C:
		return c.handlePing()
	}
}

// closeConnection safely closes the WebSocket connection with proper error handling
func (c *Client) closeConnection() {
	if err := c.conn.Close(); err != nil {
		// Only log unexpected connection close errors
		if !isExpectedCloseError(err) {
			log.Printf("Error closing connection in writePump: %v", err)
		}
	}
}

// handleMessage processes outgoing messages and returns false if the connection should be closed
func (c *Client) handleMessage(message []byte, ok bool) bool {
	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		log.Printf("Error setting write deadline: %v", err)
		return false
	}

	if !ok {
		return c.writeCloseMessage()
	}

	return c.writeTextMessage(message)
}

// writeCloseMessage sends a close message to the client
func (c *Client) writeCloseMessage() bool {
	if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
		if !isExpectedCloseError(err) {
			log.Printf("Error writing close message: %v", err)
		}
	}
	return false
}

// writeTextMessage writes a text message and any queued messages
func (c *Client) writeTextMessage(message []byte) bool {
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return false
	}

	if !c.writeMessageContent(w, message) {
		return false
	}

	if !c.writeQueuedMessages(w) {
		return false
	}

	return c.closeWriter(w)
}

// writeMessageContent writes the main message content
func (c *Client) writeMessageContent(w io.WriteCloser, message []byte) bool {
	if _, err := w.Write(message); err != nil {
		log.Printf("Error writing message: %v", err)
		return false
	}
	return true
}

// writeQueuedMessages writes any additional queued messages
func (c *Client) writeQueuedMessages(w io.WriteCloser) bool {
	n := len(c.send)
	for i := 0; i < n; i++ {
		if !c.writeQueuedMessage(w) {
			return false
		}
	}
	return true
}

// writeQueuedMessage writes a single queued message with newline separator
func (c *Client) writeQueuedMessage(w io.WriteCloser) bool {
	if _, err := w.Write([]byte{'\n'}); err != nil {
		log.Printf("Error writing newline: %v", err)
		return false
	}
	if _, err := w.Write(<-c.send); err != nil {
		log.Printf("Error writing queued message: %v", err)
		return false
	}
	return true
}

// closeWriter closes the message writer
func (c *Client) closeWriter(w io.WriteCloser) bool {
	return w.Close() == nil
}

// handlePing sends a ping message to keep the connection alive
func (c *Client) handlePing() bool {
	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		log.Printf("Error setting write deadline for ping: %v", err)
		return false
	}
	return c.conn.WriteMessage(websocket.PingMessage, nil) == nil
}

var hub = NewHub()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

// WebSocketHandler handles WebSocket upgrade requests and manages client connections.
// It validates that the request uses the GET method, upgrades the HTTP connection
// to WebSocket, creates a new Client instance, and starts the client's read/write pumps.
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed. WebSocket endpoint only accepts GET requests.", http.StatusMethodNotAllowed)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := NewClient(conn, hub, r.RemoteAddr)

	// Register the client with the hub; the hub will launch the pump goroutines.
	client.hub.register <- client
}

// HealthHandler provides a simple health check endpoint that returns server status.
// It responds with a plain text message indicating the server is running.
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "GoChat server is running!")
}

// TestPageHandler serves an HTML test page for testing WebSocket functionality.
// It provides a simple web interface to connect to the WebSocket endpoint,
// send messages, and view real-time chat communication.
func TestPageHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `<!DOCTYPE html>
<html>
<head>
    <title>GoChat WebSocket Test</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        #messages { 
            border: 1px solid #ccc; 
            height: 300px; 
            padding: 10px; 
            overflow-y: scroll; 
            margin: 10px 0;
            background-color: #f9f9f9;
        }
        input[type="text"] { 
            width: 300px; 
            padding: 5px; 
            margin-right: 10px;
        }
        button { 
            padding: 5px 15px; 
            background-color: #007cba; 
            color: white; 
            border: none; 
            cursor: pointer;
        }
        button:hover { background-color: #005a87; }
        .status { 
            margin: 10px 0; 
            padding: 5px; 
            border-radius: 3px;
        }
        .connected { background-color: #d4edda; color: #155724; }
        .disconnected { background-color: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <h1>GoChat WebSocket Test</h1>
    
    <div id="status" class="status disconnected">Disconnected</div>
    
    <div>
        <input type="text" id="messageInput" placeholder="Type a message..." disabled>
        <button id="sendButton" onclick="sendMessage()" disabled>Send</button>
        <button id="connectButton" onclick="toggleConnection()">Connect</button>
    </div>
    
    <div id="messages"></div>

    <script>
        let ws = null;
        const messagesDiv = document.getElementById('messages');
        const messageInput = document.getElementById('messageInput');
        const sendButton = document.getElementById('sendButton');
        const connectButton = document.getElementById('connectButton');
        const statusDiv = document.getElementById('status');

        function addMessage(message, type = 'info') {
            const messageElement = document.createElement('div');
            messageElement.style.margin = '5px 0';
            messageElement.style.padding = '3px';
            
            if (type === 'sent') {
                messageElement.style.color = 'blue';
                messageElement.innerHTML = '<strong>You:</strong> ' + message;
            } else if (type === 'received') {
                messageElement.style.color = 'green';
                messageElement.innerHTML = '<strong>Other:</strong> ' + message;
            } else {
                messageElement.style.color = 'gray';
                messageElement.innerHTML = '<em>' + message + '</em>';
            }
            
            messagesDiv.appendChild(messageElement);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function updateStatus(connected) {
            if (connected) {
                statusDiv.textContent = 'Connected';
                statusDiv.className = 'status connected';
                messageInput.disabled = false;
                sendButton.disabled = false;
                connectButton.textContent = 'Disconnect';
            } else {
                statusDiv.textContent = 'Disconnected';
                statusDiv.className = 'status disconnected';
                messageInput.disabled = true;
                sendButton.disabled = true;
                connectButton.textContent = 'Connect';
            }
        }

        function connect() {
            ws = new WebSocket('ws://localhost:8080/ws');
            
            ws.onopen = function(event) {
                addMessage('Connected to GoChat server');
                updateStatus(true);
            };
            
            ws.onmessage = function(event) {
                addMessage(event.data, 'received');
            };
            
            ws.onclose = function(event) {
                addMessage('Connection closed');
                updateStatus(false);
                ws = null;
            };
            
            ws.onerror = function(error) {
                addMessage('Connection error: ' + error);
                updateStatus(false);
            };
        }

        function disconnect() {
            if (ws) {
                ws.close();
            }
        }

        function toggleConnection() {
            if (ws && ws.readyState === WebSocket.OPEN) {
                disconnect();
            } else {
                connect();
            }
        }

        function sendMessage() {
            const message = messageInput.value.trim();
            if (message && ws && ws.readyState === WebSocket.OPEN) {
                ws.send(message);
                addMessage(message, 'sent');
                messageInput.value = '';
            }
        }

        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
    </script>
</body>
</html>`
	if _, err := fmt.Fprint(w, html); err != nil {
		log.Printf("Error writing HTML response: %v", err)
	}
}

// SetupRoutes configures and returns an HTTP ServeMux with all application routes.
// It sets up handlers for health check, WebSocket endpoint, and test page.
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HealthHandler)
	mux.HandleFunc("/ws", WebSocketHandler)
	mux.HandleFunc("/test", TestPageHandler)
	return mux
}

// CreateServer creates and configures an HTTP server with the specified port and handler.
// It sets reasonable timeout values for production use.
func CreateServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Config holds the server configuration settings.
// Currently it only contains the port configuration.
type Config struct {
	Port string
}

// NewConfig creates a new Config instance with default values.
// The default port is set to :8080.
func NewConfig() *Config {
	return &Config{
		Port: ":8080",
	}
}

// StartHub initializes and starts the global hub in a separate goroutine.
// This should be called before starting the HTTP server.
func StartHub() {
	go hub.Run()
	log.Println("Hub started and ready to manage WebSocket connections")
}

// StartServer starts the HTTP server and begins listening for connections.
// It returns an error if the server fails to start.
func StartServer(server *http.Server) error {
	fmt.Printf("Server listening on port %s\n", server.Addr)
	return server.ListenAndServe()
}
