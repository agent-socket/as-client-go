package as

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/agent-socket/as-go/internal/ws"
	"github.com/agent-socket/as-go/types"
)

// Agent is the long-lived handle returned by Connect. It manages a
// single WebSocket connection with automatic reconnect. Safe for
// concurrent use from multiple goroutines.
type Agent struct {
	token    string
	socketID string
	handler  Handler
	cfg      config

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// connReady is closed once on the first successful connect and
	// re-armed each time the connection drops. Send blocks on it so
	// callers can fire-and-forget without racing the initial dial.
	mu         sync.Mutex
	conn       *ws.Client
	connReady  chan struct{}
	lastErr    error
	fatalErr   error // set to stop the reconnect loop (auth failures)
}

// ErrReplyOnError is returned by Message.Reply when called on an error
// event (no sender to reply to).
var ErrReplyOnError = errors.New("as: cannot reply on error event")

// ErrClosed is returned by Send after Close or fatal error.
var ErrClosed = errors.New("as: agent closed")

func newAgent(token, socketID string, handler Handler, cfg config) *Agent {
	ctx, cancel := context.WithCancel(cfg.ctx)
	return &Agent{
		token:     token,
		socketID:  socketID,
		handler:   handler,
		cfg:       cfg,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		connReady: make(chan struct{}),
	}
}

// run is the supervisor loop. It dials, pumps events from the inner
// client until the connection drops, then reconnects with backoff.
// Exits on context cancel or fatal error.
func (a *Agent) run() {
	defer close(a.done)

	backoff := a.cfg.minBackoff
	for {
		if a.ctx.Err() != nil {
			return
		}

		connectedAt, err := a.cycle()
		if a.ctx.Err() != nil {
			return
		}
		a.setLastErr(err)

		// Auth / permission errors: stop retrying.
		if isFatal(err) {
			a.setFatal(err)
			a.handler(Message{Err: err})
			return
		}

		// Surface the error once per cycle.
		if err != nil {
			a.handler(Message{Err: err})
		}

		// Reset backoff if the connection stayed up long enough.
		if !connectedAt.IsZero() && time.Since(connectedAt) > healthyThreshold {
			backoff = a.cfg.minBackoff
		}

		// Sleep with jitter, then loop.
		sleep := backoff + jitter(backoff)
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(sleep):
		}
		backoff = nextBackoff(backoff, a.cfg.maxBackoff)
	}
}

// cycle runs one dial → pump → drop. Returns the time the connection
// became ready (zero if dial failed) and the terminating error.
func (a *Agent) cycle() (time.Time, error) {
	client := ws.New(a.cfg.endpoint, a.token)

	// Buffer one event so the dispatch goroutine doesn't block the
	// read loop even if the user's handler is slow.
	msgs := make(chan Message, 16)

	client.OnMessage(func(im types.IncomingMessage) {
		msgs <- Message{From: im.From, Data: im.Data, agent: a}
	})
	client.OnError(func(evt ws.ErrorEvent) {
		msgs <- Message{Err: wrapServerErr(evt)}
	})

	// Dispatch to the user handler on a dedicated goroutine so a slow
	// handler doesn't stall the WebSocket read loop.
	dispatchDone := make(chan struct{})
	go func() {
		defer close(dispatchDone)
		for m := range msgs {
			a.handler(m)
		}
	}()
	defer func() {
		close(msgs)
		<-dispatchDone
	}()

	if err := client.Connect(a.ctx, a.socketID); err != nil {
		return time.Time{}, err
	}

	// Publish the connection. Send races this: while a.conn is nil it
	// waits on connReady, and the close below wakes it. When the cycle
	// ends we reset a.conn=nil AND swap in a fresh unclosed chan so
	// future Sends block until the *next* cycle publishes.
	a.mu.Lock()
	a.conn = client
	close(a.connReady)
	a.mu.Unlock()

	connectedAt := time.Now()
	if a.cfg.onConnect != nil {
		// Run user callback off the hot path.
		go a.cfg.onConnect()
	}

	// Wait for the connection to close or ctx cancellation.
	select {
	case <-client.Done():
	case <-a.ctx.Done():
		_ = client.Close()
		<-client.Done()
	}

	a.mu.Lock()
	a.conn = nil
	a.connReady = make(chan struct{}) // block future Send until next connect
	a.mu.Unlock()

	return connectedAt, nil
}

// Send sends payload to the target address. The payload is
// marshaled to JSON. Blocks until the agent is connected or the
// agent's context is cancelled. Safe to call before the first
// connection is established.
func (a *Agent) Send(to string, payload any) error {
	return a.SendContext(a.ctx, to, payload)
}

// SendContext is like Send but takes a context for cancellation.
func (a *Agent) SendContext(ctx context.Context, to string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	for {
		a.mu.Lock()
		if a.fatalErr != nil {
			a.mu.Unlock()
			return a.fatalErr
		}
		conn := a.conn
		ready := a.connReady
		a.mu.Unlock()

		if conn != nil {
			if err := conn.SendRaw(ctx, to, data); err == nil {
				return nil
			} else if !errors.Is(err, ws.ErrNotConnected) {
				return err
			}
			// ErrNotConnected: connection dropped mid-send. Fall through
			// to wait for the next cycle and retry once.
		}

		select {
		case <-ready:
			// Connection established (or re-established). Loop and try.
		case <-a.done:
			return ErrClosed
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Close tears down the agent and returns once the background goroutine
// has exited. Safe to call multiple times.
func (a *Agent) Close() {
	a.cancel()
	<-a.done
}

// Done returns a channel that closes when the agent stops (Close
// called, parent context cancelled, or fatal error).
func (a *Agent) Done() <-chan struct{} { return a.done }

// Err returns the most recent error encountered by the agent. Returns
// nil during a healthy connection.
func (a *Agent) Err() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.fatalErr != nil {
		return a.fatalErr
	}
	return a.lastErr
}

func (a *Agent) setLastErr(err error) {
	a.mu.Lock()
	a.lastErr = err
	a.mu.Unlock()
}

func (a *Agent) setFatal(err error) {
	a.mu.Lock()
	a.fatalErr = err
	a.mu.Unlock()
}

// isFatal reports whether err should stop the reconnect loop.
// Auth failures (401/403) and unknown-socket (404) are terminal —
// reconnecting won't fix them and would spam the server.
func isFatal(err error) bool {
	if err == nil {
		return false
	}
	var se *types.ServerError
	if errors.As(err, &se) {
		switch se.StatusCode {
		case 401, 403, 404:
			return true
		}
	}
	return false
}

// wrapServerErr promotes a ServerError to a typed error so the
// supervisor can detect fatality via errors.As.
func wrapServerErr(evt ws.ErrorEvent) error {
	if evt.Err == nil {
		return errors.New("unknown error")
	}
	return evt.Err
}

// nextBackoff doubles backoff up to max.
func nextBackoff(cur, max time.Duration) time.Duration {
	next := cur * 2
	if next > max {
		return max
	}
	return next
}

// jitter returns a random duration in [0, d/2) — small random stagger
// to prevent thundering-herd reconnects from many clients.
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(d / 2)))
}
