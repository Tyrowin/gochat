# Cross-Platform Development Guide for GoChat

This guide explains how to develop, build, and deploy GoChat across Windows, macOS, and Linux platforms.

## Table of Contents

- [Overview](#overview)
- [Development Setup](#development-setup)
- [Building](#building)
- [Cross-Compilation](#cross-compilation)
- [Platform-Specific Code](#platform-specific-code)
- [Troubleshooting](#troubleshooting)

## Overview

GoChat is designed to work seamlessly across all major operating systems. Thanks to Go's excellent cross-platform support, you can:

- Develop on any platform (Windows, macOS, or Linux)
- Build binaries for any platform from any platform
- Use platform-specific build scripts or a unified Makefile
- Run the same tests and quality checks on all platforms

## Development Setup

### Windows

1. **Install Go**: Download from [golang.org](https://golang.org/dl/)
2. **Install Git**: Download from [git-scm.com](https://git-scm.com/)
3. **Optional - Install Make**:
   - Via Chocolatey: `choco install make`
   - Or download GnuWin32 Make

**PowerShell Setup:**

```powershell
# Clone the repository
git clone https://github.com/Tyrowin/gochat.git
cd gochat

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/air-verse/air@latest

# Build the project
.\build.ps1 current

# Run the server
.\bin\gochat.exe
```

**Development with Hot Reload:**

```powershell
# Install air if not already installed
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

### macOS

1. **Install Go**: Download from [golang.org](https://golang.org/dl/) or use Homebrew: `brew install go`
2. **Install Git**: Comes with Xcode Command Line Tools: `xcode-select --install`
3. **Install Make**: Comes with Xcode Command Line Tools

**Terminal Setup:**

```bash
# Clone the repository
git clone https://github.com/Tyrowin/gochat.git
cd gochat

# Make scripts executable
chmod +x scripts/build.sh quick-build.sh

# Install development tools
make install-tools

# Build the project
./build.sh current

# Run the server
./bin/gochat
```

**Development with Hot Reload:**

```bash
# Run with hot reload
make dev
# or
air
```

### Linux

1. **Install Go**: Use your package manager or download from [golang.org](https://golang.org/dl/)
   - Ubuntu/Debian: `sudo apt install golang-go`
   - Fedora: `sudo dnf install golang`
   - Arch: `sudo pacman -S go`
2. **Install Git**: Usually pre-installed, or `sudo apt install git`
3. **Install Make**: Usually pre-installed, or `sudo apt install make`

**Terminal Setup:**

```bash
# Clone the repository
git clone https://github.com/Tyrowin/gochat.git
cd gochat

# Make scripts executable
chmod +x scripts/build.sh quick-build.sh

# Install development tools
make install-tools

# Build the project
./build.sh current

# Run the server
./bin/gochat
```

**Development with Hot Reload:**

```bash
# Run with hot reload
make dev
# or
air
```

## Building

### Quick Build (Current Platform)

The fastest way to build for your current platform:

**Windows PowerShell:**

```powershell
.\quick-build.ps1
```

**macOS/Linux:**

```bash
./quick-build.sh
```

### Platform-Specific Build Scripts

#### Windows (build.ps1)

```powershell
# Show help
.\build.ps1 -Help

# Build for current platform
.\build.ps1 current

# Build for specific platforms
.\build.ps1 windows
.\build.ps1 linux
.\build.ps1 darwin
.\build.ps1 darwin-arm64

# Build for all platforms
.\build.ps1 all

# Create release builds
.\build.ps1 release

# Clean before building
.\build.ps1 -Clean all

# Custom output name
.\build.ps1 -Output myapp windows

# Verbose output
.\build.ps1 -Verbose all
```

#### macOS/Linux (build.sh)

```bash
# Show help
./build.sh --help

# Build for current platform
./build.sh current

# Build for specific platforms
./build.sh windows
./build.sh linux
./build.sh darwin
./build.sh darwin-arm64

# Build for all platforms
./build.sh all

# Create release builds
./build.sh release

# Clean before building
./build.sh --clean all

# Custom output name
./build.sh -o myapp windows

# Verbose output
./build.sh -v all
```

### Using Makefile (All Platforms)

```bash
# Build for current platform
make build-current

# Build for specific platforms
make build-windows
make build-linux
make build-linux-arm64
make build-darwin
make build-darwin-arm64

# Build for all platforms
make build-all

# Create optimized release builds
make release

# Show all available targets
make help

# List all supported platforms
make list-platforms
```

## Cross-Compilation

One of Go's most powerful features is the ability to build binaries for different platforms without needing complex toolchains.

### How Cross-Compilation Works

Go uses two environment variables to control the target platform:

- `GOOS`: Target operating system (e.g., `linux`, `windows`, `darwin`)
- `GOARCH`: Target architecture (e.g., `amd64`, `arm64`, `386`)

### Examples

**Build Windows executable from macOS:**

```bash
./build.sh windows
# Creates: bin/gochat-windows-amd64.exe
```

**Build macOS binary from Windows:**

```powershell
.\build.ps1 darwin-arm64
# Creates: bin/gochat-darwin-arm64
```

**Build Linux binary from any platform:**

```bash
# Unix-like systems
./build.sh linux

# Windows
.\build.ps1 linux
```

### Manual Cross-Compilation

If you need to build manually:

**Windows PowerShell:**

```powershell
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o bin/gochat-linux-amd64 ./cmd/server
```

**macOS/Linux:**

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/gochat-windows-amd64.exe ./cmd/server
```

### Supported Platforms

View all supported platforms:

```bash
go tool dist list
```

Common combinations:

- `linux/amd64` - 64-bit Linux
- `linux/arm64` - ARM64 Linux (Raspberry Pi, AWS Graviton)
- `darwin/amd64` - macOS Intel
- `darwin/arm64` - macOS Apple Silicon (M1/M2/M3)
- `windows/amd64` - 64-bit Windows
- `windows/386` - 32-bit Windows

## Platform-Specific Code

Sometimes you need code that behaves differently on different platforms.

### 1. File Name Suffixes

Create different implementations for different platforms:

```
internal/server/
├── config.go              # Shared code
├── config_windows.go      # Windows-specific
├── config_darwin.go       # macOS-specific
└── config_linux.go        # Linux-specific
```

Go automatically selects the right file based on the target platform.

**Example - config_windows.go:**

```go
//go:build windows

package server

func platformSpecificConfig() {
    // Windows-specific configuration
}
```

**Example - config_linux.go:**

```go
//go:build linux

package server

func platformSpecificConfig() {
    // Linux-specific configuration
}
```

### 2. Build Tags

Use build tags for more complex conditions:

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

### 3. Runtime Detection

Check the platform at runtime:

```go
package server

import (
    "runtime"
    "path/filepath"
)

func getPlatformSpecificPath() string {
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

### 4. File Path Handling

Use `filepath` package for cross-platform paths:

```go
import "path/filepath"

// GOOD - Works on all platforms
configPath := filepath.Join("config", "app.json")

// BAD - Only works on Unix-like systems
configPath := "config/app.json"

// BAD - Only works on Windows
configPath := "config\\app.json"
```

## Troubleshooting

### Windows-Specific Issues

**Issue: "Scripts are disabled on this system"**

```powershell
# Solution: Set execution policy
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Issue: "make: command not found"**

```powershell
# Solution: Use PowerShell scripts instead of Make
.\build.ps1 current

# Or install Make via Chocolatey
choco install make
```

**Issue: Line ending problems with Git**

```powershell
# Configure Git to handle line endings correctly
git config --global core.autocrlf true
```

### macOS-Specific Issues

**Issue: "Permission denied" when running scripts**

```bash
# Solution: Make scripts executable
chmod +x scripts/build.sh quick-build.sh
```

**Issue: "Developer cannot be verified" when running binary**

```bash
# Solution: Allow the binary in System Preferences > Security & Privacy
# Or remove quarantine attribute
xattr -d com.apple.quarantine bin/gochat
```

### Linux-Specific Issues

**Issue: "Go not found" after installation**

```bash
# Solution: Add Go to PATH
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

**Issue: "Cannot execute binary file"**

```bash
# Solution: Make sure you're running the correct architecture
file bin/gochat
# Should match your system architecture

# Rebuild for your platform
./build.sh current
```

### Cross-Compilation Issues

**Issue: "CGo is not enabled"**

```bash
# Solution: This project doesn't need CGo. Make sure CGO_ENABLED=0
# Our build scripts already set this correctly
```

**Issue: "No such file or directory" when running cross-compiled binary**

```bash
# Make sure you're running the binary on the correct platform
# Linux binary won't run on Windows, etc.

# Check the binary
file bin/gochat-linux-amd64
# Output should match the target platform
```

### General Issues

**Issue: Build is slow**

```bash
# Solution: Use cached builds
go build -i -o bin/gochat ./cmd/server

# Or clean Go cache if it's corrupted
go clean -cache -modcache -i -r
```

**Issue: Module download errors**

```bash
# Solution: Clean and re-download modules
go clean -modcache
go mod download
```

**Issue: Version conflicts**

```bash
# Solution: Tidy up dependencies
go mod tidy
```

## Best Practices

1. **Use the provided build scripts** - They handle platform differences automatically
2. **Test on multiple platforms** - If possible, test your changes on Windows, macOS, and Linux
3. **Use `filepath` package** - Always use `filepath.Join()` for paths
4. **Set `CGO_ENABLED=0`** - For maximum portability (already configured)
5. **Use build tags** - When you need platform-specific code
6. **Check file permissions** - Remember that executable permissions work differently on Unix vs Windows
7. **Handle line endings** - Configure Git properly for your platform
8. **Use the `.air.toml` config** - For consistent hot-reload across platforms

## Resources

- [Go Official Documentation](https://golang.org/doc/)
- [Cross-Compilation Guide](https://golang.org/doc/install/source#environment)
- [Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Go on Windows](https://golang.org/doc/install/windows)
- [Go on macOS](https://golang.org/doc/install/darwin)
- [Go on Linux](https://golang.org/doc/install/linux)

## Getting Help

If you encounter issues not covered here:

1. Check the [GitHub Issues](https://github.com/Tyrowin/gochat/issues)
2. Review the build script source code (`build.ps1` or `build.sh`)
3. Run with verbose output (`-Verbose` or `-v` flag)
4. Open a new issue with details about your platform and the error
