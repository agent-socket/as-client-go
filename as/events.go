package as

import (
	"encoding/json"
	"sync"

	"github.com/agent-socket/as-client-go/types"
)

// Event represents a WebSocket event type.
type Event string

const (
	// EventMessage fires when a message is received from another socket.
	EventMessage Event = "message"
	// EventConnected fires after the WebSocket connection is established.
	// For ephemeral sockets, the ConnectedEvent includes the assigned socket ID.
	EventConnected Event = "connected"
	// EventDisconnected fires when the WebSocket connection is closed.
	EventDisconnected Event = "disconnected"
	// EventHeartbeat fires when a heartbeat message is received from the server.
	EventHeartbeat Event = "heartbeat"
	// EventError fires when a read/write error occurs on the connection.
	EventError Event = "error"
)

// ConnectedEvent is emitted when the WebSocket connection is established.
type ConnectedEvent struct{}

// DisconnectedEvent is emitted when the WebSocket connection is closed.
type DisconnectedEvent struct {
	Err error // nil for clean close
}

// ErrorEvent is emitted when a connection error occurs or the server
// sends an error frame (e.g. access denied, socket not found).
// Code is populated only for server error frames.
type ErrorEvent struct {
	Err  error
	Code string // server error code (e.g. "E3001"), empty for connection errors
}

// HeartbeatEvent is emitted when a heartbeat is received.
type HeartbeatEvent struct {
	Data json.RawMessage
}

// MessageHandler handles incoming messages.
type MessageHandler func(types.IncomingMessage)

// ConnectedHandler handles connection events.
type ConnectedHandler func(ConnectedEvent)

// DisconnectedHandler handles disconnection events.
type DisconnectedHandler func(DisconnectedEvent)

// HeartbeatHandler handles heartbeat events.
type HeartbeatHandler func(HeartbeatEvent)

// ErrorHandler handles error events.
type ErrorHandler func(ErrorEvent)

// handlers stores registered event handlers.
type handlers struct {
	mu            sync.RWMutex
	message       []MessageHandler
	connected     []ConnectedHandler
	disconnected  []DisconnectedHandler
	heartbeat     []HeartbeatHandler
	errorHandlers []ErrorHandler
}

func newHandlers() *handlers {
	return &handlers{}
}

func (h *handlers) onMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.message = append(h.message, fn)
}

func (h *handlers) onConnected(fn ConnectedHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connected = append(h.connected, fn)
}

func (h *handlers) onDisconnected(fn DisconnectedHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.disconnected = append(h.disconnected, fn)
}

func (h *handlers) onHeartbeat(fn HeartbeatHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.heartbeat = append(h.heartbeat, fn)
}

func (h *handlers) onError(fn ErrorHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorHandlers = append(h.errorHandlers, fn)
}

func (h *handlers) emitMessage(msg types.IncomingMessage) {
	h.mu.RLock()
	fns := make([]MessageHandler, len(h.message))
	copy(fns, h.message)
	h.mu.RUnlock()
	for _, fn := range fns {
		fn(msg)
	}
}

func (h *handlers) emitConnected(evt ConnectedEvent) {
	h.mu.RLock()
	fns := make([]ConnectedHandler, len(h.connected))
	copy(fns, h.connected)
	h.mu.RUnlock()
	for _, fn := range fns {
		fn(evt)
	}
}

func (h *handlers) emitDisconnected(evt DisconnectedEvent) {
	h.mu.RLock()
	fns := make([]DisconnectedHandler, len(h.disconnected))
	copy(fns, h.disconnected)
	h.mu.RUnlock()
	for _, fn := range fns {
		fn(evt)
	}
}

func (h *handlers) emitHeartbeat(evt HeartbeatEvent) {
	h.mu.RLock()
	fns := make([]HeartbeatHandler, len(h.heartbeat))
	copy(fns, h.heartbeat)
	h.mu.RUnlock()
	for _, fn := range fns {
		fn(evt)
	}
}

func (h *handlers) emitError(evt ErrorEvent) {
	h.mu.RLock()
	fns := make([]ErrorHandler, len(h.errorHandlers))
	copy(fns, h.errorHandlers)
	h.mu.RUnlock()
	for _, fn := range fns {
		fn(evt)
	}
}
