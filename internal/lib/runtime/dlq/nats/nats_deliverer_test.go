package nats

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

type stubJetStream struct {
	jetstream.JetStream
	createOrUpdateStreamFunc func(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error)
	publishMsgFunc           func(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
}

func (s *stubJetStream) CreateOrUpdateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	return s.createOrUpdateStreamFunc(ctx, cfg)
}

func (s *stubJetStream) PublishMsg(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	return s.publishMsgFunc(ctx, msg, opts...)
}

type spyFallback struct {
	called             bool
	receivedEnvelope   snx_lib_dlq.Envelope
	receivedJSONString string
	err                error
}

func (f *spyFallback) OnPost(envelope snx_lib_dlq.Envelope, envelopeJSONString string) error {
	f.called = true
	f.receivedEnvelope = envelope
	f.receivedJSONString = envelopeJSONString
	return f.err
}

func normativeStreamCreation() func(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	return func(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
		return nil, nil
	}
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func Test_NewNATSDeliverer_nil_LOGGER(t *testing.T) {
	js := &stubJetStream{
		createOrUpdateStreamFunc: normativeStreamCreation(),
	}

	assert.Panics(t, func() {
		NewNATSDeliverer(nil, context.Background(), js, 1, nil)
	})
}

func Test_NewNATSDeliverer_nil_ctx(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	assert.Panics(t, func() {
		NewNATSDeliverer(logger, nil, &stubJetStream{createOrUpdateStreamFunc: normativeStreamCreation()}, 1, nil)
	})
}

func Test_NewNATSDeliverer_nil_JETSTREAM(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	assert.Panics(t, func() {
		NewNATSDeliverer(logger, context.Background(), nil, 1, nil)
	})
}

func Test_NewNATSDeliverer_StreamCreationFailure(t *testing.T) {
	js := &stubJetStream{
		createOrUpdateStreamFunc: func(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
			return nil, errors.New("connection refused")
		},
	}

	deliverer, err := NewNATSDeliverer(snx_lib_logging_doubles.NewStubLogger(), context.Background(), js, 1, nil)

	require.Error(t, err)
	assert.Nil(t, deliverer)
	assert.Contains(t, err.Error(), "failed to create DLQ stream")
}

func Test_NewNATSDeliverer_Success(t *testing.T) {
	js := &stubJetStream{
		createOrUpdateStreamFunc: normativeStreamCreation(),
	}

	deliverer, err := NewNATSDeliverer(snx_lib_logging_doubles.NewStubLogger(), context.Background(), js, 1, nil)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.NotNil(t, deliverer)
}

// ---------------------------------------------------------------------------
// OnPost tests
// ---------------------------------------------------------------------------

func Test_NATSDeliverer_OnPost_Success(t *testing.T) {
	var capturedMsg *nats.Msg

	js := &stubJetStream{
		publishMsgFunc: func(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			capturedMsg = msg
			return &jetstream.PubAck{Stream: "test-DLQ", Sequence: 42}, nil
		},
	}

	fallback := &spyFallback{}

	nd := &natsDeliverer{
		logger:   snx_lib_logging_doubles.NewStubLogger(),
		ctx:      context.Background(),
		js:       js,
		subject:  "test.system.dlq.posted",
		fallback: fallback,
	}

	envelope := snx_lib_dlq.Envelope{
		Application: "test-app",
		System:      "test-system",
	}
	jsonStr := `{"application":"test-app","system":"test-system"}`

	err := nd.OnPost(envelope, jsonStr)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.False(t, fallback.called, "fallback should not be called on success")

	require.NotNil(t, capturedMsg)
	assert.Equal(t, "test.system.dlq.posted", capturedMsg.Subject)
	assert.Equal(t, jsonStr, string(capturedMsg.Data))
}

func Test_NATSDeliverer_OnPost_PublishFailure_WithFallback(t *testing.T) {
	js := &stubJetStream{
		publishMsgFunc: func(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			return nil, errors.New("connection lost")
		},
	}

	fallback := &spyFallback{}

	nd := &natsDeliverer{
		logger:   snx_lib_logging_doubles.NewStubLogger(),
		ctx:      context.Background(),
		js:       js,
		subject:  "test.system.dlq.posted",
		fallback: fallback,
	}

	envelope := snx_lib_dlq.Envelope{
		Application: "test-app",
		System:      "test-system",
	}
	jsonStr := `{"application":"test-app","system":"test-system"}`

	err := nd.OnPost(envelope, jsonStr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish dead letter to NATS")
	assert.Contains(t, err.Error(), "published dead letter to fallback")

	assert.True(t, fallback.called, "fallback must be called when NATS publish fails")
	assert.Equal(t, jsonStr, fallback.receivedJSONString)
	assert.Equal(t, "test-app", fallback.receivedEnvelope.Application)
	assert.Equal(t, "test-system", fallback.receivedEnvelope.System)
}

func Test_NATSDeliverer_OnPost_PublishFailure_FallbackAlsoFails(t *testing.T) {
	js := &stubJetStream{
		publishMsgFunc: func(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			return nil, errors.New("connection lost")
		},
	}

	fallback := &spyFallback{err: errors.New("disk full")}

	nd := &natsDeliverer{
		logger:   snx_lib_logging_doubles.NewStubLogger(),
		ctx:      context.Background(),
		js:       js,
		subject:  "test.system.dlq.posted",
		fallback: fallback,
	}

	envelope := snx_lib_dlq.Envelope{Application: "test-app"}
	jsonStr := `{"application":"test-app"}`

	err := nd.OnPost(envelope, jsonStr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish dead letter to NATS")
	assert.Contains(t, err.Error(), "failed to publish dead letter to fallback")
	assert.True(t, fallback.called)
}

func Test_NATSDeliverer_OnPost_PublishFailure_NoFallback(t *testing.T) {
	js := &stubJetStream{
		publishMsgFunc: func(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			return nil, errors.New("connection lost")
		},
	}

	nd := &natsDeliverer{
		logger:   snx_lib_logging_doubles.NewStubLogger(),
		ctx:      context.Background(),
		js:       js,
		subject:  "test.system.dlq.posted",
		fallback: nil,
	}

	envelope := snx_lib_dlq.Envelope{Application: "test-app"}
	jsonStr := `{"application":"test-app"}`

	err := nd.OnPost(envelope, jsonStr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish dead letter to NATS")
	assert.NotContains(t, err.Error(), "fallback")
}
