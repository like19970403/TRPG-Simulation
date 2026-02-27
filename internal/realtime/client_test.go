package realtime

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var testUpgrader = websocket.Upgrader{}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// setupWSPair creates a test HTTP server with a WebSocket handler,
// connects a client, and returns the server-side and client-side connections.
func setupWSPair(t *testing.T) (serverConn *websocket.Conn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()
	serverConnCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		serverConnCh <- conn
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	cc, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	sc := <-serverConnCh

	return sc, cc, func() {
		cc.Close()
		sc.Close()
		srv.Close()
	}
}

func TestClient_SendReceive(t *testing.T) {
	sc, cc, cleanup := setupWSPair(t)
	defer cleanup()

	// Create a minimal room with required channels for the client.
	room := &Room{
		incoming:   make(chan incomingMessage, 10),
		unregister: make(chan *Client, 10),
	}

	client := NewClient(sc, room, "user-1", "test", RolePlayer, testLogger())
	client.Start()

	// Send a message through the client's send channel.
	testMsg := []byte(`{"type":"test"}`)
	client.send <- testMsg

	// Read it from the client-side connection.
	cc.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := cc.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(msg) != string(testMsg) {
		t.Errorf("got %q, want %q", string(msg), string(testMsg))
	}
}

func TestClient_ReadPump_ForwardsToRoom(t *testing.T) {
	sc, cc, cleanup := setupWSPair(t)
	defer cleanup()

	room := &Room{
		incoming:   make(chan incomingMessage, 10),
		unregister: make(chan *Client, 10),
	}

	client := NewClient(sc, room, "user-1", "test", RolePlayer, testLogger())
	client.Start()

	// Write from client-side.
	testMsg := []byte(`{"type":"hello"}`)
	if err := cc.WriteMessage(websocket.TextMessage, testMsg); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Verify it arrives in room.incoming.
	select {
	case msg := <-room.incoming:
		if string(msg.data) != string(testMsg) {
			t.Errorf("got %q, want %q", string(msg.data), string(testMsg))
		}
		if msg.client != client {
			t.Error("wrong client in incoming message")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message in room.incoming")
	}
}

func TestClient_Close_TriggersUnregister(t *testing.T) {
	sc, cc, cleanup := setupWSPair(t)
	defer cleanup()

	room := &Room{
		incoming:   make(chan incomingMessage, 10),
		unregister: make(chan *Client, 10),
	}

	client := NewClient(sc, room, "user-1", "test", RolePlayer, testLogger())
	client.Start()

	// Close the client-side connection to trigger readPump exit.
	cc.Close()

	// The client should send itself to room.unregister.
	select {
	case c := <-room.unregister:
		if c != client {
			t.Error("wrong client in unregister")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for unregister")
	}
}

func TestClient_CloseSendChannel_SendsCloseFrame(t *testing.T) {
	sc, cc, cleanup := setupWSPair(t)
	defer cleanup()

	room := &Room{
		incoming:   make(chan incomingMessage, 10),
		unregister: make(chan *Client, 10),
	}

	client := NewClient(sc, room, "user-1", "test", RolePlayer, testLogger())
	client.Start()

	// Closing the send channel should make writePump send a close frame.
	close(client.send)

	// The client-side should receive a close or error.
	cc.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := cc.ReadMessage()
	if err == nil {
		t.Fatal("expected error after close, got nil")
	}
}
