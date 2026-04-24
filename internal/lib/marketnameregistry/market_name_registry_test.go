package marketnameregistry

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MarketNameRegistry_New_CopyInitial(t *testing.T) {
	initial := []string{"a", "b"}
	r := New(initial)
	initial[0] = "z"
	assert.Equal(t, []string{"a", "b"}, r.Snapshot())
}

func Test_MarketNameRegistry_New_DeduplicatesInitialDuplicates(t *testing.T) {
	r := New([]string{"b", "a", "b", "a"})
	require.Equal(t, 2, r.len())
	assert.Equal(t, []string{"a", "b"}, r.Snapshot())
}

func Test_MarketNameRegistry_AddRemoveContainsSnapshotlen(t *testing.T) {
	r := New(nil)
	require.Equal(t, 0, r.len())
	require.False(t, r.Contains("x"))

	r.Add("x")
	require.Equal(t, 1, r.len())
	require.True(t, r.Contains("x"))
	assert.Equal(t, []string{"x"}, r.Snapshot())

	require.True(t, r.Remove("x"))
	require.False(t, r.Remove("x"))
	require.Equal(t, 0, r.len())
}

func Test_MarketNameRegistry_DuplicateAdd_SetSemantics(t *testing.T) {
	r := New(nil)
	r.Add("x")
	r.Add("x")
	require.Equal(t, 1, r.len())
	require.Equal(t, []string{"x"}, r.Snapshot())
}

func Test_MarketNameRegistry_Snapshot_SortedOrder(t *testing.T) {
	r := New([]string{"zebra", "apple", "middle"})
	assert.Equal(t, []string{"apple", "middle", "zebra"}, r.Snapshot())
}

func Test_MarketNameRegistry_Snapshot_EmptyReturnsNil(t *testing.T) {
	r := New(nil)
	assert.Nil(t, r.Snapshot())

	r = New([]string{})
	assert.Nil(t, r.Snapshot())
}

func Test_MarketNameRegistry_Snapshot_IndependentCopy(t *testing.T) {
	r := New([]string{"a"})
	snap := r.Snapshot()
	require.Len(t, snap, 1)
	snap[0] = "mutated"
	assert.Equal(t, []string{"a"}, r.Snapshot())
}

func Test_MarketNameRegistry_ZeroValue(t *testing.T) {
	var r MarketNameRegistry
	require.Equal(t, 0, r.len())
	require.False(t, r.Contains("x"))
	assert.Nil(t, r.Snapshot())
	require.False(t, r.Remove("x"))

	r.Add("x")
	require.Equal(t, 1, r.len())
	require.True(t, r.Contains("x"))
	assert.Equal(t, []string{"x"}, r.Snapshot())
	require.True(t, r.Remove("x"))
	require.Equal(t, 0, r.len())
}

func Test_MarketNameRegistry_RemoveAbsent(t *testing.T) {
	r := New(nil)
	require.False(t, r.Remove("nope"))
	r.Add("only")
	require.False(t, r.Remove("other"))
	require.True(t, r.Remove("only"))
	require.False(t, r.Remove("only"))
}

func Test_MarketNameRegistry_ConcurrentDistinctAdds(t *testing.T) {
	r := New(nil)
	const n = 128
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			r.Add(fmt.Sprintf("sym%d", i))
		}(i)
	}
	wg.Wait()
	require.Equal(t, n, r.len())
	snap := r.Snapshot()
	require.Len(t, snap, n)
	for i := range n {
		assert.Contains(t, snap, fmt.Sprintf("sym%d", i))
	}
}
