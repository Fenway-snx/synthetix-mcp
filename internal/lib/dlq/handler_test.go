package dlq_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_dlq_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq/doubles"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

// --- NewDLQHandler construction ---------------------------------------------

func Test_NewDLQHandler_nil_LOGGER_PANICS(t *testing.T) {
	t.Parallel()

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: parameter `logger` may not be `nil`",
		func() {
			snx_lib_dlq.NewDLQHandler(nil, context.Background(), snx_lib_dlq_doubles.NewStubDeliverer(), snx_lib_dlq.Envelope{})
		},
	)
}

func Test_NewDLQHandler_nil_ctx_PANICS(t *testing.T) {
	t.Parallel()

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: parameter `ctx` may not be `nil`",
		func() {
			snx_lib_dlq.NewDLQHandler(snx_lib_logging_doubles.NewSpyLogger(), nil, snx_lib_dlq_doubles.NewStubDeliverer(), snx_lib_dlq.Envelope{})
		},
	)
}

func Test_NewDLQHandler_nil_DELIVERER_PANICS(t *testing.T) {
	t.Parallel()

	snx_lib_utils_test.AssertPanicsContaining(t,
		"VIOLATION: parameter `deliverer` may not be `nil`",
		func() {
			snx_lib_dlq.NewDLQHandler(snx_lib_logging_doubles.NewSpyLogger(), context.Background(), nil, snx_lib_dlq.Envelope{})
		},
	)
}

func Test_NewDLQHandler_DEFAULT_ENVELOPE_RETURNS_WITHOUT_ERROR(t *testing.T) {
	t.Parallel()

	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		snx_lib_dlq_doubles.NewStubDeliverer(),
		snx_lib_dlq.Envelope{},
	)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.NotNil(t, dlq)
}

func Test_NewDLQHandler_NON_DEFAULT_ENVELOPE_RETURNS_WITHOUT_ERROR(t *testing.T) {
	t.Parallel()

	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		snx_lib_dlq_doubles.NewStubDeliverer(),
		snx_lib_dlq.Envelope{Application: "test-app"},
	)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.NotNil(t, dlq)
}

// --- Post: basic delivery ---------------------------------------------------

func Test_NewDLQHandler_Post_DELIVERS_TO_DELIVERER(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("hello", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
}

func Test_NewDLQHandler_Post_RETURNS_nil_ON_SUCCESS(t *testing.T) {
	t.Parallel()

	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		snx_lib_dlq_doubles.NewStubDeliverer(),
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post(map[string]string{"key": "value"}, snx_lib_dlq.Envelope{})

	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_NewDLQHandler_Post_ENVELOPE_JSON_IS_VALID(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("test-letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	assert.True(t, json.Valid([]byte(spy.Entries[0].EnvelopeJSONString)),
		"envelope JSON must be valid, got: %s", spy.Entries[0].EnvelopeJSONString,
	)
}

func Test_NewDLQHandler_Post_LETTER_CONTENT_IN_JSON(t *testing.T) {
	t.Parallel()

	type testLetter struct {
		OrderID string `json:"order_id"`
		Reason  string `json:"reason"`
	}

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post(testLetter{OrderID: "ORD-123", Reason: "margin call"}, snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	jsonStr := spy.Entries[0].EnvelopeJSONString
	assert.Contains(t, jsonStr, "ORD-123")
	assert.Contains(t, jsonStr, "margin call")
}

func Test_NewDLQHandler_Post_ENVELOPE_FIELDS_IN_JSON(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{
		Application: "trading",
		System:      "matching",
		Subsystem:   "orderbook",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	jsonStr := spy.Entries[0].EnvelopeJSONString
	assert.Contains(t, jsonStr, `"application":"trading"`)
	assert.Contains(t, jsonStr, `"system":"matching"`)
	assert.Contains(t, jsonStr, `"subsystem":"orderbook"`)
}

// --- Post: diagnostic fields ------------------------------------------------

func Test_NewDLQHandler_Post_AFFIXES_DIAGNOSTIC_FIELDS(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	env := spy.Entries[0].Envelope

	assert.NotEmpty(t, env.HostName(), "hostname should be affixed")
	assert.NotZero(t, env.PId(), "PID should be affixed")
	assert.NotEmpty(t, env.ProcessName(), "process name should be affixed")
	assert.NotEmpty(t, env.ByteOrder(), "byte order should be affixed")
	assert.NotEmpty(t, env.IPAddresses(), "IP addresses should be affixed")
	assert.NotEmpty(t, env.FileLineFunction(), "file/line/function should be affixed")
	assert.NotZero(t, env.GoroutineCount(), "goroutine count should be affixed")
	assert.False(t, env.PostTime.IsZero(), "post time should be affixed")
}

func Test_NewDLQHandler_Post_DIAGNOSTIC_FIELDS_IN_JSON(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	jsonStr := spy.Entries[0].EnvelopeJSONString

	var parsed map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

	assert.Contains(t, parsed, "host_name")
	assert.Contains(t, parsed, "pid")
	assert.Contains(t, parsed, "process_name")
	assert.Contains(t, parsed, "byte_order")
	assert.Contains(t, parsed, "ip_addresses")
	assert.Contains(t, parsed, "file_line_function")
	assert.Contains(t, parsed, "goroutine_count")
	assert.Contains(t, parsed, "post_time")
}

// --- Post: multiple posts ---------------------------------------------------

func Test_NewDLQHandler_Post_MULTIPLE_POSTS_ACCUMULATE(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.NoError(t, dlq.Post("first", snx_lib_dlq.Envelope{}))
	require.NoError(t, dlq.Post("second", snx_lib_dlq.Envelope{}))
	require.NoError(t, dlq.Post("third", snx_lib_dlq.Envelope{}))

	assert.Len(t, spy.Entries, 3)
	assert.Contains(t, spy.Entries[0].EnvelopeJSONString, "first")
	assert.Contains(t, spy.Entries[1].EnvelopeJSONString, "second")
	assert.Contains(t, spy.Entries[2].EnvelopeJSONString, "third")
}

// --- Post: error paths ------------------------------------------------------

func Test_NewDLQHandler_Post_PROPAGATES_DELIVERER_ERROR(t *testing.T) {
	t.Parallel()

	delivererErr := errors.New("NATS connection lost")
	spy := snx_lib_dlq_doubles.NewSpyDeliverer().WithOnPost(
		func(_ snx_lib_dlq.Envelope, _ string) error {
			return delivererErr
		},
	)
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})

	assert.ErrorIs(t, err, delivererErr)
}

func Test_NewDLQHandler_Post_DELIVERER_ERROR_DOES_NOT_LOG(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewSpyLogger()
	spy := snx_lib_dlq_doubles.NewSpyDeliverer().WithOnPost(
		func(_ snx_lib_dlq.Envelope, _ string) error {
			return errors.New("delivery failed")
		},
	)
	dlq, err := snx_lib_dlq.NewDLQHandler(
		logger,
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	_ = dlq.Post("letter", snx_lib_dlq.Envelope{})

	assert.Empty(t, logger.Entries(),
		"deliverer errors should not trigger logging (only prepare failures do)",
	)
}

func Test_NewDLQHandler_Post_UNMARSHALABLE_LETTER_FALLS_BACK(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	type badLetter struct {
		Ch chan int `json:"ch"`
	}

	err = dlq.Post(badLetter{Ch: make(chan int)}, snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	assert.Contains(t, spy.Entries[0].EnvelopeJSONString, `"json_conversion_was_incomplete":true`)
}

func Test_NewDLQHandler_Post_nil_LETTER_SUCCEEDS(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post(nil, snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	assert.Contains(t, spy.Entries[0].EnvelopeJSONString, `"string_form":"null"`)
}

// --- Post: default envelope merging -----------------------------------------

func Test_NewDLQHandler_Post_WITH_DEFAULT_ENVELOPE_MERGES_FIELDS(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{
			Application: "default-app",
			System:      "default-sys",
		},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	jsonStr := spy.Entries[0].EnvelopeJSONString
	assert.Contains(t, jsonStr, `"application":"default-app"`)
	assert.Contains(t, jsonStr, `"system":"default-sys"`)
}

func Test_NewDLQHandler_Post_WITH_DEFAULT_ENVELOPE_DEFAULT_OVERRIDES_CALLER(t *testing.T) {
	t.Parallel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		context.Background(),
		spy,
		snx_lib_dlq.Envelope{
			Application: "default-app",
			System:      "default-sys",
		},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{
		Application: "caller-app",
		Subsystem:   "caller-sub",
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Len(t, spy.Entries, 1)
	jsonStr := spy.Entries[0].EnvelopeJSONString
	assert.Contains(t, jsonStr, `"application":"default-app"`, "default overrides caller")
	assert.Contains(t, jsonStr, `"system":"default-sys"`, "default fills gap")
	assert.Contains(t, jsonStr, `"subsystem":"caller-sub"`, "caller preserved when default is empty")
}

// --- Post: success path does not log ----------------------------------------

func Test_NewDLQHandler_Post_SUCCESS_DOES_NOT_LOG(t *testing.T) {
	t.Parallel()

	logger := snx_lib_logging_doubles.NewSpyLogger()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		logger,
		context.Background(),
		snx_lib_dlq_doubles.NewStubDeliverer(),
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	assert.Empty(t, logger.Entries(), "successful post should not produce log entries")
}

// --- Post: shutdown behaviour -----------------------------------------------

func Test_NewDLQHandler_Post_RETURNS_ERROR_WHEN_CONTEXT_CANCELLED(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	spy := snx_lib_dlq_doubles.NewSpyDeliverer()
	dlq, err := snx_lib_dlq.NewDLQHandler(
		snx_lib_logging_doubles.NewSpyLogger(),
		ctx,
		spy,
		snx_lib_dlq.Envelope{},
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	err = dlq.Post("letter", snx_lib_dlq.Envelope{})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, spy.Entries, "deliverer must not be called after shutdown")
}
