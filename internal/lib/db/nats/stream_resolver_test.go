package nats_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
)

func newResolver() *snx_lib_db_nats.SubjectToStreamNameResolver {
	return snx_lib_db_nats.NewSubjectToStreamNameResolver()
}

func Test_SubjectToStreamNameResolver_Resolve(t *testing.T) {
	r := newResolver()

	tests := []struct {
		name    string
		subject string
		want    string
	}{
		// Exact matches (pre-cached from stream configs)
		{"execution_events/order_history", snx_lib_db_nats.TradingEventOrderHistory.String(), "snx-v1-EXECUTION_EVENTS"},
		{"execution_events/open_order_created", snx_lib_db_nats.TradingEventOpenOrderCreated.String(), "snx-v1-EXECUTION_EVENTS"},
		{"account_lifecycle/deposit", snx_lib_db_nats.RelayerEventDeposit.String(), "snx-v1-ACCOUNT_LIFECYCLE"},
		{"funding_events/rate_posted", snx_lib_db_nats.FundingRatePosted.String(), "snx-v1-FUNDING_EVENTS"},

		// Prefix matches (wildcard stream configs)
		{"adl_rankings/BTC", snx_lib_db_nats.MakeSubjectForADLRankingUpdate("BTC-USDT"), "snx-v1-ADL_RANKINGS"},
		{"adl_rankings/ETH", snx_lib_db_nats.MakeSubjectForADLRankingUpdate("ETH-USDT"), "snx-v1-ADL_RANKINGS"},
		{"relayer_txn/queue", snx_lib_db_nats.MakeSubject("relayer.txn.queue"), "snx-v1-RELAYER_TRANSACTIONS"},
		{"relayer_txn/queue_teller", snx_lib_db_nats.MakeSubject("relayer.txn.queue.teller"), "snx-v1-RELAYER_TRANSACTIONS"},
		{"relayer_pending_tx/queue_watcher", snx_lib_db_nats.MakeSubject("relayer.txn.pending.watcher"), "snx-v1-RELAYER_TRANSACTIONS"},
		{"accounts_treasury", snx_lib_db_nats.MakeSubject("ACCOUNTS_TREASURY.something"), "snx-v1-ACCOUNTS_TREASURY"},

		// Suffix matches (order events)
		{"orders/north/BTC", snx_lib_db_nats.MakeSubjectForEventNorth("BTC-USDT"), "snx-v1-ORDERS"},
		{"orders/south/BTC", snx_lib_db_nats.MakeSubjectForEventSouth("BTC-USDT"), "snx-v1-ORDERS"},
		{"orders/north/ETH", snx_lib_db_nats.MakeSubjectForEventNorth("ETH-USDT"), "snx-v1-ORDERS"},
		{"orders/south/SOL", snx_lib_db_nats.MakeSubjectForEventSouth("SOL-USDT"), "snx-v1-ORDERS"},

		// Contains matches (orderbook journal)
		{"journal/BTC", snx_lib_db_nats.MakeSubjectForOrderbookJournal("BTC-USDT"), "snx-v1-ORDERBOOK_JOURNAL"},
		{"journal/ETH", snx_lib_db_nats.MakeSubjectForOrderbookJournal("ETH-USDT"), "snx-v1-ORDERBOOK_JOURNAL"},
		{"trade_event/BTC", snx_lib_db_nats.MakeSubjectForTradeEvent("BTC-USDT"), "snx-v1-ORDERBOOK_JOURNAL"},

		// _INBOX (dedicated fast path, never cached)
		{"inbox/1", "_INBOX.abc123.1", "INBOX"},
		{"inbox/2", "_INBOX.xyz789.42", "INBOX"},

		// Core NATS subjects (no stream)
		{"core/price_feed", snx_lib_db_nats.MakeSubject("price.feed.BTC-USDT.mark"), ""},
		{"core/recovery", snx_lib_db_nats.MakeSubject("recovery.request.BTC-USDT"), ""},
		{"core/unknown", snx_lib_db_nats.MakeSubject("unknown.subject.here"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.Resolve(tt.subject))
		})
	}
}

func Test_SubjectToStreamNameResolver_CacheReturnsSameResult(t *testing.T) {
	r := newResolver()

	subj := snx_lib_db_nats.TradingEventOrderHistory.String()
	first := r.Resolve(subj)
	second := r.Resolve(subj)
	assert.Equal(t, first, second)
	assert.Equal(t, "snx-v1-EXECUTION_EVENTS", first)

	unknown := snx_lib_db_nats.MakeSubject("unknown.subject.here")
	assert.Equal(t, "", r.Resolve(unknown))
	assert.Equal(t, "", r.Resolve(unknown))
}
