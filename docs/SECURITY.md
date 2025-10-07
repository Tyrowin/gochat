# Security Documentation

GoChat implements multiple layers of security to protect against common WebSocket vulnerabilities and abuse.

## Table of Contents

- [Origin Validation](#origin-validation)
- [Rate Limiting](#rate-limiting)
- [Message Size Limits](#message-size-limits)
- [Security Scanning](#security-scanning)
- [Security Best Practices](#security-best-practices)
- [Reporting Security Issues](#reporting-security-issues)

## Origin Validation

The server validates the `Origin` header of all WebSocket connection requests to prevent Cross-Site WebSocket Hijacking (CSWSH) attacks.

### How It Works

- Every WebSocket upgrade request must include a valid `Origin` header
- The origin is normalized (scheme and host are lowercased)
- Only origins in the allowed list can establish connections
- Requests from disallowed origins are rejected with a 403 Forbidden response
- The server logs all blocked connection attempts

### Default Configuration

```go
AllowedOrigins: []string{
    "http://localhost:8080",
}
```

### Customizing Allowed Origins

Modify the allowed origins in `internal/server/config.go`:

```go
AllowedOrigins: []string{
    "http://localhost:8080",
    "https://yourdomain.com",
    "https://www.yourdomain.com",
}
```

### Allow All Origins (Development Only)

**WARNING:** Never use this in production!

```go
AllowedOrigins: []string{"*"}
```

### Security Implications

- **Prevents CSWSH attacks:** Malicious websites cannot connect to your chat server
- **Whitelist approach:** Only explicitly allowed origins can connect
- **Production requirement:** Always configure actual domain names, never use `*`
- **Multiple domains:** Include all legitimate frontend domains (www, subdomains, etc.)

### Testing Origin Validation

```bash
# This should succeed (if localhost:8080 is allowed)
websocat ws://localhost:8080/ws -H "Origin: http://localhost:8080"

# This should fail (origin not in allowed list)
websocat ws://localhost:8080/ws -H "Origin: http://malicious-site.com"
```

## Rate Limiting

GoChat implements per-connection token bucket rate limiting to prevent message flooding and abuse.

### How It Works

- Each client connection has its own rate limiter
- The limiter starts with a burst of tokens
- Each message sent consumes 1 token
- Tokens refill at a configured rate
- When a client runs out of tokens, the connection is closed

### Default Configuration

```go
RateLimit: RateLimitConfig{
    Burst:          5,              // Allow 5 messages immediately
    RefillInterval: time.Second,    // Refill 5 tokens per second
}
```

This allows:

- **Burst:** Up to 5 messages instantly
- **Sustained rate:** 5 messages per second
- **Recovery:** Full burst capacity restored every second

### Example Scenarios

**Normal Usage:**

- Client sends 3 messages per second
- Result: No issues, well within limits

**Burst Traffic:**

- Client sends 5 messages instantly
- Result: Allowed (uses burst capacity)
- Tokens refill over the next second

**Abuse Attempt:**

- Client attempts to send 100 messages instantly
- Result: First 5 succeed, connection closed after token exhaustion

**Sustained Flooding:**

- Client sends 10 messages per second continuously
- Result: First 5 succeed, then connection closed

### Customizing Rate Limits

Modify the configuration in `internal/server/config.go`:

```go
RateLimit: RateLimitConfig{
    Burst:          10,                    // Higher burst for occasional spikes
    RefillInterval: 500 * time.Millisecond, // Faster refill (10 tokens/sec)
}
```

**Recommendations:**

- **Chat applications:** 5-10 messages/second is typically sufficient
- **High-frequency trading:** May need higher limits (50-100/sec)
- **Public servers:** Keep limits conservative to prevent abuse
- **Private networks:** Can be more lenient

### Benefits

- **DoS prevention:** Stops malicious clients from overwhelming the server
- **Resource protection:** Prevents excessive CPU and bandwidth usage
- **Fair sharing:** One abusive client cannot affect others
- **Legitimate bursts:** Allows normal usage patterns (quick replies, etc.)

### Rate Limit Headers

The server does not currently expose rate limit information in headers, but clients are disconnected if limits are exceeded. Consider implementing exponential backoff in your client code.

## Message Size Limits

Message size limits prevent memory exhaustion attacks and reduce bandwidth consumption.

### Default Limits

- **Maximum message size:** 512 bytes
- **Read buffer size:** 1024 bytes
- **Write buffer size:** 1024 bytes

### How It Works

- WebSocket upgrader sets buffer sizes
- Configuration enforces maximum message size
- Messages exceeding the limit cause the connection to close
- No warning is sent - connection is terminated immediately

### Customizing Message Size

In `internal/server/config.go`:

```go
MaxMessageSize: 1024, // Allow larger messages (in bytes)
```

**Note:** Also update the WebSocket upgrader buffer sizes in `internal/server/handlers.go` if needed:

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  2048,  // Increase if needed
    WriteBufferSize: 2048,  // Increase if needed
    CheckOrigin:     checkOrigin,
}
```

### Size Recommendations

| Use Case              | Recommended Size |
| --------------------- | ---------------- |
| Short chat messages   | 256-512 bytes    |
| Regular chat messages | 512-1024 bytes   |
| Rich text/links       | 1024-2048 bytes  |
| JSON with metadata    | 2048-4096 bytes  |

### Calculating Message Size

JSON overhead adds to your message size:

```json
{ "content": "Hello" }
```

- Content: 5 bytes
- JSON structure: 15 bytes
- Total: 20 bytes

A 512-byte limit allows approximately 500 characters of message content.

## Security Scanning

GoChat uses automated security scanning tools in the CI/CD pipeline and for local development.

### Vulnerability Scanning

**govulncheck** - Official Go vulnerability database scanner

- Scans all dependencies for known CVEs
- Runs on every commit and pull request
- Checks against the Go vulnerability database
- Zero-configuration security monitoring

```bash
# Run locally
govulncheck ./...
```

### Static Security Analysis

**gosec** - Security-focused Go static analyzer

- Checks for common security issues:
  - SQL injection vulnerabilities
  - Command injection risks
  - Unsafe use of cryptography
  - File permission issues
  - Hardcoded credentials
  - And more...

```bash
# Run locally
gosec ./...
```

### Running All Security Scans

```bash
# Run comprehensive security scan
make security-scan

# This runs:
# - govulncheck for dependency vulnerabilities
# - gosec for code security issues
```

### Dependency Auditing

- **Regular updates:** Dependencies are updated frequently
- **Security patches:** Critical vulnerabilities are patched immediately
- **License compliance:** All dependencies use permissive licenses
- **Minimal dependencies:** Only essential packages are used

### CI/CD Security Pipeline

Every commit and pull request runs:

1. **govulncheck** - Dependency vulnerability scan
2. **gosec** - Static security analysis
3. **Dependency check** - License and update verification
4. **Test suite** - Including security-focused tests

## Security Best Practices

### Production Deployment

1. **Always use TLS/WSS**

   - Never use plain WS in production
   - Encrypt all traffic between clients and server
   - See [Deployment Guide](DEPLOYMENT.md)

2. **Run behind a reverse proxy**

   - Add additional security layers
   - Implement rate limiting at proxy level
   - Filter malicious requests

3. **Configure allowed origins**

   - Never use `*` (allow all) in production
   - List all legitimate frontend domains
   - Include all subdomains and variants

4. **Monitor and log**

   - Track failed connection attempts
   - Monitor rate limit violations
   - Set up alerts for unusual patterns

5. **Keep dependencies updated**
   - Run `govulncheck` regularly
   - Update Go to the latest version
   - Patch vulnerabilities promptly

### Network Security

1. **Firewall configuration**

   - Only expose necessary ports (443 for WSS)
   - Block direct access to the Go server
   - Use reverse proxy as the public endpoint

2. **DDoS protection**

   - Use a CDN or DDoS protection service
   - Implement connection limits
   - Rate limit at multiple layers

3. **IP whitelisting** (if applicable)
   - Restrict access to known IP ranges
   - Implement at firewall or reverse proxy level

### Application Security

1. **Input validation**

   - Validate all message content
   - Sanitize data before processing
   - Reject malformed JSON

2. **Error handling**

   - Don't expose internal errors to clients
   - Log errors securely
   - Avoid information leakage

3. **Authentication** (future consideration)
   - Currently, GoChat has no built-in authentication
   - Implement authentication at the application level
   - Consider JWT tokens or session-based auth

## Reporting Security Issues

If you discover a security vulnerability in GoChat, please follow responsible disclosure practices:

### Do Not

- Open a public GitHub issue
- Disclose the vulnerability publicly
- Exploit the vulnerability

### Do

1. **Contact maintainers directly**

   - Email: (Add contact email)
   - Private message on GitHub

2. **Provide details**

   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if known)

3. **Allow time for a fix**
   - Give maintainers reasonable time to address the issue
   - Coordinate disclosure timeline
   - Receive credit in security advisory (if desired)

### Security Response

- **Acknowledgment:** Within 48 hours
- **Initial assessment:** Within 7 days
- **Fix timeline:** Depends on severity
  - Critical: 1-7 days
  - High: 1-2 weeks
  - Medium: 2-4 weeks
  - Low: Next release cycle

## Security Checklist for Production

- [ ] TLS/WSS enabled with valid certificate
- [ ] Allowed origins configured (no `*`)
- [ ] Running behind reverse proxy
- [ ] Rate limits configured appropriately
- [ ] Message size limits set
- [ ] Security scanning in CI/CD
- [ ] Dependencies up to date
- [ ] Firewall rules configured
- [ ] Monitoring and logging enabled
- [ ] Incident response plan in place

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Installation and setup
- [API Documentation](API.md) - WebSocket API details
- [Deployment Guide](DEPLOYMENT.md) - Production deployment with TLS
- [Development Guide](DEVELOPMENT.md) - Security testing and development
