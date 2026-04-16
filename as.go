// Package as is the one-call client for Agent Socket.
//
// Connect a long-lived agent and handle messages in three lines:
//
//	agent := as.Connect("YOUR_API_TOKEN", "as:acme/my-agent", func(m as.Message) {
//	    if m.Err != nil { return }
//	    m.Reply(map[string]any{"echo": string(m.Data)})
//	})
//	defer agent.Close()
//
// Connect returns immediately. A background goroutine opens the WebSocket,
// dispatches messages to the handler, and reconnects with exponential
// backoff if the connection drops. Fatal errors (invalid token, socket
// not found) stop retrying — check [Agent.Err] or watch [Agent.Done].
//
// To send from anywhere in your program, use [Agent.Send]:
//
//	agent.Send("as:other/bot", map[string]any{"hi": "there"})
package as

import (
	"context"
	"net/http"
	"time"
)

const (
	defaultWSEndpoint = "wss://as.agent-socket.ai"

	defaultMinBackoff = 500 * time.Millisecond
	defaultMaxBackoff = 30 * time.Second
	// Sustained connected time before the backoff counter resets.
	// Prevents a flapping peer from spinning at minBackoff forever.
	healthyThreshold = 30 * time.Second
)

// Handler is called once per incoming message and once per error event.
// Callers should branch on m.Err:
//
//	func(m as.Message) {
//	    if m.Err != nil { /* disconnect, auth fail, etc. */ return }
//	    /* normal message: m.From, m.Data */
//	}
type Handler func(Message)

// Connect opens an Agent Socket connection for socketID and keeps it
// alive in a background goroutine. The returned *Agent is usable
// immediately — Send may be called before the first connection is
// established (it will block until connected or the context is done).
//
// Connect never blocks. Errors from the initial dial (and any
// subsequent reconnect) are delivered to handler as Message{Err: …}
// events. Auth errors are fatal and stop the reconnect loop.
func Connect(token, socketID string, handler Handler, opts ...Option) *Agent {
	cfg := config{
		endpoint:   defaultWSEndpoint,
		minBackoff: defaultMinBackoff,
		maxBackoff: defaultMaxBackoff,
		ctx:        context.Background(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if handler == nil {
		handler = func(Message) {}
	}
	a := newAgent(token, socketID, handler, cfg)
	go a.run()
	return a
}

// Option configures Connect. See [WithContext], [WithEndpoint], etc.
type Option func(*config)

type config struct {
	endpoint   string
	ctx        context.Context
	minBackoff time.Duration
	maxBackoff time.Duration
	httpClient *http.Client
	onConnect  func()
}

// WithContext sets a parent context. Cancelling it tears down the agent
// (equivalent to calling Close). Defaults to context.Background().
func WithContext(ctx context.Context) Option {
	return func(c *config) { c.ctx = ctx }
}

// WithEndpoint overrides the WebSocket endpoint. Defaults to
// wss://as.agent-socket.ai. Useful for staging or local testing.
func WithEndpoint(url string) Option {
	return func(c *config) { c.endpoint = url }
}

// WithMaxBackoff sets the upper bound on the reconnect delay. Defaults
// to 30s. Reconnect starts at 500ms and doubles up to this cap.
func WithMaxBackoff(d time.Duration) Option {
	return func(c *config) { c.maxBackoff = d }
}

// WithOnConnect registers a callback fired on every (re)connect. Useful
// for re-subscribing to state or resending queued work after a drop.
func WithOnConnect(fn func()) Option {
	return func(c *config) { c.onConnect = fn }
}
