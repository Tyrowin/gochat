// Package server manages individual WebSocket clients, handling read/write
// pumps, rate limiting, and lifecycle control for each connection.
package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection in the chat system.
// It manages the connection state, message sending channel, hub reference,
// and client address information.
type Client struct {
	conn           *websocket.Conn
	send           chan []byte
	hub            *Hub
	addr           string
	closed         bool
	maxMessageSize int64
	rateLimiter    *rateLimiter
	rateLimit      RateLimitConfig
}

// NewClient creates a new Client instance with the provided WebSocket connection,
// hub reference, and client address. The client's send channel is buffered
// to handle message queuing.
func NewClient(conn *websocket.Conn, hub *Hub, addr string) *Client {
	cfg := currentConfig()
	if conn != nil {
		conn.SetReadLimit(cfg.MaxMessageSize)
	}
	limiter := newRateLimiter(cfg.RateLimit.Burst, cfg.RateLimit.RefillInterval)

	return &Client{
		conn:           conn,
		send:           make(chan []byte, 256),
		hub:            hub,
		addr:           addr,
		closed:         false,
		maxMessageSize: cfg.MaxMessageSize,
		rateLimiter:    limiter,
		rateLimit:      cfg.RateLimit,
	}
}

// GetSendChan returns the client's send channel for reading outgoing messages.
// This channel is read-only from the caller's perspective.
func (c *Client) GetSendChan() <-chan []byte {
	return c.send
}

// setupReadConnection configures read deadlines and pong handler for the WebSocket connection
func (c *Client) setupReadConnection() {
	if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Error setting initial read deadline for %s: %v", c.addr, err)
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("Error setting read deadline in pong handler for %s: %v", c.addr, err)
		}
		return nil
	})
}

// handleReadError logs appropriate error messages based on the error type
// and returns true if the read loop should break
func (c *Client) handleReadError(err error) bool {
	if err == nil {
		return false
	}

	// Check for rate limit violations
	if errors.Is(err, websocket.ErrReadLimit) {
		log.Printf("Message from %s exceeded maximum size of %d bytes", c.addr, c.maxMessageSize)
		return true
	}

	// Check for expected close scenarios
	if websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure) {
		log.Printf("Client %s disconnected: %v", c.addr, err)
		return true
	}

	// Check for network errors
	if errors.Is(err, io.EOF) || isExpectedCloseError(err) {
		log.Printf("Client %s connection closed: %v", c.addr, err)
		return true
	}

	// Log unexpected errors with more context
	if websocket.IsUnexpectedCloseError(err,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure,
		websocket.CloseMessageTooBig) {
		log.Printf("Unexpected WebSocket error from %s: %v", c.addr, err)
		return true
	}

	// Generic error case
	log.Printf("WebSocket read error from %s: %v", c.addr, err)
	return true
}

// checkRateLimit verifies if the client has exceeded rate limits
// and returns true if the message should be processed
func (c *Client) checkRateLimit() bool {
	if c.rateLimiter != nil && !c.rateLimiter.allow() {
		log.Printf("Rate limit exceeded for %s (%d messages per %s); discarding message", c.addr, c.rateLimit.Burst, c.rateLimit.RefillInterval)
		return false
	}
	return true
}

// processMessage unmarshals, normalizes, and broadcasts a raw message
// and returns true if the message was processed successfully
func (c *Client) processMessage(rawMessage []byte) bool {
	var msg Message
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		log.Printf("Invalid message from %s: %v", c.addr, err)
		return false
	}

	normalizedMessage, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error normalizing message from %s: %v", c.addr, err)
		return false
	}

	log.Printf("Received message from %s: %s", c.addr, string(normalizedMessage))
	c.hub.broadcast <- BroadcastMessage{Sender: c, Payload: normalizedMessage}
	return true
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

	c.setupReadConnection()

	for {
		_, rawMessage, err := c.conn.ReadMessage()
		if err != nil {
			if c.handleReadError(err) {
				break
			}
		}

		if !c.checkRateLimit() {
			continue
		}

		c.processMessage(rawMessage)
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
		log.Printf("Error setting write deadline for %s: %v", c.addr, err)
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
			log.Printf("Error writing close message to %s: %v", c.addr, err)
		}
	}
	return false
}

// writeTextMessage writes a text message and any queued messages
func (c *Client) writeTextMessage(message []byte) bool {
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		log.Printf("Error creating writer for %s: %v", c.addr, err)
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
		log.Printf("Error writing message to %s: %v", c.addr, err)
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
		log.Printf("Error writing newline to %s: %v", c.addr, err)
		return false
	}
	if _, err := w.Write(<-c.send); err != nil {
		log.Printf("Error writing queued message to %s: %v", c.addr, err)
		return false
	}
	return true
}

// closeWriter closes the message writer
func (c *Client) closeWriter(w io.WriteCloser) bool {
	if err := w.Close(); err != nil {
		log.Printf("Error closing writer for %s: %v", c.addr, err)
		return false
	}
	return true
}

// handlePing sends a ping message to keep the connection alive
func (c *Client) handlePing() bool {
	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		log.Printf("Error setting write deadline for ping to %s: %v", c.addr, err)
		return false
	}
	if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		log.Printf("Error writing ping message to %s: %v", c.addr, err)
		return false
	}
	return true
}
