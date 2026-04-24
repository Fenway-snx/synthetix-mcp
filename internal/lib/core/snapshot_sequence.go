package core

// Monotonically increasing sequence number assigned to orderbook operations
// by the matching engine. Used for ordering, gap detection, and snapshot
// identity across matching, marketdata, and websocket services.
type SnapshotSequence uint64
