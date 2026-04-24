package nats_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
)

func Test_SetGlobalPrefix_AFFECTS_GLOBAL_STATE_AND_CAN_BE_REVERSED(t *testing.T) {

	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	// set it, and verify the original value
	{
		r := snx_lib_db_nats.SetGlobalPrefix("test")
		originalPrefix = r
		assert.Equal(t, "snx-v1", r)
	}

	// set it again, and verify the previously supplied value
	{
		r := snx_lib_db_nats.SetGlobalPrefix("test-2")
		assert.Equal(t, "test", r)
	}
}

func Test_SetGlobalPrefix_WITH_EMPTY_STRING(t *testing.T) {

	// set with actual empty string
	{
		var originalPrefix string

		defer func() {
			snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
		}()

		originalPrefix = snx_lib_db_nats.SetGlobalPrefix("")

		assert.Equal(t, snx_lib_db_nats.DefaultPrefix, originalPrefix)

		emptyPrefx := snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
		assert.Equal(t, "", emptyPrefx)
	}

	// set with nearly empty string
	{
		var originalPrefix string

		defer func() {
			snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
		}()

		// Passing " " and NOT ""
		originalPrefix = snx_lib_db_nats.SetGlobalPrefix(" ")

		assert.Equal(t, snx_lib_db_nats.DefaultPrefix, originalPrefix)

		emptyPrefx := snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
		// Receiving "" and NOT " "
		assert.Equal(t, "", emptyPrefx)
	}
}

func Test_MakeSubject_PREFIXES_SUBJECT_FROM_GLOGAL_STATE(t *testing.T) {
	{
		r := snx_lib_db_nats.MakeSubject("test")
		assert.Equal(t, "snx-v1.test", r)
	}
}

func Test_MakeSubjectUnsafe_PREFIXES_SUBJECT_FROM_GLOGAL_STATE(t *testing.T) {
	{
		r := snx_lib_db_nats.MakeSubjectUnsafe("test")
		assert.Equal(t, "snx-v1.test", r)
	}
}

func Test_MakeSubjectUnsafe_PREFIXES_SUBJECT_FROM_GLOGAL_STATE_AS_EMPTY_STRING(t *testing.T) {

	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	originalPrefix = snx_lib_db_nats.SetGlobalPrefix("")

	{
		r := snx_lib_db_nats.MakeSubject("test")
		assert.Equal(t, "test", r)
	}
}

func Test_MakeSubjectForEventNorth(t *testing.T) {
	// setup
	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	originalPrefix = snx_lib_db_nats.SetGlobalPrefix("NOT_USED")

	// with specific prefix
	{
		snx_lib_db_nats.SetGlobalPrefix("test-4")

		r := snx_lib_db_nats.MakeSubjectForEventNorth("BTC-USDT")
		assert.Equal(t, "test-4.orders.N.BTC-USDT", r)
	}
}

func Test_MakeSubjectForEventSouth(t *testing.T) {
	// setup
	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	originalPrefix = snx_lib_db_nats.SetGlobalPrefix("NOT_USED")

	// with specific prefix
	{
		snx_lib_db_nats.SetGlobalPrefix("test-4")

		r := snx_lib_db_nats.MakeSubjectForEventSouth("BTC-USDT")
		assert.Equal(t, "test-4.orders.S.BTC-USDT", r)
	}
}

// func MakeSubjectForEventSouth(feed string) string {

func Test_MakeSubjectForPriceFeedType(t *testing.T) {
	// setup
	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	originalPrefix = snx_lib_db_nats.SetGlobalPrefix("NOT_USED")

	// with specific prefix
	{
		snx_lib_db_nats.SetGlobalPrefix("test-4")

		r := snx_lib_db_nats.MakeSubjectForPriceFeedType("mark")
		assert.Equal(t, "test-4.prices.*.mark", r)
	}
}

func Test_MakeSubjectForPriceTopic(t *testing.T) {
	// setup
	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	originalPrefix = snx_lib_db_nats.SetGlobalPrefix("NOT_USED")

	// with specific prefix
	{
		snx_lib_db_nats.SetGlobalPrefix("test-4")

		r := snx_lib_db_nats.MakeSubjectForPriceTopic("BTC-USD", "mark")
		assert.Equal(t, "test-4.prices.BTC-USD.mark", r)
	}
}

func Test_Subject_CONSTANTS_String_FORMS(t *testing.T) {

	// setup
	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	originalPrefix = snx_lib_db_nats.SetGlobalPrefix("test-2")

	// JetStream subjects -- js.{stream}[.{group}...].{designator}
	{
		assert.Equal(t, "test-2.js.execution-events.account.state-updated", snx_lib_db_nats.TradingEventAccountStateUpdated.String())
		assert.Equal(t, "test-2.js.execution-events.order.open-created", snx_lib_db_nats.TradingEventOpenOrderCreated.String())
		assert.Equal(t, "test-2.js.execution-events.order.history", snx_lib_db_nats.TradingEventOrderHistory.String())
		assert.Equal(t, "test-2.js.execution-events.trade.history", snx_lib_db_nats.TradingEventTradeHistory.String())
		assert.Equal(t, "test-2.js.liquidation-events.position.increased", snx_lib_db_nats.TradingEventInsurancePositionIncreased.String())
		assert.Equal(t, "test-2.js.account-lifecycle.deposit.received", snx_lib_db_nats.RelayerEventDeposit.String())
		assert.Equal(t, "test-2.js.account-lifecycle.withdrawal.completed", snx_lib_db_nats.RelayerEventWithdrawCompleted.String())
		assert.Equal(t, "test-2.js.account-lifecycle.deposit.processed", snx_lib_db_nats.SubaccountEventDepositProcessed.String())
		assert.Equal(t, "test-2.js.account-lifecycle.withdrawal.relayer-command", snx_lib_db_nats.RelayerCommandWithdraw.String())
		assert.Equal(t, "test-2.js.account-lifecycle.withdrawal.command", snx_lib_db_nats.SubaccountCommandWithdraw.String())
		assert.Equal(t, "test-2.ACCOUNTS_TREASURY.trading.transfer", snx_lib_db_nats.AccountsTreasury_TradingEventTransferHistory.String())
	}

	// non-JetStream event subjects
	{
		assert.Equal(t, "test-2.subaccount.event.transfer.request", snx_lib_db_nats.SubaccountEventTransferRequest.String())
	}
}

func Test_Subject_CONSTANTS_CAN_BE_FORMATTED_AS_STRINGS(t *testing.T) {

	// setup
	var originalPrefix string

	defer func() {
		snx_lib_db_nats.SetGlobalPrefix(originalPrefix)
	}()

	originalPrefix = snx_lib_db_nats.SetGlobalPrefix("test-3")

	// JetStream subjects -- js.{stream}[.{group}...].{designator}
	{
		assert.Equal(t, "test-3.js.execution-events.account.state-updated", fmt.Sprintf("%v", snx_lib_db_nats.TradingEventAccountStateUpdated))
		assert.Equal(t, "test-3.js.execution-events.order.open-created", fmt.Sprintf("%s", snx_lib_db_nats.TradingEventOpenOrderCreated))
		assert.Equal(t, "test-3.js.execution-events.order.history", fmt.Sprintf("%v", snx_lib_db_nats.TradingEventOrderHistory))
		assert.Equal(t, "test-3.js.execution-events.trade.history", fmt.Sprintf("%v", snx_lib_db_nats.TradingEventTradeHistory))
		assert.Equal(t, "test-3.js.liquidation-events.position.increased", fmt.Sprintf("%v", snx_lib_db_nats.TradingEventInsurancePositionIncreased))
		assert.Equal(t, "test-3.js.account-lifecycle.deposit.received", fmt.Sprintf("%s", snx_lib_db_nats.RelayerEventDeposit))
		assert.Equal(t, "test-3.js.account-lifecycle.withdrawal.completed", fmt.Sprintf("%v", snx_lib_db_nats.RelayerEventWithdrawCompleted))
		assert.Equal(t, "test-3.js.account-lifecycle.deposit.processed", fmt.Sprintf("%v", snx_lib_db_nats.SubaccountEventDepositProcessed))
		assert.Equal(t, "test-3.js.account-lifecycle.withdrawal.relayer-command", fmt.Sprintf("%s", snx_lib_db_nats.RelayerCommandWithdraw))
		assert.Equal(t, "test-3.js.account-lifecycle.withdrawal.command", fmt.Sprintf("%v", snx_lib_db_nats.SubaccountCommandWithdraw))
		assert.Equal(t, "test-3.ACCOUNTS_TREASURY.trading.transfer", fmt.Sprintf("%s", snx_lib_db_nats.AccountsTreasury_TradingEventTransferHistory))
	}

	// non-JetStream event subjects
	{
		assert.Equal(t, "test-3.subaccount.event.transfer.request", fmt.Sprintf("%s", snx_lib_db_nats.SubaccountEventTransferRequest))
	}
}

func Test_OrderSubjectsMatchStreamWildcard(t *testing.T) {
	originalPrefix := snx_lib_db_nats.SetGlobalPrefix("test")
	defer snx_lib_db_nats.SetGlobalPrefix(originalPrefix)

	streamSubject := snx_lib_db_nats.CreateOrdersStreamConfig(1).Subjects[0]
	streamPrefix := strings.TrimSuffix(streamSubject, ">")

	south := snx_lib_db_nats.MakeSubjectForEventSouth("BTC-USDT")
	north := snx_lib_db_nats.MakeSubjectForEventNorth("BTC-USDT")

	assert.True(t, strings.HasPrefix(south, streamPrefix), "south subject %q should start with stream prefix %q", south, streamPrefix)
	assert.True(t, strings.HasPrefix(north, streamPrefix), "north subject %q should start with stream prefix %q", north, streamPrefix)
}

func Test_OrderBookJournalSubjectsMatchStreamWildcards(t *testing.T) {
	originalPrefix := snx_lib_db_nats.SetGlobalPrefix("test")
	defer snx_lib_db_nats.SetGlobalPrefix(originalPrefix)

	cfg := snx_lib_db_nats.CreateOrderBookJournalStreamConfig(1)

	journalStream := cfg.Subjects[0]
	journalPrefix := strings.TrimSuffix(journalStream, "*")

	tradeStream := cfg.Subjects[1]
	tradePrefix := strings.TrimSuffix(tradeStream, "*")

	journal := snx_lib_db_nats.MakeSubjectForOrderbookJournal("BTC-USDT")
	trade := snx_lib_db_nats.MakeSubjectForTradeEvent("BTC-USDT")

	assert.True(t, strings.HasPrefix(journal, journalPrefix), "journal subject %q should start with stream prefix %q", journal, journalPrefix)
	assert.True(t, strings.HasPrefix(trade, tradePrefix), "trade subject %q should start with stream prefix %q", trade, tradePrefix)
}
