// Package server classifies connection teardown errors so the client and hub
// can tell an ordinary disconnect from a fault worth logging.
package server

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/gorilla/websocket"
)

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
