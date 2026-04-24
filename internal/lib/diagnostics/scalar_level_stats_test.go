package diagnostics

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ScalarLevelStats(t *testing.T) {

	t.Run("`#Set()` (single-threaded)", func(t *testing.T) {

		var sls ScalarLevelStats

		{
			max, current := sls.Load()

			assert.Equal(t, int64(0), max)
			assert.Equal(t, int64(0), current)
		}

		sls.Set(1)

		{
			max, current := sls.Load()

			assert.Equal(t, int64(1), max)
			assert.Equal(t, int64(1), current)
		}

		sls.Set(0)

		{
			max, current := sls.Load()

			assert.Equal(t, int64(1), max)
			assert.Equal(t, int64(0), current)
		}

		sls.Set(999_999_999)

		{
			max, current := sls.Load()

			assert.Equal(t, int64(999_999_999), max)
			assert.Equal(t, int64(999_999_999), current)
		}

		sls.Set(999_999)

		{
			max, current := sls.Load()

			assert.Equal(t, int64(999_999_999), max)
			assert.Equal(t, int64(999_999), current)
		}
	})

	t.Run("`#Set()` (multi-threaded)", func(t *testing.T) {

		var sls ScalarLevelStats

		var wg sync.WaitGroup

		for i := 1_000; i != 1_100; i++ {
			id := int64(i)
			wg.Go(func() {

				sls.Set(id)

				max, current := sls.Load()

				// assert.LessOrEqual(t, current, max) // NOTE: cannot assert this because _possible_ it will fail
				assert.True(t, max >= 1_000 && max <= 1_100)
				assert.True(t, current >= 1_000 && current <= 1_100)
			})
		}

		wg.Wait()
	})

	t.Run("`#Inc()`/`#Dec()` (single-threaded)", func(t *testing.T) {

		var sls ScalarLevelStats

		{
			max, current := sls.Load()

			assert.Equal(t, int64(0), max)
			assert.Equal(t, int64(0), current)
		}

		sls.Inc()

		{
			max, current := sls.Load()

			assert.Equal(t, int64(1), max)
			assert.Equal(t, int64(1), current)
		}

		sls.Dec()

		{
			max, current := sls.Load()

			assert.Equal(t, int64(1), max)
			assert.Equal(t, int64(0), current)
		}

		for i := 0; i != 10_000; i++ {

			sls.Inc()

			max, current := sls.Load()

			assert.Equal(t, int64(1+i), max)
			assert.Equal(t, int64(1+i), current)
		}
	})

	t.Run("`#Inc()`/`#Dec()` (multi-threaded)", func(t *testing.T) {

		// up, then down
		{
			var sls ScalarLevelStats

			var wg sync.WaitGroup

			for i := 0; i != 100; i++ {

				wg.Go(func() {

					for j := 0; j != 100_000; j++ {

						sls.Inc()
					}
				})
			}

			wg.Wait()

			{
				max, current := sls.Load()

				assert.Equal(t, int64(10_000_000), max)
				assert.Equal(t, int64(10_000_000), current)
			}

			for i := 0; i != 100; i++ {

				wg.Go(func() {

					for j := 0; j != 100_000; j++ {

						sls.Dec()
					}
				})
			}

			wg.Wait()

			{
				max, current := sls.Load()

				assert.Equal(t, int64(10_000_000), max)
				assert.Equal(t, int64(0), current)
			}
		}

		// up and down x N
		{

			var sls ScalarLevelStats

			var wg sync.WaitGroup

			for i := 0; i != 100; i++ {

				wg.Go(func() {

					for j := 0; j != 100_000; j++ {

						if j%2 == 1 {

							sls.Dec()
						} else {

							sls.Inc()
						}
					}
				})
			}

			wg.Wait()

			{
				max, current := sls.Load()

				assert.LessOrEqual(t, max, int64(5_000_000))
				assert.Equal(t, int64(0), current)
			}
		}
	})
}
