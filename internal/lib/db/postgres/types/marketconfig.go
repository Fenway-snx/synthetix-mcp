package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_utils_decimal "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/decimal"
)

// Market represents a market configuration
type Market struct {
	// Boilerplate DB fields
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"` // NOTE: currently unused

	// Market descriptors
	//
	// Used by:
	// - *;
	Symbol      string `gorm:"uniqueIndex;not null;type:varchar(20)" json:"symbol"`
	Description string `gorm:"not null;type:varchar(250)" json:"description"`
	BaseAsset   string `gorm:"not null;type:varchar(10)" json:"base_asset"`
	QuoteAsset  string `gorm:"not null;type:varchar(10)" json:"quote_asset"`
	SettleAsset string `gorm:"not null;type:varchar(10)" json:"settle_asset"`

	// Market Status
	//
	// Used by:
	// - Market Config Service;
	// - Pricing Service;
	// - Trading Service;
	IsOpen           bool `gorm:"not null;" json:"is_open"`
	IsCloseOnly      bool `gorm:"not null;default:false" json:"is_close_only"`      // NOTE: currently unused
	IsCollateralOnly bool `gorm:"not null;default:false" json:"is_collateral_only"` // NOTE: currently unused
	IsPreToken       bool `gorm:"not null;default:false" json:"is_pre_token"`

	// T.B.C.
	//
	// Used by:
	// - *;
	DefaultLeverage  uint8 `gorm:"not null" json:"default_leverage"`
	PriceExponent    uint8 `gorm:"not null" json:"price_exponent"`
	QuantityExponent uint8 `gorm:"not null" json:"quantity_exponent"`

	// T.B.C.
	//
	// Used by:
	// - Market Config Service;
	// - Pricing Service;
	// - Trading Service;
	ContractSize               shopspring_decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"contract_size"`
	MinTradeAmount             shopspring_decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"min_trade_amount"`
	TickSize                   shopspring_decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"tick_size"`
	MinNotionalValue           shopspring_decimal.Decimal `gorm:"type:decimal(20,8)" json:"min_notional_value"`
	LimitOrderPriceCapRatio    shopspring_decimal.Decimal `gorm:"type:decimal(6,4);not null" json:"limit_order_price_cap_ratio"`
	LimitOrderPriceFloorRatio  shopspring_decimal.Decimal `gorm:"type:decimal(6,4);not null" json:"limit_order_price_floor_ratio"`
	MaxMarketOrderAmount       shopspring_decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"max_market_order_amount"`
	MaxLimitOrderAmount        shopspring_decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"max_limit_order_amount"`
	FundingRateImpactPrice     Price                      `gorm:"type:decimal(20,8);not null" json:"funding_rate_impact_price"`
	FundingRateCap             shopspring_decimal.Decimal `gorm:"type:decimal(6,4)" json:"funding_rate_cap"`
	FundingRateFloor           shopspring_decimal.Decimal `gorm:"type:decimal(6,4)" json:"funding_rate_floor"`
	LiquidationClearanceFee    shopspring_decimal.Decimal `gorm:"type:decimal(6,4);not null" json:"liquidation_clearance_fee"`
	MarketOrderPriceCapRatio   shopspring_decimal.Decimal `gorm:"type:decimal(6,4);not null" json:"market_order_price_cap_ratio"`
	MarketOrderPriceFloorRatio shopspring_decimal.Decimal `gorm:"type:decimal(6,4);not null" json:"market_order_price_floor_ratio"`

	// Symbols (in other exchanges)
	//
	// Used by:
	// - Market Config Service;
	// - Pricing Service;
	// - Trading Service;
	BinanceFuturesSymbol string `gorm:"type:varchar(20)" json:"binance_futures_symbol"`
	BinanceSpotSymbol    string `gorm:"type:varchar(20)" json:"binance_spot_symbol"`
	BybitFuturesSymbol   string `gorm:"type:varchar(20)" json:"bybit_futures_symbol"`
	BybitSpotSymbol      string `gorm:"type:varchar(20)" json:"bybit_spot_symbol"`
	CoinmetricsSymbol    string `gorm:"type:varchar(20)" json:"coinmetrics_symbol"`
	GateioFuturesSymbol  string `gorm:"column:gate_io_futures_symbol;type:varchar(20)" json:"gateio_futures_symbol"`
	GateioSpotSymbol     string `gorm:"column:gate_io_spot_symbol;type:varchar(20)" json:"gateio_spot_symbol"`
	HyperliquidSymbol    string `gorm:"type:varchar(20)" json:"hyperliquid_symbol"`
	KucoinFuturesSymbol  string `gorm:"type:varchar(20)" json:"kucoin_futures_symbol"`
	KucoinSpotSymbol     string `gorm:"type:varchar(20)" json:"kucoin_spot_symbol"`
	MexcFuturesSymbol    string `gorm:"column:mexc_futures_symbol;type:varchar(20)" json:"mexc_futures_symbol"`
	MexcSpotSymbol       string `gorm:"column:mexc_spot_symbol;type:varchar(20)" json:"mexc_spot_symbol"`
	OkxFuturesSymbol     string `gorm:"column:okx_futures_symbol;type:varchar(30)" json:"okx_futures_symbol"`
	OkxSpotSymbol        string `gorm:"column:okx_spot_symbol;type:varchar(30)" json:"okx_spot_symbol"`
	PythSymbol           string `gorm:"type:varchar(50)" json:"pyth_symbol"`

	// Price multipliers for markets with non-standard denominations (e.g., 1000PEPE = 1000 PEPE)
	//
	// Used by:
	// - Market Config Service;
	// - Pricing Service;
	// - Trading Service;
	BinanceSpotPriceMultiplier   float64 `gorm:"type:decimal(10,4);default:0" json:"binance_spot_price_multiplier"`
	BybitSpotPriceMultiplier     float64 `gorm:"type:decimal(10,4);default:0" json:"bybit_spot_price_multiplier"`
	CoinmetricsPriceMultiplier   float64 `gorm:"type:decimal(10,4);default:0" json:"coinmetrics_price_multiplier"`
	GateioFuturesPriceMultiplier float64 `gorm:"column:gate_io_futures_price_multiplier;type:decimal(10,4);default:0" json:"gateio_futures_price_multiplier"`
	GateioSpotPriceMultiplier    float64 `gorm:"column:gate_io_spot_price_multiplier;type:decimal(10,4);default:0" json:"gateio_spot_price_multiplier"`
	KucoinFuturesPriceMultiplier float64 `gorm:"type:decimal(10,4);default:0" json:"kucoin_futures_price_multiplier"`
	KucoinSpotPriceMultiplier    float64 `gorm:"type:decimal(10,4);default:0" json:"kucoin_spot_price_multiplier"`
	MexcFuturesPriceMultiplier   float64 `gorm:"column:mexc_futures_price_multiplier;type:decimal(10,4);default:0" json:"mexc_futures_price_multiplier"`
	MexcSpotPriceMultiplier      float64 `gorm:"column:mexc_spot_price_multiplier;type:decimal(10,4);default:0" json:"mexc_spot_price_multiplier"`
	OkxFuturesPriceMultiplier    float64 `gorm:"column:okx_futures_price_multiplier;type:decimal(10,4);default:0" json:"okx_futures_price_multiplier"`
	OkxSpotPriceMultiplier       float64 `gorm:"column:okx_spot_price_multiplier;type:decimal(10,4);default:0" json:"okx_spot_price_multiplier"`

	// OI Cap fields for per-market open interest limits
	//
	// Used by:
	// - Market Config Service;
	// - Matching Engine;
	OINotionalCap    shopspring_decimal.Decimal `gorm:"type:decimal(20,8);default:0" json:"oi_notional_cap"`      // Max notional OI in USD (0 = no cap)
	OISizeCap        shopspring_decimal.Decimal `gorm:"type:decimal(20,8);default:0" json:"oi_size_cap"`          // Max quantity OI in contracts (0 = no cap)
	OICheckThreshold shopspring_decimal.Decimal `gorm:"type:decimal(6,4);default:0.80" json:"oi_check_threshold"` // Threshold ratio to start checking (e.g., 0.80 = 80%)

	// Relationship
	//
	// Used by:
	// - Market Config Service;
	// - Trading Service;
	MaintenanceMarginTiers []MaintenanceMarginTier `gorm:"foreignKey:MarketID;constraint:OnDelete:CASCADE" json:"maintenance_margin_tiers,omitempty"`

	// Funding rate config
	//
	// Used by:
	// - Market Config Service;
	// - Pricing Service;
	ImpactNotionalUSD shopspring_decimal.Decimal `json:"impact_notional_usd"`
}

// TableName sets the table name for the Market model
func (Market) TableName() string {
	return "markets"
}

// MarketMarginTier represents maintenance margin tiers for a market
type MaintenanceMarginTier struct {
	ID                        uint64                      `gorm:"primaryKey;autoIncrement" json:"id"`
	MarketID                  uint64                      `gorm:"not null;index" json:"market_id"`
	MinPositionSize           shopspring_decimal.Decimal  `gorm:"type:decimal(20,2);not null" json:"min_position_size"`
	MaxPositionSize           *shopspring_decimal.Decimal `gorm:"type:decimal(20,2)" json:"max_position_size"` // nullable for the last tier
	MaxLeverage               uint8                       `gorm:"not null" json:"max_leverage"`
	MaintenanceMarginRatio    shopspring_decimal.Decimal  `gorm:"type:decimal(6,4);not null" json:"maintenance_margin_ratio"`     // Should be saved as decimal ratio, e.g. 0.005 for 0.5%
	InitialMarginRatio        shopspring_decimal.Decimal  `gorm:"type:decimal(6,4);not null" json:"initial_margin_ratio"`         // This is 2x MaintenanceMarginRatio. Should be saved as decimal ratio, e.g. 0.01 for 1%
	MaintenanceDeductionValue shopspring_decimal.Decimal  `gorm:"type:decimal(20,8);not null" json:"maintenance_deduction_value"` // Should be saved as USDT value, e.g. 100 for $100
	CreatedAt                 time.Time                   `json:"created_at"`
	UpdatedAt                 time.Time                   `json:"updated_at"`

	// Relationship
	Market *Market `gorm:"foreignKey:MarketID;constraint:OnDelete:CASCADE" json:"market,omitempty"`
}

// TableName sets the table name for the MarketMarginTier model
func (MaintenanceMarginTier) TableName() string {
	return "maintenance_margin_tiers"
}

// GetMaintenanceMarginRate calculates the maintenance margin rate from max leverage
// Formula: maintenance_margin_rate = 1 / (2 * max_leverage)
// MaxLeverage must be >= 1 as enforced by ValidateMaintenanceMarginTiers
func (mt *MaintenanceMarginTier) GetMaintenanceMarginRate() shopspring_decimal.Decimal {
	MaintenanceMarginRate, _ := snx_lib_utils_decimal.Inverse(shopspring_decimal.NewFromInt(2).Mul(shopspring_decimal.NewFromInt(int64(mt.MaxLeverage))))
	return MaintenanceMarginRate
}

// GetInitialMarginRate calculates the initial margin rate from the maintenance margin rate
// Formula: initial_margin_rate = 2 * maintenance_margin_rate
func (mt *MaintenanceMarginTier) GetInitialMarginRate() shopspring_decimal.Decimal {
	return mt.GetMaintenanceMarginRate().Mul(shopspring_decimal.NewFromInt(2))
}

// SLPExposureLimit represents per-market SLP exposure limits (lots, notional, percent of account).
type SLPExposureLimit struct {
	ID                  uint64                     `gorm:"primaryKey;autoIncrement" json:"id"`
	MarketID            uint64                     `gorm:"uniqueIndex;not null" json:"market_id"`
	MaxExposureLots     shopspring_decimal.Decimal `gorm:"type:decimal(20,8);not null" json:"max_exposure_lots"`
	MaxExposureNotional shopspring_decimal.Decimal `gorm:"type:decimal(20,2);not null" json:"max_exposure_notional"` // USD notional
	MaxExposurePercent  shopspring_decimal.Decimal `gorm:"type:decimal(6,2);not null" json:"max_exposure_percent"`   // e.g. 5 = 500%
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`

	// Relationship
	Market *Market `gorm:"foreignKey:MarketID;constraint:OnDelete:CASCADE" json:"market,omitempty"`
}

// TableName sets the table name for the SLPExposureLimit model
func (SLPExposureLimit) TableName() string {
	return "slp_exposure_limits"
}
