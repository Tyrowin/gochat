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

const (
	exampleOriginHTTP = "http://example.com"

	// Error message constants
	errFailedConnectSender   = "Failed to connect sender: %v"
	errFailedConnectReceiver = "Failed to connect receiver: %v"
	errFailedSetReadDeadline = "Failed to set read deadline: %v"
	errExpectedTextMessage   = "Expected text message, got type %d"
	errFailedUnmarshal       = "Failed to unmarshal message: %v"
	errFailedSendMessage     = "Failed to send message %d: %v"
	errFailedReceiveMessage  = "Failed to receive message %d: %v"
)

// Helper function to assert connection should fail with forbidden status
func assertConnectionFails(t *testing.T, wsURL string, header http.Header, errorMsg string) {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		_ = conn.Close()
		_ = resp.Body.Close()
		t.Fatal(errorMsg)
	}
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, resp.StatusCode)
		}
	}
}

// Helper function to assert connection succeeds
func assertConnectionSucceeds(t *testing.T, wsURL string, header http.Header, origin string) {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Errorf("Expected origin %q to be allowed: %v", origin, err)
		return
	}
	_ = conn.Close()
	if resp != nil {
		_ = resp.Body.Close()
	}
}

// Helper function to test missing origin header
func testMissingOriginHeader(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{serverURL}
	})

	header := http.Header{}
	assertConnectionFails(t, wsURL, header, "Expected connection to fail with missing origin")
}

// Helper function to test empty origin header
func testEmptyOriginHeader(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{serverURL}
	})

	header := http.Header{}
	header.Set("Origin", "")
	assertConnectionFails(t, wsURL, header, "Expected connection to fail with empty origin")
}

// Helper function to test malformed origins
func testMalformedOrigins(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{serverURL}
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
}

// Helper function to test case sensitivity
func testCaseSensitivity(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{exampleOriginHTTP}
	})

	caseVariations := []string{
		"http://EXAMPLE.COM",
		"http://Example.Com",
		"HTTP://example.com",
	}

	for _, origin := range caseVariations {
		header := http.Header{}
		header.Set("Origin", origin)
		assertConnectionSucceeds(t, wsURL, header, origin)
	}
}

// Helper function to test wildcard origin
func testWildcardOrigin(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{"*"}
	})

	testOrigins := []string{
		exampleOriginHTTP,
		"https://another.com",
		"http://localhost:3000",
	}

	for _, origin := range testOrigins {
		header := http.Header{}
		header.Set("Origin", origin)
		assertConnectionSucceeds(t, wsURL, header, origin)
	}
}

// Helper function to test different port rejection
func testDifferentPort(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{"http://localhost:8080"}
	})

	header := http.Header{}
	header.Set("Origin", "http://localhost:9090")
	assertConnectionFails(t, wsURL, header, "Expected connection to fail with different port")
}

// Helper function to test path component handling
func testPathComponentIgnored(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{exampleOriginHTTP}
	})

	header := http.Header{}
	header.Set("Origin", "http://example.com/some/path")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Errorf("Expected origin with path to be allowed: %v", err)
		return
	}
	_ = conn.Close()
	if resp != nil {
		_ = resp.Body.Close()
	}
}

// Helper function to test HTTP vs HTTPS scheme difference
func testSchemeDifference(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{exampleOriginHTTP}
	})

	header := http.Header{}
	header.Set("Origin", "https://example.com")
	assertConnectionFails(t, wsURL, header, "Expected HTTPS origin to be rejected when only HTTP is allowed")
}

// TestOriginValidationEdgeCases tests various edge cases for origin validation.
func TestOriginValidationEdgeCases(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Missing Origin header", func(t *testing.T) {
		testMissingOriginHeader(t, wsURL, testServer.URL)
	})

	t.Run("Empty Origin header", func(t *testing.T) {
		testEmptyOriginHeader(t, wsURL, testServer.URL)
	})

	t.Run("Malformed Origin URL", func(t *testing.T) {
		testMalformedOrigins(t, wsURL, testServer.URL)
	})

	t.Run("Case sensitivity in origin matching", func(t *testing.T) {
		testCaseSensitivity(t, wsURL, testServer.URL)
	})

	t.Run("Wildcard origin configuration", func(t *testing.T) {
		testWildcardOrigin(t, wsURL, testServer.URL)
	})

	t.Run("Origin with different port", func(t *testing.T) {
		testDifferentPort(t, wsURL, testServer.URL)
	})

	t.Run("Origin with path component ignored", func(t *testing.T) {
		testPathComponentIgnored(t, wsURL, testServer.URL)
	})

	t.Run("HTTP vs HTTPS scheme difference", func(t *testing.T) {
		testSchemeDifference(t, wsURL, testServer.URL)
	})
}

// Helper function to test message exactly at size limit
func testMessageAtSizeLimit(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	const limit int64 = 100
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.MaxMessageSize = limit
	})

	sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectSender, err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectReceiver, err)
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
		t.Fatalf(errFailedSetReadDeadline, err)
	}

	messageType, message, err := receiver.ReadMessage()
	if err != nil {
		t.Fatalf("Expected to receive at-limit message: %v", err)
	}

	if messageType != websocket.TextMessage {
		t.Errorf(errExpectedTextMessage, messageType)
	}

	var received server.Message
	if err := json.Unmarshal(message, &received); err != nil {
		t.Errorf(errFailedUnmarshal, err)
	}
}

// Helper function to test message one byte over limit
func testMessageOneByteOverLimit(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	const limit int64 = 100
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.MaxMessageSize = limit
	})

	sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectSender, err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectReceiver, err)
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
}

// Helper function to test very large message well over limit
func testVeryLargeMessage(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	const limit int64 = 64
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.MaxMessageSize = limit
	})

	sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectSender, err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectReceiver, err)
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
}

// Helper function to test multiple small messages within limit
func testMultipleSmallMessages(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	const limit int64 = 200
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.MaxMessageSize = limit
	})

	sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectSender, err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectReceiver, err)
	}
	defer func() { _ = receiver.Close() }()
	defer func() { _ = receiverResp.Body.Close() }()

	time.Sleep(50 * time.Millisecond)

	// Send multiple small messages
	for i := 0; i < 5; i++ {
		content := strings.Repeat("A", 20)
		if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, content)); err != nil {
			t.Errorf(errFailedSendMessage, i, err)
		}

		// Verify receiver gets it
		if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf(errFailedSetReadDeadline, err)
		}

		if _, _, err := receiver.ReadMessage(); err != nil {
			t.Errorf(errFailedReceiveMessage, i, err)
		}
	}
}

// Helper function to test zero-length message
func testZeroLengthMessage(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	const limit int64 = 100
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.MaxMessageSize = limit
	})

	sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectSender, err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectReceiver, err)
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
		t.Fatalf(errFailedSetReadDeadline, err)
	}

	messageType, message, err := receiver.ReadMessage()
	if err != nil {
		t.Errorf("Failed to receive zero-length message: %v", err)
	}

	if messageType != websocket.TextMessage {
		t.Errorf(errExpectedTextMessage, messageType)
	}

	var received server.Message
	if err := json.Unmarshal(message, &received); err != nil {
		t.Errorf(errFailedUnmarshal, err)
	}

	if received.Content != "" {
		t.Errorf("Expected empty content, got %q", received.Content)
	}
}

// TestMessageSizeLimitEdgeCases tests various edge cases for message size validation.
func TestMessageSizeLimitEdgeCases(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Message exactly at size limit", func(t *testing.T) {
		testMessageAtSizeLimit(t, wsURL, testServer.URL)
	})

	t.Run("Message one byte over limit", func(t *testing.T) {
		testMessageOneByteOverLimit(t, wsURL, testServer.URL)
	})

	t.Run("Very large message well over limit", func(t *testing.T) {
		testVeryLargeMessage(t, wsURL, testServer.URL)
	})

	t.Run("Multiple small messages within limit", func(t *testing.T) {
		testMultipleSmallMessages(t, wsURL, testServer.URL)
	})

	t.Run("Zero-length message", func(t *testing.T) {
		testZeroLengthMessage(t, wsURL, testServer.URL)
	})
}

// Helper function to test invalid origin with oversized message
func testInvalidOriginWithOversizedMessage(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	const limit int64 = 64
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
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
}

// Helper function to test valid origin with message size and rate limits
func testValidOriginWithSizeAndRateLimits(t *testing.T, wsURL, serverURL string) {
	t.Helper()
	configureServerForTest(t, serverURL, func(cfg *server.Config) {
		cfg.AllowedOrigins = []string{serverURL}
		cfg.MaxMessageSize = 100
		cfg.RateLimit = server.RateLimitConfig{
			Burst:          3,
			RefillInterval: 500 * time.Millisecond,
		}
	})

	sender, senderResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectSender, err)
	}
	defer func() { _ = sender.Close() }()
	defer func() { _ = senderResp.Body.Close() }()

	receiver, receiverResp, err := websocket.DefaultDialer.Dial(wsURL, newOriginHeader(serverURL))
	if err != nil {
		t.Fatalf(errFailedConnectReceiver, err)
	}
	defer func() { _ = receiver.Close() }()
	defer func() { _ = receiverResp.Body.Close() }()

	time.Sleep(50 * time.Millisecond)

	// Send messages up to rate limit
	for i := 0; i < 3; i++ {
		if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "msg")); err != nil {
			t.Errorf(errFailedSendMessage, i, err)
		}

		if err := receiver.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf(errFailedSetReadDeadline, err)
		}

		if _, _, err := receiver.ReadMessage(); err != nil {
			t.Errorf(errFailedReceiveMessage, i, err)
		}
	}

	// Next message should be rate limited
	if err := sender.WriteMessage(websocket.TextMessage, mustMarshalMessage(t, "over")); err != nil {
		t.Logf("Send error: %v", err)
	}
	expectNoMessage(t, receiver, 200*time.Millisecond)
}

// TestSecurityConstraintsCombined tests combinations of security constraints.
func TestSecurityConstraintsCombined(t *testing.T) {
	server.StartHub()

	mux := server.SetupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	wsURL := buildWebSocketURL(t, testServer.URL)

	t.Run("Invalid origin with oversized message", func(t *testing.T) {
		testInvalidOriginWithOversizedMessage(t, wsURL, testServer.URL)
	})

	t.Run("Valid origin with message size and rate limits", func(t *testing.T) {
		testValidOriginWithSizeAndRateLimits(t, wsURL, testServer.URL)
	})
}
