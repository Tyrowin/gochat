// Package main implements the entry point for the Nexus Chat Server.
package main

import (
	"fmt"
	"log"

	"github.com/Tyrowin/nexus-chat-server/internal/server"
)

func main() {
	fmt.Println("Starting Nexus Chat Server...")

	// Create configuration
	config := server.NewConfig()

	// Setup routes
	mux := server.SetupRoutes()

	// Create and start server
	httpServer := server.CreateServer(config.Port, mux)

	log.Fatal(server.StartServer(httpServer))
}
