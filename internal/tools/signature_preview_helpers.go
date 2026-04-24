package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/shopspring/decimal"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	sdkeip712 "github.com/synthetixio/synthetix-go/eip712"
)

func nowUnixSeconds() int64 {
	return snx_lib_utils_time.Now().Unix()
}

func nowUnixMillis() int64 {
	return snx_lib_utils_time.Now().UnixMilli()
}

// Round-trips EIP-712 typed data through canonical JSON so the MCP
// schema generator can emit it as a plain object. Direct emission is
// blocked because apitypes.TypedData is opaque to the generator.
func typedDataToMap(td apitypes.TypedData) (map[string]any, error) {
	serialized, err := sdkeip712.Serialize(td)
	if err != nil {
		return nil, fmt.Errorf("serialize typed data: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(serialized), &out); err != nil {
		return nil, fmt.Errorf("decode typed data: %w", err)
	}
	return out, nil
}

// Returns the 0x-prefixed keccak256 EIP-712 digest, useful for hardware
// signers or offline flows that sign the digest directly.
func eip712DigestHex(td apitypes.TypedData) (string, error) {
	digest, err := sdkeip712.Digest(td)
	if err != nil {
		return "", fmt.Errorf("hash typed data: %w", err)
	}
	return "0x" + hex.EncodeToString(digest), nil
}

// Builds the validated action payload that CreateTradeTypedData expects.
// Exactly one action sub-object must be populated and match the Action
// discriminator. closePosition queries the live position (via the
// broker-signed REST shim) when side is not explicit so the preview
// mirrors what close_position will submit; otherwise side or quantity
// drift produces INVALID_SIGNATURE.
func buildTradePreviewPayload(
	ctx context.Context,
	tradeReads *TradeReadClient,
	tc ToolContext,
	subAccountID int64,
	input previewTradeSignatureInput,
) (any, snx_lib_api_types.RequestAction, error) {
	action := input.Action
	if action == "" {
		return nil, "", fmt.Errorf("action is required")
	}
	switch action {
	case "placeOrders":
		if input.PlaceOrder == nil {
			return nil, "", fmt.Errorf("placeOrder payload is required for action=placeOrders")
		}
		validated, _, err := buildValidatedPlaceOrder(*input.PlaceOrder)
		if err != nil {
			return nil, "", err
		}
		return validated, snx_lib_api_types.RequestAction("placeOrders"), nil
	case "modifyOrder":
		if input.ModifyOrder == nil {
			return nil, "", fmt.Errorf("modifyOrder payload is required for action=modifyOrder")
		}
		payload, _, _, err := buildModifyPayload(modifyOrderInput{
			VenueOrderID:  input.ModifyOrder.VenueOrderID,
			ClientOrderID: input.ModifyOrder.ClientOrderID,
			Price:         input.ModifyOrder.Price,
			Quantity:      input.ModifyOrder.Quantity,
			TriggerPrice:  input.ModifyOrder.TriggerPrice,
		})
		if err != nil {
			return nil, "", err
		}
		return payload, snx_lib_api_types.RequestAction("modifyOrder"), nil
	case "cancelOrders":
		if input.CancelOrder == nil {
			return nil, "", fmt.Errorf("cancelOrder payload is required for action=cancelOrders")
		}
		payload, _, _, err := buildCancelPayload(cancelOrderInput{
			VenueOrderID:  input.CancelOrder.VenueOrderID,
			ClientOrderID: input.CancelOrder.ClientOrderID,
		})
		if err != nil {
			return nil, "", err
		}
		return payload, snx_lib_api_types.RequestAction("cancelOrders"), nil
	case "cancelAllOrders":
		in := cancelAllOrdersInput{}
		if input.CancelAll != nil {
			in.Symbol = input.CancelAll.Symbol
		}
		payload, _, err := buildCancelAllPayload(in)
		if err != nil {
			return nil, "", err
		}
		return payload, snx_lib_api_types.RequestAction("cancelAllOrders"), nil
	case "closePosition":
		if input.ClosePosition == nil {
			return nil, "", fmt.Errorf("closePosition payload is required for action=closePosition")
		}
		if subAccountID <= 0 {
			return nil, "", fmt.Errorf("subAccountId is required to resolve live position for closePosition preview")
		}
		closeMethod := strings.ToLower(strings.TrimSpace(input.ClosePosition.Method))
		if closeMethod == "" {
			closeMethod = "market"
		}
		orderType := "MARKET"
		tif := ""
		price := ""
		switch closeMethod {
		case "market":
		case "limit":
			orderType = "LIMIT"
			tif = "GTC"
			price = input.ClosePosition.LimitPrice
		default:
			return nil, "", fmt.Errorf("method must be market or limit")
		}
		// Derive side (BUY for short, SELL for long) and the
		// default full-position quantity. When the caller passes
		// an explicit side + quantity we skip the REST pre-flight;
		// this is the escape hatch for deployments without a
		// broker-signed positions read.
		positionSide, currentQuantity, err := resolveClosablePositionOrExplicit(
			ctx,
			tradeReads,
			tc,
			input.ClosePosition.Symbol,
			input.ClosePosition.Side,
			input.ClosePosition.Quantity,
		)
		if err != nil {
			return nil, "", err
		}
		closeQuantity := currentQuantity
		if input.ClosePosition.Side == "" && input.ClosePosition.Quantity != "" {
			parsed, parseErr := decimal.NewFromString(input.ClosePosition.Quantity)
			if parseErr != nil {
				return nil, "", fmt.Errorf("invalid close quantity: %w", parseErr)
			}
			if parsed.GreaterThan(currentQuantity) {
				return nil, "", fmt.Errorf("close quantity exceeds current position quantity")
			}
			closeQuantity = parsed
		}
		side := "SELL"
		if strings.EqualFold(positionSide, "short") {
			side = "BUY"
		}
		validated, _, err := buildValidatedPlaceOrder(previewOrderInput{
			Symbol:      input.ClosePosition.Symbol,
			Side:        side,
			Type:        orderType,
			Quantity:    closeQuantity.String(),
			Price:       price,
			TimeInForce: tif,
			ReduceOnly:  true,
		})
		if err != nil {
			return nil, "", err
		}
		return validated, snx_lib_api_types.RequestAction("placeOrders"), nil
	default:
		return nil, "", fmt.Errorf("unknown action %q: expected one of placeOrders, modifyOrder, cancelOrders, cancelAllOrders, closePosition", action)
	}
}

// Anchors the snx_lib_api_types import so future refactors of
// buildValidatedPlaceOrder cannot orphan it without breaking the build.
var _ = snx_lib_api_types.RequestAction("")
