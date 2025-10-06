# GoChat Cross-Platform Quick Reference

## Quick Build Commands

### Windows (PowerShell)

```powershell
.\scripts\quick-build.ps1              # Quick build for current platform
.\scripts\build.ps1 current            # Build for Windows
.\scripts\build.ps1 linux              # Build for Linux
.\scripts\build.ps1 darwin-arm64       # Build for macOS (Apple Silicon)
.\scripts\build.ps1 all                # Build for all platforms
.\scripts\build.ps1 release            # Create release builds
.\scripts\build.ps1 -Clean all         # Clean and build all
```

### macOS/Linux (Bash)

```bash
./scripts/quick-build.sh               # Quick build for current platform
./scripts/build.sh current             # Build for current platform
./scripts/build.sh windows             # Build for Windows
./scripts/build.sh darwin-arm64        # Build for macOS (Apple Silicon)
./scripts/build.sh all                 # Build for all platforms
./scripts/build.sh release             # Create release builds
./scripts/build.sh --clean all         # Clean and build all
```

### Make (All Platforms)

```bash
make build-current             # Build for current platform
make build-windows             # Build for Windows
make build-linux               # Build for Linux
make build-darwin-arm64        # Build for macOS (Apple Silicon)
make build-all                 # Build for all platforms
make release                   # Create release builds
make list-platforms            # List all supported platforms
```

## Development Commands

### Run Server

```powershell
# Windows
.\bin\gochat.exe

# macOS/Linux
./bin/gochat

# With Make
make run
```

### Hot Reload Development

```bash
make dev        # All platforms (requires air)
air             # Direct command (requires air)
```

### Install Development Tools

```bash
make install-tools    # Installs golangci-lint, govulncheck, air, etc.
```

## Testing & Quality

```bash
make test              # Run all tests
make test-unit         # Run unit tests only
make test-integration  # Run integration tests only
make test-coverage     # Run tests with coverage report
make lint              # Run linters
make lint-fix          # Run linters with auto-fix
make security-scan     # Run security scans
make ci-local          # Run full CI pipeline locally
```

## Cross-Compilation Environment Variables

### PowerShell

```powershell
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o bin/gochat-linux-amd64 ./cmd/server
```

### Bash

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/gochat.exe ./cmd/server
```

## Platform Targets

| Platform            | GOOS    | GOARCH | Output                   |
| ------------------- | ------- | ------ | ------------------------ |
| Windows 64-bit      | windows | amd64  | gochat-windows-amd64.exe |
| Linux 64-bit        | linux   | amd64  | gochat-linux-amd64       |
| Linux ARM64         | linux   | arm64  | gochat-linux-arm64       |
| macOS Intel         | darwin  | amd64  | gochat-darwin-amd64      |
| macOS Apple Silicon | darwin  | arm64  | gochat-darwin-arm64      |

## Common Tasks

### First Time Setup

```bash
# Clone
git clone https://github.com/Tyrowin/gochat.git
cd gochat

# Make scripts executable (macOS/Linux only)
chmod +x scripts/build.sh scripts/quick-build.sh

# Install tools
make install-tools

# Build
./scripts/quick-build.sh    # or .\scripts\quick-build.ps1 on Windows
```

### Development Workflow

```bash
# 1. Make changes to code
# 2. Run with hot reload
make dev

# OR build and run manually
make build && make run

# 3. Run tests
make test

# 4. Check code quality
make lint

# 5. Full CI check before commit
make ci-local
```

### Release Workflow

```bash
# 1. Run full quality checks
make ci-local

# 2. Create release builds
make release

# 3. Check output
ls -la bin/

# 4. Verify checksums
cat bin/checksums.txt
```

## Troubleshooting

### Windows

```powershell
# Enable script execution
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# If Make not found, use PowerShell scripts
.\scripts\build.ps1 current
```

### macOS

```bash
# Make scripts executable
chmod +x scripts/*.sh

# Remove quarantine from downloaded binaries
xattr -d com.apple.quarantine bin/gochat
```

### Linux

```bash
# Add Go to PATH if needed
export PATH=$PATH:/usr/local/go/bin

# Make sure binary is executable
chmod +x bin/gochat
```

## File Locations

```
gochat/
├── scripts/
│   ├── build.ps1           # Windows build script
│   ├── build.sh            # Unix build script
│   ├── quick-build.ps1     # Windows quick build
│   └── quick-build.sh      # Unix quick build
├── Makefile            # Make targets (all platforms)
├── .air.toml           # Hot reload config
├── .gitattributes      # Line ending config
├── bin/                # Build output directory
│   ├── gochat-linux-amd64
│   ├── gochat-darwin-arm64
│   ├── gochat-windows-amd64.exe
│   └── checksums.txt
└── docs/
    └── CROSS_PLATFORM_GUIDE.md  # Detailed guide
```

## Environment Setup

### Windows

- Install Go from golang.org
- Install Git from git-scm.com
- Optional: Install Make via Chocolatey

### macOS

- Install Xcode Command Line Tools
- Install Go via Homebrew or golang.org
- Make and Git included with Xcode

### Linux

- Install Go via package manager or golang.org
- Install Make and Git via package manager
- Usually pre-installed on most distributions

## Resources

- Full Guide: [docs/CROSS_PLATFORM_GUIDE.md](CROSS_PLATFORM_GUIDE.md)
- Main README: [README.md](../README.md)
- Go Documentation: https://golang.org/doc/
- Cross-Compilation: https://golang.org/doc/install/source#environment
