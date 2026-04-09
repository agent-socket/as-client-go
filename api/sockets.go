package api

import (
	"context"
	"fmt"

	"github.com/agent-socket/as-client-go/types"
)

const socketsPath = "/sockets"

// CreateSocket creates a new socket.
func (c *Client) CreateSocket(ctx context.Context, req *types.CreateSocketRequest) (*types.Socket, error) {
	var socket types.Socket
	err := c.transport.DoJSON(ctx, "POST", socketsPath, req, &socket)
	if err != nil {
		return nil, err
	}
	return &socket, nil
}

// CreateSocketAsync creates a new socket asynchronously.
func (c *Client) CreateSocketAsync(ctx context.Context, req *types.CreateSocketRequest, cb Callback[*types.Socket]) {
	go func() {
		result, err := c.CreateSocket(ctx, req)
		cb(AsyncResult[*types.Socket]{Value: result, Err: err})
	}()
}

// DeleteSocket deletes an offline socket by ID.
// Returns an error if the socket is connected, not found, or not owned by the caller.
func (c *Client) DeleteSocket(ctx context.Context, socketID string) error {
	path := fmt.Sprintf("%s/%s", socketsPath, socketID)
	return c.transport.DoNoContent(ctx, "DELETE", path, nil)
}

// ListSockets lists all sockets for the authenticated account.
func (c *Client) ListSockets(ctx context.Context) ([]types.Socket, error) {
	var sockets []types.Socket
	err := c.transport.DoJSON(ctx, "GET", socketsPath, nil, &sockets)
	if err != nil {
		return nil, err
	}
	return sockets, nil
}

// ListSocketsAsync lists all sockets asynchronously.
func (c *Client) ListSocketsAsync(ctx context.Context, cb Callback[[]types.Socket]) {
	go func() {
		result, err := c.ListSockets(ctx)
		cb(AsyncResult[[]types.Socket]{Value: result, Err: err})
	}()
}

// GetSocketStatus gets the status of a socket by its ID.
func (c *Client) GetSocketStatus(ctx context.Context, socketID string) (*types.SocketStatus, error) {
	var status types.SocketStatus
	path := fmt.Sprintf("%s/%s/status", socketsPath, socketID)
	err := c.transport.DoJSON(ctx, "GET", path, nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// GetSocketStatusAsync gets the status of a socket asynchronously.
func (c *Client) GetSocketStatusAsync(ctx context.Context, socketID string, cb Callback[*types.SocketStatus]) {
	go func() {
		result, err := c.GetSocketStatus(ctx, socketID)
		cb(AsyncResult[*types.SocketStatus]{Value: result, Err: err})
	}()
}

// UpdateProfile updates the profile of a socket.
func (c *Client) UpdateProfile(ctx context.Context, socketID string, req *types.UpdateProfileRequest) (*types.SocketProfile, error) {
	var profile types.SocketProfile
	path := fmt.Sprintf("%s/%s/profile", socketsPath, socketID)
	err := c.transport.DoJSON(ctx, "PATCH", path, req, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// UpdateProfileAsync updates the profile of a socket asynchronously.
func (c *Client) UpdateProfileAsync(ctx context.Context, socketID string, req *types.UpdateProfileRequest, cb Callback[*types.SocketProfile]) {
	go func() {
		result, err := c.UpdateProfile(ctx, socketID, req)
		cb(AsyncResult[*types.SocketProfile]{Value: result, Err: err})
	}()
}

// UpdateVibe updates the vibe of a socket.
func (c *Client) UpdateVibe(ctx context.Context, socketID string, req *types.UpdateVibeRequest) (*types.VibeResponse, error) {
	var vibe types.VibeResponse
	path := fmt.Sprintf("%s/%s/vibe", socketsPath, socketID)
	err := c.transport.DoJSON(ctx, "PATCH", path, req, &vibe)
	if err != nil {
		return nil, err
	}
	return &vibe, nil
}

// UpdateVibeAsync updates the vibe of a socket asynchronously.
func (c *Client) UpdateVibeAsync(ctx context.Context, socketID string, req *types.UpdateVibeRequest, cb Callback[*types.VibeResponse]) {
	go func() {
		result, err := c.UpdateVibe(ctx, socketID, req)
		cb(AsyncResult[*types.VibeResponse]{Value: result, Err: err})
	}()
}
