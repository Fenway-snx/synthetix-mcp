package nats

import (
	"fmt"
	"os"
	"strings"
	"sync"

	snx_lib_config "github.com/Fenway-snx/synthetix-mcp/internal/lib/config"
)

// ============================================================================
// See lib/db/nats/README.md for more details
//
// JetStream subjects follow:  js.{stream}[.{group}...].{designator}
//   - js            = identifies the subject as routed through JetStream
//   - {stream}      = the stream name (e.g. execution-events, account-lifecycle)
//   - {group}       = optional logical grouping within the stream; can be multiple levels deep
//   - {designator}  = specific event / action
//
// Non-JetStream subjects (core NATS pub/sub) keep legacy patterns:
//   [service].event.[entity].[state_change]
//   [service].command.[action].[entity]
//   [service].query.[entity]
//   marketdata.[channel].[symbol]
//   sys.[component].[event_type]
// ============================================================================

// NOTE: the use of a strong type is primarily to ensure that the propagation
// of this change is under the purview of the compiler.

type SubjectBase string

type Subject struct {
	SubjectBase SubjectBase
}

func makeSubject(subjectBase SubjectBase) Subject {
	return Subject{SubjectBase: subjectBase}
}

// Expresses a `Subject` as a string, fully-qualified by the global prefix
//
// Note: in the current implementation this has a small cost involving
// string concatenation.
func (s Subject) String() string {
	return MakeSubject(string(s.SubjectBase))
}

// relayer internal queue subjects
const (
	// Internal work queue for relayer on-chain transaction jobs
	// Relayer components enqueue jobs, transaction manager consumes them
	subjectBase_RelayerTxnQueue        SubjectBase = "relayer.txn.queue" // Default/catch-all subject, routed to the relayer consumer
	subjectBase_RelayerTxnQueueRelayer SubjectBase = "relayer.txn.queue.relayer"
	subjectBase_RelayerTxnQueueTeller  SubjectBase = "relayer.txn.queue.teller"
	subjectBase_RelayerTxnQueueWatcher SubjectBase = "relayer.txn.queue.watcher"
)

var (
	RelayerTxnQueue        = makeSubject(subjectBase_RelayerTxnQueue)
	RelayerTxnQueueRelayer = makeSubject(subjectBase_RelayerTxnQueueRelayer)
	RelayerTxnQueueTeller  = makeSubject(subjectBase_RelayerTxnQueueTeller)
	RelayerTxnQueueWatcher = makeSubject(subjectBase_RelayerTxnQueueWatcher)
)

// relayer pending transaction tracking subjects
const (
	subjectBase_RelayerPendingTxRelayer SubjectBase = "relayer.txn.pending.relayer"
	subjectBase_RelayerPendingTxTeller  SubjectBase = "relayer.txn.pending.teller"
	subjectBase_RelayerPendingTxWatcher SubjectBase = "relayer.txn.pending.watcher"
)

var (
	RelayerPendingTxRelayer = makeSubject(subjectBase_RelayerPendingTxRelayer)
	RelayerPendingTxTeller  = makeSubject(subjectBase_RelayerPendingTxTeller)
	RelayerPendingTxWatcher = makeSubject(subjectBase_RelayerPendingTxWatcher)
)

// JetStream subjects -- naming convention: js.{stream}[.{group}...].{designator}
//
// EXECUTION_EVENTS stream (File, WorkQueue): trading lifecycle events
const (
	subjectBase_TradingEventAccountStateUpdated  SubjectBase = "js.execution-events.account.state-updated"
	subjectBase_TradingEventAtomicPositionUpdate SubjectBase = "js.execution-events.position.atomic-update"
	subjectBase_TradingEventOpenOrderCreated     SubjectBase = "js.execution-events.order.open-created"
	subjectBase_TradingEventOrderHistory         SubjectBase = "js.execution-events.order.history"
	subjectBase_TradingEventPositionTPSL         SubjectBase = "js.execution-events.position.tpsl"
	subjectBase_TradingEventTradeHistory         SubjectBase = "js.execution-events.trade.history"
)

// LIQUIDATION_EVENTS stream (File, Limits): insurance and liquidation events for risk/audit
const (
	subjectBase_TradingEventInsurancePositionIncreased   SubjectBase = "js.liquidation-events.position.increased"
	subjectBase_TradingEventInsuranceProtectionActivated SubjectBase = "js.liquidation-events.protection.activated"
	subjectBase_TradingEventInsuranceProtectionCompleted SubjectBase = "js.liquidation-events.protection.completed"
	subjectBase_TradingEventLiquidationCompleted         SubjectBase = "js.liquidation-events.liquidation.completed"
	subjectBase_TradingEventSubaccountTakeover           SubjectBase = "js.liquidation-events.takeover.completed"
)

// ACCOUNT_LIFECYCLE stream (File, Limits): durable account events and commands
const (
	subjectBase_RelayerCommandCowOrder               SubjectBase = "js.account-lifecycle.relayer.coworder"
	subjectBase_RelayerCommandWithdraw               SubjectBase = "js.account-lifecycle.withdrawal.relayer-command"
	subjectBase_RelayerEventDeposit                  SubjectBase = "js.account-lifecycle.deposit.received"
	subjectBase_RelayerEventSnaxpotPurchase          SubjectBase = "js.account-lifecycle.snaxpot.purchase"
	subjectBase_RelayerEventWithdrawCompleted        SubjectBase = "js.account-lifecycle.withdrawal.completed"
	subjectBase_RelayerEventWithdrawSubmissionFailed SubjectBase = "js.account-lifecycle.withdrawal.submission-failed"
	subjectBase_SubaccountCommandWithdraw            SubjectBase = "js.account-lifecycle.withdrawal.command"
	subjectBase_SubaccountEventDepositProcessed      SubjectBase = "js.account-lifecycle.deposit.processed"
	subjectBase_SubaccountEventStatusUpdated         SubjectBase = "js.account-lifecycle.status.updated"
	subjectBase_SubaccountFeeRateUpdate              SubjectBase = "js.account-lifecycle.fee-rate.updated"
	subjectBase_TradingEventWithdrawProcessed        SubjectBase = "js.account-lifecycle.withdrawal.processed"
	subjectBase_TradingEventWithdrawRequest          SubjectBase = "js.account-lifecycle.withdrawal.request"
)

// Legacy subjects -- not yet routed through JetStream domain streams.
// TODO(cleanup): remove these once the remaining migrations are complete.
// These have no active publisher and are retained for compile compatibility.
const (
	subjectBase_RelayerEventWithdrawDisputed SubjectBase = "relayer.event.withdraw.disputed"
	subjectBase_SubaccountEventCreate        SubjectBase = "subaccount.event.create"
	subjectBase_TradingEventPositionCreated  SubjectBase = "trading.event.position.created"
)

// Non-JetStream event subjects (core NATS pub/sub only)
const (
	subjectBase_AccountsDeposit                           SubjectBase = "accounts.deposit"
	subjectBase_AdlRankingsUpdated                        SubjectBase = "adl.rankings.updated"
	subjectBase_RelayerEventWithdrawStatusNotification    SubjectBase = "relayer.event.withdraw.status.notification"
	subjectBase_RelayerQueryWithdrawalExpiredStatus       SubjectBase = "relayer.query.withdrawal.expired.status"
	subjectBase_SubaccountEventDelegationAdded            SubjectBase = "subaccount.event.delegation.added"
	subjectBase_SubaccountEventDelegationRevoked          SubjectBase = "subaccount.event.delegation.revoked"
	subjectBase_SubaccountEventTransferRequest            SubjectBase = "subaccount.event.transfer.request"
	subjectBase_SubaccountEventWithdrawalFinalized        SubjectBase = "subaccount.event.withdrawal.finalized"
	subjectBase_SubaccountQueryAssignLiquidator           SubjectBase = "subaccount.query.assign.liquidator"
	subjectBase_SubaccountQueryCreate                     SubjectBase = "subaccount.query.create"
	subjectBase_SubaccountQueryUpdateLeverage             SubjectBase = "subaccount.query.update.leverage"
	subjectBase_TradingEventFundingHistory                SubjectBase = "trading.event.funding.history"
	subjectBase_TradingEventLiquidationFeeSettlement      SubjectBase = "trading.event.liquidation.fee.settlement"
	subjectBase_TradingEventLiquidationNotification       SubjectBase = "trading.event.liquidation.notification"
	subjectBase_TradingEventSLPCollateralTransfer         SubjectBase = "trading.event.slp.collateral.transfer"
	subjectBase_TradingEventCollateralExchangeBroadcast   SubjectBase = "trading.event.collateral.exchange.broadcast"
	subjectBase_TradingEventSLPTakeoverBroadcast          SubjectBase = "trading.event.slp.takeover.broadcast"
	subjectBase_TradingEventVoluntaryAutoExchangeRequest  SubjectBase = "trading.event.voluntary.autoexchange.request"
	subjectBase_TradingEventVoluntaryAutoExchangeResponse SubjectBase = "trading.event.voluntary.autoexchange.response"
	subjectBase_TradingEventWithdrawActorUpdated          SubjectBase = "trading.event.withdraw.actor.updated"
)

func addStreamPrefix(streamPrefix, subject string) SubjectBase {
	return SubjectBase(fmt.Sprintf("%s.%s", streamPrefix, subject))
}

// Subject base with stream prefix
var (
	subjectBase_AccountsTreasury_TradingEventTransferHistory SubjectBase = addStreamPrefix(string(streamNameBase_AccountsTreasury), "trading.transfer")
)

var (
	AccountsDeposit                              = makeSubject(subjectBase_AccountsDeposit)
	AdlRankingsUpdated                           = makeSubject(subjectBase_AdlRankingsUpdated)
	RelayerCommandCowOrder                       = makeSubject(subjectBase_RelayerCommandCowOrder)
	RelayerCommandWithdraw                       = makeSubject(subjectBase_RelayerCommandWithdraw)
	RelayerEventDeposit                          = makeSubject(subjectBase_RelayerEventDeposit)
	RelayerEventSnaxpotPurchase                  = makeSubject(subjectBase_RelayerEventSnaxpotPurchase)
	RelayerEventWithdrawCompleted                = makeSubject(subjectBase_RelayerEventWithdrawCompleted)
	RelayerEventWithdrawDisputed                 = makeSubject(subjectBase_RelayerEventWithdrawDisputed)
	RelayerEventWithdrawStatusNotification       = makeSubject(subjectBase_RelayerEventWithdrawStatusNotification)
	RelayerEventWithdrawSubmissionFailed         = makeSubject(subjectBase_RelayerEventWithdrawSubmissionFailed)
	RelayerQueryWithdrawalExpiredStatus          = makeSubject(subjectBase_RelayerQueryWithdrawalExpiredStatus)
	SubaccountCommandWithdraw                    = makeSubject(subjectBase_SubaccountCommandWithdraw)
	SubaccountEventCreate                        = makeSubject(subjectBase_SubaccountEventCreate)
	SubaccountEventDelegationAdded               = makeSubject(subjectBase_SubaccountEventDelegationAdded)
	SubaccountEventDelegationRevoked             = makeSubject(subjectBase_SubaccountEventDelegationRevoked)
	SubaccountEventDepositProcessed              = makeSubject(subjectBase_SubaccountEventDepositProcessed)
	SubaccountEventStatusUpdated                 = makeSubject(subjectBase_SubaccountEventStatusUpdated)
	SubaccountEventTransferRequest               = makeSubject(subjectBase_SubaccountEventTransferRequest)
	SubaccountEventWithdrawalFinalized           = makeSubject(subjectBase_SubaccountEventWithdrawalFinalized)
	SubaccountFeeRateUpdate                      = makeSubject(subjectBase_SubaccountFeeRateUpdate)
	SubaccountQueryAssignLiquidator              = makeSubject(subjectBase_SubaccountQueryAssignLiquidator)
	SubaccountQueryCreate                        = makeSubject(subjectBase_SubaccountQueryCreate)
	SubaccountQueryUpdateLeverage                = makeSubject(subjectBase_SubaccountQueryUpdateLeverage)
	TradingEventAccountStateUpdated              = makeSubject(subjectBase_TradingEventAccountStateUpdated)
	TradingEventAtomicPositionUpdate             = makeSubject(subjectBase_TradingEventAtomicPositionUpdate)
	TradingEventCollateralExchangeBroadcast      = makeSubject(subjectBase_TradingEventCollateralExchangeBroadcast)
	TradingEventFundingHistory                   = makeSubject(subjectBase_TradingEventFundingHistory)
	TradingEventInsurancePositionIncreased       = makeSubject(subjectBase_TradingEventInsurancePositionIncreased)
	TradingEventInsuranceProtectionActivated     = makeSubject(subjectBase_TradingEventInsuranceProtectionActivated)
	TradingEventInsuranceProtectionCompleted     = makeSubject(subjectBase_TradingEventInsuranceProtectionCompleted)
	TradingEventLiquidationCompleted             = makeSubject(subjectBase_TradingEventLiquidationCompleted)
	TradingEventLiquidationFeeSettlement         = makeSubject(subjectBase_TradingEventLiquidationFeeSettlement)
	TradingEventLiquidationNotification          = makeSubject(subjectBase_TradingEventLiquidationNotification)
	TradingEventOpenOrderCreated                 = makeSubject(subjectBase_TradingEventOpenOrderCreated)
	TradingEventOrderHistory                     = makeSubject(subjectBase_TradingEventOrderHistory)
	TradingEventPositionCreated                  = makeSubject(subjectBase_TradingEventPositionCreated)
	TradingEventPositionTPSL                     = makeSubject(subjectBase_TradingEventPositionTPSL)
	TradingEventSLPCollateralTransfer            = makeSubject(subjectBase_TradingEventSLPCollateralTransfer)
	TradingEventSLPTakeoverBroadcast             = makeSubject(subjectBase_TradingEventSLPTakeoverBroadcast)
	TradingEventSubaccountTakeover               = makeSubject(subjectBase_TradingEventSubaccountTakeover)
	TradingEventTradeHistory                     = makeSubject(subjectBase_TradingEventTradeHistory)
	AccountsTreasury_TradingEventTransferHistory = makeSubject(subjectBase_AccountsTreasury_TradingEventTransferHistory)
	TradingEventVoluntaryAutoExchangeRequest     = makeSubject(subjectBase_TradingEventVoluntaryAutoExchangeRequest)
	TradingEventVoluntaryAutoExchangeResponse    = makeSubject(subjectBase_TradingEventVoluntaryAutoExchangeResponse)
	TradingEventWithdrawActorUpdated             = makeSubject(subjectBase_TradingEventWithdrawActorUpdated)
	TradingEventWithdrawProcessed                = makeSubject(subjectBase_TradingEventWithdrawProcessed)
	TradingEventWithdrawRequest                  = makeSubject(subjectBase_TradingEventWithdrawRequest)
)

// FUNDING_EVENTS stream (File, Limits): funding rate publications and balance settlement
const (
	subjectBase_FundingRateBalanceUpdate SubjectBase = "js.funding-events.balance.update"
	subjectBase_FundingRatePosted        SubjectBase = "js.funding-events.rate.posted"
)

var (
	FundingRatePosted        = makeSubject(subjectBase_FundingRatePosted)
	FundingRateBalanceUpdate = makeSubject(subjectBase_FundingRateBalanceUpdate)
)

// Market configuration events subjects
const (
	subjectBase_MarketEventConfig SubjectBase = "market.event.config"
)

var (
	// Pattern: market.event.[action]
	MarketEventConfig = makeSubject(subjectBase_MarketEventConfig)
)

// Funding settings events subjects
const (
	subjectBase_FundingEventSettings SubjectBase = "funding.event.settings"
)

const (
	// Batch price updates - all prices in a single message
	subjectBase_PriceBatchUpdate SubjectBase = "prices.batch"
)

var (
	// PriceBatchUpdate is the subject for batched price updates from the pricing service
	PriceBatchUpdate = makeSubject(subjectBase_PriceBatchUpdate)
)

var (
	// Pattern: funding.event.settings
	// Published when global funding settings are updated
	FundingEventSettings = makeSubject(subjectBase_FundingEventSettings)
)

// Price feed subjects
const (
	// Pattern: prices.[symbol].[feed_type]
	// These follow a similar pattern to marketdata subjects but for price feeds
	subjectBase_PriceFeedTypePattern SubjectBase = "%s.prices.*.%s"  // Subscribe to specific feed type for all symbols
	subjectBase_PriceSubjectPattern  SubjectBase = "%s.prices.%s.%s" // prices.{symbol}.{feed_type}
)

// Subjects in between matching and trading
const (
	subjectBase_OrdersStreamWildcard SubjectBase = "orders.>"       // Stream subscription wildcard
	subjectBase_NorthEventPattern    SubjectBase = "%s.orders.N.%s" // Order responses: matching -> trading
	subjectBase_SouthEventPattern    SubjectBase = "%s.orders.S.%s" // Order requests: trading -> matching
)

// Orderbook journal and trade event subjects
const (
	subjectBase_OrderbookJournalStreamWildcard SubjectBase = "journal.orderbook.*" // Stream subscription wildcard
	subjectBase_TradeEventStreamWildcard       SubjectBase = "traded.event.*"      // Stream subscription wildcard
	subjectBase_OrderbookJournalPattern        SubjectBase = "%s.journal.orderbook.%s"
	subjectBase_TradeEventPattern              SubjectBase = "%s.traded.event.%s"
)

// System/DLQ subjects
const (
	subjectBase_SystemDLQPosted SubjectBase = "system.dlq.posted"
)

var (
	SystemDLQPosted = makeSubject(subjectBase_SystemDLQPosted)
)

// System/health subjects
const (
	subjectBase_SubaccountHeartbeatAuth SubjectBase = "subaccount.heartbeat.auth"
)

var (
	SubaccountHeartbeatAuth = makeSubject(subjectBase_SubaccountHeartbeatAuth)
)

// Recovery coordination subjects
const (
	subjectBase_RecoveryRequest       SubjectBase = "recovery.request.%s"     // Request recovery for a symbol
	subjectBase_RecoveryResponse      SubjectBase = "recovery.response.%s"    // Recovery response for a symbol
	subjectBase_RecoveryStatus        SubjectBase = "recovery.status"         // Overall recovery status
	subjectBase_RecoveryStatusRequest SubjectBase = "recovery.status.request" // Request for current recovery status
)

// ============================================================================
// API
// ============================================================================

const (
	// Default prefix for all subjects.
	//
	// Note: this default is only used if the environment variable named
	// SNX_NATS_ALLOW_DEFAULT_SUBJECTS_PREFIX is true
	// can be inferred that the running process is a unit-test program; in all
	// other cases initialisation of this package will fail if the environment
	// variable named $SubjectsPrefixEnvVar is not present.
	DefaultPrefix = "snx-v1"
)

// subjectsConfig is the configuration for the subjects
// On the config file it will be nats.subjects_prefix and nats.allow_default_subjects_prefix
// On the environment variable it will be SNX_NATS_SUBJECTS_PREFIX and SNX_NATS_ALLOW_DEFAULT_SUBJECTS_PREFIX
type subjectsConfig struct {
	SubjectsPrefix             string // config: "subjects_prefix"
	AllowDefaultSubjectsPrefix bool   // config: "allow_default_subjects_prefix"
}

// Mutex for this API.
var mxApi sync.RWMutex

// Global prefix for all subjects.
var globalPrefix string

// ============================================================================
// Utility functions
// ============================================================================

func getGlobalPrefixSafe() string {
	mxApi.RLock()
	defer mxApi.RUnlock()

	return globalPrefix
}

func MakeSubjectForEventNorth(symbol string) string {
	prefix := getGlobalPrefixSafe()
	return fmt.Sprintf(string(subjectBase_NorthEventPattern), prefix, symbol)
}

func MakeSubjectForEventSouth(symbol string) string {
	prefix := getGlobalPrefixSafe()
	return fmt.Sprintf(string(subjectBase_SouthEventPattern), prefix, symbol)
}

func MakeSubjectForPriceFeedType(feed PriceType) string {
	prefix := getGlobalPrefixSafe()
	return fmt.Sprintf(string(subjectBase_PriceFeedTypePattern), prefix, feed)
}

// T.B.C.
//
// NOTE: This should be renamed since, as "subject" === "topic", its name contains redundancy.
func MakeSubjectForPriceTopic(symbol string, feed PriceType) string {
	prefix := getGlobalPrefixSafe()
	return fmt.Sprintf(string(subjectBase_PriceSubjectPattern), prefix, symbol, feed)
}

func MakeSubjectForOrderbookJournal(symbol string) string {
	prefix := getGlobalPrefixSafe()
	return fmt.Sprintf(string(subjectBase_OrderbookJournalPattern), prefix, symbol)
}

func MakeSubjectForTradeEvent(symbol string) string {
	prefix := getGlobalPrefixSafe()
	return fmt.Sprintf(string(subjectBase_TradeEventPattern), prefix, symbol)
}

func MakeSubjectForRecoveryRequest(symbol string) string {
	return makeSubject(SubjectBase(fmt.Sprintf(string(subjectBase_RecoveryRequest), symbol))).String()
}

func MakeSubjectForRecoveryResponse(symbol string) string {
	return makeSubject(SubjectBase(fmt.Sprintf(string(subjectBase_RecoveryResponse), symbol))).String()
}

func MakeSubjectForRecoveryStatus() string {
	return makeSubject(subjectBase_RecoveryStatus).String()
}

func MakeSubjectForRecoveryStatusRequest() string {
	return makeSubject(subjectBase_RecoveryStatusRequest).String()
}

// MakeSubjectForADLRankingUpdate returns a per-market subject for ADL ranking updates.
// Each message contains rankings for both sides (long and short).
// Example: "SNX-V6.adl.rankings.updated.BTC-USDT"
func MakeSubjectForADLRankingUpdate(symbol string) string {
	return AdlRankingsUpdated.String() + "." + symbol
}

// MakeSubjectForADLRankingWildcard returns the wildcard subject that captures all ADL ranking updates.
// Example: "SNX-V6.adl.rankings.updated.*"
func MakeSubjectForADLRankingWildcard() string {
	return AdlRankingsUpdated.String() + ".*"
}

// MakeSubjectUnsafe makes a subject suitably prefixed.
//
// Note: this function is not thread-safe, and therefore must only be used
// when it is certain that GetGlobalPrefix() will not be called.
//
// Preconditions:
//   - `len(subject) != 0`
func MakeSubjectUnsafe(subject string) string {

	if len(globalPrefix) == 0 {
		return subject
	}

	return fmt.Sprintf("%s.%s", globalPrefix, subject)
}

// MakeSubject makes a subject suitably prefixed.
//
// Note: this function is thread-safe, and therefore has a small cost.
func MakeSubject(subject string) string {
	mxApi.RLock()
	defer mxApi.RUnlock()

	return MakeSubjectUnsafe(subject)
}

// SetGlobalPrefix sets the global prefix for all subjects in a thread-safe
// manner.
//
// Parameters:
//   - prefix: the prefix to set. If this contains only whitespace then it
//     will be treated as an empty string. If this is empty (or whitespate),
//     then no prefix will be applied to the subjects when eliciting their
//     string representations.
//
// Returns:
// - the previous prefix
//
// Warning: although this method is thread-safe, calling it with a different
// value this after application initialisation will almost certainly break
// the system.
func SetGlobalPrefix(prefix string) string {
	mxApi.Lock()
	defer mxApi.Unlock()

	r := globalPrefix

	prefix = strings.TrimSpace(prefix)

	globalPrefix = prefix

	return r
}

// Converts an array of subjects to an array of strings suitable for passing
// into the NATS APIs.
func SubjectArrayToSubjectsStringArray(subjects []Subject) []string {
	subjectsStrings := make([]string, len(subjects))
	for i, subject := range subjects {
		subjectsStrings[i] = subject.String()
	}
	return subjectsStrings
}

// Converts an arbitrary number of subject arguments to an array of strings
// suitable for passing into the NATS APIs.
func SubjectsToSubjectsStringArray(subjects ...Subject) []string {
	return SubjectArrayToSubjectsStringArray(subjects)
}

// YEAH we need to find a way around this
// TODO: move me to a common utils area
func isProcessBeingUnitTested() bool {

	for i, arg := range os.Args {
		if i > 0 {
			if strings.HasPrefix(arg, "-test.") {
				return true
			}
		}
	}

	return false
}

func loadSubjectsConfig() (subjectsConfig, error) {
	viper, err := snx_lib_config.Load("SNX")
	if err != nil {
		return subjectsConfig{}, fmt.Errorf("failed to load config: %w", err)
	}

	keyPrefix := "nats"
	config := subjectsConfig{
		SubjectsPrefix:             viper.GetString(keyPrefix + ".subjects_prefix"),
		AllowDefaultSubjectsPrefix: viper.GetBool(keyPrefix + ".allow_default_subjects_prefix"),
	}

	return config, nil
}

func init() {
	// Algorithm:
	//
	// 1. Attempt to obtain subject prefix from the config / environment
	//    SNX_NATS_SUBJECTS_PREFIX; or
	// 2. Use the default prefix if we are permitted - as determined by via
	//    bool config / environment variable
	//    SNX_NATS_ALLOW_DEFAULT_SUBJECTS_PREFIX; or
	// 3. Infer whether being executed as part of unit-testing; or
	// 4. Panic, because we do not have it;
	//
	// The complexity is here because we want services to fail fast if
	// configuration is incomplete, and also support unit-testing of this
	// package.
	//
	// On the config file it will be nats.subjects_prefix and nats.allow_default_subjects_prefix
	// On the environment variable it will be SNX_NATS_SUBJECTS_PREFIX and SNX_NATS_ALLOW_DEFAULT_SUBJECTS_PREFIX

	var prefix string
	config, err := loadSubjectsConfig()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	if strings.Contains(config.SubjectsPrefix, ".") {
		panic("nats subjects prefix cannot contain a dot")
	}

	if len(config.SubjectsPrefix) == 0 {
		if !config.AllowDefaultSubjectsPrefix && !isProcessBeingUnitTested() {
			panic("nats subjects prefix is required")
		}
	}

	prefix = strings.TrimSpace(config.SubjectsPrefix)

	if len(config.SubjectsPrefix) == 0 {
		prefix = DefaultPrefix
	}

	// Note: there is no locking here, because we can rely on runtime to
	// serialise actions
	globalPrefix = prefix
}
