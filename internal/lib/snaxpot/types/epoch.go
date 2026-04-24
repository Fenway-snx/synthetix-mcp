package types

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

// EpochState represents the lifecycle state of a Snaxpot epoch.
type EpochState string

const (
	EpochStatePending  EpochState = "pending"
	EpochStateOpen     EpochState = "open"
	EpochStateClosed   EpochState = "closed"
	EpochStateDrawing  EpochState = "drawing"
	EpochStateResolved EpochState = "resolved"
)

func (s EpochState) String() string {
	return string(s)
}

// Epoch tracks the lifecycle of a single Snaxpot lottery epoch.
// Mirrors the on-chain epoch with additional off-chain metadata.
type Epoch struct {
	EpochID         int64      `gorm:"primaryKey;column:epoch_id" json:"epoch_id"`
	State           EpochState `gorm:"column:state;not null" json:"state"`
	VRFSeed         *string    `gorm:"column:vrf_seed" json:"vrf_seed"`
	StartTime       *time.Time `gorm:"column:start_time" json:"start_time"`
	CloseTime       *time.Time `gorm:"column:close_time" json:"close_time"`
	MerkleRoot      *string    `gorm:"column:merkle_root" json:"merkle_root"`
	JackpotAmount   *shopspring_decimal.Decimal `gorm:"type:numeric(38,18);column:jackpot_amount" json:"jackpot_amount"`
	WinningBall1    *int16     `gorm:"column:winning_ball_1" json:"winning_ball_1"`
	WinningBall2    *int16     `gorm:"column:winning_ball_2" json:"winning_ball_2"`
	WinningBall3    *int16     `gorm:"column:winning_ball_3" json:"winning_ball_3"`
	WinningBall4    *int16     `gorm:"column:winning_ball_4" json:"winning_ball_4"`
	WinningBall5    *int16     `gorm:"column:winning_ball_5" json:"winning_ball_5"`
	WinningSnaxBall *int16     `gorm:"column:winning_snax_ball" json:"winning_snax_ball"`
	TotalTickets    *int64     `gorm:"column:total_tickets" json:"total_tickets"`
	TotalTraders    *int64     `gorm:"column:total_traders" json:"total_traders"`
	VRFRequestID    *string    `gorm:"column:vrf_request_id" json:"vrf_request_id"`
	CreatedAt       *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Epoch) TableName() string {
	return "snaxpot_epochs"
}
