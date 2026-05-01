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
	Message      string `json:"message" jsonschema:"EIP-712 typed-data JSON containing domain, types, primaryType='AuthMessage', and message fields. A local sidecar signer should call preview_auth_message first, sign the typed data outside chat, and pass the serialized JSON back verbatim. Claude cannot sign this itself."`
	SignatureHex string `json:"signatureHex" jsonschema:"0x-prefixed hex-encoded 65-byte EIP-712 signature produced outside chat by the wallet that owns or has delegation for the target subaccount. Never ask a human to paste this into chat; use sample/node-scripts/authenticate-external-wallet.mjs or another local sidecar."`
}

type authenticateOutput struct {
	Meta             responseMeta      `json:"_meta"`
	AgentGuardrails  *guardrailsOutput `json:"agentGuardrails,omitempty"`
	Authenticated    bool              `json:"authenticated"`
	NextSteps        []string          `json:"nextSteps"`
	SessionExpiresAt int64             `json:"sessionExpiresAt"`
	SubAccountID     int64             `json:"subAccountId,string"`
	SessionID        string            `json:"sessionId"`
	WalletAddress    string            `json:"walletAddress"`
}

type getSessionOutput struct {
	Meta                    responseMeta      `json:"_meta"`
	ActiveSubscriptions     []string          `json:"activeSubscriptions"`
	AgentGuardrails         *guardrailsOutput `json:"agentGuardrails,omitempty"`
	AuthMode                string            `json:"authMode"`
	BrokerDefaultGuardrails *guardrailsOutput `json:"brokerDefaultGuardrails,omitempty"`
	CreatedAt               int64             `json:"createdAt"`
	ExpiresAt               int64             `json:"expiresAt"`
	LastActivityAt          int64             `json:"lastActivityAt"`
	SessionID               string            `json:"sessionId"`
	SubAccountID            int64             `json:"subAccountId,omitempty,string"`
	WalletAddress           string            `json:"walletAddress,omitempty"`
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
	MaxOrderNotional    string   `json:"maxOrderNotional,omitempty"`
	MaxOrderQuantity    string   `json:"maxOrderQuantity,omitempty"`
	MaxPositionNotional string   `json:"maxPositionNotional,omitempty"`
	MaxPositionQuantity string   `json:"maxPositionQuantity,omitempty"`
	RequestedPreset     string   `json:"requestedPreset"`
	WriteEnabled        bool     `json:"writeEnabled"`
}

type setGuardrailsInput struct {
	AllowedOrderTypes   []string `json:"allowedOrderTypes,omitempty"`
	AllowedSymbols      []string `json:"allowedSymbols,omitempty"`
	MaxOrderNotional    string   `json:"maxOrderNotional,omitempty"`
	MaxOrderQuantity    string   `json:"maxOrderQuantity,omitempty"`
	MaxPositionNotional string   `json:"maxPositionNotional,omitempty"`
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
		Description: "Bind this MCP session to a delegated Synthetix trading subaccount using an EIP-712 signed message. After authentication, account tools, signed_* wallet tools, and private streaming channels become available. The session remains authenticated until TTL expiry or delegation revocation. AGENT POLICY: only call this when you (the agent) hold the signing key locally. If get_server_info.agentBroker.enabled=true, call canonical broker tools like place_order / close_position instead — the broker authenticates internally. Never instruct a human user to paste an EIP-712 signature into chat.",
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

		// Re-fetch the now-authenticated session so we can surface any
		// session-specific guardrails and tell the agent the next call it
		// most likely needs to make.
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
		if deps != nil && deps.Cfg != nil && deps.Cfg.AgentBroker.Enabled {
			output.BrokerDefaultGuardrails = guardrailsOutputForConfig(brokerDefaultGuardrailsConfig(deps))
		}
		if state != nil {
			output.AgentGuardrails = guardrailsOutputForState(state)
			output.CreatedAt = state.CreatedAt
			output.ExpiresAt = state.ExpiresAt
			output.LastActivityAt = state.LastActivityAt
			output.SubAccountID = state.SubAccountID
			output.WalletAddress = state.WalletAddress
		} else {
			// Public sessions expose first-seen time but have no TTL state.
			output.CreatedAt = deps.PublicSessions.Observe(sessionID)
		}

		return nil, output, nil
	})

	addAuthenticatedTool(server, deps, &mcp.Tool{
		Name:        "set_guardrails",
		Description: "Optionally set per-session agent safety guardrails for trading tools. Unknown presets fail closed to read_only.",
	}, noSubAccount[setGuardrailsInput], func(ctx context.Context, tc ToolContext, input setGuardrailsInput) (*mcp.CallToolResult, setGuardrailsOutput, error) {
		sessionID := tc.SessionID
		state := tc.State
		state.AgentGuardrails = &guardrails.Config{
			AllowedOrderTypes:   append([]string{}, input.AllowedOrderTypes...),
			AllowedSymbols:      append([]string{}, input.AllowedSymbols...),
			MaxOrderNotional:    input.MaxOrderNotional,
			MaxOrderQuantity:    input.MaxOrderQuantity,
			MaxPositionNotional: input.MaxPositionNotional,
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

// Lists immediate next actions after authenticate. Guardrails are optional;
// this helper only recommends set_guardrails when the session was explicitly
// placed in read_only or the operator wants tighter per-session limits.
func nextStepsForAuthenticatedSession(state *session.State) []string {
	steps := []string{
		"Call get_session to confirm walletAddress, subAccountId, expiresAt, and active guardrails.",
	}
	if state == nil || state.AgentGuardrails == nil {
		steps = append(steps,
			"No session-specific guardrails are set. Trading is still allowed by default; call set_guardrails only if the operator wants tighter per-session limits or read_only mode.",
		)
		return steps
	}
	resolved, err := guardrails.Resolve(state.AgentGuardrails)
	if err != nil || resolved == nil || !resolved.WriteEnabled() {
		steps = append(steps,
			"Call set_guardrails with preset='standard' to re-enable trading tools, or keep read_only for view-only sessions.",
		)
		return steps
	}
	steps = append(steps,
		"Include active guardrails in the single trade confirmation if the user has not already approved the operation.",
		"Use preview_trade_signature → sign locally → signed_place_order for trading (only if you, the agent, hold the signing key).",
		"If get_server_info.agentBroker.enabled=true, prefer place_order / close_position / cancel_order / cancel_all_orders — they sign and submit in one round trip.",
		"Never ask the human user to paste an EIP-712 signature, hex digest, or private key into chat.",
	)
	return steps
}

func guardrailsOutputForState(state *session.State) *guardrailsOutput {
	if state == nil || state.AgentGuardrails == nil {
		return nil
	}

	return guardrailsOutputForConfig(state.AgentGuardrails)
}

func guardrailsOutputForConfig(cfg *guardrails.Config) *guardrailsOutput {
	if cfg == nil {
		return nil
	}

	resolved, err := guardrails.Resolve(cfg)
	if err != nil {
		return &guardrailsOutput{
			AllowedOrderTypes: []string{},
			AllowedSymbols:    []string{},
			EffectivePreset:   guardrails.PresetReadOnly,
			RequestedPreset:   cfg.Preset,
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
	if resolved.HasMaxOrderNotional() {
		out.MaxOrderNotional = resolved.MaxOrderNotional.String()
	}
	if resolved.HasMaxPositionQuantity() {
		out.MaxPositionQuantity = resolved.MaxPositionQuantity.String()
	}
	if resolved.HasMaxPositionNotional() {
		out.MaxPositionNotional = resolved.MaxPositionNotional.String()
	}
	return out
}

func brokerDefaultGuardrailsConfig(deps *ToolDeps) *guardrails.Config {
	if deps == nil || deps.Cfg == nil {
		return nil
	}
	defaults := deps.Cfg.AgentBroker
	return &guardrails.Config{
		AllowedOrderTypes:   append([]string{}, defaults.DefaultAllowedTypes...),
		AllowedSymbols:      append([]string{}, defaults.DefaultAllowedSymbols...),
		MaxOrderNotional:    defaults.DefaultMaxOrderNotional,
		MaxOrderQuantity:    defaults.DefaultMaxOrderQty,
		MaxPositionNotional: defaults.DefaultMaxPositionNotional,
		MaxPositionQuantity: defaults.DefaultMaxPositionQty,
		Preset:              defaults.DefaultPreset,
	}
}
