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
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Hub
	addr string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func NewClient(conn *websocket.Conn, hub *Hub, addr string) *Client {
	return &Client{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  hub,
		addr: addr,
	}
}

func (h *Hub) GetRegisterChan() chan<- *Client {
	return h.register
}

func (h *Hub) GetUnregisterChan() chan<- *Client {
	return h.unregister
}

func (h *Hub) GetBroadcastChan() chan<- []byte {
	return h.broadcast
}

func (c *Client) GetSendChan() <-chan []byte {
	return c.send
}

func (h *Hub) safeSend(client *Client, message []byte) bool {
	defer func() {
		recover()
	}()

	// Check if client is still registered before sending
	h.mutex.RLock()
	_, exists := h.clients[client]
	h.mutex.RUnlock()

	if !exists {
		return false
	}

	select {
	case client.send <- message:
		return true
	default:
		return false
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mutex.Unlock()
			log.Printf("Client registered from %s. Total clients: %d", client.addr, clientCount)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				clientCount := len(h.clients)
				// Close the channel while holding the lock to prevent race conditions
				close(client.send)
				h.mutex.Unlock()
				log.Printf("Client unregistered from %s. Total clients: %d", client.addr, clientCount)
			} else {
				h.mutex.Unlock()
			}

		case message := <-h.broadcast:
			h.mutex.RLock()
			clientCount := len(h.clients)
			clients := make([]*Client, 0, clientCount)
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mutex.RUnlock()

			log.Printf("Broadcasting message to %d clients", clientCount)

			var clientsToRemove []*Client

			for _, client := range clients {
				if !h.safeSend(client, message) {
					clientsToRemove = append(clientsToRemove, client)
				}
			}

			if len(clientsToRemove) > 0 {
				h.mutex.Lock()
				for _, client := range clientsToRemove {
					if _, exists := h.clients[client]; exists {
						delete(h.clients, client)
						close(client.send)
						log.Printf("Client from %s removed due to full send buffer", client.addr)
					}
				}
				h.mutex.Unlock()
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error from %s: %v", c.addr, err)
			}
			break
		}

		log.Printf("Received message from %s: %s", c.addr, string(message))
		c.hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var hub = NewHub()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
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

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  hub,
		addr: r.RemoteAddr,
	}

	client.hub.register <- client
	go client.writePump()
	client.readPump()
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
	fmt.Fprint(w, html)
}

func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HealthHandler)
	mux.HandleFunc("/ws", WebSocketHandler)
	mux.HandleFunc("/test", TestPageHandler)
	return mux
}

func CreateServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

type Config struct {
	Port string
}

func NewConfig() *Config {
	return &Config{
		Port: ":8080",
	}
}

func StartHub() {
	go hub.Run()
	log.Println("Hub started and ready to manage WebSocket connections")
}

func StartServer(server *http.Server) error {
	fmt.Printf("Server listening on port %s\n", server.Addr)
	return server.ListenAndServe()
}
