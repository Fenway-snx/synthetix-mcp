package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"

	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type responseMeta struct {
	AuthMode   string `json:"authMode"`
	ServerTime int64  `json:"serverTime"`
}

type SessionAccessVerifier interface {
	VerifySessionAccess(ctx context.Context, walletAddress string, subAccountID int64) error
}

func newResponseMeta(authMode string) responseMeta {
	return responseMeta{
		AuthMode:   authMode,
		ServerTime: snx_lib_utils_time.Now().UnixMilli(),
	}
}

type toolErrorBody struct {
	Details map[string]any `json:"details,omitempty"`
	Error   string         `json:"error"`
	Message string         `json:"message"`
}

func newToolErrorResult(code string, message string, remediation ...string) *mcp.CallToolResult {
	body := toolErrorBody{
		Error:   code,
		Message: message,
	}
	if len(remediation) > 0 {
		body.Details = map[string]any{
			"remediation": remediation,
		}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		payload = []byte(fmt.Sprintf(`{"error":"%s","message":"%s"}`, code, message))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(payload)},
		},
		IsError:           true,
		StructuredContent: json.RawMessage(payload),
	}
}

var (
	authErrorPhrases = []string{
		"authentication required",
		"session id is required",
		"session expired",
		"session not found",
		"unauthenticated",
	}
	invalidSignaturePhrases = []string{
		"invalid session authentication",
		"invalid trade action signature",
		"invalid signature",
		"signature hex",
		"signature malformed",
		"parse auth message",
		"validate trade action signature",
	}
	permissionErrorPhrases = []string{
		"does not match authenticated session",
		"not authorized",
		"permission denied",
		"forbidden",
		"delegation",
	}
	validationErrorPhrases = []string{
		"invalid ",
		"is required",
		"are required",
		" requires ",
		"must be ",
		"must align",
		"must have",
		"do not accept",
		"does not accept",
		"parse ",
		"exceeds",
		"out of range",
		"too large",
		"too small",
		"cannot be zero",
		"cannot be negative",
		"cannot close",
		"already exists",
		"duplicate",
		"malformed",
		"unsupported",
	}
	notFoundErrorPhrases = []string{
		"not found",
		"does not exist",
		"unknown symbol",
		"unknown market",
		"no such",
	}
	notImplementedErrorPhrases = []string{
		"not implemented",
		"not yet available",
		"not supported",
		"stubbed",
		"coming soon",
	}
	rateLimitErrorPhrases = []string{
		"rate limit",
		"rate-limit",
		"throttled",
		"too many requests",
	}
	timeoutErrorPhrases = []string{
		"deadline exceeded",
		"context deadline",
		"timed out",
		"timeout",
		"context canceled",
	}
)

func containsAny(lower string, phrases []string) bool {
	for _, p := range phrases {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func classifyToolError(err error) (string, string, []string) {
	if err == nil {
		return "UNKNOWN", "An unexpected error occurred.", nil
	}

	if errors.Is(err, session.ErrSessionNotFound) {
		return "AUTH_REQUIRED", "A valid authenticated MCP session is required.", []string{
			"Call authenticate to create a fresh authenticated session.",
			"Retry the tool call with the current Mcp-Session-Id header.",
		}
	}

	lower := strings.ToLower(err.Error())

	switch {
	case containsAny(lower, []string{"guardrail violation"}):
		return "GUARDRAIL_VIOLATION", "The current session guardrails do not permit this action.", []string{
			"Inspect get_session.agentGuardrails for the active preset and limits before retrying.",
			"If preset is read_only, call set_guardrails with preset='standard' to re-enable trading or keep the session view-only.",
			"If a configured limit was hit, reduce the order or ask the operator to update the guardrail configuration.",
		}
	case containsAny(lower, notImplementedErrorPhrases):
		return "NOT_IMPLEMENTED", "This tool is not implemented for the current phase.", []string{
			"Use the documented fallback path or retry after the feature is implemented.",
		}
	case containsAny(lower, invalidSignaturePhrases):
		return "INVALID_SIGNATURE", "The supplied EIP-712 payload or signature is invalid.", []string{
			"Rebuild the typed-data payload and signature, then retry.",
		}
	case containsAny(lower, authErrorPhrases):
		return "AUTH_REQUIRED", "A valid authenticated MCP session is required.", []string{
			"Call authenticate to create a fresh authenticated session.",
		}
	case containsAny(lower, permissionErrorPhrases):
		return "PERMISSION_DENIED", "The authenticated wallet is not authorized for this action.", []string{
			"Verify subaccount ownership or delegation permissions, then retry.",
		}
	case containsAny(lower, rateLimitErrorPhrases):
		return "RATE_LIMITED", "Upstream request rate limit exceeded.", []string{
			"Wait before retrying, then use exponential backoff with jitter.",
			"Do not immediately replay write tools; first confirm whether the prior request produced an order or state change.",
			"Prefer fewer, broader read calls over tight polling loops.",
		}
	case containsAny(lower, timeoutErrorPhrases):
		return "TIMEOUT", "The request timed out.", []string{
			"Retry the request. If timeouts persist, check system://status.",
		}
	case containsAny(lower, notFoundErrorPhrases):
		return "NOT_FOUND", "The requested resource was not found.", []string{
			"Verify the requested identifier exists and belongs to the current environment.",
		}
	case containsAny(lower, validationErrorPhrases):
		return "INVALID_ARGUMENT", "Request arguments failed validation.", []string{
			"Check the request fields and value ranges, then retry.",
		}
	default:
		return "BACKEND_UNAVAILABLE", "The request could not be completed.", []string{
			"Retry the request. If the problem persists, inspect upstream service health.",
		}
	}
}

func toolErrorResult(err error) *mcp.CallToolResult {
	code, message, remediation := classifyToolError(err)
	return newToolErrorResult(code, message, remediation...)
}

func toolErrorResponse[Out any](err error) (*mcp.CallToolResult, Out, error) {
	return toolErrorResult(err), initializedZeroOutput[Out](), nil
}

// Return the zero value of Out with all nil slices and
// map replaced by empty, non-nil instances. This ensures JSON
// serialization produces `[]` and `{}` instead of `null`, which is
// critical for agent UX: LLM tool-call parsers treat null arrays and
// missing keys as ambiguous, causing unnecessary retry loops. Using
// reflection here avoids maintaining explicit constructors for 40+ output
// structs scattered across tool files.
func initializedZeroOutput[Out any]() Out {
	var out Out
	ensureNonNilCollections(reflect.ValueOf(&out).Elem())
	return out
}

func ensureNonNilCollections(value reflect.Value) {
	if !value.IsValid() || !value.CanSet() {
		return
	}

	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			ensureNonNilCollections(value.Field(i))
		}
	case reflect.Map:
		if value.IsNil() {
			value.Set(reflect.MakeMap(value.Type()))
		}
	case reflect.Slice:
		if value.IsNil() {
			value.Set(reflect.MakeSlice(value.Type(), 0, 0))
		}
	}
}

func sessionIDFromRequest(req *mcp.CallToolRequest) string {
	if req == nil || req.Session == nil {
		return ""
	}

	return req.Session.ID()
}

func loadSessionState(ctx context.Context, store session.Store, sessionID string) (*session.State, error) {
	if store == nil || sessionID == "" {
		return nil, nil
	}

	state, err := store.Get(ctx, sessionID)
	if err == nil {
		return state, nil
	}
	if errors.Is(err, session.ErrSessionNotFound) {
		return nil, nil
	}

	return nil, err
}

func sanitizeSessionState(
	ctx context.Context,
	store session.Store,
	sessionID string,
	state *session.State,
	verifier SessionAccessVerifier,
) (*session.State, error) {
	if state == nil || state.AuthMode != session.AuthModeAuthenticated || state.SubAccountID <= 0 {
		return state, nil
	}
	if verifier == nil {
		return state, nil
	}
	if err := verifier.VerifySessionAccess(ctx, state.WalletAddress, state.SubAccountID); err != nil {
		if store == nil || sessionID == "" {
			return nil, nil
		}
		_, delErr := store.DeleteIfExists(ctx, sessionID)
		if delErr != nil {
			return nil, errors.Join(err, fmt.Errorf("clear revoked session: %w", delErr))
		}
		return nil, nil
	}
	return state, nil
}

func touchSession(ctx context.Context, store session.Store, sessionID string, ttl time.Duration) error {
	if store == nil || sessionID == "" {
		return nil
	}

	err := store.Touch(ctx, sessionID, ttl)
	if err == nil || errors.Is(err, session.ErrSessionNotFound) {
		return nil
	}

	return err
}

func authModeForState(state *session.State) string {
	if state == nil || state.AuthMode == "" {
		return string(session.AuthModePublic)
	}

	return string(state.AuthMode)
}

func requireAuthenticatedSession(
	ctx context.Context,
	store session.Store,
	verifier SessionAccessVerifier,
	sessionID string,
	requestedSubAccountID *int64,
) (*session.State, error) {
	state, err := loadSessionState(ctx, store, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	state, err = sanitizeSessionState(ctx, store, sessionID, state, verifier)
	if err != nil {
		return nil, err
	}
	if state == nil || state.AuthMode != session.AuthModeAuthenticated || state.SubAccountID <= 0 {
		return nil, fmt.Errorf("authentication required")
	}
	if requestedSubAccountID != nil && *requestedSubAccountID != 0 && *requestedSubAccountID != state.SubAccountID {
		return nil, fmt.Errorf("requested subaccount does not match authenticated session")
	}

	return state, nil
}

