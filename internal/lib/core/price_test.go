package core

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_Price_Zero(t *testing.T) {

	// "proper" comparison should use `#Equal()`
	{
		expected := shopspring_decimal.Zero
		actual := Price_Zero

		assert.True(t, expected.Equal(actual))
	}

	// "implementation-dependent" comparison using built-in ==
	{
		expected := shopspring_decimal.Zero
		actual := Price_Zero

		assert.Equal(t, expected, actual)
	}
}
