// Package server defines the message payload types exchanged over the wire and
// passed between the client pumps and the hub.
package server

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
