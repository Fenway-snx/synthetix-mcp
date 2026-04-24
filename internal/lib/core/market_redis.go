package core

// Redis key constants for market data synchronization
const (
	// MarketActiveKey is the Redis key for the list of active markets
	MarketActiveKey = "snx:market:active"

	// MarketDataKeyPattern is the pattern for individual market data keys
	// Usage: fmt.Sprintf(MarketDataKeyPattern, symbol)
	MarketDataKeyPattern = "snx:market:data:%s"
)
