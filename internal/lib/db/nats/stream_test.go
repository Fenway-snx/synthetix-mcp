package nats_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
)

func Test_StreamName_WithPrefix(t *testing.T) {
	tests := []struct {
		name       string
		streamName string
		expected   string
	}{
		{
			"ExecutionEvents",
			snx_lib_db_nats.StreamName_ExecutionEvents.String(),
			"snx-v1-EXECUTION_EVENTS",
		},
		{
			"LiquidationEvents",
			snx_lib_db_nats.StreamName_LiquidationEvents.String(),
			"snx-v1-LIQUIDATION_EVENTS",
		},
		{
			"AccountLifecycle",
			snx_lib_db_nats.StreamName_AccountLifecycle.String(),
			"snx-v1-ACCOUNT_LIFECYCLE",
		},
		{
			"FundingEvents",
			snx_lib_db_nats.StreamName_FundingEvents.String(),
			"snx-v1-FUNDING_EVENTS",
		},
		{
			"OrderBookJournal",
			snx_lib_db_nats.StreamName_OrderBookJournal.String(),
			"snx-v1-ORDERBOOK_JOURNAL",
		},
		{
			"Orders",
			snx_lib_db_nats.StreamName_Orders.String(),
			"snx-v1-ORDERS",
		},
		{
			"ADLRankings",
			snx_lib_db_nats.StreamName_ADLRankings.String(),
			"snx-v1-ADL_RANKINGS",
		},
		{
			"AccountsTreasury",
			snx_lib_db_nats.StreamName_AccountsTreasury.String(),
			"snx-v1-ACCOUNTS_TREASURY",
		},
		{
			"RelayerTxnQueue",
			snx_lib_db_nats.StreamName_RelayerTxnQueue.String(),
			"snx-v1-RELAYER_TRANSACTIONS",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.streamName)
		})
	}
}

func assertStreamDefaults(t *testing.T, cfg jetstream.StreamConfig, expectedName string, numReplicas int) {
	t.Helper()
	assert.Equal(t, expectedName, cfg.Name)
	assert.Equal(t, snx_lib_db_nats.FileStorage, cfg.Storage)
	assert.Equal(t, snx_lib_db_nats.LimitsRetention, cfg.Retention)
	assert.Equal(t, snx_lib_db_nats.Unlimited, cfg.MaxConsumers)
	assert.EqualValues(t, snx_lib_db_nats.DefaultMaxMsgs, cfg.MaxMsgs)
	assert.EqualValues(t, snx_lib_db_nats.DefaultMaxBytes, cfg.MaxBytes)
	assert.Equal(t, 24*time.Hour, cfg.MaxAge)
	assert.Equal(t, numReplicas, cfg.Replicas)
	assert.Len(t, cfg.Subjects, 1)
}

func Test_ExecutionEventsStream_Config(t *testing.T) {
	cfg := snx_lib_db_nats.CreateExecutionEventsStreamConfig(3)

	assert.Equal(t, snx_lib_db_nats.StreamName_ExecutionEvents.String(), cfg.Name)
	assert.Equal(t, snx_lib_db_nats.FileStorage, cfg.Storage)
	assert.Equal(t, snx_lib_db_nats.WorkQueueRetention, cfg.Retention)
	assert.Equal(t, 3, cfg.Replicas)
	assert.Len(t, cfg.Subjects, 1)
	assert.Equal(t, snx_lib_db_nats.MakeSubject("js.execution-events.>"), cfg.Subjects[0])
}

func Test_LiquidationEventsStream_Config(t *testing.T) {
	cfg := snx_lib_db_nats.CreateLiquidationEventsStreamConfig(3)
	assertStreamDefaults(t, cfg, snx_lib_db_nats.StreamName_LiquidationEvents.String(), 3)
	assert.Equal(t, snx_lib_db_nats.MakeSubject("js.liquidation-events.>"), cfg.Subjects[0])
}

func Test_AccountLifecycleStream_Config(t *testing.T) {
	cfg := snx_lib_db_nats.CreateAccountLifecycleStreamConfig(3)
	assertStreamDefaults(t, cfg, snx_lib_db_nats.StreamName_AccountLifecycle.String(), 3)
	assert.Equal(t, snx_lib_db_nats.MakeSubject("js.account-lifecycle.>"), cfg.Subjects[0])
}

func Test_FundingEventsStream_Config(t *testing.T) {
	cfg := snx_lib_db_nats.CreateFundingEventsStreamConfig(3)
	assertStreamDefaults(t, cfg, snx_lib_db_nats.StreamName_FundingEvents.String(), 3)
	assert.Equal(t, snx_lib_db_nats.MakeSubject("js.funding-events.>"), cfg.Subjects[0])
}

func Test_AccountsTreasuryStream_DefaultConfig(t *testing.T) {
	config := snx_lib_db_nats.CreateAccountsTreasuryStreamConfig(3)

	assert.Equal(t, snx_lib_db_nats.StreamName_AccountsTreasury.String(), config.Name)
	assert.Equal(t, snx_lib_db_nats.FileStorage, config.Storage)
	assert.Equal(t, snx_lib_db_nats.WorkQueueRetention, config.Retention)
	assert.Len(t, config.Subjects, 1)
	assert.EqualValues(t, snx_lib_db_nats.DefaultMaxMsgs, config.MaxMsgs)
	assert.EqualValues(t, snx_lib_db_nats.DefaultMaxBytes, config.MaxBytes)
	assert.Equal(t, 24*time.Hour, config.MaxAge)
	assert.Equal(t, 3, config.Replicas)
}

func Test_OrdersStream_DefaultConfig(t *testing.T) {
	t.Run("default configs for orders stream", func(t *testing.T) {
		config := snx_lib_db_nats.CreateOrdersStreamConfig(3)

		assert.Equal(t, config.MaxAge, time.Hour)
		assert.EqualValues(t, config.MaxBytes, snx_lib_db_nats.DefaultMaxBytes)
		assert.EqualValues(t, config.MaxMsgs, snx_lib_db_nats.DefaultMaxMsgs)
		assert.Equal(t, config.Name, snx_lib_db_nats.StreamName_Orders.String())
		assert.Equal(t, config.Storage, snx_lib_db_nats.MemoryStorage)
		assert.Len(t, config.Subjects, 1)
		assert.Contains(t, config.Subjects[0], "orders.>")
		assert.Equal(t, config.Retention, snx_lib_db_nats.WorkQueueRetention)
		assert.Equal(t, 3, config.Replicas)
	})
}

func Test_RelayerTxnQueueStream_Config(t *testing.T) {
	config := snx_lib_db_nats.CreateRelayerTxnQueueStreamConfig(3)

	assert.Equal(t, snx_lib_db_nats.StreamName_RelayerTxnQueue.String(), config.Name)
	assert.Equal(t, snx_lib_db_nats.FileStorage, config.Storage)
	assert.Equal(t, snx_lib_db_nats.WorkQueueRetention, config.Retention)
	assert.Equal(t, snx_lib_db_nats.Unlimited, config.MaxConsumers)
	assert.Equal(t, 3, config.Replicas)

	assert.Len(t, config.Subjects, 1)
	assert.Contains(t, config.Subjects, snx_lib_db_nats.MakeSubject("relayer.txn.>"))
}

// Test_AccountsStreamSplit_SubjectCompleteness verifies every subject from the
// former monolithic ACCOUNTS stream that has been migrated into the four
// domain-specific streams still maps to exactly one new stream.
//
// The historical ACCOUNTS stream also contained three legacy subjects that are
// intentionally still outside the new domain streams:
// - RelayerEventWithdrawDisputed
// - SubaccountEventCreate
// - TradingEventPositionCreated
//
// Those are excluded here until they are either migrated or removed.
func Test_AccountsStreamSplit_SubjectCompleteness(t *testing.T) {
	type streamFactory = func(int) jetstream.StreamConfig

	newStreams := map[string]streamFactory{
		"EXECUTION_EVENTS":   snx_lib_db_nats.CreateExecutionEventsStreamConfig,
		"LIQUIDATION_EVENTS": snx_lib_db_nats.CreateLiquidationEventsStreamConfig,
		"ACCOUNT_LIFECYCLE":  snx_lib_db_nats.CreateAccountLifecycleStreamConfig,
		"FUNDING_EVENTS":     snx_lib_db_nats.CreateFundingEventsStreamConfig,
	}

	oldAccountsSubjects := snx_lib_db_nats.SubjectsToSubjectsStringArray(
		snx_lib_db_nats.FundingRateBalanceUpdate,
		snx_lib_db_nats.FundingRatePosted,
		snx_lib_db_nats.RelayerCommandCowOrder,
		snx_lib_db_nats.RelayerEventDeposit,
		snx_lib_db_nats.RelayerEventWithdrawCompleted,
		snx_lib_db_nats.RelayerEventWithdrawSubmissionFailed,
		snx_lib_db_nats.SubaccountCommandWithdraw,
		snx_lib_db_nats.SubaccountEventDepositProcessed,
		snx_lib_db_nats.SubaccountEventStatusUpdated,
		snx_lib_db_nats.SubaccountFeeRateUpdate,
		snx_lib_db_nats.TradingEventAccountStateUpdated,
		snx_lib_db_nats.TradingEventAtomicPositionUpdate,
		snx_lib_db_nats.TradingEventInsurancePositionIncreased,
		snx_lib_db_nats.TradingEventInsuranceProtectionActivated,
		snx_lib_db_nats.TradingEventInsuranceProtectionCompleted,
		snx_lib_db_nats.TradingEventOpenOrderCreated,
		snx_lib_db_nats.TradingEventOrderHistory,
		snx_lib_db_nats.TradingEventPositionTPSL,
		snx_lib_db_nats.TradingEventTradeHistory,
		snx_lib_db_nats.TradingEventWithdrawProcessed,
		snx_lib_db_nats.TradingEventWithdrawRequest,
	)

	matchesWildcard := func(subject, wildcard string) bool {
		wPrefix := strings.TrimSuffix(wildcard, ">")
		return strings.HasPrefix(subject, wPrefix)
	}

	for _, subj := range oldAccountsSubjects {
		var matched []string
		for name, factory := range newStreams {
			cfg := factory(1)
			for _, wc := range cfg.Subjects {
				if matchesWildcard(subj, wc) {
					matched = append(matched, name)
				}
			}
		}
		require.Len(t, matched, 1, "subject %s should match exactly one new stream, got %v", subj, matched)
	}
}
