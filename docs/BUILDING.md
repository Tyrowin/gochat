# Building GoChat

This guide covers building GoChat for development and production, including cross-compilation for multiple platforms.

## Table of Contents

- [Quick Build](#quick-build)
- [Build Options](#build-options)
- [Cross-Compilation](#cross-compilation)
- [Platform-Specific Builds](#platform-specific-builds)
- [Build Output](#build-output)
- [Advanced Build Options](#advanced-build-options)

## Quick Build

### Using Make (Recommended)

**Build for current platform:**

```bash
make build
```

**Build for all platforms:**

```bash
make build-all
```

**Create production release builds:**

```bash
make release
```

### Using Go Directly

**On Windows (PowerShell):**

```powershell
go build -o bin\gochat.exe .\cmd\server
```

**On macOS/Linux:**

```bash
go build -o bin/gochat ./cmd/server
```

## Build Options

### Development Build

For local development and testing:

```bash
make build
# or
go build -o bin/gochat ./cmd/server
```

**Characteristics:**

- Fast compilation
- Includes debug symbols
- No optimization
- Larger binary size
- Good for debugging

### Production Build

For deployment:

```bash
make release
# or
go build -ldflags="-s -w" -trimpath -o bin/gochat ./cmd/server
```

**Characteristics:**

- Optimized for size and performance
- Debug symbols stripped (`-s -w`)
- Reproducible builds (`-trimpath`)
- Smaller binary size
- Faster execution

### Build Flags Explained

**`-ldflags="-s -w"`**

- `-s`: Omit symbol table
- `-w`: Omit DWARF debugging information
- Result: Smaller binary size

**`-trimpath`**

- Removes absolute file paths from binary
- Makes builds reproducible across machines
- Enhances security by not exposing local paths

**`CGO_ENABLED=0`**

- Disables CGo
- Creates statically linked binary
- No external dependencies
- Easier cross-compilation

## Cross-Compilation

Go makes it easy to build for different platforms from any development machine.

### How Cross-Compilation Works

Set two environment variables:

- **`GOOS`**: Target operating system (linux, darwin, windows)
- **`GOARCH`**: Target architecture (amd64, arm64, 386)

### Cross-Compilation Examples

**From Windows build for Linux:**

```powershell
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -o bin/gochat-linux-amd64 .\cmd\server
```

**From macOS build for Windows:**

```bash
GOOS=windows GOARCH=amd64 go build -o bin/gochat-windows-amd64.exe ./cmd/server
```

**From Linux build for macOS (Apple Silicon):**

```bash
GOOS=darwin GOARCH=arm64 go build -o bin/gochat-darwin-arm64 ./cmd/server
```

### Using Make for Cross-Compilation

```bash
# Build for specific platform
make build-linux           # Linux (amd64)
make build-linux-arm64     # Linux (arm64)
make build-darwin          # macOS Intel
make build-darwin-arm64    # macOS Apple Silicon
make build-windows         # Windows (amd64)

# Build for all platforms
make build-all

# List all supported platforms
make list-platforms
```

## Platform-Specific Builds

### Linux Builds

**AMD64 (x86_64):**

```bash
GOOS=linux GOARCH=amd64 go build -o bin/gochat-linux-amd64 ./cmd/server
```

**ARM64 (Raspberry Pi, AWS Graviton):**

```bash
GOOS=linux GOARCH=arm64 go build -o bin/gochat-linux-arm64 ./cmd/server
```

**ARM (32-bit):**

```bash
GOOS=linux GOARCH=arm GOARM=7 go build -o bin/gochat-linux-arm ./cmd/server
```

### macOS Builds

**Intel (x86_64):**

```bash
GOOS=darwin GOARCH=amd64 go build -o bin/gochat-darwin-amd64 ./cmd/server
```

**Apple Silicon (M1/M2/M3/M4):**

```bash
GOOS=darwin GOARCH=arm64 go build -o bin/gochat-darwin-arm64 ./cmd/server
```

### Windows Builds

**AMD64 (x86_64):**

```bash
GOOS=windows GOARCH=amd64 go build -o bin/gochat-windows-amd64.exe ./cmd/server
```

**ARM64:**

```bash
GOOS=windows GOARCH=arm64 go build -o bin/gochat-windows-arm64.exe ./cmd/server
```

**32-bit (386):**

```bash
GOOS=windows GOARCH=386 go build -o bin/gochat-windows-386.exe ./cmd/server
```

### FreeBSD Builds

```bash
GOOS=freebsd GOARCH=amd64 go build -o bin/gochat-freebsd-amd64 ./cmd/server
```

### Other Platforms

Go supports many platforms. View all:

```bash
go tool dist list
```

## Build Output

### Directory Structure

After running `make build-all` or `make release`, binaries are organized:

```
bin/
├── gochat                      # Current platform binary
├── linux/
│   ├── gochat-amd64           # Linux x86_64
│   ├── gochat-arm64           # Linux ARM64
│   └── checksums.txt          # SHA256 checksums
├── darwin/
│   ├── gochat-amd64           # macOS Intel
│   ├── gochat-arm64           # macOS Apple Silicon
│   └── checksums.txt          # SHA256 checksums
└── windows/
    ├── gochat-amd64.exe       # Windows x86_64
    └── checksums.txt          # SHA256 checksums
```

### Checksums

Release builds include SHA256 checksums for verification:

```bash
# Generate checksums
cd bin/linux
sha256sum gochat-* > checksums.txt

# Verify integrity
sha256sum -c checksums.txt
```

### Binary Size

Typical binary sizes:

| Build Type             | Size (approx) |
| ---------------------- | ------------- |
| Development            | 8-12 MB       |
| Production (optimized) | 6-8 MB        |
| Compressed (UPX)       | 3-4 MB        |

## Advanced Build Options

### Version Information

Embed version information at build time:

```bash
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
  -o bin/gochat ./cmd/server
```

In your code:

```go
package main

var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    fmt.Printf("GoChat %s (built %s)\n", Version, BuildTime)
    // ...
}
```

### Static Linking

For maximum portability (no dynamic dependencies):

```bash
CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/gochat ./cmd/server
```

**Benefits:**

- No external dependencies
- Works on any system (even minimal containers)
- Easier deployment

**Limitations:**

- Cannot use C libraries
- Larger binary size
- No dynamic linking benefits

### Compression with UPX

Further reduce binary size (optional):

```bash
# Install UPX
# macOS: brew install upx
# Linux: apt install upx-ucl
# Windows: Download from https://upx.github.io/

# Compress binary
upx --best --lzma bin/gochat

# Decompress if needed
upx -d bin/gochat
```

**Warning:** Some antivirus software flags UPX-compressed binaries.

### Custom Build Tags

Use build tags for conditional compilation:

```go
//go:build debug

package main

func init() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
}
```

Build with tag:

```bash
go build -tags debug -o bin/gochat ./cmd/server
```

### Multi-Architecture Docker Builds

Build Docker images for multiple architectures:

```bash
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t gochat:latest \
  --push \
  .
```

## Build Automation

### Makefile Targets

View all available build targets:

```bash
make help
```

**Key targets:**

| Target                    | Description                      |
| ------------------------- | -------------------------------- |
| `make build`              | Build for current platform       |
| `make build-current`      | Same as `build`                  |
| `make build-all`          | Build for all platforms          |
| `make build-linux`        | Build for Linux (amd64)          |
| `make build-linux-arm64`  | Build for Linux (arm64)          |
| `make build-darwin`       | Build for macOS Intel            |
| `make build-darwin-arm64` | Build for macOS Apple Silicon    |
| `make build-windows`      | Build for Windows (amd64)        |
| `make release`            | Production builds with checksums |
| `make clean`              | Remove all build artifacts       |

### CI/CD Builds

GitHub Actions automatically builds on every commit:

```yaml
- name: Build
  run: make build-all
```

Artifacts are stored for 90 days.

## Troubleshooting

### Build Fails on Windows

**Error:** `package github.com/gorilla/websocket: unrecognized import path`

**Solution:** Ensure Go modules are enabled:

```powershell
$env:GO111MODULE="on"
go build -o bin\gochat.exe .\cmd\server
```

### Cross-Compilation Fails

**Error:** `undefined: syscall.SomeFunction`

**Cause:** Platform-specific code not properly guarded

**Solution:** Use build tags or runtime checks:

```go
//go:build linux

package server

// Linux-specific code
```

### Binary Won't Run

**Error (Linux):** `permission denied`

**Solution:** Make binary executable:

```bash
chmod +x bin/gochat
```

**Error (macOS):** `"gochat" cannot be opened because the developer cannot be verified`

**Solution:** Allow in System Preferences > Security & Privacy, or:

```bash
xattr -d com.apple.quarantine bin/gochat
```

### Large Binary Size

**Problem:** Binary is larger than expected

**Solutions:**

1. Use production build flags: `-ldflags="-s -w"`
2. Enable trimpath: `-trimpath`
3. Strip additional symbols: `strip bin/gochat`
4. Consider UPX compression (optional)

## Performance Considerations

### Build Time Optimization

**Use build cache:**

```bash
# Cache is enabled by default
go env GOCACHE
```

**Parallel compilation:**

```bash
# Set number of CPU cores
go build -p 8 -o bin/gochat ./cmd/server
```

**Incremental builds:**

```bash
# Only rebuild changed packages
go build -i -o bin/gochat ./cmd/server
```

### Runtime Performance

**Optimize for speed:**

```bash
go build -ldflags="-s -w" -gcflags="-l=4" -o bin/gochat ./cmd/server
```

**Profile-guided optimization (PGO):**

```bash
# Collect profile
go run ./cmd/server &
# Run load tests, then stop server
# Build with profile
go build -pgo=default.pgo -o bin/gochat ./cmd/server
```

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Installation and setup
- [Development Guide](DEVELOPMENT.md) - Development workflow
- [Deployment Guide](DEPLOYMENT.md) - Production deployment
- [Contributing](CONTRIBUTING.md) - Contribution guidelines
