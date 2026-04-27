package auth

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
)

var (
	errCancelOrdersPayloadAcceptsEitherOrderIdsOrClientOrderIdsNotBoth = errors.New("cancelOrders payload: accepts either orderIds or clientOrderIds, not both")
	errModifyOrderPayloadAcceptsEitherOrderIdOrClientOrderIdNotBoth    = errors.New("modifyOrder payload: accepts either orderId or clientOrderId, not both")
)

func ensureValidatedPlaceOrders(action any) (*snx_lib_api_validation.ValidatedPlaceOrdersAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.PlaceOrdersActionPayload:
		return snx_lib_api_validation.NewValidatedPlaceOrdersAction(v)
	case snx_lib_api_validation.PlaceOrdersActionPayload:
		return snx_lib_api_validation.NewValidatedPlaceOrdersAction(&v)
	case *snx_lib_api_validation.ValidatedPlaceOrdersAction:
		return v, nil
	case snx_lib_api_validation.ValidatedPlaceOrdersAction:
		return &v, nil
	default:
		return nil, fmt.Errorf("unsupported placeOrders payload type %T", action)
	}
}

func ensureValidatedModifyOrder(action any) (*snx_lib_api_validation.ValidatedModifyOrderAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.ModifyOrderActionPayload:
		return snx_lib_api_validation.NewValidatedModifyOrderAction(v)
	case snx_lib_api_validation.ModifyOrderActionPayload:
		return snx_lib_api_validation.NewValidatedModifyOrderAction(&v)
	case *snx_lib_api_validation.ValidatedModifyOrderAction:
		return v, nil
	case snx_lib_api_validation.ValidatedModifyOrderAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeModifyOrderAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedModifyOrderAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported modifyOrder payload type %T", action)
	}
}

func ensureValidatedModifyOrderByCloid(action any) (*snx_lib_api_validation.ValidatedModifyOrderByCloidAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.ModifyOrderByCloidActionPayload:
		return snx_lib_api_validation.NewValidatedModifyOrderByCloidAction(v)
	case snx_lib_api_validation.ModifyOrderByCloidActionPayload:
		return snx_lib_api_validation.NewValidatedModifyOrderByCloidAction(&v)
	case *snx_lib_api_validation.ValidatedModifyOrderByCloidAction:
		return v, nil
	case snx_lib_api_validation.ValidatedModifyOrderByCloidAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeModifyOrderByCloidAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedModifyOrderByCloidAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported modifyOrder-by-cloid payload type %T", action)
	}
}

func ensureValidatedCancelAllOrders(action any) (*snx_lib_api_validation.ValidatedCancelAllOrdersAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.CancelAllOrdersActionPayload:
		return snx_lib_api_validation.NewValidatedCancelAllOrdersAction(v)
	case snx_lib_api_validation.CancelAllOrdersActionPayload:
		return snx_lib_api_validation.NewValidatedCancelAllOrdersAction(&v)
	case *snx_lib_api_validation.ValidatedCancelAllOrdersAction:
		return v, nil
	case snx_lib_api_validation.ValidatedCancelAllOrdersAction:
		return &v, nil
	default:
		return nil, fmt.Errorf("unsupported cancelAllOrders payload type %T", action)
	}
}

func ensureValidatedCancelOrders(action any) (*snx_lib_api_validation.ValidatedCancelOrdersAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.CancelOrdersActionPayload:
		return snx_lib_api_validation.NewValidatedCancelOrdersAction(v)
	case snx_lib_api_validation.CancelOrdersActionPayload:
		return snx_lib_api_validation.NewValidatedCancelOrdersAction(&v)
	case *snx_lib_api_validation.ValidatedCancelOrdersAction:
		return v, nil
	case snx_lib_api_validation.ValidatedCancelOrdersAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeCancelOrdersAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedCancelOrdersAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported cancelOrders payload type %T", action)
	}
}

func ensureValidatedCancelOrdersByCloid(action any) (*snx_lib_api_validation.ValidatedCancelOrdersByCloidAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.CancelOrdersByCloidActionPayload:
		return snx_lib_api_validation.NewValidatedCancelOrdersByCloidAction(v)
	case snx_lib_api_validation.CancelOrdersByCloidActionPayload:
		return snx_lib_api_validation.NewValidatedCancelOrdersByCloidAction(&v)
	case *snx_lib_api_validation.ValidatedCancelOrdersByCloidAction:
		return v, nil
	case snx_lib_api_validation.ValidatedCancelOrdersByCloidAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeCancelOrdersByCloidAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedCancelOrdersByCloidAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported cancelOrders-by-cloid payload type %T", action)
	}
}

func ensureValidatedScheduleCancel(action any) (*snx_lib_api_validation.ValidatedScheduleCancelAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.ScheduleCancelActionPayload:
		return snx_lib_api_validation.NewValidatedScheduleCancelAction(v)
	case snx_lib_api_validation.ScheduleCancelActionPayload:
		return snx_lib_api_validation.NewValidatedScheduleCancelAction(&v)
	case *snx_lib_api_validation.ValidatedScheduleCancelAction:
		return v, nil
	case snx_lib_api_validation.ValidatedScheduleCancelAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeScheduleCancelAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedScheduleCancelAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported scheduleCancel payload type %T", action)
	}
}

func ensureValidatedWithdrawCollateral(action any) (*snx_lib_api_validation.ValidatedWithdrawCollateralAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.ValidatedWithdrawCollateralAction:
		return v, nil
	case snx_lib_api_validation.ValidatedWithdrawCollateralAction:
		return &v, nil
	case *snx_lib_api_validation.WithdrawCollateralActionPayload:
		return snx_lib_api_validation.NewValidatedWithdrawCollateralAction(v)
	case snx_lib_api_validation.WithdrawCollateralActionPayload:
		return snx_lib_api_validation.NewValidatedWithdrawCollateralAction(&v)
	default:
		return nil, fmt.Errorf("unsupported withdrawCollateral payload type %T", action)
	}
}

func ensureValidatedAddDelegatedSigner(action any) (*snx_lib_api_validation.ValidatedAddDelegatedSignerAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.AddDelegatedSignerActionPayload:
		return snx_lib_api_validation.NewValidatedAddDelegatedSignerAction(v)
	case snx_lib_api_validation.AddDelegatedSignerActionPayload:
		return snx_lib_api_validation.NewValidatedAddDelegatedSignerAction(&v)
	case *snx_lib_api_validation.ValidatedAddDelegatedSignerAction:
		return v, nil
	case snx_lib_api_validation.ValidatedAddDelegatedSignerAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeAddDelegatedSignerAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedAddDelegatedSignerAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported addDelegatedSigner payload type %T", action)
	}
}

func ensureValidatedRemoveDelegatedSigner(action any) (*snx_lib_api_validation.ValidatedRemoveDelegatedSignerAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.RemoveDelegatedSignerActionPayload:
		return snx_lib_api_validation.NewValidatedRemoveDelegatedSignerAction(v)
	case snx_lib_api_validation.RemoveDelegatedSignerActionPayload:
		return snx_lib_api_validation.NewValidatedRemoveDelegatedSignerAction(&v)
	case *snx_lib_api_validation.ValidatedRemoveDelegatedSignerAction:
		return v, nil
	case snx_lib_api_validation.ValidatedRemoveDelegatedSignerAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeRemoveDelegatedSignerAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedRemoveDelegatedSignerAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported removeDelegatedSigner payload type %T", action)
	}
}

func ensureValidatedRemoveAllDelegatedSigners(action any) (*snx_lib_api_validation.ValidatedRemoveAllDelegatedSignersAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.RemoveAllDelegatedSignersActionPayload:
		return snx_lib_api_validation.NewValidatedRemoveAllDelegatedSignersAction(v)
	case snx_lib_api_validation.RemoveAllDelegatedSignersActionPayload:
		return snx_lib_api_validation.NewValidatedRemoveAllDelegatedSignersAction(&v)
	case *snx_lib_api_validation.ValidatedRemoveAllDelegatedSignersAction:
		return v, nil
	case snx_lib_api_validation.ValidatedRemoveAllDelegatedSignersAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeRemoveAllDelegatedSignersAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedRemoveAllDelegatedSignersAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported removeAllDelegatedSigners payload type %T", action)
	}
}

func ensureValidatedUpdateLeverage(action any) (*snx_lib_api_validation.ValidatedUpdateLeverageAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.UpdateLeverageActionPayload:
		return snx_lib_api_validation.NewValidatedUpdateLeverageAction(v)
	case snx_lib_api_validation.UpdateLeverageActionPayload:
		return snx_lib_api_validation.NewValidatedUpdateLeverageAction(&v)
	case *snx_lib_api_validation.ValidatedUpdateLeverageAction:
		return v, nil
	case snx_lib_api_validation.ValidatedUpdateLeverageAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeUpdateLeverageAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedUpdateLeverageAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported updateLeverage payload type %T", action)
	}
}

func ensureValidatedCreateSubaccount(action any) (*snx_lib_api_validation.ValidatedCreateSubaccountAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.CreateSubaccountActionPayload:
		return snx_lib_api_validation.NewValidatedCreateSubaccountAction(v)
	case snx_lib_api_validation.CreateSubaccountActionPayload:
		return snx_lib_api_validation.NewValidatedCreateSubaccountAction(&v)
	case *snx_lib_api_validation.ValidatedCreateSubaccountAction:
		return v, nil
	case snx_lib_api_validation.ValidatedCreateSubaccountAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeCreateSubaccountAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedCreateSubaccountAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported createSubaccount payload type %T", action)
	}
}

func ensureValidatedTransferCollateral(action any) (*snx_lib_api_validation.ValidatedTransferCollateralAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.TransferCollateralActionPayload:
		return snx_lib_api_validation.NewValidatedTransferCollateralAction(v)
	case snx_lib_api_validation.TransferCollateralActionPayload:
		return snx_lib_api_validation.NewValidatedTransferCollateralAction(&v)
	case *snx_lib_api_validation.ValidatedTransferCollateralAction:
		return v, nil
	case snx_lib_api_validation.ValidatedTransferCollateralAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeTransferCollateralAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedTransferCollateralAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported transferCollateral payload type %T", action)
	}
}

func ensureValidatedUpdateSubAccountName(action any) (*snx_lib_api_validation.ValidatedUpdateSubAccountNameAction, error) {
	switch v := action.(type) {
	case *snx_lib_api_validation.UpdateSubAccountNameActionPayload:
		return snx_lib_api_validation.NewValidatedUpdateSubAccountNameAction(v)
	case snx_lib_api_validation.UpdateSubAccountNameActionPayload:
		return snx_lib_api_validation.NewValidatedUpdateSubAccountNameAction(&v)
	case *snx_lib_api_validation.ValidatedUpdateSubAccountNameAction:
		return v, nil
	case snx_lib_api_validation.ValidatedUpdateSubAccountNameAction:
		return &v, nil
	case map[string]any:
		decoded, err := snx_lib_api_validation.DecodeUpdateSubAccountNameAction(v)
		if err != nil {
			return nil, err
		}
		return snx_lib_api_validation.NewValidatedUpdateSubAccountNameAction(decoded)
	default:
		return nil, fmt.Errorf("unsupported updateSubAccountName payload type %T", action)
	}
}

// Utility function that takes a pointer to a string-like object and returns
// its string-form if non-`nil`, or the empty string otherwise.
func stringOrEmpty[S ~string](val *S) string {
	if val != nil {
		return string(*val)
	} else {
		return ""
	}
}

func isCancelOrdersByCloidAction(action any) bool {
	switch action.(type) {
	case *snx_lib_api_validation.CancelOrdersByCloidActionPayload,
		snx_lib_api_validation.CancelOrdersByCloidActionPayload,
		*snx_lib_api_validation.ValidatedCancelOrdersByCloidAction,
		snx_lib_api_validation.ValidatedCancelOrdersByCloidAction:
		return true
	case map[string]any:
		_, ok := action.(map[string]any)["clientOrderIds"]
		return ok
	default:
		return false
	}
}

func isModifyOrderByCloidAction(action any) bool {
	switch action.(type) {
	case *snx_lib_api_validation.ModifyOrderByCloidActionPayload,
		snx_lib_api_validation.ModifyOrderByCloidActionPayload,
		*snx_lib_api_validation.ValidatedModifyOrderByCloidAction,
		snx_lib_api_validation.ValidatedModifyOrderByCloidAction:
		return true
	case map[string]any:
		_, ok := action.(map[string]any)["clientOrderId"]
		return ok
	default:
		return false
	}
}

func hasConflictingOrderIdentifiers(action any, venueField, clientField string) bool {
	payload, ok := action.(map[string]any)
	if !ok {
		return false
	}

	_, hasVenueField := payload[venueField]
	_, hasClientField := payload[clientField]

	return hasVenueField && hasClientField
}

func toPlaceOrdersTypedOrders(payload *snx_lib_api_validation.PlaceOrdersActionPayload) []map[string]any {
	orders := make([]map[string]any, len(payload.Orders))
	for i, order := range payload.Orders {
		orders[i] = map[string]any{
			"symbol":          snx_lib_api_types.SymbolToString(order.Symbol), // TODO: update with SNX-6083
			"side":            order.Side,
			"orderType":       order.OrderType,
			"price":           string(order.Price),
			"triggerPrice":    string(order.TriggerPrice),
			"quantity":        string(order.Quantity),
			"reduceOnly":      order.ReduceOnly,
			"isTriggerMarket": order.IsTriggerMarket,
			"clientOrderId":   snx_lib_api_types.ClientOrderIdToStringUnvalidated(order.ClientOrderId),
			"closePosition":   order.ClosePosition,
		}
	}
	return orders
}

// Constructs EIP-712 typed data for trade-related actions.
//
// It centralizes message + type selection per request type so handlers stay
// thin.
//
// Note: subAccountIdStr is accepted to avoid importing service-layer types
// here.
func CreateTradeTypedData(
	subAccountId SubAccountId,
	nonce Nonce,
	expiresAfter int64,
	requestAction RequestAction,
	action any,
	domainName, domainVersion string,
	chainID int,
) (apitypes.TypedData, error) {
	subAccountIdStr := string(subAccountId)

	domain := GetEIP712Domain(domainName, domainVersion, chainID)

	message := make(map[string]any)
	var primaryType string

	message[API_WKS_expiresAfter] = strconv.FormatInt(expiresAfter, 10)
	message[API_WKS_subAccountId] = subAccountIdStr

	switch requestAction {
	case "modifyOrder":
		message[API_WKS_nonce] = nonce.String()
		if hasConflictingOrderIdentifiers(action, "orderId", "clientOrderId") {
			return apitypes.TypedData{}, errModifyOrderPayloadAcceptsEitherOrderIdOrClientOrderIdNotBoth
		}
		if isModifyOrderByCloidAction(action) {
			validated, err := ensureValidatedModifyOrderByCloid(action)
			if err != nil {
				return apitypes.TypedData{}, fmt.Errorf("modifyOrder payload: %w", err)
			}
			payload := validated.Payload

			primaryType = "ModifyOrderByCloid"
			message["clientOrderId"] = snx_lib_api_types.ClientOrderIdToStringUnvalidated(validated.ClientOrderId)
			message[API_WKS_price] = stringOrEmpty(payload.Price)
			message[API_WKS_quantity] = stringOrEmpty(payload.Quantity)
			message[API_WKS_triggerPrice] = stringOrEmpty(payload.TriggerPrice)
		} else {
			validated, err := ensureValidatedModifyOrder(action)
			if err != nil {
				return apitypes.TypedData{}, fmt.Errorf("modifyOrder payload: %w", err)
			}
			payload := validated.Payload
			orderIDValue := string(validated.VenueOrderId)

			primaryType = "ModifyOrder"
			message[API_WKS_orderId] = orderIDValue
			message[API_WKS_price] = stringOrEmpty(payload.Price)
			message[API_WKS_quantity] = stringOrEmpty(payload.Quantity)
			message[API_WKS_triggerPrice] = stringOrEmpty(payload.TriggerPrice)
		}

	case "cancelAllOrders":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedCancelAllOrders(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("cancelAllOrders payload: %w", err)
		}

		primaryType = "CancelAllOrders"
		message[API_WKS_symbols] = snx_lib_api_types.SymbolsToStringsUnfiltered(validated.Symbols)

	case "cancelOrders":
		message[API_WKS_nonce] = nonce.String()
		if hasConflictingOrderIdentifiers(action, "orderIds", "clientOrderIds") {
			return apitypes.TypedData{}, errCancelOrdersPayloadAcceptsEitherOrderIdsOrClientOrderIdsNotBoth
		}
		if isCancelOrdersByCloidAction(action) {
			validated, err := ensureValidatedCancelOrdersByCloid(action)
			if err != nil {
				return apitypes.TypedData{}, fmt.Errorf("cancelOrders payload: %w", err)
			}
			encodedIds := make([]string, len(validated.ClientOrderIds))
			for i, clientOrderId := range validated.ClientOrderIds {
				encodedIds[i] = snx_lib_api_types.ClientOrderIdToStringUnvalidated(clientOrderId)
			}

			primaryType = "CancelOrdersByCloid"
			message["clientOrderIds"] = encodedIds
		} else {
			validated, err := ensureValidatedCancelOrders(action)
			if err != nil {
				return apitypes.TypedData{}, fmt.Errorf("cancelOrders payload: %w", err)
			}

			encodedIds := make([]any, len(validated.VenueOrderIds))
			for i, venueOrderId := range validated.VenueOrderIds {
				u := snx_lib_api_types.VenueOrderIdToUintUnvalidated(venueOrderId)

				encodedIds[i] = math.NewHexOrDecimal256(int64(u))
			}

			primaryType = "CancelOrders"
			message[API_WKS_orderIds] = encodedIds
		}

	case "placeOrders":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedPlaceOrders(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("placeOrders payload: %w", err)
		}
		payload := validated.Payload

		primaryType = "PlaceOrders"
		message[API_WKS_grouping] = string(payload.Grouping)
		message[API_WKS_orders] = toPlaceOrdersTypedOrders(payload)

	case "scheduleCancel":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedScheduleCancel(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("scheduleCancel payload: %w", err)
		}

		primaryType = "ScheduleCancel"
		message[API_WKS_timeoutSeconds] = strconv.FormatInt(validated.TimeoutSeconds, 10)

	case "addDelegatedSigner":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedAddDelegatedSigner(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("addDelegatedSigner payload: %w", err)
		}
		payload := validated.Payload

		primaryType = "AddDelegatedSigner"
		message["delegateAddress"] = snx_lib_api_types.WalletAddressToString(payload.DelegateAddress)
		message["permissions"] = payload.Permissions
		// expiresAt is optional - use 0 if not provided
		expiresAtValue := int64(0)
		if payload.ExpiresAt != nil {
			expiresAtValue = *payload.ExpiresAt
		}
		message["expiresAt"] = strconv.FormatInt(expiresAtValue, 10)

	case "removeDelegatedSigner":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedRemoveDelegatedSigner(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("removeDelegatedSigner payload: %w", err)
		}
		payload := validated.Payload

		primaryType = "RemoveDelegatedSigner"
		message["delegateAddress"] = snx_lib_api_types.WalletAddressToString(payload.DelegateAddress)

	case "removeAllDelegatedSigners":
		message[API_WKS_nonce] = nonce.String()
		_, err := ensureValidatedRemoveAllDelegatedSigners(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("removeAllDelegatedSigners payload: %w", err)
		}

		primaryType = "RemoveAllDelegatedSigners"

	case "updateLeverage":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedUpdateLeverage(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("updateLeverage payload: %w", err)
		}
		payload := validated.Payload

		primaryType = "UpdateLeverage"
		message[API_WKS_symbol] = snx_lib_api_types.SymbolToString(payload.Symbol) // TODO: update with SNX-6083
		message[API_WKS_leverage] = payload.Leverage

	case "createSubaccount":
		// createSubaccount uses masterSubAccountId for ownership verification
		// The subAccountId field is used as masterSubAccountId to prove user owns an existing subaccount
		message["masterSubAccountId"] = subAccountIdStr
		delete(message, API_WKS_subAccountId)
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedCreateSubaccount(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("createSubaccount payload: %w", err)
		}
		payload := validated.Payload

		primaryType = "CreateSubaccount"
		message[API_WKS_name] = payload.Name

	case "transferCollateral":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedTransferCollateral(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("transferCollateral payload: %w", err)
		}

		primaryType = "TransferCollateral"
		message["to"] = strconv.FormatInt(int64(validated.To), 10) // TODO: specific conversion function
		message[API_WKS_symbol] = string(validated.Symbol)
		message[API_WKS_amount] = validated.Amount.String()

	case "updateSubAccountName":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedUpdateSubAccountName(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("updateSubAccountName payload: %w", err)
		}
		payload := validated.Payload

		primaryType = "UpdateSubAccountName"
		message[API_WKS_name] = payload.Name
		message[API_WKS_subAccountId] = subAccountIdStr

	case "withdrawCollateral":
		message[API_WKS_nonce] = nonce.String()
		validated, err := ensureValidatedWithdrawCollateral(action)
		if err != nil {
			return apitypes.TypedData{}, fmt.Errorf("withdrawCollateral payload: %w", err)
		}
		payload := validated.Payload

		primaryType = "WithdrawCollateral"
		message[API_WKS_symbol] = snx_lib_api_types.AssetNameToString(payload.Symbol) // TODO: update with SNX-6083
		message[API_WKS_amount] = payload.Amount
		message[API_WKS_destination] = snx_lib_api_types.WalletAddressToString(payload.Destination)

	default:
		// Only get* actions (which don't require nonce) use SubAccountAction
		if requestAction.NonceRequired() {
			return apitypes.TypedData{}, fmt.Errorf("unknown action type: %s", requestAction)
		}
		primaryType = "SubAccountAction"
		message[API_WKS_action] = string(requestAction)
		// Include nonce if provided (backwards compatibility for old clients)
		if nonce > 0 {
			message[API_WKS_nonce] = nonce.String()
		}
	}

	var types map[string][]apitypes.Type
	switch primaryType {
	case "ModifyOrderByCloid":
		types = GetModifyOrderByCloidTypes()
	case "ModifyOrder":
		types = GetModifyOrderTypes()
	case "CancelAllOrders":
		types = GetCancelAllOrdersTypes()
	case "CancelOrdersByCloid":
		types = GetCancelOrdersByCloidTypes()
	case "CancelOrders":
		types = GetCancelOrderTypes()
	case "PlaceOrders":
		types = GetPlaceOrdersTypes()
	case "ScheduleCancel":
		types = GetScheduleCancelTypes()
	case "WithdrawCollateral":
		types = GetWithdrawCollateralTypes()
	case "AddDelegatedSigner":
		types = GetAddDelegatedSignerTypes()
	case "RemoveDelegatedSigner":
		types = GetRemoveDelegatedSignerTypes()
	case "RemoveAllDelegatedSigners":
		types = GetRemoveAllDelegatedSignersTypes()
	case "UpdateLeverage":
		types = GetUpdateLeverageTypes()
	case "CreateSubaccount":
		types = GetCreateSubaccountTypes()
	case "TransferCollateral":
		types = GetTransferCollateralTypes()
	case "UpdateSubAccountName":
		types = GetUpdateSubAccountNameTypes()
	default:
		// SubAccountAction for get* actions
		// Conditionally include nonce for backwards compatibility with old clients
		subAccountActionFields := []apitypes.Type{
			{Name: "subAccountId", Type: "uint256"},
			{Name: "action", Type: "string"},
		}
		if nonce > 0 {
			subAccountActionFields = append(subAccountActionFields, apitypes.Type{Name: "nonce", Type: "uint256"})
		}
		subAccountActionFields = append(subAccountActionFields, apitypes.Type{Name: "expiresAfter", Type: "uint256"})

		types = map[string][]apitypes.Type{
			"EIP712Domain": getEIP712DomainFields(),
			primaryType:    subAccountActionFields,
		}
	}

	return apitypes.TypedData{
		Types:       types,
		PrimaryType: primaryType,
		Domain:      domain,
		Message:     message,
	}, nil
}

// BuildSignatureHex composes an Ethereum signature hex string from r,s,v parts.
func BuildSignatureHex(signature TradeSignature) string {
	rTrim := signature.R
	sTrim := signature.S
	if len(rTrim) >= 2 && (rTrim[:2] == "0x" || rTrim[:2] == "0X") {
		rTrim = rTrim[2:]
	}
	if len(sTrim) >= 2 && (sTrim[:2] == "0x" || sTrim[:2] == "0X") {
		sTrim = sTrim[2:]
	}
	return fmt.Sprintf("0x%s%s%02x", rTrim, sTrim, signature.V)
}
