package core

import (
	"fmt"
)

// =========================================================================
// Constants
// =========================================================================

// =========================================================================
// Types
// =========================================================================

type OrderSide int32

const (
	OrderSideShort OrderSide = iota
	OrderSideLong
)

// Distinguishes opening vs closing legs relative to long and short
// exposure. Numeric values are stable for storage and on-the-wire enums.
type Direction int32

const (
	Direction_Short      Direction = iota // Opening or adding to short exposure (sell-side entry in typical books).
	Direction_Long                        // Opening or adding to long exposure (buy-side entry in typical books).
	Direction_CloseShort                  // Closing or reducing an existing short (buy-to-cover in typical books).
	Direction_CloseLong                   // Closing or reducing an existing long (sell-to-flatten in typical books).
)

// represents the execution type of an order
type OrderType int32

const (
	OrderTypeLimit OrderType = iota
	OrderTypeMarket
	OrderTypeStopMarket
	OrderTypeTakeProfitMarket
	OrderTypeStopLimit
	OrderTypeTakeProfitLimit
	OrderTypeTWAP
)

// =========================================================================
// API functions
// =========================================================================

// TODO: reorder the following sections at an appropriate time

// ===========================
// Utility functions
// ===========================

// ===========================
// `Direction`
// ===========================

// Converts known direction values to canonical wire strings.
// Unknown values return a diagnostic string and false.
func DirectionToString(
	v Direction,
) (r string, recognised bool) {

	switch v {
	case Direction_Short:

		return "short", true
	case Direction_Long:

		return "long", true
	case Direction_CloseShort:

		return "closeShort", true
	case Direction_CloseLong:

		return "closeLong", true
	default:

		return fmt.Sprintf("UNKNOWN-Direction<v=%v>", v), false
	}
}
