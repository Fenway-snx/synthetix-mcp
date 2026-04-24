package doubles_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func Test_SpyLogger_SATISFIES_INTERFACE(t *testing.T) {
	var _ snx_lib_logging.Logger = snx_lib_logging_doubles.NewSpyLogger()
}

func Test_SpyLogger_RECORDS_ALL_LEVELS(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()

	spy.Debug("d", "k1", "v1")
	spy.Info("i", "k2", "v2")
	spy.Warn("w", "k3", "v3")
	spy.Error("e", "k4", "v4")

	entries := spy.Entries()
	require.Len(t, entries, 4)

	assert.Equal(t, snx_lib_logging_doubles.LevelDebug, entries[0].Level)
	assert.Equal(t, "d", entries[0].Message)
	assert.Equal(t, []any{"k1", "v1"}, entries[0].KeyVals)

	assert.Equal(t, snx_lib_logging_doubles.LevelInfo, entries[1].Level)
	assert.Equal(t, snx_lib_logging_doubles.LevelWarn, entries[2].Level)
	assert.Equal(t, snx_lib_logging_doubles.LevelError, entries[3].Level)
}

func Test_SpyLogger_With_MERGES_CONTEXT(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	child := spy.With("service", "api")

	child.Info("hello", "rid", "abc")

	entries := spy.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, []any{"service", "api", "rid", "abc"}, entries[0].KeyVals)
}

func Test_SpyLogger_With_CHAINS(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	grandchild := spy.With("a", 1).With("b", 2)

	grandchild.Error("boom")

	entries := spy.Entries()
	require.Len(t, entries, 1)
	assert.Equal(t, []any{"a", 1, "b", 2}, entries[0].KeyVals)
}

func Test_SpyLogger_With_CHILD_AND_PARENT_SHARE_STORE(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	child := spy.With("scope", "child")

	spy.Info("from parent")
	child.Info("from child")

	assert.Len(t, spy.Entries(), 2)
}

func Test_SpyLogger_Messages_NO_FILTER(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	spy.Info("one")
	spy.Error("two")
	spy.Debug("three")

	assert.Equal(t, []string{"one", "two", "three"}, spy.Messages())
}

func Test_SpyLogger_Messages_FILTERED_BY_LEVEL(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	spy.Info("i1")
	spy.Error("e1")
	spy.Info("i2")
	spy.Debug("d1")

	assert.Equal(t, []string{"e1"}, spy.Messages(snx_lib_logging_doubles.LevelError))
	assert.Equal(t,
		[]string{"i1", "i2"},
		spy.Messages(snx_lib_logging_doubles.LevelInfo),
	)
	assert.Equal(t,
		[]string{"i1", "e1", "i2"},
		spy.Messages(snx_lib_logging_doubles.LevelInfo, snx_lib_logging_doubles.LevelError),
	)
}

func Test_SpyLogger_HasEntry_MATCHES(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	spy.Error("connection failed: timeout")
	spy.Info("started")

	assert.True(t, spy.HasEntry(snx_lib_logging_doubles.LevelError, "timeout"))
	assert.True(t, spy.HasEntry(snx_lib_logging_doubles.LevelError, ""))
	assert.False(t, spy.HasEntry(snx_lib_logging_doubles.LevelWarn, "timeout"))
	assert.False(t, spy.HasEntry(snx_lib_logging_doubles.LevelError, "refused"))
}

func Test_SpyLogger_Reset_CLEARS_ENTRIES(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	spy.Info("before")
	require.Len(t, spy.Entries(), 1)

	spy.Reset()
	assert.Empty(t, spy.Entries())

	spy.Info("after")
	assert.Len(t, spy.Entries(), 1)
}

func Test_SpyLogger_CONCURRENT_SAFETY(t *testing.T) {
	spy := snx_lib_logging_doubles.NewSpyLogger()
	child := spy.With("goroutine", true)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			spy.Info("parent")
		}()
		go func() {
			defer wg.Done()
			child.Error("child")
		}()
	}
	wg.Wait()

	assert.Len(t, spy.Entries(), 200)
}
