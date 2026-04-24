package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Transparent wrapper around *nats.RawConn that records publish and request-reply
// metrics. The stream label is the resolved stream name from the resolver.
type InstrumentedConn struct {
	Conn                        *nats.Conn
	subjectToStreamNameResolver *SubjectToStreamNameResolver
}

func NewInstrumentedConn(conn *nats.Conn, subjectToStreamNameResolver *SubjectToStreamNameResolver) *InstrumentedConn {
	return &InstrumentedConn{
		Conn:                        conn,
		subjectToStreamNameResolver: subjectToStreamNameResolver,
	}
}

func (ic *InstrumentedConn) Publish(subj string, data []byte) error {
	stream := ic.subjectToStreamNameResolver.Resolve(subj)
	start := snx_lib_utils_time.Now()

	err := ic.Conn.Publish(subj, data)

	corePublishDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		corePublishErrors.WithLabelValues(stream).Inc()
	}
	return err
}

func (ic *InstrumentedConn) PublishMsg(m *nats.Msg) error {
	stream := ic.subjectToStreamNameResolver.Resolve(m.Subject)
	start := snx_lib_utils_time.Now()

	err := ic.Conn.PublishMsg(m)

	corePublishDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		corePublishErrors.WithLabelValues(stream).Inc()
	}
	return err
}

func (ic *InstrumentedConn) Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	stream := ic.subjectToStreamNameResolver.Resolve(subj)
	start := snx_lib_utils_time.Now()

	msg, err := ic.Conn.Request(subj, data, timeout)

	coreRequestDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		coreRequestErrors.WithLabelValues(stream).Inc()
	}
	return msg, err
}

func (ic *InstrumentedConn) RequestWithContext(ctx context.Context, subj string, data []byte) (*nats.Msg, error) {
	stream := ic.subjectToStreamNameResolver.Resolve(subj)
	start := snx_lib_utils_time.Now()

	msg, err := ic.Conn.RequestWithContext(ctx, subj, data)

	coreRequestDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		coreRequestErrors.WithLabelValues(stream).Inc()
	}
	return msg, err
}

func (ic *InstrumentedConn) RequestMsg(m *nats.Msg, timeout time.Duration) (*nats.Msg, error) {
	stream := ic.subjectToStreamNameResolver.Resolve(m.Subject)
	start := snx_lib_utils_time.Now()

	msg, err := ic.Conn.RequestMsg(m, timeout)

	coreRequestDuration.WithLabelValues(stream).Observe(snx_lib_utils_time.Since(start).Seconds())
	if err != nil {
		coreRequestErrors.WithLabelValues(stream).Inc()
	}
	return msg, err
}
