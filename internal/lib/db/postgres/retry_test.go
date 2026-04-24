package postgres

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func Test_IsTransientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "driver.ErrBadConn",
			err:      driver.ErrBadConn,
			expected: true,
		},
		{
			name:     "wrapped driver.ErrBadConn",
			err:      fmt.Errorf("failed to create order history: %w", driver.ErrBadConn),
			expected: true,
		},
		{
			name:     "io.EOF",
			err:      io.EOF,
			expected: true,
		},
		{
			name:     "io.ErrUnexpectedEOF",
			err:      io.ErrUnexpectedEOF,
			expected: true,
		},
		{
			name:     "context.Canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context.DeadlineExceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name: "PostgreSQL connection class error (08006)",
			err: &pgconn.PgError{
				Code: "08006",
			},
			expected: true,
		},
		{
			name: "PostgreSQL unique violation (23505)",
			err: &pgconn.PgError{
				Code: "23505",
			},
			expected: false,
		},
		{
			name:     "bad connection string match",
			err:      errors.New("driver: bad connection"),
			expected: true,
		},
		{
			name:     "connection reset string match",
			err:      errors.New("read tcp: connection reset by peer"),
			expected: true,
		},
		{
			name:     "broken pipe string match",
			err:      errors.New("write: broken pipe"),
			expected: true,
		},
		{
			name:     "wrapped bad connection",
			err:      fmt.Errorf("failed to create order history: %w", errors.New("driver: bad connection")),
			expected: true,
		},
		{
			name:     "mixed case connection refused",
			err:      errors.New("dial tcp: Connection Refused"),
			expected: true,
		},
		{
			name:     "upper case broken pipe",
			err:      errors.New("write: BROKEN PIPE"),
			expected: true,
		},
		{
			name:     "generic application error",
			err:      errors.New("invalid input syntax"),
			expected: false,
		},
		{
			name:     "timeout error via net.Error",
			err:      &timeoutError{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransientError(tt.err)
			if result != tt.expected {
				t.Errorf("IsTransientError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

var _ net.Error = (*timeoutError)(nil)
