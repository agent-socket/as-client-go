package types

import "time"

// Socket represents a socket resource.
type Socket struct {
	ID                   string     `json:"id"`
	AccountID            string     `json:"account_id"`
	Status               string     `json:"status"`
	AgentName            string     `json:"agent_name,omitempty"`
	AgentDescription     string     `json:"agent_description,omitempty"`
	Vibe                 string     `json:"vibe,omitempty"`
	ConnectedSince       *time.Time `json:"connected_since,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	HeartbeatEnabled     bool       `json:"heartbeat_enabled"`
	HeartbeatIntervalSec int        `json:"heartbeat_interval_sec,omitempty"`
	HeartbeatData        string     `json:"heartbeat_data,omitempty"`
}

// CreateSocketRequest is the request body for creating a socket.
type CreateSocketRequest struct {
	Name             string `json:"name,omitempty"`
	AgentName        string `json:"agent_name,omitempty"`
	AgentDescription string `json:"agent_description,omitempty"`
	Vibe             string `json:"vibe,omitempty"`
}

// SocketStatus is the response for a socket status query.
type SocketStatus struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	AgentName        string     `json:"agent_name,omitempty"`
	AgentDescription string     `json:"agent_description,omitempty"`
	Vibe             string     `json:"vibe,omitempty"`
	ConnectedSince   *time.Time `json:"connected_since,omitempty"`
}

// UpdateHeartbeatRequest is the request body for updating heartbeat config.
type UpdateHeartbeatRequest struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	IntervalSec *int   `json:"interval_sec,omitempty"`
	Data        string `json:"data,omitempty"`
}

// HeartbeatConfig is the response for heartbeat configuration.
type HeartbeatConfig struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	IntervalSec int    `json:"interval_sec"`
	Data        string `json:"data,omitempty"`
}

// UpdateProfileRequest is the request body for updating a socket profile.
type UpdateProfileRequest struct {
	AgentName        *string `json:"agent_name,omitempty"`
	AgentDescription *string `json:"agent_description,omitempty"`
}

// SocketProfile is the response for a socket profile update.
type SocketProfile struct {
	ID               string `json:"id"`
	AgentName        string `json:"agent_name"`
	AgentDescription string `json:"agent_description"`
}

// UpdateVibeRequest is the request body for updating a socket's vibe.
type UpdateVibeRequest struct {
	Vibe string `json:"vibe"`
}

// VibeResponse is the response for a vibe update.
type VibeResponse struct {
	ID   string `json:"id"`
	Vibe string `json:"vibe"`
}
