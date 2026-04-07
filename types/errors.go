package types

import (
	"errors"
	"fmt"
)

// Sentinel errors for well-known server error codes.
var (
	ErrBadRequest       = errors.New("bad request")
	ErrAuthMissing      = errors.New("missing or invalid Authorization header")
	ErrAuthInvalid      = errors.New("invalid or expired API token")
	ErrAccessDenied     = errors.New("access denied")
	ErrSocketNotFound   = errors.New("invalid or malformed socket address")
	ErrAlreadyConnected = errors.New("socket already connected")
	ErrInternal         = errors.New("internal server error")
)

// errorCodeToSentinel maps server error codes to sentinel errors.
var errorCodeToSentinel = map[string]error{
	"E1000": ErrBadRequest,
	"E1001": ErrBadRequest,
	"E1003": ErrBadRequest,
	"E2000": ErrAuthMissing,
	"E2001": ErrAuthInvalid,
	"E3000": ErrAccessDenied,
	"E4000": ErrSocketNotFound,
	"E4090": ErrAlreadyConnected,
	"E5000": ErrInternal,
}

// ServerError represents a standardized error response from the server.
// All server errors follow the format: {"error_code": "E4090", "error_message": "..."}
type ServerError struct {
	StatusCode   int    `json:"-"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// Error implements the error interface.
func (e *ServerError) Error() string {
	return fmt.Sprintf("%s: %s (status %d)", e.ErrorCode, e.ErrorMessage, e.StatusCode)
}

// Is supports errors.Is matching against sentinel errors.
func (e *ServerError) Is(target error) bool {
	sentinel, ok := errorCodeToSentinel[e.ErrorCode]
	if !ok {
		return false
	}
	return sentinel == target
}

// APIError represents an error response from the REST API.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("api error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true if the error is a 401.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true if the error is a 403.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}
