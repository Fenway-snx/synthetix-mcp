// Package json contains types and validation for the API layer.
// These types are used for JSON marshaling/unmarshaling in REST and WebSocket APIs.
package json

import (
	"errors"
	"fmt"
	"regexp"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

var (
	regexp_rs *regexp.Regexp
)

var (
	errActionTypeMustBeCancelOrder                            = errors.New("action type must be 'cancelOrder'")
	errExpiresAfterMustBeGreaterThanNonce                     = errors.New("expiresAfter must be greater than nonce")
	errNonceRequiredRecommendedCurrentTimestampInMilliseconds = errors.New("nonce is required (recommended: current timestamp in milliseconds)")
	errOrderIdRequired                                        = errors.New("orderId is required")
	errSignatureRAndSFieldsRequired                           = errors.New("signature r and s fields are required")
	errSignatureRFieldMustBe64CharacterHexStringWith0xPrefix  = errors.New("signature r field must be a 64-character hex string with 0x prefix")
	errSignatureSFieldMustBe64CharacterHexStringWith0xPrefix  = errors.New("signature s field must be a 64-character hex string with 0x prefix")
	errSignatureVFieldMustBe0Or1Or27Or28                      = errors.New("signature v field must be 0, 1, 27, or 28")
)

func init() {
	regexp_rs = regexp.MustCompile(`^0x[a-fA-F0-9]{64}$`)
}

// Validates request signature components.
func validateSignature(r, s string, v int) error {
	// Validate signature structure
	if r == "" || s == "" {
		return errSignatureRAndSFieldsRequired
	}

	// Validate signature field format (should be hex strings)
	if !regexp_rs.MatchString(r) {
		return errSignatureRFieldMustBe64CharacterHexStringWith0xPrefix
	}
	if !regexp_rs.MatchString(s) {
		return errSignatureSFieldMustBe64CharacterHexStringWith0xPrefix
	}

	// Validate v field (should be 27 or 28, or 0/1 for newer formats)
	if v != 27 && v != 28 && v != 0 && v != 1 {
		return errSignatureVFieldMustBe0Or1Or27Or28
	}

	return nil
}

// EIP-712 signature components.
type EIP712Signature struct {
	V int    `json:"v"`
	R string `json:"r"`
	S string `json:"s"`
}

// Single order payload in the exchange format.
type PlaceOrderRequest struct {
	Symbol           Symbol                             `json:"symbol"`                  // Symbol name (e.g., "BTC-USDT", "ETH-USDT")
	Side             string                             `json:"side"`                    // "buy" or "sell"
	OrderType        string                             `json:"orderType"`               // Order type enum (e.g., "limitGtc", "market", "twap")
	Price            Price                              `json:"price"`                   // Limit Price; JSON is a decimal string (Price_None when unused)
	TriggerPrice     Price                              `json:"triggerPrice"`            // Trigger Price; JSON is a decimal string (required for trigger orders)
	TriggerPriceType snx_lib_api_types.TriggerPriceType `json:"triggerPriceType"`        // "mark" or "last" (defaults to "mark" if empty)
	Quantity         Quantity                           `json:"quantity"`                // Order Quantity; JSON is a decimal string
	ClientOrderId    ClientOrderId                      `json:"clientOrderId,omitempty"` // Client order ID (optional)
	ReduceOnly       bool                               `json:"reduceOnly"`              // Reduce-only flag
	IsTriggerMarket  bool                               `json:"isTriggerMarket"`         // Execution type for trigger orders
	PostOnly         bool                               `json:"postOnly"`
	ClosePosition    bool                               `json:"closePosition"` // Close entire position when triggered (TP/SL only)
	ExpiresAt        *int64                             `json:"expiresAt,omitempty"`
	IntervalSeconds  int64                              `json:"intervalSeconds,omitempty"` // TWAP: interval between chunks in seconds
	DurationSeconds  int64                              `json:"durationSeconds,omitempty"` // TWAP: total execution window in seconds
}

// Enforces price/trigger/isTriggerMarket rules by orderType.
func ValidateOrderTypeConstraints(order PlaceOrderRequest, index int) error {
	switch order.OrderType {
	case "limitGtc", "limitGtd", "limitIoc", "limitAlo":
		if order.Price == Price_None {
			return fmt.Errorf("order %d: price is required for %s orders", index, order.OrderType)
		}
		if order.TriggerPrice != Price_None {
			return fmt.Errorf("order %d: triggerPrice must be empty for %s orders", index, order.OrderType)
		}
		if order.IsTriggerMarket {
			return fmt.Errorf("order %d: isTriggerMarket must be false for %s orders", index, order.OrderType)
		}
		if order.OrderType == "limitGtd" {
			if order.ExpiresAt == nil || *order.ExpiresAt <= 0 {
				return fmt.Errorf("order %d: expiresAt is required for limitGtd orders", index)
			}
		} else if order.ExpiresAt != nil {
			return fmt.Errorf("order %d: expiresAt is only valid for limitGtd orders", index)
		}
	case "market":
		if order.Price != Price_None {
			return fmt.Errorf("order %d: price must be empty for market orders", index)
		}
		if order.TriggerPrice != Price_None {
			return fmt.Errorf("order %d: triggerPrice must be empty for market orders", index)
		}
		if order.IsTriggerMarket {
			return fmt.Errorf("order %d: isTriggerMarket must be false for market orders", index)
		}
		if order.ExpiresAt != nil {
			return fmt.Errorf("order %d: expiresAt is only valid for limitGtd orders", index)
		}
	case API_WKS_triggerSl, API_WKS_triggerTp:
		if order.TriggerPrice == Price_None {
			return fmt.Errorf("order %d: triggerPrice is required for %s orders", index, order.OrderType)
		}
		if order.IsTriggerMarket {
			if order.Price != Price_None {
				return fmt.Errorf("order %d: price must be empty when isTriggerMarket is true", index)
			}
		} else {
			if order.Price == Price_None {
				return fmt.Errorf("order %d: price is required when isTriggerMarket is false", index)
			}
		}
		if order.ExpiresAt != nil {
			return fmt.Errorf("order %d: expiresAt is only valid for limitGtd orders", index)
		}
	case "twap":
		if order.TriggerPrice != Price_None {
			return fmt.Errorf("order %d: triggerPrice must be empty for twap orders", index)
		}
		if order.IntervalSeconds < 0 {
			return fmt.Errorf("order %d: intervalSeconds must not be negative for twap orders", index)
		}
		if order.DurationSeconds <= 0 {
			return fmt.Errorf("order %d: durationSeconds must be positive for twap orders", index)
		}
		if order.ExpiresAt != nil {
			return fmt.Errorf("order %d: expiresAt is only valid for limitGtd orders", index)
		}
	default:
		return fmt.Errorf("order %d: invalid orderType '%s'", index, order.OrderType)
	}

	return nil
}

// Order ID shape used in error responses.
// The venue pointer serializes as null before assignment.
type ErrorOrderIdResponse struct {
	VenueId  *VenueOrderId `json:"venueId"`
	ClientId ClientOrderId `json:"clientId,omitempty"`
}

// Order status with string IDs for API responses.
type OrderStatusResponse struct {
	Resting      *RestingOrderResponse  `json:"resting,omitempty"`
	Filled       *FilledOrderResponse   `json:"filled,omitempty"`
	Canceled     *CanceledOrderResponse `json:"canceled,omitempty"`
	Error        string                 `json:"error,omitempty"`
	ErrorCode    string                 `json:"errorCode,omitempty"`
	ErrorOrderId *ErrorOrderIdResponse  `json:"order,omitempty"`
}

type RestingOrderResponse struct {
	OrderId       OrderId                      `json:"order"` // order (paired)
	DEPRECATED_ID VenueOrderId                 `json:"id"`    // [DEPRECATED] TODO: SNX-4911
	ExpiresAt     *snx_lib_api_types.Timestamp `json:"expiresAt,omitempty"`
}

type FilledOrderResponse struct {
	OrderId       OrderId                      `json:"order"` // order (paired)
	DEPRECATED_ID VenueOrderId                 `json:"id"`    // [DEPRECATED] TODO: SNX-4911
	TotalSize     string                       `json:"totalSize"`
	AvgPrice      Price                        `json:"avgPrice"`
	ExpiresAt     *snx_lib_api_types.Timestamp `json:"expiresAt,omitempty"`
}

type CanceledOrderResponse struct {
	OrderId       OrderId      `json:"order"` // order (paired)
	DEPRECATED_ID VenueOrderId `json:"id"`    // [DEPRECATED] TODO: SNX-4911
}

// Wrapper for statuses with string IDs.
type OrderDataResponse struct {
	Statuses []OrderStatusResponse `json:"statuses"`
}

// Creates an empty order status response.
func NewOrderStatusResponse() OrderStatusResponse { return OrderStatusResponse{} }

// API request for cancelling an order.
type CancelOrderRequest struct {
	Params struct {
		Action  string `json:"action"`  // "cancelOrder"
		OrderId int64  `json:"orderId"` // Order ID to cancel
	} `json:"params"`
	Nonce        Nonce                     `json:"nonce"`                  // Timestamp in milliseconds
	Signature    EIP712Signature           `json:"signature"`              // EIP712 signature with v, r, s fields
	VaultAddress WalletAddress             `json:"vaultAddress,omitempty"` // Optional vault address
	ExpiresAfter int64                     `json:"expiresAfter,omitempty"` // Optional expiration timestamp
	SubAccountId snx_lib_core.SubAccountId `json:"subaccountId,omitempty"` // Subaccount ID for the order
}

// Response from placing orders.
type PlaceOrderResponse struct {
	Status   string `json:"status"` // "ok"
	Response struct {
		Type string `json:"type"` // "order"
		Data struct {
			Statuses []OrderStatusResponse `json:"statuses"`
		} `json:"data"`
	} `json:"response"`
}

// Creates an order-specific response using the generic structure.
func NewOrderResponse(status string, statuses []OrderStatusResponse) PlaceOrderResponse {
	return PlaceOrderResponse{
		Status: status,
		Response: struct {
			Type string `json:"type"`
			Data struct {
				Statuses []OrderStatusResponse `json:"statuses"`
			} `json:"data"`
		}{
			Type: "order",
			Data: struct {
				Statuses []OrderStatusResponse `json:"statuses"`
			}{
				Statuses: statuses,
			},
		},
	}
}

// Creates an error response for order operations.
func NewOrderErrorResponse(errorMsg string) PlaceOrderResponse {
	return NewOrderResponse("error", []OrderStatusResponse{{Error: errorMsg}})
}

// Creates an error response with multiple error statuses.
func NewOrderErrorResponseMulti(errors []string) PlaceOrderResponse {
	statuses := make([]OrderStatusResponse, len(errors))
	for i, err := range errors {
		statuses[i] = OrderStatusResponse{Error: err}
	}
	return NewOrderResponse("error", statuses)
}

// Creates a success response for order operations.
func NewOrderSuccessResponse(statuses []OrderStatusResponse) PlaceOrderResponse {
	return NewOrderResponse("ok", statuses)
}

// Validates the cancel order request.
func ValidateCancelOrder(req *CancelOrderRequest) error {
	// Validate action type
	if req.Params.Action != "cancelOrder" {
		return errActionTypeMustBeCancelOrder
	}

	// Validate order ID
	if req.Params.OrderId == 0 {
		return errOrderIdRequired
	}

	// Validate nonce
	if req.Nonce == 0 {
		return errNonceRequiredRecommendedCurrentTimestampInMilliseconds
	}

	// Validate signature
	if err := validateSignature(req.Signature.R, req.Signature.S, req.Signature.V); err != nil {
		return err
	}

	// TODO: CRITICAL SECURITY - Implement EIP712 signature verification
	// Currently only validating signature FORMAT, not authenticity
	// Need to implement same security checks as ValidateNewOrder

	// Validate expiresAfter if provided
	if req.ExpiresAfter > 0 {
		if req.ExpiresAfter <= int64(req.Nonce) {
			return errExpiresAfterMustBeGreaterThanNonce
		}
	}

	return nil
}
