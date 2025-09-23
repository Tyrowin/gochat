# Makefile for nexus-chat-server
# 
# This Makefile provides convenient targets for development, testing, 
# security scanning, and building the application.

.PHONY: help build clean test test-coverage lint lint-fix security-scan deps-check deps-update run dev fmt vet all ci-local install-tools docker-build docker-run

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=nexus-chat-server
BUILD_DIR=./bin
MAIN_PATH=./cmd/server
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the application binary
build: fmt vet
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Remove build artifacts and temporary files
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@rm -f unit-$(COVERAGE_FILE) unit-$(COVERAGE_HTML)
	@rm -f integration-$(COVERAGE_FILE) integration-$(COVERAGE_HTML)
	@go clean -cache -testcache -modcache
	@echo "Clean completed"

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race ./...

## test-unit: Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -v -race ./test/unit/...

## test-integration: Run integration tests only
test-integration:
	@echo "Running integration tests..."
	go test -v -race ./test/integration/...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverpkg=./cmd/...,./internal/... -coverprofile=$(COVERAGE_FILE) ./test/...
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	go tool cover -func=$(COVERAGE_FILE)

## test-coverage-unit: Run unit tests with coverage report
test-coverage-unit:
	@echo "Running unit tests with coverage..."
	go test -v -race -coverpkg=./cmd/...,./internal/... -coverprofile=unit-$(COVERAGE_FILE) ./test/unit/...
	go tool cover -html=unit-$(COVERAGE_FILE) -o unit-$(COVERAGE_HTML)
	@echo "Unit test coverage report generated: unit-$(COVERAGE_HTML)"
	go tool cover -func=unit-$(COVERAGE_FILE)

## test-coverage-integration: Run integration tests with coverage report
test-coverage-integration:
	@echo "Running integration tests with coverage..."
	go test -v -race -coverpkg=./cmd/...,./internal/... -coverprofile=integration-$(COVERAGE_FILE) ./test/integration/...
	go tool cover -html=integration-$(COVERAGE_FILE) -o integration-$(COVERAGE_HTML)
	@echo "Integration test coverage report generated: integration-$(COVERAGE_HTML)"
	go tool cover -func=integration-$(COVERAGE_FILE)

## lint: Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make install-tools' first." && exit 1)
	golangci-lint run --config .golangci.yml

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make install-tools' first." && exit 1)
	golangci-lint run --config .golangci.yml --fix

## security-scan: Run security vulnerability scans
security-scan:
	@echo "Running security scans..."
	@echo "1. Running govulncheck..."
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
	@echo "2. Running gosec..."
	@which gosec > /dev/null || go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	gosec ./...

## deps-check: Check for outdated dependencies
deps-check:
	@echo "Checking dependencies..."
	@echo "Current dependencies:"
	go list -u -m all
	@echo ""
	@echo "Checking for vulnerabilities in dependencies..."
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "Dependencies updated"

## run: Build and run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run the application in development mode (with auto-restart)
dev:
	@echo "Starting development server..."
	@which air > /dev/null || (echo "Air not found. Install with: go install github.com/cosmtrek/air@latest" && exit 1)
	air

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "goimports not found, skipping import formatting"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## all: Run all checks and build
all: clean fmt vet lint test build
	@echo "All checks completed successfully!"

## ci-local: Run the same checks as CI pipeline locally
ci-local: clean fmt vet lint test-coverage security-scan deps-check build
	@echo "Local CI pipeline completed successfully!"

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/air-verse/air@latest
	@echo "Development tools installed"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "Docker image built: $(BINARY_NAME):$(VERSION)"

## docker-run: Run application in Docker container
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 $(BINARY_NAME):latest

# Generate mod graph for dependency analysis
## deps-graph: Generate dependency graph
deps-graph:
	@echo "Generating dependency graph..."
	go mod graph | dot -T png -o deps-graph.png
	@echo "Dependency graph saved as deps-graph.png"

# Check for license compatibility
## license-check: Check licenses of dependencies
license-check:
	@echo "Checking dependency licenses..."
	@which go-licenses > /dev/null || go install github.com/google/go-licenses@latest
	go-licenses report ./... --template licenses.tpl > licenses.txt || true
	@echo "License report saved to licenses.txt"

# Performance benchmarks
## bench: Run performance benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Check for potential race conditions
## race: Run tests with race detection
race:
	@echo "Running race detection..."
	go test -race ./...

# Generate documentation
## docs: Generate documentation
docs:
	@echo "Generating documentation..."
	@which godoc > /dev/null || go install golang.org/x/tools/cmd/godoc@latest
	@echo "Documentation server will be available at http://localhost:6060"
	godoc -http=:6060

# Create release build
## release: Create optimized release build
release: clean fmt vet lint test
	@echo "Creating release build..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Release builds created in $(BUILD_DIR)/"