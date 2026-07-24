# API Reference

Every endpoint the server exposes, and the WebSocket message protocol. For working client code, see
[Connecting a client](../guides/connecting-a-client.md).

## HTTP endpoints

Routes are registered in `internal/server/routes.go` on a plain `http.ServeMux`.

### `GET /ws` — WebSocket upgrade

| Property | Value |
| -------- | ----- |
| Methods  | `GET` only |
| Success  | `101 Switching Protocols` |
| Errors   | `405` on any other method, `403` when the `Origin` header is missing or not allowed |

The handshake headers (`Upgrade`, `Connection`, `Sec-WebSocket-Version: 13`, `Sec-WebSocket-Key`) are
set automatically by every WebSocket client library. The `Origin` header is **not** — see
[Origin matching](configuration.md#origin-matching).

Response bodies for the error cases:

```
405 Method not allowed. WebSocket endpoint only accepts GET requests.
403 Forbidden
```

The `403` body is produced by the gorilla/websocket upgrader, which calls `http.Error` with the
standard status text and also sets a `Sec-Websocket-Version: 13` response header.

Connections are also refused while the server is shutting down: the socket is accepted and then
closed immediately.

### `GET /` — health check

| Property | Value |
| -------- | ----- |
| Methods  | any |
| Status   | `200 OK` |
| `Content-Type` | `text/plain` |
| Body     | `Blip server is running!` |

`/` is registered as the `ServeMux` catch-all, so **every unmatched path returns this response**,
not a 404. The Docker and Compose health checks poll this endpoint.

### `GET /test` — built-in test page

| Property | Value |
| -------- | ----- |
| Methods  | any |
| Status   | `200 OK` |
| `Content-Type` | `text/html; charset=utf-8` |

A self-contained HTML page with connect/disconnect controls and a message log, compiled into the
binary from `internal/server/testpage.html` with `go:embed`. Its JavaScript derives the WebSocket URL
from `location`, so the page works at whatever host, port, or TLS terminator it is reached through.
It is a development aid — there is no flag to disable it, so block `/test` at the reverse proxy in
production.

## Message protocol

Both directions carry UTF-8 JSON text frames with a single field:

```json
{ "content": "Your message text here" }
```

| Field     | Type   | Required | Description                           |
| --------- | ------ | -------- | ------------------------------------- |
| `content` | string | yes      | Message body relayed to other clients |

There is no envelope, no message type, no sender identity, and no acknowledgement. The server never
sends application-level errors — failures are visible only as a dropped message or a closed socket.

### Delivery semantics

1. A client sends a JSON frame.
2. The server **normalizes it** to exactly `{"content":...}` before broadcasting. Unknown fields are
   discarded; a payload with no `content` field is relayed as `{"content":""}`. Characters that
   `encoding/json` escapes stay escaped on the way out, so the characters `<`, `>`, and `&` arrive
   as the six-character sequences `\u003c`, `\u003e`, and `\u0026`. That is JSON
   transport encoding, not sanitization: a client rendering the text must still escape on output.
3. The message is fanned out to every other connected client. **The sender does not receive its own
   message.**
4. Nothing is persisted. A client that connects later sees no history.

When several messages are queued for one client, the write pump packs them into a single frame
separated by newlines (`\n`), so a receiver may need to split on newlines before parsing JSON.

### Limits and failure modes

| Condition                                                                 | Server behavior                           | Client-visible effect                                    |
| ------------------------------------------------------------------------- | ----------------------------------------- | -------------------------------------------------------- |
| Payload larger than `MAX_MESSAGE_SIZE` (512 bytes default)                | Read limit exceeded, read pump exits      | Connection closed                                        |
| Payload is not valid JSON                                                 | Logged and discarded                      | Nothing — connection stays open                          |
| Rate limit exceeded (`RATE_LIMIT_BURST` per `RATE_LIMIT_REFILL_INTERVAL`) | Message dropped                           | Message silently never arrives; connection stays open    |
| Send queue full (256 messages)                                            | Client removed from hub                   | Connection closed                                        |
| No pong within 60s of the last read deadline reset                        | Read pump exits                           | Connection closed                                        |

The size limit applies to the raw frame, so JSON punctuation counts: `{"content":""}` is 14 bytes of
overhead, leaving roughly 498 bytes of text under the default limit.

Rate limiting is a token bucket per connection. The bucket holds `RATE_LIMIT_BURST` tokens and
refills continuously at `burst / interval` tokens per second, so the defaults allow a 5-message
burst and 5 messages per second sustained. Exceeding it **drops messages, it does not disconnect**.

### Keepalive

The server sends a WebSocket ping every 54 seconds and requires a pong within the 60-second read
deadline. Standard client libraries answer pings automatically; application-level heartbeat messages
are unnecessary and count against the rate limit.

## Related

- [Configuration reference](configuration.md) — the variables behind every limit above
- [Connecting a client](../guides/connecting-a-client.md) — JavaScript, Go, and Python examples
- [Architecture overview](../architecture/overview.md) — how a message travels through the server
