# GoChat

[![CI Pipeline](https://github.com/Tyrowin/gochat/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Tyrowin/gochat/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Tyrowin/gochat)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/Tyrowin/gochat)](https://goreportcard.com/report/github.com/Tyrowin/gochat)

A high-performance, production-ready WebSocket chat server built with Go. GoChat provides real-time multi-client communication with built-in security features, comprehensive testing, and cross-platform support.

## Features

- **Real-time Communication** - WebSocket-based instant messaging
- **Multi-client Support** - Handle thousands of concurrent connections
- **Built-in Security** - Origin validation, rate limiting, and message size limits
- **Production Ready** - Comprehensive testing, CI/CD pipeline, and deployment guides
- **Cross-platform** - Build and run on Windows, macOS, and Linux
- **Zero Dependencies** - Statically linked binaries with no external runtime dependencies
- **Easy Deployment** - Simple binary deployment with reverse proxy support

## Quick Start

```bash
# Clone the repository
git clone https://github.com/Tyrowin/gochat.git
cd gochat

# Build the server
make build

# Run the server
./bin/gochat
```

The server starts on `http://localhost:8080`. Visit `http://localhost:8080/test` to try the interactive test page.

## Documentation

### Getting Started
- **[Getting Started Guide](docs/GETTING_STARTED.md)** - Installation, building, and running the server
- **[API Documentation](docs/API.md)** - WebSocket API reference and code examples

### Deployment
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment with Nginx/Caddy, TLS/WSS setup, and process management
- **[Security Documentation](docs/SECURITY.md)** - Security features, configuration, and best practices

### Development
- **[Development Guide](docs/DEVELOPMENT.md)** - Development setup, testing, and CI/CD
- **[Building Guide](docs/BUILDING.md)** - Build instructions and cross-compilation
- **[Contributing Guide](docs/CONTRIBUTING.md)** - How to contribute to the project

## Architecture

### Simple and Focused

GoChat follows a clean, modular architecture:

```
Client (Browser/App) 
    ↓ WebSocket (ws:// or wss://)
Reverse Proxy (Nginx/Caddy) 
    ↓ HTTP
GoChat Server (Go)
    ├── Hub (Message Broker)
    ├── Clients (WebSocket Connections)
    └── Security (Rate Limiting, Origin Validation)
```

### Key Components

- **Hub** - Central message broker coordinating all connected clients
- **Client** - WebSocket connection handler with read/write pumps
- **Rate Limiter** - Token bucket per-connection rate limiting
- **Origin Validator** - CSWSH attack prevention
- **Handlers** - HTTP/WebSocket request handlers

See [Development Guide](docs/DEVELOPMENT.md#project-structure) for detailed architecture information.

## Technology Stack

- **Language:** Go 1.25.1+
- **WebSocket Library:** [gorilla/websocket](https://github.com/gorilla/websocket)
- **Testing:** Go standard library + custom test helpers
- **CI/CD:** GitHub Actions
- **Code Quality:** golangci-lint, gosec, govulncheck

## Project Status

GoChat is actively maintained and production-ready. We welcome contributions!

- **Stability:** Stable, used in production
- **Test Coverage:** 80%+ with unit and integration tests
- **Security:** Regular dependency scanning and security audits
- **Performance:** Handles thousands of concurrent connections

## Use Cases

- **Chat Applications** - Real-time messaging systems
- **Live Notifications** - Push notifications to web clients
- **Collaborative Tools** - Real-time collaboration features
- **Gaming** - Multiplayer game communication
- **IoT** - Device-to-server real-time communication
- **Monitoring Dashboards** - Live data updates

## Community and Support

- **Issues:** [GitHub Issues](https://github.com/Tyrowin/gochat/issues) - Bug reports and feature requests
- **Discussions:** [GitHub Discussions](https://github.com/Tyrowin/gochat/discussions) - Questions and community chat
- **Contributing:** See [Contributing Guide](docs/CONTRIBUTING.md)
- **Security:** See [Security Policy](docs/SECURITY.md#reporting-security-issues)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [gorilla/websocket](https://github.com/gorilla/websocket)
- Inspired by the Go community's best practices
- Thanks to all contributors

---

**Ready to get started?** Check out the [Getting Started Guide](docs/GETTING_STARTED.md) or explore the [API Documentation](docs/API.md) to integrate GoChat into your application.
