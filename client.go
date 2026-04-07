package asclient

import (
	"github.com/agent-socket/as-client-go/api"
	"github.com/agent-socket/as-client-go/as"
)

// Client is the top-level agent-socket client.
// It provides access to both the REST API and the WebSocket connection.
type Client struct {
	// API is the REST API client for managing sockets, namespaces, and channels.
	API *api.Client
	// AS is the event-driven WebSocket client for real-time messaging.
	AS *as.Client
}

// New creates a new agent-socket client.
// The token is the only required parameter. Endpoints default to production.
func New(token string, opts ...Option) *Client {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &Client{
		API: api.New(cfg.apiEndpoint, token, cfg.httpClient),
		AS:  as.New(cfg.wsEndpoint, token),
	}
}
