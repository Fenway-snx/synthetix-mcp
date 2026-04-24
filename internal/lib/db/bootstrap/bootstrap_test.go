package bootstrap

import (
	"context"
	"testing"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	"github.com/Fenway-snx/synthetix-mcp/internal/lib/core/tier"
	postgrestypes "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/types"
	snx_lib_db_testhelpers "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/testhelpers"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return snx_lib_db_testhelpers.NewDB(t, "test_bootstrap")
}

func migrateMarketTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.Market{},
		&postgrestypes.MaintenanceMarginTier{},
		&postgrestypes.MarketPrice{},
		&postgrestypes.SLPExposureLimit{},
	))
}

func seedMarkets(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	markets := []postgrestypes.Market{
		{ID: 1, Symbol: "BTC-USDT", BaseAsset: "BTC", QuoteAsset: "USDT", SettleAsset: "USDT", IsOpen: true, DefaultLeverage: 20, ContractSize: shopspring_decimal.NewFromInt(1), MinTradeAmount: shopspring_decimal.NewFromFloat(0.001), TickSize: shopspring_decimal.NewFromFloat(0.01), CreatedAt: now},
		{ID: 2, Symbol: "ETH-USDT", BaseAsset: "ETH", QuoteAsset: "USDT", SettleAsset: "USDT", IsOpen: true, DefaultLeverage: 20, ContractSize: shopspring_decimal.NewFromInt(1), MinTradeAmount: shopspring_decimal.NewFromFloat(0.01), TickSize: shopspring_decimal.NewFromFloat(0.01), CreatedAt: now},
		{ID: 3, Symbol: "DOGE-USDT", BaseAsset: "DOGE", QuoteAsset: "USDT", SettleAsset: "USDT", IsOpen: false, DefaultLeverage: 10, ContractSize: shopspring_decimal.NewFromInt(1), MinTradeAmount: shopspring_decimal.NewFromFloat(1), TickSize: shopspring_decimal.NewFromFloat(0.0001), CreatedAt: now},
	}
	for _, m := range markets {
		require.NoError(t, db.Create(&m).Error)
	}

	tiers := []postgrestypes.MaintenanceMarginTier{
		{MarketID: 1, MinPositionSize: shopspring_decimal.Zero, MaxLeverage: 125, MaintenanceMarginRatio: shopspring_decimal.NewFromFloat(0.004), InitialMarginRatio: shopspring_decimal.NewFromFloat(0.008), MaintenanceDeductionValue: shopspring_decimal.Zero},
		{MarketID: 2, MinPositionSize: shopspring_decimal.Zero, MaxLeverage: 100, MaintenanceMarginRatio: shopspring_decimal.NewFromFloat(0.005), InitialMarginRatio: shopspring_decimal.NewFromFloat(0.01), MaintenanceDeductionValue: shopspring_decimal.Zero},
	}
	for _, tier := range tiers {
		require.NoError(t, db.Create(&tier).Error)
	}
}

// --- Markets ---

func Test_LoadMarketBySymbol(t *testing.T) {
	db := openTestDB(t)
	migrateMarketTables(t, db)
	seedMarkets(t, db)
	client := NewClient(db)

	t.Run("existing symbol", func(t *testing.T) {
		market, err := client.LoadMarketBySymbol(context.Background(), "BTC-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, "BTC-USDT", market.Symbol)
		assert.True(t, market.IsOpen)
		assert.Len(t, market.MaintenanceMarginTiers, 1)
	})

	t.Run("non-existent symbol", func(t *testing.T) {
		_, err := client.LoadMarketBySymbol(context.Background(), "NONEXISTENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load market NONEXISTENT")
	})

	t.Run("closed market still loadable", func(t *testing.T) {
		market, err := client.LoadMarketBySymbol(context.Background(), "DOGE-USDT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, "DOGE-USDT", market.Symbol)
		assert.False(t, market.IsOpen)
	})
}

func Test_LoadActiveMarkets(t *testing.T) {
	db := openTestDB(t)
	migrateMarketTables(t, db)
	seedMarkets(t, db)
	client := NewClient(db)

	t.Run("returns only open markets", func(t *testing.T) {
		markets, err := client.LoadActiveMarkets(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, markets, 2)

		symbols := make([]string, len(markets))
		for i, m := range markets {
			symbols[i] = m.Symbol
		}
		assert.Contains(t, symbols, "BTC-USDT")
		assert.Contains(t, symbols, "ETH-USDT")
		assert.NotContains(t, symbols, "DOGE-USDT")
	})

	t.Run("preloads margin tiers", func(t *testing.T) {
		markets, err := client.LoadActiveMarkets(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		for _, m := range markets {
			assert.NotEmpty(t, m.MaintenanceMarginTiers)
		}
	})

	t.Run("empty when no open markets", func(t *testing.T) {
		emptyDB := openTestDB(t)
		migrateMarketTables(t, emptyDB)
		emptyClient := NewClient(emptyDB)

		markets, err := emptyClient.LoadActiveMarkets(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, markets)
	})
}

func Test_LoadAllMarkets(t *testing.T) {
	db := openTestDB(t)
	migrateMarketTables(t, db)
	seedMarkets(t, db)
	client := NewClient(db)

	t.Run("returns all markets including closed", func(t *testing.T) {
		markets, err := client.LoadAllMarkets(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, markets, 3)
	})

	t.Run("preloads margin tiers", func(t *testing.T) {
		markets, err := client.LoadAllMarkets(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		var btc postgrestypes.Market
		for _, m := range markets {
			if m.Symbol == "BTC-USDT" {
				btc = m
			}
		}
		assert.Len(t, btc.MaintenanceMarginTiers, 1)
	})
}

func Test_LoadAllMarketPrices(t *testing.T) {
	db := openTestDB(t)
	migrateMarketTables(t, db)
	client := NewClient(db)

	prices := []postgrestypes.MarketPrice{
		{Symbol: "BTC-USDT", IndexPrice: shopspring_decimal.NewFromInt(100000), LastPrice: shopspring_decimal.NewFromInt(100010), MarkPrice: shopspring_decimal.NewFromInt(100005)},
		{Symbol: "ETH-USDT", IndexPrice: shopspring_decimal.NewFromInt(3500), LastPrice: shopspring_decimal.NewFromInt(3501), MarkPrice: shopspring_decimal.NewFromInt(3500)},
	}
	for _, p := range prices {
		require.NoError(t, db.Create(&p).Error)
	}

	t.Run("returns all prices", func(t *testing.T) {
		result, err := client.LoadAllMarketPrices(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})

	t.Run("empty when no prices", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.MarketPrice{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllMarketPrices(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

func Test_LoadAllMarkets_Empty(t *testing.T) {
	db := openTestDB(t)
	migrateMarketTables(t, db)
	client := NewClient(db)

	result, err := client.LoadAllMarkets(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Empty(t, result)
}

func Test_LoadSLPExposureLimits(t *testing.T) {
	db := openTestDB(t)
	migrateMarketTables(t, db)
	seedMarkets(t, db)
	client := NewClient(db)

	limits := []postgrestypes.SLPExposureLimit{
		{MarketID: 1, MaxExposureLots: shopspring_decimal.NewFromInt(100), MaxExposureNotional: shopspring_decimal.NewFromInt(10000000), MaxExposurePercent: shopspring_decimal.NewFromInt(5)},
	}
	for _, l := range limits {
		require.NoError(t, db.Create(&l).Error)
	}

	t.Run("returns limits with preloaded market", func(t *testing.T) {
		result, err := client.LoadSLPExposureLimits(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 1)
		assert.NotNil(t, result[0].Market)
		assert.Equal(t, "BTC-USDT", result[0].Market.Symbol)
	})

	t.Run("empty when no exposure limits", func(t *testing.T) {
		emptyDB := openTestDB(t)
		migrateMarketTables(t, emptyDB)
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadSLPExposureLimits(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})

	t.Run("returns correct exposure values", func(t *testing.T) {
		result, err := client.LoadSLPExposureLimits(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		require.Len(t, result, 1)
		assert.True(t, result[0].MaxExposureLots.Equal(shopspring_decimal.NewFromInt(100)))
		assert.True(t, result[0].MaxExposureNotional.Equal(shopspring_decimal.NewFromInt(10000000)))
		assert.True(t, result[0].MaxExposurePercent.Equal(shopspring_decimal.NewFromInt(5)))
	})
}

// --- Positions ---

func Test_LoadAllPositions(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.Position{}))
	client := NewClient(db)

	now := time.Now()
	positions := []postgrestypes.Position{
		{SubAccountID: 1, Symbol: "BTC-USDT", Side: int32(snx_lib_core.PositionSideLong), Quantity: shopspring_decimal.NewFromFloat(0.5), EntryPrice: shopspring_decimal.NewFromInt(100000), CreatedAt: &now},
		{SubAccountID: 2, Symbol: "ETH-USDT", Side: int32(snx_lib_core.PositionSideShort), Quantity: shopspring_decimal.NewFromFloat(10), EntryPrice: shopspring_decimal.NewFromInt(3500), CreatedAt: &now},
	}
	for _, p := range positions {
		require.NoError(t, db.Create(&p).Error)
	}

	t.Run("returns all positions", func(t *testing.T) {
		result, err := client.LoadAllPositions(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})

	t.Run("empty when no positions", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.Position{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllPositions(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

// --- Positions (OI Totals) ---

func Test_LoadAllOITotals(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.Position{}))
	client := NewClient(db)

	now := time.Now()
	positions := []postgrestypes.Position{
		{SubAccountID: 1, Symbol: "BTC-USDT", Side: int32(snx_lib_core.PositionSideLong), Quantity: shopspring_decimal.NewFromFloat(0.5), EntryPrice: shopspring_decimal.NewFromInt(100000), CreatedAt: &now},
		{SubAccountID: 2, Symbol: "BTC-USDT", Side: int32(snx_lib_core.PositionSideLong), Quantity: shopspring_decimal.NewFromFloat(1.5), EntryPrice: shopspring_decimal.NewFromInt(100000), CreatedAt: &now},
		{SubAccountID: 3, Symbol: "BTC-USDT", Side: int32(snx_lib_core.PositionSideShort), Quantity: shopspring_decimal.NewFromFloat(3), EntryPrice: shopspring_decimal.NewFromInt(99000), CreatedAt: &now},
		{SubAccountID: 4, Symbol: "ETH-USDT", Side: int32(snx_lib_core.PositionSideLong), Quantity: shopspring_decimal.NewFromFloat(10), EntryPrice: shopspring_decimal.NewFromInt(3500), CreatedAt: &now},
		{SubAccountID: 5, Symbol: "ETH-USDT", Side: int32(snx_lib_core.PositionSideShort), Quantity: shopspring_decimal.NewFromFloat(7), EntryPrice: shopspring_decimal.NewFromInt(3500), CreatedAt: &now},
	}
	for _, p := range positions {
		require.NoError(t, db.Create(&p).Error)
	}

	t.Run("aggregates long and short OI per symbol", func(t *testing.T) {
		result, err := client.LoadAllOITotals(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)

		btc := result["BTC-USDT"]
		require.NotNil(t, btc)
		assert.True(t, btc.LongOI.Equal(shopspring_decimal.NewFromFloat(2)), "BTC long OI should be 0.5+1.5=2")
		assert.True(t, btc.ShortOI.Equal(shopspring_decimal.NewFromFloat(3)), "BTC short OI should be 3")

		eth := result["ETH-USDT"]
		require.NotNil(t, eth)
		assert.True(t, eth.LongOI.Equal(shopspring_decimal.NewFromFloat(10)), "ETH long OI should be 10")
		assert.True(t, eth.ShortOI.Equal(shopspring_decimal.NewFromFloat(7)), "ETH short OI should be 7")
	})

	t.Run("symbol field is set correctly", func(t *testing.T) {
		result, err := client.LoadAllOITotals(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		for symbol, oi := range result {
			assert.Equal(t, symbol, oi.Symbol)
		}
	})

	t.Run("empty when no positions", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.Position{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllOITotals(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})

	t.Run("only long positions produces zero short OI", func(t *testing.T) {
		longDB := openTestDB(t)
		require.NoError(t, longDB.AutoMigrate(&postgrestypes.Position{}))
		longClient := NewClient(longDB)

		longPos := postgrestypes.Position{SubAccountID: 1, Symbol: "SOL-USDT", Side: int32(snx_lib_core.PositionSideLong), Quantity: shopspring_decimal.NewFromFloat(100), EntryPrice: shopspring_decimal.NewFromInt(200), CreatedAt: &now}
		require.NoError(t, longDB.Create(&longPos).Error)

		result, err := longClient.LoadAllOITotals(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 1)
		assert.True(t, result["SOL-USDT"].LongOI.Equal(shopspring_decimal.NewFromFloat(100)))
		assert.True(t, result["SOL-USDT"].ShortOI.IsZero())
	})
}

// --- Subaccounts ---

func Test_LoadAllSubaccounts(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.SubAccount{}))
	client := NewClient(db)

	subaccounts := []postgrestypes.SubAccount{
		{ID: 100, Name: "main", WalletAddress: "0xaaa"},
		{ID: 101, Name: "alt", WalletAddress: "0xbbb"},
	}
	for _, sa := range subaccounts {
		require.NoError(t, db.Create(&sa).Error)
	}

	t.Run("returns all subaccounts", func(t *testing.T) {
		result, err := client.LoadAllSubaccounts(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})

	t.Run("empty when no subaccounts", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.SubAccount{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllSubaccounts(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})

	t.Run("returns correct fields", func(t *testing.T) {
		result, err := client.LoadAllSubaccounts(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		saMap := make(map[int64]postgrestypes.SubAccount)
		for _, sa := range result {
			saMap[sa.ID] = sa
		}
		assert.Equal(t, "main", saMap[100].Name)
		assert.Equal(t, "0xaaa", saMap[100].WalletAddress)
		assert.Equal(t, "alt", saMap[101].Name)
		assert.Equal(t, "0xbbb", saMap[101].WalletAddress)
	})
}

func Test_LoadAllSubaccountCollaterals(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.Collateral{},
		&postgrestypes.SubAccount{},
		&postgrestypes.SubAccountCollateral{},
		&postgrestypes.CollateralHaircutTier{},
	))
	client := NewClient(db)

	collateral := postgrestypes.Collateral{ID: 1, Collateral: "USDT", Market: "USDT-USD", QuantityPrecision: 8, LTV: shopspring_decimal.NewFromFloat(1.0), LLTV: shopspring_decimal.NewFromFloat(1.0)}
	require.NoError(t, db.Create(&collateral).Error)

	sa := postgrestypes.SubAccount{ID: 100, Name: "main", WalletAddress: "0xaaa"}
	require.NoError(t, db.Create(&sa).Error)

	sac := postgrestypes.SubAccountCollateral{SubAccountID: 100, CollateralID: 1, Quantity: shopspring_decimal.NewFromInt(1000)}
	require.NoError(t, db.Create(&sac).Error)

	t.Run("returns collaterals with preloaded collateral config", func(t *testing.T) {
		result, err := client.LoadAllSubaccountCollaterals(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 1)
		assert.NotNil(t, result[0].Collateral)
		assert.Equal(t, "USDT", result[0].Collateral.Collateral)
	})
}

func Test_LoadAllSubaccountCollaterals_Empty(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.Collateral{},
		&postgrestypes.SubAccountCollateral{},
		&postgrestypes.CollateralHaircutTier{},
	))
	client := NewClient(db)

	result, err := client.LoadAllSubaccountCollaterals(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Empty(t, result)
}

func Test_LoadAllSubaccountLeverages(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.SubAccount{},
		&postgrestypes.SubAccountLeverage{},
	))
	client := NewClient(db)

	sa := postgrestypes.SubAccount{ID: 100, Name: "main", WalletAddress: "0xaaa"}
	require.NoError(t, db.Create(&sa).Error)

	lev := postgrestypes.SubAccountLeverage{SubAccountID: 100, Symbol: "BTC-USDT", Leverage: 50}
	require.NoError(t, db.Create(&lev).Error)

	t.Run("returns all leverage overrides", func(t *testing.T) {
		result, err := client.LoadAllSubaccountLeverages(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 1)
		assert.Equal(t, uint32(50), result[0].Leverage)
	})

	t.Run("empty when no leverage overrides", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.SubAccountLeverage{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllSubaccountLeverages(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})

	t.Run("multiple leverages per subaccount", func(t *testing.T) {
		lev2 := postgrestypes.SubAccountLeverage{SubAccountID: 100, Symbol: "ETH-USDT", Leverage: 30}
		require.NoError(t, db.Create(&lev2).Error)

		result, err := client.LoadAllSubaccountLeverages(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})
}

// --- Orders ---

func Test_LoadOpenLimitOrders(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.OpenOrder{}))
	client := NewClient(db)

	now := time.Now()
	earlier := now.Add(-time.Minute)
	orders := []postgrestypes.OpenOrder{
		{VenueOrderId: 1, SubAccountID: 100, Symbol: "BTC-USDT", Type: int32(snx_lib_core.OrderTypeLimit), Price: "100000", Quantity: "0.5", RemainingQuantity: "0.5", CreatedAt: &earlier},
		{VenueOrderId: 2, SubAccountID: 100, Symbol: "ETH-USDT", Type: int32(snx_lib_core.OrderTypeLimit), Price: "3500", Quantity: "10", RemainingQuantity: "10", CreatedAt: &now},
		{VenueOrderId: 3, SubAccountID: 101, Symbol: "BTC-USDT", Type: 1, Price: "99000", Quantity: "1", RemainingQuantity: "1", CreatedAt: &now}, // Market order (type=1)
	}
	for _, o := range orders {
		require.NoError(t, db.Create(&o).Error)
	}

	t.Run("returns only limit orders ordered by created_at", func(t *testing.T) {
		result, err := client.LoadOpenLimitOrders(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
		assert.Equal(t, uint64(1), result[0].VenueOrderId)
		assert.Equal(t, uint64(2), result[1].VenueOrderId)
	})

	t.Run("empty when no limit orders", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.OpenOrder{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadOpenLimitOrders(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})

	t.Run("excludes non-limit types", func(t *testing.T) {
		onlyMarketDB := openTestDB(t)
		require.NoError(t, onlyMarketDB.AutoMigrate(&postgrestypes.OpenOrder{}))
		onlyMarketClient := NewClient(onlyMarketDB)

		marketOrder := postgrestypes.OpenOrder{VenueOrderId: 10, SubAccountID: 100, Symbol: "BTC-USDT", Type: 1, Price: "99000", Quantity: "1", RemainingQuantity: "1", CreatedAt: &now}
		require.NoError(t, onlyMarketDB.Create(&marketOrder).Error)

		result, err := onlyMarketClient.LoadOpenLimitOrders(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

func Test_LoadAllOpenOrders(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.OpenOrder{}))
	client := NewClient(db)

	now := time.Now()
	earlier := now.Add(-time.Minute)
	orders := []postgrestypes.OpenOrder{
		{VenueOrderId: 1, SubAccountID: 100, Symbol: "BTC-USDT", Type: 0, Price: "100000", Quantity: "0.5", RemainingQuantity: "0.5", CreatedAt: &now},
		{VenueOrderId: 2, SubAccountID: 100, Symbol: "ETH-USDT", Type: 1, Price: "3500", Quantity: "10", RemainingQuantity: "10", CreatedAt: &earlier},
	}
	for _, o := range orders {
		require.NoError(t, db.Create(&o).Error)
	}

	t.Run("returns all orders regardless of type", func(t *testing.T) {
		result, err := client.LoadAllOpenOrders(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
		assert.Equal(t, uint64(2), result[0].VenueOrderId)
	})

	t.Run("orders are sorted by created_at ascending", func(t *testing.T) {
		result, err := client.LoadAllOpenOrders(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		require.Len(t, result, 2)
		assert.True(t, result[0].CreatedAt.Before(*result[1].CreatedAt) || result[0].CreatedAt.Equal(*result[1].CreatedAt))
	})

	t.Run("empty when no open orders", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.OpenOrder{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllOpenOrders(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

// --- Collaterals ---

func Test_LoadCollateralConfigs(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.Collateral{},
		&postgrestypes.CollateralHaircutTier{},
	))
	client := NewClient(db)

	collateral := postgrestypes.Collateral{ID: 1, Collateral: "USDT", Market: "USDT-USD", QuantityPrecision: 8, LTV: shopspring_decimal.NewFromFloat(1.0), LLTV: shopspring_decimal.NewFromFloat(1.0)}
	require.NoError(t, db.Create(&collateral).Error)

	haircutTiers := []postgrestypes.CollateralHaircutTier{
		{CollateralID: 1, TierName: "tier_a", MinAmountUSDT: shopspring_decimal.Zero, CollateralValueRatio: shopspring_decimal.NewFromFloat(1.0), CollateralValueHaircut: shopspring_decimal.Zero, Status: 1},
		{CollateralID: 1, TierName: "tier_b", MinAmountUSDT: shopspring_decimal.NewFromInt(10000), CollateralValueRatio: shopspring_decimal.NewFromFloat(0.975), CollateralValueHaircut: shopspring_decimal.NewFromFloat(0.025), Status: 1},
		{CollateralID: 1, TierName: "tier_inactive", MinAmountUSDT: shopspring_decimal.Zero, CollateralValueRatio: shopspring_decimal.NewFromFloat(0.5), CollateralValueHaircut: shopspring_decimal.NewFromFloat(0.5), Status: 2},
	}
	for _, ht := range haircutTiers {
		require.NoError(t, db.Create(&ht).Error)
	}
	// GORM treats 0 as zero-value and applies default:1, so use raw SQL to set status=0
	require.NoError(t, db.Exec("UPDATE collateral_haircut_tiers SET status = 0 WHERE tier_name = 'tier_inactive'").Error)

	t.Run("returns only active haircut tiers ordered by name", func(t *testing.T) {
		result, err := client.LoadCollateralConfigs(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 1)
		assert.Len(t, result[0].CollateralHaircutTiers, 2)
		assert.Equal(t, "tier_a", result[0].CollateralHaircutTiers[0].TierName)
		assert.Equal(t, "tier_b", result[0].CollateralHaircutTiers[1].TierName)
	})
}

func Test_LoadCollateralConfigs_Empty(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.Collateral{},
		&postgrestypes.CollateralHaircutTier{},
	))
	client := NewClient(db)

	result, err := client.LoadCollateralConfigs(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Empty(t, result)
}

func Test_LoadCollateralConfigs_MultipleCollaterals(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.Collateral{},
		&postgrestypes.CollateralHaircutTier{},
	))
	client := NewClient(db)

	collaterals := []postgrestypes.Collateral{
		{ID: 1, Collateral: "USDT", Market: "USDT-USD", QuantityPrecision: 8, LTV: shopspring_decimal.NewFromFloat(1.0), LLTV: shopspring_decimal.NewFromFloat(1.0)},
		{ID: 2, Collateral: "USDC", Market: "USDC-USD", QuantityPrecision: 6, LTV: shopspring_decimal.NewFromFloat(0.95), LLTV: shopspring_decimal.NewFromFloat(0.98)},
	}
	for _, c := range collaterals {
		require.NoError(t, db.Create(&c).Error)
	}

	haircutTier := postgrestypes.CollateralHaircutTier{
		CollateralID: 1, TierName: "tier_a", MinAmountUSDT: shopspring_decimal.Zero,
		CollateralValueRatio: shopspring_decimal.NewFromFloat(1.0), CollateralValueHaircut: shopspring_decimal.Zero, Status: 1,
	}
	require.NoError(t, db.Create(&haircutTier).Error)

	result, err := client.LoadCollateralConfigs(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Len(t, result, 2)

	var usdtCollateral postgrestypes.Collateral
	for _, c := range result {
		if c.Collateral == "USDT" {
			usdtCollateral = c
		}
	}
	assert.Len(t, usdtCollateral.CollateralHaircutTiers, 1)
}

// --- SLP ---

func Test_LoadSLPMappings(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.Market{},
		&postgrestypes.Collateral{},
		&postgrestypes.SubAccount{},
		&postgrestypes.SubAccountSLP{},
		&postgrestypes.CollateralHaircutTier{},
		&postgrestypes.MaintenanceMarginTier{},
	))
	client := NewClient(db)

	market := postgrestypes.Market{ID: 1, Symbol: "BTC-USDT", BaseAsset: "BTC", QuoteAsset: "USDT", SettleAsset: "USDT", IsOpen: true, DefaultLeverage: 20, ContractSize: shopspring_decimal.NewFromInt(1), MinTradeAmount: shopspring_decimal.NewFromFloat(0.001), TickSize: shopspring_decimal.NewFromFloat(0.01)}
	require.NoError(t, db.Create(&market).Error)

	collateral := postgrestypes.Collateral{ID: 1, Collateral: "USDT", Market: "USDT-USD", QuantityPrecision: 8, LTV: shopspring_decimal.NewFromFloat(1.0), LLTV: shopspring_decimal.NewFromFloat(1.0)}
	require.NoError(t, db.Create(&collateral).Error)

	sa := postgrestypes.SubAccount{ID: 100, Name: "slp", WalletAddress: "0xslp"}
	require.NoError(t, db.Create(&sa).Error)

	marketID := int64(1)
	collateralID := int64(1)
	slp := postgrestypes.SubAccountSLP{SubAccountID: 100, MarketID: &marketID, CollateralID: &collateralID}
	require.NoError(t, db.Create(&slp).Error)

	t.Run("returns mappings with preloaded relations", func(t *testing.T) {
		result, err := client.LoadSLPMappings(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 1)
		assert.NotNil(t, result[0].Market)
		assert.NotNil(t, result[0].Collateral)
		assert.Equal(t, "BTC-USDT", result[0].Market.Symbol)
	})

	t.Run("empty when no SLP mappings", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(
			&postgrestypes.Market{},
			&postgrestypes.Collateral{},
			&postgrestypes.SubAccountSLP{},
			&postgrestypes.MaintenanceMarginTier{},
			&postgrestypes.CollateralHaircutTier{},
		))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadSLPMappings(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

// --- Funding ---

func Test_LoadFundingSettings(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.FundingSettings{}))
	client := NewClient(db)

	t.Run("returns nil when no settings exist", func(t *testing.T) {
		result, err := client.LoadFundingSettings(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Nil(t, result)
	})

	t.Run("returns settings when present", func(t *testing.T) {
		settings := postgrestypes.FundingSettings{
			BaseInterestRatePer8Hours: shopspring_decimal.NewFromFloat(0.0001),
			TargetOrderbookDepth:      10,
		}
		require.NoError(t, db.Create(&settings).Error)

		result, err := client.LoadFundingSettings(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		require.NotNil(t, result)
		assert.True(t, result.BaseInterestRatePer8Hours.Equal(shopspring_decimal.NewFromFloat(0.0001)))
		assert.Equal(t, 10, result.TargetOrderbookDepth)
	})
}

// --- Fees (bootstrap) ---

func migrateTierTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&postgrestypes.Tier{}))
	require.NoError(t, db.Exec(`
		CREATE OR REPLACE VIEW tiers_view AS
		SELECT t.*
		FROM tiers t
		INNER JOIN (
			SELECT MAX(id) AS id FROM tiers GROUP BY tier_id
		) latest ON t.id = latest.id;
	`).Error)
}

func Test_LoadWalletTierMappings(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.WalletTier{}))
	migrateTierTables(t, db)
	client := NewClient(db)

	now := time.Now()

	// Seed tier definitions: volume and custom types
	tierDefs := []postgrestypes.Tier{
		{Id: 1, TierId: "tier_0", TierType: tier.Type_volume, TierName: "Regular", MinTradeVolume: shopspring_decimal.Zero, MakerFeeRate: shopspring_decimal.NewFromFloat(0.001), TakerFeeRate: shopspring_decimal.NewFromFloat(0.002), CreatedAt: now},
		{Id: 2, TierId: "market_maker", TierType: tier.Type_custom, TierName: "Market Maker", MinTradeVolume: shopspring_decimal.Zero, MakerFeeRate: shopspring_decimal.Zero, TakerFeeRate: shopspring_decimal.Zero, CreatedAt: now},
		{Id: 3, TierId: "top_tier", TierType: tier.Type_custom, TierName: "Top Tier", MinTradeVolume: shopspring_decimal.Zero, MakerFeeRate: shopspring_decimal.Zero, TakerFeeRate: shopspring_decimal.NewFromFloat(0.00017), CreatedAt: now},
	}
	for _, td := range tierDefs {
		require.NoError(t, db.Create(&td).Error)
	}

	walletTiers := []postgrestypes.WalletTier{
		{Id: 1, WalletAddress: "0xaaa", TierId: "tier_0", CreatedAt: now},
		{Id: 2, WalletAddress: "0xaaa", TierId: "market_maker", CreatedAt: now},
		{Id: 3, WalletAddress: "0xbbb", TierId: "tier_0", CreatedAt: now},
		{Id: 4, WalletAddress: "0xccc", TierId: "top_tier", CreatedAt: now},
	}
	for _, wt := range walletTiers {
		require.NoError(t, db.Create(&wt).Error)
	}

	t.Run("returns only custom tier assignments", func(t *testing.T) {
		result, err := client.LoadWalletTierMappings(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)

		tierMap := make(map[string]tier.Id)
		for _, wt := range result {
			tierMap[wt.WalletAddress] = wt.TierId
		}
		assert.Equal(t, tier.Id("market_maker"), tierMap["0xaaa"])
		assert.Equal(t, tier.Id("top_tier"), tierMap["0xccc"])
	})

	t.Run("excludes wallets with only volume tier assignments", func(t *testing.T) {
		result, err := client.LoadWalletTierMappings(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		tierMap := make(map[string]tier.Id)
		for _, wt := range result {
			tierMap[wt.WalletAddress] = wt.TierId
		}
		_, hasBbb := tierMap["0xbbb"]
		assert.False(t, hasBbb, "wallet with only volume tier assignment should not be returned")
	})

	t.Run("empty when no wallet tiers", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.WalletTier{}))
		migrateTierTables(t, emptyDB)
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadWalletTierMappings(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

func Test_LoadAllTiers(t *testing.T) {
	db := openTestDB(t)
	migrateTierTables(t, db)
	client := NewClient(db)

	now := time.Now()
	tiers := []postgrestypes.Tier{
		{Id: 1, TierId: "tier_0", TierType: tier.Type_volume, TierName: "Starter", MinTradeVolume: shopspring_decimal.Zero, MakerFeeRate: shopspring_decimal.NewFromFloat(0.001), TakerFeeRate: shopspring_decimal.NewFromFloat(0.002), CreatedAt: now},
		{Id: 2, TierId: "tier_0", TierType: tier.Type_volume, TierName: "Starter v2", MinTradeVolume: shopspring_decimal.Zero, MakerFeeRate: shopspring_decimal.NewFromFloat(0.0009), TakerFeeRate: shopspring_decimal.NewFromFloat(0.0018), CreatedAt: now},
		{Id: 3, TierId: "tier_1", TierType: tier.Type_volume, TierName: "Bronze", MinTradeVolume: shopspring_decimal.NewFromInt(100000), MakerFeeRate: shopspring_decimal.NewFromFloat(0.0008), TakerFeeRate: shopspring_decimal.NewFromFloat(0.0016), CreatedAt: now},
	}
	for _, tier := range tiers {
		require.NoError(t, db.Create(&tier).Error)
	}

	t.Run("returns latest definition per tier_id", func(t *testing.T) {
		result, err := client.LoadAllTiers(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)

		tierMap := make(map[tier.Id]postgrestypes.Tier)
		for _, tier := range result {
			tierMap[tier.TierId] = tier
		}
		assert.Equal(t, tier.Name("Starter v2"), tierMap["tier_0"].TierName)
		assert.Equal(t, tier.Name("Bronze"), tierMap["tier_1"].TierName)
	})
}

func Test_LoadAllTiers_Empty(t *testing.T) {
	db := openTestDB(t)
	migrateTierTables(t, db)
	client := NewClient(db)

	result, err := client.LoadAllTiers(context.Background())
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Empty(t, result)
}

func Test_LoadSubaccountVolumes14Day(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&postgrestypes.TradeHistory{}))
	client := NewClient(db)

	now := time.Now()
	trades := []postgrestypes.TradeHistory{
		{SubAccountID: 100, Symbol: "BTC-USDT", FilledValue: "50000", TradedAt: now.Add(-24 * time.Hour)},
		{SubAccountID: 100, Symbol: "ETH-USDT", FilledValue: "10000", TradedAt: now.Add(-48 * time.Hour)},
		{SubAccountID: 101, Symbol: "BTC-USDT", FilledValue: "25000", TradedAt: now.Add(-72 * time.Hour)},
		{SubAccountID: 102, Symbol: "BTC-USDT", FilledValue: "99999", TradedAt: now.Add(-15 * 24 * time.Hour)}, // outside 14-day window
	}
	for _, trade := range trades {
		require.NoError(t, db.Create(&trade).Error)
	}

	t.Run("aggregates volumes within 14 day window", func(t *testing.T) {
		result, err := client.LoadSubaccountVolumes14Day(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
		assert.True(t, result[100].Equal(shopspring_decimal.NewFromInt(60000)))
		assert.True(t, result[101].Equal(shopspring_decimal.NewFromInt(25000)))
		_, exists := result[102]
		assert.False(t, exists)
	})

	t.Run("empty when no trades", func(t *testing.T) {
		emptyDB := openTestDB(t)
		require.NoError(t, emptyDB.AutoMigrate(&postgrestypes.TradeHistory{}))
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadSubaccountVolumes14Day(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

// --- Error Path Tests ---
// These tests verify that every bootstrap loader returns a wrapped error
// when the underlying database connection is broken.

func closedDBClient(t *testing.T) *Client {
	t.Helper()
	db := openTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NoError(t, sqlDB.Close())
	return NewClient(db)
}

// --- Markets Error Paths ---

func Test_LoadMarketBySymbol_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadMarketBySymbol(context.Background(), "BTC-USDT")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load market BTC-USDT")
}

func Test_LoadActiveMarkets_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadActiveMarkets(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load markets")
}

func Test_LoadAllMarkets_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllMarkets(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load all markets")
}

func Test_LoadAllMarketPrices_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllMarketPrices(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load market prices")
}

func Test_LoadSLPExposureLimits_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadSLPExposureLimits(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load SLP exposure limits")
}

// --- Fees Error Paths ---

func Test_LoadSubaccountVolumes14Day_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadSubaccountVolumes14Day(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load 14-day subaccount volumes")
}

func Test_LoadWalletTierMappings_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadWalletTierMappings(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load wallet tier mappings")
}

func Test_LoadAllTiers_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllTiers(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load tier definitions")
}

// --- Subaccounts Error Paths ---

func Test_LoadAllSubaccounts_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllSubaccounts(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load subaccounts")
}

func Test_LoadAllSubaccountCollaterals_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllSubaccountCollaterals(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load subaccount collaterals")
}

func Test_LoadAllSubaccountLeverages_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllSubaccountLeverages(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load subaccount leverages")
}

// --- Orders Error Paths ---

func Test_LoadOpenLimitOrders_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadOpenLimitOrders(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load open limit orders")
}

func Test_LoadAllOpenOrders_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllOpenOrders(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load all open orders")
}

// --- Positions Error Paths ---

func Test_LoadAllPositions_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllPositions(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load positions")
}

func Test_LoadAllOITotals_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllOITotals(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load OI totals")
}

// --- Collaterals Error Paths ---

func Test_LoadCollateralConfigs_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadCollateralConfigs(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load collateral configs")
}

// --- SLP Error Paths ---

func Test_LoadSLPMappings_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadSLPMappings(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load SLP mappings")
}

// --- Funding Error Paths ---

func Test_LoadFundingSettings_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadFundingSettings(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load funding settings")
}

// --- Exchanges ---

func migrateExchangeTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&postgrestypes.FuturesExchange{},
		&postgrestypes.SpotExchange{},
		&postgrestypes.AggregateConfig{},
		&postgrestypes.CollateralPricingConfiguration{},
	))
}

func Test_LoadAllFuturesExchanges(t *testing.T) {
	db := openTestDB(t)
	migrateExchangeTables(t, db)
	client := NewClient(db)

	exchanges := []postgrestypes.FuturesExchange{
		{Key: "binance", Name: "Binance", Enabled: true, AggregateWeight: 3, URL: "wss://binance.com"},
		{Key: "bybit", Name: "Bybit", Enabled: false, AggregateWeight: 2, URL: "wss://bybit.com"},
	}
	for _, ex := range exchanges {
		require.NoError(t, db.Create(&ex).Error)
	}

	t.Run("returns all exchanges", func(t *testing.T) {
		result, err := client.LoadAllFuturesExchanges(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})

	t.Run("returns correct fields", func(t *testing.T) {
		result, err := client.LoadAllFuturesExchanges(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		byKey := make(map[string]postgrestypes.FuturesExchange)
		for _, ex := range result {
			byKey[ex.Key] = ex
		}
		assert.Equal(t, "Binance", byKey["binance"].Name)
		assert.True(t, byKey["binance"].Enabled)
		assert.Equal(t, 3, byKey["binance"].AggregateWeight)
		assert.False(t, byKey["bybit"].Enabled)
	})

	t.Run("empty when no exchanges", func(t *testing.T) {
		emptyDB := openTestDB(t)
		migrateExchangeTables(t, emptyDB)
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllFuturesExchanges(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

func Test_LoadAllFuturesExchanges_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllFuturesExchanges(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load futures exchanges")
}

func Test_LoadAllSpotExchanges(t *testing.T) {
	db := openTestDB(t)
	migrateExchangeTables(t, db)
	client := NewClient(db)

	spotExchanges := []postgrestypes.SpotExchange{
		{Key: "binance", Name: "Binance Spot", Enabled: true, AggregateIndexWeight: 2, URL: "wss://binance.com/spot"},
		{Key: "okx", Name: "OKX Spot", Enabled: true, AggregateIndexWeight: 1, URL: "wss://okx.com/spot"},
	}
	for _, ex := range spotExchanges {
		require.NoError(t, db.Create(&ex).Error)
	}

	t.Run("returns all spot exchanges", func(t *testing.T) {
		result, err := client.LoadAllSpotExchanges(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})

	t.Run("returns correct fields", func(t *testing.T) {
		result, err := client.LoadAllSpotExchanges(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		byKey := make(map[string]postgrestypes.SpotExchange)
		for _, ex := range result {
			byKey[ex.Key] = ex
		}
		assert.Equal(t, "Binance Spot", byKey["binance"].Name)
		assert.Equal(t, 2, byKey["binance"].AggregateIndexWeight)
		assert.Equal(t, "OKX Spot", byKey["okx"].Name)
	})

	t.Run("empty when no spot exchanges", func(t *testing.T) {
		emptyDB := openTestDB(t)
		migrateExchangeTables(t, emptyDB)
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAllSpotExchanges(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

func Test_LoadAllSpotExchanges_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAllSpotExchanges(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load spot exchanges")
}

func Test_LoadAggregateConfigs(t *testing.T) {
	db := openTestDB(t)
	migrateExchangeTables(t, db)
	client := NewClient(db)

	pollMS := 1000
	configs := []postgrestypes.AggregateConfig{
		{ConfigType: "price", PollIntervalMS: &pollMS, MinExchangesRequired: 3, PriceStalenessThresholdMS: 5000},
		{ConfigType: "index", MinExchangesRequired: 2, PriceStalenessThresholdMS: 10000},
	}
	for _, cfg := range configs {
		require.NoError(t, db.Create(&cfg).Error)
	}

	t.Run("returns all aggregate configs", func(t *testing.T) {
		result, err := client.LoadAggregateConfigs(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})

	t.Run("returns correct fields", func(t *testing.T) {
		result, err := client.LoadAggregateConfigs(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		byType := make(map[string]postgrestypes.AggregateConfig)
		for _, cfg := range result {
			byType[cfg.ConfigType] = cfg
		}
		assert.Equal(t, 3, byType["price"].MinExchangesRequired)
		require.NotNil(t, byType["price"].PollIntervalMS)
		assert.Equal(t, 1000, *byType["price"].PollIntervalMS)
		assert.Equal(t, 2, byType["index"].MinExchangesRequired)
		assert.Nil(t, byType["index"].PollIntervalMS)
	})

	t.Run("empty when no configs", func(t *testing.T) {
		emptyDB := openTestDB(t)
		migrateExchangeTables(t, emptyDB)
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadAggregateConfigs(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

func Test_LoadAggregateConfigs_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadAggregateConfigs(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load aggregate configs")
}

func Test_LoadCollateralPricingConfigs(t *testing.T) {
	db := openTestDB(t)
	migrateExchangeTables(t, db)
	client := NewClient(db)

	configs := []postgrestypes.CollateralPricingConfiguration{
		{Collateral: "WETH", CoinmetricsSymbol: "weth", Status: "active", PriceConversionPair: "eth-usdt", PriceStalenessThresholdMS: 5000},
		{Collateral: "USDC", CoinmetricsSymbol: "usdc", Status: "active", PriceConversionPair: "usdc-usdt", PriceStalenessThresholdMS: 3000},
	}
	for _, cfg := range configs {
		require.NoError(t, db.Create(&cfg).Error)
	}

	t.Run("returns all collateral pricing configs", func(t *testing.T) {
		result, err := client.LoadCollateralPricingConfigs(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Len(t, result, 2)
	})

	t.Run("returns correct fields", func(t *testing.T) {
		result, err := client.LoadCollateralPricingConfigs(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		byCollateral := make(map[string]postgrestypes.CollateralPricingConfiguration)
		for _, cfg := range result {
			byCollateral[cfg.Collateral] = cfg
		}
		assert.Equal(t, "weth", byCollateral["WETH"].CoinmetricsSymbol)
		assert.Equal(t, "active", byCollateral["WETH"].Status)
		assert.Equal(t, int32(5000), byCollateral["WETH"].PriceStalenessThresholdMS)
		assert.Equal(t, "eth-usdt", byCollateral["WETH"].PriceConversionPair)
	})

	t.Run("empty when no configs", func(t *testing.T) {
		emptyDB := openTestDB(t)
		migrateExchangeTables(t, emptyDB)
		emptyClient := NewClient(emptyDB)

		result, err := emptyClient.LoadCollateralPricingConfigs(context.Background())
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Empty(t, result)
	})
}

func Test_LoadCollateralPricingConfigs_DB_ERROR(t *testing.T) {
	client := closedDBClient(t)
	_, err := client.LoadCollateralPricingConfigs(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load collateral pricing configs")
}
