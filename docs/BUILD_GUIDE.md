# GoChat Build Guide - Quick Reference

## Quick Commands

### Build for Current Platform

**With Make:**

```bash
make build
```

**With Go:**

```powershell
# Windows
go build -o bin\gochat.exe .\cmd\server

# macOS/Linux
go build -o bin/gochat ./cmd/server
```

### Cross-Compile

**Windows → Linux:**

```powershell
# With Make
make build-linux

# With Go
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin\gochat-linux-amd64 .\cmd\server
```

**macOS/Linux → Windows:**

```bash
# With Make
make build-windows

# With Go
GOOS=windows GOARCH=amd64 go build -o bin/gochat-windows-amd64.exe ./cmd/server
```

### Build All Platforms

```bash
make build-all
```

This creates organized binaries in platform-specific directories:

**Linux binaries:**

- `bin/linux/gochat-amd64`
- `bin/linux/gochat-arm64`

**macOS binaries:**

- `bin/darwin/gochat-amd64` (Intel)
- `bin/darwin/gochat-arm64` (Apple Silicon)

**Windows binaries:**

- `bin/windows/gochat-amd64.exe`

### Create Release

```bash
make release
```

Creates all platform binaries plus SHA256 checksums.

## Development

### Hot Reload

```bash
make dev
# or
air
```

### Run Tests

```bash
make test
```

### Lint & Format

```bash
make fmt
make lint
```

### Full CI Check

```bash
make ci-local
```

## VS Code

Press **Ctrl+Shift+B** (Windows/Linux) or **Cmd+Shift+B** (macOS) to build.

## Supported Platforms

| Platform            | GOOS    | GOARCH | Make Target          |
| ------------------- | ------- | ------ | -------------------- |
| Windows 64-bit      | windows | amd64  | `build-windows`      |
| Linux 64-bit        | linux   | amd64  | `build-linux`        |
| Linux ARM64         | linux   | arm64  | `build-linux-arm64`  |
| macOS Intel         | darwin  | amd64  | `build-darwin`       |
| macOS Apple Silicon | darwin  | arm64  | `build-darwin-arm64` |

View all: `make list-platforms` or `go tool dist list`

## More Information

- [Cross-Platform Development Guide](CROSS_PLATFORM.md) - Detailed guide
- [Main README](../README.md) - Project overview
- [Go Cross-Compilation](https://golang.org/doc/install/source#environment) - Official docs
