package pricing

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	nats_api "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func startTestNATSConn(t *testing.T) *nats_api.Conn {
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

func makePriceBatchPayload(
	t *testing.T,
	batch snx_lib_core.PriceBatchUpdate,
) []byte {
	t.Helper()

	payload, err := json.Marshal(batch)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	return payload
}

func Test_PriceCache_GetPrice_MISSING_VALUE_RETURNS_NONE(t *testing.T) {
	pc := &PriceCache{
		logger: snx_lib_logging_doubles.NewStubLogger(),
		prices: make(marketNameToPriceTypeToPriceMap),
	}

	got := pc.GetPrice(
		snx_lib_core.MarketName("BTC-USDT"),
		snx_lib_core.PriceType_mark,
	)

	assert.Equal(t, snx_lib_api_types.Price_None, got)
}

func Test_PriceCache_handleBatch_STORES_PRICES_BY_SYMBOL_AND_TYPE(t *testing.T) {
	pc := &PriceCache{
		logger: snx_lib_logging_doubles.NewStubLogger(),
		prices: make(marketNameToPriceTypeToPriceMap),
	}

	batch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(101),
			},
		},
		IndexPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(99),
			},
			snx_lib_core.MarketName("ETH-USDT"): {
				MarketName:   snx_lib_core.MarketName("ETH-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(199),
			},
		},
		LastPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(100),
			},
		},
	}

	pc.handleBatch(&nats_api.Msg{Data: makePriceBatchPayload(t, batch)})

	assert.Equal(
		t,
		snx_lib_api_types.Price("101"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_mark),
	)
	assert.Equal(
		t,
		snx_lib_api_types.Price("99"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_index),
	)
	assert.Equal(
		t,
		snx_lib_api_types.Price("100"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_last),
	)
	assert.Equal(
		t,
		snx_lib_api_types.Price("199"),
		pc.GetPrice(snx_lib_core.MarketName("ETH-USDT"), snx_lib_core.PriceType_index),
	)
}

func Test_PriceCache_handleBatch_UPDATES_ONLY_PROVIDED_VALUES(t *testing.T) {
	pc := &PriceCache{
		logger: snx_lib_logging_doubles.NewStubLogger(),
		prices: make(marketNameToPriceTypeToPriceMap),
	}

	firstBatch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(101),
			},
		},
		IndexPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(99),
			},
		},
		LastPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(100),
			},
		},
	}
	pc.handleBatch(&nats_api.Msg{Data: makePriceBatchPayload(t, firstBatch)})

	secondBatch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(105),
			},
		},
	}
	pc.handleBatch(&nats_api.Msg{Data: makePriceBatchPayload(t, secondBatch)})

	assert.Equal(
		t,
		snx_lib_api_types.Price("105"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_mark),
	)
	assert.Equal(
		t,
		snx_lib_api_types.Price("99"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_index),
	)
	assert.Equal(
		t,
		snx_lib_api_types.Price("100"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_last),
	)
}

func Test_PriceCache_handleBatch_INVALID_JSON_DOES_NOT_MUTATE_CACHE(t *testing.T) {
	logger := snx_lib_logging_doubles.NewSpyLogger()
	pc := &PriceCache{
		logger: logger,
		prices: make(marketNameToPriceTypeToPriceMap),
	}

	batch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(101),
			},
		},
	}
	pc.handleBatch(&nats_api.Msg{Data: makePriceBatchPayload(t, batch)})

	pc.handleBatch(&nats_api.Msg{Data: []byte("{invalid-json")})

	assert.Equal(
		t,
		snx_lib_api_types.Price("101"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_mark),
	)
	assert.True(t, logger.HasEntry(snx_lib_logging_doubles.LevelError, "failed to unmarshal price batch"))
}

func Test_NewPriceCache_SUBSCRIBES_AND_RECEIVES_PUBLISHED_BATCH(t *testing.T) {
	nc := startTestNATSConn(t)
	logger := snx_lib_logging_doubles.NewStubLogger()

	pc, err := NewPriceCache(logger, nc)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	t.Cleanup(pc.Stop)

	batch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(12345),
			},
		},
	}
	err = nc.Publish(snx_lib_db_nats.PriceBatchUpdate.String(), makePriceBatchPayload(t, batch))
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NoError(t, nc.Flush(), "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Eventually(t, func() bool {
		return pc.GetPrice(
			snx_lib_core.MarketName("BTC-USDT"),
			snx_lib_core.PriceType_mark,
		) == snx_lib_api_types.Price("12345")
	}, time.Second, 10*time.Millisecond)
}

func Test_NewPriceCache_CLOSED_CONNECTION_RETURNS_ERROR(t *testing.T) {
	nc := startTestNATSConn(t)
	nc.Close()

	pc, err := NewPriceCache(snx_lib_logging_doubles.NewStubLogger(), nc)

	require.Error(t, err)
	require.Nil(t, pc)
}

func Test_PriceCache_Stop_UNSUBSCRIBES(t *testing.T) {
	nc := startTestNATSConn(t)
	pc, err := NewPriceCache(snx_lib_logging_doubles.NewStubLogger(), nc)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	initialBatch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(101),
			},
		},
	}
	err = nc.Publish(snx_lib_db_nats.PriceBatchUpdate.String(), makePriceBatchPayload(t, initialBatch))
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NoError(t, nc.Flush(), "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	require.Eventually(t, func() bool {
		return pc.GetPrice(
			snx_lib_core.MarketName("BTC-USDT"),
			snx_lib_core.PriceType_mark,
		) == snx_lib_api_types.Price("101")
	}, time.Second, 10*time.Millisecond)

	pc.Stop()
	pc.Stop()

	updatedBatch := snx_lib_core.PriceBatchUpdate{
		MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
			snx_lib_core.MarketName("BTC-USDT"): {
				MarketName:   snx_lib_core.MarketName("BTC-USDT"),
				CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(202),
			},
		},
	}
	err = nc.Publish(snx_lib_db_nats.PriceBatchUpdate.String(), makePriceBatchPayload(t, updatedBatch))
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NoError(t, nc.Flush(), "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(
		t,
		snx_lib_api_types.Price("101"),
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_mark),
	)
}

func Test_PriceCache_Stop_WITH_NIL_SUBSCRIPTION_DOES_NOTHING(t *testing.T) {
	pc := &PriceCache{}
	pc.Stop()
}

func Test_PriceCache_handleBatch_GetPrice_NO_RACE(t *testing.T) {
	pc := &PriceCache{
		logger: snx_lib_logging_doubles.NewStubLogger(),
		prices: make(marketNameToPriceTypeToPriceMap),
	}

	const (
		writerCount      = 4
		readerCount      = 8
		updatesPerWriter = 400
	)

	payloads := make([][]byte, updatesPerWriter)
	for i := 0; i < updatesPerWriter; i++ {
		batch := snx_lib_core.PriceBatchUpdate{
			MarkPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
				snx_lib_core.MarketName("BTC-USDT"): {
					MarketName:   snx_lib_core.MarketName("BTC-USDT"),
					CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(int64(10000 + i)),
				},
			},
			IndexPrices: snx_lib_core.MapMarketNameToMarketPriceUpdateElement{
				snx_lib_core.MarketName("ETH-USDT"): {
					MarketName:   snx_lib_core.MarketName("ETH-USDT"),
					CurrentPrice: snx_lib_core.PriceFromIntUnvalidated(int64(20000 + i)),
				},
			},
		}

		payload := makePriceBatchPayload(t, batch)
		payloads[i] = payload
	}

	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < writerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			for _, payload := range payloads {
				pc.handleBatch(&nats_api.Msg{Data: payload})
			}
		}()
	}

	for i := 0; i < readerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < updatesPerWriter*2; j++ {
				_ = pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_mark)
				_ = pc.GetPrice(snx_lib_core.MarketName("ETH-USDT"), snx_lib_core.PriceType_index)
				_ = pc.GetPrice(snx_lib_core.MarketName("SOL-USDT"), snx_lib_core.PriceType_last)
			}
		}()
	}

	close(start)
	wg.Wait()

	assert.NotEqual(
		t,
		snx_lib_api_types.Price_None,
		pc.GetPrice(snx_lib_core.MarketName("BTC-USDT"), snx_lib_core.PriceType_mark),
	)
	assert.NotEqual(
		t,
		snx_lib_api_types.Price_None,
		pc.GetPrice(snx_lib_core.MarketName("ETH-USDT"), snx_lib_core.PriceType_index),
	)
}
