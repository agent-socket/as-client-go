package api

import (
	"context"

	"github.com/agent-socket/as-client-go/types"
)

const namespacesPath = "/namespaces"

// CreateNamespace creates a new namespace.
func (c *Client) CreateNamespace(ctx context.Context, req *types.CreateNamespaceRequest) (*types.Namespace, error) {
	var ns types.Namespace
	err := c.transport.DoJSON(ctx, "POST", namespacesPath, req, &ns)
	if err != nil {
		return nil, err
	}
	return &ns, nil
}

// CreateNamespaceAsync creates a new namespace asynchronously.
func (c *Client) CreateNamespaceAsync(ctx context.Context, req *types.CreateNamespaceRequest, cb Callback[*types.Namespace]) {
	go func() {
		result, err := c.CreateNamespace(ctx, req)
		cb(AsyncResult[*types.Namespace]{Value: result, Err: err})
	}()
}

// ListNamespaces lists all namespaces for the authenticated account.
func (c *Client) ListNamespaces(ctx context.Context) ([]types.Namespace, error) {
	var namespaces []types.Namespace
	err := c.transport.DoJSON(ctx, "GET", namespacesPath, nil, &namespaces)
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

// ListNamespacesAsync lists all namespaces asynchronously.
func (c *Client) ListNamespacesAsync(ctx context.Context, cb Callback[[]types.Namespace]) {
	go func() {
		result, err := c.ListNamespaces(ctx)
		cb(AsyncResult[[]types.Namespace]{Value: result, Err: err})
	}()
}
