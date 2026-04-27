package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/synthetixio/synthetix-go/types"
)

type fakeGuardrailHydrationClient struct {
	openOrders []risksnapshot.HydrationOrder
	positions  []risksnapshot.HydrationPosition
}

func (f fakeGuardrailHydrationClient) GetOpenOrders(context.Context, int64, int, int) ([]risksnapshot.HydrationOrder, error) {
	return f.openOrders, nil
}

func (f fakeGuardrailHydrationClient) GetPositions(context.Context, int64, int, int) ([]risksnapshot.HydrationPosition, error) {
	return f.positions, nil
}

type fakeGuardrailPriceReader map[string]types.MarketPriceResponse

func (f fakeGuardrailPriceReader) GetMarketPrices(context.Context) (map[string]types.MarketPriceResponse, error) {
	return map[string]types.MarketPriceResponse(f), nil
}

func TestGuardrailOrderErrorsEnforcesNotionalCaps(t *testing.T) {
	ctx := context.Background()
	manager := risksnapshot.NewManager(fakeGuardrailHydrationClient{
		positions: []risksnapshot.HydrationPosition{
			{Symbol: "BTC-USDT", Side: "long", Quantity: "1"},
		},
	})
	state := &session.State{
		SubAccountID: 1,
		AgentGuardrails: &guardrails.Config{
			Preset:              guardrails.PresetStandard,
			AllowedSymbols:      []string{"*"},
			AllowedOrderTypes:   []string{"*"},
			MaxOrderNotional:    "100",
			MaxPositionNotional: "1000",
		},
	}
	prices := fakeGuardrailPriceReader{
		"BTC-USDT": {MarkPrice: "100"},
	}

	err := enforcePlaceOrderGuardrails(ctx, "session", state, manager, prices, normalizedOrderOutput{
		Symbol:   "BTC-USDT",
		Side:     "BUY",
		Type:     "MARKET",
		Quantity: "2",
	})
	if err == nil || !strings.Contains(err.Error(), "maxOrderNotional") {
		t.Fatalf("expected maxOrderNotional violation, got %v", err)
	}
}

func TestGuardrailOrderErrorsEnforcesPositionNotionalCaps(t *testing.T) {
	ctx := context.Background()
	manager := risksnapshot.NewManager(fakeGuardrailHydrationClient{
		positions: []risksnapshot.HydrationPosition{
			{Symbol: "BTC-USDT", Side: "long", Quantity: "1"},
		},
	})
	state := &session.State{
		SubAccountID: 1,
		AgentGuardrails: &guardrails.Config{
			Preset:              guardrails.PresetStandard,
			AllowedSymbols:      []string{"*"},
			AllowedOrderTypes:   []string{"*"},
			MaxOrderNotional:    "1000",
			MaxPositionNotional: "250",
		},
	}
	prices := fakeGuardrailPriceReader{
		"BTC-USDT": {MarkPrice: "100"},
	}

	err := enforcePlaceOrderGuardrails(ctx, "session", state, manager, prices, normalizedOrderOutput{
		Symbol:   "BTC-USDT",
		Side:     "BUY",
		Type:     "MARKET",
		Quantity: "2",
	})
	if err == nil || !strings.Contains(err.Error(), "maxPositionNotional") {
		t.Fatalf("expected maxPositionNotional violation, got %v", err)
	}
}

func TestCancelOrderGuardrailsIgnoreOrderSizeCaps(t *testing.T) {
	ctx := context.Background()
	manager := risksnapshot.NewManager(fakeGuardrailHydrationClient{
		openOrders: []risksnapshot.HydrationOrder{
			{
				ClientOrderID:     "urgent-cancel",
				OrderType:         "LIMIT",
				Price:             "100000",
				Quantity:          "10",
				RemainingQuantity: "10",
				Side:              "BUY",
				Symbol:            "BTC-USDT",
				VenueOrderID:      "123",
			},
		},
	})
	state := &session.State{
		SubAccountID: 1,
		AgentGuardrails: &guardrails.Config{
			Preset:              guardrails.PresetStandard,
			AllowedSymbols:      []string{"BTC-USDT"},
			AllowedOrderTypes:   []string{"MARKET"},
			MaxOrderNotional:    "1",
			MaxOrderQuantity:    "0.0001",
			MaxPositionNotional: "1",
			MaxPositionQuantity: "0.0001",
		},
	}

	order, err := enforceCancelOrderGuardrails(ctx, "session", state, manager, cancelOrderInput{}, "123", "")
	if err != nil {
		t.Fatalf("expected cancel to ignore size/notional/type caps, got %v", err)
	}
	if order == nil || order.ClientOrderID != "urgent-cancel" {
		t.Fatalf("expected open order context, got %#v", order)
	}
}

func TestCancelAllGuardrailsIgnoreOrderSizeCaps(t *testing.T) {
	ctx := context.Background()
	manager := risksnapshot.NewManager(fakeGuardrailHydrationClient{
		openOrders: []risksnapshot.HydrationOrder{
			{
				ClientOrderID:     "urgent-cancel-all",
				OrderType:         "LIMIT",
				Price:             "100000",
				Quantity:          "10",
				RemainingQuantity: "10",
				Side:              "BUY",
				Symbol:            "BTC-USDT",
				VenueOrderID:      "123",
			},
		},
	})
	state := &session.State{
		SubAccountID: 1,
		AgentGuardrails: &guardrails.Config{
			Preset:              guardrails.PresetStandard,
			AllowedSymbols:      []string{"BTC-USDT"},
			AllowedOrderTypes:   []string{"MARKET"},
			MaxOrderNotional:    "1",
			MaxOrderQuantity:    "0.0001",
			MaxPositionNotional: "1",
			MaxPositionQuantity: "0.0001",
		},
	}

	if err := enforceCancelAllGuardrails(ctx, "session", state, manager, cancelAllOrdersInput{}); err != nil {
		t.Fatalf("expected cancel_all to ignore size/notional/type caps, got %v", err)
	}
}
