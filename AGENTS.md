# Agent Instructions for GoChat

## Project Overview

GoChat is a high-performance, production-ready WebSocket chat server built with Go. It provides real-time multi-client communication with built-in security features, comprehensive testing, and cross-platform support.

**Key Technologies:**
- Go 1.25.1+
- WebSocket (gorilla/websocket)
- GitHub Actions for CI/CD
- Docker for containerization

## Code Standards

### Go Style Guidelines

- **Formatting:** Follow standard Go formatting (`gofmt`)
- **Imports:** Organize imports in order: stdlib, external packages, internal packages
- **Line length:** Maximum 120 characters (recommended)
- **Naming conventions:** Use idiomatic Go naming (camelCase for private, PascalCase for exported)
- **Comments:** 
  - All exported functions, types, and constants must have doc comments
  - Package-level documentation should be in `doc.go` files
  - Comments should explain "why", not "what"

### Error Handling

- Always check and handle errors
- Provide context with error messages using `fmt.Errorf` with `%w` for error wrapping
- Don't ignore errors with blank identifiers unless absolutely necessary
- Return errors instead of panicking in library code

### Testing Requirements

- **Unit tests** are required for all new functionality
- Use **table-driven tests** where applicable
- Test edge cases and error conditions
- Maintain or improve code coverage (current target: 80%+)
- Tests must pass race detection (`go test -race`)
- Integration tests for WebSocket functionality

### Security Practices

- Never commit secrets or credentials
- Always validate user input
- Implement proper rate limiting for WebSocket connections
- Validate origins for CORS protection
- Run security scans before committing (`make security-scan`)

## Development Workflow

### Before Making Changes

1. **Format code:** `make fmt`
2. **Run linter:** `make lint`
3. **Run tests:** `make test`
4. **Security scan:** `make security-scan`
5. **Full CI check:** `make ci-local`

### Code Organization

```
├── cmd/server/          # Main application entry point
├── internal/server/     # Core server implementation
│   ├── hub.go          # Central message broker
│   ├── client.go       # WebSocket client handler
│   ├── handlers.go     # HTTP/WebSocket handlers
│   ├── rate_limiter.go # Rate limiting implementation
│   └── ...
├── test/               # Test suites
│   ├── unit/          # Unit tests
│   └── integration/   # Integration tests
└── docs/              # Documentation
```

### Building and Testing

**Quick commands:**
- Build: `make build`
- Test all: `make test`
- Test with coverage: `make test-coverage`
- Run locally: `make run`
- Development mode: `make dev` (with auto-reload using Air)

**CI/CD Pipeline:**
- Runs on every push and pull request
- Includes: formatting, linting, security scans, tests, build verification
- All checks must pass before merging

### Project-Specific Patterns

#### WebSocket Message Handling

- Messages are broadcast through the Hub
- Each client has read/write pumps running in separate goroutines
- Rate limiting is applied per connection using token bucket algorithm
- Maximum message size is configurable (default: 512 bytes)

#### Configuration

- Use environment variables for configuration
- Default values should be sensible and secure
- Document all configuration options in `.env.example`
- See `internal/server/config.go` for configuration structure

#### Concurrency

- Use channels for goroutine communication
- Properly close channels when done
- Use context for cancellation and timeouts
- Always protect shared state with mutexes

## Make Targets Reference

### Development
- `make help` - Show all available commands
- `make build` - Build for current platform (includes fmt and vet)
- `make test` - Run all tests with race detection
- `make test-coverage` - Generate coverage report
- `make run` - Build and run the application
- `make dev` - Run with auto-reload (Air)

### Code Quality
- `make fmt` - Format all Go code
- `make lint` - Run golangci-lint
- `make lint-fix` - Run linter with auto-fix
- `make security-scan` - Run govulncheck and gosec
- `make vet` - Run go vet

### CI/CD
- `make ci-local` - Run full CI pipeline locally
- `make all` - Run all checks and build

### Cross-Platform
- `make build-all` - Build for all platforms
- `make release` - Create optimized release builds

## Documentation

- **Getting Started:** `docs/GETTING_STARTED.md`
- **API Reference:** `docs/API.md`
- **Development Guide:** `docs/DEVELOPMENT.md`
- **Contributing:** `docs/CONTRIBUTING.md`
- **Building:** `docs/BUILDING.md`
- **Deployment:** `docs/DEPLOYMENT.md`
- **Security:** `docs/SECURITY.md`

## Common Patterns to Follow

### Adding a New Handler

1. Define handler function in `internal/server/handlers.go`
2. Register route in `internal/server/routes.go`
3. Add tests in `test/integration/` or `test/unit/`
4. Update API documentation in `docs/API.md`

### Adding Configuration

1. Add field to `Config` struct in `internal/server/config.go`
2. Set default value in `DefaultConfig()`
3. Add environment variable loading in `LoadConfig()`
4. Document in `.env.example`
5. Update `docs/DEPLOYMENT.md` configuration section

### Writing Tests

```go
// Table-driven test example
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Dependencies

- Prefer standard library where possible
- Only add new dependencies if absolutely necessary
- Run `make deps-check` before updating dependencies
- Keep dependencies up to date for security patches

## Pre-commit Checklist

When suggesting code changes, ensure:

- [ ] Code is formatted (`make fmt`)
- [ ] Linting passes (`make lint`)
- [ ] Tests pass (`make test`)
- [ ] Security scans pass (`make security-scan`)
- [ ] Documentation is updated
- [ ] No debug code or commented code left in
- [ ] Commit messages are clear and descriptive

## Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

**Types:** feat, fix, docs, style, refactor, test, chore

**Example:**
```
feat: Add rate limiting per user ID

- Implement user-based rate limiting
- Add configuration for rate limits
- Update tests and documentation
- Closes #123
```

## Important Notes

- **Cross-platform support:** Code must work on Windows, macOS, and Linux
- **WebSocket library:** Use gorilla/websocket for all WebSocket operations
- **Statically linked binaries:** Always build with `CGO_ENABLED=0`
- **Production-ready:** Consider performance, security, and reliability in all changes
