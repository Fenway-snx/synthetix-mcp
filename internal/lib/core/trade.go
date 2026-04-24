package core

// FillType constants define the type of order execution
const (
	FillTypeMaker = "Maker"
	FillTypeTaker = "Taker"
)

// IsMakerFillType checks if a fill type represents a maker order
func IsMakerFillType(fillType string) bool {
	return fillType == FillTypeMaker || fillType == "MAKER" || fillType == "maker"
}

// IsTakerFillType checks if a fill type represents a taker order
func IsTakerFillType(fillType string) bool {
	return fillType == FillTypeTaker || fillType == "TAKER" || fillType == "taker"
}
