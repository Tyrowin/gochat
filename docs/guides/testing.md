# Testing

How the suite is organized and how to run, extend, and measure it.

## Layout

All tests live under `test/`, outside the packages they exercise, so they see `internal/server`
through its exported API only.

```
test/
├── unit/                    # package unit — components in isolation
│   ├── error_handling_test.go   # read/write error paths, registration accounting
│   ├── handlers_test.go         # health handler, routing, server construction
│   ├── hub_test.go              # hub channels, client count, shutdown lifecycle
│   └── websocket_test.go        # upgrader config, method and header validation
├── integration/             # package integration — real servers over real sockets
│   ├── setup_test.go            # shared plumbing: test servers, dialing, assertions
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

The integration suite takes roughly 7.5 seconds — what remains is a handful of tests that wait on
real timeouts, such as the rate-limiter refill; the unit suite finishes in about 2 seconds. Always
keep `-race` on — the hub, the client pumps, and the
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

`make test-coverage` measured **70.4% of statements** on 2026-07-24 (unit 56.6%, integration 61.1%).
That figure spans `./cmd/...` and `./internal/...` together, and `cmd/server` has no tests of its
own, so `internal/server` alone measures higher — 76.3% with `-coverpkg=./internal/...`. The same
number appears in the [README](../../README.md#status); update both together. CI collects coverage
and uploads it to Codecov but does not enforce a threshold — nothing fails a build for dropping
coverage.

## Helpers

`test/testhelpers` provides the shared plumbing. Use it instead of hand-rolling servers and dials:

| Helper                                            | Purpose                                                             |
| ------------------------------------------------- | ------------------------------------------------------------------- |
| `CreateTestServer(t, handler)`                    | `httptest` server for a handler, closed when the test ends           |
| `CreateTestServerWithTimeouts(t, handler, ServerTimeouts)` | Same, with explicit read/write/idle HTTP timeouts           |
| `WaitFor(t, timeout, what, cond)`                 | Poll a condition to a deadline — use instead of `time.Sleep`         |
| `WaitForServer(t, url, timeout)`                  | Block until a just-started server accepts requests                   |
| `Dial(t, wsURL, origin)`                          | Dial a `ws://` URL from a given `Origin`, closed when the test ends  |
| `DialPair(t, wsURL, origin)`                      | The sender/receiver pair delivery tests need                         |
| `ConnectWebSocket(url)`                           | Dial with the default dev origin; returns an error instead of failing |
| `SendMessage(conn, content)`                      | Send `{"content": ...}`                                              |
| `CloseWebSocket(conn)`                            | Close cleanly                                                        |
| `MakeRequest(t, method, url)`                     | HTTP request, fully read; returns a `Response` with the body closed  |
| `AssertStatusCode` / `AssertContentType` / `AssertBody` | Common assertions over a `Response`                            |

The `integration` package layers its own helpers on top in `setup_test.go` — `newTestServer` (a
server backed by a hub of its own), `dial` / `dialPair` / `dialClients` (which return only once the
hub has registered every connection), and `waitForUnregister`. Prefer those inside that package:
they make client-count assertions exact.

## Writing tests

Follow the conventions already in the suite:

- Name tests `TestSubjectBehavior` — `TestWebSocketOriginValidation`, `TestHubShutdownTimeout`.
- Put anything that needs a listening socket in `test/integration`, everything else in `test/unit`.
- Use table-driven subtests with `t.Run` for multiple scenarios of one behavior.
- Cover the failure path, not just the happy one — most bugs in this codebase live in error handling
  and shutdown ordering.
- Reset shared state. The active config is a package-level global and `server.GlobalHub()` is
  process-wide; give a test its own hub with `server.SetupRoutesWithHub` (integration tests get this
  from `newTestServer`) and restore the config with `server.SetConfig(nil)` rather than mutating
  global state and leaving it changed.
- Prefer waiting on a channel or polling with a deadline over `time.Sleep` for synchronization. The
  hub's `ClientCount()` is answered by its own event loop, so a reply proves every registration,
  unregistration, and broadcast queued before it has been processed — that is the barrier to wait on,
  via `testhelpers.WaitFor`. A `time.Sleep` is only acceptable when elapsed wall-clock time is the
  behavior under test, such as waiting for the rate limiter to refill; say so in a comment.

### Benchmarks

`make bench` runs `go test -bench=. -benchmem ./...`. The benchmarks live alongside the code they
measure, in `internal/server/*_internal_test.go`, because they exercise unexported hot paths:
broadcast fan-out, message normalization, the rate limiter, and origin checks.

They assert allocation counts implicitly rather than wall-clock time — the numbers move with the
machine, but a path that was allocation-free and stops being so is a regression worth catching. Run
`go test -bench . -benchmem ./internal/...` before and after a change to the hot path and compare
the `allocs/op` column.

## In CI

The `test` job runs `go test -race -shuffle=on -coverprofile=coverage.out -covermode=atomic ./...`
on Ubuntu with the toolchain from `go.mod`. `-shuffle=on` randomizes test order, so a suite that
depends on ordering fails in CI even when it passes locally. A separate `bench` job runs every
benchmark once (`-benchtime=10x`) to keep them compiling. See
[Contributing](../contributing.md#continuous-integration).

## Related

- [Contributing](../contributing.md) — the full pre-push check list
- [Make targets reference](../reference/make-targets.md) — every test target
- [Architecture overview](../architecture/overview.md) — what the concurrency tests are protecting
