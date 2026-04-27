package tools

import (
	"strconv"

	"github.com/synthetixio/synthetix-go/types"
)

// MapMarketFromREST converts a REST MarketResponse into the
// MarketOutput shape emitted by list_markets / get_market_summary /
// market://specs/{symbol}. Field coverage drifts deliberately because
// the REST /v1/info getMarkets response is richer (explicit
// price/quantity exponents and increment strings) but omits several fields
// (DefaultLeverage, FundingRateCap/Floor, ImpactNotionalUsd,
// SettleAsset, uint64 Id). The omitted fields appear in the output
// as JSON zero values so existing clients keep parsing successfully.
func MapMarketFromREST(market *types.MarketResponse) MarketOutput {
	if market == nil {
		return MarketOutput{}
	}

	tiers := make([]MaintenanceTierOutput, 0, len(market.MaintenanceMarginTiers))
	for _, tier := range market.MaintenanceMarginTiers {
		tiers = append(tiers, MaintenanceTierOutput{
			InitialMarginRatio:     tier.InitialMarginFraction,
			MaintenanceMarginRatio: tier.MaintenanceMarginFraction,
			MaxPositionSize:        tier.UpperBound,
			MinPositionSize:        tier.LowerBound,
		})
	}

	return MarketOutput{
		BaseAsset:              market.BaseAsset,
		ContractSize:           strconv.FormatUint(uint64(market.ContractSize), 10),
		Description:            market.Description,
		IsOpen:                 market.IsOpen,
		MaintenanceMarginTiers: tiers,
		MinNotionalValue:       market.MinNotionalValue,
		MinTradeAmount:         market.MinOrderSize,
		QuoteAsset:             market.QuoteAsset,
		Symbol:                 market.Symbol,
		TickSize:               market.PriceIncrement,
	}
}

// MapPriceLevelsFromREST converts REST PriceLevel tuples into the
// orderbook wire shape emitted by get_orderbook.
func MapPriceLevelsFromREST(levels []types.PriceLevel) []priceLevelOutput {
	out := make([]priceLevelOutput, 0, len(levels))
	for _, lvl := range levels {
		out = append(out, priceLevelOutput{
			Price:    lvl.Price,
			Quantity: lvl.Quantity,
		})
	}
	return out
}

// MapFundingRateFromREST converts a REST FundingRateResponse into
// the FundingRateEntry shape emitted by get_funding_rate and
// get_market_summary. Returns nil when the upstream payload is empty.
func MapFundingRateFromREST(resp *types.FundingRateResponse) *FundingRateEntry {
	if resp == nil {
		return nil
	}
	return &FundingRateEntry{
		EstimatedFundingRate: resp.EstimatedFundingRate,
		FundingIntervalMs:    resp.FundingInterval,
		LastSettlementRate:   resp.LastSettlementRate,
		LastSettlementTime:   resp.LastSettlementTime,
		NextFundingTime:      resp.NextFundingTime,
		Symbol:               resp.Symbol,
	}
}

// MarketPriceFromREST projects REST MarketPriceResponse fields onto
// the smaller marketPriceOutput shape used in get_market_summary.
// The REST payload has no per-field update timestamp — the envelope
// returns a single Timestamp for the whole payload — so UpdatedAt
// carries that value when present.
func MarketPriceFromREST(p *types.MarketPriceResponse) marketPriceOutput {
	if p == nil {
		return marketPriceOutput{}
	}
	return marketPriceOutput{
		IndexPrice: p.IndexPrice,
		LastPrice:  p.LastPrice,
		MarkPrice:  p.MarkPrice,
		UpdatedAt:  p.Timestamp,
	}
}

// MarketSummaryFromREST builds the summaryOutput block of
// get_market_summary from the REST getMarketPrices response.
func MarketSummaryFromREST(p *types.MarketPriceResponse) summaryOutput {
	if p == nil {
		return summaryOutput{}
	}
	return summaryOutput{
		BestAskPrice:    p.BestAsk,
		BestBidPrice:    p.BestBid,
		LastTradedPrice: p.LastPrice,
		LastTradedTime:  p.Timestamp,
		PrevDayPrice:    p.PrevDayPrice,
		QuoteVolume24h:  p.QuoteVolume24h,
		Volume24h:       p.Volume24h,
	}
}
