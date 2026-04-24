package core

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_FeeRate(t *testing.T) {
	t.Run("simple fee rate tests", func(t *testing.T) {
		tests := []struct {
			name          string
			initialTaker  shopspring_decimal.Decimal
			initialMaker  shopspring_decimal.Decimal
			updateTaker   shopspring_decimal.Decimal
			updateMaker   shopspring_decimal.Decimal
			expectedTaker shopspring_decimal.Decimal
			expectedMaker shopspring_decimal.Decimal
		}{
			{
				name:          "update fees from zero",
				initialTaker:  shopspring_decimal.Zero,
				initialMaker:  shopspring_decimal.Zero,
				updateTaker:   shopspring_decimal.NewFromFloat(0.001),
				updateMaker:   shopspring_decimal.NewFromFloat(0.0005),
				expectedTaker: shopspring_decimal.NewFromFloat(0.001),
				expectedMaker: shopspring_decimal.NewFromFloat(0.0005),
			},
			{
				name:          "update fees from non-zero values",
				initialTaker:  shopspring_decimal.NewFromFloat(0.002),
				initialMaker:  shopspring_decimal.NewFromFloat(0.001),
				updateTaker:   shopspring_decimal.NewFromFloat(0.0015),
				updateMaker:   shopspring_decimal.NewFromFloat(0.0008),
				expectedTaker: shopspring_decimal.NewFromFloat(0.0015),
				expectedMaker: shopspring_decimal.NewFromFloat(0.0008),
			},
			{
				name:          "update fees to zero",
				initialTaker:  shopspring_decimal.NewFromFloat(0.001),
				initialMaker:  shopspring_decimal.NewFromFloat(0.0005),
				updateTaker:   shopspring_decimal.Zero,
				updateMaker:   shopspring_decimal.Zero,
				expectedTaker: shopspring_decimal.Zero,
				expectedMaker: shopspring_decimal.Zero,
			},
			{
				name:          "update with negative fees",
				initialTaker:  shopspring_decimal.NewFromFloat(0.001),
				initialMaker:  shopspring_decimal.NewFromFloat(0.0005),
				updateTaker:   shopspring_decimal.NewFromFloat(-0.0001),
				updateMaker:   shopspring_decimal.NewFromFloat(-0.0002),
				expectedTaker: shopspring_decimal.NewFromFloat(-0.0001),
				expectedMaker: shopspring_decimal.NewFromFloat(-0.0002),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {

				feeRate := NewFeeRate(tc.initialTaker, tc.initialMaker)
				assert.NotNil(t, feeRate, "could not start fee rate")

				prev, updated := feeRate.SetFeeRates(tc.updateTaker, tc.updateMaker)
				assert.True(t, updated)
				assert.True(t, prev.GetMakerFeeRate().Equal(tc.initialMaker))
				assert.True(t, prev.GetTakerFeeRate().Equal(tc.initialTaker))

				newMakerFeeRate := feeRate.GetMakerFeeRate()
				newTakerFeeRate := feeRate.GetTakerFeeRate()

				assert.True(t, tc.expectedTaker.Equal(newTakerFeeRate),
					"Taker Fee mismatch: expected %s, got '%s'",
					tc.expectedTaker.String(), newTakerFeeRate.String())

				assert.True(t, tc.expectedMaker.Equal(newMakerFeeRate),
					"Maker Fee mismatch: expected %s, got '%s'",
					tc.expectedMaker.String(), newMakerFeeRate.String())
			})
		}
	})
}
