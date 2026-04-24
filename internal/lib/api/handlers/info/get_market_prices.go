package info

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_pricing "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/pricing"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_repository "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/repository"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

var (
	errSubaccountClientNotConfiguredForFundingRateLookup = errors.New("subaccount client not configured for funding rate lookup")
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getMarketPrices
*/

// Represents the current market price for a symbol.
type MarketPriceResponse struct {
	Symbol         Symbol      `json:"symbol"`         // Trading pair symbol (e.g., "BTC-USD")
	BestBid        Price       `json:"bestBid"`        // Current market price
	BestAsk        Price       `json:"bestAsk"`        // Current market price
	MarkPrice      Price       `json:"markPrice"`      // Current mark price
	IndexPrice     Price       `json:"indexPrice"`     // Current index price
	LastPrice      Price       `json:"lastPrice"`      // Current last price
	PrevDayPrice   Price       `json:"prevDayPrice"`   // Previous day closing price (24h ago)
	Volume24h      string      `json:"volume24h"`      // 24-hour trading volume in base asset // TODO: change to use `Volume` type
	QuoteVolume24h string      `json:"quoteVolume24h"` // ... // TODO: change to use `Volume` type
	FundingRate    FundingRate `json:"fundingRate"`    // Current funding rate
	OpenInterest   Volume      `json:"openInterest"`   // Total open interest in base asset
	Timestamp      Timestamp   `json:"timestamp"`      // Epoch time (ms) when data was last updated
}

type marketPricesCache struct {
	mu       sync.RWMutex
	data     map[Symbol]MarketPriceResponse
	cachedAt time.Time
}

func (c *marketPricesCache) get(cacheTTL time.Duration) (map[Symbol]MarketPriceResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil || snx_lib_utils_time.Since(c.cachedAt) >= cacheTTL {
		return nil, false
	}

	return c.data, true
}

func (c *marketPricesCache) set(data map[Symbol]MarketPriceResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.cachedAt = snx_lib_utils_time.Now()
}

// Returns a handler that fetches market prices with an optional response-level
// cache. Set cacheTTL <= 0 to disable caching. priceCache may be nil; when
// non-nil, mark/index/last prices are read from the in-memory NATS-fed cache.
func NewMarketPricesHandler(
	cacheTTL time.Duration,
	priceCache *snx_lib_api_pricing.PriceCache,
) func(InfoContext, HandlerParams) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	cache := &marketPricesCache{}

	return func(ctx InfoContext, params HandlerParams) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
		if cacheTTL > 0 {
			if data, ok := cache.get(cacheTTL); ok {
				return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, data)
			}
		}

		data, err := fetchMarketPricesData(ctx, priceCache)
		if err != nil {
			return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Could not fetch market prices", err)
		}

		if cacheTTL > 0 {
			cache.set(data)
		}

		return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, data)
	}
}

//dd:span
func fetchMarketPricesData(
	ctx InfoContext,
	priceCache *snx_lib_api_pricing.PriceCache,
) (map[Symbol]MarketPriceResponse, error) {
	markets, err := snx_lib_api_handlers_utils.QueryMarkets(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("could not pull market configuration: %w", err)
	}

	openMarkets := make([]snx_lib_api_handlers_utils.MarketResponse, 0, len(markets))
	symbols := make([]string, 0, len(markets))
	for _, market := range markets {
		if market.IsOpen {
			openMarkets = append(openMarkets, market)
			symbols = append(symbols, string(market.Symbol))
		}
	}

	var (
		allFundingRates map[string]shopspring_decimal.Decimal
		allOI           map[snx_lib_api_handlers_utils.Symbol]snx_lib_api_handlers_utils.OITotals
		allMarketData   *v4grpc.GetAllMarketDataResponse
	)

	prefetch, prefetchCtx := errgroup.WithContext(ctx.Context)

	prefetch.Go(func() error {
		var err error
		allFundingRates, err = fetchAllFundingRates(prefetchCtx, ctx)
		if err != nil {
			ctx.Logger.Error("failed to fetch funding rates", "error", err)
		}
		return err
	})

	prefetch.Go(func() error {
		var err error
		allOI, err = snx_lib_api_handlers_utils.OpenInterestForAllMarkets(prefetchCtx, ctx.SubaccountClient)
		if err != nil {
			ctx.Logger.Error("failed to fetch all OI totals", "error", err)
		}
		return err
	})

	prefetch.Go(func() error {
		timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
		var err error
		allMarketData, err = ctx.MarketDataClient.GetAllMarketData(prefetchCtx, &v4grpc.GetAllMarketDataRequest{
			TimestampMs: timestampMs,
			TimestampUs: timestampUs,
			Symbols:     symbols,
		})
		if err != nil {
			ctx.Logger.Error("failed to fetch bulk market data", "error", err)
		}
		return err
	})

	if err := prefetch.Wait(); err != nil {
		return nil, fmt.Errorf("could not fetch market data prerequisites: %w", err)
	}

	mdBySymbol := make(map[string]*v4grpc.MarketDataResponse, len(allMarketData.Markets))
	for _, md := range allMarketData.Markets {
		mdBySymbol[md.Symbol] = md
	}

	if len(mdBySymbol) < len(openMarkets) {
		missing := make([]string, 0, len(openMarkets)-len(mdBySymbol))
		for _, m := range openMarkets {
			if mdBySymbol[string(m.Symbol)] == nil {
				missing = append(missing, string(m.Symbol))
			}
		}
		ctx.Logger.Error("bulk market data response missing symbols",
			"expected", len(openMarkets),
			"received", len(mdBySymbol),
			"missing_symbols", missing,
		)
	}

	_, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	marketPrices := make(map[Symbol]MarketPriceResponse, len(openMarkets))
	priceRepo := newPriceRepository(ctx)
	for _, market := range openMarkets {
		symbol := string(market.Symbol)
		md := mdBySymbol[symbol]
		if md == nil {
			continue
		}

		oi := allOI[market.Symbol]
		fundingRate, ok := allFundingRates[symbol]
		if !ok {
			ctx.Logger.Warn("Missing funding rate for symbol", "symbol", symbol)
		}

		resp := MarketPriceResponse{
			Symbol:       market.Symbol,
			OpenInterest: oi.Long.Add(oi.Short),
			FundingRate:  FundingRate(fundingRate.String()),
			Timestamp:    Timestamp(timestampMs),
		}

		resp.BestBid = snx_lib_api_types.PriceFromStringUnvalidated(md.BestBidPrice)
		resp.BestAsk = snx_lib_api_types.PriceFromStringUnvalidated(md.BestAskPrice)
		resp.PrevDayPrice = snx_lib_api_types.PriceFromStringUnvalidated(md.PrevDayPrice)
		resp.Volume24h = md.Volume_24H
		resp.QuoteVolume24h = md.QuoteVolume_24H

		cacheMarketName := snx_lib_core.MarketName(symbol)
		resp.MarkPrice, err = loadPriceFromCacheOrFallback(ctx, priceCache, priceRepo, cacheMarketName, snx_lib_core.PriceType_mark)
		if err != nil {
			return nil, err
		}
		resp.IndexPrice, err = loadPriceFromCacheOrFallback(ctx, priceCache, priceRepo, cacheMarketName, snx_lib_core.PriceType_index)
		if err != nil {
			return nil, err
		}
		resp.LastPrice, err = loadPriceFromCacheOrFallback(ctx, priceCache, priceRepo, cacheMarketName, snx_lib_core.PriceType_last)
		if err != nil {
			return nil, err
		}

		marketPrices[market.Symbol] = resp
	}

	return marketPrices, nil
}

func newPriceRepository(ctx InfoContext) *snx_lib_db_repository.PriceRepository {
	if ctx.Rc == nil {
		return nil
	}

	return snx_lib_db_repository.NewPriceRepository(ctx.Rc)
}

func loadPriceFromCacheOrFallback(
	ctx InfoContext,
	priceCache *snx_lib_api_pricing.PriceCache,
	priceRepo *snx_lib_db_repository.PriceRepository,
	marketName snx_lib_core.MarketName,
	priceType snx_lib_core.PriceType,
) (Price, error) {
	var fetchPriceHistory func(context.Context, string, snx_lib_core.PriceType, int) ([]snx_lib_db_repository.PriceData, error)
	if priceRepo != nil {
		fetchPriceHistory = priceRepo.GetPriceHistory
	}

	return loadPriceFromCacheOrFallbackWithFetcher(ctx, priceCache, marketName, priceType, fetchPriceHistory)
}

func loadPriceFromCacheOrFallbackWithFetcher(
	ctx InfoContext,
	priceCache *snx_lib_api_pricing.PriceCache,
	marketName snx_lib_core.MarketName,
	priceType snx_lib_core.PriceType,
	fetchPriceHistory func(context.Context, string, snx_lib_core.PriceType, int) ([]snx_lib_db_repository.PriceData, error),
) (Price, error) {
	if priceCache != nil {
		if cachedPrice := Price(priceCache.GetPrice(marketName, priceType)); cachedPrice != Price_None { // TODO: make price-cache use strong type
			return cachedPrice, nil
		}

		ctx.Logger.Debug("price cache miss, falling back to repository",
			"symbol", marketName,
			"price_type", priceType,
		)
	}

	if fetchPriceHistory == nil {
		return Price_None, nil
	}

	priceHistory, err := fetchPriceHistory(ctx.Context, string(marketName), priceType, 1)
	if err != nil {
		return Price_None, fmt.Errorf("could not get %s price for %s: %w", priceType, marketName, err)
	}

	return snx_lib_api_handlers_utils.PriceFromPriceData(priceHistory), nil
}

func fetchAllFundingRates(gCtx context.Context, ctx InfoContext) (map[string]shopspring_decimal.Decimal, error) {
	result := make(map[string]shopspring_decimal.Decimal)
	if ctx.SubaccountClient == nil {
		return result, errSubaccountClientNotConfiguredForFundingRateLookup
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.GetLatestFundingRates(gCtx, &v4grpc.GetLatestFundingRatesRequest{
		TimestampMs: timestampMs,
		TimestampUs: timestampUs,
	})
	if err != nil {
		return result, fmt.Errorf("failed to fetch funding rates: %w", err)
	}
	if grpcResp == nil {
		return result, nil
	}

	for _, info := range grpcResp.FundingRates {
		if info == nil || info.EstimatedFundingRate == "" {
			continue
		}
		rate, err := shopspring_decimal.NewFromString(info.EstimatedFundingRate)
		if err != nil {
			ctx.Logger.Warn("Invalid funding rate value", "symbol", info.Symbol, "funding_rate", info.EstimatedFundingRate, "error", err)
			continue
		}
		result[info.Symbol] = rate
	}

	return result, nil
}
