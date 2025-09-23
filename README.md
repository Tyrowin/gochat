# GoChat

[![CI Pipeline](https://github.com/Tyrowin/gochat/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Tyrowin/gochat/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Tyrowin/gochat)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/Tyrowin/gochat)](https://goreportcard.com/report/github.com/Tyrowin/gochat)

High-performance, standalone, multi-client chat server built using Go and WebSockets.

## Features

- Real-time WebSocket-based chat communication
- Multi-client support with concurrent connections
- Built-in security and vulnerability scanning
- Comprehensive CI/CD pipeline with automated testing
- Static code analysis with golangci-lint
- Dependency vulnerability scanning with govulncheck

## Prerequisites

- Go 1.25.1 or later
- Git
- Make (for using the Makefile)

## Quick Start

1. **Clone the repository**

   ```bash
   git clone https://github.com/Tyrowin/gochat.git
   cd gochat
   ```

2. **Install development tools**

   ```bash
   make install-tools
   ```

3. **Build the application**

   ```bash
   make build
   ```

4. **Run the server**
   ```bash
   make run
   ```

## Development

### Prerequisites for Development

Install the required development tools:

```bash
make install-tools
```

This will install:

- `golangci-lint` - Static code analysis
- `govulncheck` - Vulnerability scanner
- `gosec` - Security analyzer
- `goimports` - Import formatter
- `air` - Live reload for development

### Available Make Targets

Run `make help` to see all available targets:

```bash
make help
```

### Common Development Commands

- **Format and lint code**: `make fmt lint`
- **Run tests**: `make test`
- **Run tests with coverage**: `make test-coverage`
- **Security scanning**: `make security-scan`
- **Check dependencies**: `make deps-check`
- **Development mode with auto-reload**: `make dev`
- **Run full CI pipeline locally**: `make ci-local`

### Code Quality and Security

This project uses several tools to ensure code quality, security, and best practices:

#### Static Analysis

- **golangci-lint**: Comprehensive Go linter with multiple analyzers
  ```bash
  make lint
  ```

#### Security Scanning

- **govulncheck**: Official Go vulnerability scanner
- **gosec**: Security analyzer for Go code
  ```bash
  make security-scan
  ```

#### Dependency Management

- **Vulnerability checking**: Automated scanning of dependencies
- **License compliance**: Check licenses of all dependencies
  ```bash
  make deps-check
  make license-check
  ```

### Configuration Files

- **`.golangci.yml`**: golangci-lint configuration with enabled linters and rules
- **`.github/workflows/ci.yml`**: GitHub Actions CI/CD pipeline
- **`Makefile`**: Development and build automation

### CI/CD Pipeline

The project includes a comprehensive GitHub Actions CI/CD pipeline that runs on every push and pull request:

1. **Build and Test**: Compiles code and runs all tests with coverage
2. **Static Analysis**: Runs golangci-lint with comprehensive rule set
3. **Security Scan**: Vulnerability scanning with govulncheck and Nancy
4. **Dependency Check**: Validates dependencies and checks for updates
5. **Multi-version Build**: Tests against multiple Go versions
6. **Docker Security**: Scans Docker images with Trivy (if applicable)
7. **Quality Gate**: Ensures all checks pass before allowing merges

### Running CI Pipeline Locally

To run the same checks that run in CI:

```bash
make ci-local
```

This will run:

- Code formatting
- Linting
- Security scanning
- Tests with coverage
- Dependency checks
- Build verification

### Performance and Benchmarking

- **Run benchmarks**: `make bench`
- **Race condition detection**: `make race`
- **Generate dependency graph**: `make deps-graph`

## Building and Deployment

### Local Build

```bash
make build
```

### Release Build

Create optimized builds for multiple platforms:

```bash
make release
```

### Docker

```bash
# Build Docker image
make docker-build

# Run in Docker container
make docker-run
```

## Project Structure

```
gochat/
├── cmd/
│   └── server/          # Application entry point
│       └── main.go
├── internal/            # Private application code
│   ├── client/          # Client connection handling
│   └── hub/             # Chat hub and message routing
├── .github/
│   └── workflows/
│       └── ci.yml       # GitHub Actions CI pipeline
├── .golangci.yml        # Linter configuration
├── Makefile            # Build and development automation
├── go.mod              # Go module definition
└── README.md           # This file
```

## Code Quality Standards

This project enforces the following standards:

- **Go formatting**: Standard `gofmt` formatting
- **Import organization**: Organized with `goimports`
- **Linting**: Comprehensive linting with golangci-lint
- **Security**: Regular security scanning with multiple tools
- **Testing**: Minimum test coverage requirements
- **Documentation**: All exported functions and types must be documented

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Run the full CI pipeline locally (`make ci-local`)
4. Commit your changes (`git commit -am 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

### Before Submitting a PR

Ensure your code passes all quality checks:

```bash
make ci-local
```

This will run all the same checks that run in the CI pipeline.

## Security

- All dependencies are automatically scanned for vulnerabilities
- Security issues are tracked and remediated promptly
- Code is analyzed for security vulnerabilities using gosec
- Docker images (if used) are scanned with Trivy

To report security vulnerabilities, please create a private issue or contact the maintainers directly.

## License

This project is licensed under the terms specified in the [LICENSE](LICENSE) file.
