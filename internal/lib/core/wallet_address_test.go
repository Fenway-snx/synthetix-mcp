package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ChecksumWalletAddress(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantChecksum string
		wantErr      bool
	}{
		{
			name:         "lowercase returns EIP-55 checksummed",
			input:        "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			wantChecksum: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			wantErr:      false,
		},
		{
			name:         "uppercase normalizes to checksummed",
			input:        "0X5AAEB6053F3E94C9B9A09F33669435E7EF1BEAED",
			wantChecksum: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			wantErr:      false,
		},
		{
			name:         "trimmed whitespace",
			input:        "  0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed  ",
			wantChecksum: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			wantErr:      false,
		},
		{
			name:    "empty string returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only returns error",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "invalid hex returns error",
			input:   "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ChecksumWalletAddress(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, Err_WalletAddress_Invalid)
				return
			}
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.wantChecksum, got)
		})
	}
}

func Test_MaskAddress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard 42-char address",
			input: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
			want:  "0x5aAe***eAed",
		},
		{
			name:  "lowercase address",
			input: "0x1234567890abcdef1234567890abcdef12345678",
			want:  "0x1234***5678",
		},
		{
			name:  "zero address",
			input: "0x0000000000000000000000000000000000000000",
			want:  "0x0000***0000",
		},
		{
			name:  "exactly 10 chars",
			input: "0x12345678",
			want:  "0x1234***5678",
		},
		{
			name:  "9 chars returns redacted",
			input: "0x1234567",
			want:  "***",
		},
		{
			name:  "empty string returns redacted",
			input: "",
			want:  "***",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MaskAddress(tt.input))
		})
	}
}

func Test_NewWalletAddress(t *testing.T) {
	tests := []struct {
		name        string
		strWallet   string
		expectError bool
		err         error
	}{
		// TODO: Are we actually going to be case insensitive and return it checksummed?
		{
			name:        "valid lowercase address",
			strWallet:   "0x1234567890abcdef1234567890abcdef12345678",
			expectError: false,
			err:         nil,
		},
		{
			name:        "valid uppercase address",
			strWallet:   "0xABCDEF1234567890ABCDEF1234567890ABCDEF12",
			expectError: false,
			err:         nil,
		},
		{
			name:        "valid zero address",
			strWallet:   "0x0000000000000000000000000000000000000000",
			expectError: false,
			err:         nil,
		},
		{
			name:        "valid mixed case address",
			strWallet:   "0xDeadBeefDeadBeefDeadBeefDeadBeefDeadBeef",
			expectError: false,
			err:         nil,
		},
		{
			name:        "empty string",
			strWallet:   "",
			expectError: true,
			err:         Err_WalletAddress_Invalid,
		},
		// @Unit we should probably not allow this
		{
			name:        "valid address without 0x prefix",
			strWallet:   "1234567890abcdef1234567890abcdef12345678",
			expectError: false,
			err:         nil,
		},
		{
			name:        "too short address",
			strWallet:   "0x1234",
			expectError: true,
			err:         Err_WalletAddress_Invalid,
		},
		{
			name:        "too long address",
			strWallet:   "0x1234567890abcdef1234567890abcdef1234567890",
			expectError: true,
			err:         Err_WalletAddress_Invalid,
		},
		{
			name:        "invalid hex characters",
			strWallet:   "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",
			expectError: true,
			err:         Err_WalletAddress_Invalid,
		},
		{
			name:        "random string",
			strWallet:   "not-a-valid-address",
			expectError: true,
			err:         Err_WalletAddress_Invalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewWalletAddress(tt.strWallet)

			if tt.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.err)
				assert.Equal(t, WalletAddress_Zero, result)
			} else {
				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, WalletAddress(tt.strWallet), result)
			}
		})
	}
}
