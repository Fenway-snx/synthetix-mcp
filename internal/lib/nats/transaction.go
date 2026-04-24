package nats

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// MethodName identifies the contract method being called
type MethodName string

const (
	MethodName_CastWatcherVotes    MethodName = "castWatcherVotes"
	MethodName_DisburseWithdrawals MethodName = "disburseWithdrawals"
	MethodName_DisputeWithdrawals  MethodName = "disputeWithdrawals"
	MethodName_RequestWithdrawal   MethodName = "requestWithdrawal"
)

// Identifies which on-chain signer handles a given method.
// Single source of truth for the method→role mapping, consumed by both
// the publish side (NATS subject routing) and the consume side (signer
// selection in the transaction manager).
type SignerRole string

const (
	SignerRole_Relayer SignerRole = "relayer"
	SignerRole_Teller  SignerRole = "teller"
	SignerRole_Watcher SignerRole = "watcher"
)

// Returns the signer role responsible for the given contract method.
func RoleForMethod(method MethodName) (SignerRole, error) {
	switch method {
	case MethodName_RequestWithdrawal:
		return SignerRole_Relayer, nil
	case MethodName_CastWatcherVotes, MethodName_DisputeWithdrawals:
		return SignerRole_Watcher, nil
	case MethodName_DisburseWithdrawals:
		return SignerRole_Teller, nil
	default:
		return "", fmt.Errorf("unknown method name: %s", method)
	}
}

// TransactionJob represents an on-chain transaction job published to the relayer queue.
// Relayer components (teller, watcher, disburser) publish jobs, and the transaction manager consumes them.
type TransactionJob struct {
	// MethodName identifies the contract method being called (for logging/routing)
	MethodName MethodName `json:"method_name"`

	// ContractAddress is the target contract address
	ContractAddress snx_lib_core.WalletAddress `json:"contract_address"`

	// Calldata is the hex-encoded transaction calldata (without 0x prefix)
	Calldata string `json:"calldata"`

	// Value is the ETH value to send with the transaction (in wei, as string)
	// Use "0" for transactions that don't transfer ETH
	Value string `json:"value"`

	// OffchainWithdrawalId is the offchain withdrawal ID (only for requestWithdrawal transactions)
	// Stored as string for JSON serialization. Used to publish failure/success events.
	OffchainWithdrawalId string `json:"offchain_withdrawal_id,omitempty"`
}

// NewTransactionJob creates a new TransactionJob with the given parameters.
// calldata should be the raw bytes (not hex encoded) - this function will encode it.
// Value is hardcoded to "0" as all transactions in this system don't transfer ETH.
// offchainWithdrawalId is optional - only needed for requestWithdrawal transactions.
func NewTransactionJob(
	methodName MethodName,
	contractAddress string,
	calldata []byte,
	offchainWithdrawalId string,
) TransactionJob {
	return TransactionJob{
		MethodName:           methodName,
		ContractAddress:      snx_lib_core.WalletAddress(contractAddress),
		Calldata:             hex.EncodeToString(calldata),
		Value:                "0",
		OffchainWithdrawalId: offchainWithdrawalId,
	}
}

// Deterministic dedup key for JetStream publisher-side deduplication.
// Format: txjob:{method}:{contract}:{calldata}
func TransactionJobMsgID(job TransactionJob) string {
	return fmt.Sprintf("txjob:%s:%s:%s", job.MethodName, job.ContractAddress, job.Calldata)
}

// Routes a transaction job to the correct per-signer subject based on method name.
func subjectForMethod(method MethodName) (string, error) {
	role, err := RoleForMethod(method)
	if err != nil {
		return "", err
	}

	switch role {
	case SignerRole_Relayer:
		return snx_lib_db_nats.RelayerTxnQueueRelayer.String(), nil
	case SignerRole_Teller:
		return snx_lib_db_nats.RelayerTxnQueueTeller.String(), nil
	case SignerRole_Watcher:
		return snx_lib_db_nats.RelayerTxnQueueWatcher.String(), nil
	default:
		return "", fmt.Errorf("no subject configured for role: %s", role)
	}
}

// Publishes a transaction job to the per-signer relayer transaction queue.
func PublishTransactionJob(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	js jetstream.JetStream,
	job TransactionJob,
) error {
	data, err := json.Marshal(job)
	if err != nil {
		logger.Error("Failed to marshal transaction job",
			"calldata", "0x"+job.Calldata,
			"contract_address", job.ContractAddress,
			"error", err,
			"method_name", job.MethodName,
			"value", job.Value,
		)
		return fmt.Errorf("marshal transaction job: %w", err)
	}

	subject, err := subjectForMethod(job.MethodName)
	if err != nil {
		logger.Error("Failed to resolve subject for transaction job",
			"error", err,
			"method_name", job.MethodName,
		)
		return fmt.Errorf("resolve subject for %s: %w", job.MethodName, err)
	}

	msgID := TransactionJobMsgID(job)
	ack, err := js.PublishMsg(ctx, &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{"Nats-Msg-Id": []string{msgID}},
	})
	if err != nil {
		logger.Error("Failed to publish transaction job to queue",
			"calldata", "0x"+job.Calldata,
			"contract_address", job.ContractAddress,
			"error", err,
			"method_name", job.MethodName,
			"msg_id", msgID,
			"subject", subject,
			"value", job.Value,
		)
		return fmt.Errorf("publish transaction job: %w", err)
	}

	logger.Info("Published transaction job to queue",
		"calldata", "0x"+job.Calldata,
		"contract_address", job.ContractAddress,
		"method_name", job.MethodName,
		"msg_id", msgID,
		"sequence", ack.Sequence,
		"stream", ack.Stream,
		"subject", subject,
		"value", job.Value,
	)

	return nil
}

// PendingTxMessage represents a submitted-but-unconfirmed on-chain transaction.
// Published after SendTransaction succeeds so the pending-tx monitor can track confirmation.
type PendingTxMessage struct {
	Calldata             string                     `json:"calldata"`
	ContractAddress      snx_lib_core.WalletAddress `json:"contract_address"`
	GasFeeCap            string                     `json:"gas_fee_cap"` // base-10 decimal string in wei (big.Int.String() format)
	GasLimit             uint64                     `json:"gas_limit"`
	GasTipCap            string                     `json:"gas_tip_cap"` // base-10 decimal string in wei (big.Int.String() format)
	MethodName           MethodName                 `json:"method_name"`
	Nonce                uint64                     `json:"nonce"`
	OffchainWithdrawalId string                     `json:"offchain_withdrawal_id,omitempty"`
	SignerAddress        snx_lib_core.WalletAddress `json:"signer_address"`
	SubmittedAt          time.Time                  `json:"submitted_at"`
	TxHash               string                     `json:"tx_hash"`
}

// Deterministic dedup key for JetStream publisher-side deduplication.
// Format: pending-tx:{signer}:{nonce}:{tx_hash}
func PendingTxMsgID(msg PendingTxMessage) string {
	return fmt.Sprintf("pending-tx:%s:%d:%s", msg.SignerAddress, msg.Nonce, msg.TxHash)
}

// Routes a pending-tx message to the correct per-signer subject based on method name.
func pendingTxSubjectForMethod(method MethodName) (string, error) {
	role, err := RoleForMethod(method)
	if err != nil {
		return "", err
	}

	switch role {
	case SignerRole_Relayer:
		return snx_lib_db_nats.RelayerPendingTxRelayer.String(), nil
	case SignerRole_Teller:
		return snx_lib_db_nats.RelayerPendingTxTeller.String(), nil
	case SignerRole_Watcher:
		return snx_lib_db_nats.RelayerPendingTxWatcher.String(), nil
	default:
		return "", fmt.Errorf("no pending-tx subject for role: %s", role)
	}
}

// Publishes a pending transaction message to the per-signer pending-tx
// subjects on the shared relayer transaction stream.
func PublishPendingTxMessage(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	js jetstream.JetStream,
	msg PendingTxMessage,
) error {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("Failed to marshal pending tx message",
			"error", err,
			"method_name", msg.MethodName,
			"nonce", msg.Nonce,
			"signer_address", msg.SignerAddress,
			"tx_hash", msg.TxHash,
		)
		return fmt.Errorf("marshal pending tx message: %w", err)
	}

	subject, err := pendingTxSubjectForMethod(msg.MethodName)
	if err != nil {
		logger.Error("Failed to resolve subject for pending tx message",
			"error", err,
			"method_name", msg.MethodName,
		)
		return fmt.Errorf("resolve pending-tx subject for %s: %w", msg.MethodName, err)
	}

	msgID := PendingTxMsgID(msg)
	ack, err := js.PublishMsg(ctx, &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{"Nats-Msg-Id": []string{msgID}},
	})
	if err != nil {
		logger.Error("Failed to publish pending tx message",
			"error", err,
			"method_name", msg.MethodName,
			"msg_id", msgID,
			"nonce", msg.Nonce,
			"signer_address", msg.SignerAddress,
			"subject", subject,
			"tx_hash", msg.TxHash,
		)
		return fmt.Errorf("publish pending tx message: %w", err)
	}

	logger.Info("Published pending tx message",
		"method_name", msg.MethodName,
		"msg_id", msgID,
		"nonce", msg.Nonce,
		"sequence", ack.Sequence,
		"signer_address", msg.SignerAddress,
		"stream", ack.Stream,
		"subject", subject,
		"tx_hash", msg.TxHash,
	)

	return nil
}
