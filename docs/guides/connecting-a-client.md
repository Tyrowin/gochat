# Connecting a Client

How to talk to GoChat from your own code. For the exact wire contract, see the
[API reference](../reference/api.md).

Two rules cover most first-time failures:

1. **Send an `Origin` header the server allows.** Browsers do it for you; every other client must set
   it explicitly, and a missing header is rejected with `403`.
2. **Send JSON**, not bare text: `{"content":"hello"}`. Plain strings are dropped silently.

## Browser (JavaScript)

The browser sets `Origin` to the page's own origin, so the page must be served from a host listed in
`ALLOWED_ORIGINS`.

```javascript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.addEventListener("open", () => {
  ws.send(JSON.stringify({ content: "Hello from the browser" }));
});

ws.addEventListener("message", (event) => {
  // Queued messages arrive newline-separated in one frame.
  for (const line of event.data.split("\n")) {
    if (line) console.log("received:", JSON.parse(line).content);
  }
});

ws.addEventListener("close", (e) => console.log("closed:", e.code, e.reason));
ws.addEventListener("error", (e) => console.error("error:", e));
```

You will not receive your own messages — echo them locally if the UI needs them.

### Reconnecting

The server closes connections on shutdown, on an oversized message, and when a client falls behind.
Reconnect with backoff:

```javascript
let delay = 1000;

function connect() {
  const ws = new WebSocket("ws://localhost:8080/ws");
  ws.onopen = () => { delay = 1000; };
  ws.onclose = () => {
    setTimeout(connect, delay);
    delay = Math.min(delay * 2, 30000);
  };
  ws.onmessage = (e) => console.log(e.data);
}

connect();
```

Do not add an application-level heartbeat: the server already pings every 54 seconds and your
library answers automatically. Heartbeat messages would consume rate-limit tokens for nothing.

## Go (gorilla/websocket)

```go
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Message struct {
	Content string `json:"content"`
}

func main() {
	headers := http.Header{}
	headers.Set("Origin", "http://localhost:8080") // required

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", headers)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(Message{Content: "Hello from Go"}); err != nil {
		log.Fatalf("write: %v", err)
	}

	for {
		var received Message
		if err := conn.ReadJSON(&received); err != nil {
			log.Printf("read: %v", err)
			return
		}
		log.Printf("received: %s", received.Content)
	}
}
```

Omitting the `Origin` header fails the handshake with
`websocket: bad handshake` and a `403` from the server.

## Python (websockets)

```python
import asyncio
import json
import websockets

async def chat():
    async with websockets.connect(
        "ws://localhost:8080/ws",
        additional_headers={"Origin": "http://localhost:8080"},
    ) as ws:
        await ws.send(json.dumps({"content": "Hello from Python"}))
        async for raw in ws:
            for line in raw.split("\n"):
                if line:
                    print("received:", json.loads(line)["content"])

asyncio.run(chat())
```

> The header keyword changed across `websockets` releases (`extra_headers` in older versions). Check
> the [library documentation](https://websockets.readthedocs.io/) for the version you install.

## Command line (websocat)

Useful for probing a running server:

```bash
# Succeeds against the default configuration
websocat -H='Origin: http://localhost:8080' ws://localhost:8080/ws

# Rejected: origin not in ALLOWED_ORIGINS
websocat -H='Origin: http://evil.example' ws://localhost:8080/ws
```

Type `{"content":"hi"}` and press Enter to send. Install with `brew install websocat` or
`cargo install websocat`.

## Production checklist for clients

- Dial `wss://` rather than `ws://` — browsers block plain `ws://` from HTTPS pages. See
  [Deploying to production](deploying-to-production.md).
- Add your real front-end origin to `ALLOWED_ORIGINS`, including `www` and any subdomains.
- Keep payloads under `MAX_MESSAGE_SIZE` (512 bytes by default) — an oversized frame closes the
  connection with no error message.
- Throttle to the configured rate limit. Exceeding it drops messages without any signal.

## Related

- [API reference](../reference/api.md) — endpoints, protocol, failure modes
- [Configuration reference](../reference/configuration.md) — origins, size, rate limits
- [Getting started](../getting-started.md) — run the server first
