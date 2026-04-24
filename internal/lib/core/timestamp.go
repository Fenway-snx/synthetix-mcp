package core

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	TimestampMaximumValidValue = 0x000E_9326_DD03_C000 - 1 // January 1, 2100 less one millisecond
)

var (
	errTimestampCannotMarshalInvalidValue = errors.New("cannot marshal invalid timestamp value")
	errTimestampInvalidValue              = errors.New("timestamp invalid value")
	errTimestampNil                       = errors.New("timestamp cannot be converted from nil")
	errTimestampValueOutOfRange           = errors.New("timestamp value out of range")
)

// An efficient, immutable type that holds UTC-based times for use
// throughout the core, and interoperable with time.Time, with Protobuf's
// timestamp, and - being convertible to different units (seconds,
// milliseconds, microseconds) epoch time - also with the time values used
// in the API.
//
// Design parameters:
// - immutable, so no accidental modifications;
// - small, so can (and should) be used by value;
// - strong type, so no accidental conversions;
// - no control fields, so implicit comparison is meaningful;
//
// Note:
// Because this is a type for use internal to the system, it is biased
// towards an assumption of correctness, and it is invalid to create an
// instance with an out-of-range value, and will produce undefined
// behaviour.
type Timestamp struct {
	epochMicros int64 // Number of microseconds since Unix epoch
}

// Special sentinel values.
var (
	Timestamp_Zero    Timestamp = Timestamp{epochMicros: 0}  // Represents the case where no time value was specified or obtained, such as when attempting to unmarshal from a missing field
	Timestamp_Invalid Timestamp = Timestamp{epochMicros: -1} // A time that is not valid. This value will always be obtained as the placeholder return value in functions that fail
	Timestamp_Never   Timestamp = Timestamp{epochMicros: -2} // Represents a time that will never come
)

// ===========================
// Creation functions
// ===========================

// Returns the corresponding timestamp to the current UTC time.
func TimestampNow() Timestamp {
	now := snx_lib_utils_time.Now()

	return TimestampFromTimeTime(now)
}

// Returns the corresponding timestamp to the given date elements describing
// a UTC point in time.
func TimestampDate(
	year int,
	month time.Month,
	day int,
	hour int,
	min int,
	sec int,
	nsec int,
) (Timestamp, error) {

	t := time.Date(
		year,
		month,
		day,
		hour,
		min,
		sec,
		nsec,
		time.UTC,
	)

	return timestampFromMicrosecondsInt64(t.UnixMicro())
}

func TimestampFromMicro(epochMicros int64) Timestamp {
	return Timestamp{epochMicros}
}

func TimestampFromMilli(epochMillis int64) Timestamp {
	epochMicros := epochMillis * 1_000

	return Timestamp{epochMicros}
}

func TimestampFromSec(epochSecs int64) Timestamp {
	epochMicros := epochSecs * 1_000_000

	return Timestamp{epochMicros}
}

func TimestampFromTimeTime(t time.Time) Timestamp {
	epochMicros := t.UnixMicro()

	return Timestamp{epochMicros}
}

// ===========================
// Methods
// ===========================

func (ts Timestamp) _internalCount_us() int64 {
	return ts.epochMicros
}

func (ts Timestamp) Nanooseconds() int64 {
	return ts._internalCount_us() * 1_000
}

func (ts Timestamp) Microseconds() int64 {
	return ts._internalCount_us()
}

func (ts Timestamp) Milliseconds() int64 {
	return ts._internalCount_us() / 1_000
}

func (ts Timestamp) Seconds() int64 {
	return ts._internalCount_us() / 1_000_000
}

func (ts Timestamp) String() string {
	return strconv.FormatInt(ts._internalCount_us(), 10)
}

// ===========================
// JSON Marshaling
// ===========================

// Marshal the microsecond value of the timestamp.
func (ts Timestamp) MarshalJSON() (bytes []byte, err error) {
	// Always marshal as an integer, containing microseconds

	i_us := ts._internalCount_us()

	if i_us < 0 || i_us > TimestampMaximumValidValue {
		err = errTimestampCannotMarshalInvalidValue
	} else {
		bytes, err = json.Marshal(i_us)
	}

	return
}

// Unmarshal from a microsecond representation.
func (ts *Timestamp) UnmarshalJSON(data []byte) (err error) {
	var i_us int64

	err = json.Unmarshal(data, &i_us)
	if err == nil {
		*ts, err = timestampFromMicrosecondsInt64(i_us)
	}

	return
}

// ===========================
// Helper functions
// ===========================

// [PRIVATE] Helper function for TimestampFromMilliseconds[T]().
func timestampFromMicrosecondsInt64(
	v int64,
) (Timestamp, error) {

	if v < 0 {
		return Timestamp_Invalid, errTimestampInvalidValue
	}

	if v > TimestampMaximumValidValue {
		return Timestamp_Invalid, errTimestampValueOutOfRange
	}

	return Timestamp{epochMicros: v}, nil
}

// ===========================
// API functions
// ===========================

// ===========================
// Methods
// ===========================
