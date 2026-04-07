package as

import (
	"encoding/json"
	"fmt"

	"github.com/agent-socket/as-client-go/types"
	"github.com/gorilla/websocket"
)

// Send sends a message to a target socket or channel.
// The data parameter is marshaled to JSON.
func (c *Client) Send(to string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal message data: %w", err)
	}

	msg := types.OutgoingMessage{
		To:   to,
		Data: jsonData,
	}

	return c.writeConn(func(conn *websocket.Conn) error {
		return conn.WriteJSON(msg)
	})
}

// SendRaw sends a message with pre-encoded JSON data.
func (c *Client) SendRaw(to string, data json.RawMessage) error {
	msg := types.OutgoingMessage{
		To:   to,
		Data: data,
	}

	return c.writeConn(func(conn *websocket.Conn) error {
		return conn.WriteJSON(msg)
	})
}
