package api

import (
	"net/http"

	transport "github.com/agent-socket/as-go/internal/http"
)

// Client is the REST API client for agent-socket.
type Client struct {
	transport *transport.Transport
}

// New creates a new API client.
func New(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		transport: transport.NewTransport(baseURL, token, httpClient),
	}
}

// AsyncResult holds the result of an asynchronous API call.
type AsyncResult[T any] struct {
	Value T
	Err   error
}

// Callback is a function invoked with the result of an async API call.
type Callback[T any] func(AsyncResult[T])
