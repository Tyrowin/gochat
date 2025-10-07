# Getting Started with GoChat

This guide will help you get GoChat up and running on your system.

## Prerequisites

- Go 1.25.1 or later
- Git
- Make (optional but recommended)
  - **Windows**: Install via [Chocolatey](https://chocolatey.org/) - `choco install make`
  - **macOS**: Install Xcode Command Line Tools - `xcode-select --install`
  - **Linux**: Usually pre-installed (`apt install make` or `yum install make`)

**Note**: If you don't have Make, you can use Go commands directly (see examples below).

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/Tyrowin/gochat.git
   cd gochat
   ```

2. Verify Go installation:

   ```bash
   go version
   # Should show Go 1.25.1 or later
   ```

## Building the Server

### Using Make (recommended)

```bash
make build
```

### Using Go directly

**On Windows (PowerShell):**

```powershell
go build -o bin\gochat.exe .\cmd\server
```

**On macOS/Linux:**

```bash
go build -o bin/gochat ./cmd/server
```

The compiled binary will be placed in the `bin/` directory.

## Running the Server

### Start the Server

**On Windows:**

```powershell
.\bin\gochat.exe
```

**On macOS/Linux:**

```bash
./bin/gochat
```

**Using Make:**

```bash
make run
```

### Expected Output

The server will start on `http://localhost:8080` and display:

```
Starting GoChat server...
Server starting on port :8080
Hub started - ready to accept connections
```

## Available Endpoints

Once the server is running, you can access:

- **`GET /`** - Health check endpoint

  - Returns: "GoChat server is running!"
  - Use this to verify the server is operational

- **`GET /ws`** - WebSocket connection endpoint

  - This is where clients connect for real-time chat
  - See [API Documentation](API.md) for details

- **`GET /test`** - Built-in test page
  - Interactive HTML page for testing WebSocket functionality
  - Navigate to `http://localhost:8080/test` in your browser
  - Allows you to connect, send messages, and see real-time chat

## Quick Test

1. Start the server (see above)
2. Open your browser and navigate to `http://localhost:8080/test`
3. Click "Connect" to establish a WebSocket connection
4. Type a message and click "Send"
5. Open another browser window/tab to the same URL to test multi-client chat

## Next Steps

- Learn about the [WebSocket API](API.md) to integrate GoChat into your application
- Review [Security features](SECURITY.md) to understand built-in protections
- See [Deployment Guide](DEPLOYMENT.md) for production deployment
- Check [Development Guide](DEVELOPMENT.md) to contribute or customize

## Troubleshooting

### Port Already in Use

If you see an error like "address already in use", another application is using port 8080:

**Windows:**

```powershell
# Find what's using port 8080
netstat -ano | findstr :8080
# Kill the process (replace PID with the actual process ID)
taskkill /PID <PID> /F
```

**macOS/Linux:**

```bash
# Find what's using port 8080
lsof -i :8080
# Kill the process
kill -9 <PID>
```

### Permission Denied

If you get a "permission denied" error on macOS/Linux, make the binary executable:

```bash
chmod +x bin/gochat
```

### Connection Refused

If you can't connect to the server:

1. Verify the server is running and shows "Server starting on port :8080"
2. Check your firewall settings
3. Ensure you're using the correct URL: `http://localhost:8080`
4. Try `http://127.0.0.1:8080` instead

### Origin Not Allowed

If WebSocket connections are rejected:

- By default, only `http://localhost:8080` is allowed as an origin
- To allow other origins, modify the configuration in `internal/server/config.go`
- See [Security Documentation](SECURITY.md) for details on origin validation
