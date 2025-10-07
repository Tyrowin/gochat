# Multi-stage Dockerfile for GoChat Server
# This creates a minimal, secure production image

# Stage 1: Build stage
FROM golang:1.25.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache ca-certificates git tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with security optimizations
# -trimpath removes file system paths from the executable
# -ldflags="-s -w" strips debug information to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION:-dev}" \
    -o gochat \
    ./cmd/server

# Stage 2: Runtime stage
FROM alpine:3.20

# Install runtime dependencies and create non-root user
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 gochat && \
    adduser -D -u 1000 -G gochat gochat

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/gochat .

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Change ownership to non-root user
RUN chown -R gochat:gochat /app

# Switch to non-root user
USER gochat

# Expose the default port (can be overridden with environment variable)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the application
ENTRYPOINT ["/app/gochat"]
