package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type authStatusOutput struct {
	Meta              responseMeta `json:"_meta"`
	Authenticated     bool         `json:"authenticated"`
	AuthMode          string       `json:"authMode"`
	BrokerEnabled     bool         `json:"brokerEnabled"`
	BrokerReady       bool         `json:"brokerReady"`
	BrokerSubAccountID int64       `json:"brokerSubAccountId,omitempty,string"`
	Issues            []string     `json:"issues"`
	SessionSubAccountID int64      `json:"sessionSubAccountId,omitempty,string"`
	SessionWallet      string      `json:"sessionWallet,omitempty"`
}

type healthCheckOutput struct {
	LatencyMs int64  `json:"latencyMs"`
	OK        bool   `json:"ok"`
	Status    string `json:"status"`
}

type systemHealthOutput struct {
	Meta responseMeta      `json:"_meta"`
	Auth authStatusOutput  `json:"auth"`
	REST healthCheckOutput `json:"rest"`
	WS   healthCheckOutput `json:"ws"`
}

func RegisterSystemTools(server *mcp.Server, deps *ToolDeps) {
	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_auth_status",
		Description: "Return session authentication readiness, broker availability, subaccount binding, and remediation issues.",
	}, func(_ context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, authStatusOutput, error) {
		return nil, buildAuthStatus(deps, tc), nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_system_health",
		Description: "Check MCP auth readiness plus REST and WebSocket client wiring in one call. Use this as the first smoke test after connecting.",
	}, func(ctx context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, systemHealthOutput, error) {
		return nil, systemHealthOutput{
			Meta: newResponseMeta(authModeFromContext(tc)),
			Auth: buildAuthStatus(deps, tc),
			REST: checkRESTHealth(ctx, deps),
			WS:   checkWSHealth(deps),
		}, nil
	})
}

func buildAuthStatus(deps *ToolDeps, tc ToolContext) authStatusOutput {
	out := authStatusOutput{
		Meta:     newResponseMeta(authModeFromContext(tc)),
		AuthMode: authModeFromContext(tc),
		Issues:   []string{},
	}
	if tc.State != nil && tc.State.AuthMode == session.AuthModeAuthenticated {
		out.Authenticated = true
		out.SessionSubAccountID = tc.State.SubAccountID
		out.SessionWallet = tc.State.WalletAddress
	}
	if deps != nil && deps.BrokerStatus != nil {
		status := deps.BrokerStatus.Status()
		out.BrokerEnabled = true
		out.BrokerSubAccountID = status.SubAccountID
		out.BrokerReady = status.WalletAddress != "" && status.SubAccountID > 0
		if out.Authenticated && status.SubAccountID > 0 && status.SubAccountID != out.SessionSubAccountID {
			out.Issues = append(out.Issues, "session subaccount does not match broker subaccount")
		}
	}
	if !out.Authenticated && !out.BrokerEnabled {
		out.Issues = append(out.Issues, "authenticate before using signed account or trading tools")
	}
	if out.BrokerEnabled && !out.BrokerReady {
		out.Issues = append(out.Issues, "broker is enabled but has not resolved a subaccount yet")
	}
	return out
}

func checkRESTHealth(ctx context.Context, deps *ToolDeps) healthCheckOutput {
	start := snx_lib_utils_time.Now()
	if deps == nil || deps.Clients == nil || deps.Clients.RESTInfo == nil {
		return healthCheckOutput{OK: false, Status: "not_configured"}
	}
	status, err := deps.Clients.RESTInfo.GetExchangeStatus(ctx)
	latency := snx_lib_utils_time.Since(start).Milliseconds()
	if err != nil {
		return healthCheckOutput{LatencyMs: latency, OK: false, Status: fmt.Sprintf("error: %v", err)}
	}
	if status == nil {
		return healthCheckOutput{LatencyMs: latency, OK: false, Status: "empty_response"}
	}
	return healthCheckOutput{LatencyMs: latency, OK: status.AcceptingOrders, Status: status.ExchangeStatus}
}

func checkWSHealth(deps *ToolDeps) healthCheckOutput {
	if deps == nil || deps.Clients == nil || deps.Clients.WSInfo == nil {
		return healthCheckOutput{OK: false, Status: "not_configured"}
	}
	return healthCheckOutput{OK: true, Status: "configured"}
}
