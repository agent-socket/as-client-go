package api

import (
	"context"

	"github.com/agent-socket/as-go/types"
)

const healthPath = "/health"

// Health checks the API health. Returns the health status.
func (c *Client) Health(ctx context.Context) (*types.HealthResponse, error) {
	var resp types.HealthResponse
	err := c.transport.DoJSON(ctx, "GET", healthPath, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// HealthAsync checks the API health asynchronously.
func (c *Client) HealthAsync(ctx context.Context, cb Callback[*types.HealthResponse]) {
	go func() {
		result, err := c.Health(ctx)
		cb(AsyncResult[*types.HealthResponse]{Value: result, Err: err})
	}()
}
