# Cross-Platform Development Guide for GoChat

Go's built-in cross-compilation support makes it easy to build GoChat for any platform from any platform.

## Prerequisites

- Go 1.25.1 or later
- Git
- Make (optional, but recommended)
  - **Windows**: Install via [Chocolatey](https://chocolatey.org/) (`choco install make`)
  - **macOS**: Included with Xcode Command Line Tools
  - **Linux**: Usually pre-installed (`apt install make` / `yum install make`)

## Building

### With Make (Recommended)

```bash
# Build for current platform
make build
# or
make build-current

# Build for specific platforms
make build-linux           # Linux (amd64)
make build-linux-arm64     # Linux (ARM64)
make build-darwin          # macOS (Intel)
make build-darwin-arm64    # macOS (Apple Silicon)
make build-windows         # Windows (amd64)

# Build for all platforms
make build-all

# Create optimized release builds
make release

# List all supported platforms
make list-platforms
```

### Without Make (Direct Go Commands)

**Windows (PowerShell):**

```powershell
# Build for current platform
go build -o bin\gochat.exe .\cmd\server

# Cross-compile for Linux
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin\gochat-linux-amd64 .\cmd\server

# Cross-compile for macOS
$env:GOOS="darwin"; $env:GOARCH="arm64"; go build -o bin\gochat-darwin-arm64 .\cmd\server
```

**macOS/Linux:**

```bash
# Build for current platform
go build -o bin/gochat ./cmd/server

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o bin/gochat-windows-amd64.exe ./cmd/server

# Cross-compile for macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bin/gochat-darwin-arm64 ./cmd/server
```

## Cross-Compilation

Build for any platform from any platform:

| From    | To      | Command                   |
| ------- | ------- | ------------------------- |
| Windows | Linux   | `make build-linux`        |
| Windows | macOS   | `make build-darwin-arm64` |
| macOS   | Windows | `make build-windows`      |
| macOS   | Linux   | `make build-linux`        |
| Linux   | Windows | `make build-windows`      |
| Linux   | macOS   | `make build-darwin-arm64` |

## Build Output

Binaries are organized in platform-specific directories within `bin/`:

```
bin/
├── gochat                    # Current platform binary (from `make build`)
├── linux/
│   ├── gochat-amd64          # Linux 64-bit
│   ├── gochat-arm64          # Linux ARM64 (Raspberry Pi, AWS Graviton)
│   └── checksums.txt         # SHA256 checksums (from `make release`)
├── darwin/
│   ├── gochat-amd64          # macOS Intel
│   ├── gochat-arm64          # macOS Apple Silicon (M1/M2/M3)
│   └── checksums.txt         # SHA256 checksums (from `make release`)
└── windows/
    ├── gochat-amd64.exe      # Windows 64-bit
    └── checksums.txt         # SHA256 checksums (from `make release`)
```

**Note:**

- `make build` creates the binary for your current platform in `bin/`
- Platform-specific targets (`make build-linux`, `make build-windows`, etc.) create binaries in their respective directories
- `make build-all` builds for all platforms at once

## Platform-Specific Code

### File Name Suffixes

Go automatically selects the right file based on the target platform:

```
internal/server/
├── config.go              # Shared code
├── config_windows.go      # Windows-specific
├── config_darwin.go       # macOS-specific
└── config_linux.go        # Linux-specific
```

Example `config_windows.go`:

```go
//go:build windows

package server

func getPlatformConfig() string {
    return "Windows configuration"
}
```

### Build Tags

For more complex conditions:

```go
//go:build linux && amd64

package server

// This code only compiles for 64-bit Linux
```

```go
//go:build darwin || linux

package server

// This code compiles on both macOS and Linux
```

```go
//go:build !windows

package server

// This code compiles on all platforms except Windows
```

### Runtime Detection

Check the platform at runtime:

```go
package server

import (
    "runtime"
    "path/filepath"
)

func getPlatformPath() string {
    switch runtime.GOOS {
    case "windows":
        return filepath.Join("C:", "Program Files", "gochat")
    case "darwin":
        return filepath.Join("/Applications", "gochat")
    case "linux":
        return filepath.Join("/usr", "local", "bin", "gochat")
    default:
        return "./gochat"
    }
}
```

### Cross-Platform File Paths

Always use the `path/filepath` package:

```go
import "path/filepath"

// GOOD - Works on all platforms
configPath := filepath.Join("config", "app.json")

// BAD - Only works on Unix-like systems
configPath := "config/app.json"

// BAD - Only works on Windows
configPath := "config\\app.json"
```

## Development

### Hot Reload with Air

```bash
# Install air (if not already installed)
go install github.com/air-verse/air@latest

# Run with hot reload
make dev
# or
air
```

The `.air.toml` configuration works across all platforms.

### VS Code Integration

Press `Ctrl+Shift+B` (or `Cmd+Shift+B` on macOS) to access build tasks:

- Build (Current Platform)
- Build All Platforms
- Run Server
- Dev (Hot Reload)
- Test All
- Create Release

## Troubleshooting

### Windows

**"make: command not found"**

```powershell
# Install Make via Chocolatey
choco install make

# Or use Go directly
go build -o bin\gochat.exe .\cmd\server
```

**"Scripts are disabled on this system"**

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### macOS

**"Permission denied"**

```bash
chmod +x bin/gochat
```

**"Developer cannot be verified"**

- Go to System Preferences > Security & Privacy and allow the binary
- Or remove quarantine: `xattr -d com.apple.quarantine bin/gochat`

### Linux

**"Go not found"**

```bash
# Add Go to PATH
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### All Platforms

**Build is slow**

```bash
# Use cached builds
go build -i -o bin/gochat ./cmd/server

# Or clean cache if corrupted
go clean -cache -modcache -i -r
```

## Best Practices

1. ✅ Use `filepath.Join()` for all file paths
2. ✅ Set `CGO_ENABLED=0` for portable binaries (already configured)
3. ✅ Use build tags for platform-specific code
4. ✅ Test on multiple platforms when possible
5. ✅ Use the Makefile for consistent builds
6. ✅ Check the `.gitattributes` file handles line endings correctly

## Resources

- [Go Cross-Compilation](https://golang.org/doc/install/source#environment)
- [Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Main README](../README.md)

## Summary

**With Make:**

```bash
make build          # Current platform
make build-all      # All platforms
make release        # Release builds with checksums
```

**Without Make:**

```bash
# Current platform
go build -o bin/gochat ./cmd/server

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o bin/gochat-linux-amd64 ./cmd/server
```

That's it! Go's cross-compilation makes building for any platform simple and straightforward.
