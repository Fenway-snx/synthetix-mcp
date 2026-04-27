package types

import (
	shopspring_decimal "github.com/shopspring/decimal"
)

// A price at the API level, whose wire representation is as a string
// containing a recognisably numeric value.
type Price = string

const (
	Price_None Price = ""
)

// ===========================
// `Price`
// ===========================

// TODO: PriceFromDecimal(v) (Price, err)

func PriceFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) Price {
	return Price(stringFromDecimalOrBlankWhenZero(v))
}

// Converts a price from a string obtained from a trusted source, without
// any validation.
func PriceFromDecimalUnvalidated(
	v shopspring_decimal.Decimal,
) Price {
	return Price(stringFromDecimalUnvalidated(v))
}

// Converts a price from a string obtained from a trusted source, without
// any validation.
func PriceFromStringUnvalidated(
	s string,
) Price {
	return Price(s)
}

// Converts a price pointer from a string obtained from a trusted source,
// without any validation.
func PricePtrFromStringUnvalidated(
	s string,
) *Price {
	p := Price(s)

	return &p
}

func PricePtrToStringPtr(
	p *Price,
) *string {
	if p == nil {
		return nil
	} else {
		s := string(*p)

		return &s
	}
}
