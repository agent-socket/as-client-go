package api

import (
	"context"
	"fmt"

	"github.com/agent-socket/as-go/types"
)

const channelsPath = "/channels"

// CreateChannel creates a new channel.
func (c *Client) CreateChannel(ctx context.Context, req *types.CreateChannelRequest) (*types.Channel, error) {
	var channel types.Channel
	err := c.transport.DoJSON(ctx, "POST", channelsPath, req, &channel)
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// CreateChannelAsync creates a new channel asynchronously.
func (c *Client) CreateChannelAsync(ctx context.Context, req *types.CreateChannelRequest, cb Callback[*types.Channel]) {
	go func() {
		result, err := c.CreateChannel(ctx, req)
		cb(AsyncResult[*types.Channel]{Value: result, Err: err})
	}()
}

// ListChannels lists all channels for the authenticated account.
func (c *Client) ListChannels(ctx context.Context) ([]types.Channel, error) {
	var channels []types.Channel
	err := c.transport.DoJSON(ctx, "GET", channelsPath, nil, &channels)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// ListChannelsAsync lists all channels asynchronously.
func (c *Client) ListChannelsAsync(ctx context.Context, cb Callback[[]types.Channel]) {
	go func() {
		result, err := c.ListChannels(ctx)
		cb(AsyncResult[[]types.Channel]{Value: result, Err: err})
	}()
}

// AddMember adds a socket to a channel.
func (c *Client) AddMember(ctx context.Context, channelID string, req *types.AddMemberRequest) (*types.Member, error) {
	var member types.Member
	path := fmt.Sprintf("%s/%s/members", channelsPath, channelID)
	err := c.transport.DoJSON(ctx, "POST", path, req, &member)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// AddMemberAsync adds a socket to a channel asynchronously.
func (c *Client) AddMemberAsync(ctx context.Context, channelID string, req *types.AddMemberRequest, cb Callback[*types.Member]) {
	go func() {
		result, err := c.AddMember(ctx, channelID, req)
		cb(AsyncResult[*types.Member]{Value: result, Err: err})
	}()
}

// RemoveMember removes a socket from a channel.
func (c *Client) RemoveMember(ctx context.Context, channelID string, socketID string) error {
	path := fmt.Sprintf("%s/%s/members/%s", channelsPath, channelID, socketID)
	return c.transport.DoNoContent(ctx, "DELETE", path, nil)
}

// RemoveMemberAsync removes a socket from a channel asynchronously.
func (c *Client) RemoveMemberAsync(ctx context.Context, channelID string, socketID string, cb Callback[struct{}]) {
	go func() {
		err := c.RemoveMember(ctx, channelID, socketID)
		cb(AsyncResult[struct{}]{Err: err})
	}()
}

// ListMembers lists all members of a channel.
func (c *Client) ListMembers(ctx context.Context, channelID string) ([]types.Member, error) {
	var members []types.Member
	path := fmt.Sprintf("%s/%s/members", channelsPath, channelID)
	err := c.transport.DoJSON(ctx, "GET", path, nil, &members)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// ListMembersAsync lists all members of a channel asynchronously.
func (c *Client) ListMembersAsync(ctx context.Context, channelID string, cb Callback[[]types.Member]) {
	go func() {
		result, err := c.ListMembers(ctx, channelID)
		cb(AsyncResult[[]types.Member]{Value: result, Err: err})
	}()
}
