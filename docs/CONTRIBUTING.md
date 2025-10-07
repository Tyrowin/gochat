# Contributing to GoChat

Thank you for your interest in contributing to GoChat! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on what is best for the community
- Show empathy towards other community members
- Accept constructive criticism gracefully

### Unacceptable Behavior

- Harassment, discrimination, or offensive comments
- Trolling or insulting/derogatory comments
- Publishing others' private information
- Any other conduct inappropriate in a professional setting

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- Go 1.25.1 or later installed
- Git configured with your name and email
- A GitHub account
- Familiarity with Go and WebSocket concepts

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR_USERNAME/gochat.git
cd gochat
```

3. Add upstream remote:

```bash
git remote add upstream https://github.com/Tyrowin/gochat.git
```

4. Install development tools:

```bash
make install-tools
```

5. Verify your setup:

```bash
make ci-local
```

## How to Contribute

### Types of Contributions

We welcome various types of contributions:

**Code Contributions:**

- Bug fixes
- New features
- Performance improvements
- Code refactoring

**Documentation:**

- Fix typos or improve clarity
- Add examples
- Translate documentation
- Write tutorials or guides

**Testing:**

- Add test cases
- Improve test coverage
- Report bugs with detailed reproduction steps

**Other:**

- Improve error messages
- Enhance logging
- Optimize build process
- Update dependencies

## Development Workflow

### 1. Create a Branch

Create a descriptive branch name:

```bash
git checkout -b feature/add-user-authentication
git checkout -b fix/websocket-memory-leak
git checkout -b docs/improve-api-examples
```

Branch naming conventions:

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test improvements

### 2. Make Changes

- Write clean, readable code
- Follow Go best practices
- Add comments for complex logic
- Update documentation as needed

### 3. Write Tests

- Add unit tests for new functions
- Add integration tests for new features
- Ensure all tests pass
- Maintain or improve code coverage

### 4. Run Quality Checks

Before committing, run:

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Security scan
make security-scan

# Or run everything
make ci-local
```

### 5. Commit Changes

Write clear, descriptive commit messages:

```bash
git add .
git commit -m "Add user authentication feature

- Implement JWT-based authentication
- Add login/logout endpoints
- Update tests and documentation
- Closes #123"
```

**Commit message format:**

- First line: Brief summary (50 chars or less)
- Blank line
- Detailed description (wrap at 72 chars)
- Reference related issues

### 6. Keep Up to Date

Regularly sync with upstream:

```bash
git fetch upstream
git rebase upstream/main
```

### 7. Push and Create PR

```bash
git push origin feature/add-user-authentication
```

Then create a Pull Request on GitHub.

## Code Standards

### Go Style Guide

Follow the official [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).

**Key points:**

- Use `gofmt` for formatting (automatic with `make fmt`)
- Use meaningful variable names
- Keep functions small and focused
- Document exported functions and types
- Handle errors explicitly
- Avoid global variables when possible

### Documentation

**Package documentation:**

```go
// Package server provides WebSocket server functionality for real-time chat.
//
// The server handles WebSocket connections, message broadcasting, and
// client lifecycle management with built-in security features.
package server
```

**Function documentation:**

```go
// NewClient creates a new client instance for a WebSocket connection.
// It initializes the send channel and associates the client with the hub.
// The client is ready to be registered with the hub and start message pumps.
func NewClient(conn *websocket.Conn, hub *Hub, remoteAddr string) *Client {
    // ...
}
```

**Inline comments:**

```go
// Check if the client has exceeded rate limits before processing
if !client.rateLimiter.allow() {
    client.conn.Close()
    return
}
```

### Error Handling

**Always check errors:**

```go
// Bad
data, _ := json.Marshal(message)

// Good
data, err := json.Marshal(message)
if err != nil {
    return fmt.Errorf("failed to marshal message: %w", err)
}
```

**Provide context:**

```go
if err := client.conn.WriteMessage(messageType, data); err != nil {
    return fmt.Errorf("failed to send message to client %s: %w", client.id, err)
}
```

### Code Organization

**File structure:**

- One main concept per file
- Group related functionality
- Keep files under 500 lines when possible

**Package structure:**

```
internal/server/
├── client.go        # Client-specific code
├── hub.go          # Hub-specific code
├── handlers.go     # HTTP handlers
├── config.go       # Configuration
└── types.go        # Shared types
```

## Testing Requirements

### Test Coverage

- Aim for 80%+ test coverage
- All new code must include tests
- Tests must pass with `-race` flag

### Writing Tests

**Unit tests:**

```go
func TestRateLimiter_Allow(t *testing.T) {
    tests := []struct {
        name    string
        burst   int
        calls   int
        want    bool
    }{
        {"within limit", 5, 3, true},
        {"exceeds limit", 5, 10, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            rl := newRateLimiter(tt.burst, time.Second)
            // Test implementation
        })
    }
}
```

**Integration tests:**

```go
func TestMultiClientChat(t *testing.T) {
    server := testhelpers.StartTestServer(t)
    defer server.Close()

    client1 := testhelpers.ConnectClient(t, server.URL)
    client2 := testhelpers.ConnectClient(t, server.URL)

    // Test implementation
}
```

### Running Tests

```bash
# All tests
make test

# Specific package
go test -v -race ./internal/server

# With coverage
make test-coverage
```

## Pull Request Process

### Before Submitting

Checklist:

- [ ] Code follows style guidelines
- [ ] Tests pass locally (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Security scans pass (`make security-scan`)
- [ ] Documentation updated
- [ ] Commit messages are clear
- [ ] Branch is up to date with main

### PR Description Template

```markdown
## Description

Brief description of changes

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## How Has This Been Tested?

Describe testing approach

## Checklist

- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review
- [ ] I have commented complex code
- [ ] I have updated documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix/feature works
- [ ] New and existing tests pass locally
- [ ] Any dependent changes have been merged

## Related Issues

Closes #(issue number)
```

### Review Process

1. **Automated checks** - CI pipeline must pass
2. **Code review** - Maintainers review the code
3. **Feedback** - Address review comments
4. **Approval** - At least one maintainer approval required
5. **Merge** - Maintainers merge the PR

### After Merge

- Delete your feature branch
- Sync your fork with upstream
- Celebrate your contribution!

## Reporting Issues

### Bug Reports

Include:

- **Description:** Clear description of the bug
- **Steps to reproduce:** Detailed steps
- **Expected behavior:** What should happen
- **Actual behavior:** What actually happens
- **Environment:**
  - OS: Windows/macOS/Linux
  - Go version: `go version`
  - GoChat version/commit
- **Logs:** Relevant error messages or logs
- **Screenshots:** If applicable

**Example:**

```markdown
## Bug: WebSocket connection fails on Windows

### Description

WebSocket connections fail immediately after upgrade on Windows 11.

### Steps to Reproduce

1. Start server with `.\bin\gochat.exe`
2. Open test page at http://localhost:8080/test
3. Click "Connect"

### Expected Behavior

WebSocket connection establishes successfully

### Actual Behavior

Connection fails with error: "WebSocket handshake failed"

### Environment

- OS: Windows 11 Pro
- Go version: go1.25.1 windows/amd64
- GoChat version: commit abc123

### Logs
```

Error: WebSocket upgrade failed: ...

```

```

### Feature Requests

Include:

- **Problem statement:** What problem does this solve?
- **Proposed solution:** How should it work?
- **Alternatives considered:** Other approaches
- **Use case:** Real-world scenario
- **Impact:** Who benefits from this?

## Community

### Getting Help

- **Documentation:** Check [docs/](../docs) first
- **Issues:** Search existing issues
- **Discussions:** Use GitHub Discussions for questions
- **Pull Requests:** Reference related issues

### Communication

- Be patient - maintainers volunteer their time
- Be respectful and constructive
- Help others when you can
- Share your knowledge

## Recognition

Contributors are recognized in:

- GitHub contributors page
- Release notes
- Project documentation

Thank you for contributing to GoChat!

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Setup instructions
- [Development Guide](DEVELOPMENT.md) - Development workflow and tools
- [Building](BUILDING.md) - Build instructions
- [API Documentation](API.md) - WebSocket API reference
