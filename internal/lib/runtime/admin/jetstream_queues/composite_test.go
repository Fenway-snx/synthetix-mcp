package jetstreamqueues

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CompositeJetStreamQueueDepthCollector_MergeOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return []JetStreamQueueDepth{{Stream: "B", Consumer: "x"}}, nil
	})
	b := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return []JetStreamQueueDepth{{Stream: "A", Consumer: "y"}}, nil
	})
	got, err := NewCompositeJetStreamQueueDepthCollector(a, b).CollectJetStreamQueueDepths(ctx)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 2)
	assert.Equal(t, "A", got[0].Stream)
	assert.Equal(t, "B", got[1].Stream)
}

func Test_CompositeJetStreamQueueDepthCollector_ErrorPropagation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	boom := errors.New("collect failed")
	ok := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return []JetStreamQueueDepth{{Stream: "S", Consumer: "c"}}, nil
	})
	fail := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return nil, boom
	})
	got, err := NewCompositeJetStreamQueueDepthCollector(ok, fail).CollectJetStreamQueueDepths(ctx)
	require.ErrorIs(t, err, boom)
	require.Len(t, got, 1)
	assert.Equal(t, "c", got[0].Consumer)
}

func Test_CompositeJetStreamQueueDepthCollector_JoinsAllErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	e1 := errors.New("first")
	e2 := errors.New("second")
	c1 := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return nil, e1
	})
	c2 := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return nil, e2
	})
	got, err := NewCompositeJetStreamQueueDepthCollector(c1, c2).CollectJetStreamQueueDepths(ctx)
	require.ErrorIs(t, err, e1)
	require.ErrorIs(t, err, e2)
	assert.Empty(t, got)
}

func Test_CompositeJetStreamQueueDepthCollector_PartialAfterMiddleFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	eMid := errors.New("middle failed")
	first := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return []JetStreamQueueDepth{{Stream: "A", Consumer: "1"}}, nil
	})
	mid := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return nil, eMid
	})
	last := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return []JetStreamQueueDepth{{Stream: "B", Consumer: "2"}}, nil
	})
	got, err := NewCompositeJetStreamQueueDepthCollector(first, mid, last).CollectJetStreamQueueDepths(ctx)
	require.ErrorIs(t, err, eMid)
	require.Len(t, got, 2)
	assert.Equal(t, "A", got[0].Stream)
	assert.Equal(t, "B", got[1].Stream)
}

func Test_CompositeJetStreamQueueDepthCollector_EmptyParts(t *testing.T) {
	t.Parallel()
	got, err := NewCompositeJetStreamQueueDepthCollector().CollectJetStreamQueueDepths(t.Context())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Empty(t, got)
}

func Test_CompositeJetStreamQueueDepthCollector_SkipsNilParts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ok := FuncCollector(func(context.Context) ([]JetStreamQueueDepth, error) {
		return []JetStreamQueueDepth{{Stream: "S", Consumer: "only"}}, nil
	})
	var nilCollector JetStreamQueueDepthCollector
	got, err := NewCompositeJetStreamQueueDepthCollector(nilCollector, ok).CollectJetStreamQueueDepths(ctx)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 1)
	assert.Equal(t, "only", got[0].Consumer)
}

func Test_CompositeJetStreamQueueDepthCollector_NilReceiver(t *testing.T) {
	t.Parallel()
	var c *CompositeJetStreamQueueDepthCollector
	got, err := c.CollectJetStreamQueueDepths(t.Context())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Empty(t, got)
}

func Test_CompositeJetStreamQueueDepthCollector_InterfaceCompliance(t *testing.T) {
	var _ JetStreamQueueDepthCollector = (*CompositeJetStreamQueueDepthCollector)(nil)
}
