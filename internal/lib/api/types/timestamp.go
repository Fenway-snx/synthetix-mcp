// API-specific strong time type.

package types

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	TimestampMaximumValidValue = 0x0000_03BB_2CC3_D800 - 1 // January 1, 2100 less one millisecond
)

var (
	errTimestampCannotMarshalInvalidValue = errors.New("cannot marshal invalid timestamp value")
	errTimestampInvalidValue              = errors.New("timestamp invalid value")
	errTimestampNil                       = errors.New("timestamp cannot be converted from nil")
	errTimestampValueOutOfRange           = errors.New("timestamp value out of range")
)

// Unix epoch timestamps, expressed in milliseconds.
type Timestamp int64

// Special sentinel values.
//
// NOTE: the value of these values will not be realised until we complete
// the time-abstraction throughout all of the API layers.
const (
	Timestamp_Zero    Timestamp = 0  // Represents the case where no time value was specified or obtained, such as when attempting to unmarshal from a missing field
	Timestamp_Invalid Timestamp = -1 // A time that is not valid. This value will always be obtained as the placeholder return value in functions that fail
	Timestamp_Never   Timestamp = -2 // Represents a time that will never come
)

// ===========================
// Creation functions
// ===========================

// Returns the corresponding timestamp to the current UTC time.
func TimestampNow() Timestamp {
	now := snx_lib_utils_time.Now()

	r, _ := timestampFromMillisecondsInt64(now.UnixMilli())

	return r
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

	return timestampFromMillisecondsInt64(t.UnixMilli())
}

// ===========================
// Methods
// ===========================

func (ts Timestamp) _internalCount_ms() int64 {
	return int64(ts)
}

func (ts Timestamp) _internalCountIfValidOtherwiseZero() int64 {
	i_ms, isValid := ts._internal_IsValid()

	if isValid {
		return i_ms
	} else {
		return 0
	}
}

func (ts Timestamp) _internal_IsValid() (i_ms int64, isValid bool) {
	i_ms = ts._internalCount_ms()

	isValid = i_ms >= 0 && i_ms < TimestampMaximumValidValue

	return
}

func (ts Timestamp) IsValid() bool {
	_, isValid := ts._internal_IsValid()

	return isValid
}

// Obtains the number of whole nanoseconds in a valid Timestamp, or 0
// otherwise.
func (ts Timestamp) Nanoseconds() int64 {
	return ts._internalCountIfValidOtherwiseZero() * 1_000_000
}

// Obtains the number of whole microseconds in a valid Timestamp, or 0
// otherwise.
func (ts Timestamp) Microseconds() int64 {
	return ts._internalCountIfValidOtherwiseZero() * 1_000
}

// Obtains the number of whole milliseconds in a valid Timestamp, or 0
// otherwise.
func (ts Timestamp) Milliseconds() int64 {
	return ts._internalCountIfValidOtherwiseZero()
}

// Obtains the number of whole seconds in a valid Timestamp, or 0 otherwise.
func (ts Timestamp) Seconds() int64 {
	return ts._internalCountIfValidOtherwiseZero() / 1_000
}

// Attempts to convert to a time.Time value.
func (ts Timestamp) AsTime() (time.Time, error) {
	i_ms, isValid := ts._internal_IsValid()

	if !isValid {
		return time.Time{}, errTimestampInvalidValue
	} else {
		return time.UnixMilli(i_ms).UTC(), nil
	}
}

// ===========================
// JSON Marshaling
// ===========================

// Marshal the millisecond value of the timestamp.
func (ts Timestamp) MarshalJSON() (bytes []byte, err error) {
	// Always marshal as an integer, containing milliseconds

	i_ms := ts._internalCount_ms()

	if i_ms < 0 {
		err = errTimestampCannotMarshalInvalidValue
	} else {
		bytes, err = json.Marshal(i_ms)
	}

	return
}

// Unmarshal from a millisecond representation.
func (ts *Timestamp) UnmarshalJSON(data []byte) (err error) {
	// When unmarshaling a number into an `any`, Go obtains a `float64`, so we
	// don't do that, and instead as for the following:
	//
	// 1. int64 (milliseconds);
	// 2. string (milliseconds);

	// 1. int64 (milliseconds);
	{
		var i int64

		err = json.Unmarshal(data, &i)
		if err == nil {
			*ts, err = timestampFromMillisecondsInt64(i)

			return
		}
	}

	// 2. string (milliseconds);
	{
		var s string

		err = json.Unmarshal(data, &s)
		if err == nil {

			s = strings.TrimSpace(s)
			s = strings.ReplaceAll(s, "_", "") // be permissive of underscores - this mainly to help in unit-testing

			if s == "" {
				*ts = Timestamp_Zero

				return
			} else {
				var i int64
				if i, err = strconv.ParseInt(s, 10, 64); err != nil {
					return
				} else {
					*ts, err = timestampFromMillisecondsInt64(i)

					return
				}
			}
		}
	}

	err = errTimestampCannotMarshalInvalidValue

	return
}

// ===========================
// Helper functions
// ===========================

// [PRIVATE] Helper function for TimestampFromMilliseconds[T]().
func timestampFromMillisecondsInt64(
	v int64,
) (Timestamp, error) {

	if v < 0 {
		return Timestamp_Invalid, errTimestampInvalidValue
	}

	if v > TimestampMaximumValidValue {
		return Timestamp_Invalid, errTimestampValueOutOfRange
	}

	return Timestamp(v), nil
}

// [PRIVATE] Helper function for TimestampFromMilliseconds[T]().
func timestampFromMillisecondsUint64(
	v uint64,
) (Timestamp, error) {

	const maxInt64Unsigned uint64 = math.MaxInt64

	if v > maxInt64Unsigned {
		return Timestamp_Invalid, errTimestampInvalidValue
	}

	if v > TimestampMaximumValidValue {
		return Timestamp_Invalid, errTimestampValueOutOfRange
	}

	return Timestamp(int64(v)), nil
}

// [PRIVATE] Attempts to obtain a Timestamp from the given raw number of
// milliseconds since epoch.
//
// Note:
// This function MUST NOT be called with any type other than `int`, `int32`,
// `int64`. The larger list of types is purely to allow calling from
// TimestampFromMilliseconds().
func timestampFromMillisecondsSigned[T int32 | int64 | int | uint32 | uint64 | uint](
	v T,
) (Timestamp, error) {

	var i64 int64

	switch i := any(v).(type) {
	case int32:
		i64 = int64(i)
	case int64:
		i64 = int64(i)
	case int:
		i64 = int64(i)
	default:

		return Timestamp_Invalid, errVIOLATIONUnexpectedType
	}

	return timestampFromMillisecondsInt64(i64)
}

// [PRIVATE] Attempts to obtain a Timestamp from the given raw number of
// milliseconds since epoch.
//
// Note:
// This function MUST NOT be called with any type other than `uint`,
// `uint32`, `uint64`. The larger list of types is purely to allow calling
// from TimestampFromMilliseconds().
func timestampFromMillisecondsUnsigned[T uint32 | uint64 | uint | int32 | int64 | int](
	v T,
) (Timestamp, error) {

	var u64 uint64

	switch u := any(v).(type) {
	case uint32:
		u64 = uint64(u)
	case uint64:
		u64 = uint64(u)
	case uint:
		u64 = uint64(u)
	default:

		return Timestamp_Invalid, errVIOLATIONUnexpectedType
	}

	return timestampFromMillisecondsUint64(u64)
}

// ===========================
// API functions
// ===========================

// Attempts to convert a number of milliseconds into an API Timestamp.
func TimestampFromMilliseconds[T int32 | int64 | int | uint32 | uint64 | uint](
	v T,
) (Timestamp, error) {

	switch any(v).(type) {
	case int32, int64, int:
		return timestampFromMillisecondsSigned(v)
	case uint32, uint64, uint:
		return timestampFromMillisecondsUnsigned(v)
	}

	return Timestamp_Invalid, errVIOLATIONUnexpectedType
}

// Attempts to convert a time.Time into an API Timestamp.
func TimestampFromTimeTime(
	v *time.Time,
) (Timestamp, error) {

	if v == nil {
		return Timestamp_Invalid, errTimestampNil
	}

	return timestampFromMillisecondsInt64(v.UnixMilli())
}

// Attempts to convert a timestamppb.Timestamp into an API Timestamp.
func TimestampFromTimestampPB(
	v *timestamppb.Timestamp,
) (Timestamp, error) {

	if v == nil {
		return Timestamp_Invalid, errTimestampNil
	}

	t := v.AsTime()

	return timestampFromMillisecondsInt64(t.UnixMilli())
}

// Attempts to convert a timestamppb.Timestamp into an API Timestamp,
// obtaining the sentinel value Timestamp_zero upon failure.
func TimestampFromTimestampPBOrZero(
	v *timestamppb.Timestamp,
) Timestamp {

	if r, err := TimestampFromTimestampPB(v); err != nil {
		return Timestamp_Zero
	} else {
		return r
	}
}

// Attempts to convert a timestamppb.Timestamp into a pointer to an API
// Timestamp, obtaining nil upon failure.
func TimestampPtrFromTimestampPBOrNil(
	v *timestamppb.Timestamp,
) *Timestamp {
	if v == nil {
		return nil
	}

	if r, err := TimestampFromTimestampPB(v); err != nil {
		return nil
	} else {
		return &r
	}
}

// Attempts to convert a Timestamp into a pointer to a
// timestamppb.Timestamp.
func TimestampToTimestampPB(
	ts Timestamp,
) (*timestamppb.Timestamp, error) {
	i_ms, isValid := ts._internal_IsValid()

	if !isValid {
		return nil, errTimestampInvalidValue
	} else {
		t := time.UnixMilli(i_ms)

		return timestamppb.New(t), nil
	}
}

// Attempts to convert a Timestamp into a pointer to a
// timestamppb.Timestamp obtaining nil upon failure.
func TimestampToTimestampPBOrNil(
	ts Timestamp,
) *timestamppb.Timestamp {
	i_ms, isValid := ts._internal_IsValid()

	if !isValid {
		return nil
	} else {
		t := time.UnixMilli(i_ms)

		return timestamppb.New(t)
	}
}

// ===========================
// Methods
// ===========================

// Subtracts one Timestamp from the receiving instance, obtaining a
// time.Duration value.
func (ts Timestamp) Sub(rhs Timestamp) time.Duration {
	lhs_ms := ts._internalCount_ms()
	rhs_ms := rhs._internalCount_ms()

	ms := lhs_ms - rhs_ms

	return time.Millisecond * time.Duration(ms)
}

// Attempts to subtract a time.Durationm from the receive instance,
// obtaining a Timestamp if successful.
func (ts Timestamp) SubDuration(rhs time.Duration) (Timestamp, error) {
	lhs_ms := ts._internalCount_ms()
	rhs_ms := rhs.Milliseconds()

	ms := lhs_ms - rhs_ms

	if ms < 0 {
		return Timestamp_Invalid, errTimestampValueOutOfRange
	} else {
		return Timestamp(ms), nil
	}
}
