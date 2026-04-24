package nats

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockJS struct {
	jetstream.JetStream
	publishFn         func(ctx context.Context, subject string, payload []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
	publishMsgFn      func(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
	publishAsyncFn    func(subject string, payload []byte, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, error)
	publishMsgAsyncFn func(msg *nats.Msg, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, error)
}

func (m *mockJS) Publish(ctx context.Context, subject string, payload []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	return m.publishFn(ctx, subject, payload, opts...)
}

func (m *mockJS) PublishMsg(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	return m.publishMsgFn(ctx, msg, opts...)
}

func (m *mockJS) PublishAsync(subject string, payload []byte, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
	return m.publishAsyncFn(subject, payload, opts...)
}

func (m *mockJS) PublishMsgAsync(msg *nats.Msg, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
	return m.publishMsgAsyncFn(msg, opts...)
}

func jsHistogramSampleCount(h *prometheus.HistogramVec, label string) uint64 {
	var m dto.Metric
	h.WithLabelValues(label).(prometheus.Metric).Write(&m)
	return m.GetHistogram().GetSampleCount()
}

func Test_InstrumentedJetStream_Publish_Success(t *testing.T) {
	resolver := NewSubjectToStreamNameResolver()
	subject := TradingEventOrderHistory.String()
	expectedStream := StreamName_ExecutionEvents.String()

	mock := &mockJS{
		publishFn: func(_ context.Context, _ string, _ []byte, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			return &jetstream.PubAck{Stream: expectedStream}, nil
		},
	}

	ijs := NewInstrumentedJetStream(mock, resolver)
	errsBefore := testutil.ToFloat64(jsPublishErrors.WithLabelValues(expectedStream))
	countBefore := jsHistogramSampleCount(jsPublishDuration, expectedStream)

	ack, err := ijs.Publish(context.Background(), subject, []byte("test"))

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, expectedStream, ack.Stream)
	assert.Equal(t, errsBefore, testutil.ToFloat64(jsPublishErrors.WithLabelValues(expectedStream)))
	assert.Equal(t, countBefore+1, jsHistogramSampleCount(jsPublishDuration, expectedStream))
}

func Test_InstrumentedJetStream_Publish_Error(t *testing.T) {
	resolver := NewSubjectToStreamNameResolver()
	subject := TradingEventOrderHistory.String()
	expectedStream := StreamName_ExecutionEvents.String()

	mock := &mockJS{
		publishFn: func(_ context.Context, _ string, _ []byte, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			return nil, errors.New("publish failed")
		},
	}

	ijs := NewInstrumentedJetStream(mock, resolver)
	errsBefore := testutil.ToFloat64(jsPublishErrors.WithLabelValues(expectedStream))
	countBefore := jsHistogramSampleCount(jsPublishDuration, expectedStream)

	_, err := ijs.Publish(context.Background(), subject, []byte("test"))

	require.Error(t, err)
	assert.Equal(t, errsBefore+1, testutil.ToFloat64(jsPublishErrors.WithLabelValues(expectedStream)))
	assert.Equal(t, countBefore+1, jsHistogramSampleCount(jsPublishDuration, expectedStream))
}

func Test_InstrumentedJetStream_PublishMsg_Success(t *testing.T) {
	resolver := NewSubjectToStreamNameResolver()
	subject := FundingRatePosted.String()
	expectedStream := StreamName_FundingEvents.String()

	mock := &mockJS{
		publishMsgFn: func(_ context.Context, msg *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			assert.Equal(t, subject, msg.Subject)
			return &jetstream.PubAck{Stream: expectedStream}, nil
		},
	}

	ijs := NewInstrumentedJetStream(mock, resolver)
	countBefore := jsHistogramSampleCount(jsPublishDuration, expectedStream)

	ack, err := ijs.PublishMsg(context.Background(), &nats.Msg{Subject: subject, Data: []byte("test")})

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, expectedStream, ack.Stream)
	assert.Equal(t, countBefore+1, jsHistogramSampleCount(jsPublishDuration, expectedStream))
}

func Test_InstrumentedJetStream_PublishAsync_Success(t *testing.T) {
	resolver := NewSubjectToStreamNameResolver()
	subject := MakeSubjectForEventSouth("BTC-USDT")
	expectedStream := StreamName_Orders.String()

	mock := &mockJS{
		publishAsyncFn: func(_ string, _ []byte, _ ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
			return nil, nil
		},
	}

	ijs := NewInstrumentedJetStream(mock, resolver)
	errsBefore := testutil.ToFloat64(jsAsyncEnqueueErrors.WithLabelValues(expectedStream))
	countBefore := jsHistogramSampleCount(jsAsyncEnqueueDuration, expectedStream)

	_, err := ijs.PublishAsync(subject, []byte("test"))

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, errsBefore, testutil.ToFloat64(jsAsyncEnqueueErrors.WithLabelValues(expectedStream)))
	assert.Equal(t, countBefore+1, jsHistogramSampleCount(jsAsyncEnqueueDuration, expectedStream))
}

func Test_InstrumentedJetStream_PublishAsync_Error(t *testing.T) {
	resolver := NewSubjectToStreamNameResolver()
	subject := MakeSubjectForEventSouth("BTC-USDT")
	expectedStream := StreamName_Orders.String()

	mock := &mockJS{
		publishAsyncFn: func(_ string, _ []byte, _ ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
			return nil, errors.New("enqueue failed")
		},
	}

	ijs := NewInstrumentedJetStream(mock, resolver)
	errsBefore := testutil.ToFloat64(jsAsyncEnqueueErrors.WithLabelValues(expectedStream))
	countBefore := jsHistogramSampleCount(jsAsyncEnqueueDuration, expectedStream)

	_, err := ijs.PublishAsync(subject, []byte("test"))

	require.Error(t, err)
	assert.Equal(t, errsBefore+1, testutil.ToFloat64(jsAsyncEnqueueErrors.WithLabelValues(expectedStream)))
	assert.Equal(t, countBefore+1, jsHistogramSampleCount(jsAsyncEnqueueDuration, expectedStream))
}

func Test_InstrumentedJetStream_PublishMsgAsync_ResolvesStream(t *testing.T) {
	resolver := NewSubjectToStreamNameResolver()
	subject := MakeSubjectForOrderbookJournal("ETH-USDT")
	expectedStream := StreamName_OrderBookJournal.String()

	mock := &mockJS{
		publishMsgAsyncFn: func(msg *nats.Msg, _ ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
			assert.Equal(t, subject, msg.Subject)
			return nil, nil
		},
	}

	ijs := NewInstrumentedJetStream(mock, resolver)
	errsBefore := testutil.ToFloat64(jsAsyncEnqueueErrors.WithLabelValues(expectedStream))
	countBefore := jsHistogramSampleCount(jsAsyncEnqueueDuration, expectedStream)

	_, err := ijs.PublishMsgAsync(&nats.Msg{Subject: subject, Data: []byte("test")})

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, errsBefore, testutil.ToFloat64(jsAsyncEnqueueErrors.WithLabelValues(expectedStream)))
	assert.Equal(t, countBefore+1, jsHistogramSampleCount(jsAsyncEnqueueDuration, expectedStream))
}
