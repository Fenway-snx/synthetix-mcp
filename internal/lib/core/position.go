package core

type PositionSide int32

const (
	PositionSideShort PositionSide = iota // Sell position
	PositionSideLong                      // Buy position
)

func (s PositionSide) Int32() int32 {
	return int32(s)
}
