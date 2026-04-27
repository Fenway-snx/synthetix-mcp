package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_NowPlusDuration(t *testing.T) {

	{
		now := time.Now()
		t2 := NowPlusDuration(0 * time.Second)

		duration_us := t2.Sub(now).Microseconds()

		assert.LessOrEqual(t, int64(0), duration_us)
		assert.Greater(t, int64(1_000), duration_us)
	}
}

func Test_RandomDurationInRange(t *testing.T) {

	{
		min := 0 * time.Second
		exclusiveMax := 1 * time.Second

		r := RandomDurationInRange(min, exclusiveMax)

		assert.LessOrEqual(t, min, r)
		assert.Greater(t, exclusiveMax, r)
	}

	{
		min := 123 * time.Millisecond
		exclusiveMax := 125 * time.Millisecond

		r := RandomDurationInRange(min, exclusiveMax)

		assert.LessOrEqual(t, min, r)
		assert.Greater(t, exclusiveMax, r)

		var found123 bool
		var found124 bool

		for i := 0; i < 1_000_000; i++ {

			ms := RandomDurationInRange(min, exclusiveMax).Milliseconds()

			switch ms {
			case 123:
				found123 = true
			case 124:
				found124 = true
			}

			if found123 && found124 {
				break
			}
		}

		assert.True(t, found123)
		assert.True(t, found124)
	}
}
