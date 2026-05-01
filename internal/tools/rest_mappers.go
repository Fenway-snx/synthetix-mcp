package tools

import (
	"strconv"

	"github.com/synthetixio/synthetix-go/types"
)

// Converts REST market rows into the public market output shape.
// Omitted REST fields stay as JSON zero values for compatibility.
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

// Converts REST price-level tuples into the public orderbook shape.
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

// Converts REST funding-rate payloads into the public output shape.
// Returns nil when the upstream payload is empty.
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

// Projects REST market-price fields onto the smaller summary shape.
// The envelope timestamp becomes the output update time.
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

// Builds the market summary block from REST price data.
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
