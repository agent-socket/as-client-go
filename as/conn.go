package as

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/agent-socket/as-client-go/types"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	authHeaderKey = "Authorization"
	bearerPrefix  = "Bearer "

	// Wire protocol message types from the server.
	wireTypeConnected = "connected"
	wireTypeHeartbeat = "heartbeat"
)

var errNotConnected = errors.New("not connected")
var errAlreadyDialed = errors.New("already connected")

// Client is the event-driven WebSocket client for agent-socket.
type Client struct {
	endpoint string
	token    string

	mu       sync.Mutex
	conn     *websocket.Conn
	dialing  bool
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
	if c.conn != nil || c.dialing {
		c.mu.Unlock()
		return errAlreadyDialed
	}
	c.dialing = true
	c.mu.Unlock()

	conn, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			authHeaderKey: []string{bearerPrefix + c.token},
		},
	})
	if err != nil {
		c.mu.Lock()
		c.dialing = false
		c.mu.Unlock()
		if serverErr := parseDialError(resp); serverErr != nil {
			return serverErr
		}
		return fmt.Errorf("websocket dial: %w", err)
	}

	done := make(chan struct{})

	c.mu.Lock()
	c.conn = conn
	c.done = done
	c.dialing = false
	c.mu.Unlock()

	// Use a detached context for the read loop — the dial context governs
	// only the handshake. The connection lifetime is independent.
	go c.readLoop(context.Background(), conn, done)

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

	err := conn.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		<-done
		return err
	}

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

// writeJSON sends a JSON message on the connection. coder/websocket handles
// concurrent writes safely, so no external serialization is needed.
func (c *Client) writeJSON(ctx context.Context, v any) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return errNotConnected
	}

	return wsjson.Write(ctx, conn, v)
}

// readLoop reads messages from the WebSocket connection and dispatches events.
// It owns the given conn and done channel — when it exits, it cleans up both.
func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn, done chan struct{}) {
	defer func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
			c.done = c.closed
		}
		c.mu.Unlock()

		conn.CloseNow()
		close(done)
	}()

	connectedEmitted := false

	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			closeStatus := websocket.CloseStatus(err)
			if closeStatus == websocket.StatusNormalClosure || closeStatus == websocket.StatusGoingAway {
				c.handlers.emitDisconnected(DisconnectedEvent{})
			} else {
				c.handlers.emitError(ErrorEvent{Err: err})
				c.handlers.emitDisconnected(DisconnectedEvent{Err: err})
			}
			return
		}

		var serverMsg types.ServerMessage
		if err := json.Unmarshal(rawMsg, &serverMsg); err != nil {
			c.handlers.emitError(ErrorEvent{Err: fmt.Errorf("unmarshal server message: %w", err)})
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

// parseDialError attempts to parse a standardized server error from a failed
// WebSocket dial response. Returns nil if the response is nil or unparseable.
func parseDialError(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read server error response: %w", err)
	}

	var serverErr types.ServerError
	if json.Unmarshal(body, &serverErr) != nil {
		return nil
	}
	if serverErr.ErrorCode == "" {
		return nil
	}

	serverErr.StatusCode = resp.StatusCode
	return &serverErr
}
