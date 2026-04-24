package types

import (
	shopspring_decimal "github.com/shopspring/decimal"
)

type MaintenanceMarginTier struct {
	MinPositionSize              string                     `json:"minPositionSize"`              // Minimum position size (notional value in USD)
	MaxPositionSize              string                     `json:"maxPositionSize"`              // Maximum position size (notional value in USD), empty string for unlimited
	MaxLeverage                  uint32                     `json:"maxLeverage"`                  // Maximum leverage for this tier
	InitialMarginRequirement     shopspring_decimal.Decimal `json:"initialMarginRequirement"`     // e.g., 5%
	MaintenanceMarginRequirement shopspring_decimal.Decimal `json:"maintenanceMarginRequirement"` // Maintenance margin requirement
	MaintenanceDeductionValue    shopspring_decimal.Decimal `json:"maintenanceDeductionValue"`    // Deduction value for margin calculation
}

// Represents a trading market configuration which can be returned from the API.
type MarketResponse struct {
	Symbol      Symbol `json:"symbol"`      // e.g., "SOLUSD"
	Description string `json:"description"` // e.g., "Solana"
	BaseAsset   Asset  `json:"baseAsset"`   // Symbol like "SOL"
	QuoteAsset  Asset  `json:"quoteAsset"`  // Symbol like "USD"

	// Market Config fields
	IsOpen                     bool                       `json:"isOpen"`                     // Tradeable?
	IsCloseOnly                bool                       `json:"isCloseOnly"`                // In close only mode?
	PriceExponent              int64                      `json:"priceExponent"`              // Number of decimal places for price
	QuantityExponent           int64                      `json:"quantityExponent"`           // Number of decimal places for quantity
	PriceIncrement             shopspring_decimal.Decimal `json:"priceIncrement"`             // What is the price tick?
	MinOrderSize               shopspring_decimal.Decimal `json:"minOrderSize"`               // Minimum order size
	OrderSizeIncrement         shopspring_decimal.Decimal `json:"orderSizeIncrement"`         // Order size increment
	ContractSize               uint32                     `json:"contractSize"`               // Contract size for the market
	MaxMarketOrderSize         shopspring_decimal.Decimal `json:"maxMarketOrderSize"`         // Maximum market order size
	MaxLimitOrderSize          shopspring_decimal.Decimal `json:"maxLimitOrderSize"`          // Maximum limit order size
	MinOrderPrice              shopspring_decimal.Decimal `json:"minOrderPrice"`              // Minimum order price
	LimitOrderPriceCapRatio    shopspring_decimal.Decimal `json:"limitOrderPriceCapRatio"`    // Limit order price cap ratio
	LimitOrderPriceFloorRatio  shopspring_decimal.Decimal `json:"limitOrderPriceFloorRatio"`  // Limit order price floor ratio
	MarketOrderPriceCapRatio   shopspring_decimal.Decimal `json:"marketOrderPriceCapRatio"`   // Market order price cap ratio
	MarketOrderPriceFloorRatio shopspring_decimal.Decimal `json:"marketOrderPriceFloorRatio"` // Market order price floor ratio
	LiquidationClearanceFee    shopspring_decimal.Decimal `json:"liquidationClearanceFee"`    // Liquidation clearance fee
	MinNotionalValue           shopspring_decimal.Decimal `json:"minNotionalValue"`           // Minimum notional value
	MaintenanceMarginTiers     []MaintenanceMarginTier    `json:"maintenanceMarginTiers"`     // Maintenance margin tiers for this market
}
