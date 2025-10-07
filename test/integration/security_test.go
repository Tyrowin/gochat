// Package integration contains security-focused integration tests.
//
// These tests verify that the security constraints are properly enforced,
// including origin validation, message size limits, and rate limiting.
package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tyrowin/gochat/internal/server"
	"github.com/gorilla/websocket"
)

// TestOriginValidationEdgeCases tests various edge cases for origin validation.
func TestOriginValidationEdgeCases(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Missing Origin header", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{testServer.URL}
		})

		header := http.Header{}
		// No Origin header set
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			_ = conn.Close()
			_ = resp.Body.Close()
			t.Fatal("Expected connection to fail with missing origin")
		}
		if resp != nil {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("Expected status %d, got %d", http.StatusForbidden, resp.StatusCode)
			}
		}
	})

	t.Run("Empty Origin header", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{testServer.URL}
		})

		header := http.Header{}
		header.Set("Origin", "")
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			_ = conn.Close()
			_ = resp.Body.Close()
			t.Fatal("Expected connection to fail with empty origin")
		}
		if resp != nil {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("Expected status %d, got %d", http.StatusForbidden, resp.StatusCode)
			}
		}
	})

	t.Run("Malformed Origin URL", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{testServer.URL}
		})

		malformedOrigins := []string{
			"not-a-url",
			"://missing-scheme",
			"http://",
			"ftp://unsupported-scheme.com",
			"javascript:alert(1)",
		}

		for _, origin := range malformedOrigins {
			header := http.Header{}
			header.Set("Origin", origin)
			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
			if err == nil {
				_ = conn.Close()
				_ = resp.Body.Close()
				t.Errorf("Expected connection to fail with malformed origin %q", origin)
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	})

	t.Run("Case sensitivity in origin matching", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{"http://example.com"}
		})

		// These should all be normalized to lowercase and match
		caseVariations := []string{
			"http://EXAMPLE.COM",
			"http://Example.Com",
			"HTTP://example.com",
		}

		for _, origin := range caseVariations {
			header := http.Header{}
			header.Set("Origin", origin)
			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
			if err != nil {
				t.Errorf("Expected origin %q to be allowed (case-insensitive): %v", origin, err)
			} else {
				_ = conn.Close()
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	})

	t.Run("Wildcard origin configuration", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{"*"}
		})

		// Any origin should be allowed
		testOrigins := []string{
			"http://example.com",
			"https://another.com",
			"http://localhost:3000",
		}

		for _, origin := range testOrigins {
			header := http.Header{}
			header.Set("Origin", origin)
			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
			if err != nil {
				t.Errorf("Expected origin %q to be allowed with wildcard: %v", origin, err)
			} else {
				_ = conn.Close()
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	})

	t.Run("Origin with different port", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{"http://localhost:8080"}
		})

		// Same host but different port should be rejected
		header := http.Header{}
		header.Set("Origin", "http://localhost:9090")
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			_ = conn.Close()
			_ = resp.Body.Close()
			t.Fatal("Expected connection to fail with different port")
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	})

	t.Run("Origin with path component ignored", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{"http://example.com"}
		})

		// Path in origin should be ignored during normalization
		header := http.Header{}
		header.Set("Origin", "http://example.com/some/path")
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			t.Errorf("Expected origin with path to be allowed: %v", err)
		} else {
			_ = conn.Close()
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	})

	t.Run("HTTP vs HTTPS scheme difference", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{"http://example.com"}
		})

		// HTTPS should not match HTTP
		header := http.Header{}
		header.Set("Origin", "https://example.com")
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			_ = conn.Close()
			_ = resp.Body.Close()
			t.Fatal("Expected HTTPS origin to be rejected when only HTTP is allowed")
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	})
}

// TestMessageSizeLimitEdgeCases tests various edge cases for message size validation.
func TestMessageSizeLimitEdgeCases(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Message exactly at size limit", func(t *testing.T) {
		const limit int64 = 100
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.MaxMessageSize = limit
		})

		sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect sender: %v", err)
		}
		defer func() { _ = sender.Close() }()
		defer func() { _ = senderResp.Body.Close() }()

		receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect receiver: %v", err)
		}
		defer func() { _ = receiver.Close() }()
		defer func() { _ = receiverResp.Body.Close() }()

		time.Sleep(50 * time.Millisecond)

		// Create a message that's exactly at the limit
		// JSON overhead: {"content":""} = 14 bytes, so content needs to be limit - 14
		contentSize := int(limit) - 14
		if contentSize <= 0 {
			t.Skip("Limit too small for test")
		}

		content := strings.Repeat("A", contentSize)
		payload := mustMarshalMessage(t, content)

		if int64(len(payload)) > limit {
			t.Logf("Payload size %d exceeds limit %d, adjusting", len(payload), limit)
			// Adjust content size
			contentSize = int(limit) - len(payload) + len(content)
			if contentSize <= 0 {
				t.Skip("Cannot create exact-size message")
			}
			content = strings.Repeat("A", contentSize)
			payload = mustMarshalMessage(t, content)
		}

		if err := sender.WriteMessage(websocket.TextMessage, payload); err != nil {
			t.Fatalf("Failed to send at-limit message: %v", err)
		}

		// Receiver should get the message
		if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf("Failed to set read deadline: %v", err)
		}

		messageType, message, err := receiver.ReadMessage()
		if err != nil {
			t.Fatalf("Expected to receive at-limit message: %v", err)
		}

		if messageType != websocket.TextMessage {
			t.Errorf("Expected text message, got type %d", messageType)
		}

		var received server.Message
		if err := json.Unmarshal(message, &received); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}
	})

	t.Run("Message one byte over limit", func(t *testing.T) {
		const limit int64 = 100
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.MaxMessageSize = limit
		})

		sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect sender: %v", err)
		}
		defer func() { _ = sender.Close() }()
		defer func() { _ = senderResp.Body.Close() }()

		receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect receiver: %v", err)
		}
		defer func() { _ = receiver.Close() }()
		defer func() { _ = receiverResp.Body.Close() }()

		time.Sleep(50 * time.Millisecond)

		// Create message that exceeds limit by 1 byte
		oversizedContent := strings.Repeat("A", int(limit)+1)
		oversizedPayload := mustMarshalMessage(t, oversizedContent)

		if err := sender.WriteMessage(websocket.TextMessage, oversizedPayload); err != nil && !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
			t.Logf("Send error (expected): %v", err)
		}

		expectNoMessage(t, receiver, 300*time.Millisecond)
	})

	t.Run("Very large message well over limit", func(t *testing.T) {
		const limit int64 = 64
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.MaxMessageSize = limit
		})

		sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect sender: %v", err)
		}
		defer func() { _ = sender.Close() }()
		defer func() { _ = senderResp.Body.Close() }()

		receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect receiver: %v", err)
		}
		defer func() { _ = receiver.Close() }()
		defer func() { _ = receiverResp.Body.Close() }()

		time.Sleep(50 * time.Millisecond)

		// Create a very large message
		hugeContent := strings.Repeat("X", int(limit)*10)
		hugePayload := mustMarshalMessage(t, hugeContent)

		if err := sender.WriteMessage(websocket.TextMessage, hugePayload); err != nil {
			t.Logf("Expected error sending huge message: %v", err)
		}

		expectNoMessage(t, receiver, 300*time.Millisecond)

		// Verify sender connection is closed
		if err := sender.SetReadDeadline(time.Now().Add(300 * time.Millisecond)); err != nil {
			t.Logf("Set deadline error: %v", err)
		}
		if _, _, readErr := sender.ReadMessage(); readErr == nil {
			t.Error("Expected sender connection to be closed")
		}
	})

	t.Run("Multiple small messages within limit", func(t *testing.T) {
		const limit int64 = 200
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.MaxMessageSize = limit
		})

		sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect sender: %v", err)
		}
		defer func() { _ = sender.Close() }()
		defer func() { _ = senderResp.Body.Close() }()

		receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect receiver: %v", err)
		}
		defer func() { _ = receiver.Close() }()
		defer func() { _ = receiverResp.Body.Close() }()

		time.Sleep(50 * time.Millisecond)

		// Send multiple small messages
		for i := 0; i < 5; i++ {
			content := strings.Repeat("A", 20)
			if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, content)); err != nil {
				t.Errorf("Failed to send message %d: %v", i, err)
			}

			// Verify receiver gets it
			if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatalf("Failed to set read deadline: %v", err)
			}

			if _, _, err := receiver.ReadMessage(); err != nil {
				t.Errorf("Failed to receive message %d: %v", i, err)
			}
		}
	})

	t.Run("Zero-length message", func(t *testing.T) {
		const limit int64 = 100
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.MaxMessageSize = limit
		})

		sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect sender: %v", err)
		}
		defer func() { _ = sender.Close() }()
		defer func() { _ = senderResp.Body.Close() }()

		receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect receiver: %v", err)
		}
		defer func() { _ = receiver.Close() }()
		defer func() { _ = receiverResp.Body.Close() }()

		time.Sleep(50 * time.Millisecond)

		// Send message with empty content
		if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "")); err != nil {
			t.Errorf("Failed to send zero-length message: %v", err)
		}

		// Receiver should get it
		if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf("Failed to set read deadline: %v", err)
		}

		messageType, message, err := receiver.ReadMessage()
		if err != nil {
			t.Errorf("Failed to receive zero-length message: %v", err)
		}

		if messageType != websocket.TextMessage {
			t.Errorf("Expected text message, got type %d", messageType)
		}

		var received server.Message
		if err := json.Unmarshal(message, &received); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}

		if received.Content != "" {
			t.Errorf("Expected empty content, got %q", received.Content)
		}
	})
}

// TestSecurityConstraintsCombined tests combinations of security constraints.
func TestSecurityConstraintsCombined(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Invalid origin with oversized message", func(t *testing.T) {
		const limit int64 = 64
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{"http://allowed.com"}
			cfg.MaxMessageSize = limit
		})

		header := http.Header{}
		header.Set("Origin", "http://blocked.com")
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			_ = conn.Close()
			_ = resp.Body.Close()
			t.Fatal("Expected connection to fail with invalid origin")
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	})

	t.Run("Valid origin with message size and rate limits", func(t *testing.T) {
		configureServerForTest(t, testServer.URL, func(cfg *server.Config) {
			cfg.AllowedOrigins = []string{testServer.URL}
			cfg.MaxMessageSize = 100
			cfg.RateLimit = server.RateLimitConfig{
				Burst:          3,
				RefillInterval: 500 * time.Millisecond,
			}
		})

		sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect sender: %v", err)
		}
		defer func() { _ = sender.Close() }()
		defer func() { _ = senderResp.Body.Close() }()

		receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(testServer.URL))
		if err != nil {
			t.Fatalf("Failed to connect receiver: %v", err)
		}
		defer func() { _ = receiver.Close() }()
		defer func() { _ = receiverResp.Body.Close() }()

		time.Sleep(50 * time.Millisecond)

		// Send messages up to rate limit
		for i := 0; i < 3; i++ {
			if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "msg")); err != nil {
				t.Errorf("Failed to send message %d: %v", i, err)
			}

			if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatalf("Failed to set read deadline: %v", err)
			}

			if _, _, err := receiver.ReadMessage(); err != nil {
				t.Errorf("Failed to receive message %d: %v", i, err)
			}
		}

		// Next message should be rate limited
		if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "over")); err != nil {
			t.Logf("Send error: %v", err)
		}
		expectNoMessage(t, receiver, 200*time.Millisecond)
	})
}
