package jetstreamqueues

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test double for jetstream.Consumer; only Info is exercised.
type stubConsumer struct {
	infoFn func(context.Context) (*jetstream.ConsumerInfo, error)
}

func (s *stubConsumer) Fetch(int, ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	return nil, nil
}

func (s *stubConsumer) FetchBytes(int, ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	return nil, nil
}

func (s *stubConsumer) FetchNoWait(int) (jetstream.MessageBatch, error) { return nil, nil }

func (s *stubConsumer) Consume(jetstream.MessageHandler, ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, error) {
	return nil, nil
}

func (s *stubConsumer) Messages(...jetstream.PullMessagesOpt) (jetstream.MessagesContext, error) {
	return nil, nil
}

func (s *stubConsumer) Next(...jetstream.FetchOpt) (jetstream.Msg, error) { return nil, nil }

func (s *stubConsumer) Info(ctx context.Context) (*jetstream.ConsumerInfo, error) {
	if s.infoFn != nil {
		return s.infoFn(ctx)
	}
	return nil, nil
}

func (s *stubConsumer) CachedInfo() *jetstream.ConsumerInfo { return nil }

var _ jetstream.Consumer = (*stubConsumer)(nil)

func Test_DepthsFromConsumers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	consumers := map[string]jetstream.Consumer{
		"k1": &stubConsumer{
			infoFn: func(context.Context) (*jetstream.ConsumerInfo, error) {
				return &jetstream.ConsumerInfo{
					Stream:        "S1",
					NumPending:    3,
					NumAckPending: 2,
					Config: jetstream.ConsumerConfig{
						Durable: "dur-one",
					},
				}, nil
			},
		},
		"k2": &stubConsumer{
			infoFn: func(context.Context) (*jetstream.ConsumerInfo, error) {
				return nil, errors.New("nats unavailable")
			},
		},
	}

	got, err := DepthsFromConsumers(ctx, consumers)
	require.Error(t, err)
	assert.Nil(t, got)

	consumersOK := map[string]jetstream.Consumer{
		"k1": &stubConsumer{
			infoFn: func(context.Context) (*jetstream.ConsumerInfo, error) {
				return &jetstream.ConsumerInfo{
					Stream:        "S1",
					NumPending:    3,
					NumAckPending: 2,
					Config: jetstream.ConsumerConfig{
						Durable: "dur-one",
					},
				}, nil
			},
		},
	}

	got, err = DepthsFromConsumers(ctx, consumersOK)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 1)
	assert.Equal(t, "S1", got[0].Stream)
	assert.Equal(t, "dur-one", got[0].Consumer)
	assert.Equal(t, int64(3), got[0].Pending)
	assert.Equal(t, int64(2), got[0].AckPending)
}

func Test_DepthsFromConsumers_SortedByStreamThenConsumer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	makeConsumer := func(stream, durable string, pending, ack int) *stubConsumer {
		return &stubConsumer{
			infoFn: func(context.Context) (*jetstream.ConsumerInfo, error) {
				return &jetstream.ConsumerInfo{
					Stream:        stream,
					NumPending:    uint64(pending),
					NumAckPending: ack,
					Config: jetstream.ConsumerConfig{
						Durable: durable,
					},
				}, nil
			},
		}
	}

	consumers := map[string]jetstream.Consumer{
		"z": makeConsumer("ZSTREAM", "z", 1, 0),
		"a": makeConsumer("A", "a2", 2, 0),
		"b": makeConsumer("A", "a1", 3, 0),
		"c": makeConsumer("B", "b1", 4, 0),
	}

	got, err := DepthsFromConsumers(ctx, consumers)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 4)

	want := []JetStreamQueueDepth{
		{Stream: "A", Consumer: "a1", Pending: 3, AckPending: 0},
		{Stream: "A", Consumer: "a2", Pending: 2, AckPending: 0},
		{Stream: "B", Consumer: "b1", Pending: 4, AckPending: 0},
		{Stream: "ZSTREAM", Consumer: "z", Pending: 1, AckPending: 0},
	}
	assert.Equal(t, want, got)
}

func Test_DepthsFromConsumers_EmptyMap(t *testing.T) {
	t.Parallel()

	got, err := DepthsFromConsumers(context.Background(), nil)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Nil(t, got)
}

func Test_QueueDepthsFromConsumerMap(t *testing.T) {
	t.Parallel()

	var mu sync.RWMutex
	consumers := map[string]jetstream.Consumer{
		"k1": &stubConsumer{
			infoFn: func(context.Context) (*jetstream.ConsumerInfo, error) {
				return &jetstream.ConsumerInfo{
					Stream:        "S1",
					NumPending:    1,
					NumAckPending: 0,
					Config: jetstream.ConsumerConfig{
						Durable: "d1",
					},
				}, nil
			},
		},
	}

	got, err := QueueDepthsFromConsumerMap(context.Background(), &mu, consumers)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 1)
	assert.Equal(t, "S1", got[0].Stream)
	assert.Equal(t, "d1", got[0].Consumer)
}

func Test_MergeSortJetStreamQueueDepths(t *testing.T) {
	t.Parallel()

	a := []JetStreamQueueDepth{
		{Stream: "B", Consumer: "c2"},
		{Stream: "A", Consumer: "c1"},
	}
	b := []JetStreamQueueDepth{
		{Stream: "A", Consumer: "c0"},
	}

	got := MergeSortJetStreamQueueDepths(a, nil, b)
	require.Len(t, got, 3)
	assert.Equal(t, "A", got[0].Stream)
	assert.Equal(t, "c0", got[0].Consumer)
	assert.Equal(t, "A", got[1].Stream)
	assert.Equal(t, "c1", got[1].Consumer)
	assert.Equal(t, "B", got[2].Stream)
}

func Test_DepthFromConsumerInfo_Nil(t *testing.T) {
	t.Parallel()

	row := depthFromConsumerInfo(nil)
	assert.Equal(t, JetStreamQueueDepth{}, row)
}

func Test_DepthsFromConsumers_NilInfoNoError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	consumers := map[string]jetstream.Consumer{
		"k1": &stubConsumer{}, // Info() returns (nil, nil); must not panic.
	}

	got, err := DepthsFromConsumers(ctx, consumers)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 1)
	assert.Equal(t, JetStreamQueueDepth{}, got[0])
}

func Test_DepthsFromConsumers_SkipsNilConsumer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	consumers := map[string]jetstream.Consumer{
		"reserved": nil,
		"k1": &stubConsumer{
			infoFn: func(context.Context) (*jetstream.ConsumerInfo, error) {
				return &jetstream.ConsumerInfo{
					Stream:        "S1",
					NumPending:    1,
					NumAckPending: 0,
					Config: jetstream.ConsumerConfig{
						Durable: "d1",
					},
				}, nil
			},
		},
	}

	got, err := DepthsFromConsumers(ctx, consumers)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 1)
	assert.Equal(t, "S1", got[0].Stream)
	assert.Equal(t, "d1", got[0].Consumer)
}

func Test_QueueDepthsFromConsumerMap_SkipsNilConsumer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var mu sync.RWMutex
	consumers := map[string]jetstream.Consumer{
		"reserved": nil,
		"k1": &stubConsumer{
			infoFn: func(context.Context) (*jetstream.ConsumerInfo, error) {
				return &jetstream.ConsumerInfo{
					Stream:        "S1",
					NumPending:    2,
					NumAckPending: 1,
					Config: jetstream.ConsumerConfig{
						Durable: "d1",
					},
				}, nil
			},
		},
	}

	got, err := QueueDepthsFromConsumerMap(ctx, &mu, consumers)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.Len(t, got, 1)
	assert.Equal(t, int64(2), got[0].Pending)
	assert.Equal(t, int64(1), got[0].AckPending)
}

func Test_ConsumerNameFromInfo_PrefersDurable(t *testing.T) {
	t.Parallel()

	info := &jetstream.ConsumerInfo{
		Name: "ephemeral-name",
		Config: jetstream.ConsumerConfig{
			Durable: "my-durable",
			Name:    "ignored",
		},
	}
	row := depthFromConsumerInfo(info)
	assert.Equal(t, "my-durable", row.Consumer)
}

func Test_ConsumerNameFromInfo_FallbackName(t *testing.T) {
	t.Parallel()

	info := &jetstream.ConsumerInfo{
		Name: "server-name",
		Config: jetstream.ConsumerConfig{
			Name: "cfg-name",
		},
	}
	row := depthFromConsumerInfo(info)
	assert.Equal(t, "cfg-name", row.Consumer)
}
