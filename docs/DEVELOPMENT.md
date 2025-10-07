# Development Guide

This guide covers setting up a development environment, running tests, and contributing to GoChat.

## Table of Contents

- [Development Setup](#development-setup)
- [Development Tools](#development-tools)
- [Hot Reload Development](#hot-reload-development)
- [Code Quality](#code-quality)
- [Testing](#testing)
- [CI/CD Pipeline](#cicd-pipeline)
- [Project Structure](#project-structure)

## Development Setup

### Prerequisites

- Go 1.25.1 or later
- Git
- Make (optional but recommended)

### Clone and Setup

```bash
# Clone repository
git clone https://github.com/Tyrowin/gochat.git
cd gochat

# Install development tools
make install-tools

# Verify setup
go version
make help
```

## Development Tools

### Install All Tools

```bash
make install-tools
```

This installs:

- **golangci-lint** - Static code analysis and linting
- **govulncheck** - Go vulnerability database scanner
- **gosec** - Security-focused static analyzer
- **goimports** - Import organization and formatting
- **air** - Live reload for rapid development

### Individual Tool Installation

**golangci-lint:**

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**govulncheck:**

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

**gosec:**

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

**goimports:**

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

**air:**

```bash
go install github.com/cosmtrek/air@latest
```

## Hot Reload Development

For rapid development with automatic reloading on file changes:

```bash
make dev
```

This uses [Air](https://github.com/cosmtrek/air) to:

- Watch for file changes
- Automatically rebuild the application
- Restart the server
- Preserve the terminal output

**Manual air configuration** (`.air.toml`):

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/server"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "bin"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

## Code Quality

### Available Make Targets

View all available commands:

```bash
make help
```

### Common Development Tasks

**Code Formatting:**

```bash
make fmt
```

- Runs `gofmt` on all Go files
- Organizes imports with `goimports`
- Ensures consistent code style

**Linting:**

```bash
make lint
```

- Runs `golangci-lint` with comprehensive rule set
- Checks code quality, style, and potential bugs
- Configuration in `.golangci.yml`

**Security Scanning:**

```bash
make security-scan
```

- Runs `govulncheck` for dependency vulnerabilities
- Runs `gosec` for security issues in code

**Dependency Management:**

```bash
# Check for dependency issues
make deps-check

# Update dependencies
make deps-update

# Verify licenses
make license-check
```

### Code Quality Standards

**Formatting Rules:**

- Standard Go formatting (`gofmt`)
- Organized imports (stdlib, external, internal)
- Maximum line length: 120 characters (recommended)

**Documentation:**

- All exported functions must have doc comments
- Package-level documentation in `doc.go` files
- Comments should explain "why", not "what"

**Error Handling:**

- Always check errors
- Provide context with error messages
- Use `fmt.Errorf` with `%w` for error wrapping

**Testing:**

- Unit tests for all new functionality
- Table-driven tests where applicable
- Test edge cases and error conditions
- Maintain or improve code coverage

## Testing

### Run All Tests

```bash
make test
```

This runs:

- All unit tests
- All integration tests
- Race condition detection enabled
- Verbose output

### Test with Coverage

```bash
make test-coverage
```

This generates:

- Coverage report in `coverage.out`
- HTML coverage visualization (opens in browser)

**View coverage manually:**

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Tests

**Unit tests only:**

```bash
go test -v -race ./test/unit
```

**Integration tests only:**

```bash
go test -v -race ./test/integration
```

**Specific test file:**

```bash
go test -v -race ./test/unit/handlers_test.go
```

**Specific test function:**

```bash
go test -v -race -run TestWebSocketHandler ./test/unit
```

### Test Structure

```
test/
├── integration/          # Integration tests
│   ├── multiclient_test.go
│   ├── security_test.go
│   ├── server_test.go
│   ├── shutdown_test.go
│   └── websocket_test.go
├── unit/                # Unit tests
│   ├── error_handling_test.go
│   ├── handlers_test.go
│   ├── hub_test.go
│   └── websocket_test.go
└── testhelpers/         # Shared test utilities
    └── helpers.go
```

### Writing Tests

**Example unit test:**

```go
func TestNewClient(t *testing.T) {
    hub := NewHub()
    conn := &mockWebSocketConn{}

    client := NewClient(conn, hub, "127.0.0.1:1234")

    if client == nil {
        t.Fatal("Expected client to be created")
    }
    if client.conn != conn {
        t.Error("Client connection not set correctly")
    }
}
```

**Example integration test:**

```go
func TestMultiClientBroadcast(t *testing.T) {
    server := testhelpers.StartTestServer(t)
    defer server.Close()

    // Connect multiple clients
    client1 := testhelpers.ConnectClient(t, server.URL)
    client2 := testhelpers.ConnectClient(t, server.URL)

    // Send message from client1
    msg := Message{Content: "Hello"}
    client1.WriteJSON(msg)

    // Verify client2 receives it
    var received Message
    if err := client2.ReadJSON(&received); err != nil {
        t.Fatalf("Failed to receive message: %v", err)
    }

    if received.Content != msg.Content {
        t.Errorf("Expected %q, got %q", msg.Content, received.Content)
    }
}
```

### Benchmarks

Run benchmarks to measure performance:

```bash
make bench
```

**Write benchmarks:**

```go
func BenchmarkMessageBroadcast(b *testing.B) {
    hub := NewHub()
    // Setup...

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        hub.broadcast <- message
    }
}
```

### Race Detection

Always run tests with race detection:

```bash
go test -race ./...
```

The CI pipeline automatically runs tests with `-race` flag.

## CI/CD Pipeline

### GitHub Actions

The project uses GitHub Actions for continuous integration. Pipeline runs on:

- Every push to any branch
- Every pull request

### Pipeline Stages

1. **Code Formatting** - Verify `gofmt` compliance
2. **Static Analysis** - Run `golangci-lint`
3. **Security Scanning** - Run `govulncheck` and `gosec`
4. **Unit Tests** - Run all tests with race detection
5. **Integration Tests** - Run integration test suite
6. **Coverage Check** - Ensure minimum coverage
7. **Dependency Check** - Verify dependencies are up to date
8. **Build Verification** - Build for multiple platforms

### Run CI Locally

Before pushing code, run the same checks locally:

```bash
make ci-local
```

This runs:

- Code formatting check
- Linting
- Security scans
- All tests with race detection
- Dependency checks
- Build verification

**Individual CI steps:**

```bash
# Format check
make fmt-check

# Lint
make lint

# Security
make security-scan

# Tests
make test

# Dependencies
make deps-check

# Build
make build
```

### CI Configuration

**File:** `.github/workflows/ci.yml`

Key features:

- Runs on Ubuntu latest
- Tests against Go 1.25.1
- Caches dependencies for speed
- Uploads test coverage
- Builds for multiple platforms

## Project Structure

```
gochat/
├── cmd/
│   └── server/              # Application entry point
│       └── main.go          # Server initialization and graceful shutdown
├── internal/
│   └── server/              # Core server implementation
│       ├── client.go        # WebSocket client lifecycle
│       ├── config.go        # Server configuration
│       ├── handlers.go      # HTTP/WebSocket handlers
│       ├── hub.go           # Client registry and broadcasting
│       ├── http_server.go   # HTTP server setup
│       ├── origin.go        # Origin validation
│       ├── rate_limiter.go  # Rate limiting
│       ├── routes.go        # Route registration
│       └── types.go         # Shared types
├── test/
│   ├── integration/         # Integration tests
│   ├── unit/               # Unit tests
│   └── testhelpers/        # Test utilities
├── docs/                   # Documentation
├── .github/
│   └── workflows/
│       └── ci.yml          # CI/CD pipeline
├── .golangci.yml           # Linter configuration
├── Makefile                # Build automation
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
└── README.md               # Project overview
```

### Key Components

**client.go:**

- Manages individual WebSocket connections
- Implements read/write pumps
- Handles message validation
- Rate limiting per connection

**hub.go:**

- Central message broker
- Client registry
- Broadcast coordination
- Connection lifecycle management

**handlers.go:**

- HTTP request handlers
- WebSocket upgrade logic
- Health check endpoint
- Test page serving

**rate_limiter.go:**

- Token bucket implementation
- Per-connection rate limiting
- Configurable limits

**origin.go:**

- Origin header validation
- CSWSH attack prevention
- Configurable allowed origins

## Development Workflow

### Standard Workflow

1. **Create a branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes**

   - Write code
   - Add tests
   - Update documentation

3. **Test locally**

   ```bash
   make ci-local
   ```

4. **Commit changes**

   ```bash
   git add .
   git commit -m "Add feature: description"
   ```

5. **Push to GitHub**

   ```bash
   git push origin feature/your-feature-name
   ```

6. **Open Pull Request**
   - Describe changes
   - Link related issues
   - Wait for CI to pass

### Pre-commit Checklist

- [ ] Code is formatted (`make fmt`)
- [ ] Linting passes (`make lint`)
- [ ] Tests pass (`make test`)
- [ ] Security scans pass (`make security-scan`)
- [ ] Documentation updated
- [ ] No debug code left in
- [ ] Commit message is clear

### Debugging

**Enable verbose logging:**

```go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

**Use delve debugger:**

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server
```

**VS Code debugging:**

`.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server"
    }
  ]
}
```

## Performance Profiling

### CPU Profiling

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

### Memory Profiling

```bash
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

### Runtime Profiling

Add to your code:

```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Access profiles at `http://localhost:6060/debug/pprof/`

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Installation and setup
- [Building](BUILDING.md) - Build and cross-compilation
- [Contributing](CONTRIBUTING.md) - Contribution guidelines
- [API Documentation](API.md) - WebSocket API
- [Security](SECURITY.md) - Security features
