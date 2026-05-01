package core

import (
	"database/sql/driver"
	"fmt"
)

var (
	SubAccountId_Zero = SubAccountId(0)
)

// Internal (Trading Core) representation of a sub-account identifier.
type SubAccountId int64

// Okay I am just going to keep this quick and simple for now but
// we will need to add some guard rails around it
const (
	SubAccountId_LCF            SubAccountId = 102 // Liquidation Clerance Fees
	SubAccountId_SlpPlaceHolder SubAccountId = 104 // A place holder for slp takeovers
	SubAccountId_TF             SubAccountId = 101 // Trading Fees
	SubAccountId_WF             SubAccountId = 103 // Withdrawal Fees
)

// Implements the sql.Scanner interface for database operations.
func (s *SubAccountId) Scan(value any) error {
	if value == nil {
		*s = SubAccountId_Zero
		return nil
	}
	switch v := value.(type) {
	case int64:
		*s = SubAccountId(v)
	case int32:
		*s = SubAccountId(v)
	case int:
		*s = SubAccountId(v)
	default:
		return fmt.Errorf("cannot scan %T into SubAccountId", value)
	}
	return nil
}

// Implements the driver.Valuer interface for database operations.
func (s SubAccountId) Value() (driver.Value, error) {
	return int64(s), nil
}
