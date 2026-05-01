package core

import (
	"errors"
	"fmt"
	"testing"

	snx_lib_status_codes "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/status_codes"
)

func Test_CodedError_ImplementsError(t *testing.T) {
	var _ error = (*CodedError)(nil)
}

func Test_CodedError_ErrorAndCode(t *testing.T) {
	err := NewCodedError("test error", snx_lib_status_codes.ErrorCodeInsufficientMargin)
	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test error")
	}
	if err.Code() != snx_lib_status_codes.ErrorCodeInsufficientMargin {
		t.Errorf("Code() = %q, want %q", err.Code(), snx_lib_status_codes.ErrorCodeInsufficientMargin)
	}
}

func Test_ErrorsIs_Direct(t *testing.T) {
	if !errors.Is(ErrInsufficientMargin, ErrInsufficientMargin) {
		t.Error("errors.Is(ErrInsufficientMargin, ErrInsufficientMargin) should be true")
	}
}

func Test_ErrorsIs_Wrapped(t *testing.T) {
	wrapped := fmt.Errorf("processing order: %w", ErrInsufficientMargin)
	if !errors.Is(wrapped, ErrInsufficientMargin) {
		t.Error("errors.Is(wrapped, ErrInsufficientMargin) should be true")
	}
}

func Test_ErrorCodeFrom_Direct(t *testing.T) {
	got := ErrorCodeFrom(ErrInsufficientMargin)
	if got != snx_lib_status_codes.ErrorCodeInsufficientMargin {
		t.Errorf("ErrorCodeFrom(ErrInsufficientMargin) = %q, want %q", got, snx_lib_status_codes.ErrorCodeInsufficientMargin)
	}
}

func Test_ErrorCodeFrom_Wrapped(t *testing.T) {
	wrapped := fmt.Errorf("wrap: %w", ErrInsufficientMargin)
	got := ErrorCodeFrom(wrapped)
	if got != snx_lib_status_codes.ErrorCodeInsufficientMargin {
		t.Errorf("ErrorCodeFrom(wrapped) = %q, want %q", got, snx_lib_status_codes.ErrorCodeInsufficientMargin)
	}
}

func Test_ErrorCodeFrom_PlainError(t *testing.T) {
	got := ErrorCodeFrom(errors.New("plain"))
	if got != "" {
		t.Errorf("ErrorCodeFrom(plain error) = %q, want empty", got)
	}
}

func Test_ErrorCodeFrom_WithFallback(t *testing.T) {
	got := ErrorCodeFrom(errors.New("plain"), snx_lib_status_codes.ErrorCodeOrderRejectedByEngine)
	if got != snx_lib_status_codes.ErrorCodeOrderRejectedByEngine {
		t.Errorf("ErrorCodeFrom(plain, fallback) = %q, want %q", got, snx_lib_status_codes.ErrorCodeOrderRejectedByEngine)
	}
}

func Test_ErrorCodeFrom_Nil(t *testing.T) {
	got := ErrorCodeFrom(nil)
	if got != "" {
		t.Errorf("ErrorCodeFrom(nil) = %q, want empty", got)
	}
}

func Test_ErrorCodeFrom_NilWithFallback(t *testing.T) {
	got := ErrorCodeFrom(nil, snx_lib_status_codes.ErrorCodeOrderRejectedByEngine)
	if got != snx_lib_status_codes.ErrorCodeOrderRejectedByEngine {
		t.Errorf("ErrorCodeFrom(nil, fallback) = %q, want %q", got, snx_lib_status_codes.ErrorCodeOrderRejectedByEngine)
	}
}

func Test_ErrorCodeFrom_CodedErrorPrioritizedOverFallback(t *testing.T) {
	got := ErrorCodeFrom(ErrOrderNotFound, snx_lib_status_codes.ErrorCodeOrderRejectedByEngine)
	if got != snx_lib_status_codes.ErrorCodeOrderNotFound {
		t.Errorf("ErrorCodeFrom(ErrOrderNotFound, fallback) = %q, want %q", got, snx_lib_status_codes.ErrorCodeOrderNotFound)
	}
}

func Test_ErrorCodeFrom_DoubleWrapped(t *testing.T) {
	inner := fmt.Errorf("inner: %w", ErrMarketNotFound)
	outer := fmt.Errorf("outer: %w", inner)
	got := ErrorCodeFrom(outer)
	if got != snx_lib_status_codes.ErrorCodeMarketNotFound {
		t.Errorf("ErrorCodeFrom(double wrapped) = %q, want %q", got, snx_lib_status_codes.ErrorCodeMarketNotFound)
	}
}

func Test_ErrorCodeFrom_FailedValidationWrappingCodedError(t *testing.T) {
	// Outer error is plain; inner structured error should be found.
	wrapped := fmt.Errorf("%w: %w", Err_Order_FailedValidation, ErrInsufficientMargin)
	got := ErrorCodeFrom(wrapped)
	if got != snx_lib_status_codes.ErrorCodeInsufficientMargin {
		t.Errorf("ErrorCodeFrom(FailedValidation wrapping InsufficientMargin) = %q, want %q", got, snx_lib_status_codes.ErrorCodeInsufficientMargin)
	}
}

func Test_CodedError_Sentinels(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected snx_lib_status_codes.ErrorCode
	}{
		{"wick insurance", Err_Order_BlockedDuringWickInsurance, snx_lib_status_codes.ErrorCodeWickInsuranceActive},
		{"reduce only exceeds", Err_Order_InvalidReduceOnly_ExceedsPosition, snx_lib_status_codes.ErrorCodeReduceOnlyWouldIncrease},
		{"reduce only no position", Err_Order_InvalidReduceOnly_NoPosition, snx_lib_status_codes.ErrorCodeReduceOnlyNoPosition},
		{"reduce only same side", Err_Order_InvalidReduceOnly_SameSide, snx_lib_status_codes.ErrorCodeReduceOnlySameSide},
		{"duplicate clientOrderId", ErrDuplicateClientOrderId, snx_lib_status_codes.ErrorCodeIdempotencyConflict},
		{"insufficient margin", ErrInsufficientMargin, snx_lib_status_codes.ErrorCodeInsufficientMargin},
		{"invalid order side", ErrInvalidOrderSide, snx_lib_status_codes.ErrorCodeInvalidOrderSide},
		{"invalid trigger price", ErrInvalidTriggerPrice, snx_lib_status_codes.ErrorCodeInvalidTriggerPrice},
		{"market closed", ErrMarketClosed, snx_lib_status_codes.ErrorCodeMarketClosed},
		{"market not found", ErrMarketNotFound, snx_lib_status_codes.ErrorCodeMarketNotFound},
		{"market not open", ErrMarketNotOpen, snx_lib_status_codes.ErrorCodeMarketClosed},
		{"order not found", ErrOrderNotFound, snx_lib_status_codes.ErrorCodeOrderNotFound},
		{"order rejected", ErrOrderRejected, snx_lib_status_codes.ErrorCodeOrderRejectedByEngine},
		{"quantity below filled", ErrQuantityBelowFilled, snx_lib_status_codes.ErrorCodeQuantityBelowFilled},
		{"quantity below minimum", ErrQuantityBelowMinimum, snx_lib_status_codes.ErrorCodeQuantityTooSmall},
		{"context cancelled", ErrContextCancelled, snx_lib_status_codes.ErrorCodeOperationTimeout},
		{"context timeout", ErrContextTimeout, snx_lib_status_codes.ErrorCodeOperationTimeout},
		{"invalid sub account id", ErrInvalidSubAccountId, snx_lib_status_codes.ErrorCodeInvalidValue},
		{"sub account id required", ErrSubAccountIdRequired, snx_lib_status_codes.ErrorCodeInvalidValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorCodeFrom(tt.err)
			if got != tt.expected {
				t.Errorf("ErrorCodeFrom(%v) = %q, want %q", tt.err, got, tt.expected)
			}

			// Also verify wrapping works
			wrapped := fmt.Errorf("context: %w", tt.err)
			got = ErrorCodeFrom(wrapped)
			if got != tt.expected {
				t.Errorf("ErrorCodeFrom(wrapped %v) = %q, want %q", tt.err, got, tt.expected)
			}
		})
	}
}
