package realtime

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 40 * time.Second // 30s ping interval + 10s grace
	pingPeriod     = 30 * time.Second
	maxMessageSize = 4096
	sendBufferSize = 256
)

// Client represents a single WebSocket connection.
type Client struct {
	conn     *websocket.Conn
	room     *Room
	send     chan []byte
	userID   string
	username string
	role     SenderRole
	logger   *slog.Logger
	done     chan struct{}
}

// NewClient creates a Client. Does NOT start pumps yet — call Start().
func NewClient(conn *websocket.Conn, room *Room, userID, username string, role SenderRole, logger *slog.Logger) *Client {
	return &Client{
		conn:     conn,
		room:     room,
		send:     make(chan []byte, sendBufferSize),
		userID:   userID,
		username: username,
		role:     role,
		logger:   logger,
		done:     make(chan struct{}),
	}
}

// Send queues a message for the client. Returns false if the buffer is full.
func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// UserID returns the client's user ID.
func (c *Client) UserID() string { return c.userID }

// Role returns the client's role.
func (c *Client) Role() SenderRole { return c.role }

// Start launches the read and write pump goroutines.
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// readPump reads messages from the WebSocket and forwards to the room.
func (c *Client) readPump() {
	defer func() {
		c.room.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Warn("client read error", "user", c.userID, "error", err)
			}
			return
		}
		c.room.incoming <- incomingMessage{client: c, data: msg}
	}
}

// writePump sends messages from c.send to the WebSocket and handles ping.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		close(c.done)
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed by room; send close frame.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
