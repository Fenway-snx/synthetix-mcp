package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetRateLimitsSubaccountSnapshot_PublicCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		snapshot *GetRateLimitsSubaccountSnapshot
		wantUsed int
		wantCap  int
	}{
		{
			name:     "NIL_RECEIVER_RETURNS_ZEROS",
			snapshot: nil,
			wantUsed: 0,
			wantCap:  0,
		},
		{
			name: "TYPICAL_AFTER_DEBIT",
			snapshot: &GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 1155,
				Limit:           1200,
			},
			wantUsed: 45,
			wantCap:  1200,
		},
		{
			name: "FULL_BUCKET",
			snapshot: &GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 100,
				Limit:           100,
			},
			wantUsed: 0,
			wantCap:  100,
		},
		{
			name: "CLAMPS_NEGATIVE_USED",
			snapshot: &GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 15,
				Limit:           10,
			},
			wantUsed: 0,
			wantCap:  10,
		},
		{
			name: "ZERO_LIMIT_FROM_LIMITER",
			snapshot: &GetRateLimitsSubaccountSnapshot{
				AvailableTokens: 0,
				Limit:           0,
			},
			wantUsed: 0,
			wantCap:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotUsed, gotCap := tt.snapshot.PublicCounts()
			assert.Equal(t, tt.wantUsed, gotUsed)
			assert.Equal(t, tt.wantCap, gotCap)
		})
	}
}
