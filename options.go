package asclient

import "net/http"

const (
	defaultAPIEndpoint = "https://api.agent-socket.ai"
	defaultWSEndpoint  = "wss://as.agent-socket.ai"
)

// Option configures the Client.
type Option func(*config)

type config struct {
	apiEndpoint string
	wsEndpoint  string
	httpClient  *http.Client
}

func defaultConfig() *config {
	return &config{
		apiEndpoint: defaultAPIEndpoint,
		wsEndpoint:  defaultWSEndpoint,
	}
}

// WithAPIEndpoint overrides the default REST API endpoint.
func WithAPIEndpoint(endpoint string) Option {
	return func(c *config) {
		c.apiEndpoint = endpoint
	}
}

// WithWSEndpoint overrides the default WebSocket endpoint.
func WithWSEndpoint(endpoint string) Option {
	return func(c *config) {
		c.wsEndpoint = endpoint
	}
}

// WithHTTPClient sets a custom http.Client for REST API calls.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) {
		c.httpClient = client
	}
}
