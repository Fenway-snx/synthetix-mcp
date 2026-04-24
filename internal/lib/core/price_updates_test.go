package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PriceUpdate_LookupMarketPriceFor(t *testing.T) {

	t.Run("empty", func(t *testing.T) {

		pu := PriceBatchUpdate{}

		{
			marketName := MarketName("BTC-USDT")

			_, found := pu.LookupMarketPriceFor(marketName, PriceType_index)

			assert.False(t, found)
		}
	})

	t.Run("some index prices", func(t *testing.T) {

		var tN int64 = 42

		btcusdtName := MarketName("BTC-USDT")
		ethusdtName := MarketName("ETH-USDT")

		pu := PriceBatchUpdate{
			PricesAt: time.UnixMilli(tN).UTC(),
			LastPrices: MapMarketNameToMarketPriceUpdateElement{
				btcusdtName: {
					MarketName:    btcusdtName,
					UpdatedAt:     time.UnixMilli(tN).UTC(),
					CurrentPrice:  PriceFromIntUnvalidated(123457),
					PreviousPrice: PriceFromIntUnvalidated(123456),
					Source:        "Binance",
				},
				ethusdtName: {
					MarketName:    ethusdtName,
					UpdatedAt:     time.UnixMilli(tN - 1).UTC(),
					CurrentPrice:  PriceFromIntUnvalidated(99),
					PreviousPrice: PriceFromIntUnvalidated(99),
					Source:        "Binance",
				},
			},
		}

		{
			marketName := MarketName("BTC-USDT")

			_, found := pu.LookupMarketPriceFor(marketName, PriceType_index)

			assert.False(t, found)
		}

		{
			marketName := MarketName("BTC-USDT")

			r, found := pu.LookupMarketPriceFor(marketName, PriceType_last)

			require.True(t, found)
			assert.Equal(t, marketName, r.MarketName)

			latestPrice, isChangedInMostRecentBatch := r.LatestPrice()

			assert.True(t, isChangedInMostRecentBatch)
			assert.Equal(t, PriceFromIntUnvalidated(123457), latestPrice)
		}

		{
			marketName := MarketName("ETH-USDT")

			r, found := pu.LookupMarketPriceFor(marketName, PriceType_last)

			require.True(t, found)
			assert.Equal(t, marketName, r.MarketName)

			latestPrice, isChangedInMostRecentBatch := r.LatestPrice()

			assert.False(t, isChangedInMostRecentBatch)
			assert.Equal(t, PriceFromIntUnvalidated(99), latestPrice)
		}

		{
			marketName := MarketName("BTC-USDT")

			_, found := pu.LookupMarketPriceFor(marketName, PriceType_mark)

			assert.False(t, found)
		}
	})
}

func Test_LatestPrice_ZeroCurrentPrice_FallsBackToPreviousPrice(t *testing.T) {

	var tN int64 = 100

	// Simulate a scenario where CurrentPrice is zero (e.g. deserialization
	// failure) but PreviousPrice holds the last known good value.
	r := MarketPriceUpdateSymbolResult{
		MarketName:    MarketName("BTC-USDT"),
		CurrentPrice:  Price_Zero,
		PreviousPrice: PriceFromIntUnvalidated(50000),
		UpdatedAt:     time.UnixMilli(tN - 1).UTC(),
		PricesAt:      time.UnixMilli(tN).UTC(),
	}

	latestPrice, isChanged := r.LatestPrice()

	assert.Equal(t, PriceFromIntUnvalidated(50000), latestPrice, "should fall back to PreviousPrice")
	assert.False(t, isChanged, "fallback should never be marked as changed")
}

func Test_LatestPrice_BothZero_ReturnsZero(t *testing.T) {

	var tN int64 = 100

	// Worst case: both prices are zero. LatestPrice returns zero —
	// callers MUST guard with IsPositive().
	r := MarketPriceUpdateSymbolResult{
		MarketName:    MarketName("BTC-USDT"),
		CurrentPrice:  Price_Zero,
		PreviousPrice: Price_Zero,
		UpdatedAt:     time.UnixMilli(tN - 1).UTC(),
		PricesAt:      time.UnixMilli(tN).UTC(),
	}

	latestPrice, isChanged := r.LatestPrice()

	assert.True(t, latestPrice.IsZero(), "both zero means zero return — callers must check")
	assert.False(t, isChanged)
}

func Test_LatestPrice_PositiveCurrentPrice_StaleUpdate_ReturnsCurrentPrice(t *testing.T) {

	var tN int64 = 100

	// Price was set in a prior batch (UpdatedAt < PricesAt) but is still
	// a valid positive value. The new logic correctly returns CurrentPrice
	// (not PreviousPrice).
	r := MarketPriceUpdateSymbolResult{
		MarketName:    MarketName("ETH-USDT"),
		CurrentPrice:  PriceFromIntUnvalidated(3000),
		PreviousPrice: PriceFromIntUnvalidated(2900),
		UpdatedAt:     time.UnixMilli(tN - 5).UTC(),
		PricesAt:      time.UnixMilli(tN).UTC(),
	}

	latestPrice, isChanged := r.LatestPrice()

	assert.Equal(t, PriceFromIntUnvalidated(3000), latestPrice, "should return CurrentPrice even if stale")
	assert.False(t, isChanged, "stale price should not be marked as changed")
}

func Test_LookupMarketIndexPriceFor(t *testing.T) {

	var tN int64 = 42
	btcName := MarketName("BTC-USDT")

	pu := PriceBatchUpdate{
		PricesAt: time.UnixMilli(tN).UTC(),
		IndexPrices: MapMarketNameToMarketPriceUpdateElement{
			btcName: {
				MarketName:    btcName,
				UpdatedAt:     time.UnixMilli(tN).UTC(),
				CurrentPrice:  PriceFromIntUnvalidated(100),
				PreviousPrice: PriceFromIntUnvalidated(99),
			},
		},
	}

	r, found := pu.LookupMarketIndexPriceFor(btcName)
	require.True(t, found)
	assert.Equal(t, PriceFromIntUnvalidated(100), r.CurrentPrice)

	_, found = pu.LookupMarketIndexPriceFor(MarketName("MISSING"))
	assert.False(t, found)
}

func Test_LookupMarketLastPriceFor(t *testing.T) {

	var tN int64 = 42
	btcName := MarketName("BTC-USDT")

	pu := PriceBatchUpdate{
		PricesAt: time.UnixMilli(tN).UTC(),
		LastPrices: MapMarketNameToMarketPriceUpdateElement{
			btcName: {
				MarketName:    btcName,
				UpdatedAt:     time.UnixMilli(tN).UTC(),
				CurrentPrice:  PriceFromIntUnvalidated(200),
				PreviousPrice: PriceFromIntUnvalidated(199),
			},
		},
	}

	r, found := pu.LookupMarketLastPriceFor(btcName)
	require.True(t, found)
	assert.Equal(t, PriceFromIntUnvalidated(200), r.CurrentPrice)
}

func Test_LookupMarketMarkPriceFor(t *testing.T) {

	var tN int64 = 42
	btcName := MarketName("BTC-USDT")

	pu := PriceBatchUpdate{
		PricesAt: time.UnixMilli(tN).UTC(),
		MarkPrices: MapMarketNameToMarketPriceUpdateElement{
			btcName: {
				MarketName:    btcName,
				UpdatedAt:     time.UnixMilli(tN).UTC(),
				CurrentPrice:  PriceFromIntUnvalidated(300),
				PreviousPrice: PriceFromIntUnvalidated(299),
			},
		},
	}

	r, found := pu.LookupMarketMarkPriceFor(btcName)
	require.True(t, found)
	assert.Equal(t, PriceFromIntUnvalidated(300), r.CurrentPrice)
}

func Test_LookupLastPriceFor(t *testing.T) {

	var tN int64 = 42
	btcName := MarketName("BTC-USDT")

	pu := PriceBatchUpdate{
		PricesAt: time.UnixMilli(tN).UTC(),
		LastPrices: MapMarketNameToMarketPriceUpdateElement{
			btcName: {
				MarketName:    btcName,
				UpdatedAt:     time.UnixMilli(tN).UTC(),
				CurrentPrice:  PriceFromIntUnvalidated(400),
				PreviousPrice: PriceFromIntUnvalidated(399),
			},
		},
	}

	r, found := pu.LookupLastPriceFor(btcName)
	require.True(t, found)
	assert.Equal(t, PriceFromIntUnvalidated(400), r.CurrentPrice)
}

func Test_LookupCollateralPriceFor(t *testing.T) {

	var tN int64 = 42
	assetName := AssetName("WETH")

	pu := PriceBatchUpdate{
		PricesAt: time.UnixMilli(tN).UTC(),
		Collaterals: MapAssetNameToCollateralPriceUpdateElement{
			assetName: {
				AssetName:     assetName,
				CurrentPrice:  PriceFromIntUnvalidated(3000),
				PreviousPrice: PriceFromIntUnvalidated(2999),
				UpdatedAt:     time.UnixMilli(tN).UTC(),
			},
		},
	}

	r, found := pu.LookupCollateralPriceFor(assetName)
	require.True(t, found)
	assert.Equal(t, assetName, r.AssetName)
	assert.Equal(t, PriceFromIntUnvalidated(3000), r.CurrentPrice)
	assert.Equal(t, PriceFromIntUnvalidated(2999), r.PreviousPrice)
	assert.Equal(t, time.UnixMilli(tN).UTC(), r.UpdatedAt)

	_, found = pu.LookupCollateralPriceFor(AssetName("WBTC"))
	assert.False(t, found)
}

func Test_LookupMarketPriceFor_InvalidFeed_ReturnsFalse(t *testing.T) {

	pu := PriceBatchUpdate{}

	_, found := pu.LookupMarketPriceFor(MarketName("BTC-USDT"), PriceType("invalid"))
	assert.False(t, found)
}

func Test_LatestPrice_NegativeCurrentPrice_FallsBackToPreviousPrice(t *testing.T) {

	var tN int64 = 100

	// Negative CurrentPrice should fall back to PreviousPrice.
	r := MarketPriceUpdateSymbolResult{
		MarketName:    MarketName("BTC-USDT"),
		CurrentPrice:  PriceFromIntUnvalidated(-1),
		PreviousPrice: PriceFromIntUnvalidated(45000),
		UpdatedAt:     time.UnixMilli(tN).UTC(),
		PricesAt:      time.UnixMilli(tN).UTC(),
	}

	latestPrice, isChanged := r.LatestPrice()

	assert.Equal(t, PriceFromIntUnvalidated(45000), latestPrice, "should fall back to PreviousPrice")
	assert.False(t, isChanged, "fallback should never be marked as changed")
}
