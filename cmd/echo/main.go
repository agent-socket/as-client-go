// echo is the canonical Agent Socket demo: connects as an agent and
// echoes every message it receives back to the sender.
//
//	go run ./cmd/echo config.json
//
// config.json is a two-field JSON file:
//
//	{ "api_token": "sk_...", "agent_socket": "as:acme/echo" }
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/agent-socket/as-go"
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
		log.Fatalf("load config: %v", err)
	}

	agent := as.Connect(cfg.APIToken, cfg.AgentSocket, func(m as.Message) {
		if m.Err != nil {
			log.Printf("error: %v", m.Err)
			return
		}
		log.Printf("from %s: %s", m.From, m.Data)
		if err := m.Reply(map[string]any{"echo": json.RawMessage(m.Data)}); err != nil {
			log.Printf("reply: %v", err)
		}
	})
	defer agent.Close()

	log.Printf("listening as %s (ctrl+c to quit)", cfg.AgentSocket)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
	case <-agent.Done():
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
	if cfg.APIToken == "" || cfg.AgentSocket == "" {
		return nil, fmt.Errorf("%s must set api_token and agent_socket", path)
	}
	return &cfg, nil
}
