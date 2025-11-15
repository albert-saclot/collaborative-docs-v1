package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"collaborative-docs/internal/server"
)

func main() {
	// Create server with configuration
	srv := server.New(server.Config{
		Port:       ":8080",
		StaticDir:  "static",
		LogEnabled: true,
	})

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down server...")
		if err := srv.Shutdown(); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	// Start server (blocks until error or shutdown)
	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
