package core

const (
	CollateralQuantityPrecision = 8
	CollateralValuePrecision    = 2
	MarginPrecision             = 2
)

// Side string constants
const (
	SideBuy  = "BUY"
	SideSell = "SELL"
)

// TODO: make it a strong type called TransactionItem
// Sub account transaction types
const (
	TransactionActionDeposit                 = "DEPOSIT"
	TransactionActionWithdrawal              = "WITHDRAWAL"
	TransactionActionTransfer                = "TRANSFER"
	TransactionActionFundingPayment          = "FUNDING_PAYMENT"
	TransactionActionSLPTakeoverLiquidated   = "SLP_TAKEOVER_LIQUIDATED"
	TransactionActionSLPTakeoverReceiver     = "SLP_TAKEOVER_RECEIVER"
	TransactionActionDelegationCreate        = "DELEGATION_CREATE"
	TransactionActionDelegationUpdate        = "DELEGATION_UPDATE"
	TransactionActionDelegationDelete        = "DELEGATION_DELETE"
	TransactionActionDelegationDeleteAll     = "DELEGATION_DELETE_ALL"
	TransactionActionLiquidationClearanceFee = "LIQUIDATION_CLEARANCE"
)
