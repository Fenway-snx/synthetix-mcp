package core

import (
	shopspring_decimal "github.com/shopspring/decimal"
)

// TODO: make this strong typed
type FeeRateAmount = shopspring_decimal.Decimal

/*
NOTE: I don't need to make this thread safe as it will be
used by the sub account actor which should always be single threaded
*/

type FeeRate struct {
	takerFeeRate shopspring_decimal.Decimal
	makerFeeRate shopspring_decimal.Decimal
}

func NewFeeRate(takerFeeRate, makerFeeRate shopspring_decimal.Decimal) *FeeRate {
	return &FeeRate{
		takerFeeRate: takerFeeRate,
		makerFeeRate: makerFeeRate,
	}
}

func (f FeeRate) GetTakerFeeRate() shopspring_decimal.Decimal {
	return f.takerFeeRate
}

func (f FeeRate) GetMakerFeeRate() shopspring_decimal.Decimal {
	return f.makerFeeRate
}

func (f *FeeRate) SetFeeRates(takerFeeRate, makerFeeRate shopspring_decimal.Decimal) (previousRates FeeRate, updated bool) {

	previousRates.takerFeeRate = f.takerFeeRate
	previousRates.makerFeeRate = f.makerFeeRate

	if !f.takerFeeRate.Equal(takerFeeRate) {
		updated = true
		f.takerFeeRate = takerFeeRate
	}

	if !f.makerFeeRate.Equal(makerFeeRate) {
		updated = true
		f.makerFeeRate = makerFeeRate
	}

	return previousRates, updated
}
