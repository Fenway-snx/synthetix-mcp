package nats

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// ============================================================================
// https://docs.nats.io/nats-concepts/jetstream/consumers
// Consumer Name == Durable
// A durable name cannot contain whitespace, ., *, >
// as per nats docs:
// In addition to the choice of being push or pull, a consumer can also be ephemeral or durable.
// A consumer is considered durable when an explicit name is set on the Durable field when creating
// the consumer, or when InactiveThreshold is set.

// To keep the existing pattern we implicity replace dots(.) with dashes(-)

// Pattern: [service]-[subject]

// To save some logic we are reusing some code from the topics logic

// ============================================================================

type ConsumerNameBase string

type ConsumerName struct {
	Base ConsumerNameBase
}

func (s ConsumerName) String() string {
	return makeConsumerNameSafe(string(s.Base))
}

func makeConsumerNameSafe(consumerName string) string {
	prefix := getGlobalPrefixSafe()

	parsedConsumerName := strings.ReplaceAll(consumerName, ".", "-")

	for strings.HasSuffix(parsedConsumerName, "-") {
		parsedConsumerName = strings.TrimSuffix(parsedConsumerName, "-")
	}
	for strings.HasPrefix(parsedConsumerName, "-") {
		parsedConsumerName = strings.TrimPrefix(parsedConsumerName, "-")
	}

	return fmt.Sprintf("%s-%s", prefix, parsedConsumerName)
}

func makeConsumerName(consumerNameBase ConsumerNameBase) ConsumerName {
	return ConsumerName{Base: consumerNameBase}
}

const (
	consumerName_CandlestickTradePerSymbol       ConsumerNameBase = "candlestick-trade.%s"  // candlestick-trade-{symbol} e.g. candlestick-trade-BTC-USDT
	consumerName_MarketDataJournalPerSymbol      ConsumerNameBase = "marketdata-journal.%s" // marketdata-journal-{symbol} e.g. marketdata-journal-BTC-USDT
	consumerName_MarketDataTradePerSymbol        ConsumerNameBase = "marketdata-trade.%s"   // marketdata-trade-{symbol} e.g. marketdata-trade-BTC-USDT
	consumerName_RelayerPendingTxRelayerConsumer ConsumerNameBase = "relayer-pending-tx-relayer-consumer"
	consumerName_RelayerPendingTxTellerConsumer  ConsumerNameBase = "relayer-pending-tx-teller-consumer"
	consumerName_RelayerPendingTxWatcherConsumer ConsumerNameBase = "relayer-pending-tx-watcher-consumer"
	consumerName_RelayerTxnQueueConsumer         ConsumerNameBase = "relayer-txn-queue-consumer"
	consumerName_RelayerTxnQueueTellerConsumer   ConsumerNameBase = "relayer-txn-queue-teller-consumer"
	consumerName_RelayerTxnQueueWatcherConsumer  ConsumerNameBase = "relayer-txn-queue-watcher-consumer"
	consumerName_TradingADLRankings              ConsumerNameBase = "trading-adl-rankings"
	consumerName_TradingDepositProcessed         ConsumerNameBase = "trading-deposit-processed"
	consumerName_TradingFeeRateUpdate            ConsumerNameBase = "trading-fee-rate-update"
	consumerName_TradingFundingRateBalanceUpdate ConsumerNameBase = "trading-funding-rate-balance-update"
	consumerName_TradingNConsurmersPerSymbol     ConsumerNameBase = "trading.%s" // trading-{subject} e.g. trading-BTC-USDT-N plus the prefixes
	consumerName_TradingSubaccountStatusUpdate   ConsumerNameBase = "trading-subaccount-status-update"
)

var (
	ConsumerNameRelayerPendingTxRelayer = makeConsumerName(consumerName_RelayerPendingTxRelayerConsumer)
	ConsumerNameRelayerPendingTxTeller  = makeConsumerName(consumerName_RelayerPendingTxTellerConsumer)
	ConsumerNameRelayerPendingTxWatcher = makeConsumerName(consumerName_RelayerPendingTxWatcherConsumer)

	ConsumerNameRelayerTxnQueue        = makeConsumerName(consumerName_RelayerTxnQueueConsumer)
	ConsumerNameRelayerTxnQueueTeller  = makeConsumerName(consumerName_RelayerTxnQueueTellerConsumer)
	ConsumerNameRelayerTxnQueueWatcher = makeConsumerName(consumerName_RelayerTxnQueueWatcherConsumer)

	MakeConsumerNameForTradingADLRankings              = makeConsumerName(consumerName_TradingADLRankings)
	MakeConsumerNameForTradingDepositProcessed         = makeConsumerName(consumerName_TradingDepositProcessed)
	MakeConsumerNameForTradingFeeRateUpdate            = makeConsumerName(consumerName_TradingFeeRateUpdate)
	MakeConsumerNameForTradingFundingRateBalanceUpdate = makeConsumerName(consumerName_TradingFundingRateBalanceUpdate)
	MakeConsumerNameForTradingSubaccountStatusUpdate   = makeConsumerName(consumerName_TradingSubaccountStatusUpdate)
)

func MakeConsumerNameForTradingConsumers(subject string) string {
	return makeConsumerName(ConsumerNameBase(fmt.Sprintf(string(consumerName_TradingNConsurmersPerSymbol), subject))).String()
}

func MakeConsumerNameForMarketDataJournal(symbol string) string {
	return makeConsumerName(ConsumerNameBase(fmt.Sprintf(string(consumerName_MarketDataJournalPerSymbol), symbol))).String()
}

func MakeConsumerNameForMarketDataTrade(symbol string) string {
	return makeConsumerName(ConsumerNameBase(fmt.Sprintf(string(consumerName_MarketDataTradePerSymbol), symbol))).String()
}

func MakeConsumerNameForCandlestickTrade(symbol string) string {
	return makeConsumerName(ConsumerNameBase(fmt.Sprintf(string(consumerName_CandlestickTradePerSymbol), symbol))).String()
}

func MakeConsumerConfigWithDefaults(consumerName, subject string) jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       AckWaitDefault,
		MaxDeliver:    Unlimited,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
		MaxAckPending: Unlimited,
	}
}

// Creates a consumer config with specified deliver policy
// deliverPolicy: jetstream.DeliverAllPolicy, DeliverNewPolicy, DeliverLastPolicy, etc.
// Note: If you don't want to set a policy, just use MakeConsumerConfigWithDefaults()
func MakeConsumerConfig(consumerName, subject string, deliverPolicy jetstream.DeliverPolicy) jetstream.ConsumerConfig {
	config := MakeConsumerConfigWithDefaults(consumerName, subject)
	config.DeliverPolicy = deliverPolicy
	return config
}

// Per-signer consumer configs for the relayer transaction queue.
// Configured for sequential processing of on-chain transactions:
// - MaxAckPending: 1 - only one job at a time
// - AckWait: 7 minutes - on-chain txns can take time, with buffer
// - MaxDeliver: 1 - no retries, one shot
// - DeliverAll - process any pending jobs on startup
const MaxProcessTimeWithBuffer = 7 * time.Minute

func relayerBaseConsumerConfig(consumerName ConsumerName) jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Durable:         consumerName.String(),
		AckPolicy:       jetstream.AckExplicitPolicy,
		AckWait:         MaxProcessTimeWithBuffer,
		MaxDeliver:      1,
		DeliverPolicy:   jetstream.DeliverAllPolicy,
		ReplayPolicy:    jetstream.ReplayInstantPolicy,
		MaxAckPending:   1,
		MaxRequestBatch: 1,
	}
}

// Catches both the legacy subject (relayer.txn.queue) and the role-specific
// subject (relayer.txn.queue.relayer) so the relayer consumer acts as the default route.
// Uses FilterSubjects (not FilterSubject) — NATS 2.10+ rejects configs with both fields present.
func CreateRelayerTxnQueueRelayerConsumerConfig() jetstream.ConsumerConfig {
	cfg := relayerBaseConsumerConfig(ConsumerNameRelayerTxnQueue)
	cfg.FilterSubjects = []string{RelayerTxnQueue.String(), RelayerTxnQueueRelayer.String()}
	return cfg
}

func CreateRelayerTxnQueueTellerConsumerConfig() jetstream.ConsumerConfig {
	cfg := relayerBaseConsumerConfig(ConsumerNameRelayerTxnQueueTeller)
	cfg.FilterSubject = RelayerTxnQueueTeller.String()
	return cfg
}

func CreateRelayerTxnQueueWatcherConsumerConfig() jetstream.ConsumerConfig {
	cfg := relayerBaseConsumerConfig(ConsumerNameRelayerTxnQueueWatcher)
	cfg.FilterSubject = RelayerTxnQueueWatcher.String()
	return cfg
}

// Consumer config for pending transaction monitoring.
// The relayer service calls this once per signer, providing its own name and subject.
// - DeliverAll: replays unacked messages on crash recovery
// - AckWait 10min: generous timeout for on-chain confirmation monitoring
// - MaxAckPending 1: process one nonce at a time per signer
const PendingTxAckWait = 10 * time.Minute

func CreateRelayerPendingTxConsumerConfig(consumerName ConsumerName, filterSubject Subject) jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Durable:       consumerName.String(),
		FilterSubject: filterSubject.String(),
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       PendingTxAckWait,
		MaxDeliver:    Unlimited,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
		MaxAckPending: 1,
	}
}
