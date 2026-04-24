package json

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_CategorizeError(t *testing.T) {
	tests := []struct {
		code          ErrorCode
		wantCategory  ErrorCategory
		wantRetryable bool
	}{
		// Request validation errors - not retryable
		{ErrorCodeInvalidFormat, ErrorCategoryRequest, false},
		{ErrorCodeInvalidValue, ErrorCategoryRequest, false},
		{ErrorCodeMissingRequiredField, ErrorCategoryRequest, false},
		{ErrorCodeValidationError, ErrorCategoryRequest, false},

		// Auth errors - not retryable
		{ErrorCodeForbidden, ErrorCategoryAuth, false},
		{ErrorCodeUnauthorized, ErrorCategoryAuth, false},

		// Rate limiting - retryable after backoff
		{ErrorCodeRateLimitExceeded, ErrorCategoryRateLimit, true},

		// Trading errors - not retryable (business logic rejections)
		{ErrorCodeFOKNotFilled, ErrorCategoryTrading, false},
		{ErrorCodeIOCNotFilled, ErrorCategoryTrading, false},
		{ErrorCodeIdempotencyConflict, ErrorCategoryTrading, false},
		{ErrorCodeInsufficientMargin, ErrorCategoryTrading, false},
		{ErrorCodeInvalidOrderSide, ErrorCategoryTrading, false},
		{ErrorCodeInvalidTriggerPrice, ErrorCategoryTrading, false},
		{ErrorCodeMarketClosed, ErrorCategoryTrading, false},
		{ErrorCodeMarketNotFound, ErrorCategoryTrading, false},
		{ErrorCodeMaxOrdersPerMarket, ErrorCategoryTrading, false},
		{ErrorCodeMaxSubAccountsExceeded, ErrorCategoryTrading, false},
		{ErrorCodeMaxTotalOrders, ErrorCategoryTrading, false},
		{ErrorCodeNoLiquidity, ErrorCategoryTrading, false},
		{ErrorCodeOICapExceeded, ErrorCategoryTrading, false},
		{ErrorCodeOperationCancelFailed, ErrorCategoryTrading, false},
		{ErrorCodeOrderNotFound, ErrorCategoryTrading, false},
		{ErrorCodeOrderRejectedByEngine, ErrorCategoryTrading, false},
		{ErrorCodePositionNotFound, ErrorCategoryTrading, false},
		{ErrorCodePostOnlyWouldTrade, ErrorCategoryTrading, false},
		{ErrorCodePriceOutOfBounds, ErrorCategoryTrading, false},
		{ErrorCodeQuantityBelowFilled, ErrorCategoryTrading, false},
		{ErrorCodeQuantityTooSmall, ErrorCategoryTrading, false},
		{ErrorCodeReduceOnlyNoPosition, ErrorCategoryTrading, false},
		{ErrorCodeReduceOnlySameSide, ErrorCategoryTrading, false},
		{ErrorCodeReduceOnlyWouldIncrease, ErrorCategoryTrading, false},
		{ErrorCodeSelfTradePrevented, ErrorCategoryTrading, false},

		// Wick insurance - retryable (temporary protection)
		{ErrorCodeWickInsuranceActive, ErrorCategoryTrading, true},

		// Timeout - retryable
		{ErrorCodeOperationTimeout, ErrorCategorySystem, true},

		// System errors - retryable
		{ErrorCodeCacheError, ErrorCategorySystem, true},
		{ErrorCodeDatabaseError, ErrorCategorySystem, true},
		{ErrorCodeInternalError, ErrorCategorySystem, true},
		{ErrorCodeRequestTimeout, ErrorCategorySystem, true},
		{ErrorCodeServiceUnavailable, ErrorCategorySystem, true},

		// System errors - not retryable
		{ErrorCodeAssetNotFound, ErrorCategorySystem, false},
		{ErrorCodeInvalidMarketConfig, ErrorCategorySystem, false},
		{ErrorCodeMarketAlreadyExists, ErrorCategorySystem, false},
		{ErrorCodeMethodNotAllowed, ErrorCategorySystem, false},
		{ErrorCodeNotFound, ErrorCategorySystem, false},

		// Unknown code defaults to SYSTEM, not retryable
		{ErrorCode(""), ErrorCategorySystem, false},
		{ErrorCode("UNKNOWN_CODE"), ErrorCategorySystem, false},
	}

	for _, tt := range tests {
		name := string(tt.code)
		if name == "" {
			name = "empty_code"
		}
		t.Run(name, func(t *testing.T) {
			category, retryable := CategorizeError(tt.code)
			assert.Equal(t, tt.wantCategory, category, "category mismatch for code %s", tt.code)
			assert.Equal(t, tt.wantRetryable, retryable, "retryable mismatch for code %s", tt.code)
		})
	}
}

func Test_NewSuccessResponse(t *testing.T) {
	type testData struct {
		Value string `json:"value"`
	}

	requestId := ClientRequestId("req-123")
	data := testData{Value: "test"}

	resp := NewSuccessResponse(requestId, data)

	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, data, resp.Response)
	assert.Equal(t, requestId, resp.ClientRequestId)
	assert.Nil(t, resp.Error)
	assert.Greater(t, resp.Timestamp, int64(0))
}

func Test_NewSuccessResponse_EmptyClientRequestId(t *testing.T) {
	resp := NewSuccessResponse(ClientRequestId(""), "data")

	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, ClientRequestId(""), resp.ClientRequestId)
}

func Test_NewErrorResponse(t *testing.T) {
	requestId := ClientRequestId("req-456")
	code := ErrorCodeValidationError
	message := "invalid field"
	details := map[string]string{"field": "quantity"}

	resp := NewErrorResponse[any](requestId, code, message, details)

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, requestId, resp.ClientRequestId)
	require.NotNil(t, resp.Error)
	assert.Equal(t, code, resp.Error.Code)
	assert.Equal(t, ErrorCategoryRequest, resp.Error.Category)
	assert.Equal(t, message, resp.Error.Message)
	assert.False(t, resp.Error.Retryable)
	assert.Equal(t, details, resp.Error.Details)
	assert.Greater(t, resp.Timestamp, int64(0))
}

func Test_NewErrorResponse_NilDetails(t *testing.T) {
	resp := NewErrorResponse[any](ClientRequestId("req"), ErrorCodeInternalError, "error", nil)

	require.NotNil(t, resp.Error)
	assert.Nil(t, resp.Error.Details)
}

func Test_NewErrorResponse_RetryableError(t *testing.T) {
	resp := NewErrorResponse[any](ClientRequestId("req"), ErrorCodeRateLimitExceeded, "too many requests", nil)

	require.NotNil(t, resp.Error)
	assert.True(t, resp.Error.Retryable)
	assert.Equal(t, ErrorCategoryRateLimit, resp.Error.Category)
}

func Test_NewValidationErrorResponse(t *testing.T) {
	requestId := ClientRequestId("req-789")
	message := "missing required field"
	details := map[string]string{"field": "symbol"}

	resp := NewValidationErrorResponse[any](requestId, message, details)

	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeValidationError, resp.Error.Code)
	assert.Equal(t, ErrorCategoryRequest, resp.Error.Category)
	assert.Equal(t, message, resp.Error.Message)
	assert.False(t, resp.Error.Retryable)
	assert.Equal(t, details, resp.Error.Details)
}

func Test_NewBusinessErrorResponse(t *testing.T) {
	requestId := ClientRequestId("req-business")
	code := ErrorCodeOrderNotFound
	message := "order 123 not found"

	resp := NewBusinessErrorResponse[any](requestId, code, message)

	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, code, resp.Error.Code)
	assert.Equal(t, ErrorCategoryTrading, resp.Error.Category)
	assert.Equal(t, message, resp.Error.Message)
	assert.False(t, resp.Error.Retryable)
	assert.Nil(t, resp.Error.Details)
}

func Test_NewSystemErrorResponse(t *testing.T) {
	requestId := ClientRequestId("req-system")
	message := "internal server error"

	// Without DEBUG env var, actualError should not be exposed
	resp := NewSystemErrorResponse[any](requestId, message, assert.AnError)

	assert.Equal(t, "error", resp.Status)
	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeInternalError, resp.Error.Code)
	assert.Equal(t, ErrorCategorySystem, resp.Error.Category)
	assert.Equal(t, message, resp.Error.Message)
	assert.True(t, resp.Error.Retryable)
	assert.Nil(t, resp.Error.Details)
}

func Test_NewSystemErrorResponse_WithDebug(t *testing.T) {
	t.Setenv("DEBUG", "true")

	requestId := ClientRequestId("req-debug")
	message := "internal error"
	actualErr := assert.AnError

	resp := NewSystemErrorResponse[any](requestId, message, actualErr)

	require.NotNil(t, resp.Error)
	assert.NotNil(t, resp.Error.Details)
	assert.Equal(t, actualErr.Error(), resp.Error.Details["error"])
}

func Test_MapFromErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want map[string]string
	}{
		{
			name: "nil error",
			err:  nil,
			want: nil,
		},
		{
			name: "with error",
			err:  assert.AnError,
			want: map[string]string{"error": assert.AnError.Error()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapFromErr(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_APIResponse_TypedResponse(t *testing.T) {
	type OrderData struct {
		OrderId int64  `json:"orderId"`
		Status  string `json:"status"`
	}

	data := OrderData{OrderId: 12345, Status: "filled"}
	resp := NewSuccessResponse(ClientRequestId("req"), data)

	assert.Equal(t, int64(12345), resp.Response.OrderId)
	assert.Equal(t, "filled", resp.Response.Status)
}

func Test_APIResponse_MarshalJSON_OutputsBothRequestIdFields(t *testing.T) {
	requestId := ClientRequestId("test-request-123")
	resp := NewSuccessResponse(requestId, "data")

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var result map[string]any
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// Should have both legacy "request_id" and new "requestId" fields
	assert.Equal(t, string(requestId), result["request_id"], "legacy request_id field should be present")
	assert.Equal(t, string(requestId), result["requestId"], "new requestId field should be present")
}

func Test_APIResponse_MarshalJSON_EmptyClientRequestId(t *testing.T) {
	resp := NewSuccessResponse(ClientRequestId(""), "data")

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var result map[string]any
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	// Empty request IDs should be omitted (omitempty)
	_, hasLegacy := result["request_id"]
	_, hasNew := result["requestId"]
	assert.False(t, hasLegacy, "empty request_id should be omitted")
	assert.False(t, hasNew, "empty requestId should be omitted")
}

func Test_isDeadlineError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "context.DeadlineExceeded",
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "wrapped context.DeadlineExceeded",
			err:  fmt.Errorf("rpc call failed: %w", context.DeadlineExceeded),
			want: true,
		},
		{
			name: "context.Canceled must not match",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "wrapped context.Canceled must not match",
			err:  fmt.Errorf("aborted: %w", context.Canceled),
			want: false,
		},
		{
			name: "gRPC DeadlineExceeded status",
			err:  status.Error(codes.DeadlineExceeded, "context deadline exceeded"),
			want: true,
		},
		{
			name: "gRPC Internal status must not match",
			err:  status.Error(codes.Internal, "something broke"),
			want: false,
		},
		{
			name: "generic error must not match",
			err:  assert.AnError,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isDeadlineError(tt.err))
		})
	}
}

func Test_New503RequestTimeoutResponse(t *testing.T) {
	t.Parallel()

	resp := New503RequestTimeoutResponse[any]("req-503-timeout")

	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeRequestTimeout, resp.Error.Code)
	assert.Equal(t, "Request timed out", resp.Error.Message)
	assert.True(t, resp.Error.Retryable)
}

func Test_NewSystemErrorResponse_DeadlineExceeded_ReturnsRequestTimeout(t *testing.T) {
	t.Parallel()

	resp := NewSystemErrorResponse[any]("req-timeout", "handler failed", context.DeadlineExceeded)

	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeRequestTimeout, resp.Error.Code)
	assert.Equal(t, "Request timed out", resp.Error.Message)
	assert.True(t, resp.Error.Retryable)
}

func Test_NewSystemErrorResponse_Canceled_Returns500(t *testing.T) {
	t.Parallel()

	resp := NewSystemErrorResponse[any]("req-cancel", "handler failed", context.Canceled)

	require.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeInternalError, resp.Error.Code, "context.Canceled must route to 500, not REQUEST_TIMEOUT")
}

func Test_HasTimeoutError(t *testing.T) {
	t.Parallel()

	t.Run("timeout response", func(t *testing.T) {
		resp := NewSystemErrorResponse[any]("req", "fail", context.DeadlineExceeded)
		assert.True(t, resp.HasTimeoutError())
	})

	t.Run("internal error response", func(t *testing.T) {
		resp := NewSystemErrorResponse[any]("req", "fail", assert.AnError)
		assert.False(t, resp.HasTimeoutError())
	})

	t.Run("success response", func(t *testing.T) {
		resp := NewSuccessResponse("req", "data")
		assert.False(t, resp.HasTimeoutError())
	})

	t.Run("nil receiver", func(t *testing.T) {
		var resp *APIResponse[any]
		assert.False(t, resp.HasTimeoutError())
	})
}
