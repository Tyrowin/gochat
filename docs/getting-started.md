# Getting Started

Build GoChat, run it, and watch two browser tabs chat with each other. About 10 minutes.

## Prerequisites

- **Go 1.25.12 or later** — check with `go version`. Older 1.24/1.25 toolchains work too; Go
  downloads the version named in `go.mod` automatically.
- **Git**
- **GNU Make** (optional — every step below also shows the plain `go` command)
  - Windows: `choco install make`
  - macOS: `xcode-select --install`
  - Linux: `apt install make` or `yum install make`

## 1. Clone the repository

```bash
git clone https://github.com/maltemindedal/gochat.git
cd gochat
```

## 2. Build the server

```bash
make build
```

Without Make:

```bash
go build -o bin/gochat ./cmd/server        # macOS / Linux
go build -o bin/gochat.exe ./cmd/server    # Windows
```

The binary lands in `bin/`. `make build` runs `go fmt` and `go vet` first, so a formatting or vet
failure stops the build.

## 3. Run it

```bash
./bin/gochat          # macOS / Linux
.\bin\gochat.exe      # Windows PowerShell
```

Or `make run`, which rebuilds first.

Expected output:

```
Starting GoChat server...
2026/07/23 15:14:04 Hub started and ready to manage WebSocket connections
2026/07/23 15:14:04 Server starting on port :8080
2026/07/23 15:14:04 Server listening on :8080
```

The server is now on <http://localhost:8080>. Confirm it in another terminal:

```bash
curl http://localhost:8080/
# GoChat server is running!
```

## 4. Chat with yourself

1. Open <http://localhost:8080/test> in a browser tab.
2. Click **Connect**. The status bar turns green.
3. Open the same URL in a second tab and click **Connect** there too.
4. Type a message in the first tab and press Enter.

The message appears in the second tab. It does **not** appear as an incoming message in the first —
the server broadcasts to everyone *except* the sender, so your own text only shows as the local
"You:" echo the page adds.

The server logs each step:

```
Client registered from 127.0.0.1:54321. Total clients: 1
Received message from 127.0.0.1:54321: {"content":"hello"}
Broadcasting message to 1 clients
```

## 5. Stop it

Press `Ctrl+C`. The server drains in two steps and logs:

```
Received shutdown signal: interrupt
Step 1: Stopping HTTP server...
Step 2: Shutting down WebSocket hub...
Server stopped gracefully
```

## What you just ran

Three endpoints, no database, no authentication, no message history:

- `GET /` — health check, and the catch-all for any unmatched path
- `GET /ws` — the WebSocket endpoint clients connect to
- `GET /test` — the page you just used

Defaults: port `:8080`, only `http://localhost:8080` accepted as an origin, 512-byte messages, 5
messages/second per connection. All five settings are environment variables — see the
[configuration reference](reference/configuration.md).

## Troubleshooting

**`address already in use` / `bind: Only one usage of each socket address`**
Something else holds port 8080. Find it with `lsof -i :8080` (macOS/Linux) or
`netstat -ano | findstr :8080` (Windows), or move GoChat: `SERVER_PORT=:9090 ./bin/gochat`.

**The test page says "Connection error" and the log says `Blocked WebSocket connection from disallowed origin`**
You reached the page at a hostname other than `localhost:8080` — `127.0.0.1:8080` counts as a
different origin. Either use `http://localhost:8080/test` or allow the origin you are using:

```bash
ALLOWED_ORIGINS=http://localhost:8080,http://127.0.0.1:8080 ./bin/gochat
```

**Messages stop arriving but the connection stays open**
You are hitting the rate limit — more than 5 messages per second. Excess messages are dropped
silently. Raise `RATE_LIMIT_BURST` if that is too strict.

**`permission denied` on macOS/Linux**
`chmod +x bin/gochat`.

## Next steps

- [Connecting a client](guides/connecting-a-client.md) — talk to the server from your own code
- [Configuration reference](reference/configuration.md) — every environment variable
- [API reference](reference/api.md) — endpoints, message format, failure modes
- [Architecture overview](architecture/overview.md) — how the hub and client pumps fit together
- [Deploying to production](guides/deploying-to-production.md) — TLS, reverse proxy, systemd
