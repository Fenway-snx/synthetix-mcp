package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// FixedTimeProvider

func Test_NewFixedTimeProvider(t *testing.T) {
	tm0 := time.Now() // NOTE: deliberately using std lib

	tp := NewFixedTimeProvider(tm0)

	tm1 := tp.Now()

	time.Sleep(time.Millisecond)

	tm2 := tp.Now()

	time.Sleep(time.Millisecond)

	tm3 := tp.Now()

	assert.Equal(t, tm0, tm1)
	assert.Equal(t, tm0, tm2)
	assert.Equal(t, tm0, tm3)

	for i := 0; i != 1_000; i++ {

		tmN := tp.Now()

		assert.Equal(t, tm0, tmN)
	}
}

// RealTimeProvider

func Test_NewRealTimeProvider(t *testing.T) {
	tp := NewRealTimeProvider()

	for i := 0; i != 1_000; i++ {

		tmBefore := time.Now()
		tm := tp.Now()
		tmAfter := time.Now()

		tmBefore = tmBefore.UTC()
		tmAfter = tmAfter.UTC()

		assert.True(t, TimeWithinRangeInclusiveEnd(tm, tmBefore, tmAfter), "expected time %v to be within the range [%v, %v]", tm, tmBefore, tmAfter)
	}
}

// ProgrammedTimeProviderBuilder

func Test_ProgrammedTimeProviderBuilder_1(t *testing.T) {

	ptpb := ProgrammedTimeProviderBuilder{}

	tm0 := time.Date(2026, 1, 1, 15, 46, 41, 123456789, time.UTC)

	ptpb.Push(
		NewFixedTimeProvider(tm0),
		1,
	)

	tp, pc := ptpb.Build()

	assert.Equal(t, int64(1), pc)

	tm1 := tp.Now()

	assert.Equal(t, tm0, tm1)
}

func Test_ProgrammedTimeProviderBuilder_2(t *testing.T) {

	ptpb := ProgrammedTimeProviderBuilder{}

	tm0 := time.Date(2026, 1, 1, 15, 46, 41, 123456789, time.UTC)

	ptpb.Push(
		NewFixedTimeProvider(tm0),
		1,
	)

	ptpb.Push(
		NewFixedTimeProvider(tm0),
		2,
	)

	tp, pc := ptpb.Build()

	assert.Equal(t, int64(3), pc)

	tm1 := tp.Now()

	assert.Equal(t, tm0, tm1)

	tm2 := tp.Now()

	assert.Equal(t, tm0, tm2)

	tm3 := tp.Now()

	assert.Equal(t, tm0, tm3)
}

// FixedIncrementTimeProvider

func Test_NewFixedIncrementTimeProvider(t *testing.T) {

	tm0 := time.Now()

	tp := NewFixedIncrementTimeProvider(tm0, time.Millisecond*3)

	tm1 := tp.Now()

	d1 := tm1.Sub(tm0)

	assert.Equal(t, 3*time.Millisecond*1, d1)

	tm2 := tp.Now()

	d2 := tm2.Sub(tm0)

	assert.Equal(t, 3*time.Millisecond*2, d2)

	tm3 := tp.Now()

	d3 := tm3.Sub(tm0)

	assert.Equal(t, 3*time.Millisecond*3, d3)
}

// RandomTimeProvider

func Test_NewRandomTimeProvider(t *testing.T) {

	{
		tm0 := time.Date(2026, 1, 1, 17, 8, 43, 123456789, time.UTC)

		dFrom := -100 * time.Microsecond
		dTo := 10 * time.Millisecond

		tmBefore := tm0.Add(dFrom)
		tmAfter := tm0.Add(dTo)

		tp := NewRandomTimeProvider(tm0, dFrom, dTo)

		for i := 0; i != 1_000; i++ {
			tm := tp.Now()

			assert.True(t, TimeWithinRangeExclusiveEnd(tm, tmBefore, tmAfter), "expected time %v to be within the range [%v, %v)", tm, tmBefore, tmAfter)
		}
	}
}
