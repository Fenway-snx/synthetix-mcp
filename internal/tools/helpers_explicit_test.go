package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type explicitOutput struct {
	Items  []string
	Labels map[string]string
}

type explicitSessionStore struct {
	state    *session.State
	getErr   error
	touchErr error
}

func (s *explicitSessionStore) Delete(context.Context, string) error {
	return nil
}

func (s *explicitSessionStore) DeleteIfExists(context.Context, string) (bool, error) {
	return true, nil
}

func (s *explicitSessionStore) Get(context.Context, string) (*session.State, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.state, nil
}

func (s *explicitSessionStore) Save(context.Context, string, *session.State, time.Duration) error {
	return nil
}

func (s *explicitSessionStore) Touch(context.Context, string, time.Duration) error {
	return s.touchErr
}

func TestInitializedZeroOutputInitializesNilCollections(t *testing.T) {
	output := initializedZeroOutput[explicitOutput]()

	if output.Items == nil {
		t.Fatal("expected nil slice to be initialized")
	}
	if len(output.Items) != 0 {
		t.Fatalf("expected initialized slice to be empty, got %d items", len(output.Items))
	}
	if output.Labels == nil {
		t.Fatal("expected nil map to be initialized")
	}
	if len(output.Labels) != 0 {
		t.Fatalf("expected initialized map to be empty, got %d entries", len(output.Labels))
	}
}

func TestClassifyToolErrorSessionNotFound(t *testing.T) {
	code, message, remediation := classifyToolError(session.ErrSessionNotFound)

	if code != "AUTH_REQUIRED" {
		t.Fatalf("expected AUTH_REQUIRED, got %q", code)
	}
	if message != "A valid authenticated MCP session is required." {
		t.Fatalf("unexpected auth-required message %q", message)
	}
	if len(remediation) != 2 {
		t.Fatalf("expected two remediation steps, got %d", len(remediation))
	}
}

func TestClassifyToolErrorRecognizesInvalidSignature(t *testing.T) {
	code, message, remediation := classifyToolError(errors.New("validate trade action signature: invalid trade action signature"))

	if code != "INVALID_SIGNATURE" {
		t.Fatalf("expected INVALID_SIGNATURE, got %q", code)
	}
	if message != "The supplied EIP-712 payload or signature is invalid." {
		t.Fatalf("unexpected invalid signature message %q", message)
	}
	if len(remediation) != 1 || remediation[0] != "Rebuild the typed-data payload and signature, then retry." {
		t.Fatalf("unexpected remediation payload %#v", remediation)
	}
}

func TestClassifyToolErrorFallsBackToPhraseMatching(t *testing.T) {
	code, message, remediation := classifyToolError(errors.New("delegation permission denied for requested action"))

	if code != "PERMISSION_DENIED" {
		t.Fatalf("expected PERMISSION_DENIED, got %q", code)
	}
	if message != "The authenticated wallet is not authorized for this action." {
		t.Fatalf("unexpected permission message %q", message)
	}
	if len(remediation) != 1 || !strings.Contains(remediation[0], "delegation") {
		t.Fatalf("unexpected permission remediation %#v", remediation)
	}
}

func TestClassifyToolErrorPhraseTightening(t *testing.T) {
	tests := []struct {
		name         string
		err          string
		expectedCode string
	}{
		{"ambiguous_invalidation", "cache invalidation failed", "BACKEND_UNAVAILABLE"},
		{"ambiguous_required_internal", "required for internal processing", "BACKEND_UNAVAILABLE"},
		{"ambiguous_must_not", "you must not confuse these", "BACKEND_UNAVAILABLE"},
		{"valid_invalid_field", "invalid order quantity", "INVALID_ARGUMENT"},
		{"valid_is_required", "symbol is required", "INVALID_ARGUMENT"},
		{"valid_are_required", "subscriptions are required", "INVALID_ARGUMENT"},
		{"valid_requires_id", "modify_order requires venueOrderId or clientOrderId", "INVALID_ARGUMENT"},
		{"valid_must_be", "side must be BUY or SELL", "INVALID_ARGUMENT"},
		{"valid_must_align", "must align to tick size 0.5", "INVALID_ARGUMENT"},
		{"valid_exceeds", "close quantity exceeds current position quantity", "INVALID_ARGUMENT"},
		{"valid_unsupported", "unsupported order type", "INVALID_ARGUMENT"},
		{"valid_do_not_accept", "MARKET orders do not accept timeInForce in MCP", "INVALID_ARGUMENT"},
		{"valid_cannot_close", "cannot close symbol with both long and short exposure", "INVALID_ARGUMENT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := classifyToolError(errors.New(tt.err))
			if code != tt.expectedCode {
				t.Fatalf("classifyToolError(%q) = %q, want %q", tt.err, code, tt.expectedCode)
			}
		})
	}
}

// Pins every tool-error branch against explicit expected output.
// Mismatches fail fast with labelled cases.
func TestClassifyToolErrorRoundTripPreservesBehavior(t *testing.T) {
	type expected struct {
		code        string
		message     string
		remediation int
	}
	type caseT struct {
		name string
		err  error
		want expected
	}

	cases := []caseT{
		{
			name: "nil_returns_unknown",
			err:  nil,
			want: expected{"UNKNOWN", "An unexpected error occurred.", 0},
		},
		{
			name: "session_not_found_has_two_remediations",
			err:  session.ErrSessionNotFound,
			want: expected{"AUTH_REQUIRED", "A valid authenticated MCP session is required.", 2},
		},

		// Plain errors: one probe per switch arm to lock precedence.
		// Guardrail violation wins over permission-denied even though the
		// phrase "permission denied" would otherwise match.
		{
			name: "phrase_guardrail_wins_over_permission",
			err:  errors.New("guardrail violation: permission denied for preset"),
			want: expected{"GUARDRAIL_VIOLATION", "The current session guardrails do not permit this action.", 3},
		},
		{
			name: "phrase_not_implemented",
			err:  errors.New("this feature is not implemented yet"),
			want: expected{"NOT_IMPLEMENTED", "This tool is not implemented for the current phase.", 1},
		},
		{
			name: "phrase_invalid_signature",
			err:  errors.New("validate trade action signature: invalid trade action signature"),
			want: expected{"INVALID_SIGNATURE", "The supplied EIP-712 payload or signature is invalid.", 1},
		},
		{
			name: "phrase_auth_required",
			err:  errors.New("session expired"),
			want: expected{"AUTH_REQUIRED", "A valid authenticated MCP session is required.", 1},
		},
		{
			name: "phrase_permission_denied",
			err:  errors.New("delegation not granted"),
			want: expected{"PERMISSION_DENIED", "The authenticated wallet is not authorized for this action.", 1},
		},
		{
			name: "phrase_rate_limited",
			err:  errors.New("too many requests"),
			want: expected{"RATE_LIMITED", "Upstream request rate limit exceeded.", 3},
		},
		{
			name: "phrase_timeout",
			err:  errors.New("deadline exceeded on upstream call"),
			want: expected{"TIMEOUT", "The request timed out.", 1},
		},
		{
			name: "phrase_not_found",
			err:  errors.New("unknown symbol FAKE-PERP"),
			want: expected{"NOT_FOUND", "The requested resource was not found.", 1},
		},
		{
			name: "phrase_invalid_argument",
			err:  errors.New("symbol is required"),
			want: expected{"INVALID_ARGUMENT", "Request arguments failed validation.", 1},
		},
		{
			name: "phrase_unmatched_falls_back",
			err:  errors.New("some opaque upstream hiccup with no known phrase"),
			want: expected{"BACKEND_UNAVAILABLE", "The request could not be completed.", 1},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			code, message, remediation := classifyToolError(tc.err)
			if code != tc.want.code {
				t.Fatalf("code = %q, want %q", code, tc.want.code)
			}
			if message != tc.want.message {
				t.Fatalf("message = %q, want %q", message, tc.want.message)
			}
			if len(remediation) != tc.want.remediation {
				t.Fatalf("remediation len = %d, want %d (%#v)", len(remediation), tc.want.remediation, remediation)
			}
		})
	}
}

func TestRequireAuthenticatedSessionAcceptsMatchingSubaccount(t *testing.T) {
	subAccountID := int64(77)
	store := &explicitSessionStore{
		state: &session.State{
			AuthMode:      session.AuthModeAuthenticated,
			SubAccountID:  subAccountID,
			WalletAddress: "0xabc",
		},
	}

	state, err := requireAuthenticatedSession(context.Background(), store, nil, "session-1", &subAccountID)
	if err != nil {
		t.Fatalf("expected authenticated session, got %v", err)
	}
	if state.SubAccountID != 77 {
		t.Fatalf("expected matching subaccount 77, got %d", state.SubAccountID)
	}
}

func TestRequireAuthenticatedSessionRejectsSubaccountMismatch(t *testing.T) {
	requestedSubAccountID := int64(88)
	store := &explicitSessionStore{
		state: &session.State{
			AuthMode:      session.AuthModeAuthenticated,
			SubAccountID:  77,
			WalletAddress: "0xabc",
		},
	}

	_, err := requireAuthenticatedSession(context.Background(), store, nil, "session-1", &requestedSubAccountID)
	if err == nil {
		t.Fatal("expected subaccount mismatch to fail")
	}
	if !strings.Contains(err.Error(), "requested subaccount does not match authenticated session") {
		t.Fatalf("unexpected mismatch error %v", err)
	}
}

func TestTouchSessionIgnoresMissingSession(t *testing.T) {
	store := &explicitSessionStore{touchErr: session.ErrSessionNotFound}

	err := touchSession(context.Background(), store, "session-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("expected missing session touch to be ignored, got %v", err)
	}
}

func TestActiveSubscriptionsNormalizesNilSlice(t *testing.T) {
	reader := &fakeSubscriptionReader{bySession: map[string][]string{
		"session-1": nil,
	}}

	result := activeSubscriptions(reader, "session-1")

	if result == nil {
		t.Fatal("expected nil slice to be normalized to empty slice")
	}
	if len(result) != 0 {
		t.Fatalf("expected no active subscriptions, got %#v", result)
	}
}

func TestTradingPayloadBuildersHandleEachIdentifierPathExplicitly(t *testing.T) {
	normalizedName, reduceOnly, canonicalType, canonicalTIF, err := mapMCPOrderType("LIMIT", "IOC")
	if err != nil {
		t.Fatalf("expected LIMIT/IOC mapping to succeed, got %v", err)
	}
	if normalizedName != "limitIoc" || reduceOnly || canonicalType != "LIMIT" || canonicalTIF != "IOC" {
		t.Fatalf("unexpected LIMIT/IOC normalization %q %t %q %q", normalizedName, reduceOnly, canonicalType, canonicalTIF)
	}

	modifyPayload, venueOrderID, clientOrderID, err := buildModifyPayload(modifyOrderInput{
		VenueOrderID: "42",
		Price:        "42000",
		Quantity:     "1.5",
	})
	if err != nil {
		t.Fatalf("expected venue modify payload to validate, got %v", err)
	}
	if venueOrderID != 42 || clientOrderID != "" {
		t.Fatalf("unexpected venue modify identifiers %d %q", venueOrderID, clientOrderID)
	}
	if _, ok := modifyPayload.(*validation.ValidatedModifyOrderAction); !ok {
		t.Fatalf("expected validated venue modify payload, got %T", modifyPayload)
	}

	cloidPayload, venueIDs, clientIDs, err := buildCancelPayload(cancelOrderInput{ClientOrderID: "cloid-7"})
	if err != nil {
		t.Fatalf("expected client cancel payload to validate, got %v", err)
	}
	if venueIDs != nil {
		t.Fatalf("expected no venue IDs for CLOID cancel, got %#v", venueIDs)
	}
	if len(clientIDs) != 1 || clientIDs[0] != "cloid-7" {
		t.Fatalf("unexpected client cancel identifiers %#v", clientIDs)
	}
	if _, ok := cloidPayload.(*validation.ValidatedCancelOrdersByCloidAction); !ok {
		t.Fatalf("expected validated CLOID cancel payload, got %T", cloidPayload)
	}

	cancelAllPayload, symbols, err := buildCancelAllPayload(cancelAllOrdersInput{Symbol: "BTC-USDT"})
	if err != nil {
		t.Fatalf("expected cancel-all payload to validate, got %v", err)
	}
	if len(symbols) != 1 || symbols[0] != "BTC-USDT" {
		t.Fatalf("unexpected cancel-all symbols %#v", symbols)
	}
	if _, ok := cancelAllPayload.(*validation.ValidatedCancelAllOrdersAction); !ok {
		t.Fatalf("expected validated cancel-all payload, got %T", cancelAllPayload)
	}
}

func TestTradingPayloadBuildersRejectMutuallyExclusiveIdentifiers(t *testing.T) {
	_, _, _, err := buildModifyPayload(modifyOrderInput{
		VenueOrderID:  "42",
		ClientOrderID: "cloid-42",
	})
	if err == nil {
		t.Fatal("expected modify payload with two identifiers to fail")
	}
	if !strings.Contains(err.Error(), "either venueOrderId or clientOrderId") {
		t.Fatalf("unexpected modify payload error %v", err)
	}

	_, _, _, err = buildCancelPayload(cancelOrderInput{
		VenueOrderID:  "42",
		ClientOrderID: "cloid-42",
	})
	if err == nil {
		t.Fatal("expected cancel payload with two identifiers to fail")
	}
	if !strings.Contains(err.Error(), "either venueOrderId or clientOrderId") {
		t.Fatalf("unexpected cancel payload error %v", err)
	}
}

func TestResolveClosablePositionExplicitLong(t *testing.T) {
	side, qty, err := resolveClosablePositionOrExplicit(
		context.Background(),
		nil,
		ToolContext{},
		"BTC-USDT",
		"long",
		"2.5",
	)
	if err != nil {
		t.Fatalf("expected explicit long branch to succeed, got %v", err)
	}
	if side != "long" {
		t.Fatalf("expected long side, got %q", side)
	}
	if qty.String() != "2.5" {
		t.Fatalf("expected quantity 2.5, got %s", qty.String())
	}
}

func TestResolveClosablePositionExplicitRejectsInvalidSide(t *testing.T) {
	_, _, err := resolveClosablePositionOrExplicit(
		context.Background(),
		nil,
		ToolContext{},
		"BTC-USDT",
		"flat",
		"1",
	)
	if err == nil {
		t.Fatal("expected invalid side to fail")
	}
	if !strings.Contains(err.Error(), "side must be") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestResolveClosablePositionExplicitRequiresQuantity(t *testing.T) {
	_, _, err := resolveClosablePositionOrExplicit(
		context.Background(),
		nil,
		ToolContext{},
		"BTC-USDT",
		"long",
		"",
	)
	if err == nil {
		t.Fatal("expected missing quantity to fail")
	}
	if !strings.Contains(err.Error(), "quantity is required") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestResolveClosablePositionNoReadsFallbackErrors(t *testing.T) {
	_, _, err := resolveClosablePosition(
		context.Background(),
		nil,
		ToolContext{},
		"BTC-USDT",
	)
	if err == nil {
		t.Fatal("expected nil tradeReads fallback to fail")
	}
	if !errors.Is(err, ErrReadUnavailable) {
		t.Fatalf("expected ErrReadUnavailable, got %v", err)
	}
}

func TestErrorDetailForCodeReturnsFallbackForUnknownCode(t *testing.T) {
	detail := errorDetailForCode("SOMETHING_NEW")
	if detail == nil {
		t.Fatal("expected fallback error detail")
	}
	if detail.Retryable {
		t.Fatalf("expected unknown code fallback to be non-retryable, got %#v", detail)
	}
	if len(detail.Remediation) != 2 {
		t.Fatalf("expected two fallback remediation steps, got %#v", detail.Remediation)
	}
}
