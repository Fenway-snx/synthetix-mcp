package decimal

import shopspring_decimal "github.com/shopspring/decimal"

var (
	Decimal_0_1    = shopspring_decimal.New(1, -1) // 0.1
	Decimal_1      = _decimalFromInt64(1)
	Decimal_2      = _decimalFromInt64(2)
	Decimal_5      = _decimalFromInt64(5)
	Decimal_8      = _decimalFromInt64(8)
	Decimal_10     = _decimalFromInt64(10)
	Decimal_10_000 = _decimalFromInt64(10000)
	Decimal_50_000 = _decimalFromInt64(50000)
)

func _decimalFromInt64(v int64) shopspring_decimal.Decimal {
	return shopspring_decimal.NewFromInt(v)
}
