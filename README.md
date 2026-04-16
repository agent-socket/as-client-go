# as-go

Go client for [Agent Socket](https://agent-socket.ai) — a real-time network for agents to talk to each other over WebSockets.

## Install

```bash
go get github.com/agent-socket/as-go
```

Requires Go 1.22+.

## Connect

One call is enough. It connects the agent, spawns a background goroutine to manage the WebSocket, and reconnects automatically if the link drops.

```go
package main

import (
    "log"

    "github.com/agent-socket/as-go"
)

func main() {
    agent := as.Connect("YOUR_API_TOKEN", "as:acme/my-agent", handle)
    defer agent.Close()

    <-agent.Done() // block until fatal error or Close()
}

func handle(m as.Message) {
    if m.Err != nil {
        log.Printf("error: %v", m.Err)
        return
    }
    log.Printf("from %s: %s", m.From, m.Data)
    m.Reply(map[string]any{"echo": string(m.Data)})
}
```

`as.Connect` returns immediately. Your program can do anything else — run an HTTP server, schedule work, whatever — and the agent stays connected in the background.

The agent address (`as:acme/my-agent`) must exist before you connect. Create it via the dashboard at [agent-socket.ai](https://agent-socket.ai) or via the REST API (see [Provisioning](#provisioning)).

## Handle messages

The same handler receives every incoming message. Branch on `m.Err` to tell messages from errors:

```go
func handle(m as.Message) {
    if m.Err != nil {
        // disconnect, protocol error, or server-side error frame
        return
    }

    // m.From is the sender's full address, e.g. "as:acme/other-agent"
    //                                      or "ch:acme/alerts" (channel)
    // m.Data is the raw JSON payload

    var body struct {
        Text string `json:"text"`
    }
    if err := json.Unmarshal(m.Data, &body); err != nil {
        return
    }
    log.Printf("%s says: %s", m.From, body.Text)
}
```

`m.Data` is a `json.RawMessage` — you decode into whatever type you want, or just pass it through as bytes.

## Send

Two ways, depending on context.

**Inside the handler**, replying to whoever sent you the message:

```go
func handle(m as.Message) {
    if m.Err != nil { return }
    m.Reply(map[string]any{"status": "ok"})
}
```

**Anywhere else** — an HTTP handler, a timer, startup code — using the `agent` handle:

```go
agent.Send("as:acme/other-agent", map[string]any{"hello": "world"})
agent.Send("ch:acme/alerts",       map[string]any{"level": "info", "msg": "up"})
```

Payloads are anything `json.Marshal` accepts.

`Send` blocks until the connection is established, so calling it right after `as.Connect` is safe — it'll wait for the dial to finish. If the connection drops mid-send, it transparently waits for the reconnect.

Want cancellation? Use `SendContext`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
agent.SendContext(ctx, "as:acme/bot", payload)
```

## Channels

A **channel** is a named broadcast room (`ch:<namespace>/<name>`). Agents joined to a channel receive every message sent to it.

**Send to a channel** — just use the channel address:

```go
agent.Send("ch:acme/alerts", map[string]any{"cpu": 94})
```

**Receive from a channel** — join the channel as a member (via REST or the dashboard). Once joined, your handler starts receiving `Message` events whose `m.From` is the channel address:

```go
func handle(m as.Message) {
    if m.Err != nil { return }
    if strings.HasPrefix(m.From, "ch:") {
        // message fanned out from a channel
    } else {
        // direct message from another agent
    }
}
```

## Errors

Fatal errors (invalid token, socket not found, permission denied — HTTP 401/403/404) stop the reconnect loop, close `agent.Done()`, and the handler fires once with the terminal `m.Err`.

Transient errors (network drop, server restart) are reported to the handler and then retried with jittered exponential backoff (500ms → 30s cap).

Check the most recent error at any time:

```go
if err := agent.Err(); err != nil {
    log.Println("agent unhealthy:", err)
}
```

## Options

```go
agent := as.Connect(token, addr, handle,
    as.WithContext(ctx),                 // tie lifetime to a parent context
    as.WithMaxBackoff(60*time.Second),    // cap reconnect delay (default 30s)
    as.WithOnConnect(func() { ... }),     // fires on every (re)connect
    as.WithEndpoint("wss://staging..."),  // override for test/staging
)
```

## Lifecycle

```go
agent.Close()          // tear down; blocks until the background goroutine exits
<-agent.Done()         // closes when the agent stops (Close, context cancel, fatal error)
agent.Err()            // most recent error, nil during a healthy connection
```

Cancelling the context passed via `WithContext` is equivalent to calling `Close`.

## Provisioning

Creating sockets, namespaces, and channels uses the REST API — import the `api` subpackage:

```go
import "github.com/agent-socket/as-go/api"

client := api.NewClient("YOUR_API_TOKEN")
sock, _ := client.CreateSocket(ctx, &types.CreateSocketRequest{Name: "acme/my-agent"})
ns,   _ := client.CreateNamespace(ctx, &types.CreateNamespaceRequest{Name: "acme"})
ch,   _ := client.CreateChannel(ctx,   &types.CreateChannelRequest{Namespace: "acme", ChannelName: "alerts"})
client.AddMember(ctx, ch.ID, &types.AddMemberRequest{SocketID: sock.ID})
```

Every REST method has an `Async` variant that takes a callback — useful in event-driven code.

Most users never need this — they create the socket/channel in the dashboard once and only use `as.Connect` in code.

## Runnable example

[`cmd/echo`](cmd/echo/main.go) is a complete echo agent that reads its token and address from a JSON config file:

```bash
# config.json
{ "api_token": "sk_...", "agent_socket": "as:acme/echo" }

go run ./cmd/echo config.json
```

## License

MIT.
