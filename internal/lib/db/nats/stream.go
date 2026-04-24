package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// ============================================================================
// Anything related to streams should be contained in this file
//
// Stream names rules:
// Spaces, tabs, period (.), greater than (>) or asterisk (*) are prohibited.
// https://docs.nats.io/running-a-nats-service/nats_admin/jetstream_admin/naming
//
// ============================================================================

type streamNameBase string

type StreamName struct {
	StreamBase streamNameBase
}

func (s StreamName) String() string {
	return makeStreamNameSafe(string(s.StreamBase))
}

func makeStreamNameSafe(streamName string) string {
	prefix := getGlobalPrefixSafe()

	return fmt.Sprintf("%s-%s", prefix, streamName)
}

func makeStreamName(StreamBase streamNameBase) StreamName {
	return StreamName{StreamBase}
}

// Domain streams for general-purpose / support subjects.
const (
	streamNameBase_DLQ streamNameBase = "DLQ" // dead letter queue stream for cross-service failure tracking
)

// Domain streams for account and trading lifecycle subjects.
// Each captures subjects via a js.{stream-name}.> wildcard.
const (
	streamNameBase_AccountLifecycle  streamNameBase = "ACCOUNT_LIFECYCLE"  // deposits, withdrawals, status, fee rates, commands
	streamNameBase_FundingEvents     streamNameBase = "FUNDING_EVENTS"     // funding rate posting and balance settlement
	streamNameBase_LiquidationEvents streamNameBase = "LIQUIDATION_EVENTS" // insurance position, protection, liquidation events
	streamNameBase_ExecutionEvents   streamNameBase = "EXECUTION_EVENTS"   // orders, trades, positions, account state
)

const (
	streamNameBase_ADLRankings      streamNameBase = "ADL_RANKINGS"         // ADL ranking updates from subaccount to trading
	streamNameBase_AccountsTreasury streamNameBase = "ACCOUNTS_TREASURY"    // treasury events between trading and subaccount
	streamNameBase_OrderBookJournal streamNameBase = "ORDERBOOK_JOURNAL"    // matching -> market data
	streamNameBase_Orders           streamNameBase = "ORDERS"               // trading <-> matching
	streamNameBase_RelayerTxnQueue  streamNameBase = "RELAYER_TRANSACTIONS" // relayer on-chain transaction jobs, with per-signer consumers
)

var (
	StreamName_DLQ = makeStreamName(streamNameBase_DLQ)
)

var (
	StreamName_AccountLifecycle  = makeStreamName(streamNameBase_AccountLifecycle)
	StreamName_FundingEvents     = makeStreamName(streamNameBase_FundingEvents)
	StreamName_LiquidationEvents = makeStreamName(streamNameBase_LiquidationEvents)
	StreamName_ExecutionEvents   = makeStreamName(streamNameBase_ExecutionEvents)
)

var (
	StreamName_ADLRankings      = makeStreamName(streamNameBase_ADLRankings)
	StreamName_AccountsTreasury = makeStreamName(streamNameBase_AccountsTreasury)
	StreamName_OrderBookJournal = makeStreamName(streamNameBase_OrderBookJournal)
	StreamName_Orders           = makeStreamName(streamNameBase_Orders)
	StreamName_RelayerTxnQueue  = makeStreamName(streamNameBase_RelayerTxnQueue)
)

func CreateOrdersStreamConfig(numReplicas int) jetstream.StreamConfig {
	return jetstream.StreamConfig{
		MaxAge:    time.Hour,
		MaxBytes:  DefaultMaxBytes,
		MaxMsgs:   DefaultMaxMsgs,
		Name:      StreamName_Orders.String(),
		Replicas:  numReplicas,
		Retention: WorkQueueRetention,
		Storage:   MemoryStorage,
		Subjects:  []string{MakeSubject(string(subjectBase_OrdersStreamWildcard))},
	}
}

func CreateOrderBookJournalStreamConfig(numReplicas int) jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:    StreamName_OrderBookJournal.String(),
		Storage: FileStorage,
		Subjects: []string{
			MakeSubject(string(subjectBase_OrderbookJournalStreamWildcard)),
			MakeSubject(string(subjectBase_TradeEventStreamWildcard)),
		},
		Retention: LimitsRetention,
		MaxMsgs:   DefaultMaxMsgs * 10,
		MaxBytes:  DefaultMaxBytes * 10,
		MaxAge:    MaxAgeDefault,
		Replicas:  numReplicas,
	}
}

func CreateExecutionEventsStreamConfig(numReplicas int) jetstream.StreamConfig {
	return makeStreamConfigWithDefaults(
		StreamName_ExecutionEvents.String(),
		[]string{MakeSubject("js.execution-events.>")},
		WorkQueueRetention,
		numReplicas,
	)
}

func CreateLiquidationEventsStreamConfig(numReplicas int) jetstream.StreamConfig {
	return makeStreamConfigWithDefaults(
		StreamName_LiquidationEvents.String(),
		[]string{MakeSubject("js.liquidation-events.>")},
		LimitsRetention,
		numReplicas,
	)
}

func CreateAccountLifecycleStreamConfig(numReplicas int) jetstream.StreamConfig {
	return makeStreamConfigWithDefaults(
		StreamName_AccountLifecycle.String(),
		[]string{MakeSubject("js.account-lifecycle.>")},
		LimitsRetention,
		numReplicas,
	)
}

func CreateFundingEventsStreamConfig(numReplicas int) jetstream.StreamConfig {
	return makeStreamConfigWithDefaults(
		StreamName_FundingEvents.String(),
		[]string{MakeSubject("js.funding-events.>")},
		LimitsRetention,
		numReplicas,
	)
}

// CreateADLRankingsStreamConfig creates a stream for ADL ranking update events.
// Uses wildcard subject "adl.rankings.updated.*" to capture per-market messages
// (e.g. "adl.rankings.updated.BTC-USDT"). Each message contains both long and short
// rankings for a single market, keeping payload small regardless of total position count
// and avoiding NATS max_payload (1MB default) issues at scale.
// MaxMsgsPerSubject=1 ensures only the latest ranking per market is retained.
// Uses LimitsRetention - the trading service subscribes with DeliverLastPerSubjectPolicy
// to get the latest ranking for each market on startup.
func CreateADLRankingsStreamConfig(numReplicas int) jetstream.StreamConfig {
	subjects := []string{MakeSubjectForADLRankingWildcard()}
	return jetstream.StreamConfig{
		Name:              StreamName_ADLRankings.String(),
		Subjects:          subjects,
		Storage:           FileStorage,
		Retention:         LimitsRetention,
		MaxConsumers:      Unlimited,
		MaxMsgs:           DefaultMaxMsgs,
		MaxMsgsPerSubject: 1,
		MaxBytes:          DefaultMaxBytes,
		MaxAge:            MaxAgeDefault,
		Replicas:          numReplicas,
		NoAck:             false,
	}
}

// Creates the DLQ stream for dead letter queue entries. Uses FileStorage
// for durability and LimitsRetention so entries remain available for
// investigation. MaxAge is 7 days — substantially longer than operational
// streams — because DLQ entries require human attention.
func CreateDLQStreamConfig(numReplicas int) jetstream.StreamConfig {
	return jetstream.StreamConfig{
		MaxAge:       7 * 24 * time.Hour,
		MaxBytes:     DefaultMaxBytes,
		MaxConsumers: Unlimited,
		MaxMsgs:      DefaultMaxMsgs,
		Name:         StreamName_DLQ.String(),
		NoAck:        false,
		Replicas:     numReplicas,
		Retention:    LimitsRetention,
		Storage:      FileStorage,
		Subjects:     SubjectsToSubjectsStringArray(SystemDLQPosted),
	}
}

// Creates the relayer transaction queue stream config.
// This shared stream carries both transaction submission jobs and the pending
// transaction subjects that will be monitored in a later phase.
func CreateRelayerTxnQueueStreamConfig(numReplicas int) jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:         StreamName_RelayerTxnQueue.String(),
		Subjects:     []string{MakeSubject("relayer.txn.>")},
		Storage:      FileStorage,
		Retention:    WorkQueueRetention,
		MaxConsumers: Unlimited,
		Replicas:     numReplicas,
	}
}

func CreateAccountsTreasuryStreamConfig(numReplicas int) jetstream.StreamConfig {
	// TODO: move withdrawal and deposit events here
	subjects := []string{
		MakeSubject(fmt.Sprintf("%s.>", streamNameBase_AccountsTreasury)),
	}

	return makeStreamConfigWithDefaults(StreamName_AccountsTreasury.String(), subjects, WorkQueueRetention, numReplicas)
}

const (
	DefaultMaxMsgs  = 100_000
	DefaultMaxBytes = 100 * 1024 * 1024
)

func makeStreamConfigWithDefaults(
	streamName string,
	subjects []string,
	retention jetstream.RetentionPolicy,
	numReplicas int,
) jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:         streamName,
		Subjects:     subjects,
		Storage:      FileStorage,
		Retention:    retention,
		MaxConsumers: Unlimited,
		MaxMsgs:      DefaultMaxMsgs,
		MaxBytes:     DefaultMaxBytes,
		MaxAge:       MaxAgeDefault,
		Replicas:     numReplicas,
		NoAck:        false,
	}
}
