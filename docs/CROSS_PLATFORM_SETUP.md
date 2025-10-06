# Cross-Platform Development Setup Summary

This document summarizes the cross-platform development capabilities of the GoChat project.

## Overview

GoChat supports seamless development, building, and deployment across **Windows**, **macOS**, and **Linux**. You can develop on any platform and build binaries for any platform using Make or direct Go commands.

## What's Included

### 1. Enhanced Makefile

Comprehensive cross-platform build targets:

- `build-current` - Build for current platform
- `build-linux` - Build for Linux (amd64)
- `build-linux-arm64` - Build for Linux (ARM64)
- `build-darwin` - Build for macOS (Intel)
- `build-darwin-arm64` - Build for macOS (Apple Silicon)
- `build-windows` - Build for Windows (amd64)
- `build-all` - Build for all platforms
- `release` - Create optimized release builds for all platforms
- `list-platforms` - Show all supported GOOS/GOARCH combinations

### 2. Development Configuration

#### Air Configuration (`.air.toml`)

- Hot reload configuration for development
- Works consistently across all platforms
- Excludes test files and build directories
- Configurable build commands and delays

#### VS Code Integration

**Tasks (`.vscode/tasks.json`)**

- Build (Current Platform) - Default build task
- Build All Platforms
- Run Server
- Dev (Hot Reload)
- Test All - Default test task
- Test with Coverage
- Lint
- Security Scan
- Full CI Check
- Clean Build Artifacts
- Create Release

All tasks work on Windows, macOS, and Linux with platform-specific command adjustments.

**Launch Configurations (`.vscode/launch.json`)**

- Launch Server (with pre-build)
- Launch Server (no build)
- Debug Tests (Current File)
- Debug Tests (All)
- Attach to Process

**Extensions (`.vscode/extensions.json`)**
Recommended extensions for cross-platform Go development:

- Go
- YAML
- Makefile Tools
- PowerShell (for Windows users)
- GitLens
- Markdown Lint
- EditorConfig
- Code Spell Checker
- Docker
- GitHub Actions

### 3. Configuration Files

#### `.gitattributes`

Ensures consistent line ending handling across platforms:

- Shell scripts (`.sh`) always use LF
- PowerShell scripts (`.ps1`) use CRLF on Windows
- Go source files use LF
- Binary files properly marked

#### `.editorconfig`

Ensures consistent coding style across all editors and platforms:

- Go files: tabs, size 4
- YAML/JSON: spaces, size 2
- Makefiles: tabs (required)
- Shell scripts: spaces, LF line endings
- PowerShell: spaces, CRLF line endings

### 4. Documentation

#### Comprehensive Guide (`docs/CROSS_PLATFORM_GUIDE.md`)

Complete guide covering:

- Development setup for each platform
- Building and cross-compilation
- Platform-specific code techniques
- Troubleshooting for each platform
- Best practices
- Resources and help

#### Quick Reference (`docs/QUICK_REFERENCE.md`)

One-page reference card with:

- All build commands
- Development workflow
- Testing commands
- Cross-compilation examples
- Troubleshooting quick fixes
- File locations

#### Updated README

Main README now includes:

- Cross-platform features highlighted
- Platform-specific prerequisites
- Quick build instructions for all platforms
- Comprehensive cross-platform build section
- Links to detailed guides

## File Structure

```
gochat/
├── .editorconfig                    # Editor configuration (all platforms)
├── .gitattributes                   # Git line ending configuration
├── .air.toml                        # Hot reload configuration
├── Makefile                         # Cross-platform build targets
├── .vscode/
│   ├── tasks.json                   # VS Code tasks (all platforms)
│   ├── launch.json                  # Debug configurations
│   └── extensions.json              # Recommended extensions
├── docs/
│   ├── CROSS_PLATFORM_GUIDE.md      # Comprehensive guide
│   └── QUICK_REFERENCE.md           # Quick reference card
└── bin/                             # Build output directory
    ├── gochat-linux-amd64
    ├── gochat-linux-arm64
    ├── gochat-darwin-amd64
    ├── gochat-darwin-arm64
    ├── gochat-windows-amd64.exe
    └── checksums.txt                # SHA256 checksums
```

## Supported Platforms

### Development Platforms (where you code)

- ✅ Windows 10/11 (PowerShell 5.1+ or 7+)
- ✅ macOS (Intel and Apple Silicon)
- ✅ Linux (any distribution with Bash)

### Target Platforms (what you can build for)

- ✅ Windows (amd64)
- ✅ Linux (amd64, arm64)
- ✅ macOS (Intel amd64, Apple Silicon arm64)
- ✅ And many more via `go tool dist list`

## Key Features

### 1. True Cross-Compilation

Build binaries for **any** platform from **any** platform:

- Build Windows .exe from macOS
- Build macOS binary from Windows
- Build Linux binary from Windows
- No complex toolchains needed

### 2. Platform-Native Scripts

Choose your preferred workflow:

- **Windows users**: Use PowerShell scripts (`.ps1`)
- **macOS/Linux users**: Use Bash scripts (`.sh`)
- **Everyone**: Use Make targets (if Make is installed)

### 3. Consistent Development Experience

Same workflow on all platforms:

```bash
# Clone
git clone <repo>
cd gochat

# Build
./quick-build.sh    # or .\quick-build.ps1

# Run
./bin/gochat        # or .\bin\gochat.exe

# Develop with hot reload
air
```

### 4. VS Code Integration

Press `Ctrl+Shift+B` (or `Cmd+Shift+B` on macOS) to:

- Build for current platform
- Build for all platforms
- Run tests
- Start dev server
- Create release builds

All tasks work on all platforms with no configuration changes.

### 5. Proper Line Ending Handling

- Shell scripts always use LF (required for Unix)
- PowerShell scripts use CRLF on Windows
- Go source files consistent across platforms
- No more "bad interpreter" errors

### 6. Consistent Code Formatting

EditorConfig ensures:

- Tabs vs spaces handled correctly
- Consistent indentation
- Proper line endings
- Works with any editor (VS Code, Vim, IntelliJ, etc.)

## Quick Start Guide

### Windows

```powershell
# First time setup
git clone https://github.com/Tyrowin/gochat.git
cd gochat

# Build
.\quick-build.ps1

# Run
.\bin\gochat.exe

# Develop with hot reload
air
```

### macOS/Linux

```bash
# First time setup
git clone https://github.com/Tyrowin/gochat.git
cd gochat
chmod +x scripts/*.sh

# Build
./quick-build.sh

# Run
./bin/gochat

# Develop with hot reload
air
```

## Common Tasks

### Build for Current Platform

```bash
# Windows
.\quick-build.ps1

# macOS/Linux
./quick-build.sh

# With Make
make build-current
```

### Build for All Platforms

```bash
# Windows
.\build.ps1 all

# macOS/Linux
./build.sh all

# With Make
make build-all
```

### Create Release

```bash
# Windows
.\build.ps1 release

# macOS/Linux
./build.sh release

# With Make
make release
```

### Development with Hot Reload

```bash
# All platforms
air

# Or with Make
make dev
```

## Testing the Setup

1. **Test quick build:**

   ```bash
   # Your platform's quick build command
   ./quick-build.sh  # or .\quick-build.ps1
   ```

2. **Test cross-compilation:**

   ```bash
   # Build for a different platform
   ./build.sh windows  # or .\build.ps1 linux
   ```

3. **Test hot reload:**

   ```bash
   air
   # Make a change to a .go file
   # Should automatically rebuild and restart
   ```

4. **Test VS Code tasks:**
   - Open VS Code
   - Press `Ctrl+Shift+B` / `Cmd+Shift+B`
   - Select "Build (Current Platform)"
   - Should build successfully

## Benefits

### For Developers

- Work on your preferred platform
- No platform lock-in
- Consistent experience everywhere
- Easy onboarding for new contributors
- Professional tooling support

### For Users

- Get binaries for their platform easily
- No need to build from source
- Checksums for security verification
- Optimized, self-contained binaries

### For Contributors

- Clear documentation
- Easy setup process
- Automated builds and tests
- Platform-specific help available

## Maintenance

### Keeping Scripts in Sync

When adding new build features:

1. Update `build.sh`
2. Update `build.ps1` (equivalent functionality)
3. Update `Makefile` targets
4. Update documentation
5. Test on at least two platforms

### Adding New Platforms

To add support for a new platform (e.g., FreeBSD):

1. Add build target to Makefile
2. Add case to build scripts
3. Update documentation
4. Test cross-compilation works

### Updating Dependencies

```bash
# All platforms
go get -u ./...
go mod tidy
```

## Troubleshooting

See the comprehensive troubleshooting sections in:

- [Cross-Platform Development Guide](./CROSS_PLATFORM_GUIDE.md#troubleshooting)
- [Quick Reference](./QUICK_REFERENCE.md#troubleshooting)

## Resources

- **Full Documentation**: [CROSS_PLATFORM_GUIDE.md](./CROSS_PLATFORM_GUIDE.md)
- **Quick Reference**: [QUICK_REFERENCE.md](./QUICK_REFERENCE.md)
- **Main README**: [../README.md](../README.md)
- **Go Cross-Compilation**: https://golang.org/doc/install/source#environment
- **EditorConfig**: https://editorconfig.org/
- **Air (Hot Reload)**: https://github.com/air-verse/air

## Contributing

When contributing to this project:

1. ✅ Test changes on multiple platforms if possible
2. ✅ Update both `.sh` and `.ps1` scripts if modifying builds
3. ✅ Use `filepath` package for cross-platform paths
4. ✅ Check line endings are correct (`.gitattributes` helps)
5. ✅ Update documentation if adding new features
6. ✅ Run `make ci-local` before submitting PR

## Next Steps

1. **Read the guides**: Start with [CROSS_PLATFORM_GUIDE.md](./CROSS_PLATFORM_GUIDE.md)
2. **Try the tools**: Test building for different platforms
3. **Set up VS Code**: Install recommended extensions
4. **Join development**: See CONTRIBUTING.md

---

**Happy cross-platform development! 🚀**
