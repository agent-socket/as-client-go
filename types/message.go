package types

import "encoding/json"

// OutgoingMessage is a message sent over WebSocket to another socket or channel.
type OutgoingMessage struct {
	To   string          `json:"to"`
	Data json.RawMessage `json:"data"`
}

// IncomingMessage is a message received over WebSocket from another socket.
type IncomingMessage struct {
	From string          `json:"from"`
	Data json.RawMessage `json:"data"`
}

// ConnectedMessage is sent by the server after an ephemeral socket connects.
type ConnectedMessage struct {
	Type     string `json:"type"`
	SocketID string `json:"socket_id"`
}

// HeartbeatMessage is sent by the server periodically.
type HeartbeatMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// ServerMessage is used to detect the type of an incoming server frame.
type ServerMessage struct {
	Type string          `json:"type,omitempty"`
	From string          `json:"from,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}
