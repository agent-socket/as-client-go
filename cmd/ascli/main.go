package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	asclient "github.com/agent-socket/as-client-go"
	"github.com/agent-socket/as-client-go/as"
	"github.com/agent-socket/as-client-go/types"
	"gopkg.in/yaml.v3"
)

const defaultConfigName = ".ascli.yaml"
const defaultProfileName = "default"

// configFile is the top-level YAML structure.
type configFile struct {
	Profiles map[string]profile `yaml:"profiles"`
}

// profile is a named set of credentials and endpoint overrides.
type profile struct {
	APIToken   string `yaml:"api_token"`
	WSEndpoint string `yaml:"ws_endpoint,omitempty"`
}

func main() {
	profileName := flag.String("profile", defaultProfileName, "config profile name")
	configPath := flag.String("config", "", "path to config file (default: ~/.ascli.yaml)")
	flag.Parse()

	cfg, err := loadProfile(*configPath, *profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var opts []asclient.Option
	if cfg.WSEndpoint != "" {
		opts = append(opts, asclient.WithWSEndpoint(cfg.WSEndpoint))
	}

	c := asclient.New(cfg.APIToken, opts...)

	c.AS.OnConnected(func(evt as.ConnectedEvent) {
		_ = evt
		fmt.Println("connected")
	})

	c.AS.OnMessage(func(msg types.IncomingMessage) {
		fmt.Printf("< %s: %s\n", msg.From, string(msg.Data))
	})

	c.AS.OnError(func(evt as.ErrorEvent) {
		fmt.Printf("error: %v\n", evt.Err)
	})

	c.AS.OnDisconnected(func(evt as.DisconnectedEvent) {
		if evt.Err != nil {
			fmt.Printf("disconnected: %v\n", evt.Err)
		} else {
			fmt.Println("disconnected")
		}
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println("ascli ready. Commands:")
	fmt.Println("  CONNECT <socket_id>")
	fmt.Println("  SEND <socket_id> <message>")
	fmt.Println("  CLOSE")
	fmt.Println("  QUIT")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print("> ")
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "CONNECT":
			if len(parts) < 2 {
				fmt.Println("error: usage: CONNECT <socket_id>")
				break
			}
			if err := c.AS.Connect(ctx, parts[1]); err != nil {
				fmt.Printf("error: %v\n", err)
			}

		case "SEND":
			if len(parts) < 3 {
				fmt.Println("error: usage: SEND <socket_id> <message>")
				break
			}
			to := parts[1]
			msg := parts[2]
			// Send as raw JSON if it looks like JSON, otherwise wrap as a string.
			var data json.RawMessage
			if isJSON(msg) {
				data = json.RawMessage(msg)
			} else {
				data, _ = json.Marshal(msg)
			}
			if err := c.AS.SendRaw(ctx, to, data); err != nil {
				fmt.Printf("error: %v\n", err)
			}

		case "CLOSE":
			if err := c.AS.Close(); err != nil {
				fmt.Printf("error: %v\n", err)
			}

		case "QUIT", "EXIT":
			c.AS.Close()
			return

		default:
			fmt.Printf("error: unknown command %q\n", cmd)
		}

		select {
		case <-ctx.Done():
			c.AS.Close()
			return
		default:
		}

		fmt.Print("> ")
	}
}

// isJSON returns true if s starts with { or [.
func isJSON(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

// loadProfile reads the config file and returns the named profile.
func loadProfile(configPath, profileName string) (*profile, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home dir: %w", err)
		}
		configPath = filepath.Join(home, defaultConfigName)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var cfg configFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	p, ok := cfg.Profiles[profileName]
	if !ok {
		available := make([]string, 0, len(cfg.Profiles))
		for k := range cfg.Profiles {
			available = append(available, k)
		}
		return nil, fmt.Errorf("profile %q not found in %s (available: %s)", profileName, configPath, strings.Join(available, ", "))
	}

	if p.APIToken == "" {
		return nil, fmt.Errorf("api_token is required in profile %q", profileName)
	}

	return &p, nil
}
