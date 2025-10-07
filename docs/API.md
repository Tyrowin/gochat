# WebSocket API Documentation

This document describes the GoChat WebSocket API for real-time chat communication.

## Connection Endpoint

**Endpoint:** `ws://localhost:8080/ws` (or `wss://yourdomain.com/ws` in production)

**Method:** GET (WebSocket upgrade)

**Required Headers:**

- `Upgrade: websocket`
- `Connection: Upgrade`
- `Sec-WebSocket-Version: 13`
- `Sec-WebSocket-Key: <base64-encoded-key>`
- `Origin: <allowed-origin>` (must match server configuration)

**Note:** Most WebSocket client libraries handle these headers automatically.

## Message Protocol

GoChat uses a simple JSON-based message protocol for all client-server communication.

### Message Format

```json
{
  "content": "Your message text here"
}
```

### Field Definitions

| Field     | Type   | Required | Description                                            |
| --------- | ------ | -------- | ------------------------------------------------------ |
| `content` | string | Yes      | The message text to broadcast to all connected clients |

### Constraints

- **Maximum message size:** 512 bytes (configurable)
- **Format:** Must be valid JSON
- **Content field:** Required and must be a string

### Message Flow

1. Client connects to the WebSocket endpoint
2. Client sends JSON messages with the `content` field
3. Server validates the message format and size
4. Server broadcasts the message to all other connected clients (excluding the sender)
5. Each client receives messages from all other clients in real-time

### Important Notes

- Messages are **broadcast to all clients except the sender**
- **No message history** is stored - only real-time communication
- Invalid JSON or oversized messages will cause the connection to close
- Rate limiting applies per connection (see [Security](SECURITY.md))

## Code Examples

### JavaScript (Browser)

```javascript
// Connect to the WebSocket server
const ws = new WebSocket("ws://localhost:8080/ws");

// Connection opened
ws.addEventListener("open", (event) => {
  console.log("Connected to GoChat server");

  // Send a message
  const message = {
    content: "Hello from JavaScript!",
  };
  ws.send(JSON.stringify(message));
});

// Receive messages
ws.addEventListener("message", (event) => {
  const message = JSON.parse(event.data);
  console.log("Received:", message.content);
});

// Connection closed
ws.addEventListener("close", (event) => {
  console.log("Disconnected from server");
});

// Error handling
ws.addEventListener("error", (error) => {
  console.error("WebSocket error:", error);
});
```

### Python (with websockets library)

```python
import asyncio
import websockets
import json

async def chat():
    uri = "ws://localhost:8080/ws"
    async with websockets.connect(uri) as websocket:
        # Send a message
        message = {"content": "Hello from Python!"}
        await websocket.send(json.dumps(message))

        # Receive messages
        async for message in websocket:
            data = json.loads(message)
            print(f"Received: {data['content']}")

asyncio.run(chat())
```

### Go (with gorilla/websocket)

```go
package main

import (
    "encoding/json"
    "log"
    "github.com/gorilla/websocket"
)

type Message struct {
    Content string `json:"content"`
}

func main() {
    // Connect to server
    conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // Send message
    msg := Message{Content: "Hello from Go!"}
    if err := conn.WriteJSON(msg); err != nil {
        log.Fatal(err)
    }

    // Receive messages
    for {
        var received Message
        if err := conn.ReadJSON(&received); err != nil {
            log.Fatal(err)
        }
        log.Printf("Received: %s", received.Content)
    }
}
```

### Node.js (with ws library)

```javascript
const WebSocket = require("ws");

const ws = new WebSocket("ws://localhost:8080/ws");

ws.on("open", function open() {
  console.log("Connected to GoChat server");

  // Send a message
  const message = {
    content: "Hello from Node.js!",
  };
  ws.send(JSON.stringify(message));
});

ws.on("message", function incoming(data) {
  const message = JSON.parse(data);
  console.log("Received:", message.content);
});

ws.on("close", function close() {
  console.log("Disconnected from server");
});

ws.on("error", function error(err) {
  console.error("WebSocket error:", err);
});
```

### cURL (for testing)

While cURL doesn't natively support WebSocket upgrades, you can use tools like `websocat` for command-line testing:

```bash
# Install websocat
# On macOS: brew install websocat
# On Linux: cargo install websocat

# Connect and send a message
echo '{"content":"Hello from command line!"}' | websocat ws://localhost:8080/ws
```

## Interactive Testing

### Built-in Test Page

Navigate to `http://localhost:8080/test` in your browser to access an interactive HTML test page.

**Features:**

- Connect/disconnect from the WebSocket server
- Send messages and see them broadcast to other connected clients
- View connection status in real-time
- See message history during the session
- Test multi-client chat by opening multiple browser windows

### Testing Multi-Client Chat

1. Open `http://localhost:8080/test` in two or more browser windows/tabs
2. Click "Connect" in each window
3. Send a message from one window
4. Observe the message appearing in all other windows (but not the sender's)

## Error Handling

### Connection Errors

**Origin Not Allowed:**

```
WebSocket connection failed: Error during WebSocket handshake: Unexpected response code: 403
```

- **Cause:** The Origin header doesn't match the allowed origins
- **Solution:** Add your origin to the allowed list in the server configuration
- **See:** [Security Documentation](SECURITY.md#origin-validation)

**Connection Refused:**

```
WebSocket connection failed: Connection refused
```

- **Cause:** Server is not running or not accessible
- **Solution:** Verify the server is running and the URL is correct

### Message Errors

**Invalid JSON:**

- Sending non-JSON data will cause the connection to close
- Always use `JSON.stringify()` or equivalent to serialize messages

**Message Too Large:**

- Messages exceeding 512 bytes (default) will close the connection
- Keep messages concise or adjust the server's `MaxMessageSize` configuration

**Rate Limit Exceeded:**

- Sending too many messages too quickly will close the connection
- Default limit: 5 messages per second with burst capacity of 5
- See [Security Documentation](SECURITY.md#rate-limiting) for details

## Production Considerations

### Use WSS (WebSocket Secure)

In production, always use WSS instead of WS:

```javascript
const ws = new WebSocket("wss://chat.yourdomain.com/ws");
```

**Why:**

- Required for HTTPS websites (browsers block WS from HTTPS pages)
- Encrypts all traffic
- Prevents man-in-the-middle attacks

See [Deployment Guide](DEPLOYMENT.md) for TLS setup instructions.

### Origin Configuration

Update the server's allowed origins to include your production domain:

```go
AllowedOrigins: []string{
    "https://yourdomain.com",
    "https://www.yourdomain.com",
}
```

### Connection Timeouts

WebSocket connections are long-lived. Ensure your reverse proxy is configured with appropriate timeouts:

**Nginx:**

```nginx
proxy_read_timeout 86400;  # 24 hours
proxy_send_timeout 86400;
```

**Caddy:**
Caddy handles WebSocket timeouts automatically.

## Advanced Usage

### Reconnection Logic

Implement automatic reconnection in case of connection loss:

```javascript
let ws;
let reconnectInterval = 1000; // Start with 1 second
const maxReconnectInterval = 30000; // Max 30 seconds

function connect() {
  ws = new WebSocket("ws://localhost:8080/ws");

  ws.onopen = () => {
    console.log("Connected");
    reconnectInterval = 1000; // Reset on successful connection
  };

  ws.onclose = () => {
    console.log("Disconnected. Reconnecting...");
    setTimeout(() => {
      reconnectInterval = Math.min(reconnectInterval * 2, maxReconnectInterval);
      connect();
    }, reconnectInterval);
  };

  ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    console.log("Received:", message.content);
  };
}

connect();
```

### Heartbeat/Ping-Pong

To detect dead connections, implement a heartbeat mechanism:

```javascript
let heartbeatInterval;

ws.onopen = () => {
  // Send heartbeat every 30 seconds
  heartbeatInterval = setInterval(() => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ content: "ping" }));
    }
  }, 30000);
};

ws.onclose = () => {
  clearInterval(heartbeatInterval);
};
```

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Installation and basic setup
- [Security](SECURITY.md) - Security features and configuration
- [Deployment](DEPLOYMENT.md) - Production deployment guide
- [Development](DEVELOPMENT.md) - Contributing and development guide
