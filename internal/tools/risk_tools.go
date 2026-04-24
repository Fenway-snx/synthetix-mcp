package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type getRateLimitsOutput struct {
	Meta          responseMeta         `json:"_meta"`
	Notes         []string             `json:"notes"`
	PerIP         perIPRateLimitOutput `json:"perIP"`
	PerSubAccount perSubAccountOutput  `json:"perSubAccount"`
}

type perIPRateLimitOutput struct {
	RequestsPerSecond int64          `json:"requestsPerSecond"`
	TokenCosts        map[string]int `json:"tokenCosts"`
	TokensPerWindow   int64          `json:"tokensPerWindow"`
	WindowMs          int64          `json:"windowMs"`
}

type perSubAccountOutput struct {
	RequestsPerSecond int64          `json:"requestsPerSecond"`
	TokenCosts        map[string]int `json:"tokenCosts"`
	TokensPerWindow   int64          `json:"tokensPerWindow"`
	WindowMs          int64          `json:"windowMs"`
}

// schedule_cancel_all (dead-man's switch) is intentionally NOT
// registered. The backend has no support for it in Phase 1, so
// registering a tool that always returns NOT_IMPLEMENTED would only
// teach agents to retry it. Instead it stays in
// get_server_info.disabledFeatures so operators can still see the
// capability gap.
func RegisterRiskTools(
	server *mcp.Server,
	deps *ToolDeps,
) {
	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_rate_limits",
		Description: "Return the current per-IP and per-subaccount rate limits in requests/second. Use this to understand throttling thresholds and plan request pacing.",
	}, func(_ context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, getRateLimitsOutput, error) {
		return nil, buildRateLimitOutput(deps.Cfg, tc.State), nil
	})
}

func buildRateLimitOutput(cfg *config.Config, state *session.State) getRateLimitsOutput {
	effectiveSubaccountLimit := int64(cfg.OrderRateLimiterConfig.GeneralRateLimit)
	if state != nil && state.SubAccountID > 0 {
		if limit, ok := cfg.OrderRateLimiterConfig.SpecificRateLimits[snx_lib_core.SubAccountId(state.SubAccountID)]; ok {
			effectiveSubaccountLimit = int64(limit)
		}
	}
	return getRateLimitsOutput{
		Meta: newResponseMeta(authModeForState(state)),
		Notes: []string{
			"perIP applies at the MCP edge before authentication using the shared token-bucket limiter",
			"perSubAccount applies to authenticated tool requests after session binding using the shared token-bucket limiter",
			"requestsPerSecond is an approximate 1-token baseline derived from tokensPerWindow/windowMs",
		},
		PerIP: perIPRateLimitOutput{
			RequestsPerSecond: int64(cfg.PublicRPSPerIP),
			TokenCosts:        CopyTokenCosts(cfg.IPHandlerTokenCosts),
			TokensPerWindow:   int64(cfg.IPRateLimiterConfig.IPRateLimit),
			WindowMs:          cfg.IPRateLimiterConfig.WindowMs,
		},
		PerSubAccount: perSubAccountOutput{
			RequestsPerSecond: int64(config.DerivedTokensPerSecond(effectiveSubaccountLimit, cfg.OrderRateLimiterConfig.WindowMs)),
			TokenCosts:        CopyTokenCosts(cfg.HandlerTokenCosts),
			TokensPerWindow:   effectiveSubaccountLimit,
			WindowMs:          cfg.OrderRateLimiterConfig.WindowMs,
		},
	}
}
