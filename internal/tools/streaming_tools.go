package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/streaming"
)

type subscribeInput struct {
	Subscriptions []subscribeRequestItem `json:"subscriptions" jsonschema:"List of channel subscriptions to add. Each subscription specifies a channel and optional params."`
}

type subscribeRequestItem struct {
	Channel string         `json:"channel" jsonschema:"Channel name: candles, marketPrices, orderbook, trades."`
	Params  map[string]any `json:"params,omitempty" jsonschema:"Channel-specific parameters, e.g. {\"symbol\":\"BTC-USDT\"} for market channels, {\"timeframe\":\"1m\"} for candles."`
}

type subscribeOutput struct {
	Meta                responseMeta             `json:"_meta"`
	ActiveSubscriptions []streaming.Subscription `json:"activeSubscriptions"`
	Warnings            []string                 `json:"warnings"`
}

type unsubscribeInput struct {
	Channels []string `json:"channels" jsonschema:"Channel names to unsubscribe from, e.g. ['marketPrices','orderbook']."`
	Symbol   string   `json:"symbol,omitempty" jsonschema:"Only unsubscribe from channels matching this symbol. Omit to unsubscribe from the channel across all symbols."`
}

type unsubscribeOutput struct {
	Meta                   responseMeta             `json:"_meta"`
	RemainingSubscriptions []streaming.Subscription `json:"remainingSubscriptions"`
	RemovedSubscriptions   []streaming.Subscription `json:"removedSubscriptions"`
}

func RegisterStreamingTools(
	server *mcp.Server,
	deps *ToolDeps,
	manager *streaming.Manager,
) {
	subscribeTool := &mcp.Tool{
		Name:        "subscribe",
		Description: "Subscribe to real-time public event streams: candles, marketPrices, orderbook, trades. Pass `symbol` (and `timeframe` for candles, `depth` for orderbook) in params. Private/account streams are not available on this MCP endpoint; connect to /v1/ws/trade via the mcp-signer-bridge for fills, margin and order events. Events are delivered as server-sent notifications on the MCP connection.",
	}
	applyToolSchemas[subscribeInput, subscribeOutput](subscribeTool)
	mcp.AddTool(server, subscribeTool, func(ctx context.Context, req *mcp.CallToolRequest, input subscribeInput) (*mcp.CallToolResult, subscribeOutput, error) {
		if len(input.Subscriptions) == 0 {
			return toolErrorResponse[subscribeOutput](fmt.Errorf("subscriptions are required"))
		}

		sessionID := sessionIDFromRequest(req)
		state, err := loadSessionState(ctx, deps.Store, sessionID)
		if err != nil && err != session.ErrSessionNotFound {
			return toolErrorResponse[subscribeOutput](err)
		}
		state, err = sanitizeSessionState(ctx, deps.Store, sessionID, state, deps.Verifier)
		if err != nil {
			return toolErrorResponse[subscribeOutput](err)
		}
		authMode := authModeForState(state)
		if err := touchSession(ctx, deps.Store, sessionID, deps.Cfg.SessionTTL); err != nil {
			return toolErrorResponse[subscribeOutput](err)
		}

		requests := make([]streaming.SubscribeRequest, 0, len(input.Subscriptions))
		for _, sub := range input.Subscriptions {
			// Private/account streams were removed in Phase 1.4 along
			// This endpoint has no source for
			// per-subaccount events. Hard-reject the legacy channel
			// name so callers get a clear remediation path rather than
			// silently subscribing and waiting for events that never
			// arrive.
			if sub.Channel == "accountEvents" {
				return newToolErrorResult(
					"PRIVATE_STREAMS_NOT_SUPPORTED",
					"accountEvents subscriptions are not supported on this MCP endpoint.",
					"Connect to /v1/ws/trade via mcp-signer-bridge (or the Synthetix trading SDK) to receive fills, margin and order events for your subaccount.",
				), subscribeOutput{}, nil
			}
			requests = append(requests, streaming.SubscribeRequest{
				Channel: sub.Channel,
				Params:  sub.Params,
			})
		}

		active, warnings, err := manager.Subscribe(sessionID, requests)
		if err != nil {
			return toolErrorResponse[subscribeOutput](err)
		}

		return nil, subscribeOutput{
			Meta:                newResponseMeta(authMode),
			ActiveSubscriptions: active,
			Warnings:            warnings,
		}, nil
	})

	unsubscribeTool := &mcp.Tool{
		Name:        "unsubscribe",
		Description: "Remove one or more active streaming subscriptions. Specify channel names to unsubscribe from, and optionally a symbol to only remove subscriptions for that market. Returns remaining active subscriptions.",
	}
	applyToolSchemas[unsubscribeInput, unsubscribeOutput](unsubscribeTool)
	mcp.AddTool(server, unsubscribeTool, func(ctx context.Context, req *mcp.CallToolRequest, input unsubscribeInput) (*mcp.CallToolResult, unsubscribeOutput, error) {
		if len(input.Channels) == 0 {
			return toolErrorResponse[unsubscribeOutput](fmt.Errorf("channels are required"))
		}
		sessionID := sessionIDFromRequest(req)
		state, err := loadSessionState(ctx, deps.Store, sessionID)
		if err != nil && err != session.ErrSessionNotFound {
			return toolErrorResponse[unsubscribeOutput](err)
		}
		state, err = sanitizeSessionState(ctx, deps.Store, sessionID, state, deps.Verifier)
		if err != nil {
			return toolErrorResponse[unsubscribeOutput](err)
		}
		if err := touchSession(ctx, deps.Store, sessionID, deps.Cfg.SessionTTL); err != nil {
			return toolErrorResponse[unsubscribeOutput](err)
		}

		removed, remaining, err := manager.Unsubscribe(sessionID, input.Channels, input.Symbol)
		if err != nil {
			return toolErrorResponse[unsubscribeOutput](err)
		}
		return nil, unsubscribeOutput{
			Meta:                   newResponseMeta(authModeForState(state)),
			RemainingSubscriptions: remaining,
			RemovedSubscriptions:   removed,
		}, nil
	})
}
