package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"collaborative-docs/internal/server"
)

func main() {
	port := getEnv("PORT", "8080")
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	srv := server.New(server.Config{
		Port:           port,
		StaticDir:      getEnv("STATIC_DIR", "static"),
		LogEnabled:     getEnv("LOG_ENABLED", "true") == "true",
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", ""),
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down server...")
		if err := srv.Shutdown(); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
