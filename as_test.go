package as_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agent-socket/as-go"
	"github.com/agent-socket/as-go/types"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// testServer is a minimal WebSocket endpoint that tests drive with a
// per-connection handler. Each incoming dial fires onConn with the
// attempt number (1-based) and the upgraded conn.
type testServer struct {
	*httptest.Server
	attempts int32
}

func newTestServer(t *testing.T, onConn func(attempt int32, conn *websocket.Conn)) *testServer {
	t.Helper()
	ts := &testServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&ts.attempts, 1)
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		onConn(n, conn)
	}))
	return ts
}

// wsURL converts the server's http://host URL to a ws://host URL so the
// client library recognizes it as a WebSocket endpoint.
func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

// waitFor blocks until want messages have been received via ch or
// until the test deadline hits.
func waitFor(t *testing.T, ch <-chan as.Message, want int, timeout time.Duration) []as.Message {
	t.Helper()
	got := make([]as.Message, 0, want)
	deadline := time.After(timeout)
	for len(got) < want {
		select {
		case m := <-ch:
			got = append(got, m)
		case <-deadline:
			t.Fatalf("timed out waiting for %d messages, got %d", want, len(got))
		}
	}
	return got
}

// Test_Reconnect_AfterDrop verifies the supervisor reconnects after the
// server abruptly closes the WebSocket mid-session, and the handler
// continues to receive messages from the new connection.
func Test_Reconnect_AfterDrop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ts := newTestServer(t, func(attempt int32, conn *websocket.Conn) {
		switch attempt {
		case 1:
			// Send one message, then slam the connection shut with an
			// abnormal close — simulates a network drop.
			_ = wsjson.Write(context.Background(), conn, types.ServerMessage{
				From: "as:test/sender", Data: json.RawMessage(`"first"`),
			})
			_ = conn.Close(websocket.StatusInternalError, "drop")
		case 2:
			// Second dial: send one message, then hold the connection
			// open until the test finishes to confirm sustained health.
			_ = wsjson.Write(context.Background(), conn, types.ServerMessage{
				From: "as:test/sender", Data: json.RawMessage(`"second"`),
			})
			<-ctx.Done()
			_ = conn.Close(websocket.StatusNormalClosure, "")
		}
	})
	defer ts.Close()

	// Buffered large enough that the test server never blocks on our handler.
	events := make(chan as.Message, 16)
	agent := as.Connect("test-token", "as:test/agent",
		func(m as.Message) { events <- m },
		as.WithContext(ctx),
		as.WithEndpoint(wsURL(ts.URL)),
		as.WithMaxBackoff(100*time.Millisecond), // speed up the retry
	)
	defer agent.Close()

	// We expect: first message, a disconnect error, second message.
	// Order is read from the channel; assert shape, not exact ordering
	// of the error vs. first message (the error may arrive just after).
	got := waitFor(t, events, 3, 5*time.Second)

	var messages, errs int
	for _, m := range got {
		if m.Err != nil {
			errs++
		} else {
			messages++
		}
	}
	if messages < 2 {
		t.Errorf("want at least 2 non-error messages after reconnect, got %d", messages)
	}
	if errs < 1 {
		t.Errorf("want at least 1 error event for the drop, got %d", errs)
	}

	// The supervisor must NOT have declared the agent fatal — the
	// connection is currently healthy (held open by attempt==2).
	if err := agent.Err(); err != nil {
		var se *types.ServerError
		if errors.As(err, &se) {
			t.Errorf("unexpected fatal error after reconnect: %v", err)
		}
	}

	// Second attempt should have happened.
	if n := atomic.LoadInt32(&ts.attempts); n < 2 {
		t.Errorf("want at least 2 dial attempts after drop, got %d", n)
	}
}

// Test_Reconnect_StopsOnAuthError verifies the supervisor gives up on
// 401 — no retry loop — and closes Done().
func Test_Reconnect_StopsOnAuthError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Emit a structured server error the client can parse.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(types.ServerError{
			ErrorCode:    "E1001",
			ErrorMessage: "bad token",
		})
	}))
	defer server.Close()

	events := make(chan as.Message, 8)
	agent := as.Connect("bad-token", "as:test/agent",
		func(m as.Message) { events <- m },
		as.WithContext(ctx),
		as.WithEndpoint(wsURL(server.URL)),
		as.WithMaxBackoff(50*time.Millisecond),
	)
	defer agent.Close()

	// The agent should close Done() within a couple of seconds —
	// isFatal returns true for 401 so the supervisor exits immediately
	// after surfacing the error once.
	select {
	case <-agent.Done():
	case <-time.After(3 * time.Second):
		t.Fatalf("agent never gave up on 401")
	}

	if err := agent.Err(); err == nil {
		t.Fatalf("want fatal error on 401, got nil")
	}
}
