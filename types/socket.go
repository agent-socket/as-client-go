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
