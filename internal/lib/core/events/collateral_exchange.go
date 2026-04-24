package events

import (
	"time"

	shopspring_decimal "github.com/shopspring/decimal"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

type CollateralExchangeEvent struct {
	Type             snx_lib_core.CollateralExchangeType `json:"type"`
	FromSubAccountId snx_lib_core.SubAccountId           `json:"from_sub_account_id"`
	ToSubAccountId   snx_lib_core.SubAccountId           `json:"to_sub_account_id"`
	FromCollateral   string                              `json:"from_collateral"`
	ToCollateral     string                              `json:"to_collateral"`
	FromQuantity     shopspring_decimal.Decimal          `json:"from_quantity"`
	ToQuantity       shopspring_decimal.Decimal          `json:"to_quantity"`
	Fee              shopspring_decimal.Decimal          `json:"fee"`
	ExchangedAt      time.Time                           `json:"exchanged_at"`
}
