package auth

import (
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	ActionModifyOrder = "modify_order"
	ActionCancelOrder = "cancel_order"
	ActionPlaceOrders = "place_orders"
)

// GetModifyOrderTypes returns EIP-712 types for modify order
func GetModifyOrderTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"ModifyOrder": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "orderId", Type: "uint256"},
			{Name: "price", Type: "string"},        // Optional - empty string if not modified
			{Name: "quantity", Type: "string"},     // Optional - empty string if not modified
			{Name: "triggerPrice", Type: "string"}, // Optional - empty string if not modified
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetModifyOrderByCloidTypes returns EIP-712 types for modify order by cloid
func GetModifyOrderByCloidTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"ModifyOrderByCloid": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "clientOrderId", Type: "string"},
			{Name: "price", Type: "string"},
			{Name: "quantity", Type: "string"},
			{Name: "triggerPrice", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// Returns EIP-712 types for cancel all orders
func GetCancelAllOrdersTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"CancelAllOrders": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "symbols", Type: "string[]"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetCancelOrderTypes returns EIP-712 types for cancel order
func GetCancelOrderTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"CancelOrders": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "orderIds", Type: "uint256[]"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetCancelOrdersByCloidTypes returns EIP-712 types for cancel order by cloid
func GetCancelOrdersByCloidTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"CancelOrdersByCloid": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "clientOrderIds", Type: "string[]"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetPlaceOrdersTypes returns EIP-712 types for place orders
func GetPlaceOrdersTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"Order": {
			{Name: "symbol", Type: "string"},
			{Name: "side", Type: "string"},
			{Name: "orderType", Type: "string"},
			{Name: "price", Type: "string"},
			{Name: "triggerPrice", Type: "string"},
			{Name: "quantity", Type: "string"},
			{Name: "reduceOnly", Type: "bool"},
			{Name: "isTriggerMarket", Type: "bool"},
			{Name: "clientOrderId", Type: "string"},
			{Name: "closePosition", Type: "bool"},
		},
		"PlaceOrders": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "orders", Type: "Order[]"},
			{Name: "grouping", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetScheduleCancelTypes returns EIP-712 types for schedule cancel.
func GetScheduleCancelTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"ScheduleCancel": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "timeoutSeconds", Type: "uint256"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetWithdrawCollateralTypes returns EIP-712 types for withdraw collateral
func GetWithdrawCollateralTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"WithdrawCollateral": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "symbol", Type: "string"},
			{Name: "amount", Type: "string"},
			{Name: "destination", Type: "address"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetAddDelegatedSignerTypes returns EIP-712 types for add delegated signer
func GetAddDelegatedSignerTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"AddDelegatedSigner": {
			{Name: "delegateAddress", Type: "address"},
			{Name: "subAccountId", Type: "uint256"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
			{Name: "expiresAt", Type: "uint256"},
			{Name: "permissions", Type: "string[]"},
		},
	}
}

// GetRemoveDelegatedSignerTypes returns EIP-712 types for remove delegated signer
func GetRemoveDelegatedSignerTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"RemoveDelegatedSigner": {
			{Name: "delegateAddress", Type: "address"},
			{Name: "subAccountId", Type: "uint256"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetRemoveAllDelegatedSignersTypes returns EIP-712 types for remove all delegated signers
func GetRemoveAllDelegatedSignersTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"RemoveAllDelegatedSigners": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetUpdateLeverageTypes returns EIP-712 types for update leverage
func GetUpdateLeverageTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"UpdateLeverage": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "symbol", Type: "string"},
			{Name: "leverage", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetCreateSubaccountTypes returns EIP-712 types for create subaccount
func GetCreateSubaccountTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"CreateSubaccount": {
			{Name: "masterSubAccountId", Type: "uint256"},
			{Name: "name", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetTransferCollateralTypes returns EIP-712 types for transfer collateral
func GetTransferCollateralTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"TransferCollateral": {
			{Name: "amount", Type: "string"},
			{Name: "expiresAfter", Type: "uint256"},
			{Name: "nonce", Type: "uint256"},
			{Name: "subAccountId", Type: "uint256"},
			{Name: "symbol", Type: "string"},
			{Name: "to", Type: "uint256"},
		},
	}
}

// GetVoluntaryAutoExchangeTypes returns EIP-712 types for voluntary auto-exchange
func GetVoluntaryAutoExchangeTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"VoluntaryAutoExchange": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "sourceAsset", Type: "string"},
			{Name: "targetUSDTAmount", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetClearSnaxpotPreferenceTypes returns EIP-712 types for clear Snaxpot
// preference.
func GetClearSnaxpotPreferenceTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"ClearSnaxpotPreference": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "scope", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetSaveSnaxpotTicketsTypes returns EIP-712 types for save Snaxpot tickets.
// Ticket entries are encoded as a nested SnaxpotTicket struct array.
func GetSaveSnaxpotTicketsTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"SnaxpotTicket": {
			{Name: "ball1", Type: "uint256"},
			{Name: "ball2", Type: "uint256"},
			{Name: "ball3", Type: "uint256"},
			{Name: "ball4", Type: "uint256"},
			{Name: "ball5", Type: "uint256"},
			{Name: "snaxBall", Type: "uint256"},
			{Name: "ticketSerial", Type: "uint256"},
		},
		"SaveSnaxpotTickets": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "entries", Type: "SnaxpotTicket[]"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetSetSnaxpotPreferenceTypes returns EIP-712 types for set Snaxpot
// preference.
func GetSetSnaxpotPreferenceTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"SetSnaxpotPreference": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "snaxBall", Type: "uint256"},
			{Name: "scope", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}

// GetUpdateSubAccountNameTypes returns EIP-712 types for update subaccount name
func GetUpdateSubAccountNameTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": getEIP712DomainFields(),
		"UpdateSubAccountName": {
			{Name: "subAccountId", Type: "uint256"},
			{Name: "name", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "expiresAfter", Type: "uint256"},
		},
	}
}
