package postgres

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var transientErrorRe *regexp.Regexp

func init() {
	patterns := []string{
		"bad connection",
		"broken pipe",
		"connection refused",
		"connection reset",
		"connection timed out",
		"no connection",
	}

	escaped := make([]string, len(patterns))
	for i, p := range patterns {
		escaped[i] = regexp.QuoteMeta(p)
	}
	transientErrorRe = regexp.MustCompile(`(?i)(?:` + strings.Join(escaped, "|") + `)`)
}

// IsTransientError reports whether err represents a transient database failure
// (bad connection, network reset, timeout) that may succeed on retry.
// Context cancellation and deadline errors are NOT considered transient
// because the caller has already given up.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, driver.ErrBadConn) {
		return true
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// PostgreSQL connection-class errors (SQLSTATE class 08)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Code, "08") {
		return true
	}

	if transientErrorRe.MatchString(err.Error()) {
		return true
	}

	return false
}
