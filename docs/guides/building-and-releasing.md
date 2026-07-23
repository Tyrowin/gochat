# Building and Releasing

Build for your own machine, cross-compile for every supported platform, and cut a release.

Full target list: [Make targets reference](../reference/make-targets.md).

## Build for the current platform

```bash
make build
```

Runs `go fmt ./...` and `go vet ./...`, then writes `bin/gochat` (`bin/gochat.exe` on Windows).
Use `make build-raw` to skip the checks during a tight edit loop.

Without Make:

```bash
go build -o bin/gochat ./cmd/server        # macOS / Linux
go build -o bin\gochat.exe .\cmd\server    # Windows PowerShell
```

## Cross-compile

The server is pure Go with one dependency, so cross-compiling needs no toolchain beyond Go itself.

```bash
make build-linux           # bin/linux/gochat-amd64
make build-linux-arm64     # bin/linux/gochat-arm64
make build-darwin          # bin/MacOS/gochat-amd64
make build-darwin-arm64    # bin/MacOS/gochat-arm64
make build-windows         # bin/windows/gochat-amd64.exe
make build-windows-arm64   # bin/windows/gochat-arm64.exe
make build-all             # all six
```

Each target sets `CGO_ENABLED=0` with `GOOS`/`GOARCH`. For a platform without a Make target, set the
variables directly:

```bash
GOOS=freebsd GOARCH=amd64 go build -o bin/gochat-freebsd-amd64 ./cmd/server
GOOS=linux GOARCH=arm GOARM=7 go build -o bin/gochat-linux-armv7 ./cmd/server
```

`make list-platforms` prints everything Go supports.

## Release builds

```bash
make release
```

This runs `clean fmt vet lint test` first — a lint or test failure aborts the release — then builds
all six platforms with:

- `CGO_ENABLED=0` — static binary, runs on any distro and in `scratch`/`alpine` images
- `-trimpath` — no local filesystem paths in the binary, reproducible across machines
- `-ldflags="-s -w"` — symbol table and DWARF data stripped, roughly 30% smaller
- `-a -installsuffix cgo` — force a full rebuild of all packages

It then writes `checksums.txt` (SHA256) into each platform directory:

```
bin/
├── linux/    gochat-amd64, gochat-arm64, checksums.txt
├── MacOS/    gochat-amd64, gochat-arm64, checksums.txt
└── windows/  gochat-amd64.exe, gochat-arm64.exe, checksums.txt
```

Verify with `sha256sum -c checksums.txt` (`shasum -a 256 -c` on macOS).

> **Note:** `make clean` — which `release` runs first — wipes the shared Go module, build, and test
> caches for every project on the machine, so the next build is slow.

### Version stamping

The Makefile passes `-X main.Version`, `-X main.Commit`, and `-X main.BuildTime`, which `main`
declares and logs in its first record at startup:

```
level=INFO msg="starting GoChat server" version=v1.2.0 commit=48ab334 build_time=2026-07-23T13:51:57Z
```

The `Dockerfile` stamps `main.Version` only, from its `VERSION` build argument
(`docker build --build-arg VERSION=v1.2.0 .`).

On non-Windows hosts the values come from `git describe --tags --always --dirty` and
`git rev-parse --short HEAD`; on Windows the Makefile defaults them to `dev`/`unknown` and stamps
only the build time.

## Container images

```bash
docker build -t gochat:latest .
make docker-build             # tags gochat:$(VERSION) and gochat:latest
```

The multi-stage `Dockerfile` compiles with the same static/stripped flags and copies the binary into
`alpine:3.24`. Details in [Running with Docker](running-with-docker.md).

Multi-architecture images need buildx:

```bash
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64 -t gochat:latest --push .
```

The build stage reads buildx's `TARGETOS` and `TARGETARCH` build arguments, so each manifest entry
gets a binary for its own platform. Both default to `linux/amd64` for a plain `docker build`.

## Troubleshooting

**`go: cannot find main module`** — run the command from the repository root.

**Cross-compiled binary is much larger than expected** — you used `make build` rather than
`make release`; development builds keep symbols and debug info.

**macOS refuses to run a downloaded binary** — `xattr -d com.apple.quarantine bin/gochat`.

**`permission denied` on Linux** — `chmod +x bin/gochat`.

## Related

- [Make targets reference](../reference/make-targets.md) — every target
- [Running with Docker](running-with-docker.md) — image build and run
- [Deploying to production](deploying-to-production.md) — where the binary goes
- [Testing](testing.md) — the suite `make release` depends on
