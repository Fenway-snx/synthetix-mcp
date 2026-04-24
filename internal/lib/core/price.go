package core

import (
	"errors"

	shopspring_decimal "github.com/shopspring/decimal"
)

var (
	errInvalidPrice = errors.New("invalid price")
)

// Price type
//
// Note:
// It is intended to make this a strong type in the future.
type Price = shopspring_decimal.Decimal

var (
	Price_Zero = Price(shopspring_decimal.Zero)
)

func _PriceFromInt64(v int64) (Price, bool) {

	// TODO: validate outlandish prices

	return shopspring_decimal.NewFromInt(v), true
}

func PriceFromInt(v int64) (price Price, err error) {

	price, isValid := _PriceFromInt64(v)

	if !isValid {
		err = errInvalidPrice
	}
	return
}

func PriceFromIntUnvalidated(v int64) (price Price) {

	price, _ = _PriceFromInt64(v)

	return
}
