# Deployment Guide

This guide covers deploying GoChat to production environments with proper security, scalability, and reliability.

## Table of Contents

- [Production Deployment Overview](#production-deployment-overview)
- [Reverse Proxy Configuration](#reverse-proxy-configuration)
- [TLS/WSS Setup](#tlswss-setup)
- [Process Management](#process-management)
- [Docker Deployment](#docker-deployment)
- [Monitoring and Logging](#monitoring-and-logging)
- [Performance Tuning](#performance-tuning)

## Production Deployment Overview

### Architecture

GoChat should always run behind a reverse proxy in production:

```
Internet → Reverse Proxy (Nginx/Caddy) → GoChat Server
           (TLS termination)              (localhost:8080)
```

### Why a Reverse Proxy?

- **TLS/SSL termination** - Handle HTTPS/WSS encryption
- **Load balancing** - Distribute traffic across multiple instances
- **Security** - Additional protection layer
- **Static files** - Serve static assets efficiently
- **Request filtering** - Block malicious requests
- **Logging** - Centralized access logs

### Deployment Checklist

- [ ] Build optimized binary (`make release`)
- [ ] Configure allowed origins for your domain
- [ ] Set up reverse proxy (Nginx or Caddy)
- [ ] Configure TLS certificate (Let's Encrypt recommended)
- [ ] Set up process management (systemd, Docker, or supervisor)
- [ ] Configure firewall rules
- [ ] Set up monitoring and logging
- [ ] Test failover and restart scenarios
- [ ] Document rollback procedures

## Reverse Proxy Configuration

### Nginx Configuration

**File:** `/etc/nginx/sites-available/gochat`

```nginx
upstream gochat_backend {
    server 127.0.0.1:8080;
    # For multiple instances (load balancing):
    # server 127.0.0.1:8080;
    # server 127.0.0.1:8081;
    # server 127.0.0.1:8082;
}

server {
    listen 443 ssl http2;
    server_name chat.yourdomain.com;

    # TLS Configuration
    ssl_certificate /etc/letsencrypt/live/chat.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/chat.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # SSL session cache for performance
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # WebSocket Endpoint
    location /ws {
        proxy_pass http://gochat_backend;
        proxy_http_version 1.1;

        # WebSocket upgrade headers
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Standard proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket timeouts (24 hours)
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;

        # Disable buffering for real-time communication
        proxy_buffering off;
    }

    # Health Check and Test Page
    location / {
        proxy_pass http://gochat_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Access and error logs
    access_log /var/log/nginx/gochat-access.log;
    error_log /var/log/nginx/gochat-error.log;
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name chat.yourdomain.com;
    return 301 https://$server_name$request_uri;
}
```

**Enable the configuration:**

```bash
# Create symbolic link
sudo ln -s /etc/nginx/sites-available/gochat /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

### Caddy Configuration

Caddy automatically handles TLS certificates via Let's Encrypt.

**File:** `Caddyfile`

```caddy
chat.yourdomain.com {
    # Automatic HTTPS via Let's Encrypt

    # Security Headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    # WebSocket Endpoint
    @websocket {
        path /ws
    }
    handle @websocket {
        reverse_proxy localhost:8080 {
            # Preserve client IP
            header_up X-Real-IP {remote_host}
            header_up X-Forwarded-For {remote_host}
            header_up X-Forwarded-Proto {scheme}
        }
    }

    # Health Check and Test Page
    handle {
        reverse_proxy localhost:8080
    }

    # Logging
    log {
        output file /var/log/caddy/gochat.log
        format json
    }
}
```

**Run Caddy:**

```bash
# Run directly
caddy run

# Or as a service
sudo systemctl enable caddy
sudo systemctl start caddy
sudo systemctl status caddy
```

## TLS/WSS Setup

### Why TLS is Required

- **Browser security:** Modern browsers require WSS for HTTPS pages
- **Data encryption:** Protects messages from eavesdropping
- **MITM prevention:** Prevents man-in-the-middle attacks
- **Production requirement:** Always use TLS in production

### Using Let's Encrypt (Free)

#### With Nginx

**Install Certbot:**

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install certbot python3-certbot-nginx

# RHEL/CentOS
sudo yum install certbot python3-certbot-nginx
```

**Obtain Certificate:**

```bash
sudo certbot --nginx -d chat.yourdomain.com
```

**Auto-renewal:**

Certbot automatically sets up a cron job or systemd timer. Test renewal:

```bash
sudo certbot renew --dry-run
```

#### With Caddy

Caddy automatically obtains and renews TLS certificates from Let's Encrypt. No additional configuration needed!

### Custom TLS Certificates

If using a commercial certificate:

**Nginx:**

```nginx
ssl_certificate /path/to/your/fullchain.pem;
ssl_certificate_key /path/to/your/privkey.pem;
```

**Caddy:**

```caddy
chat.yourdomain.com {
    tls /path/to/cert.pem /path/to/key.pem
}
```

### Client Configuration

After setting up TLS, update clients to use WSS:

```javascript
const ws = new WebSocket("wss://chat.yourdomain.com/ws");
```

### Server Configuration

Update allowed origins in `internal/server/config.go`:

```go
AllowedOrigins: []string{
    "https://chat.yourdomain.com",
    "https://www.yourdomain.com",
}
```

## Process Management

### Systemd (Linux)

**File:** `/etc/systemd/system/gochat.service`

```ini
[Unit]
Description=GoChat WebSocket Server
After=network.target
Documentation=https://github.com/Tyrowin/gochat

[Service]
Type=simple
User=gochat
Group=gochat
WorkingDirectory=/opt/gochat
ExecStart=/opt/gochat/bin/gochat
Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/gochat

# Resource limits
LimitNOFILE=65536

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=gochat

[Install]
WantedBy=multi-user.target
```

**Setup:**

```bash
# Create user
sudo useradd -r -s /bin/false gochat

# Create directory and set ownership
sudo mkdir -p /opt/gochat/bin
sudo chown -R gochat:gochat /opt/gochat

# Copy binary
sudo cp bin/gochat /opt/gochat/bin/

# Enable and start service
sudo systemctl enable gochat
sudo systemctl start gochat

# Check status
sudo systemctl status gochat

# View logs
sudo journalctl -u gochat -f
```

**Management Commands:**

```bash
# Start
sudo systemctl start gochat

# Stop
sudo systemctl stop gochat

# Restart
sudo systemctl restart gochat

# Reload (if supporting graceful reload)
sudo systemctl reload gochat

# Check status
sudo systemctl status gochat

# View logs
sudo journalctl -u gochat -n 100 -f
```

### Supervisor (Alternative)

**File:** `/etc/supervisor/conf.d/gochat.conf`

```ini
[program:gochat]
command=/opt/gochat/bin/gochat
directory=/opt/gochat
user=gochat
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/var/log/gochat/gochat.log
```

## Docker Deployment

GoChat includes production-ready Docker support with a multi-stage build process, minimal image size, and security best practices.

### Quick Start with Docker

**1. Copy and configure environment file:**

```bash
cp .env.example .env
# Edit .env with your configuration
```

**2. Build and run with Docker Compose:**

```bash
docker-compose up -d
```

**3. Check logs:**

```bash
docker-compose logs -f gochat
```

### Environment Configuration

GoChat can be configured using environment variables. See `.env.example` for all available options:

```bash
# Server Configuration
SERVER_PORT=:8080

# Allowed Origins for CORS (comma-separated)
ALLOWED_ORIGINS=http://localhost:8080,https://chat.example.com

# Maximum Message Size (bytes)
MAX_MESSAGE_SIZE=512

# Rate Limiting
RATE_LIMIT_BURST=5
RATE_LIMIT_REFILL_INTERVAL=1
```

### Docker Compose Configuration

The included `docker-compose.yml` provides:

- Container health checks
- Automatic restarts
- Network isolation
- Easy environment configuration

**File:** `docker-compose.yml`

```yaml
version: "3.8"

services:
  gochat:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: gochat-server
    ports:
      - "8080:8080"
    environment:
      - SERVER_PORT=:8080
      - ALLOWED_ORIGINS=http://localhost:8080,https://chat.example.com
      - MAX_MESSAGE_SIZE=512
      - RATE_LIMIT_BURST=5
      - RATE_LIMIT_REFILL_INTERVAL=1
    restart: unless-stopped
    healthcheck:
      test:
        [
          "CMD",
          "wget",
          "--no-verbose",
          "--tries=1",
          "--spider",
          "http://localhost:8080/",
        ]
      interval: 30s
      timeout: 3s
      start_period: 5s
      retries: 3
    networks:
      - gochat-network

networks:
  gochat-network:
    driver: bridge
```

### Manual Docker Build and Run

**Build the image:**

```bash
docker build -t gochat:latest .
```

**Run the container:**

```bash
docker run -d \
  --name gochat \
  -p 8080:8080 \
  -e SERVER_PORT=:8080 \
  -e ALLOWED_ORIGINS="https://chat.example.com" \
  -e MAX_MESSAGE_SIZE=512 \
  -e RATE_LIMIT_BURST=10 \
  -e RATE_LIMIT_REFILL_INTERVAL=2 \
  --restart unless-stopped \
  gochat:latest
```

### Dockerfile Details

The Dockerfile uses a multi-stage build process:

**Stage 1: Build**

- Based on `golang:1.25.1-alpine`
- Compiles the application with optimizations
- Strips debug information to reduce binary size

**Stage 2: Runtime**

- Based on `alpine:3.20` (minimal size)
- Runs as non-root user (security)
- Includes health check
- Only contains the compiled binary and essential dependencies

### Production Docker Deployment

**1. Build for production:**

```bash
docker build --no-cache -t gochat:1.0.0 .
```

**2. Tag for registry:**

```bash
docker tag gochat:1.0.0 your-registry.com/gochat:1.0.0
docker tag gochat:1.0.0 your-registry.com/gochat:latest
```

**3. Push to registry:**

```bash
docker push your-registry.com/gochat:1.0.0
docker push your-registry.com/gochat:latest
```

**4. Deploy on server:**

```bash
# Pull the image
docker pull your-registry.com/gochat:1.0.0

# Run with production config
docker run -d \
  --name gochat \
  -p 8080:8080 \
  --env-file .env \
  --restart always \
  your-registry.com/gochat:1.0.0
```

### Docker Behind Reverse Proxy

When running Docker behind Nginx or Caddy:

**docker-compose.yml:**

```yaml
services:
  gochat:
    build: .
    expose:
      - "8080"
    environment:
      - SERVER_PORT=:8080
      - ALLOWED_ORIGINS=https://chat.example.com
    networks:
      - web

networks:
  web:
    external: true
```

**Nginx configuration:**

```nginx
upstream gochat {
    server gochat:8080;
}

server {
    listen 443 ssl http2;
    server_name chat.example.com;

    location / {
        proxy_pass http://gochat;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Docker Management Commands

```bash
# View logs
docker-compose logs -f gochat

# Restart container
docker-compose restart gochat

# Stop and remove
docker-compose down

# Update and redeploy
docker-compose pull
docker-compose up -d

# Check health status
docker inspect --format='{{.State.Health.Status}}' gochat-server

# Execute command in container
docker exec -it gochat-server /bin/sh
```

### Security Best Practices

The Dockerfile includes several security features:

1. **Non-root user:** Application runs as user `gochat` (UID 1000)
2. **Minimal base image:** Alpine Linux reduces attack surface
3. **No unnecessary tools:** Only essential runtime dependencies
4. **Read-only filesystem:** Can be enforced with `--read-only` flag
5. **Health checks:** Automated container health monitoring

### Docker Performance Optimization

**Multi-stage build benefits:**

- Final image size: ~15MB (vs ~1GB with full Go image)
- Faster deployment and startup
- Reduced network transfer time
- Lower storage costs

**Build cache optimization:**

- Dependencies downloaded separately for better caching
- Source code copied last to maximize cache hits

          "http://localhost:8080/",
        ]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

networks:
gochat-network:
driver: bridge

````

### Build and Run

```bash
# Build image
docker build -t gochat:latest .

# Run container
docker run -d \
  --name gochat \
  --restart unless-stopped \
  -p 8080:8080 \
  gochat:latest

# Using Docker Compose
docker-compose up -d

# View logs
docker logs -f gochat

# Stop
docker stop gochat

# Remove
docker rm gochat
````

## Monitoring and Logging

### Logging Best Practices

1. **Structured logging** - Use JSON format for easier parsing
2. **Log levels** - Implement INFO, WARN, ERROR levels
3. **Log rotation** - Prevent disk space issues
4. **Centralized logging** - Use ELK stack, Splunk, or similar

### Health Checks

```bash
# Simple health check
curl http://localhost:8080/

# With reverse proxy
curl https://chat.yourdomain.com/
```

### Monitoring Metrics

Consider monitoring:

- Active WebSocket connections
- Messages per second
- Connection errors
- Rate limit violations
- Memory usage
- CPU usage
- Network I/O

### Tools

- **Prometheus** - Metrics collection
- **Grafana** - Visualization
- **AlertManager** - Alerting
- **ELK Stack** - Log aggregation

## Performance Tuning

### System Limits

**File:** `/etc/security/limits.conf`

```
gochat soft nofile 65536
gochat hard nofile 65536
```

### Kernel Parameters

**File:** `/etc/sysctl.conf`

```
# Increase TCP buffer sizes
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# Increase connection backlog
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 4096

# Enable TCP Fast Open
net.ipv4.tcp_fastopen = 3
```

Apply changes:

```bash
sudo sysctl -p
```

### Go Runtime

Set GOMAXPROCS to match CPU cores (usually automatic).

### Load Balancing

For high traffic, run multiple instances:

```nginx
upstream gochat_backend {
    least_conn;  # Load balancing algorithm
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}
```

## Firewall Configuration

### UFW (Ubuntu)

```bash
# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Block direct access to GoChat port
sudo ufw deny 8080/tcp

# Enable firewall
sudo ufw enable
```

### iptables

```bash
# Allow HTTP/HTTPS
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Block external access to port 8080
iptables -A INPUT -p tcp --dport 8080 -i eth0 -j DROP
iptables -A INPUT -p tcp --dport 8080 -i lo -j ACCEPT
```

## Backup and Recovery

### What to Backup

- Configuration files
- TLS certificates (if not using Let's Encrypt)
- Custom code modifications
- Deployment scripts

### Rollback Plan

1. Keep previous binary versions
2. Document configuration changes
3. Test rollback procedure
4. Have a tested recovery process

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Initial setup
- [API Documentation](API.md) - WebSocket API
- [Security](SECURITY.md) - Security features and best practices
- [Building](BUILDING.md) - Build and compilation instructions
