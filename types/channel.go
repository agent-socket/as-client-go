package types

import "time"

// Channel represents a channel resource.
type Channel struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Name        string    `json:"name"`
	MemberCount int       `json:"member_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateChannelRequest is the request body for creating a channel.
type CreateChannelRequest struct {
	Name string `json:"name"`
}

// Member represents a channel member.
type Member struct {
	ChannelID string    `json:"channel_id"`
	SocketID  string    `json:"socket_id"`
	AddedAt   time.Time `json:"added_at"`
}

// AddMemberRequest is the request body for adding a member to a channel.
type AddMemberRequest struct {
	SocketID string `json:"socket_id"`
}
