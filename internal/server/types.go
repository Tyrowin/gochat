// Package server defines shared message payload types and utility helpers that
// are reused across client and hub logic.
package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"unicode/utf8"

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

const messagePrefix = `{"content":`

// normalizeMessage validates an inbound frame and re-encodes it into the
// canonical wire format, dropping any fields the protocol does not define.
//
// The result is built with a single allocation sized from the input, which
// avoids the intermediate buffer that encoding/json would otherwise create.
func normalizeMessage(raw []byte) ([]byte, error) {
	if isCanonicalMessage(raw) {
		return bytes.Clone(raw), nil
	}

	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}

	out := make([]byte, 0, len(messagePrefix)+len(msg.Content)+3)
	out = append(out, messagePrefix...)
	out = appendJSONString(out, msg.Content)
	return append(out, '}'), nil
}

// canonicalPrefix and canonicalSuffix bracket a message that is already in the
// exact form normalizeMessage would produce.
const (
	canonicalPrefix = `{"content":"`
	canonicalSuffix = `"}`
)

// isCanonicalMessage reports whether raw is already byte-identical to its own
// normalized form, which lets the caller skip the JSON round trip.
//
// It only answers true when every byte between the quotes is one the encoder
// would copy verbatim: valid UTF-8, no escapes, no control characters, and
// none of the characters encoding/json escapes. Anything else — extra fields,
// whitespace, escapes, non-string content — falls through to the slow path.
func isCanonicalMessage(raw []byte) bool {
	if len(raw) < len(canonicalPrefix)+len(canonicalSuffix) {
		return false
	}

	if !bytes.HasPrefix(raw, []byte(canonicalPrefix)) || !bytes.HasSuffix(raw, []byte(canonicalSuffix)) {
		return false
	}

	body := raw[len(canonicalPrefix) : len(raw)-len(canonicalSuffix)]
	for i := 0; i < len(body); {
		b := body[i]
		if b < utf8.RuneSelf {
			if !jsonSafe[b] {
				return false
			}
			i++
			continue
		}

		r, size := utf8.DecodeRune(body[i:])
		if r == utf8.RuneError && size == 1 {
			return false
		}
		if r == '\u2028' || r == '\u2029' {
			return false
		}
		i += size
	}

	return true
}

const hexDigits = "0123456789abcdef"

// jsonSafe reports, per ASCII byte, whether it can be copied verbatim into a
// JSON string. It mirrors encoding/json, including its HTML escaping of
// '<', '>' and '&'.
var jsonSafe = func() (table [utf8.RuneSelf]bool) {
	for b := range table {
		table[b] = b >= 0x20 && b != '"' && b != '\\' && b != '<' && b != '>' && b != '&'
	}
	return table
}()

// appendJSONString appends s to dst as a quoted JSON string, byte-for-byte
// identical to what [json.Marshal] would produce for the same value.
func appendJSONString(dst []byte, s string) []byte {
	dst = append(dst, '"')

	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if jsonSafe[b] {
				i++
				continue
			}

			dst = append(dst, s[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[b>>4], hexDigits[b&0xF])
			}
			i++
			start = i
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			dst = append(dst, s[start:i]...)
			dst = append(dst, `\ufffd`...)
		case r == '\u2028' || r == '\u2029':
			// Escaped so the payload stays valid inside a JavaScript literal.
			dst = append(dst, s[start:i]...)
			dst = append(dst, '\\', 'u', '2', '0', '2', hexDigits[r&0xF])
		default:
			i += size
			continue
		}
		i += size
		start = i
	}

	dst = append(dst, s[start:]...)
	return append(dst, '"')
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
