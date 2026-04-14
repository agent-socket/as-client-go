// asconnect connects to a named socket and stays online until interrupted.
// Useful for testing the onboarding "waiting for agent to come online" flow —
// the dashboard detects the connection and advances to the online state.
//
// Usage:
//
//	asconnect as:namespace/agent-name
//	asconnect -profile prod as:namespace/agent-name
//	asconnect -config ./myconfig.yaml as:namespace/agent-name
package main

import (
	"context"
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

const (
	defaultConfigName  = ".ascli.yaml"
	defaultProfileName = "default"
)

type configFile struct {
	Profiles map[string]profile `yaml:"profiles"`
}

type profile struct {
	APIToken   string `yaml:"api_token"`
	WSEndpoint string `yaml:"ws_endpoint,omitempty"`
}

func main() {
	profileName := flag.String("profile", defaultProfileName, "config profile name")
	configPath := flag.String("config", "", "path to config file (default: ~/.ascli.yaml)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: asconnect [flags] <socket_id>\n\n")
		fmt.Fprintf(os.Stderr, "connects to a named socket and stays online until interrupted.\n\n")
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	socketID := flag.Arg(0)

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
		fmt.Printf("connected as %s\n", socketID)
	})
	c.AS.OnMessage(func(msg types.IncomingMessage) {
		fmt.Printf("< %s: %s\n", msg.From, string(msg.Data))
	})
	c.AS.OnError(func(evt as.ErrorEvent) {
		fmt.Fprintf(os.Stderr, "error: %v\n", evt.Err)
	})
	c.AS.OnDisconnected(func(evt as.DisconnectedEvent) {
		if evt.Err != nil {
			fmt.Fprintf(os.Stderr, "disconnected: %v\n", evt.Err)
		} else {
			fmt.Println("disconnected")
		}
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := c.AS.Connect(ctx, socketID); err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}

	select {
	case <-ctx.Done():
		fmt.Println("\ninterrupted")
	case <-c.AS.Done():
	}
	_ = c.AS.Close()
}

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
