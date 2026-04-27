package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValidateLogTags(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "empty string is rejected",
			raw:     "",
			wantErr: "must not be empty",
		},
		{
			name: "single valid pair",
			raw:  "env:prod",
		},
		{
			name: "multiple valid pairs",
			raw:  "env:prod,region:ap-northeast-1",
		},
		{
			name: "value containing colon is valid",
			raw:  "key:val:extra",
		},
		{
			name: "whitespace around pairs is accepted",
			raw:  " env:prod , region:ap-northeast-1 ",
		},
		{
			name:    "empty segment from double comma is rejected",
			raw:     "env:prod,,region:ap-northeast-1",
			wantErr: "empty segment",
		},
		{
			name:    "trailing comma produces empty segment and is rejected",
			raw:     "env:prod,",
			wantErr: "empty segment",
		},
		{
			name:    "pair without colon is rejected",
			raw:     "invalid",
			wantErr: "missing ':'",
		},
		{
			name:    "pair with leading colon has empty key and is rejected",
			raw:     ":value",
			wantErr: "empty key",
		},
		{
			name:    "first invalid pair in a list is caught",
			raw:     "good:pair,badpair,region:ap-northeast-1",
			wantErr: "missing ':'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateLogTags(tc.raw)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func Test_ParseLogTags(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []any
	}{
		{
			name: "empty string returns nil",
			raw:  "",
			want: nil,
		},
		{
			name: "single pair",
			raw:  "env:prod",
			want: []any{"env", "prod"},
		},
		{
			name: "multiple pairs",
			raw:  "env:prod,region:ap-northeast-1",
			want: []any{"env", "prod", "region", "ap-northeast-1"},
		},
		{
			name: "value containing colon uses first colon only",
			raw:  "key:val:extra",
			want: []any{"key", "val:extra"},
		},
		{
			name: "whitespace around pair is trimmed",
			raw:  " env:prod , region:ap-northeast-1 ",
			want: []any{"env", "prod", "region", "ap-northeast-1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseLogTags(tc.raw))
		})
	}
}
