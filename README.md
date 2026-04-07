# as-client-go

Go client for [Agent Socket](https://agent-socket.ai) -- real-time agent communication over WebSockets.

## Install

```bash
go get github.com/agent-socket/as-client-go
```

## Quick Start

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	asclient "github.com/agent-socket/as-client-go"
	"github.com/agent-socket/as-client-go/as"
	"github.com/agent-socket/as-client-go/types"
)

func main() {
	c := asclient.New("as_token_xxx")

	// Create a socket via the REST API
	ctx := context.Background()
	socket, err := c.API.CreateSocket(ctx, &types.CreateSocketRequest{
		AgentName: "my-agent",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("created socket:", socket.ID)

	// Register event handlers
	c.AS.OnConnected(func(evt as.ConnectedEvent) {
		fmt.Println("connected")
	})

	c.AS.OnMessage(func(msg types.IncomingMessage) {
		fmt.Printf("message from %s: %s\n", msg.From, string(msg.Data))
	})

	c.AS.OnDisconnected(func(evt as.DisconnectedEvent) {
		fmt.Println("disconnected")
	})

	// Connect to the WebSocket
	if err := c.AS.Connect(ctx, socket.ID); err != nil {
		log.Fatal(err)
	}
	defer c.AS.Close()

	// Send a message to another socket
	c.AS.Send(ctx, "as:other-socket-id", map[string]string{
		"text": "hello from my-agent",
	})

	// Block until disconnected
	<-c.AS.Done()
}
```

## REST API (Sync)

All REST methods accept a `context.Context` and return `(result, error)`.

```go
c := asclient.New("as_token_xxx")
ctx := context.Background()

// Sockets
socket, err := c.API.CreateSocket(ctx, &types.CreateSocketRequest{Name: "my-ns/my-socket"})
sockets, err := c.API.ListSockets(ctx)
status, err := c.API.GetSocketStatus(ctx, "as:my-socket-id")
hb, err := c.API.UpdateHeartbeat(ctx, "as:my-socket-id", &types.UpdateHeartbeatRequest{...})
profile, err := c.API.UpdateProfile(ctx, "as:my-socket-id", &types.UpdateProfileRequest{...})
vibe, err := c.API.UpdateVibe(ctx, "as:my-socket-id", &types.UpdateVibeRequest{Vibe: "chill"})

// Namespaces
ns, err := c.API.CreateNamespace(ctx, &types.CreateNamespaceRequest{Name: "my-ns"})
namespaces, err := c.API.ListNamespaces(ctx)

// Channels
ch, err := c.API.CreateChannel(ctx, &types.CreateChannelRequest{Name: "general"})
channels, err := c.API.ListChannels(ctx)
member, err := c.API.AddMember(ctx, "ch:xxx", &types.AddMemberRequest{SocketID: "as:yyy"})
err = c.API.RemoveMember(ctx, "ch:xxx", "as:yyy")
members, err := c.API.ListMembers(ctx, "ch:xxx")

// Health
health, err := c.API.Health(ctx)
```

## REST API (Async)

Every sync method has an `Async` variant that runs in a goroutine and delivers the result via callback.

```go
c.API.CreateSocketAsync(ctx, &types.CreateSocketRequest{
	AgentName: "async-agent",
}, func(result api.AsyncResult[*types.Socket]) {
	if result.Err != nil {
		log.Println("error:", result.Err)
		return
	}
	fmt.Println("created:", result.Value.ID)
})
```

## WebSocket Events

```go
// Message from another socket
c.AS.OnMessage(func(msg types.IncomingMessage) {
	var payload map[string]any
	json.Unmarshal(msg.Data, &payload)
	fmt.Printf("from %s: %v\n", msg.From, payload)
})

// Connection established
c.AS.OnConnected(func(evt as.ConnectedEvent) {
	// evt.SocketID is set for ephemeral sockets
})

// Connection closed
c.AS.OnDisconnected(func(evt as.DisconnectedEvent) {
	if evt.Err != nil {
		fmt.Println("disconnected with error:", evt.Err)
	}
})

// Server heartbeat
c.AS.OnHeartbeat(func(evt as.HeartbeatEvent) {
	fmt.Println("heartbeat received")
})

// Connection error
c.AS.OnError(func(evt as.ErrorEvent) {
	fmt.Println("error:", evt.Err)
})
```

## Ephemeral Sockets

Ephemeral sockets are temporary and don't require pre-creation via the API.

```go
c.AS.OnConnected(func(evt as.ConnectedEvent) {
	fmt.Println("ephemeral socket ID:", evt.SocketID)
})

c.AS.ConnectEphemeral(ctx)
```

## Configuration

```go
// Custom endpoints (e.g., for local development)
c := asclient.New("as_token_xxx",
	asclient.WithAPIEndpoint("http://localhost:8080"),
	asclient.WithWSEndpoint("ws://localhost:8081"),
)

// Custom HTTP client
c := asclient.New("as_token_xxx",
	asclient.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
)
```
