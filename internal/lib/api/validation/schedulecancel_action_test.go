package validation

import (
	"fmt"
	"testing"

	snx_lib_runtime_deadmanswitch "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/deadmanswitch"
)

func Test_ValidateScheduleCancelAction(t *testing.T) {
	bounds, err := snx_lib_runtime_deadmanswitch.LoadTimeoutBounds()
	if err != nil {
		t.Fatalf("expected valid dead-man-switch bounds, got %v", err)
	}

	t.Run("accepts disable", func(t *testing.T) {
		timeoutSeconds := int64(0)
		validatedTimeout, err := ValidateScheduleCancelAction(&ScheduleCancelActionPayload{
			Action:         "scheduleCancel",
			TimeoutSeconds: &timeoutSeconds,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if validatedTimeout != 0 {
			t.Fatalf("expected timeout 0, got %d", validatedTimeout)
		}
	})

	t.Run("accepts bounded timeout", func(t *testing.T) {
		timeoutSeconds := int64(60)
		validatedTimeout, err := ValidateScheduleCancelAction(&ScheduleCancelActionPayload{
			Action:         "scheduleCancel",
			TimeoutSeconds: &timeoutSeconds,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if validatedTimeout != 60 {
			t.Fatalf("expected timeout 60, got %d", validatedTimeout)
		}
	})

	t.Run("rejects missing timeout", func(t *testing.T) {
		_, err := ValidateScheduleCancelAction(&ScheduleCancelActionPayload{Action: "scheduleCancel"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "timeoutSeconds is required" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects too small non-zero timeout", func(t *testing.T) {
		timeoutSeconds := bounds.MinTimeoutSeconds - 1
		_, err := ValidateScheduleCancelAction(&ScheduleCancelActionPayload{
			Action:         "scheduleCancel",
			TimeoutSeconds: &timeoutSeconds,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != fmt.Sprintf("timeoutSeconds must be 0 or at least %d", bounds.MinTimeoutSeconds) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects too large timeout", func(t *testing.T) {
		timeoutSeconds := bounds.MaxTimeoutSeconds + 1
		_, err := ValidateScheduleCancelAction(&ScheduleCancelActionPayload{
			Action:         "scheduleCancel",
			TimeoutSeconds: &timeoutSeconds,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != fmt.Sprintf("timeoutSeconds must be less than or equal to %d", bounds.MaxTimeoutSeconds) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func Test_NewValidatedScheduleCancelAction(t *testing.T) {
	timeoutSeconds := int64(60)
	validated, err := NewValidatedScheduleCancelAction(&ScheduleCancelActionPayload{
		Action:         "scheduleCancel",
		TimeoutSeconds: &timeoutSeconds,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if validated.TimeoutSeconds != 60 {
		t.Fatalf("expected timeout 60, got %d", validated.TimeoutSeconds)
	}
}
