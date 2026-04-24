package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Timestamp(t *testing.T) {

	t.Run("TimestampNow()", func(t *testing.T) {

		t1 := time.Now().UTC()
		t2 := TimestampNow()
		t3 := time.Now().UTC()

		ms1 := t1.UnixMilli()
		ms2 := t2.Milliseconds()
		ms3 := t3.UnixMilli()

		assert.LessOrEqual(t, ms1, ms2)
		assert.LessOrEqual(t, ms2, ms3)

		us1 := t1.UnixMicro()
		us2 := t2.Microseconds()
		us3 := t3.UnixMicro()

		assert.LessOrEqual(t, us1, us2)
		assert.LessOrEqual(t, us2, us3)
	})

	t.Run("TimestampDate()", func(t *testing.T) {

		// epoch + 0
		{
			ts, err := TimestampDate(1970, time.January, 1, 0, 0, 0, 0)

			require.Nil(t, err)

			assert.Equal(t, Timestamp_Zero, ts)
		}

		// epoch + 1ns
		{
			ts, err := TimestampDate(1970, time.January, 1, 0, 0, 0, 1)

			require.Nil(t, err)

			assert.Equal(t, Timestamp_Zero, ts) // it's 0 because Timestamp internal repr is µs
		}

		// epoch + 1µs
		{
			ts, err := TimestampDate(1970, time.January, 1, 0, 0, 0, 1_000)

			require.Nil(t, err)

			assert.Equal(t, int64(1), ts.Microseconds())
		}

		// epoch + 1ms
		{
			ts, err := TimestampDate(1970, time.January, 1, 0, 0, 0, 1_000_000)

			require.Nil(t, err)

			assert.Equal(t, int64(1_000), ts.Microseconds())
		}

		// epoch + 1ns
		{
			ts, err := TimestampDate(2025, time.November, 12, 10, 48, 27, 123_456_789)

			require.Nil(t, err)

			assert.Equal(t, int64(1_762_944_507_123), ts.Milliseconds())
			assert.Equal(t, int64(1_762_944_507_123_456), ts.Microseconds())
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

				assert.Equal(t, "1762944507123456", s)
			}

			// unmarshal
			{
				var ts Timestamp

				err := json.Unmarshal([]byte(s), &ts)

				require.Nil(t, err)

				assert.Equal(t, int64(1762944507123456), ts.Microseconds())
			}
		}
	})
}
