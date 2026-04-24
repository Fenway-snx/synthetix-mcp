package nats

// CowOrderRequestEvent represents a CoW Protocol order submission request sent over NATS
// to be processed by the Relayer service.
type CowOrderRequestEvent struct {
	// Required fields
	Signature  string `json:"signature"` // Optional: EOA signature over Cow order digest (if omitted, relayer signs using configured trader key)
	SellToken  string `json:"sell_token" validate:"required"`
	BuyToken   string `json:"buy_token" validate:"required"`
	SellAmount string `json:"sell_amount" validate:"required"`
	BuyAmount  string `json:"buy_amount" validate:"required"`

	// Optional behaviour params (envelope is always composed by the relayer)
	ValidTo           int64  `json:"valid_to"`           // Defaults to now+300s if 0; must fit uint32
	AppData           string `json:"app_data"`           // Defaults 0x00..00 (bytes32)
	Kind              string `json:"kind"`               // Defaults "sell"
	PartiallyFillable bool   `json:"partially_fillable"` // Defaults false
	SellTokenBalance  string `json:"sell_token_balance"` // Defaults "erc20"
	BuyTokenBalance   string `json:"buy_token_balance"`  // Defaults "erc20"
	ApiURL            string `json:"api_url"`            // Optional override of CoW orders API URL

	// Metadata
	Timestamp uint64 `json:"timestamp"` // Producer timestamp (optional)
}
