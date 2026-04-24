package core

// Represents the reason a position was closed.
type CloseReason string

const (
	CloseReasonClose       CloseReason = "close"
	CloseReasonFlip        CloseReason = "flip"
	CloseReasonLiquidation CloseReason = "liquidation"
)
