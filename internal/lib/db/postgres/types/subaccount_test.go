package types

import (
	"database/sql/driver"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SubAccountType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  SubAccountType
		want string
	}{
		{
			name: "normal account",
			typ:  AccountTypeNormal,
			want: "normal",
		},
		{
			name: "slp account",
			typ:  AccountTypeSLP,
			want: "slp",
		},
		{
			name: "unknown account",
			typ:  SubAccountType(99),
			want: "unknown(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.typ.String())
		})
	}
}

func Test_SubAccountType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		typ  SubAccountType
		want string
	}{
		{
			name: "normal account",
			typ:  AccountTypeNormal,
			want: "0",
		},
		{
			name: "slp account",
			typ:  AccountTypeSLP,
			want: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.typ)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func Test_SubAccountType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SubAccountType
		wantErr bool
	}{
		{
			name:  "normal account",
			input: "0",
			want:  AccountTypeNormal,
		},
		{
			name:  "slp account",
			input: "1",
			want:  AccountTypeSLP,
		},
		{
			name:    "invalid json",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var typ SubAccountType
			err := json.Unmarshal([]byte(tt.input), &typ)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tt.want, typ)
			}
		})
	}
}

func Test_SubAccountType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		typ  SubAccountType
		want bool
	}{
		{
			name: "normal account",
			typ:  AccountTypeNormal,
			want: true,
		},
		{
			name: "slp account",
			typ:  AccountTypeSLP,
			want: true,
		},
		{
			name: "invalid account",
			typ:  SubAccountType(99),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.typ.IsValid())
		})
	}
}

func Test_SubAccountType_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    SubAccountType
		wantErr bool
	}{
		{
			name:  "scan int64 normal",
			value: int64(0),
			want:  AccountTypeNormal,
		},
		{
			name:  "scan int64 slp",
			value: int64(1),
			want:  AccountTypeSLP,
		},
		{
			name:  "scan int32",
			value: int32(1),
			want:  AccountTypeSLP,
		},
		{
			name:  "scan int",
			value: int(0),
			want:  AccountTypeNormal,
		},
		{
			name:  "scan nil",
			value: nil,
			want:  AccountTypeNormal, // defaults to normal
		},
		{
			name:    "scan invalid type",
			value:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var typ SubAccountType
			err := typ.Scan(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
				assert.Equal(t, tt.want, typ)
			}
		})
	}
}

func Test_SubAccountType_Value(t *testing.T) {
	tests := []struct {
		name string
		typ  SubAccountType
		want driver.Value
	}{
		{
			name: "normal account",
			typ:  AccountTypeNormal,
			want: int64(0),
		},
		{
			name: "slp account",
			typ:  AccountTypeSLP,
			want: int64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.typ.Value()
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.want, got)
		})
	}
}
