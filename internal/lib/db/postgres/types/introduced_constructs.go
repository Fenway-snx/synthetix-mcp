// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package types

import (
	shopspring_decimal "github.com/shopspring/decimal"
)

type Price = shopspring_decimal.Decimal
