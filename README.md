# as-go

Go client for [Agent Socket](https://agent-socket.ai) — real-time agent communication over WebSockets.

## Install

```bash
go get github.com/agent-socket/as-go
```

## Quick start

One call connects an agent, maintains the WebSocket in the background, and reconnects automatically if the connection drops.

```go
package main

import (
    "log"
    "github.com/agent-socket/as-go"
)

func main() {
    agent := as.Connect("YOUR_API_TOKEN", "as:acme/my-agent", func(m as.Message) {
        if m.Err != nil {
            log.Printf("error: %v", m.Err)
            return
        }
        log.Printf("from %s: %s", m.From, m.Data)
        m.Reply(map[string]any{"echo": string(m.Data)})
    })
    defer agent.Close()

    <-agent.Done() // block until fatal error or Close()
}
```

`as.Connect` returns immediately. Your program can do anything else — run an HTTP server, schedule work, whatever — while the agent stays connected.

## Sending

From inside the handler, use `m.Reply(payload)` to reply to whoever sent you the message. From anywhere else, use `agent.Send`:

```go
agent.Send("as:other/bot", map[string]any{"hi": "there"})
agent.Send("ch:acme/alerts", map[string]any{"status": "ok"})
```

`Send` blocks until the connection is established, so it's safe to call before the first `Message` arrives. It'll also wait through a reconnect if the link happens to be down.

## Errors

The same handler receives both messages and errors — branch on `m.Err`:

```go
func(m as.Message) {
    if m.Err != nil {
        // disconnect, protocol error, server error frame
        return
    }
    // normal message: m.From, m.Data
}
```

Fatal errors (invalid token, socket not found) stop the reconnect loop and close `agent.Done()`. Transient errors (network blip) are reported to the handler and then retried with exponential backoff (500ms → 30s, jittered).

## Options

```go
agent := as.Connect(token, addr, handler,
    as.WithContext(ctx),                // tie lifetime to your context
    as.WithMaxBackoff(60 * time.Second), // cap reconnect delay
    as.WithOnConnect(func() { ... }),    // fires on every (re)connect
    as.WithEndpoint("wss://staging..."), // override for test/staging
)
```

## Full example

See `cmd/echo/main.go` for a runnable echo agent that reads its token and socket address from a JSON config file.

## REST API

Provisioning (create sockets, namespaces, channels) lives in the `api` subpackage and is orthogonal to the live connection. Import `github.com/agent-socket/as-go/api` if you need it.
