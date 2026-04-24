package dlq

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

// --- inline spy (avoids circular import with lib/dlq/doubles) ----------------

type delivererEntry struct {
	envelope           Envelope
	envelopeJSONString string
}

type inlineSpyDeliverer struct {
	entries []delivererEntry
}

func (s *inlineSpyDeliverer) OnPost(envelope Envelope, envelopeJSONString string) error {
	s.entries = append(s.entries, delivererEntry{envelope, envelopeJSONString})
	return nil
}

// --- sentinel cache values ---------------------------------------------------

func sentinelCache() cachedInvariants {
	return cachedInvariants{
		byteOrder:   "test-byte-order",
		commit:      "test-commit-abc123",
		hostName:    "cached-sentinel-host",
		ipAddresses: []string{"192.0.2.1"},
		pId:         99999,
		processName: "sentinel-process",
	}
}

// --- Baseline: handler alone uses its cache ----------------------------------

func Test_dlqHandler_Post_USES_CACHED_INVARIANTS(t *testing.T) {
	t.Parallel()

	spy := &inlineSpyDeliverer{}

	handler := &dlqHandler{
		logger:    snx_lib_logging_doubles.NewStubLogger(),
		ctx:       context.Background(),
		deliverer: spy,
		cached:    sentinelCache(),
	}

	err := handler.Post("letter", Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.entries, 1)

	assertSentinelInvariants(t, spy.entries[0].envelopeJSONString)
}

// --- Decorator path must also use the handler's cache ------------------------

func Test_defaultEnvelopeDecorator_Post_USES_HANDLER_CACHED_INVARIANTS(t *testing.T) {
	t.Parallel()

	spy := &inlineSpyDeliverer{}

	handler := &dlqHandler{
		logger:    snx_lib_logging_doubles.NewStubLogger(),
		ctx:       context.Background(),
		deliverer: spy,
		cached:    sentinelCache(),
	}

	decorator := NewDefaultEnvelopeDecorator(handler, Envelope{
		Application: "test-app",
	})

	err := decorator.Post("letter", Envelope{Subsystem: "test-sub"})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.entries, 1)

	assertSentinelInvariants(t, spy.entries[0].envelopeJSONString)
}

// --- helpers -----------------------------------------------------------------

func assertSentinelInvariants(t *testing.T, envelopeJSON string) {
	t.Helper()

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(envelopeJSON), &parsed))

	assert.Equal(t, "cached-sentinel-host", parsed["host_name"],
		"hostname should come from handler cache, not from os.Hostname()",
	)
	assert.Equal(t, float64(99999), parsed["pid"],
		"PID should come from handler cache, not from os.Getpid()",
	)
	assert.Equal(t, "sentinel-process", parsed["process_name"],
		"process name should come from handler cache",
	)
	assert.Equal(t, "test-byte-order", parsed["byte_order"],
		"byte order should come from handler cache",
	)
	assert.Equal(t, "test-commit-abc123", parsed["commit"],
		"commit should come from handler cache",
	)

	ips, ok := parsed["ip_addresses"].([]any)
	require.True(t, ok, "ip_addresses should be an array")
	assert.Equal(t, []any{"192.0.2.1"}, ips,
		"IP addresses should come from handler cache",
	)
}
