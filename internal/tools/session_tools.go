package tools

import (
	"context"
	"fmt"

	internal_auth "github.com/Fenway-snx/synthetix-mcp/internal/auth"
	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SessionAuthenticator interface {
	Authenticate(ctx context.Context, sessionID string, message string, signatureHex string) (*internal_auth.AuthenticateResult, error)
}

type SessionSubscriptionResetter interface {
	ClearPrivateSubscriptions(sessionID string)
}

type authenticateInput struct {
	Message      string `json:"message" jsonschema:"EIP-712 typed-data JSON containing domain, types, primaryType='AuthMessage', and message fields. Call preview_auth_message first to get the exact typed-data object the server expects; sign it with any EIP-712-capable signer (viem.signTypedData, eth_signTypedData_v4, ethers Wallet._signTypedData, Web3.py sign_typed_data) and pass the serialized JSON back verbatim."`
	SignatureHex string `json:"signatureHex" jsonschema:"0x-prefixed hex-encoded 65-byte EIP-712 signature produced by the wallet that owns or has delegation for the target subaccount. The signed payload must equal the message argument above."`
}

type authenticateOutput struct {
	Meta             responseMeta `json:"_meta"`
	AgentGuardrails  *guardrailsOutput `json:"agentGuardrails,omitempty"`
	Authenticated    bool         `json:"authenticated"`
	NextSteps        []string     `json:"nextSteps"`
	SessionExpiresAt int64        `json:"sessionExpiresAt"`
	SubAccountID     int64        `json:"subAccountId,string"`
	SessionID        string       `json:"sessionId"`
	WalletAddress    string       `json:"walletAddress"`
}

type getSessionOutput struct {
	Meta                responseMeta      `json:"_meta"`
	ActiveSubscriptions []string          `json:"activeSubscriptions"`
	AgentGuardrails     *guardrailsOutput `json:"agentGuardrails,omitempty"`
	AuthMode            string            `json:"authMode"`
	CreatedAt           int64             `json:"createdAt"`
	ExpiresAt           int64             `json:"expiresAt"`
	LastActivityAt      int64             `json:"lastActivityAt"`
	SessionID           string            `json:"sessionId"`
	SubAccountID        int64             `json:"subAccountId,omitempty,string"`
	WalletAddress       string            `json:"walletAddress,omitempty"`
}

type restoreSessionInput struct {
	SessionID string `json:"sessionId" jsonschema:"The current Mcp-Session-Id value. Must match the active session; cross-session restore is not supported."`
}

type restoreSessionOutput struct {
	Meta                responseMeta      `json:"_meta"`
	ActiveSubscriptions []string          `json:"activeSubscriptions"`
	AgentGuardrails     *guardrailsOutput `json:"agentGuardrails,omitempty"`
	AuthMode            string            `json:"authMode"`
	Restored            bool              `json:"restored"`
	SessionExpiresAt    int64             `json:"sessionExpiresAt"`
	SessionID           string            `json:"sessionId"`
	SubAccountID        int64             `json:"subAccountId,omitempty,string"`
	WalletAddress       string            `json:"walletAddress,omitempty"`
}

type guardrailsOutput struct {
	AllowedOrderTypes   []string `json:"allowedOrderTypes"`
	AllowedSymbols      []string `json:"allowedSymbols"`
	EffectivePreset     string   `json:"effectivePreset"`
	MaxOrderQuantity    string   `json:"maxOrderQuantity,omitempty"`
	MaxPositionQuantity string   `json:"maxPositionQuantity,omitempty"`
	RequestedPreset     string   `json:"requestedPreset"`
	WriteEnabled        bool     `json:"writeEnabled"`
}

type setGuardrailsInput struct {
	AllowedOrderTypes   []string `json:"allowedOrderTypes,omitempty"`
	AllowedSymbols      []string `json:"allowedSymbols,omitempty"`
	MaxOrderQuantity    string   `json:"maxOrderQuantity,omitempty"`
	MaxPositionQuantity string   `json:"maxPositionQuantity,omitempty"`
	Preset              string   `json:"preset"`
}

type setGuardrailsOutput struct {
	Meta            responseMeta      `json:"_meta"`
	AgentGuardrails *guardrailsOutput `json:"agentGuardrails"`
	SessionID       string            `json:"sessionId"`
}

type SessionSubscriptionReader interface {
	ActiveChannels(sessionID string) []string
}

func RegisterSessionTools(
	server *mcp.Server,
	deps *ToolDeps,
	authenticator SessionAuthenticator,
	subscriptions SessionSubscriptionResetter,
) {
	addPublicTool(server, deps, &mcp.Tool{
		Name:        "authenticate",
		Description: "Bind this MCP session to a delegated Synthetix trading subaccount using an EIP-712 signed message. After authentication, account tools, trading tools, and private streaming channels become available. The session remains authenticated until TTL expiry or delegation revocation. AGENT POLICY: only call this when you (the agent) hold the signing key locally. If get_server_info.agentBroker.enabled=true, call quick_place_order / quick_close_position / etc. instead — the broker authenticates internally. Never instruct a human user to paste an EIP-712 signature into chat.",
	}, func(ctx context.Context, tc ToolContext, input authenticateInput) (*mcp.CallToolResult, authenticateOutput, error) {
		sessionID := tc.SessionID

		result, err := authenticator.Authenticate(ctx, sessionID, input.Message, input.SignatureHex)
		if err != nil {
			return toolErrorResponse[authenticateOutput](err)
		}
		if subscriptions != nil {
			subscriptions.ClearPrivateSubscriptions(sessionID)
		}
		// Authenticated sessions carry their own CreatedAt in session.State,
		// so drop the public first-seen entry to avoid unbounded growth.
		deps.PublicSessions.Forget(sessionID)

		// Re-fetch the now-authenticated session so we can surface the
		// resolved guardrails (which default to read_only) and tell the
		// agent the next call it most likely needs to make. The original
		// transcript demonstrated agents repeatedly trying place_order
		// against the read_only fallback because authenticate's response
		// gave no indication the session was write-disabled.
		refreshed, _ := loadSessionState(ctx, deps.Store, sessionID)
		out := authenticateOutput{
			Meta:             newResponseMeta(string(session.AuthModeAuthenticated)),
			Authenticated:    result.Authenticated,
			NextSteps:        []string{},
			SessionExpiresAt: result.SessionExpiresAt,
			SessionID:        sessionID,
			SubAccountID:     result.SubAccountID,
			WalletAddress:    result.WalletAddress,
		}
		out.AgentGuardrails = guardrailsOutputForState(refreshed)
		out.NextSteps = nextStepsForAuthenticatedSession(refreshed)
		return nil, out, nil
	})
}

func RegisterSessionStateTools(
	server *mcp.Server,
	deps *ToolDeps,
	subscriptions SessionSubscriptionReader,
) {
	addPublicTool(server, deps, &mcp.Tool{
		Name:        "get_session",
		Description: "Return the current MCP session state: authentication mode, bound wallet/subaccount, session timestamps (created, expires, last activity), and active streaming subscriptions. Use this to verify session health or debug auth issues.",
	}, func(_ context.Context, tc ToolContext, _ struct{}) (*mcp.CallToolResult, getSessionOutput, error) {
		sessionID := tc.SessionID
		state := tc.State

		output := getSessionOutput{
			Meta:                newResponseMeta(authModeForState(state)),
			ActiveSubscriptions: activeSubscriptions(subscriptions, sessionID),
			AuthMode:            authModeForState(state),
			SessionID:           sessionID,
		}
		if state != nil {
			output.AgentGuardrails = guardrailsOutputForState(state)
			output.CreatedAt = state.CreatedAt
			output.ExpiresAt = state.ExpiresAt
			output.LastActivityAt = state.LastActivityAt
			output.SubAccountID = state.SubAccountID
			output.WalletAddress = state.WalletAddress
		} else {
			// Public (unauthenticated) sessions have no persisted state,
			// so surface the in-memory first-seen timestamp so callers
			// can reason about session age. ExpiresAt/LastActivityAt
			// deliberately remain zero for public sessions: they have
			// no TTL and the MCP SDK manages their lifecycle.
			output.CreatedAt = deps.PublicSessions.Observe(sessionID)
		}

		return nil, output, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "set_guardrails",
		Description: "Set per-session agent safety guardrails for trading tools. Unknown presets fail closed to read_only.",
	}, noSubAccount[setGuardrailsInput], func(ctx context.Context, tc ToolContext, input setGuardrailsInput) (*mcp.CallToolResult, setGuardrailsOutput, error) {
		sessionID := tc.SessionID
		state := tc.State
		state.AgentGuardrails = &guardrails.Config{
			AllowedOrderTypes:   append([]string{}, input.AllowedOrderTypes...),
			AllowedSymbols:      append([]string{}, input.AllowedSymbols...),
			MaxOrderQuantity:    input.MaxOrderQuantity,
			MaxPositionQuantity: input.MaxPositionQuantity,
			Preset:              input.Preset,
		}
		if _, err := guardrails.Resolve(state.AgentGuardrails); err != nil {
			return newToolErrorResult(
				"INVALID_ARGUMENT",
				err.Error(),
				"Retry set_guardrails with a known preset and valid symbol/order quantity constraints.",
			), setGuardrailsOutput{}, nil
		}
		if err := deps.Store.Save(ctx, sessionID, state, deps.Cfg.SessionTTL); err != nil {
			return toolErrorResponse[setGuardrailsOutput](fmt.Errorf("save guardrails: %w", err))
		}

		refreshed, err := loadSessionState(ctx, deps.Store, sessionID)
		if err != nil {
			return toolErrorResponse[setGuardrailsOutput](err)
		}
		if refreshed == nil {
			return newToolErrorResult(
				"AUTH_REQUIRED",
				"The MCP session is no longer available.",
				"Call authenticate to create a fresh authenticated session before setting guardrails.",
			), setGuardrailsOutput{}, nil
		}

		return nil, setGuardrailsOutput{
			Meta:            newResponseMeta(authModeForState(refreshed)),
			AgentGuardrails: guardrailsOutputForState(refreshed),
			SessionID:       sessionID,
		}, nil
	})

	addPublicTool(server, deps, &mcp.Tool{
		Name:        "restore_session",
		Description: "Extend the TTL of the current authenticated MCP session. Only works for the current Mcp-Session-Id. If the session has expired or was revoked, call authenticate instead to create a new session.",
	}, func(_ context.Context, tc ToolContext, input restoreSessionInput) (*mcp.CallToolResult, restoreSessionOutput, error) {
		currentSessionID := tc.SessionID
		if currentSessionID == "" || input.SessionID == "" {
			return toolErrorResponse[restoreSessionOutput](session.ErrSessionNotFound)
		}
		if input.SessionID != currentSessionID {
			return newToolErrorResult(
				"AUTH_REQUIRED",
				"restore_session may only restore the current MCP session context.",
				"Retry restore_session using the current Mcp-Session-Id value.",
				"Call authenticate if the current MCP session is no longer authenticated.",
			), restoreSessionOutput{}, nil
		}
		if tc.State == nil {
			return newToolErrorResult(
				"AUTH_REQUIRED",
				"There is no stored authenticated session state to restore for this MCP session.",
				"Call authenticate to create a fresh authenticated session.",
			), restoreSessionOutput{}, nil
		}

		return nil, restoreSessionOutput{
			Meta:                newResponseMeta(authModeForState(tc.State)),
			ActiveSubscriptions: activeSubscriptions(subscriptions, currentSessionID),
			AgentGuardrails:     guardrailsOutputForState(tc.State),
			AuthMode:            authModeForState(tc.State),
			Restored:            true,
			SessionExpiresAt:    tc.State.ExpiresAt,
			SessionID:           currentSessionID,
			SubAccountID:        tc.State.SubAccountID,
			WalletAddress:       tc.State.WalletAddress,
		}, nil
	})
}

func activeSubscriptions(reader SessionSubscriptionReader, sessionID string) []string {
	if reader == nil || sessionID == "" {
		return []string{}
	}
	channels := reader.ActiveChannels(sessionID)
	if channels == nil {
		return []string{}
	}
	return channels
}

// Lists immediate next actions after authenticate. Surfaces the
// set_guardrails step inline so the agent doesn't loop on
// GUARDRAIL_VIOLATION from the default read_only fallback.
func nextStepsForAuthenticatedSession(state *session.State) []string {
	steps := []string{
		"Call get_session to confirm walletAddress, subAccountId, expiresAt, and active guardrails.",
	}
	if state == nil || state.AgentGuardrails == nil {
		steps = append(steps,
			"Call set_guardrails with preset='standard' (or 'read_only' for view-only sessions) before any trading tool. Sessions default to read_only and will reject trades with GUARDRAIL_VIOLATION until set_guardrails has been called.",
		)
		return steps
	}
	resolved, err := guardrails.Resolve(state.AgentGuardrails)
	if err != nil || resolved == nil || !resolved.WriteEnabled() {
		steps = append(steps,
			"Call set_guardrails with preset='standard' to enable trading tools. The current session is in read_only mode and will reject orders with GUARDRAIL_VIOLATION.",
		)
		return steps
	}
	steps = append(steps,
		"Use preview_trade_signature → sign locally → place_order for trading (only if you, the agent, hold the signing key).",
		"If get_server_info.agentBroker.enabled=true, prefer quick_place_order / quick_close_position / quick_cancel_order / quick_cancel_all_orders — they sign and submit in one round trip.",
		"Never ask the human user to paste an EIP-712 signature, hex digest, or private key into chat.",
	)
	return steps
}

func guardrailsOutputForState(state *session.State) *guardrailsOutput {
	if state == nil || state.AgentGuardrails == nil {
		return nil
	}

	resolved, err := guardrails.Resolve(state.AgentGuardrails)
	if err != nil {
		return &guardrailsOutput{
			AllowedOrderTypes: []string{},
			AllowedSymbols:    []string{},
			EffectivePreset:   guardrails.PresetReadOnly,
			RequestedPreset:   state.AgentGuardrails.Preset,
			WriteEnabled:      false,
		}
	}

	out := &guardrailsOutput{
		AllowedOrderTypes: append([]string{}, resolved.AllowedOrderTypes...),
		AllowedSymbols:    append([]string{}, resolved.AllowedSymbols...),
		EffectivePreset:   resolved.EffectivePreset,
		RequestedPreset:   resolved.RequestedPreset,
		WriteEnabled:      resolved.WriteEnabled(),
	}
	if resolved.HasMaxOrderQuantity() {
		out.MaxOrderQuantity = resolved.MaxOrderQuantity.String()
	}
	if resolved.HasMaxPositionQuantity() {
		out.MaxPositionQuantity = resolved.MaxPositionQuantity.String()
	}
	return out
}
