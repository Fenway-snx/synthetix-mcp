package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
)

var (
	errValueDoesNotContainOrderIdArray = errors.New("value does not contain an order id array")

	errfmtInvalidOrderId = `invalid order id {void=%d, cloid="%s"}`
)

// =========================================================================
// Supporting types
// =========================================================================

// ===========================
// `OrderId`
// ===========================

// NOTE: need to define in this package to avoid import cycles

type OrderId struct {
	VenueId  uint64 `json:"void"`
	ClientId string `json:"cloid,omitempty"`
}

// ===========================
// `OrderIdArray`
// ===========================

// Custom type for PostgreSQL array handling of `[]uint64`.
type OrderIdArray []OrderId

// Implements the sql.Scanner interface for database reads.
//
// Handles three valid JSON shapes stored in the JSONB column:
//   - array:  [{"void":1001}, {"void":1002}]  (canonical)
//   - object: {"void":1001}                    (legacy/single-entry; wrapped into an array)
//   - empty:  [], null, ""                     (treated as empty)
func (a *OrderIdArray) Scan(value any) error {
	*a = []OrderId{}

	if value == nil {
		return nil
	}

	raw, ok := value.([]byte)
	if !ok {
		s, ok := value.(string)
		if ok {
			s = strings.TrimSpace(s)
			if s == "" || s == "[]" {
				return nil
			}
			raw = []byte(s)
		} else {
			return errValueDoesNotContainOrderIdArray
		}
	}

	if len(raw) == 0 {
		return nil
	}

	// Detect whether the JSONB value is an array, a single object, or a
	// JSON null literal and handle each so legacy rows don't break reads.
	switch raw[0] {
	case '[':
		if err := json.Unmarshal(raw, a); err != nil {
			return err
		}
	case '{':
		var single OrderId
		if err := json.Unmarshal(raw, &single); err != nil {
			return err
		}
		*a = []OrderId{single}
	case 'n':
		if string(raw) == "null" {
			return nil
		}
		return errValueDoesNotContainOrderIdArray
	default:
		return errValueDoesNotContainOrderIdArray
	}

	for _, oid := range *a {
		if oid.VenueId == 0 {
			return fmt.Errorf(errfmtInvalidOrderId, oid.VenueId, oid.ClientId)
		}
	}

	return nil
}

// Implements the driver.Valuer interface for database writes
func (a OrderIdArray) Value() (driver.Value, error) {
	if len(a) == 0 {

		return "[]", nil
	} else {

		return json.Marshal(a)
	}
}

// =========================================================================
// Database types
// =========================================================================

// Position represents a trading position in the database
type Position struct {
	ID                       uint64                     `gorm:"primaryKey;column:id" json:"id"`
	SubAccountID             int64                      `gorm:"column:sub_account_id;index:idx_subaccount_id;uniqueIndex:idx_subaccount_symbol" json:"sub_account_id"`
	Symbol                   string                     `gorm:"column:symbol;index:idx_symbol;uniqueIndex:idx_subaccount_symbol" json:"symbol"`
	Side                     int32                      `gorm:"column:side" json:"side"`                                                                       // 0: Short, 1: Long (matching PositionSide enum)
	EntryPrice               shopspring_decimal.Decimal `gorm:"type:varchar(255);column:entry_price" json:"entry_price"`                                       // Entry price as string
	Quantity                 shopspring_decimal.Decimal `gorm:"type:varchar(255);column:quantity" json:"quantity"`                                             // Realized PnL as string
	UPNL                     shopspring_decimal.Decimal `gorm:"type:varchar(255);column:upnl;default:'0'" json:"upnl"`                                         // Unrealized PnL as string
	UsedMargin               shopspring_decimal.Decimal `gorm:"type:varchar(255);column:used_margin;default:'0'" json:"used_margin"`                           // Margin in use as string
	MaintenanceMargin        shopspring_decimal.Decimal `gorm:"type:varchar(255);column:maintenance_margin;default:'0'" json:"maintenance_margin"`             // Min margin required as string
	LiquidationPrice         shopspring_decimal.Decimal `gorm:"type:varchar(255);column:liquidation_price;default:'0'" json:"liquidation_price"`               // Liquidation trigger price as string
	NetPositionFundingPnl    shopspring_decimal.Decimal `gorm:"type:varchar(255);column:net_position_funding_pnl;default:'0'" json:"net_position_funding_pnl"` // Net funding PnL for this position as string
	AccumulatedRealizedPnl   shopspring_decimal.Decimal `gorm:"type:varchar(255);column:accumulated_realized_pnl;default:'0'" json:"accumulated_realized_pnl"`
	AccumulatedFees          shopspring_decimal.Decimal `gorm:"type:varchar(255);column:accumulated_fees;default:'0'" json:"accumulated_fees"`
	AccumulatedCloseValue    shopspring_decimal.Decimal `gorm:"type:varchar(255);column:accumulated_close_value;default:'0'" json:"accumulated_close_value"`
	AccumulatedCloseQuantity shopspring_decimal.Decimal `gorm:"type:varchar(255);column:accumulated_close_quantity;default:'0'" json:"accumulated_close_quantity"`
	Action                   int32                      `gorm:"column:action" json:"action"`                   // 0: Open, 1: Close, 2: Update (matching PositionEventAction enum)
	ADLBucket                int64                      `gorm:"column:adl_bucket;default:1" json:"adl_bucket"` // ADL priority bucket (1–5, 5 = highest risk)
	TakeProfitOrders         OrderIdArray               `gorm:"column:take_profit_orders;type:jsonb" json:"take_profit_orders"`
	StopLossOrders           OrderIdArray               `gorm:"column:stop_loss_orders;type:jsonb" json:"stop_loss_orders"`
	ClosedAt                 *time.Time                 `gorm:"column:closed_at" json:"closed_at"` // TODO: this will be set but the row will get deleted every time we close a position
	CreatedAt                *time.Time                 `gorm:"column:created_at;autoCreateTime:false" json:"created_at"`
	ModifiedAt               *time.Time                 `gorm:"column:modified_at" json:"modified_at"`
	UpdatedAt                time.Time                  `gorm:"column:updated_at;index;autoUpdateTime:false" json:"updated_at"`
}

// TableName returns the table name for GORM
func (Position) TableName() string {
	return "positions"
}
