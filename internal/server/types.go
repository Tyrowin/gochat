// Package server defines shared message payload types and the connection-error
// helpers reused across client and hub logic.
package server

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/gorilla/websocket"
)

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

// isExpectedCloseError reports whether an error is part of normal connection
// teardown rather than a fault worth logging.
func isExpectedCloseError(err error) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, net.ErrClosed) ||
		errors.Is(err, websocket.ErrCloseSent) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) {
		return true
	}

	// Broken pipes and connection resets surface as platform-specific syscall
	// errors that are not worth enumerating per GOOS.
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer")
}
