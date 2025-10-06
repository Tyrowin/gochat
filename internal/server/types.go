// Package server defines shared message payload types and utility helpers that
// are reused across client and hub logic.
package server

import "strings"

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

// isExpectedCloseError checks if an error is expected during connection closure.
func isExpectedCloseError(err error) bool {
	if err == nil {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "websocket: close sent") ||
		strings.Contains(errStr, "broken pipe")
}
