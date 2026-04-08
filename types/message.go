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

// ErrorFrame is a server-to-client error sent when a message cannot be
// processed or delivered. Identified by type="error".
type ErrorFrame struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
