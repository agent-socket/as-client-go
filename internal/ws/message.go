package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agent-socket/as-go/types"
)

// Send sends a message to a target socket or channel.
// The data parameter is marshaled to JSON.
func (c *Client) Send(ctx context.Context, to string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal message data: %w", err)
	}

	msg := types.OutgoingMessage{
		To:   to,
		Data: jsonData,
	}

	return c.writeJSON(ctx, msg)
}

// SendRaw sends a message with pre-encoded JSON data.
func (c *Client) SendRaw(ctx context.Context, to string, data json.RawMessage) error {
	msg := types.OutgoingMessage{
		To:   to,
		Data: data,
	}

	return c.writeJSON(ctx, msg)
}
