package server

import (
	"log"
	"net/http"
	"strings"

	"collaborative-docs/internal/hub"

	"github.com/gorilla/websocket"
)

var allowedOrigins []string

func init() {
	allowedOrigins = []string{
		"http://localhost:8080",
		"http://127.0.0.1:8080",
	}
}

func setAllowedOrigins(origins string) {
	if origins == "" {
		return
	}
	allowedOrigins = []string{}
	for _, o := range strings.Split(origins, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}

	log.Printf("rejected websocket connection from origin: %s", origin)
	return false
}

// handleRoot redirects / to the default document.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/doc/default", http.StatusTemporaryRedirect)
}

// handleDoc serves the document editor HTML.
func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, s.config.StaticDir+"/index.html")
}

// handleWebSocket upgrades HTTP connections to WebSocket and registers clients.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	documentID, err := extractDocumentID(r.URL.Path, "/ws/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	if s.config.LogEnabled {
		log.Printf("websocket connected for document: %s", documentID)
	}

	client := hub.NewClient(s.hub, conn, documentID)
	s.hub.Register(client)

	// Start client read/write pumps
	go client.WritePump()
	go client.ReadPump()
}

// extractDocumentID parses and validates a document ID from a URL path.
func extractDocumentID(path, prefix string) (string, error) {
	documentID := strings.TrimSpace(strings.TrimPrefix(path, prefix))

	if documentID == "" {
		return "", &ValidationError{Field: "documentID", Reason: "required in URL path"}
	}

	if !isValidDocumentID(documentID) {
		return "", &ValidationError{
			Field:  "documentID",
			Reason: "must contain only alphanumeric characters, hyphens, and underscores",
		}
	}

	return documentID, nil
}

// isValidDocumentID validates document ID format.
func isValidDocumentID(id string) bool {
	if len(id) == 0 || len(id) > 100 {
		return false
	}

	for _, c := range id {
		if !isAlphanumericOrHyphenUnderscore(c) {
			return false
		}
	}

	return true
}

func isAlphanumericOrHyphenUnderscore(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' ||
		c == '_'
}

// ValidationError represents a validation failure.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return e.Field + " " + e.Reason
}
