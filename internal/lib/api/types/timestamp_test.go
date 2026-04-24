package types

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_utils_test "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/test"
)

func Test_TimeStamp(t *testing.T) {

	t.Run("timestampFromMillisecondsInt64()", func(t *testing.T) {

		tests := []struct {
			name     string
			value    int64
			isValid  bool
			expected Timestamp
			contains string
		}{
			{
				value:    0,
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				value:    1,
				isValid:  true,
				expected: Timestamp(1),
			},
			{
				value:    1762316001000,
				isValid:  true,
				expected: Timestamp(1762316001000),
			},
			{
				value:    4102444799999,
				isValid:  true,
				expected: Timestamp(4102444799999),
			},
			{
				value:    4_102_444_800_000,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				value:    1762292158160494000,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				value:    -1,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
			{
				value:    -2,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
		}

		for _, tt := range tests {

			name := tt.name
			if name == "" {
				name = fmt.Sprintf("%d", tt.value)
			}

			t.Run(name, func(t *testing.T) {

				r, err := timestampFromMillisecondsInt64(tt.value)

				if tt.isValid {

					assert.Nil(t, err)

					assert.Equal(t, tt.expected, r)
				} else {

					require.NotNil(t, err, "expected `err` to be not `nil`")
					assert.Contains(t, err.Error(), tt.contains)

					assert.Equal(t, tt.expected, r)
				}
			})
		}
	})

	t.Run("timestampFromMillisecondsUint64()", func(t *testing.T) {

		tests := []struct {
			name     string
			value    uint64
			isValid  bool
			expected Timestamp
			contains string
		}{
			{
				value:    0,
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				value:    1,
				isValid:  true,
				expected: Timestamp(1),
			},
			{
				value:    1762316001000,
				isValid:  true,
				expected: Timestamp(1762316001000),
			},
			{
				value:    4102444799999,
				isValid:  true,
				expected: Timestamp(4102444799999),
			},
			{
				value:    4_102_444_800_000,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				value:    9223372036854775807,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				value:    9223372036854775808,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
		}

		for _, tt := range tests {

			name := tt.name
			if name == "" {
				name = fmt.Sprintf("%d", tt.value)
			}

			t.Run(name, func(t *testing.T) {

				r, err := timestampFromMillisecondsUint64(tt.value)

				if tt.isValid {

					assert.Nil(t, err)

					assert.Equal(t, tt.expected, r)
				} else {

					require.NotNil(t, err, "expected `err` to be not `nil`")
					assert.Contains(t, err.Error(), tt.contains)

					assert.Equal(t, tt.expected, r)
				}
			})
		}
	})

	t.Run("TimestampFromMilliseconds()", func(t *testing.T) {

		tests := []struct {
			name     string
			value    int64
			uvalue   *uint64
			isValid  bool
			expected Timestamp
			contains string
		}{
			{
				value:    0,
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				uvalue:   snx_lib_utils_test.MakePointerOf(uint64(0)),
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				value:    1,
				isValid:  true,
				expected: Timestamp(1),
			},
			{
				uvalue:   snx_lib_utils_test.MakePointerOf(uint64(1)),
				isValid:  true,
				expected: Timestamp(1),
			},
			{
				value:    1762316001000,
				isValid:  true,
				expected: Timestamp(1762316001000),
			},
			{
				uvalue:   snx_lib_utils_test.MakePointerOf(uint64(1762316001000)),
				isValid:  true,
				expected: Timestamp(1762316001000),
			},
			{
				value:    4102444799999,
				isValid:  true,
				expected: Timestamp(4102444799999),
			},
			{
				uvalue:   snx_lib_utils_test.MakePointerOf(uint64(4102444799999)),
				isValid:  true,
				expected: Timestamp(4102444799999),
			},
			{
				value:    4_102_444_800_000,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				uvalue:   snx_lib_utils_test.MakePointerOf(uint64(4_102_444_800_000)),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				value:    9223372036854775807,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				uvalue:   snx_lib_utils_test.MakePointerOf(uint64(9223372036854775807)),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				uvalue:   snx_lib_utils_test.MakePointerOf(uint64(9223372036854775808)),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
			{
				value:    -1,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
			{
				value:    -2,
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
		}

		for _, tt := range tests {

			name := tt.name
			if name == "" {
				name = fmt.Sprintf("%d", tt.value)
			}

			t.Run(name, func(t *testing.T) {

				var r Timestamp
				var err error

				if tt.uvalue != nil {

					r, err = TimestampFromMilliseconds(*tt.uvalue)
				} else {

					r, err = TimestampFromMilliseconds(tt.value)
				}

				if tt.isValid {

					assert.Nil(t, err)

					assert.Equal(t, tt.expected, r)
				} else {

					require.NotNil(t, err, "expected `err` to be not `nil`")
					assert.Contains(t, err.Error(), tt.contains)

					assert.Equal(t, tt.expected, r)
				}
			})
		}
	})

	t.Run("TimestampFromTimeTime()", func(t *testing.T) {

		tests := []struct {
			name     string
			value    time.Time
			isValid  bool
			expected Timestamp
			contains string
		}{
			{
				name:     "0.0",
				value:    time.Unix(0, 0),
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				name:     "0.000000001",
				value:    time.Unix(0, 1),
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				name:     "0.001",
				value:    time.Unix(0, 1_000_000),
				isValid:  true,
				expected: Timestamp(1),
			},
			{
				name:     "Wednesday, 5 November 2025 04:13:21",
				value:    time.Unix(1_762_316_001, 0),
				isValid:  true,
				expected: Timestamp(1_762_316_001_000),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    time.Unix(4_102_444_799, 999_000_000),
				isValid:  true,
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    time.Unix(4_102_444_799, 999_999_999),
				isValid:  true,
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:     "Friday, 1 January 2100 00:00:00",
				value:    time.Unix(4_102_444_800, 0),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				name:     "int64.MAX",
				value:    time.Unix(9_223_372_036_854_775, 807_000_000),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				name:     "-1.0",
				value:    time.Unix(-1, 0),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
			{
				name:     "-2.0",
				value:    time.Unix(-2, 0),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
		}

		for _, tt := range tests {

			name := tt.name
			if name == "" {
				name = fmt.Sprintf("%v", tt.value)
			}

			t.Run(name, func(t *testing.T) {

				var r Timestamp
				var err error

				r, err = TimestampFromTimeTime(&tt.value)

				if tt.isValid {

					require.Nil(t, err, "expected `err` to be `nil`, but instead got '%[1]v' (%[1]T)", err)

					assert.Equal(t, tt.expected, r)
				} else {

					require.NotNil(t, err, "expected `err` to be not `nil`")
					assert.Contains(t, err.Error(), tt.contains)

					assert.Equal(t, tt.expected, r)
				}
			})
		}
	})

	t.Run("TimestampFromTimestampPB()", func(t *testing.T) {

		tests := []struct {
			name     string
			value    *timestamppb.Timestamp
			isValid  bool
			expected Timestamp
			contains string
		}{
			{
				name:     "0.0",
				value:    timestamppb.New(time.Unix(0, 0)),
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				name:     "0.000000001",
				value:    timestamppb.New(time.Unix(0, 1)),
				isValid:  true,
				expected: Timestamp_Zero,
			},
			{
				name:     "0.001",
				value:    timestamppb.New(time.Unix(0, 1_000_000)),
				isValid:  true,
				expected: Timestamp(1),
			},
			{
				name:     "Wednesday, 5 November 2025 04:13:21",
				value:    timestamppb.New(time.Unix(1_762_316_001, 0)),
				isValid:  true,
				expected: Timestamp(1_762_316_001_000),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    timestamppb.New(time.Unix(4_102_444_799, 999_000_000)),
				isValid:  true,
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    timestamppb.New(time.Unix(4_102_444_799, 999_999_999)),
				isValid:  true,
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:     "Friday, 1 January 2100 00:00:00",
				value:    timestamppb.New(time.Unix(4_102_444_800, 0)),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				name:     "int64.MAX",
				value:    timestamppb.New(time.Unix(9_223_372_036_854_775, 807_000_000)),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp value out of range",
			},
			{
				name:     "-1.0",
				value:    timestamppb.New(time.Unix(-1, 0)),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
			{
				name:     "-2.0",
				value:    timestamppb.New(time.Unix(-2, 0)),
				isValid:  false,
				expected: Timestamp_Invalid,
				contains: "timestamp invalid value",
			},
		}

		for _, tt := range tests {

			name := tt.name
			if name == "" {
				name = fmt.Sprintf("%v", tt.value)
			}

			t.Run(name, func(t *testing.T) {

				var r Timestamp
				var err error

				r, err = TimestampFromTimestampPB(tt.value)

				if tt.isValid {

					require.Nil(t, err, "expected `err` to be `nil`, but instead got '%[1]v' (%[1]T)", err)

					assert.Equal(t, tt.expected, r)
				} else {

					require.NotNil(t, err, "expected `err` to be not `nil`")
					assert.Contains(t, err.Error(), tt.contains)

					assert.Equal(t, tt.expected, r)
				}
			})
		}
	})

	t.Run("TimestampFromTimestampPBOrZero()", func(t *testing.T) {

		tests := []struct {
			name     string
			value    *timestamppb.Timestamp
			expected Timestamp
		}{
			{
				name:     "0.0",
				value:    timestamppb.New(time.Unix(0, 0)),
				expected: Timestamp_Zero,
			},
			{
				name:     "0.000000001",
				value:    timestamppb.New(time.Unix(0, 1)),
				expected: Timestamp_Zero,
			},
			{
				name:     "0.001",
				value:    timestamppb.New(time.Unix(0, 1_000_000)),
				expected: Timestamp(1),
			},
			{
				name:     "Wednesday, 5 November 2025 04:13:21",
				value:    timestamppb.New(time.Unix(1_762_316_001, 0)),
				expected: Timestamp(1_762_316_001_000),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    timestamppb.New(time.Unix(4_102_444_799, 999_000_000)),
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    timestamppb.New(time.Unix(4_102_444_799, 999_999_999)),
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:     "Friday, 1 January 2100 00:00:00",
				value:    timestamppb.New(time.Unix(4_102_444_800, 0)),
				expected: Timestamp_Zero,
			},
			{
				name:     "int64.MAX",
				value:    timestamppb.New(time.Unix(9_223_372_036_854_775, 807_000_000)),
				expected: Timestamp_Zero,
			},
			{
				name:     "-1.0",
				value:    timestamppb.New(time.Unix(-1, 0)),
				expected: Timestamp_Zero,
			},
			{
				name:     "-2.0",
				value:    timestamppb.New(time.Unix(-2, 0)),
				expected: Timestamp_Zero,
			},
		}

		for _, tt := range tests {

			name := tt.name
			if name == "" {
				name = fmt.Sprintf("%v", tt.value)
			}

			t.Run(name, func(t *testing.T) {

				r := TimestampFromTimestampPBOrZero(tt.value)

				assert.Equal(t, tt.expected, r)
			})
		}
	})

	t.Run("TimestampPtrFromTimestampPBOrNil()", func(t *testing.T) {

		tests := []struct {
			name     string
			value    *timestamppb.Timestamp
			isNil    bool
			expected Timestamp
		}{
			{
				name:     "0.0",
				value:    timestamppb.New(time.Unix(0, 0)),
				expected: Timestamp_Zero,
			},
			{
				name:     "0.000000001",
				value:    timestamppb.New(time.Unix(0, 1)),
				expected: Timestamp_Zero,
			},
			{
				name:     "0.001",
				value:    timestamppb.New(time.Unix(0, 1_000_000)),
				expected: Timestamp(1),
			},
			{
				name:     "Wednesday, 5 November 2025 04:13:21",
				value:    timestamppb.New(time.Unix(1_762_316_001, 0)),
				expected: Timestamp(1_762_316_001_000),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    timestamppb.New(time.Unix(4_102_444_799, 999_000_000)),
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:     "Thursday, 31 December 2099 23:59:59.999",
				value:    timestamppb.New(time.Unix(4_102_444_799, 999_999_999)),
				expected: Timestamp(4_102_444_799_999),
			},
			{
				name:  "Friday, 1 January 2100 00:00:00",
				value: timestamppb.New(time.Unix(4_102_444_800, 0)),
				isNil: true,
			},
			{
				name:  "int64.MAX",
				value: timestamppb.New(time.Unix(9_223_372_036_854_775, 807_000_000)),
				isNil: true,
			},
			{
				name:  "-1.0",
				value: timestamppb.New(time.Unix(-1, 0)),
				isNil: true,
			},
			{
				name:  "-2.0",
				value: timestamppb.New(time.Unix(-2, 0)),
				isNil: true,
			},
		}

		for _, tt := range tests {

			name := tt.name
			if name == "" {
				name = fmt.Sprintf("%v", tt.value)
			}

			t.Run(name, func(t *testing.T) {

				r := TimestampPtrFromTimestampPBOrNil(tt.value)

				if tt.isNil {

					assert.Nil(t, r)
				} else {

					require.NotNil(t, r, "expected `r` to be not `nil`")

					assert.Equal(t, tt.expected, *r)
				}
			})
		}
	})

	t.Run("TimestampToTimestampPB()", func(t *testing.T) {

		// Timestamp_Invalid
		{
			tb, err := TimestampToTimestampPB(Timestamp_Invalid)

			require.NotNil(t, err, "expected `err` to not be `nil`")

			assert.Nil(t, tb)
		}

		// Timestamp_Zero
		{
			tb, err := TimestampToTimestampPB(Timestamp_Zero)

			require.Nil(t, err, "expected `err` to be `nil`, but instead received '%[1]v' (of type %[1]T)", err)

			assert.NotNil(t, tb)

			tm := tb.AsTime()

			assert.Equal(t, int64(0), tm.UnixMilli())
		}

		// specific valid date
		{
			ts, _ := TimestampDate(2025, time.November, 12, 19, 13, 24, 123_456_789)

			tb, err := TimestampToTimestampPB(ts)

			require.Nil(t, err, "expected `err` to be `nil`, but instead received '%[1]v' (of type %[1]T)", err)

			require.NotNil(t, tb)

			tm := tb.AsTime()

			assert.Equal(t, int64(1_762_974_804_123), tm.UnixMilli())
		}

		// specific invalid date
		{
			tm := time.Date(2125, time.January, 1, 0, 0, 0, 0, time.UTC)

			// direct conversions are now allowed - this only for testing
			ts := Timestamp(tm.UnixMilli())

			tb, err := TimestampToTimestampPB(ts)

			require.NotNil(t, err, "expected `err` to not be `nil`")

			assert.Nil(t, tb)
		}
	})

	t.Run("TimestampToTimestampPBOrNil()", func(t *testing.T) {

		// Timestamp_Invalid
		{
			tb := TimestampToTimestampPBOrNil(Timestamp_Invalid)

			assert.Nil(t, tb)
		}

		// Timestamp_Zero
		{
			tb := TimestampToTimestampPBOrNil(Timestamp_Zero)

			require.NotNil(t, tb)

			tm := tb.AsTime()

			assert.Equal(t, int64(0), tm.UnixMilli())
		}

		// specific valid date
		{
			ts, _ := TimestampDate(2025, time.November, 12, 19, 13, 24, 123_456_789)

			tb := TimestampToTimestampPBOrNil(ts)

			require.NotNil(t, tb)

			tm := tb.AsTime()

			assert.Equal(t, int64(1_762_974_804_123), tm.UnixMilli())
		}

		// specific invalid date
		{
			tm := time.Date(2125, time.January, 1, 0, 0, 0, 0, time.UTC)

			// direct conversions are now allowed - this only for testing
			ts := Timestamp(tm.UnixMilli())

			tb := TimestampToTimestampPBOrNil(ts)

			assert.Nil(t, tb)
		}
	})

	t.Run("TimestampNow()", func(t *testing.T) {

		t1 := time.Now()
		t2 := TimestampNow()
		t3 := time.Now()

		ms1 := t1.UnixMilli()
		ms2 := int64(t2)
		ms3 := t3.UnixMilli()

		assert.LessOrEqual(t, ms1, ms2)
		assert.LessOrEqual(t, ms2, ms3)
	})

	t.Run("TimestampDate()", func(t *testing.T) {

		{
			ts, err := TimestampDate(2025, time.November, 12, 10, 48, 27, 123_456_789)

			require.Nil(t, err)

			assert.Equal(t, Timestamp(1_762_944_507_123), ts)
		}

		{
			ts, err := TimestampDate(1970, time.January, 1, 0, 0, 0, 1_000_000)

			require.Nil(t, err)

			assert.Equal(t, Timestamp(1), ts)
		}

		{
			ts, err := TimestampDate(1970, time.January, 1, 0, 0, 0, 0)

			require.Nil(t, err)

			assert.Equal(t, Timestamp_Zero, ts)
		}
	})

	t.Run("marshaling", func(t *testing.T) {

		// marshal / unmarshal valid date
		{
			var s string

			// marshal
			{
				ts, _ := TimestampDate(2025, time.November, 12, 10, 48, 27, 123_456_789)

				bytes, err := json.Marshal(ts)

				require.Nil(t, err)

				s = string(bytes)

				assert.Equal(t, "1762944507123", s)
			}

			// unmarshal
			{
				var ts Timestamp

				err := json.Unmarshal([]byte(s), &ts)

				require.Nil(t, err)

				assert.Equal(t, Timestamp(1762944507123), ts)
			}

			// unmarshal from specific JSON int
			{
				input := "1762944507123"

				var ts Timestamp

				err := json.Unmarshal([]byte(input), &ts)

				require.Nil(t, err)

				assert.Equal(t, Timestamp(1762944507123), ts)
			}

			// unmarshal from specific JSON string
			{
				input := `"1762944507123"`

				var ts Timestamp

				err := json.Unmarshal([]byte(input), &ts)

				require.Nil(t, err)

				assert.Equal(t, Timestamp(1762944507123), ts)
			}

			// unmarshal from specific JSON string containing embedded underscores
			{
				input := `"1_762_944_507_123"`

				var ts Timestamp

				err := json.Unmarshal([]byte(input), &ts)

				require.Nil(t, err)

				assert.Equal(t, Timestamp(1762944507123), ts)
			}
		}
	})

	t.Run("Nanoseconds(), Microseconds(), Milliseconds()", func(t *testing.T) {

		tests := []struct {
			value        Timestamp
			nanoseconds  int64
			microseconds int64
			milliseconds int64
			seconds      int64
		}{
			{
				value:        Timestamp_Zero,
				nanoseconds:  0,
				microseconds: 0,
				milliseconds: 0,
				seconds:      0,
			},
			{
				value:        Timestamp(123),
				nanoseconds:  123_000_000,
				microseconds: 123_000,
				milliseconds: 123,
				seconds:      0,
			},
			{
				value:        Timestamp(1_762_944_507_123),
				nanoseconds:  1_762_944_507_123_000_000,
				microseconds: 1_762_944_507_123_000,
				milliseconds: 1_762_944_507_123,
				seconds:      1_762_944_507,
			},
			{
				value:        Timestamp_Invalid,
				nanoseconds:  0,
				microseconds: 0,
				milliseconds: 0,
				seconds:      0,
			},
		}

		for _, tt := range tests {

			assert.Equal(t, tt.nanoseconds, tt.value.Nanoseconds())
			assert.Equal(t, tt.microseconds, tt.value.Microseconds())
			assert.Equal(t, tt.milliseconds, tt.value.Milliseconds())
		}
	})

	t.Run("Sub()", func(t *testing.T) {

		// substract self
		{
			ts, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			d := ts.Sub(ts)

			assert.Equal(t, time.Duration(0), d)

			// rhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)
		}

		// substract timestamp 1ns earlier
		{
			lhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 1)
			rhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			d := lhs.Sub(rhs)

			assert.Equal(t, 0*time.Nanosecond, d) // it's 0 because Timestamp internal repr is ms
		}

		// substract timestamp 1µs earlier
		{
			lhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 1_000)
			rhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			d := lhs.Sub(rhs)

			assert.Equal(t, 0*time.Microsecond, d) // it's 0 because Timestamp internal repr is ms
		}

		// substract timestamp 1ms earlier
		{
			lhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 1_000_000)
			rhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			d := lhs.Sub(rhs)

			assert.Equal(t, 1*time.Millisecond, d)
		}

		// substract timestamp 1s earlier
		{
			lhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)
			rhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 36, 0)

			d := lhs.Sub(rhs)

			assert.Equal(t, 1*time.Second, d)
		}

		// subtract Timestamp_zero
		{
			lhs, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			d := lhs.Sub(Timestamp_Zero)

			assert.Equal(t, time.Duration(lhs.Milliseconds())*time.Millisecond, d)
		}
	})

	t.Run("SubDuration()", func(t *testing.T) {

		// substract 0
		{
			ts1, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			ts2, _ := ts1.SubDuration(0 * time.Second)

			assert.Equal(t, ts1, ts2)
		}

		// substract 1ns
		{
			ts1, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			ts2, _ := ts1.SubDuration(1 * time.Nanosecond)

			expected := ts1 // it's same because Timestamp internal repr is ms

			assert.Equal(t, expected, ts2)
		}

		// substract 1µs
		{
			ts1, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			ts2, _ := ts1.SubDuration(1 * time.Microsecond)

			expected := ts1 // it's same because Timestamp internal repr is ms

			assert.Equal(t, expected, ts2)
		}

		// substract 1ms
		{
			ts1, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			ts2, _ := ts1.SubDuration(1 * time.Millisecond)

			expected, _ := TimestampDate(2024, time.November, 12, 19, 55, 36, 999_000_000)

			assert.Equal(t, expected, ts2)
		}

		// substract 1s
		{
			ts1, _ := TimestampDate(2024, time.November, 12, 19, 55, 37, 0)

			ts2, _ := ts1.SubDuration(1 * time.Second)

			expected, _ := TimestampDate(2024, time.November, 12, 19, 55, 36, 0)

			assert.Equal(t, expected, ts2)
		}

	})
}
