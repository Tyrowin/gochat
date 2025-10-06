# GoChat

[![CI Pipeline](https://github.com/Tyrowin/gochat/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Tyrowin/gochat/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Tyrowin/gochat)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/Tyrowin/gochat)](https://goreportcard.com/report/github.com/Tyrowin/gochat)

High-performance, standalone, multi-client chat server built using Go and WebSockets.

## Features

- Real-time WebSocket-based chat communication
- Multi-client support with concurrent connections
- **Cross-platform development support** - Build and develop on Windows, macOS, or Linux
- **Cross-compilation** - Build binaries for any platform from any platform
- Built-in security and vulnerability scanning
- Comprehensive CI/CD pipeline with automated testing
- Static code analysis with golangci-lint
- Dependency vulnerability scanning with govulncheck

## Prerequisites

- Go 1.25.1 or later
- Git
- Make (optional but recommended for easier builds)
  - **Windows**: Install via [Chocolatey](https://chocolatey.org/) (`choco install make`)
  - **macOS**: Install Xcode Command Line Tools (`xcode-select --install`)
  - **Linux**: Usually pre-installed (`apt install make` or `yum install make`)

**Note**: If you don't have Make, you can use Go commands directly (see [Build Guide](docs/BUILD_GUIDE.md)).

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

   **With Make:**

   ```bash
   make build
   ```

   **Or with Go directly:**

   ```bash
   go build -o bin/gochat ./cmd/server
   ```

4. **Run the server**

   **On Windows:**

   ```powershell
   .\bin\gochat.exe
   ```

   **On macOS/Linux:**

   ```bash
   ./bin/gochat
   ```

   **Or using Make:**

   ```bash
   make run
   ```

The server will start on `http://localhost:8080` with the following endpoints:

- `/` - Health check
- `/ws` - WebSocket connection endpoint
- `/test` - Test page for WebSocket functionality

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

### Cross-Platform Development

GoChat supports development and building on **Windows**, **macOS**, and **Linux** with full cross-compilation capabilities. You can build binaries for any platform from any platform.

#### Using Make (Recommended)

The Makefile works on all platforms (requires `make`):

```bash
# Build for current platform
make build-current

# Build for specific platforms
make build-linux           # Linux (amd64)
make build-linux-arm64     # Linux (arm64)
make build-darwin          # macOS Intel
make build-darwin-arm64    # macOS Apple Silicon
make build-windows         # Windows (amd64)

# Build for all platforms
make build-all

# Create optimized release builds
make release

# List all supported platforms
make list-platforms
```

#### Without Make (Direct Go Commands)

If you don't have Make installed, you can use Go directly:

**Windows (PowerShell):**

```powershell
# Build for current platform
go build -o bin\gochat.exe .\cmd\server

# Build for Linux
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin\gochat-linux-amd64 .\cmd\server

# Build for macOS
$env:GOOS="darwin"; $env:GOARCH="arm64"; go build -o bin\gochat-darwin-arm64 .\cmd\server
```

**macOS/Linux:**

```bash
# Build for current platform
go build -o bin/gochat ./cmd/server

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o bin/gochat-windows-amd64.exe ./cmd/server

# Build for macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bin/gochat-darwin-arm64 ./cmd/server
```

#### Cross-Compilation Examples

Go's cross-compilation support means you can build for any platform from any platform:

**From Windows → Build for Linux:**

```bash
make build-linux
```

**From macOS → Build for Windows:**

```bash
make build-windows
```

**From Linux → Build for macOS (Apple Silicon):**

```bash
make build-darwin-arm64
```

#### Understanding the Build Output

After building, binaries are organized in the `./bin` directory:

```
bin/
├── gochat                      # Current platform binary (from `make build`)
├── linux/
│   ├── gochat-amd64           # Linux 64-bit
│   ├── gochat-arm64           # Linux ARM64 (Raspberry Pi, AWS Graviton)
│   └── checksums.txt          # SHA256 checksums (from `make release`)
├── darwin/
│   ├── gochat-amd64           # macOS Intel
│   ├── gochat-arm64           # macOS Apple Silicon (M1/M2/M3)
│   └── checksums.txt          # SHA256 checksums (from `make release`)
└── windows/
    ├── gochat-amd64.exe       # Windows 64-bit
    └── checksums.txt          # SHA256 checksums (from `make release`)
```

**Note:** Platform-specific build targets create binaries in their respective subdirectories, making it easier to manage and distribute builds for different platforms.

#### Platform-Specific Code

If you need to write platform-specific code, Go provides several approaches:

**File Name Suffixes:**

```
config_windows.go   # Only compiled on Windows
config_linux.go     # Only compiled on Linux
config_darwin.go    # Only compiled on macOS
```

**Build Tags:**

```go
//go:build linux && amd64

package mypackage

// This code only compiles for 64-bit Linux
```

**Runtime Detection:**

```go
import "runtime"

if runtime.GOOS == "windows" {
    // Windows-specific code
} else if runtime.GOOS == "darwin" {
    // macOS-specific code
}
```

#### CGo Considerations

This project is built with `CGO_ENABLED=0` for maximum portability and easier cross-compilation. The binaries are completely self-contained with no external dependencies.

If you need CGo for a specific feature, you'll need to set up cross-compilation toolchains for each target platform.

### Local Build (Traditional)

```bash
make build
```

### Release Build

Create optimized builds for multiple platforms:

```bash
make release
```

This creates production-ready binaries for:

- Linux (amd64 and arm64)
- macOS (Intel and Apple Silicon)
- Windows (amd64)

All binaries are:

- Optimized with `-trimpath` for reproducible builds
- Built with `CGO_ENABLED=0` for static linking
- Accompanied by SHA256 checksums

**For detailed cross-platform development instructions, see [Cross-Platform Development Guide](docs/CROSS_PLATFORM.md).**

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
│   └── server/          # Core HTTP/WebSocket server components
│       ├── client.go        # WebSocket client connection management
│       ├── config.go        # Runtime configuration and security controls
│       ├── handlers.go      # HTTP and WebSocket request handlers
│       ├── hub.go           # Client registry and broadcast coordination
│       ├── http_server.go   # HTTP server setup helpers
│       ├── origin.go        # Origin validation helpers
│       ├── rate_limiter.go  # Per-connection rate limiting
│       ├── routes.go        # Route registration
│       └── types.go         # Shared message and utility types
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
