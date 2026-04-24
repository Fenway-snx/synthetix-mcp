package tools

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	"github.com/Fenway-snx/synthetix-mcp/internal/server/backend"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/metrics"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

// ToolDeps holds cross-cutting dependencies shared by all tool registrations.
// Service-specific clients (narrow interfaces for testability) remain as
// additional parameters on individual Register* functions.
type ToolDeps struct {
	Cfg            *config.Config
	Clients        *backend.Clients
	Limiter        ToolRateLimiter
	PublicSessions *PublicSessionTracker
	Store          session.Store
	Verifier       SessionAccessVerifier
	// Projection of agentbroker.Broker.Status() consumed by public
	// tools. Interface-typed to keep this package free of an
	// agentbroker import; nil when the broker is disabled.
	BrokerStatus BrokerStatusProvider
}

// Satisfied by *agentbroker.Broker via an adapter in server.go.
type BrokerStatusProvider interface {
	Status() BrokerStatusSnapshot
}

// 1:1 mirror of agentbroker.BrokerStatus. JSON tags must stay
// identical so the public get_server_info wire format is preserved
// if the indirection is later collapsed to a direct embed.
type BrokerStatusSnapshot struct {
	ChainID          int      `json:"chainId,omitempty"`
	DefaultPreset    string   `json:"defaultPreset,omitempty"`
	DelegationID     uint64   `json:"delegationId,omitempty"`
	DomainName       string   `json:"domainName,omitempty"`
	DomainVersion    string   `json:"domainVersion,omitempty"`
	ExpiresAtUnix    int64    `json:"expiresAtUnix,omitempty"`
	OwnerAddress     string   `json:"ownerAddress,omitempty"`
	Permissions      []string `json:"permissions,omitempty"`
	SubAccountID     int64    `json:"subAccountId,omitempty"`
	SubaccountSource string   `json:"subaccountSource,omitempty"`
	WalletAddress    string   `json:"walletAddress,omitempty"`
}

// ToolContext carries the resolved session identity into a tool handler,
// removing the need for each handler to repeat session lookup boilerplate.
type ToolContext struct {
	SessionID string
	State     *session.State
}

// recordToolOutcome emits the standard (tool_calls_total, tool_call_duration)
// pair for a single tool invocation. Consolidates the ~20 metric-emission
// call sites that previously lived inline in each middleware function and in
// subscribe/unsubscribe, which otherwise tended to drift (e.g. subscribe
// was missing metrics entirely until now).
func recordToolOutcome(toolName string, outcome string, start time.Time) {
	metrics.ToolCallsTotal(toolName, outcome).Inc()
	metrics.ToolCallDuration(toolName).Observe(snx_lib_utils_time.Since(start).Seconds())
}

// outcomeForResult maps a (result, handlerErr) pair returned by a tool
// handler to the metric label used for tool_calls_total. handlerErr takes
// precedence because it indicates a transport-level failure, not a
// business-logic error the handler explicitly surfaced via result.
func outcomeForResult(result *mcp.CallToolResult, handlerErr error) string {
	if handlerErr != nil {
		return "error"
	}
	if result != nil && result.IsError {
		return "error"
	}
	return "ok"
}

// addAuthenticatedTool registers a tool that requires an authenticated
// session. The middleware sequence is:
//
//	require authenticated session → rate limit → touch session TTL → handler
//
// subAccountID extracts an optional subaccount ID from the input for
// cross-checking against the session-bound account. Pass noSubAccount
// when the input does not carry a subaccount field.
func addAuthenticatedTool[In, Out any](
	server *mcp.Server,
	deps *ToolDeps,
	tool *mcp.Tool,
	subAccountID func(In) *int64,
	handler func(ctx context.Context, tc ToolContext, input In) (*mcp.CallToolResult, Out, error),
) {
	applyToolSchemas[In, Out](tool)
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		start := snx_lib_utils_time.Now()
		sessionID := sessionIDFromRequest(req)
		state, err := requireAuthenticatedSession(ctx, deps.Store, deps.Verifier, sessionID, subAccountID(input))
		if err != nil {
			recordToolOutcome(tool.Name, "auth_failed", start)
			return toolErrorResponse[Out](err)
		}
		if err := maybeRateLimitTool(ctx, deps.Limiter, state, tool.Name); err != nil {
			recordToolOutcome(tool.Name, "rate_limited", start)
			return toolErrorResponse[Out](err)
		}
		if err := touchSession(ctx, deps.Store, sessionID, deps.Cfg.SessionTTL); err != nil {
			recordToolOutcome(tool.Name, "error", start)
			return toolErrorResponse[Out](err)
		}
		// Mirror the session-store Touch mutation on the local copy so
		// handlers see the post-touch ExpiresAt / LastActivityAt without a
		// second Get round trip.
		session.ApplyTouchTimes(state, deps.Cfg.SessionTTL, snx_lib_utils_time.Now())
		result, out, handlerErr := handler(ctx, ToolContext{SessionID: sessionID, State: state}, input)
		recordToolOutcome(tool.Name, outcomeForResult(result, handlerErr), start)
		return result, out, handlerErr
	})
}

// addPublicTool registers a tool that works with or without an authenticated
// session. The middleware sequence is:
//
//	load session (ignore not-found) → sanitize → rate limit → touch → handler
func addPublicTool[In, Out any](
	server *mcp.Server,
	deps *ToolDeps,
	tool *mcp.Tool,
	handler func(ctx context.Context, tc ToolContext, input In) (*mcp.CallToolResult, Out, error),
) {
	applyToolSchemas[In, Out](tool)
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		start := snx_lib_utils_time.Now()
		sessionID := sessionIDFromRequest(req)
		state, err := loadSessionState(ctx, deps.Store, sessionID)
		if err != nil && !errors.Is(err, session.ErrSessionNotFound) {
			recordToolOutcome(tool.Name, "error", start)
			return toolErrorResponse[Out](err)
		}
		state, err = sanitizeSessionState(ctx, deps.Store, sessionID, state, deps.Verifier)
		if err != nil {
			recordToolOutcome(tool.Name, "error", start)
			return toolErrorResponse[Out](err)
		}
		if err := maybeRateLimitTool(ctx, deps.Limiter, state, tool.Name); err != nil {
			recordToolOutcome(tool.Name, "rate_limited", start)
			return toolErrorResponse[Out](err)
		}
		if err := touchSession(ctx, deps.Store, sessionID, deps.Cfg.SessionTTL); err != nil {
			recordToolOutcome(tool.Name, "error", start)
			return toolErrorResponse[Out](err)
		}
		// See addAuthenticatedTool.
		session.ApplyTouchTimes(state, deps.Cfg.SessionTTL, snx_lib_utils_time.Now())
		// Record first-seen for genuinely public sessions so get_session
		// can report a non-zero createdAt. Authenticated sessions track
		// CreatedAt in their persisted state and don't need this.
		if state == nil {
			deps.PublicSessions.Observe(sessionID)
		}
		result, out, handlerErr := handler(ctx, ToolContext{SessionID: sessionID, State: state}, input)
		recordToolOutcome(tool.Name, outcomeForResult(result, handlerErr), start)
		return result, out, handlerErr
	})
}

// addRateLimitedTool registers a tool that needs only IP-based rate limiting
// with no session state (e.g. ping, get_server_info).
func addRateLimitedTool[In, Out any](
	server *mcp.Server,
	deps *ToolDeps,
	tool *mcp.Tool,
	handler func(ctx context.Context, input In) (*mcp.CallToolResult, Out, error),
) {
	applyToolSchemas[In, Out](tool)
	mcp.AddTool(server, tool, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		start := snx_lib_utils_time.Now()
		if err := maybeRateLimitTool(ctx, deps.Limiter, nil, tool.Name); err != nil {
			recordToolOutcome(tool.Name, "rate_limited", start)
			return toolErrorResponse[Out](err)
		}
		result, out, handlerErr := handler(ctx, input)
		recordToolOutcome(tool.Name, outcomeForResult(result, handlerErr), start)
		return result, out, handlerErr
	})
}

func noSubAccount[In any](_ In) *int64 { return nil }

// applyToolSchemas pre-populates the tool's input and output schemas via
// reflection and rewrites any properties named like 64-bit Synthetix IDs
// (subAccountId, venueId, positionId, ...) from "integer" to digit-string.
//
// We have to set these here (rather than letting mcp.AddTool derive them
// implicitly) because the go-sdk's auto-reflection always emits integer
// types for Go int64/uint64 fields. Since we marshal those fields with
// the ",string" encoding/json option to preserve precision for JS clients,
// the server-side output validator would otherwise reject every response.
//
// If schema generation fails (e.g. because the type is unsupported by
// jsonschema-go), we leave the schema unset and let mcp.AddTool surface
// the reflection error the same way it does today. This keeps the error
// path identical to the pre-existing behavior for types we can't handle.
//
// An empty struct{} input produces an "object" schema with no properties,
// which matches what mcp.AddTool would generate itself.
func applyToolSchemas[In, Out any](tool *mcp.Tool) {
	if tool.InputSchema == nil {
		if schema, err := schemaForInput[In](); err == nil {
			tool.InputSchema = schema
		}
	}
	if tool.OutputSchema == nil {
		if schema, err := schemaForOutput[Out](); err == nil {
			tool.OutputSchema = schema
		}
	}
}
