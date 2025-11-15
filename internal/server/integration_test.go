package server_test

import (
	"collaborative-docs/internal/hub"
	"collaborative-docs/internal/server/testutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// serveWs is a test helper that handles WebSocket upgrade and client registration.
func serveWs(h *hub.Hub, w http.ResponseWriter, r *http.Request, documentID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	log.Printf("websocket connected for document: %s", documentID)

	client := hub.NewClient(h, conn, documentID)
	h.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

// TestWebSocketServer verifies that two clients can connect and receive broadcast messages.
func TestWebSocketServer(t *testing.T) {
	h := hub.NewHub()
	go h.Run()

	testDocID := "test-doc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(h, w, r, testDocID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn1 := testutil.MustConnect(t, wsURL)
	defer conn1.Close()

	testutil.WaitForRegistration()
	testutil.AssertClientCount(t, h, testDocID, 1)

	conn2 := testutil.MustConnect(t, wsURL)
	defer conn2.Close()

	testutil.WaitForRegistration()
	testutil.AssertClientCount(t, h, testDocID, 2)

	testMessage := "Hello from client 1"
	testutil.SendMessage(t, conn1, testMessage)
	testutil.WaitForBroadcast()

	if got := testutil.ReadNextContent(t, conn1); got != testMessage {
		t.Errorf("client 1 received %q, want %q", got, testMessage)
	}

	if got := testutil.ReadNextContent(t, conn2); got != testMessage {
		t.Errorf("client 2 received %q, want %q", got, testMessage)
	}
}

// TestMultipleClients verifies that messages broadcast to all connected clients.
func TestMultipleClients(t *testing.T) {
	h := hub.NewHub()
	go h.Run()

	testDocID := "test-doc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(h, w, r, testDocID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	numClients := 5
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conns[i] = testutil.MustConnect(t, wsURL)
		defer conns[i].Close()
	}

	time.Sleep(200 * time.Millisecond)
	testutil.AssertClientCount(t, h, testDocID, numClients)

	testMessage := "Broadcast to all"
	testutil.SendMessage(t, conns[0], testMessage)
	testutil.WaitForBroadcast()

	for i, conn := range conns {
		if got := testutil.ReadNextContent(t, conn); got != testMessage {
			t.Errorf("client %d received %q, want %q", i, got, testMessage)
		}
	}
}

// TestClientDisconnect verifies that disconnected clients are properly cleaned up.
func TestClientDisconnect(t *testing.T) {
	h := hub.NewHub()
	go h.Run()

	testDocID := "test-doc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(h, w, r, testDocID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn1 := testutil.MustConnect(t, wsURL)
	conn2 := testutil.MustConnect(t, wsURL)
	defer conn2.Close()

	testutil.WaitForRegistration()
	testutil.AssertClientCount(t, h, testDocID, 2)

	conn1.Close()
	testutil.WaitForRegistration()
	testutil.AssertClientCount(t, h, testDocID, 1)

	testMessage := "Message after disconnect"
	testutil.SendMessage(t, conn2, testMessage)
	testutil.WaitForBroadcast()

	if got := testutil.ReadNextContent(t, conn2); got != testMessage {
		t.Errorf("received %q, want %q", got, testMessage)
	}
}

// TestRapidMessages verifies that rapid message sending is handled correctly.
func TestRapidMessages(t *testing.T) {
	h := hub.NewHub()
	go h.Run()

	testDocID := "test-doc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(h, w, r, testDocID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn := testutil.MustConnect(t, wsURL)
	defer conn.Close()

	testutil.WaitForRegistration()

	numMessages := 10
	for i := 0; i < numMessages; i++ {
		testutil.SendMessage(t, conn, "Rapid message")
		time.Sleep(10 * time.Millisecond)
	}

	receivedCount := 0
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < numMessages; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
		receivedCount++
	}

	if receivedCount < 1 {
		t.Errorf("received %d messages, want at least 1", receivedCount)
	}
}

// TestEmptyMessage verifies that empty messages are handled correctly.
func TestEmptyMessage(t *testing.T) {
	h := hub.NewHub()
	go h.Run()

	testDocID := "test-doc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(h, w, r, testDocID)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn := testutil.MustConnect(t, wsURL)
	defer conn.Close()

	testutil.WaitForRegistration()

	testutil.SendMessage(t, conn, "")
	testutil.WaitForBroadcast()

	if got := testutil.ReadNextContent(t, conn); len(got) != 0 {
		t.Errorf("received non-empty message: %q", got)
	}
}
