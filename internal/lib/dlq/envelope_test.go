package dlq

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// --- isDefault --------------------------------------------------------------

func Test_Envelope_isDefault_ZERO_VALUE(t *testing.T) {
	t.Parallel()

	var e Envelope

	assert.True(t, e.isDefault())
}

func Test_Envelope_isDefault_APPLICATION_SET(t *testing.T) {
	t.Parallel()

	e := Envelope{Application: "app"}

	assert.False(t, e.isDefault())
}

func Test_Envelope_isDefault_SYSTEM_SET(t *testing.T) {
	t.Parallel()

	e := Envelope{System: "sys"}

	assert.False(t, e.isDefault())
}

func Test_Envelope_isDefault_SUBSYSTEM_SET(t *testing.T) {
	t.Parallel()

	e := Envelope{Subsystem: "sub"}

	assert.False(t, e.isDefault())
}

func Test_Envelope_isDefault_POST_TIME_SET(t *testing.T) {
	t.Parallel()

	e := Envelope{PostTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}

	assert.False(t, e.isDefault())
}

func Test_Envelope_isDefault_MULTIPLE_FIELDS_SET(t *testing.T) {
	t.Parallel()

	e := Envelope{
		Application: "app",
		System:      "sys",
		Subsystem:   "sub",
	}

	assert.False(t, e.isDefault())
}

// --- mergeOver --------------------------------------------------------------

func Test_Envelope_mergeOver_BOTH_ZERO(t *testing.T) {
	t.Parallel()

	result := Envelope{}.mergeOver(Envelope{})

	assert.Equal(t, Envelope{}, result)
}

func Test_Envelope_mergeOver_LHS_FIELDS_SURVIVE_WHEN_RHS_IS_ZERO(t *testing.T) {
	t.Parallel()

	lhs := Envelope{
		Application: "lhs-app",
		System:      "lhs-sys",
		Subsystem:   "lhs-sub",
	}

	result := lhs.mergeOver(Envelope{})

	assert.Equal(t, "lhs-app", result.Application)
	assert.Equal(t, "lhs-sys", result.System)
	assert.Equal(t, "lhs-sub", result.Subsystem)
}

func Test_Envelope_mergeOver_RHS_FIELDS_WIN_WHEN_LHS_IS_ZERO(t *testing.T) {
	t.Parallel()

	rhs := Envelope{
		Application: "rhs-app",
		System:      "rhs-sys",
		Subsystem:   "rhs-sub",
	}

	result := Envelope{}.mergeOver(rhs)

	assert.Equal(t, "rhs-app", result.Application)
	assert.Equal(t, "rhs-sys", result.System)
	assert.Equal(t, "rhs-sub", result.Subsystem)
}

func Test_Envelope_mergeOver_RHS_OVERRIDES_LHS_PUBLIC_FIELDS(t *testing.T) {
	t.Parallel()

	pt := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	lhs := Envelope{
		Application: "lhs-app",
		System:      "lhs-sys",
		Subsystem:   "lhs-sub",
	}
	rhs := Envelope{
		Application: "rhs-app",
		System:      "rhs-sys",
		PostTime:    pt,
	}

	result := lhs.mergeOver(rhs)

	assert.Equal(t, "rhs-app", result.Application)
	assert.Equal(t, "rhs-sys", result.System)
	assert.Equal(t, "lhs-sub", result.Subsystem, "lhs value retained when rhs is zero")
	assert.Equal(t, pt, result.PostTime)
}

func Test_Envelope_mergeOver_LHS_INTERNALS_PRESERVED(t *testing.T) {
	t.Parallel()

	lhs := Envelope{}
	lhs.envelopeInternals.HostName = "lhs-host"
	lhs.envelopeInternals.PId = 42

	rhs := Envelope{Application: "rhs-app"}
	rhs.envelopeInternals.HostName = "rhs-host"
	rhs.envelopeInternals.PId = 99

	result := lhs.mergeOver(rhs)

	assert.Equal(t, "rhs-app", result.Application)
	assert.Equal(t, "lhs-host", result.HostName(), "lhs internals must be preserved")
	assert.Equal(t, 42, result.PId(), "lhs internals must be preserved")
}

// --- affix (without cache) --------------------------------------------------

func Test_Envelope_affix_POPULATES_ALL_FIELDS_ON_ZERO_ENVELOPE(t *testing.T) {
	t.Parallel()

	e := Envelope{
		PostTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	e.affix(0, nil)

	assert.NotEmpty(t, e.ByteOrder())
	assert.NotEmpty(t, e.HostName())
	assert.NotEmpty(t, e.IPAddresses())
	assert.NotZero(t, e.PId())
	assert.NotEmpty(t, e.ProcessName())
	assert.NotEmpty(t, e.FileLineFunction())
	assert.NotZero(t, e.GoroutineCount())
}

// Verifies that affix populates PostTime from the time provider
// when it is zero. Not parallel because it mutates the global
// time provider.
func Test_Envelope_affix_POPULATES_POST_TIME(t *testing.T) {
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	cleanup := snx_lib_utils_time.SetTimeProvider(
		snx_lib_utils_time.NewFixedTimeProvider(fixedTime),
	)
	defer cleanup()

	var e Envelope

	e.affix(0, nil)

	assert.Equal(t, fixedTime, e.PostTime)
}

func Test_Envelope_affix_DOES_NOT_OVERWRITE_PRESET_FIELDS(t *testing.T) {
	t.Parallel()

	presetTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	e := Envelope{
		PostTime: presetTime,
	}
	e.envelopeInternals.HostName = "preset-host"
	e.envelopeInternals.PId = 12345
	e.envelopeInternals.ByteOrder = "preset-order"
	e.envelopeInternals.FileLineFunction = "preset:1:fn"
	e.envelopeInternals.GoroutineCount = 7

	e.affix(0, nil)

	assert.Equal(t, presetTime, e.PostTime, "preset PostTime must survive")
	assert.Equal(t, "preset-host", e.HostName(), "preset HostName must survive")
	assert.Equal(t, 12345, e.PId(), "preset PId must survive")
	assert.Equal(t, "preset-order", e.ByteOrder(), "preset ByteOrder must survive")
	assert.Equal(t, "preset:1:fn", e.FileLineFunction(), "preset FileLineFunction must survive")
	assert.Equal(t, 7, e.GoroutineCount(), "preset GoroutineCount must survive")
}

// Calling affix twice on the same envelope produces the same result.
func Test_Envelope_affix_IDEMPOTENT(t *testing.T) {
	t.Parallel()

	e := Envelope{
		PostTime: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}

	e.affix(0, nil)

	snapshot := e

	e.affix(0, nil)

	assert.Equal(t, snapshot.PostTime, e.PostTime)
	assert.Equal(t, snapshot.HostName(), e.HostName())
	assert.Equal(t, snapshot.PId(), e.PId())
	assert.Equal(t, snapshot.ByteOrder(), e.ByteOrder())
	assert.Equal(t, snapshot.ProcessName(), e.ProcessName())
	assert.Equal(t, snapshot.Commit(), e.Commit())
	assert.Equal(t, snapshot.FileLineFunction(), e.FileLineFunction())
	assert.Equal(t, snapshot.GoroutineCount(), e.GoroutineCount())
}

// --- affix (with cache) -----------------------------------------------------

func Test_Envelope_affix_WITH_CACHE_STAMPS_INVARIANTS(t *testing.T) {
	t.Parallel()

	cached := &cachedInvariants{
		byteOrder:   "test-endian",
		commit:      "abc123",
		hostName:    "cached-host",
		ipAddresses: []string{"10.0.0.1"},
		pId:         9999,
		processName: "cached-proc",
	}

	e := Envelope{
		PostTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	e.affix(0, cached)

	assert.Equal(t, "test-endian", e.ByteOrder())
	assert.Equal(t, "abc123", e.Commit())
	assert.Equal(t, "cached-host", e.HostName())
	assert.Equal(t, []string{"10.0.0.1"}, e.IPAddresses())
	assert.Equal(t, 9999, e.PId())
	assert.Equal(t, "cached-proc", e.ProcessName())
}

func Test_Envelope_affix_WITH_CACHE_STILL_COMPUTES_PER_POST_FIELDS(t *testing.T) {
	t.Parallel()

	cached := &cachedInvariants{
		byteOrder:   "test-endian",
		commit:      "abc123",
		hostName:    "cached-host",
		ipAddresses: []string{"10.0.0.1"},
		pId:         9999,
		processName: "cached-proc",
	}

	e := Envelope{
		PostTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	e.affix(0, cached)

	assert.NotEmpty(t, e.FileLineFunction(), "FileLineFunction must be computed fresh")
	assert.NotZero(t, e.GoroutineCount(), "GoroutineCount must be computed fresh")
}

func Test_Envelope_affix_WITH_CACHE_DOES_NOT_OVERWRITE_PRESET(t *testing.T) {
	t.Parallel()

	cached := &cachedInvariants{
		byteOrder:   "cached-endian",
		commit:      "cached-commit",
		hostName:    "cached-host",
		ipAddresses: []string{"10.0.0.1"},
		pId:         9999,
		processName: "cached-proc",
	}

	e := Envelope{
		PostTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	e.envelopeInternals.HostName = "preset-host"
	e.envelopeInternals.PId = 42

	e.affix(0, cached)

	assert.Equal(t, "preset-host", e.HostName(), "preset overrides cache")
	assert.Equal(t, 42, e.PId(), "preset overrides cache")
	assert.Equal(t, "cached-endian", e.ByteOrder(), "non-preset field uses cache")
}

// --- prepare ----------------------------------------------------------------

func Test_Envelope_prepare_SIMPLE_STRUCT(t *testing.T) {
	t.Parallel()

	letter := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		Name: "Alice",
		Age:  30,
	}

	var e Envelope

	jsonStr, err := e.prepare(letter)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	var parsed map[string]json.RawMessage

	require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

	var letterField struct {
		StringForm                  string `json:"string_form"`
		JSONConversionWasIncomplete bool   `json:"json_conversion_was_incomplete"`
	}
	require.NoError(t, json.Unmarshal(parsed["letter"], &letterField))
	assert.Contains(t, letterField.StringForm, `"name":"Alice"`)
	assert.Contains(t, letterField.StringForm, `"age":30`)
	assert.False(t, letterField.JSONConversionWasIncomplete)
}

func Test_Envelope_prepare_STRING_LETTER(t *testing.T) {
	t.Parallel()

	var e Envelope

	jsonStr, err := e.prepare("hello world")

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Contains(t, jsonStr, `"string_form":"\"hello world\""`)
}

func Test_Envelope_prepare_nil_LETTER(t *testing.T) {
	t.Parallel()

	var e Envelope

	jsonStr, err := e.prepare(nil)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Contains(t, jsonStr, `"string_form":"null"`)
}

func Test_Envelope_prepare_UNMARSHALABLE_LETTER_FALLS_BACK(t *testing.T) {
	t.Parallel()

	type badLetter struct {
		Ch chan int `json:"ch"`
	}

	var e Envelope

	jsonStr, err := e.prepare(badLetter{Ch: make(chan int)})

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Contains(t, jsonStr, `"json_conversion_was_incomplete":true`)
}

func Test_Envelope_prepare_RESULT_IS_VALID_JSON(t *testing.T) {
	t.Parallel()

	e := Envelope{
		Application: "test-app",
		System:      "test-sys",
	}

	jsonStr, err := e.prepare(map[string]string{"key": "value"})

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.True(t, json.Valid([]byte(jsonStr)), "result must be valid JSON")
}

func Test_Envelope_prepare_ENVELOPE_FIELDS_IN_OUTPUT(t *testing.T) {
	t.Parallel()

	e := Envelope{
		Application: "my-app",
		System:      "my-sys",
		Subsystem:   "my-sub",
	}

	jsonStr, err := e.prepare("letter-body")

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Contains(t, jsonStr, `"application":"my-app"`)
	assert.Contains(t, jsonStr, `"system":"my-sys"`)
	assert.Contains(t, jsonStr, `"subsystem":"my-sub"`)
}

// --- computeInvariants ------------------------------------------------------

func Test_computeInvariants_ALL_FIELDS_POPULATED(t *testing.T) {
	t.Parallel()

	c := computeInvariants()

	assert.NotEmpty(t, c.byteOrder)
	assert.NotEmpty(t, c.hostName)
	assert.NotEmpty(t, c.ipAddresses)
	assert.NotZero(t, c.pId)
	assert.NotEmpty(t, c.processName)
}

func Test_computeInvariants_MATCHES_LIVE_VALUES(t *testing.T) {
	t.Parallel()

	c := computeInvariants()

	expectedHost, _ := os.Hostname()
	expectedPid := os.Getpid()
	expectedProc := filepath.Base(os.Args[0])

	assert.Equal(t, expectedHost, c.hostName)
	assert.Equal(t, expectedPid, c.pId)
	assert.Equal(t, expectedProc, c.processName)
}

// --- byteOrder --------------------------------------------------------------

func Test_byteOrder_RETURNS_KNOWN_VALUE(t *testing.T) {
	t.Parallel()

	bo := byteOrder()

	assert.Contains(t, []string{"big-endian", "little-endian"}, bo)
}

func Test_byteOrder_IS_STABLE(t *testing.T) {
	t.Parallel()

	assert.Equal(t, byteOrder(), byteOrder())
}

// --- Accessors --------------------------------------------------------------

func Test_Envelope_LetterString_RETURNS_INTERNALS(t *testing.T) {
	t.Parallel()

	var e Envelope

	e.envelopeInternals.Letter.String = `{"msg":"hello"}`
	e.envelopeInternals.Letter.JSONConversionWasIncomplete = true

	ls, incomplete := e.LetterString()

	assert.Equal(t, `{"msg":"hello"}`, ls)
	assert.True(t, incomplete)
}

func Test_Envelope_ACCESSORS_REFLECT_INTERNALS(t *testing.T) {
	t.Parallel()

	var e Envelope

	e.envelopeInternals.ByteOrder = "little-endian"
	e.envelopeInternals.Commit = "deadbeef"
	e.envelopeInternals.FileLineFunction = "foo.go:10:Bar"
	e.envelopeInternals.GoroutineCount = 42
	e.envelopeInternals.HostName = "testhost"
	e.envelopeInternals.IPAddresses = []string{"1.2.3.4"}
	e.envelopeInternals.PId = 777
	e.envelopeInternals.ProcessName = "testproc"

	assert.Equal(t, "little-endian", e.ByteOrder())
	assert.Equal(t, "deadbeef", e.Commit())
	assert.Equal(t, "foo.go:10:Bar", e.FileLineFunction())
	assert.Equal(t, 42, e.GoroutineCount())
	assert.Equal(t, "testhost", e.HostName())
	assert.Equal(t, []string{"1.2.3.4"}, e.IPAddresses())
	assert.Equal(t, 777, e.PId())
	assert.Equal(t, "testproc", e.ProcessName())
}

// --- affix FileLineFunction depth -------------------------------------------

// Verifies that the captured FileLineFunction references this test
// file, confirming depth arithmetic is correct.
func Test_Envelope_affix_FILE_LINE_FUNCTION_CAPTURES_CALLER(t *testing.T) {
	t.Parallel()

	e := Envelope{
		PostTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	e.affix(0, nil)

	assert.Contains(t, e.FileLineFunction(), "envelope_test.go",
		"FileLineFunction should reference this test file, got: %s", e.FileLineFunction())
}
