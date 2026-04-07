package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	asclient "github.com/agent-socket/as-client-go"
	"github.com/agent-socket/as-client-go/as"
	"github.com/agent-socket/as-client-go/types"
)

type config struct {
	APIToken    string `json:"api_token"`
	AgentSocket string `json:"agent_socket"`
}

func main() {
	cfgPath := "config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	c := asclient.New(cfg.APIToken)

	c.AS.OnConnected(func(evt as.ConnectedEvent) {
		log.Printf("connected to %s", cfg.AgentSocket)
	})

	c.AS.OnMessage(func(msg types.IncomingMessage) {
		var payload any
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			return
		}

		log.Printf("message from %s: %s", msg.From, string(msg.Data))

		reply := map[string]any{
			"echo": payload,
			"time": time.Now().UTC().Format(time.RFC3339),
		}

		if err := c.AS.Send(context.Background(), msg.From, reply); err != nil {
			log.Printf("failed to send reply: %v", err)
		}
	})

	c.AS.OnError(func(evt as.ErrorEvent) {
		log.Printf("error: %v", evt.Err)
	})

	c.AS.OnDisconnected(func(evt as.DisconnectedEvent) {
		if evt.Err != nil {
			log.Printf("disconnected with error: %v", evt.Err)
		} else {
			log.Println("disconnected")
		}
	})

	ctx := context.Background()
	if err := c.AS.Connect(ctx, cfg.AgentSocket); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer c.AS.Close()

	log.Println("listening for messages (ctrl+c to quit)")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-sig:
		log.Println("shutting down")
	case <-c.AS.Done():
	}
}

func loadConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.APIToken == "" {
		return nil, fmt.Errorf("api_token is required in %s", path)
	}
	if cfg.AgentSocket == "" {
		return nil, fmt.Errorf("agent_socket is required in %s", path)
	}

	return &cfg, nil
}
