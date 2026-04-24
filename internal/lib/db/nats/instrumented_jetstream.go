package nats

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Transparent wrapper around jetstream.JetStream that records publish latency
// and error metrics. All non-publish methods pass through via embedding.
// The stream label is resolved from the subject at publish time via SubjectToStreamNameResolver.
type InstrumentedJetStream struct {
	jetstream.JetStream
	subjectToStreamNameResolver *SubjectToStreamNameResolver
}

func NewInstrumentedJetStream(js jetstream.JetStream, subjectToStreamNameResolver *SubjectToStreamNameResolver) *InstrumentedJetStream {
	return &InstrumentedJetStream{
		JetStream:                   js,
		subjectToStreamNameResolver: subjectToStreamNameResolver,
	}
}

func (ijs *InstrumentedJetStream) Publish(ctx context.Context, subject string, payload []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	stream := ijs.subjectToStreamNameResolver.Resolve(subject)
	start := snx_lib_utils_time.Now()

	ack, err := ijs.JetStream.Publish(ctx, subject, payload, opts...)

	jsPublishDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		jsPublishErrors.WithLabelValues(stream).Inc()
	}
	return ack, err
}

func (ijs *InstrumentedJetStream) PublishMsg(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	stream := ijs.subjectToStreamNameResolver.Resolve(msg.Subject)
	start := snx_lib_utils_time.Now()

	ack, err := ijs.JetStream.PublishMsg(ctx, msg, opts...)

	jsPublishDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		jsPublishErrors.WithLabelValues(stream).Inc()
	}
	return ack, err
}

func (ijs *InstrumentedJetStream) PublishAsync(subject string, payload []byte, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
	stream := ijs.subjectToStreamNameResolver.Resolve(subject)
	start := snx_lib_utils_time.Now()

	future, err := ijs.JetStream.PublishAsync(subject, payload, opts...)

	jsAsyncEnqueueDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		jsAsyncEnqueueErrors.WithLabelValues(stream).Inc()
	}
	return future, err
}

func (ijs *InstrumentedJetStream) PublishMsgAsync(msg *nats.Msg, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, error) {
	stream := ijs.subjectToStreamNameResolver.Resolve(msg.Subject)
	start := snx_lib_utils_time.Now()

	future, err := ijs.JetStream.PublishMsgAsync(msg, opts...)

	jsAsyncEnqueueDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		jsAsyncEnqueueErrors.WithLabelValues(stream).Inc()
	}
	return future, err
}
