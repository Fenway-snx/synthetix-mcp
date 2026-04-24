package types

type CollateralConfigTierResponse struct {
	ID            int64  `json:"id"`
	Haircut       string `json:"haircut"`
	MaxAmount     string `json:"maxAmount"`
	MinAmount     string `json:"minAmount"`
	Name          string `json:"name"`
	ValueAddition string `json:"valueAddition"`
	ValueRatio    string `json:"valueRatio"`
}

type CollateralConfigResponse struct {
	Collateral  string                         `json:"collateral"`
	DepositCap  string                         `json:"depositCap"`
	LLTV        string                         `json:"lltv"`
	LTV         string                         `json:"ltv"`
	Market      string                         `json:"market"`
	Tiers       []CollateralConfigTierResponse `json:"tiers"`
	WithdrawFee string                         `json:"withdrawFee"`
}
