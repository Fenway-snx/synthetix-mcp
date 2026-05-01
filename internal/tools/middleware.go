package tools

import (
	"context"
	"errors"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/server/backend"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

// Holds cross-cutting dependencies shared by tool registrations.
// Narrow service clients stay as parameters on each registration function.
type ToolDeps struct {
	Cfg            *config.Config
	Clients        *backend.Clients
	PublicSessions *PublicSessionTracker
	Store          session.Store
	Verifier       SessionAccessVerifier
	// Broker posture consumed by public tools; nil when disabled.
	BrokerStatus BrokerStatusProvider
	// Optional live-order snapshots used by public "my orders" overlays.
	SnapshotManager *risksnapshot.Manager
}

// Implemented by the broker adapter in server wiring.
type BrokerStatusProvider interface {
	Status() BrokerStatusSnapshot
}

// Mirrors broker status while preserving the public wire format.
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

// Carries resolved session identity into a tool handler.
type ToolContext struct {
	SessionID string
	State     *session.State
}

// Registers a tool that requires an authenticated session.
// Optional subaccount extraction cross-checks the session-bound account.
func addAuthenticatedTool[In, Out any](
	server *mcp.Server,
	deps *ToolDeps,
	tool *mcp.Tool,
	subAccountID func(In) *int64,
	handler func(ctx context.Context, tc ToolContext, input In) (*mcp.CallToolResult, Out, error),
) {
	applyToolSchemas[In, Out](tool)
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		sessionID := sessionIDFromRequest(req)
		state, err := requireAuthenticatedSession(ctx, deps.Store, deps.Verifier, sessionID, subAccountID(input))
		if err != nil {
			return toolErrorResponse[Out](err)
		}
		if err := touchSession(ctx, deps.Store, sessionID, deps.Cfg.SessionTTL); err != nil {
			return toolErrorResponse[Out](err)
		}
		// Mirror the session-store Touch mutation on the local copy so
		// handlers see the post-touch ExpiresAt / LastActivityAt without a
		// second Get round trip.
		session.ApplyTouchTimes(state, deps.Cfg.SessionTTL, snx_lib_utils_time.Now())
		return handler(ctx, ToolContext{SessionID: sessionID, State: state}, input)
	})
}

// Registers a tool that works with or without an authenticated session.
func addPublicTool[In, Out any](
	server *mcp.Server,
	deps *ToolDeps,
	tool *mcp.Tool,
	handler func(ctx context.Context, tc ToolContext, input In) (*mcp.CallToolResult, Out, error),
) {
	applyToolSchemas[In, Out](tool)
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		sessionID := sessionIDFromRequest(req)
		state, err := loadSessionState(ctx, deps.Store, sessionID)
		if err != nil && !errors.Is(err, session.ErrSessionNotFound) {
			return toolErrorResponse[Out](err)
		}
		state, err = sanitizeSessionState(ctx, deps.Store, sessionID, state, deps.Verifier)
		if err != nil {
			return toolErrorResponse[Out](err)
		}
		if err := touchSession(ctx, deps.Store, sessionID, deps.Cfg.SessionTTL); err != nil {
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
		return handler(ctx, ToolContext{SessionID: sessionID, State: state}, input)
	})
}

// Registers a tool that needs no session state.
func addUnauthenticatedTool[In, Out any](
	server *mcp.Server,
	deps *ToolDeps,
	tool *mcp.Tool,
	handler func(ctx context.Context, input In) (*mcp.CallToolResult, Out, error),
) {
	applyToolSchemas[In, Out](tool)
	mcp.AddTool(server, tool, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		return handler(ctx, input)
	})
}

func noSubAccount[In any](_ In) *int64 { return nil }

// Pre-populates schemas and rewrites 64-bit ID fields to digit strings.
// This keeps JSON-string encoded IDs valid for JavaScript clients.
// Unsupported reflection cases fall through to the SDK's normal error path.
func applyToolSchemas[In, Out any](tool *mcp.Tool) {
	if tool.InputSchema == nil {
		if schema, err := schemaForInput[In](); err == nil {
			tool.InputSchema = schema
		}
	}
	if tool.OutputSchema == nil {
		tool.OutputSchema = permissiveObjectOutputSchema()
	}
}

func permissiveObjectOutputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
	}
}
