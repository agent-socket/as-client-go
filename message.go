package as

import (
	"context"
	"encoding/json"
)

// Message is the single event type the handler receives. For incoming
// messages, From and Data are set and Err is nil. For connection or
// protocol errors, Err is non-nil and From/Data are undefined.
type Message struct {
	// From is the sender's full address (e.g. "as:acme/other-agent"
	// or "ch:acme/alerts"). Empty on error events.
	From string

	// Data is the raw JSON payload. Unmarshal into your own type
	// with json.Unmarshal(m.Data, &mytype).
	Data json.RawMessage

	// Err is non-nil for error events: connection drops, auth
	// failures, or server-sent error frames. For a server error
	// frame the underlying error carries the code (e.g. "E7000").
	Err error

	// agent is set for non-error messages so Reply can Send via
	// the same connection. Unset on error events.
	agent *Agent
}

// Reply sends payload back to m.From using the Agent this message
// arrived on. Convenience wrapper for Agent.Send(m.From, payload).
// Returns an error on error events (where m.From is empty).
func (m Message) Reply(payload any) error {
	if m.Err != nil || m.From == "" || m.agent == nil {
		return ErrReplyOnError
	}
	return m.agent.Send(m.From, payload)
}

// ReplyContext is like Reply but takes a context for cancellation.
func (m Message) ReplyContext(ctx context.Context, payload any) error {
	if m.Err != nil || m.From == "" || m.agent == nil {
		return ErrReplyOnError
	}
	return m.agent.SendContext(ctx, m.From, payload)
}
