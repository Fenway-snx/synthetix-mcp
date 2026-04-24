package core

// OrderbookJournalOperation constants represent journal operation types for orderbook changes
const (
	// OrderbookJournalOpAdd represents an order being added to the orderbook
	OrderbookJournalOpAdd = "orderbook_add"

	// OrderbookJournalOpModify represents an order being modified in the orderbook
	OrderbookJournalOpModify = "orderbook_modify"

	// OrderbookJournalOpRecoveryEnd represents the end of orderbook recovery
	OrderbookJournalOpRecoveryEnd = "orderbook_recovery_end"

	// OrderbookJournalOpRecoveryStart represents the start of orderbook recovery
	OrderbookJournalOpRecoveryStart = "orderbook_recovery_start"

	// OrderbookJournalOpRemove represents an order being removed from the orderbook
	OrderbookJournalOpRemove = "orderbook_remove"
)

// Pagination constants for orderbook recovery
const (
	// OrderbookRecoveryPageSize is the default page size for paginated orderbook recovery requests
	OrderbookRecoveryPageSize = 1000
)
