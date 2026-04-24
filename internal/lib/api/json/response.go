// Package json contains types and validation for the API layer.
// These types are used for JSON marshaling/unmarshaling in both REST and WebSocket APIs.
package json

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Response represents a generic API response structure
type APIResponse[T any] struct {
	Status          string          `json:"status" required:"true"` // "ok" or "error"
	Response        T               `json:"response,omitempty"`
	Error           *ErrorData      `json:"error,omitempty"`
	Message         string          `json:"message,omitempty"`
	Timestamp       int64           `json:"timestamp,omitempty"` // Server timestamp in milliseconds
	ClientRequestId ClientRequestId `json:"requestId,omitempty"`
}

// Custom marshaling to output both "request_id" (legacy) and "requestId" (new) fields for backwards compatibility.
func (r APIResponse[T]) MarshalJSON() ([]byte, error) {
	type Alias APIResponse[T]
	aux := struct {
		Alias
		LegacyClientRequestId ClientRequestId `json:"request_id,omitempty"`
	}{
		Alias:                 Alias(r),
		LegacyClientRequestId: r.ClientRequestId,
	}
	return json.Marshal(aux)
}

// ErrorData represents a generic error data structure
type ErrorData struct {
	Code      ErrorCode         `json:"code,omitempty"`
	Category  ErrorCategory     `json:"category,omitempty"`
	Message   string            `json:"message,omitempty"`
	Retryable bool              `json:"retryable"`
	Details   map[string]string `json:"details,omitempty"`
}

// Helper function to create an instance of `map[string]string` from an err.
func MapFromErr(err error) map[string]string {
	if err == nil {
		return nil
	} else {
		return map[string]string{"error": err.Error()}
	}
}

// Creates a success response with the given type and data.
func NewSuccessResponse[T any](clientRequestId ClientRequestId, data T) *APIResponse[T] {
	now := snx_lib_utils_time.Now()
	return &APIResponse[T]{
		Status:          "ok",
		Response:        data,
		Timestamp:       now.UnixMilli(),
		ClientRequestId: clientRequestId,
	}
}

// Creates a new error response.
func NewErrorResponse[T any](clientRequestId ClientRequestId, code ErrorCode, message string, details map[string]string) *APIResponse[T] {
	now := snx_lib_utils_time.Now()
	category, retryable := CategorizeError(code)

	return &APIResponse[T]{
		Status: "error",
		Error: &ErrorData{
			Code:      code,
			Category:  category,
			Message:   message,
			Retryable: retryable,
			Details:   details,
		},
		Timestamp:       now.UnixMilli(),
		ClientRequestId: clientRequestId,
	}
}

// Creates a validation error response, always specifying
// ErrorCodeValidationError for the code.
func NewValidationErrorResponse[T any](clientRequestId ClientRequestId, message string, details map[string]string) *APIResponse[T] {
	return NewErrorResponse[T](clientRequestId, ErrorCodeValidationError, message, details)
}

// Creates a business logic error response.
func NewBusinessErrorResponse[T any](clientRequestId ClientRequestId, code ErrorCode, message string) *APIResponse[T] {
	return NewErrorResponse[T](clientRequestId, code, message, nil)
}

// New503RequestTimeoutResponse is the JSON body for HTTP 503 when the request deadline was
// exceeded. Pair with [net/http.StatusServiceUnavailable] only for timeouts (same rule as
// WebSocket buildTimeoutJSON: 503 + REQUEST_TIMEOUT).
func New503RequestTimeoutResponse[T any](clientRequestId ClientRequestId) *APIResponse[T] {
	return NewErrorResponse[T](clientRequestId, ErrorCodeRequestTimeout, "Request timed out", nil)
}

// Creates a system error response. When the underlying error is a
// context deadline exceeded (from any internal service timeout), the
// response carries ErrorCodeRequestTimeout so that upstream dispatch
// layers can surface a 503 instead of 500.
func NewSystemErrorResponse[T any](clientRequestId ClientRequestId, message string, actualError error) *APIResponse[T] {
	if isDeadlineError(actualError) {
		return New503RequestTimeoutResponse[T](clientRequestId)
	}

	if os.Getenv("DEBUG") == "true" {
		return NewErrorResponse[T](clientRequestId, ErrorCodeInternalError, message, map[string]string{"error": actualError.Error()})
	}

	return NewErrorResponse[T](clientRequestId, ErrorCodeInternalError, message, nil)
}

// isDeadlineError reports whether err originates from a context deadline.
// It checks the standard sentinel (direct context usage) and the gRPC
// status code (gRPC wraps deadline errors with codes.DeadlineExceeded).
func isDeadlineError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	return status.Code(err) == codes.DeadlineExceeded
}

// Reports whether this response carries a timeout error code set by
// NewSystemErrorResponse when it detected a deadline exceeded error.
func (r *APIResponse[T]) HasTimeoutError() bool {
	return r != nil && r.Error != nil && r.Error.Code == ErrorCodeRequestTimeout
}

// various response type labels used throughout

// Common response types that can be used across the API
const (
	ResponseTypeAccount      = "account"
	ResponseTypeError        = "error"
	ResponseTypeMarketData   = "marketData"
	ResponseTypeOrder        = "order"
	ResponseTypeSubAccount   = "subaccount"
	ResponseTypeSubscription = "subscription"
)
