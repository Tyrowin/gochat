# Configuration Reference

GoChat is configured entirely through environment variables, read once at startup by
`NewConfigFromEnv` (`internal/server/config.go`). There are no command-line flags and no
configuration file.

An annotated template lives in [`.env.example`](../../.env.example). The process does not read
`.env` itself — use your shell, `docker compose`, or a systemd unit to load it.

## Variables

| Variable                     | Type                   | Default              | Effect                                                                              |
| ---------------------------- | ---------------------- | -------------------- | ----------------------------------------------------------------------------------- |
| `SERVER_PORT`                | `[host]:port`          | `:8080`              | Listen address passed to `http.Server.Addr`.                                          |
| `ALLOWED_ORIGINS`            | comma-separated list   | `http://localhost:8080` | Origins accepted on the WebSocket handshake. `*` allows every origin.              |
| `MAX_MESSAGE_SIZE`           | integer (bytes)        | `512`                | Read limit per WebSocket message. Exceeding it closes the connection.                 |
| `RATE_LIMIT_BURST`           | integer (messages)     | `5`                  | Token bucket capacity per connection.                                                 |
| `RATE_LIMIT_REFILL_INTERVAL` | integer (whole seconds)| `1`                  | Time to refill the whole bucket. Sub-second values are not expressible.               |

### Validation behavior

Every variable is optional. Values that fail to parse do not stop the server — the default is
used and a line is written to the log:

```
Invalid MAX_MESSAGE_SIZE provided; using default value
```

Specifics:

- `SERVER_PORT` — a value with no `:` is treated as a bare port and rewritten with a leading colon,
  so `SERVER_PORT=9000` and `SERVER_PORT=:9000` are equivalent. `127.0.0.1:9000` binds to loopback only.
- `MAX_MESSAGE_SIZE`, `RATE_LIMIT_BURST`, `RATE_LIMIT_REFILL_INTERVAL` — must parse as integers
  greater than zero. Zero, negatives, and non-numeric values fall back to the default.
- `RATE_LIMIT_REFILL_INTERVAL` is parsed with `strconv.Atoi` and multiplied by `time.Second`.
  Duration strings such as `500ms` or `1s` are **invalid** and fall back to the default.

### Origin matching

`ALLOWED_ORIGINS` entries and the incoming `Origin` header are both normalized to
`scheme://host` with scheme and host lowercased, then compared exactly.

- The port is part of the host: `http://localhost:8080` does not match `http://localhost:3000`.
- Paths are ignored: `https://example.com/app` is stored as `https://example.com`.
- Entries with no scheme or no host (`example.com`, `localhost:8080`) are rejected at startup with
  `Ignoring invalid origin in configuration: ...`.
- A request with **no** `Origin` header is rejected. Browsers always send one; non-browser clients
  must set it explicitly.
- `*` anywhere in the list allows every origin, including requests whose `Origin` is otherwise
  unparseable. Rejected handshakes return `403 Forbidden`.

## Values that are not configurable

These are compile-time constants. Changing them requires editing the source.

| Value                        | Setting                    | Location                    |
| ---------------------------- | -------------------------- | --------------------------- |
| WebSocket read/write buffers | 1024 bytes each            | `internal/server/handlers.go` |
| Client send queue depth      | 256 messages               | `internal/server/client.go`   |
| Ping interval                | 54s                        | `internal/server/client.go`   |
| Read deadline                | 60s, reset on each pong     | `internal/server/client.go`   |
| Write deadline               | 10s per write               | `internal/server/client.go`   |
| HTTP read / write timeout    | 15s each                   | `internal/server/http_server.go` |
| HTTP idle timeout            | 60s                        | `internal/server/http_server.go` |
| Shutdown budget              | 15s HTTP + 15s hub, 30s cap | `cmd/server/main.go`         |

## Examples

Local development, allowing a separate front-end dev server:

```bash
export SERVER_PORT=:8080
export ALLOWED_ORIGINS=http://localhost:8080,http://localhost:3000
export MAX_MESSAGE_SIZE=512
export RATE_LIMIT_BURST=5
export RATE_LIMIT_REFILL_INTERVAL=1
```

Production behind a reverse proxy on loopback:

```bash
export SERVER_PORT=127.0.0.1:8080
export ALLOWED_ORIGINS=https://chat.example.com
export MAX_MESSAGE_SIZE=1024
export RATE_LIMIT_BURST=10
export RATE_LIMIT_REFILL_INTERVAL=2
```

## Related

- [API reference](api.md) — what the limits mean on the wire
- [Security hardening](../guides/security-hardening.md) — how to choose values
- [Architecture overview](../architecture/overview.md) — why the limits exist
