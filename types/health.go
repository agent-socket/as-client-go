package types

// HealthResponse is the response from the health endpoint.
type HealthResponse struct {
	Status string `json:"status"`
}
