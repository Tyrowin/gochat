// Package server defines the message payload types exchanged over the wire and
// passed between the client pumps and the hub.
package server

// Message represents the V1 JSON message format exchanged between clients.
type Message struct {
	Content string `json:"content"`
}

// BroadcastMessage encapsulates a message being broadcast by the hub,
// including the originating client so it can be excluded from delivery.
//
// Sender is the hub's own view of a client rather than a [Client], because the
// hub compares it against the clients it holds. A caller outside the package
// either leaves it nil or passes a [Client], which satisfies that view.
type BroadcastMessage struct {
	Sender  clientConn
	Payload []byte
}
