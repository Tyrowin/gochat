// Package main implements the entry point for the GoChat server.
package main

import (
	"fmt"
	"log"

	"github.com/Tyrowin/gochat/internal/server"
)

func main() {
	fmt.Println("Starting GoChat server...")

	// Create configuration
	config := server.NewConfig()

	// Setup routes
	mux := server.SetupRoutes()

	// Create and start server
	httpServer := server.CreateServer(config.Port, mux)

	log.Fatal(server.StartServer(httpServer))
}
