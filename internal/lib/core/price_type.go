package core

// PriceType represents the type of price feed
type PriceType string

// Price feed type constants
const (
	// Represents aggregate mark price feed (weighted median from multiple futures exchanges).
	PriceType_aggregate_mark PriceType = "aggregate_mark"

	// Represents aggregate index price feed (weighted median from multiple spot exchanges).
	PriceType_aggregate_index PriceType = "aggregate_index"

	// Represents collateral price feed.
	PriceType_collateral PriceType = "collateral"

	// Represents reference/index price feed (Coinmetrics RTRR - Real Time Reference Rate).
	PriceType_index PriceType = "index"

	// Represents last traded price feed (from market data).
	PriceType_last PriceType = "last"

	// Represents mark price feed (calculated from mid price + spread EMA).
	PriceType_mark PriceType = "mark"

	// Represents mid price feed (average of best bid and ask).
	PriceType_mid PriceType = "mid"

	// Represents perpetual futures price from external exchange.
	PriceType_perp PriceType = "perp"
)

// PriceTypes represents all price types
//
// NOTE: This is precisely the worst place in the known universe to define this! 🤦‍♂️ (SNX-5504)
var PriceTypes = []PriceType{
	PriceType_mark,
}

// IsValid checks if the price type is valid
func (pt PriceType) IsValid() bool {
	switch pt {
	case
		PriceType_aggregate_mark,
		PriceType_aggregate_index,
		PriceType_collateral,
		PriceType_index,
		PriceType_last,
		PriceType_mark,
		PriceType_mid,
		PriceType_perp:

		return true
	default:

		return false
	}
}
