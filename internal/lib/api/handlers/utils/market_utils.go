package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_db_repository "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/repository"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const marketsCacheTTL = 60 * time.Second

// Stores a cached markets response and when it was written.
// cachedAt is compared against marketsCacheTTL to determine staleness.
type cachedMarkets struct {
	data     []MarketResponse
	cachedAt time.Time
}

var (
	marketsCacheMu      sync.RWMutex
	marketsCacheEntries = make(map[bool]*cachedMarkets)
)

// Clears the in-memory markets cache. Intended for use in tests to avoid
// cross-test cache interference.
func ResetMarketsCache() {
	marketsCacheMu.Lock()
	clear(marketsCacheEntries)
	marketsCacheMu.Unlock()
}

func ConvertProtoToMarketResponse(protoMarket *v4grpc.Market) (MarketResponse, error) {
	// Convert strings to decimals for MarketResponse
	tickSize, err := shopspring_decimal.NewFromString(protoMarket.TickSize)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid tick_size: %w", err)
	}

	minTradeAmount, err := shopspring_decimal.NewFromString(protoMarket.MinTradeAmount)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid min_trade_amount: %w", err)
	}

	contractSize, err := shopspring_decimal.NewFromString(protoMarket.ContractSize)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid contract_size: %w", err)
	}

	maxMarketOrderAmount, err := shopspring_decimal.NewFromString(protoMarket.MaxMarketOrderAmount)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid max_market_order_amount: %w", err)
	}

	maxLimitOrderAmount, err := shopspring_decimal.NewFromString(protoMarket.MaxLimitOrderAmount)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid max_limit_order_amount: %w", err)
	}

	minOrderPrice := shopspring_decimal.Zero // Not in proto, default to 0

	limitOrderPriceCapRatio, err := shopspring_decimal.NewFromString(protoMarket.LimitOrderPriceCapRatio)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid limit_order_price_cap_ratio: %w", err)
	}

	limitOrderPriceFloorRatio, err := shopspring_decimal.NewFromString(protoMarket.LimitOrderPriceFloorRatio)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid limit_order_price_floor_ratio: %w", err)
	}

	marketOrderPriceCapRatio, err := shopspring_decimal.NewFromString(protoMarket.MarketOrderPriceCapRatio)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid market_order_price_cap_ratio: %w", err)
	}

	marketOrderPriceFloorRatio, err := shopspring_decimal.NewFromString(protoMarket.MarketOrderPriceFloorRatio)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid market_order_price_floor_ratio: %w", err)
	}

	liquidationClearanceFee, err := shopspring_decimal.NewFromString(protoMarket.LiquidationClearanceFee)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid liquidation_clearance_fee: %w", err)
	}

	minNotionalValue, err := shopspring_decimal.NewFromString(protoMarket.MinNotionalValue)
	if err != nil {
		return MarketResponse{}, fmt.Errorf("invalid min_notional_value: %w", err)
	}

	// Convert maintenance margin tiers
	maintenanceMarginTiers := make([]MaintenanceMarginTier, len(protoMarket.MaintenanceMarginTiers))
	for j, protoTier := range protoMarket.MaintenanceMarginTiers {
		maintenanceMarginRatio, err := shopspring_decimal.NewFromString(protoTier.MaintenanceMarginRatio)
		if err != nil {
			return MarketResponse{}, fmt.Errorf("invalid maintenance_margin_ratio for tier %d: %w", j, err)
		}

		// Use initial margin ratio from market config
		initialMarginRatio, err := shopspring_decimal.NewFromString(protoTier.InitialMarginRatio)
		if err != nil {
			return MarketResponse{}, fmt.Errorf("invalid initial_margin_ratio for tier %d: %w", j, err)
		}

		maintenanceDeductionValue := shopspring_decimal.Zero
		if protoTier.MaintenanceDeductionValue != "" {
			maintenanceDeductionValue, err = shopspring_decimal.NewFromString(protoTier.MaintenanceDeductionValue)
			if err != nil {
				return MarketResponse{}, fmt.Errorf("invalid maintenance_deduction_value for tier %d: %w", j, err)
			}
		}

		maintenanceMarginTiers[j] = MaintenanceMarginTier{
			MinPositionSize:              protoTier.MinPositionSize,
			MaxPositionSize:              protoTier.MaxPositionSize, // Empty string if unlimited
			MaxLeverage:                  protoTier.MaxLeverage,
			InitialMarginRequirement:     initialMarginRatio,
			MaintenanceMarginRequirement: maintenanceMarginRatio,
			MaintenanceDeductionValue:    maintenanceDeductionValue,
		}
	}

	return MarketResponse{
		Symbol:                     Symbol(protoMarket.Symbol),
		Description:                protoMarket.Description,
		BaseAsset:                  snx_lib_api_types.AssetNameFromStringUnvalidated(protoMarket.BaseAsset),
		QuoteAsset:                 snx_lib_api_types.AssetNameFromStringUnvalidated(protoMarket.QuoteAsset),
		IsOpen:                     protoMarket.IsOpen,
		IsCloseOnly:                false, // Not available in proto, default to false
		PriceExponent:              protoMarket.PriceExponent,
		QuantityExponent:           protoMarket.QuantityExponent,
		PriceIncrement:             tickSize,
		MinOrderSize:               minTradeAmount,
		OrderSizeIncrement:         shopspring_decimal.Zero, // Not in proto, default to 0
		ContractSize:               uint32(contractSize.BigInt().Uint64()),
		MaxMarketOrderSize:         maxMarketOrderAmount,
		MaxLimitOrderSize:          maxLimitOrderAmount,
		MinOrderPrice:              minOrderPrice,
		LimitOrderPriceCapRatio:    limitOrderPriceCapRatio,
		LimitOrderPriceFloorRatio:  limitOrderPriceFloorRatio,
		MarketOrderPriceCapRatio:   marketOrderPriceCapRatio,
		MarketOrderPriceFloorRatio: marketOrderPriceFloorRatio,
		LiquidationClearanceFee:    liquidationClearanceFee,
		MinNotionalValue:           minNotionalValue,
		MaintenanceMarginTiers:     maintenanceMarginTiers,
	}, nil
}

func OpenInterestForMarket(
	ctx context.Context,
	subaccountClient v4grpc.SubaccountServiceClient,
	symbol Symbol,
) (
	longOpenInterest Volume,
	shortOpenInterest Volume,
	err error,
) {

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	req := &v4grpc.GetPositionsBySymbolRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Symbol:      string(symbol),
	}

	grpcResp, err := subaccountClient.GetPositionsBySymbol(ctx, req)
	if err == nil {

		// NOTE: we deliberately ignore failure - which is really only
		// theoretical - for now; a later implementation will not be using
		// strings in the core.
		longOpenInterest, _ = shopspring_decimal.NewFromString(grpcResp.TotalLongQuantity)
		shortOpenInterest, _ = shopspring_decimal.NewFromString(grpcResp.TotalShortQuantity)
	}

	return
}

// OITotals holds long and short open interest for a single symbol.
type OITotals struct {
	Long  shopspring_decimal.Decimal
	Short shopspring_decimal.Decimal
}

// OpenInterestForAllMarkets fetches OI totals for every symbol in a single gRPC call.
func OpenInterestForAllMarkets(
	ctx context.Context,
	subaccountClient v4grpc.SubaccountServiceClient,
) (map[Symbol]OITotals, error) {
	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()

	resp, err := subaccountClient.GetAllOITotals(ctx, &v4grpc.GetAllOITotalsRequest{
		TimestampMs: timestampMs,
		TimestampUs: timestampUs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all OI totals: %w", err)
	}

	result := make(map[Symbol]OITotals, len(resp.Items))
	for _, item := range resp.Items {
		longQty, err := shopspring_decimal.NewFromString(item.TotalLongQuantity)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid total_long_quantity for symbol %s: %w",
				item.Symbol,
				err,
			)
		}
		shortQty, err := shopspring_decimal.NewFromString(item.TotalShortQuantity)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid total_short_quantity for symbol %s: %w",
				item.Symbol,
				err,
			)
		}
		result[Symbol(item.Symbol)] = OITotals{Long: longQty, Short: shortQty}
	}
	return result, nil
}

func PriceFromPriceData(
	data []snx_lib_db_repository.PriceData,
) Price {
	if len(data) == 0 {
		return snx_lib_api_types.PriceFromDecimalUnvalidated(shopspring_decimal.Zero)
	} else {
		return snx_lib_api_types.PriceFromDecimalUnvalidated(data[0].Price)
	}
}

func getCachedMarkets(activeOnly bool) ([]MarketResponse, bool) {
	marketsCacheMu.RLock()
	defer marketsCacheMu.RUnlock()

	entry, ok := marketsCacheEntries[activeOnly]
	if !ok || snx_lib_utils_time.Since(entry.cachedAt) >= marketsCacheTTL {
		return nil, false
	}

	return entry.data, true
}

func setCachedMarkets(activeOnly bool, markets []MarketResponse) {
	marketsCacheMu.Lock()
	defer marketsCacheMu.Unlock()

	marketsCacheEntries[activeOnly] = &cachedMarkets{
		data:     markets,
		cachedAt: snx_lib_utils_time.Now(),
	}
}

func QueryMarkets(ctx InfoContext, activeOnly bool) ([]MarketResponse, error) {
	if data, ok := getCachedMarkets(activeOnly); ok {
		return data, nil
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	req := &v4grpc.GetAllMarketsRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		ActiveOnly:  activeOnly,
	}

	grpcResp, err := ctx.MarketConfigClient.GetAllMarkets(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get markets from marketconfig service: %w", err)
	}

	markets := make([]MarketResponse, len(grpcResp.Markets))
	for i, protoMarket := range grpcResp.Markets {
		market, err := ConvertProtoToMarketResponse(protoMarket)
		if err != nil {
			return nil, fmt.Errorf("failed to convert market %s: %w", protoMarket.Symbol, err)
		}
		markets[i] = market
	}

	setCachedMarkets(activeOnly, markets)

	return markets, nil
}
