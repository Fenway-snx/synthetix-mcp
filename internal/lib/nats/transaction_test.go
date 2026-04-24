package nats

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

// mockJetStream satisfies jetstream.JetStream; only PublishMsg is functional.
type mockJetStream struct {
	jetstream.JetStream
	publishMsgFn func(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
}

func (m *mockJetStream) PublishMsg(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	return m.publishMsgFn(ctx, msg, opts...)
}

func Test_RoleForMethod_MapsCorrectly(t *testing.T) {
	tests := []struct {
		name         string
		method       MethodName
		expectedRole SignerRole
	}{
		{"RequestWithdrawal maps to relayer", MethodName_RequestWithdrawal, SignerRole_Relayer},
		{"CastWatcherVotes maps to watcher", MethodName_CastWatcherVotes, SignerRole_Watcher},
		{"DisputeWithdrawals maps to watcher", MethodName_DisputeWithdrawals, SignerRole_Watcher},
		{"DisburseWithdrawals maps to teller", MethodName_DisburseWithdrawals, SignerRole_Teller},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, err := RoleForMethod(tt.method)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expectedRole, role)
		})
	}
}

func Test_RoleForMethod_UnknownMethod_ReturnsError(t *testing.T) {
	_, err := RoleForMethod("nonExistentMethod")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown method name")
}

func Test_SubjectForMethod_RoutesCorrectly(t *testing.T) {
	tests := []struct {
		name            string
		method          MethodName
		expectedSubject snx_lib_db_nats.Subject
	}{
		{"RequestWithdrawal routes to relayer queue", MethodName_RequestWithdrawal, snx_lib_db_nats.RelayerTxnQueueRelayer},
		{"CastWatcherVotes routes to watcher queue", MethodName_CastWatcherVotes, snx_lib_db_nats.RelayerTxnQueueWatcher},
		{"DisputeWithdrawals routes to watcher queue", MethodName_DisputeWithdrawals, snx_lib_db_nats.RelayerTxnQueueWatcher},
		{"DisburseWithdrawals routes to teller queue", MethodName_DisburseWithdrawals, snx_lib_db_nats.RelayerTxnQueueTeller},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, err := subjectForMethod(tt.method)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expectedSubject.String(), subject)
		})
	}
}

func Test_SubjectForMethod_UnknownMethod_ReturnsError(t *testing.T) {
	_, err := subjectForMethod("nonExistentMethod")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown method name")
}

func Test_PendingTxSubjectForMethod_RoutesCorrectly(t *testing.T) {
	tests := []struct {
		name            string
		method          MethodName
		expectedSubject snx_lib_db_nats.Subject
	}{
		{"RequestWithdrawal routes to relayer pending tx", MethodName_RequestWithdrawal, snx_lib_db_nats.RelayerPendingTxRelayer},
		{"CastWatcherVotes routes to watcher pending tx", MethodName_CastWatcherVotes, snx_lib_db_nats.RelayerPendingTxWatcher},
		{"DisputeWithdrawals routes to watcher pending tx", MethodName_DisputeWithdrawals, snx_lib_db_nats.RelayerPendingTxWatcher},
		{"DisburseWithdrawals routes to teller pending tx", MethodName_DisburseWithdrawals, snx_lib_db_nats.RelayerPendingTxTeller},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, err := pendingTxSubjectForMethod(tt.method)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expectedSubject.String(), subject)
		})
	}
}

func Test_PendingTxSubjectForMethod_UnknownMethod_ReturnsError(t *testing.T) {
	_, err := pendingTxSubjectForMethod("nonExistentMethod")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown method name")
}

func Test_NewTransactionJob_EncodesCalldataAndSetsDefaults(t *testing.T) {
	calldata := []byte{0xde, 0xad, 0xbe, 0xef}
	job := NewTransactionJob(
		MethodName_RequestWithdrawal,
		"0x1234567890123456789012345678901234567890",
		calldata,
		"42",
	)

	assert.Equal(t, MethodName_RequestWithdrawal, job.MethodName)
	assert.Equal(t, "deadbeef", job.Calldata)
	assert.Equal(t, "0", job.Value)
	assert.Equal(t, "42", job.OffchainWithdrawalId)
	assert.Equal(t, "0x1234567890123456789012345678901234567890", string(job.ContractAddress))
}

func Test_NewTransactionJob_EmptyOffchainId(t *testing.T) {
	job := NewTransactionJob(
		MethodName_CastWatcherVotes,
		"0x0000000000000000000000000000000000000001",
		[]byte{0x01},
		"",
	)

	assert.Equal(t, MethodName_CastWatcherVotes, job.MethodName)
	assert.Empty(t, job.OffchainWithdrawalId)
}

func Test_NewTransactionJob_EmptyCalldata(t *testing.T) {
	job := NewTransactionJob(
		MethodName_DisburseWithdrawals,
		"0x0000000000000000000000000000000000000001",
		[]byte{},
		"",
	)

	assert.Equal(t, "", job.Calldata)
}

func Test_TransactionJobMsgID_Deterministic(t *testing.T) {
	job := NewTransactionJob(
		MethodName_RequestWithdrawal,
		"0xContractAddr",
		[]byte{0xde, 0xad},
		"w-1",
	)

	id1 := TransactionJobMsgID(job)
	id2 := TransactionJobMsgID(job)
	assert.Equal(t, id1, id2)
	assert.Equal(t, "txjob:requestWithdrawal:0xContractAddr:dead", id1)
}

func Test_PendingTxMsgID_Deterministic(t *testing.T) {
	msg := PendingTxMessage{
		SignerAddress: "0xSignerAddr",
		Nonce:         42,
		TxHash:        "0xabc123",
		MethodName:    MethodName_RequestWithdrawal,
	}

	id1 := PendingTxMsgID(msg)
	id2 := PendingTxMsgID(msg)
	assert.Equal(t, id1, id2)
	assert.Equal(t, "pending-tx:0xSignerAddr:42:0xabc123", id1)
}

func Test_PendingTxMsgID_DifferentNonces_DifferentIDs(t *testing.T) {
	base := PendingTxMessage{
		SignerAddress: "0xSigner",
		Nonce:         1,
		TxHash:        "0xhash1",
	}
	replaced := base
	replaced.Nonce = 2
	replaced.TxHash = "0xhash2"

	assert.NotEqual(t, PendingTxMsgID(base), PendingTxMsgID(replaced))
}

// ---------------------------------------------------------------------------
// PublishTransactionJob
// ---------------------------------------------------------------------------

func Test_PublishTransactionJob_Success(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	job := NewTransactionJob(
		MethodName_RequestWithdrawal,
		"0xContractAddr",
		[]byte{0xca, 0xfe},
		"w-99",
	)

	var captured *nats.Msg
	js := &mockJetStream{
		publishMsgFn: func(_ context.Context, msg *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			captured = msg
			return &jetstream.PubAck{Stream: "TEST_STREAM", Sequence: 1}, nil
		},
	}

	err := PublishTransactionJob(logger, ctx, js, job)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	expectedSubject := snx_lib_db_nats.RelayerTxnQueueRelayer.String()
	assert.Equal(t, expectedSubject, captured.Subject)

	expectedMsgID := TransactionJobMsgID(job)
	assert.Equal(t, []string{expectedMsgID}, captured.Header["Nats-Msg-Id"])

	var decoded TransactionJob
	require.NoError(t, json.Unmarshal(captured.Data, &decoded))
	assert.Equal(t, job, decoded)
}

func Test_PublishTransactionJob_AllMethods(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	tests := []struct {
		name            string
		method          MethodName
		expectedSubject string
	}{
		{"relayer method", MethodName_RequestWithdrawal, snx_lib_db_nats.RelayerTxnQueueRelayer.String()},
		{"watcher method (votes)", MethodName_CastWatcherVotes, snx_lib_db_nats.RelayerTxnQueueWatcher.String()},
		{"watcher method (dispute)", MethodName_DisputeWithdrawals, snx_lib_db_nats.RelayerTxnQueueWatcher.String()},
		{"teller method", MethodName_DisburseWithdrawals, snx_lib_db_nats.RelayerTxnQueueTeller.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := NewTransactionJob(tt.method, "0xAddr", []byte{0x01}, "")

			var captured *nats.Msg
			js := &mockJetStream{
				publishMsgFn: func(_ context.Context, msg *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
					captured = msg
					return &jetstream.PubAck{Stream: "S", Sequence: 1}, nil
				},
			}

			err := PublishTransactionJob(logger, ctx, js, job)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expectedSubject, captured.Subject)
		})
	}
}

func Test_PublishTransactionJob_UnknownMethod_ReturnsError(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	job := TransactionJob{
		MethodName:      "bogusMethod",
		ContractAddress: "0xAddr",
		Calldata:        "cafe",
		Value:           "0",
	}

	js := &mockJetStream{
		publishMsgFn: func(context.Context, *nats.Msg, ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			t.Fatal("PublishMsg should not be called for unknown method")
			return nil, nil
		},
	}

	err := PublishTransactionJob(logger, ctx, js, job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve subject for bogusMethod")
}

func Test_PublishTransactionJob_PublishFails_ReturnsError(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	job := NewTransactionJob(
		MethodName_DisburseWithdrawals,
		"0xAddr",
		[]byte{0xab},
		"",
	)

	js := &mockJetStream{
		publishMsgFn: func(context.Context, *nats.Msg, ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			return nil, errors.New("connection refused")
		},
	}

	err := PublishTransactionJob(logger, ctx, js, job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish transaction job")
	assert.Contains(t, err.Error(), "connection refused")
}

// ---------------------------------------------------------------------------
// PublishPendingTxMessage
// ---------------------------------------------------------------------------

func Test_PublishPendingTxMessage_Success(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	msg := PendingTxMessage{
		Calldata:             "deadbeef",
		ContractAddress:      "0xContract",
		GasFeeCap:            "30000000000",
		GasLimit:             21000,
		GasTipCap:            "1500000000",
		MethodName:           MethodName_RequestWithdrawal,
		Nonce:                7,
		OffchainWithdrawalId: "w-42",
		SignerAddress:        "0xSigner",
		SubmittedAt:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		TxHash:               "0xtxhash",
	}

	var captured *nats.Msg
	js := &mockJetStream{
		publishMsgFn: func(_ context.Context, m *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			captured = m
			return &jetstream.PubAck{Stream: "PENDING_STREAM", Sequence: 5}, nil
		},
	}

	err := PublishPendingTxMessage(logger, ctx, js, msg)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	expectedSubject := snx_lib_db_nats.RelayerPendingTxRelayer.String()
	assert.Equal(t, expectedSubject, captured.Subject)

	expectedMsgID := PendingTxMsgID(msg)
	assert.Equal(t, []string{expectedMsgID}, captured.Header["Nats-Msg-Id"])

	var decoded PendingTxMessage
	require.NoError(t, json.Unmarshal(captured.Data, &decoded))
	assert.Equal(t, msg, decoded)
}

func Test_PublishPendingTxMessage_AllMethods(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	tests := []struct {
		name            string
		method          MethodName
		expectedSubject string
	}{
		{"relayer method", MethodName_RequestWithdrawal, snx_lib_db_nats.RelayerPendingTxRelayer.String()},
		{"watcher method (votes)", MethodName_CastWatcherVotes, snx_lib_db_nats.RelayerPendingTxWatcher.String()},
		{"watcher method (dispute)", MethodName_DisputeWithdrawals, snx_lib_db_nats.RelayerPendingTxWatcher.String()},
		{"teller method", MethodName_DisburseWithdrawals, snx_lib_db_nats.RelayerPendingTxTeller.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := PendingTxMessage{
				MethodName:    tt.method,
				SignerAddress: "0xSigner",
				Nonce:         1,
				TxHash:        "0xhash",
			}

			var captured *nats.Msg
			js := &mockJetStream{
				publishMsgFn: func(_ context.Context, m *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
					captured = m
					return &jetstream.PubAck{Stream: "S", Sequence: 1}, nil
				},
			}

			err := PublishPendingTxMessage(logger, ctx, js, msg)
			require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.expectedSubject, captured.Subject)
		})
	}
}

func Test_PublishPendingTxMessage_UnknownMethod_ReturnsError(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	msg := PendingTxMessage{
		MethodName:    "bogusMethod",
		SignerAddress: "0xSigner",
		Nonce:         1,
		TxHash:        "0xhash",
	}

	js := &mockJetStream{
		publishMsgFn: func(context.Context, *nats.Msg, ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			t.Fatal("PublishMsg should not be called for unknown method")
			return nil, nil
		},
	}

	err := PublishPendingTxMessage(logger, ctx, js, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve pending-tx subject for bogusMethod")
}

func Test_PublishPendingTxMessage_PublishFails_ReturnsError(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	ctx := context.Background()

	msg := PendingTxMessage{
		MethodName:    MethodName_CastWatcherVotes,
		SignerAddress: "0xSigner",
		Nonce:         3,
		TxHash:        "0xfail",
	}

	js := &mockJetStream{
		publishMsgFn: func(context.Context, *nats.Msg, ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
			return nil, errors.New("timeout")
		},
	}

	err := PublishPendingTxMessage(logger, ctx, js, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish pending tx message")
	assert.Contains(t, err.Error(), "timeout")
}
