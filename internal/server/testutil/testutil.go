package testutil

import (
	"github.com/albert-saclot/collaborative-docs-v1/internal/hub"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// MustConnect connects to a WebSocket URL and fails the test on error.
func MustConnect(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	return conn
}

// ReadNextContent reads the next content message from the connection,
// automatically skipping over any user_count system messages.
func ReadNextContent(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("failed to read message: %v", err)
		}

		var msg hub.Message
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if msg.Type == "user_count" {
				continue
			}
			if msg.Type == "content" {
				return msg.Content
			}
			return string(msgBytes)
		}

		// Legacy format - check for batched messages
		parts := strings.Split(string(msgBytes), "\n")
		for _, part := range parts {
			if !strings.HasPrefix(part, "USER_COUNT:") {
				return part
			}
		}
	}
}

// SendMessage sends a text message and fails the test on error.
func SendMessage(t *testing.T, conn *websocket.Conn, message string) {
	t.Helper()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
}

// AssertClientCount checks that the hub has the expected client count for a document.
func AssertClientCount(t *testing.T, h *hub.Hub, documentID string, want int) {
	t.Helper()
	if got := h.ClientCountForDocument(documentID); got != want {
		t.Errorf("client count for %s = %d, want %d", documentID, got, want)
	}
}

// WaitForRegistration provides a small delay for client registration to complete.
func WaitForRegistration() {
	time.Sleep(100 * time.Millisecond)
}

// WaitForBroadcast provides a small delay for message broadcasting to complete.
func WaitForBroadcast() {
	time.Sleep(50 * time.Millisecond)
}
