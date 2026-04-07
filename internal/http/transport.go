package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/agent-socket/as-client-go/types"
)

const (
	authHeaderKey   = "Authorization"
	contentTypeKey  = "Content-Type"
	contentTypeJSON = "application/json"
	bearerPrefix    = "Bearer "
)

// Transport handles authenticated HTTP requests to the API.
type Transport struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewTransport creates a new Transport.
func NewTransport(baseURL, token string, httpClient *http.Client) *Transport {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Transport{
		baseURL:    baseURL,
		token:      token,
		httpClient: httpClient,
	}
}

// Do executes an HTTP request with authentication and returns the raw response.
func (t *Transport) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := t.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set(authHeaderKey, bearerPrefix+t.token)
	if body != nil {
		req.Header.Set(contentTypeKey, contentTypeJSON)
	}

	return t.httpClient.Do(req)
}

// DoJSON executes an HTTP request and decodes the JSON response into dest.
func (t *Transport) DoJSON(ctx context.Context, method, path string, body any, dest any) error {
	resp, err := t.Do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp)
	}

	if dest == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

// DoNoContent executes an HTTP request expecting no response body (e.g., 204).
func (t *Transport) DoNoContent(ctx context.Context, method, path string, body any) error {
	resp, err := t.Do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp)
	}

	return nil
}

func parseErrorResponse(resp *http.Response) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &types.APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read error response: %v", err),
		}
	}

	apiErr := &types.APIError{StatusCode: resp.StatusCode}

	// Try to parse structured error, fall back to raw body.
	var parsed struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(data, &parsed) == nil {
		if parsed.Message != "" {
			apiErr.Message = parsed.Message
		} else if parsed.Error != "" {
			apiErr.Message = parsed.Error
		} else {
			apiErr.Message = string(data)
		}
	} else {
		apiErr.Message = string(data)
	}

	return apiErr
}
