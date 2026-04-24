package core

import (
	"strings"
	"time"
)

// Represents a price update for a market.
//
// Invariants:
//   - Zero and negative prices are rejected at ingestion by the
//     PriceBatchPublisher, so CurrentPrice should always be positive for
//     any entry that has been populated by the publisher.
//   - For new entries, PreviousPrice is initialised to the same value as
//     CurrentPrice to avoid a zero-fallback in LatestPrice().
//
// Note:
// The three fields `CurrentPrice`, `PreviousPrice`, and `UpdatedAt` along
// with the `PriceBatchUpdate#PricesAt` field together may be used to understand
// precisely the recency of the current price for a given market, according
// to the following logic:
//
//  1. If `UpdatedAt` == `PricesAt` this means that `CurrentPrice` was set in
//     the current batch update;
//  2. If `UpdatedAt` != `PricesAt` (in which case it MUST be the case that
//     `UpdatedAt` < `PricesAt`), then `CurrentPrice` has not changed in the
//     current batch update. `CurrentPrice` retains its last known value, and
//     `PreviousPrice` holds the value prior to the most recent change.
//     `UpdatedAt` holds the time at which `CurrentPrice` was last set.
type MarketPriceUpdateElement struct {
	MarketName      MarketName `json:"market_name"`                 // Name of the market for which this element applies.
	UpdatedAt       time.Time  `json:"updated_at"`                  // Application-level timestamp. If different from the `PricesAt` time of its enclosing packet indicates that the value of `"currentPrice"` has not changed in the latest delivery, in which case this value specifies when the value provided in `"previousPrice"` was obtained.
	CurrentPrice    Price      `json:"current_price"`               // The current known value of the price, or Price_Zero if value was not obtained for some reason.
	PreviousPrice   Price      `json:"previous_price"`              // The previous known value of the price, or Price_Zero if value was not obtained for some reason. If `CurrentPrice == PreviousPrice` then the price for this symbol has not changed.
	Source          string     `json:"source,omitempty"`            // Optional string describing the source, e.g. "Binance".
	FundingRate     *Price     `json:"funding_rate,omitempty"`      // Estimated hourly funding rate; present only for mark-price elements when a funding rate is available.
	NextFundingTime *int64     `json:"next_funding_time,omitempty"` // Unix ms of next UTC hour boundary; present only when FundingRate is set.
}

type MapMarketNameToMarketPriceUpdateElement map[MarketName]MarketPriceUpdateElement

// Represents a price update for a collateral asset.
type CollateralPriceUpdateElement struct {
	AssetName     AssetName `json:"asset_name"`
	CurrentPrice  Price     `json:"current_price"`
	PreviousPrice Price     `json:"previous_price"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type MapAssetNameToCollateralPriceUpdateElement map[AssetName]CollateralPriceUpdateElement

func NormalizeCollateralConversionPair(pair string) string {
	normalized := strings.ToLower(strings.TrimSpace(pair))
	if normalized == "" {
		return strings.ToLower(NominatedCollateral)
	}
	return normalized
}

// The data sent from the Pricing Service to its recipients.
//
// Note:
// For consumers (such as in the Trading Service) an instance of this type
// may be shared among multiple contemporaneous goroutines, and so should be
// treated as strictly immutable. It is recommended to use the accessor
// methods such as `LookupMarketPriceFor()`.
type PriceBatchUpdate struct {
	PricesAt    time.Time                                  `json:"prices_at"`    // Application-level timestamp representing the time of this batch.
	IndexPrices MapMarketNameToMarketPriceUpdateElement    `json:"index_prices"` // The index prices for supported markets.
	LastPrices  MapMarketNameToMarketPriceUpdateElement    `json:"last_prices"`  // The last prices for supported markets.
	MarkPrices  MapMarketNameToMarketPriceUpdateElement    `json:"mark_prices"`  // The mark prices for supported markets.
	Collaterals MapAssetNameToCollateralPriceUpdateElement `json:"collaterals"`  // The prices for collateral assets.
}

// Result structure querying a `PriceBatchUpdate`.
type MarketPriceUpdateSymbolResult struct {
	MarketName    MarketName
	CurrentPrice  Price
	PreviousPrice Price
	UpdatedAt     time.Time
	PricesAt      time.Time
}

// Obtains the most recent valid price for the result, and indicates whether
// it was changed in the most recent batch update.
//
// The function uses a validity-based check: it returns CurrentPrice whenever
// it is positive (which covers both "changed in this batch" and "unchanged
// but still valid from a prior batch"). Only if CurrentPrice is invalid
// (zero or negative) does it fall back to PreviousPrice.
//
// Under normal operation both CurrentPrice and PreviousPrice are guaranteed
// positive by the PriceBatchPublisher's ingestion guards, so the fallback
// path should not be reached.
func (r *MarketPriceUpdateSymbolResult) LatestPrice() (latestPrice Price, isChangedInMostRecentBatch bool) {

	if r.CurrentPrice.IsPositive() {
		isChangedInMostRecentBatch = r.UpdatedAt.Equal(r.PricesAt)
		latestPrice = r.CurrentPrice
	} else {
		// CurrentPrice is invalid (zero/negative), fall back to PreviousPrice.
		latestPrice = r.PreviousPrice
	}

	return
}

// Looks up the price, if any present, for the given market in "index"
// prices.
func (pu *PriceBatchUpdate) LookupMarketIndexPriceFor(marketName MarketName) (
	result MarketPriceUpdateSymbolResult,
	found bool,
) {
	return pu._LookupMarketInMap(pu.IndexPrices, marketName)
}

// Looks up the price, if any present, for the given market in "last"
// prices.
func (pu *PriceBatchUpdate) LookupMarketLastPriceFor(marketName MarketName) (
	result MarketPriceUpdateSymbolResult,
	found bool,
) {
	return pu._LookupMarketInMap(pu.LastPrices, marketName)
}

// Looks up the price, if any present, for the given market in "mark"
// prices.
func (pu *PriceBatchUpdate) LookupMarketMarkPriceFor(marketName MarketName) (
	result MarketPriceUpdateSymbolResult,
	found bool,
) {
	return pu._LookupMarketInMap(pu.MarkPrices, marketName)
}

// Looks up the price, if any present, for the given collateral asset.
func (pu *PriceBatchUpdate) LookupCollateralPriceFor(assetName AssetName) (
	result CollateralPriceUpdateElement,
	found bool,
) {
	result, found = pu.Collaterals[assetName]
	return
}

// Looks up the price, if any present, for the given market and type.
func (pu *PriceBatchUpdate) LookupMarketPriceFor(marketName MarketName, feed PriceType) (MarketPriceUpdateSymbolResult, bool) {
	var m MapMarketNameToMarketPriceUpdateElement

	switch feed {
	case PriceType_index:

		m = pu.IndexPrices
	case PriceType_last:

		m = pu.LastPrices
	case PriceType_mark:

		m = pu.MarkPrices
	default:

		return MarketPriceUpdateSymbolResult{}, false
	}

	return pu._LookupMarketInMap(m, marketName)
}

func (pu *PriceBatchUpdate) _LookupMarketInMap(m MapMarketNameToMarketPriceUpdateElement, marketName MarketName) (
	result MarketPriceUpdateSymbolResult,
	found bool,
) {

	record, found := m[marketName]

	if found {
		result = MarketPriceUpdateSymbolResult{
			MarketName:    record.MarketName,
			CurrentPrice:  record.CurrentPrice,
			PreviousPrice: record.PreviousPrice,
			UpdatedAt:     record.UpdatedAt,
			PricesAt:      pu.PricesAt,
		}
	}

	return
}

func (pu *PriceBatchUpdate) LookupLastPriceFor(marketName MarketName) (
	result MarketPriceUpdateSymbolResult,
	found bool,
) {

	result, found = pu.LookupMarketPriceFor(marketName, PriceType_last)

	return
}
