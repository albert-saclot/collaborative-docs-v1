package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/albert-saclot/collaborative-docs-v1/internal/hub"
)

// Config holds server configuration.
type Config struct {
	Port           string
	StaticDir      string
	LogEnabled     bool
	AllowedOrigins string
}

// Server represents the HTTP server and its dependencies.
type Server struct {
	config     Config
	hub        *hub.Hub
	httpServer *http.Server
	mux        *http.ServeMux
}

// New creates and initializes a new Server instance.
func New(cfg Config) *Server {
	h := hub.NewHub()

	if cfg.AllowedOrigins != "" {
		setAllowedOrigins(cfg.AllowedOrigins)
	}

	s := &Server{
		config: cfg,
		hub:    h,
		mux:    http.NewServeMux(),
	}

	s.registerRoutes()

	s.httpServer = &http.Server{
		Addr:         cfg.Port,
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Run starts the hub and HTTP server. Blocks until server stops.
func (s *Server) Run() error {
	// Start hub in background
	go s.hub.Run()

	if s.config.LogEnabled {
		log.Println("hub started successfully")
		log.Printf("server starting on http://localhost%s", s.config.Port)
		log.Printf("document URLs: http://localhost%s/doc/{documentID}", s.config.Port)
		log.Printf("websocket endpoint: ws://localhost%s/ws/{documentID}", s.config.Port)
	}

	// Start HTTP server (blocks)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Shutdown gracefully stops the server and hub.
func (s *Server) Shutdown() error {
	// Shutdown hub first to stop accepting new messages
	s.hub.Shutdown()

	// Then shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/", s.handleRoot)
	s.mux.HandleFunc("/doc/", s.handleDoc)
	s.mux.HandleFunc("/ws/", s.handleWebSocket)
	s.mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(s.config.StaticDir))))
}
