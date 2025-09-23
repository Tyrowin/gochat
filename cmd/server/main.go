/*
GoChat is a real-time WebSocket-based chat server.

The server provides WebSocket endpoints for real-time communication
and includes a built-in test page for development and testing.

Usage:

	gochat

The server will start on port 8080 by default and provide the following endpoints:

  - / - Health check endpoint
  - /ws - WebSocket endpoint for chat connections
  - /test - HTML test page for WebSocket functionality
*/
package main

import (
	"fmt"
	"log"

	"github.com/Tyrowin/gochat/internal/server"
)

func main() {
	fmt.Println("Starting GoChat server...")

	config := server.NewConfig()
	server.StartHub()
	mux := server.SetupRoutes()
	httpServer := server.CreateServer(config.Port, mux)

	log.Fatal(server.StartServer(httpServer))
}
