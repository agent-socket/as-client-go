package types

import "time"

// Namespace represents a namespace resource.
type Namespace struct {
	Name      string    `json:"name"`
	AccountID string    `json:"account_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateNamespaceRequest is the request body for creating a namespace.
type CreateNamespaceRequest struct {
	Name string `json:"name"`
}
