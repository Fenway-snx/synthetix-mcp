package core

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

const ADLBucketDefault int64 = 1

// Represents a trading position. Timestamp fields may alias the same
// *time.Time: you may reassign a field (including nil) to refer to a
// different instant; do not mutate the pointed-to time through a field
// (e.g. *order.CreatedAt).
type Position struct {
	ID                       uint64
	SubAccountId             SubAccountId
	EntryPrice               shopspring_decimal.Decimal // Entry price
	Quantity                 shopspring_decimal.Decimal // Size
	UPNL                     shopspring_decimal.Decimal // Unrealized PnL for this position
	IMR                      shopspring_decimal.Decimal // Initial margin required when position was opened
	MMR                      shopspring_decimal.Decimal // Maintenance margin required to avoid liquidation
	MarginReserved           shopspring_decimal.Decimal // Actual margin reserved for this position
	Side                     PositionSide
	LiquidationPrice         shopspring_decimal.Decimal
	TakeProfitOrderIds       []OrderId                  // Take profit order IDs
	StopLossOrderIds         []OrderId                  // Stop loss order IDs
	NetFunding               shopspring_decimal.Decimal // Accumulated funding payments over position lifetime
	AccumulatedRealizedPnl   shopspring_decimal.Decimal // Accumulated realized PnL from partial closes
	AccumulatedFees          shopspring_decimal.Decimal // Accumulated trading fees over position lifetime (open, increase, reduce, close)
	AccumulatedCloseValue    shopspring_decimal.Decimal // Sum of (fillPrice * fillQty) across reducing fills for weighted avg close price
	AccumulatedCloseQuantity shopspring_decimal.Decimal // Sum of fillQty across reducing fills for weighted avg close price
	ADLBucket                int64                      // ADL priority bucket (1–5, 5 = highest risk)
	CreatedAt                *time.Time
	ClosedAt                 *time.Time
	ModifiedAt               *time.Time
}

type PositionSide int32

const (
	PositionSideShort PositionSide = iota // Sell position
	PositionSideLong                      // Buy position
)

func (s PositionSide) Int32() int32 {
	return int32(s)
}
