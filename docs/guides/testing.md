# Testing

How the suite is organized and how to run, extend, and measure it.

## Layout

All tests live under `test/`, outside the packages they exercise, so they see `internal/server`
through its exported API only.

```
test/
├── unit/                    # package unit — components in isolation
│   ├── error_handling_test.go   # read/write error paths, panic recovery
│   ├── handlers_test.go         # health handler, routing, server construction
│   ├── hub_test.go              # hub channels, registration, broadcast, shutdown
│   └── websocket_test.go        # upgrader config, method and header validation
├── integration/             # package integration — real servers over real sockets
│   ├── multiclient_test.go      # many clients exchanging messages concurrently
│   ├── security_test.go         # origin validation, size limits, rate limiting
│   ├── server_test.go           # health endpoint, timeouts, full startup path
│   ├── shutdown_test.go         # graceful shutdown with and without clients
│   └── websocket_test.go        # connection lifecycle, broadcasting
└── testhelpers/             # shared helpers (no tests of its own)
    └── helpers.go
```

Integration tests spin up `httptest` servers and dial them with a real `gorilla/websocket` client,
so they exercise the actual handshake, origin check, and pumps.

## Running

```bash
make test                  # everything, with -race and -v
make test-unit             # ./test/unit/... only
make test-integration      # ./test/integration/... only
make race                  # -race without -v
```

Plain Go:

```bash
go test ./...                                   # everything
go test -v -race ./test/unit/...                # one package
go test -v -race -run TestHubShutdown ./test/unit  # one test
```

The integration suite takes roughly 12 seconds because several tests wait on real timeouts; the unit
suite finishes in about 1.5 seconds. Always keep `-race` on — the hub, the client pumps, and the
config store are all concurrent.

## Coverage

```bash
make test-coverage               # coverage.out + coverage.html + per-function table
make test-coverage-unit          # unit-coverage.*
make test-coverage-integration   # integration-coverage.*
```

These pass `-coverpkg=./cmd/...,./internal/...` so coverage is attributed to the code under test
rather than to the test packages, and print `go tool cover -func` at the end. Open `coverage.html`
in a browser for the annotated source.

Measured **71.0% of statements** across `internal/server` on 2026-07-23 (unit 56.1%, integration
68.2%). CI collects coverage and uploads it to Codecov but does not enforce a threshold — nothing
fails a build for dropping coverage.

## Helpers

`test/testhelpers` provides the shared plumbing. Use it instead of hand-rolling servers and dials:

| Helper                                       | Purpose                                             |
| -------------------------------------------- | --------------------------------------------------- |
| `CreateTestServer(handler)`                  | `httptest` server for a handler                      |
| `CreateTestServerWithConfig(...)`            | Same, with a specific `server.Config` applied         |
| `ConnectWebSocket(url)`                      | Dial a `ws://` URL and return the connection          |
| `SendMessage(conn, content)`                 | Send `{"content": ...}`                               |
| `ReceiveMessage(conn)`                       | Read one JSON message into a map                      |
| `SendRawMessage` / `ReceiveRawMessage`       | Frame-level access for protocol edge cases            |
| `MakeRequest(t, method, url)`                | Plain HTTP request                                    |
| `AssertStatusCode` / `AssertContentType` / `AssertMessageContent` | Common assertions          |
| `CloseWebSocket(conn)`                       | Close cleanly                                         |

## Writing tests

Follow the conventions already in the suite:

- Name tests `TestSubjectBehavior` — `TestWebSocketOriginValidation`, `TestHubShutdownTimeout`.
- Put anything that needs a listening socket in `test/integration`, everything else in `test/unit`.
- Use table-driven subtests with `t.Run` for multiple scenarios of one behavior.
- Cover the failure path, not just the happy one — most bugs in this codebase live in error handling
  and shutdown ordering.
- Reset shared state. The hub and the active config are package-level globals; use
  `CreateTestServerWithConfig` and `server.SetConfig(nil)` rather than mutating global state directly
  and leaving it changed.
- Prefer waiting on a channel or polling with a deadline over `time.Sleep` for synchronization.

There are currently no benchmarks. `make bench` runs `go test -bench=. -benchmem ./...` and will pick
them up if you add any.

## In CI

The `build-and-test` job runs `go test -v -race -coverprofile=coverage.out ./...` on Ubuntu with the
pinned Go version, and the `build-matrix` job re-runs `go test -v ./...` (no race detector) against
Go 1.24.x and 1.25.x. See [Contributing](../contributing.md#continuous-integration).

## Related

- [Contributing](../contributing.md) — the full pre-push check list
- [Make targets reference](../reference/make-targets.md) — every test target
- [Architecture overview](../architecture/overview.md) — what the concurrency tests are protecting
