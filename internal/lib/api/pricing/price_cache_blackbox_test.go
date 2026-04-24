package pricing_test

import (
	"encoding/json"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	nats_api "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_pricing "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/pricing"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func startTestNATSConnBlackbox(t *testing.T) *nats_api.Conn {
	t.Helper()

	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	server := natsserver.RunServer(&opts)
	t.Cleanup(server.Shutdown)

	nc, err := nats_api.Connect(server.ClientURL(), nats_api.Timeout(2*time.Second))
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	t.Cleanup(nc.Close)

	return nc
}

func makePriceBatchPayloadBlackbox(
	t *testing.T,
	batch snx_lib_core.PriceBatchUpdate,
) []byte {
	t.Helper()

	payload, err := json.Marshal(batch)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	return payload
}

func Test_NewPriceCache_GETPRICE_RETURNS_PUBLISHED_MARK_PRICE(t *testing.T) {
	nc := startTestNATSConnBlackbox(t)

	pc, err := snx_lib_api_pricing.NewPriceCache(snx_lib_logging_doubles.NewStubLogger(), nc)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	t.Cleanup(pc.Stop)

	batch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(54321),
			},
		},
	}
	err = nc.Publish(snx_lib_db_nats.PriceBatchUpdate.String(), makePriceBatchPayloadBlackbox(t, batch))
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NoError(t, nc.Flush(), "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Eventually(t, func() bool {
		return pc.GetPrice(
			snx_lib_core.MarketName("BTC-USDT"),
			snx_lib_core.PriceType_mark,
		) == snx_lib_api_types.Price("54321")
	}, time.Second, 10*time.Millisecond)
}

func Test_NewPriceCache_CLOSED_CONNECTION_RETURNS_ERROR(t *testing.T) {
	nc := startTestNATSConnBlackbox(t)
	nc.Close()

	pc, err := snx_lib_api_pricing.NewPriceCache(snx_lib_logging_doubles.NewStubLogger(), nc)

	require.Error(t, err)
	assert.Nil(t, pc)
}
