package nats

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startTestNATS(t *testing.T) *nats.Conn {
	t.Helper()
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	s := natsserver.RunServer(&opts)
	t.Cleanup(s.Shutdown)

	nc, err := nats.Connect(s.ClientURL(), nats.Timeout(2*time.Second))
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	t.Cleanup(nc.Close)
	return nc
}

func histogramSampleCount(h *prometheus.HistogramVec, label string) uint64 {
	var m dto.Metric
	h.WithLabelValues(label).(prometheus.Metric).Write(&m)
	return m.GetHistogram().GetSampleCount()
}

func Test_InstrumentedConn_Publish_Success(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	stream := StreamName_ExecutionEvents.String()
	errsBefore := testutil.ToFloat64(corePublishErrors.WithLabelValues(stream))
	countBefore := histogramSampleCount(corePublishDuration, stream)

	err := ic.Publish(TradingEventOrderHistory.String(), []byte("test"))

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, errsBefore, testutil.ToFloat64(corePublishErrors.WithLabelValues(stream)))
	assert.Equal(t, countBefore+1, histogramSampleCount(corePublishDuration, stream))
}

func Test_InstrumentedConn_Publish_Error(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	stream := StreamName_ExecutionEvents.String()
	nc.Close()

	errsBefore := testutil.ToFloat64(corePublishErrors.WithLabelValues(stream))
	countBefore := histogramSampleCount(corePublishDuration, stream)

	err := ic.Publish(TradingEventOrderHistory.String(), []byte("test"))

	require.Error(t, err)
	assert.Equal(t, errsBefore+1, testutil.ToFloat64(corePublishErrors.WithLabelValues(stream)))
	assert.Equal(t, countBefore+1, histogramSampleCount(corePublishDuration, stream))
}

func Test_InstrumentedConn_PublishMsg_Success(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	stream := StreamName_FundingEvents.String()
	errsBefore := testutil.ToFloat64(corePublishErrors.WithLabelValues(stream))
	countBefore := histogramSampleCount(corePublishDuration, stream)

	err := ic.PublishMsg(&nats.Msg{Subject: FundingRatePosted.String(), Data: []byte("test")})

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, errsBefore, testutil.ToFloat64(corePublishErrors.WithLabelValues(stream)))
	assert.Equal(t, countBefore+1, histogramSampleCount(corePublishDuration, stream))
}

func Test_InstrumentedConn_PublishMsg_Error(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	stream := StreamName_FundingEvents.String()
	nc.Close()

	errsBefore := testutil.ToFloat64(corePublishErrors.WithLabelValues(stream))
	countBefore := histogramSampleCount(corePublishDuration, stream)

	err := ic.PublishMsg(&nats.Msg{Subject: FundingRatePosted.String(), Data: []byte("test")})

	require.Error(t, err)
	assert.Equal(t, errsBefore+1, testutil.ToFloat64(corePublishErrors.WithLabelValues(stream)))
	assert.Equal(t, countBefore+1, histogramSampleCount(corePublishDuration, stream))
}

func Test_InstrumentedConn_Request_Success(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	subj := MakeSubject("req.echo")
	sub, err := nc.Subscribe(subj, func(msg *nats.Msg) { _ = msg.Respond([]byte("pong")) })
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	defer sub.Unsubscribe()

	label := ""
	errsBefore := testutil.ToFloat64(coreRequestErrors.WithLabelValues(label))
	countBefore := histogramSampleCount(coreRequestDuration, label)

	resp, err := ic.Request(subj, []byte("ping"), 2*time.Second)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, []byte("pong"), resp.Data)
	assert.Equal(t, errsBefore, testutil.ToFloat64(coreRequestErrors.WithLabelValues(label)))
	assert.Equal(t, countBefore+1, histogramSampleCount(coreRequestDuration, label))
}

func Test_InstrumentedConn_Request_NoResponder(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	label := ""
	errsBefore := testutil.ToFloat64(coreRequestErrors.WithLabelValues(label))
	countBefore := histogramSampleCount(coreRequestDuration, label)

	_, err := ic.Request(MakeSubject("no.responder.here"), []byte("test"), 50*time.Millisecond)

	require.Error(t, err)
	assert.Equal(t, errsBefore+1, testutil.ToFloat64(coreRequestErrors.WithLabelValues(label)))
	assert.Equal(t, countBefore+1, histogramSampleCount(coreRequestDuration, label))
}

func Test_InstrumentedConn_RequestWithContext_Success(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	subj := MakeSubject("ctx.echo")
	sub, err := nc.Subscribe(subj, func(msg *nats.Msg) { _ = msg.Respond([]byte("pong")) })
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	defer sub.Unsubscribe()

	label := ""
	errsBefore := testutil.ToFloat64(coreRequestErrors.WithLabelValues(label))
	countBefore := histogramSampleCount(coreRequestDuration, label)

	resp, err := ic.RequestWithContext(context.Background(), subj, []byte("ping"))

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, []byte("pong"), resp.Data)
	assert.Equal(t, errsBefore, testutil.ToFloat64(coreRequestErrors.WithLabelValues(label)))
	assert.Equal(t, countBefore+1, histogramSampleCount(coreRequestDuration, label))
}

func Test_InstrumentedConn_RequestWithContext_NoResponder(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	label := ""
	errsBefore := testutil.ToFloat64(coreRequestErrors.WithLabelValues(label))
	countBefore := histogramSampleCount(coreRequestDuration, label)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := ic.RequestWithContext(ctx, MakeSubject("no.responder.ctx"), []byte("test"))

	require.Error(t, err)
	assert.Equal(t, errsBefore+1, testutil.ToFloat64(coreRequestErrors.WithLabelValues(label)))
	assert.Equal(t, countBefore+1, histogramSampleCount(coreRequestDuration, label))
}

func Test_InstrumentedConn_RequestMsg_Success(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	subj := MakeSubject("msg.echo")
	sub, err := nc.Subscribe(subj, func(msg *nats.Msg) { _ = msg.Respond([]byte("pong")) })
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	defer sub.Unsubscribe()

	label := ""
	errsBefore := testutil.ToFloat64(coreRequestErrors.WithLabelValues(label))
	countBefore := histogramSampleCount(coreRequestDuration, label)

	resp, err := ic.RequestMsg(&nats.Msg{Subject: subj, Data: []byte("ping")}, 2*time.Second)

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, []byte("pong"), resp.Data)
	assert.Equal(t, errsBefore, testutil.ToFloat64(coreRequestErrors.WithLabelValues(label)))
	assert.Equal(t, countBefore+1, histogramSampleCount(coreRequestDuration, label))
}

func Test_InstrumentedConn_RequestMsg_NoResponder(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	label := ""
	errsBefore := testutil.ToFloat64(coreRequestErrors.WithLabelValues(label))
	countBefore := histogramSampleCount(coreRequestDuration, label)

	_, err := ic.RequestMsg(&nats.Msg{Subject: MakeSubject("no.responder.msg"), Data: []byte("test")}, 50*time.Millisecond)

	require.Error(t, err)
	assert.Equal(t, errsBefore+1, testutil.ToFloat64(coreRequestErrors.WithLabelValues(label)))
	assert.Equal(t, countBefore+1, histogramSampleCount(coreRequestDuration, label))
}

func Test_InstrumentedConn_NonStreamSubject(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	errsBefore := testutil.ToFloat64(corePublishErrors.WithLabelValues(""))
	countBefore := histogramSampleCount(corePublishDuration, "")

	err := ic.Publish(MakeSubject("price.feed.BTC-USDT.mark"), []byte("test"))

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, errsBefore, testutil.ToFloat64(corePublishErrors.WithLabelValues("")))
	assert.Equal(t, countBefore+1, histogramSampleCount(corePublishDuration, ""))
}

func Test_InstrumentedConn_RawConn(t *testing.T) {
	nc := startTestNATS(t)
	ic := NewInstrumentedConn(nc, NewSubjectToStreamNameResolver())

	assert.Equal(t, nc, ic.Conn)
}
