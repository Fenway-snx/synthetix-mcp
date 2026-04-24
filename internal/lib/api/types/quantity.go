package types

import (
	shopspring_decimal "github.com/shopspring/decimal"
)

// A quantity at the API level, whose wire representation is as a non-empty
// string containing a recognisably numeric value.
type Quantity string

const (
	Quantity_None Quantity = ""
)

// ===========================
// `Quantity`
// ===========================

// TODO: QuantityFromDecimal(v) (Quantity, err)

func QuantityFromDecimalOrBlankWhenZero(
	v shopspring_decimal.Decimal,
) Quantity {
	return Quantity(stringFromDecimalOrBlankWhenZero(v))
}

// Converts a quantity from a string obtained from a trusted source, without
// any validation.
func QuantityFromStringUnvalidated(
	s string,
) Quantity {
	return Quantity(s)
}

// Converts a quantity from a string obtained from a trusted source, without
// any validation.
func QuantityPtrFromStringUnvalidated(
	s string,
) *Quantity {
	q := Quantity(s)

	return &q
}

// Converts a quantity pointer from a string obtained from a trusted source,
// without any validation.
func QuantityPtrFromStringPtrUnvalidated(
	p *string,
) *Quantity {
	if p == nil {
		return nil
	} else {
		q := Quantity(*p)

		return &q
	}
}

func QuantityPtrToStringPtr(
	p *Quantity,
) *string {
	if p == nil {
		return nil
	} else {
		s := string(*p)

		return &s
	}
}
