package auth

type TradeSignature struct {
	R string `json:"r" validate:"required"`
	S string `json:"s" validate:"required"`
	V uint8  `json:"v" validate:"required"`
}
