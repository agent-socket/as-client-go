package as

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/agent-socket/as-client-go/types"
	"github.com/gorilla/websocket"
)

const (
	authHeaderKey = "Authorization"
	bearerPrefix  = "Bearer "

	// Wire protocol message types from the server.
	wireTypeConnected = "connected"
	wireTypeHeartbeat = "heartbeat"
)

var errNotConnected = errors.New("not connected")
var errAlreadyConnected = errors.New("already connected")

// Client is the event-driven WebSocket client for agent-socket.
type Client struct {
	endpoint string
	token    string
	dialer   *websocket.Dialer

	mu       sync.Mutex
	conn     *websocket.Conn
	handlers *handlers
	done     chan struct{}
	closed   chan struct{} // always non-nil, acts as default "not connected" sentinel
}

// New creates a new WebSocket client.
func New(endpoint, token string) *Client {
	closed := make(chan struct{})
	close(closed)
	return &Client{
		endpoint: endpoint,
		token:    token,
		dialer:   websocket.DefaultDialer,
		handlers: newHandlers(),
		closed:   closed,
		done:     closed, // done starts as closed (not connected)
	}
}

// OnMessage registers a handler for incoming messages.
func (c *Client) OnMessage(fn MessageHandler) {
	c.handlers.onMessage(fn)
}

// OnConnected registers a handler for connection events.
func (c *Client) OnConnected(fn ConnectedHandler) {
	c.handlers.onConnected(fn)
}

// OnDisconnected registers a handler for disconnection events.
func (c *Client) OnDisconnected(fn DisconnectedHandler) {
	c.handlers.onDisconnected(fn)
}

// OnHeartbeat registers a handler for heartbeat events.
func (c *Client) OnHeartbeat(fn HeartbeatHandler) {
	c.handlers.onHeartbeat(fn)
}

// OnError registers a handler for error events.
func (c *Client) OnError(fn ErrorHandler) {
	c.handlers.onError(fn)
}

// Connect opens a WebSocket connection for a named socket.
func (c *Client) Connect(ctx context.Context, socketID string) error {
	url := fmt.Sprintf("%s/%s", c.endpoint, socketID)
	return c.dial(ctx, url)
}

// ConnectEphemeral opens a WebSocket connection for an ephemeral socket.
// The assigned socket ID is delivered via the Connected event.
func (c *Client) ConnectEphemeral(ctx context.Context) error {
	url := fmt.Sprintf("%s/es", c.endpoint)
	return c.dial(ctx, url)
}

func (c *Client) dial(ctx context.Context, url string) error {
	c.mu.Lock()
	if c.conn != nil {
		c.mu.Unlock()
		return errAlreadyConnected
	}
	c.mu.Unlock()

	header := http.Header{}
	header.Set(authHeaderKey, bearerPrefix+c.token)

	conn, _, err := c.dialer.DialContext(ctx, url, header)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	done := make(chan struct{})

	c.mu.Lock()
	c.conn = conn
	c.done = done
	c.mu.Unlock()

	go c.readLoop(conn, done)

	return nil
}

// Close closes the WebSocket connection gracefully.
func (c *Client) Close() error {
	c.mu.Lock()
	conn := c.conn
	done := c.done
	c.mu.Unlock()

	if conn == nil {
		return nil
	}

	// Send close frame under the mutex to prevent concurrent writes.
	// The writeLocked helper serializes with Send/SendRaw.
	err := c.writeLocked(conn, func() error {
		return conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
	})
	if err != nil {
		// Force close if we can't send the close frame.
		conn.Close()
		<-done
		return err
	}

	// Wait for readLoop to finish.
	<-done
	return nil
}

// Done returns a channel that is closed when the connection is closed.
// Returns an already-closed channel if not connected.
func (c *Client) Done() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done
}

// writeConn acquires the write lock, verifies the connection is still active,
// and performs the write atomically. This eliminates the TOCTOU race between
// checking c.conn and writing to it.
func (c *Client) writeConn(fn func(conn *websocket.Conn) error) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return errNotConnected
	}

	return c.writeLocked(conn, func() error {
		return fn(conn)
	})
}

// writeLocked serializes a write operation. All writes to the websocket
// connection must go through this to satisfy gorilla/websocket's concurrency
// contract (no concurrent writers).
func (c *Client) writeLocked(conn *websocket.Conn, fn func() error) error {
	// We use the connection's own mutex-like behavior by holding c.mu briefly
	// to get the conn pointer, then use a dedicated write serializer.
	// Since gorilla/websocket doesn't support concurrent writes, we serialize
	// all writes through this path.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check that the connection hasn't been replaced or nilled out.
	if c.conn != conn {
		return errNotConnected
	}

	return fn()
}

// readLoop reads messages from the WebSocket connection and dispatches events.
// It owns the given conn and done channel — when it exits, it cleans up both.
func (c *Client) readLoop(conn *websocket.Conn, done chan struct{}) {
	defer func() {
		c.mu.Lock()
		// Only nil out c.conn if it's still the connection we own.
		if c.conn == conn {
			c.conn = nil
			c.done = c.closed
		}
		c.mu.Unlock()

		conn.Close()
		close(done)
	}()

	connectedEmitted := false

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.handlers.emitDisconnected(DisconnectedEvent{})
			} else {
				c.handlers.emitError(ErrorEvent{Err: err})
				c.handlers.emitDisconnected(DisconnectedEvent{Err: err})
			}
			return
		}

		var serverMsg types.ServerMessage
		if json.Unmarshal(rawMsg, &serverMsg) != nil {
			continue
		}

		switch {
		case serverMsg.Type == wireTypeConnected:
			var connected types.ConnectedMessage
			if json.Unmarshal(rawMsg, &connected) == nil {
				connectedEmitted = true
				c.handlers.emitConnected(ConnectedEvent{SocketID: connected.SocketID})
			}

		case serverMsg.Type == wireTypeHeartbeat:
			c.handlers.emitHeartbeat(HeartbeatEvent{Data: serverMsg.Data})

		case serverMsg.From != "":
			if !connectedEmitted {
				connectedEmitted = true
				c.handlers.emitConnected(ConnectedEvent{})
			}
			c.handlers.emitMessage(types.IncomingMessage{
				From: serverMsg.From,
				Data: serverMsg.Data,
			})
		}
	}
}
