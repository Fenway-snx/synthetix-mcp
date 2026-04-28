package agentbroker

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
)

type fakeDomainProvider struct{}

func (fakeDomainProvider) DomainName() string    { return "Synthetix" }
func (fakeDomainProvider) DomainVersion() string { return "1" }
func (fakeDomainProvider) ChainID() int          { return 1 }

// fakeSubaccountResolver is the REST-shaped stand-in for the old
// Resolver returns owned + delegated subaccount IDs without
// the owner/permissions/expiry metadata that is no longer available
// from /v1/info.
type fakeSubaccountResolver struct {
	resp *ResolvedSubAccounts
	err  error
}

func (f *fakeSubaccountResolver) GetSubAccountIdsWithDelegations(ctx context.Context, wallet string) (*ResolvedSubAccounts, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	return &ResolvedSubAccounts{}, nil
}

func newTestKey(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key, hex.EncodeToString(crypto.FromECDSA(key))
}

func TestNewRejectsMissingKey(t *testing.T) {
	_, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
	})
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

func TestNewLoadsKeyFromFile(t *testing.T) {
	_, hex := newTestKey(t)
	dir := t.TempDir()
	path := dir + "/key.hex"
	if err := writeFile(path, hex); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyFile:     path,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	if broker.WalletAddress() == (common.Address{}) {
		t.Fatal("expected non-zero wallet address")
	}
}

func TestNewMaterializesGuardrailDefaults(t *testing.T) {
	_, keyHex := newTestKey(t)
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyHex:      keyHex,
		SubAccountID:       42,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}

	defaults := broker.GuardrailDefaults()
	if defaults.Preset != "standard" {
		t.Fatalf("expected standard preset, got %q", defaults.Preset)
	}
	if len(defaults.AllowedSymbols) != 1 || defaults.AllowedSymbols[0] != "*" {
		t.Fatalf("expected wildcard symbol default, got %#v", defaults.AllowedSymbols)
	}
	if len(defaults.AllowedOrderTypes) != 2 || defaults.AllowedOrderTypes[0] != "LIMIT" || defaults.AllowedOrderTypes[1] != "MARKET" {
		t.Fatalf("expected LIMIT/MARKET order type defaults, got %#v", defaults.AllowedOrderTypes)
	}
}

func TestSignAuthMessageProducesVerifiableSignature(t *testing.T) {
	key, keyHex := newTestKey(t)
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}

	const subAccountID int64 = 12345
	message, signatureHex, err := broker.SignAuthMessage(subAccountID)
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	// The auth manager re-parses the JSON typed-data and recovers the
	// signer with VerifyEIP712Signature, so this test mirrors that
	// path: if the signature recovers our wallet address, the broker
	// output is acceptable to the live auth manager.
	var typedData apitypes.TypedData
	if err := json.Unmarshal([]byte(message), &typedData); err != nil {
		t.Fatalf("unmarshal typed data: %v", err)
	}
	recovered, err := snx_lib_auth.VerifyEIP712Signature(typedData, signatureHex)
	if err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	expected := crypto.PubkeyToAddress(key.PublicKey)
	if recovered != expected {
		t.Fatalf("recovered %s, expected %s", recovered.Hex(), expected.Hex())
	}
}

func TestSignTradeActionRoundTrip(t *testing.T) {
	key, keyHex := newTestKey(t)
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}

	const subAccountID int64 = 999
	nonce, expires := broker.AllocateNonce()
	if expires <= nonce {
		t.Fatalf("expected expires(%d) > nonce(%d)", expires, nonce)
	}

	payload := &snx_lib_api_validation.PlaceOrdersActionPayload{
		Action:   "placeOrders",
		Grouping: snx_lib_api_validation.GroupingValues_na,
		Source:   "mcp",
		Orders: []snx_lib_api_json.PlaceOrderRequest{{
			Symbol:    "BTC-USDT",
			Side:      "buy",
			OrderType: "limitGtc",
			Price:     "100000",
			Quantity:  snx_lib_api_json.Quantity("1"),
		}},
	}
	validated, err := snx_lib_api_validation.NewValidatedPlaceOrdersAction(payload)
	if err != nil {
		t.Fatalf("validate payload: %v", err)
	}
	signature, err := broker.SignTradeAction(
		subAccountID, nonce, expires,
		snx_lib_api_types.RequestAction("placeOrders"), validated,
	)
	if err != nil {
		t.Fatalf("sign trade action: %v", err)
	}
	if signature.V != 27 && signature.V != 28 {
		t.Fatalf("expected v in {27,28}, got %d", signature.V)
	}
	if !strings.HasPrefix(signature.R, "0x") || len(signature.R) != 66 {
		t.Fatalf("malformed r: %s", signature.R)
	}
	if !strings.HasPrefix(signature.S, "0x") || len(signature.S) != 66 {
		t.Fatalf("malformed s: %s", signature.S)
	}

	// Recompute the typed-data we expect the signer to have built and
	// confirm BuildSignatureHex round-trips back through
	// VerifyEIP712Signature to the broker's wallet.
	typedData, err := snx_lib_auth.CreateTradeTypedData(
		snx_lib_auth.SubAccountId(strconv.FormatInt(subAccountID, 10)),
		snx_lib_auth.Nonce(nonce),
		expires,
		snx_lib_api_types.RequestAction("placeOrders"),
		validated,
		"Synthetix", "1", 1,
	)
	if err != nil {
		t.Fatalf("build expected typed data: %v", err)
	}
	recovered, err := snx_lib_auth.VerifyEIP712Signature(typedData, snx_lib_auth.BuildSignatureHex(signature))
	if err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	expected := crypto.PubkeyToAddress(key.PublicKey)
	if recovered != expected {
		t.Fatalf("recovered %s, expected %s", recovered.Hex(), expected.Hex())
	}
}

func TestEnsureSubAccountPrefersOwnedThenDelegated(t *testing.T) {
	_, keyHex := newTestKey(t)
	resolver := &fakeSubaccountResolver{
		resp: &ResolvedSubAccounts{
			Owned:     []int64{42},
			Delegated: []int64{7},
		},
	}
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: resolver,
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	id, err := broker.EnsureSubAccount(context.Background())
	if err != nil {
		t.Fatalf("ensure subaccount: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected owned subaccount 42, got %d", id)
	}
	// Second call must use the cached value, not re-hit the resolver.
	resolver.resp = nil
	id2, err := broker.EnsureSubAccount(context.Background())
	if err != nil {
		t.Fatalf("ensure subaccount (cached): %v", err)
	}
	if id2 != 42 {
		t.Fatalf("expected cached id 42, got %d", id2)
	}
}

func TestEnsureSubAccountFallsBackToDelegations(t *testing.T) {
	_, keyHex := newTestKey(t)
	resolver := &fakeSubaccountResolver{
		resp: &ResolvedSubAccounts{
			Delegated: []int64{7},
		},
	}
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: resolver,
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	id, err := broker.EnsureSubAccount(context.Background())
	if err != nil {
		t.Fatalf("ensure subaccount: %v", err)
	}
	if id != 7 {
		t.Fatalf("expected delegated subaccount 7, got %d", id)
	}
}

func TestEnsureSubAccountErrorsWithNoneFound(t *testing.T) {
	_, keyHex := newTestKey(t)
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	_, err = broker.EnsureSubAccount(context.Background())
	if err == nil {
		t.Fatal("expected error when no subaccounts found")
	}
}

func writeFile(path string, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o600)
}

// Locks the action-level allowlist gate for owner-only actions that
// are intentionally outside the broker's scope.
func TestSignTradeActionRejectsDisallowedAction(t *testing.T) {
	_, keyHex := newTestKey(t)
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	nonce, expires := broker.AllocateNonce()
	disallowed := []snx_lib_api_types.RequestAction{
		"createSubaccount",
	}
	for _, action := range disallowed {
		_, err := broker.SignTradeAction(1, nonce, expires, action, struct{}{})
		if !errors.Is(err, ErrDisallowedAction) {
			t.Fatalf("expected ErrDisallowedAction for %q, got %v", action, err)
		}
	}
}

// Confirms the chainId/domain gate catches replayable cross-chain or
// cross-protocol payloads before crypto.Sign runs.
func TestSignTypedDataRejectsForeignDomain(t *testing.T) {
	_, keyHex := newTestKey(t)
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	cases := []struct {
		name string
		td   apitypes.TypedData
	}{
		{
			name: "wrong_domain_name",
			td: apitypes.TypedData{
				PrimaryType: "AuthMessage",
				Domain: apitypes.TypedDataDomain{
					Name: "NotSynthetix", Version: "1",
					ChainId: math.NewHexOrDecimal256(1),
				},
			},
		},
		{
			name: "wrong_domain_version",
			td: apitypes.TypedData{
				PrimaryType: "AuthMessage",
				Domain: apitypes.TypedDataDomain{
					Name: "Synthetix", Version: "2",
					ChainId: math.NewHexOrDecimal256(1),
				},
			},
		},
		{
			name: "wrong_chain_id",
			td: apitypes.TypedData{
				PrimaryType: "AuthMessage",
				Domain: apitypes.TypedDataDomain{
					Name: "Synthetix", Version: "1",
					ChainId: math.NewHexOrDecimal256(8453),
				},
			},
		},
		{
			name: "missing_chain_id",
			td: apitypes.TypedData{
				PrimaryType: "AuthMessage",
				Domain:      apitypes.TypedDataDomain{Name: "Synthetix", Version: "1"},
			},
		},
		{
			name: "disallowed_primary_type",
			td: apitypes.TypedData{
				PrimaryType: "Permit", // ERC-20 Permit; absolutely never signed by broker.
				Domain: apitypes.TypedDataDomain{
					Name: "Synthetix", Version: "1",
					ChainId: math.NewHexOrDecimal256(1),
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := broker.signTypedData(tc.td)
			if !errors.Is(err, ErrDisallowedPrimaryType) {
				t.Fatalf("expected ErrDisallowedPrimaryType, got %v", err)
			}
		})
	}
}

// Documents the pre-resolution snapshot get_server_info emits before
// any agent has connected.
func TestStatusBeforeEnsureSubAccount(t *testing.T) {
	_, keyHex := newTestKey(t)
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: &fakeSubaccountResolver{},
		PrivateKeyHex:      keyHex,
		SubAccountID:       42,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	status := broker.Status()
	if status.WalletAddress == "" {
		t.Fatal("expected wallet address in pre-resolution status")
	}
	if status.ChainID != 1 {
		t.Fatalf("expected chainId 1, got %d", status.ChainID)
	}
	if status.DomainName != "Synthetix" {
		t.Fatalf("expected Synthetix domain, got %q", status.DomainName)
	}
	if status.DefaultPreset != "standard" {
		t.Fatalf("expected standard preset, got %q", status.DefaultPreset)
	}
	if status.SubAccountID != 42 {
		t.Fatalf("expected pinned subaccount 42, got %d", status.SubAccountID)
	}
	if status.SubaccountSource != SubaccountSourcePreconfigured {
		t.Fatalf("expected preconfigured source, got %q", status.SubaccountSource)
	}
}

// Delegated posture: after the REST migration, owner address /
// permissions / expiry are no longer available from /v1/info, so
// Status() reports the delegated subaccount id and best-effort
// placeholders. The test locks in the placeholder contract so later
// regressions in signed action shape are deliberate.
func TestStatusAfterDelegatedDiscovery(t *testing.T) {
	_, keyHex := newTestKey(t)
	resolver := &fakeSubaccountResolver{
		resp: &ResolvedSubAccounts{
			Delegated: []int64{1234},
		},
	}
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: resolver,
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	if _, err := broker.EnsureSubAccount(context.Background()); err != nil {
		t.Fatalf("ensure subaccount: %v", err)
	}
	status := broker.Status()
	if status.SubaccountSource != SubaccountSourceDelegated {
		t.Fatalf("expected delegated source, got %q", status.SubaccountSource)
	}
	if status.SubAccountID != 1234 {
		t.Fatalf("expected subaccount 1234, got %d", status.SubAccountID)
	}
	// Owner/permissions/expiry are not surfaced by /v1/info.
	if status.OwnerAddress != "" {
		t.Fatalf("expected owner address unknown on REST path, got %q", status.OwnerAddress)
	}
	if status.ExpiresAtUnix != 0 {
		t.Fatalf("expected unknown expiry on REST path, got %d", status.ExpiresAtUnix)
	}
}

// Owned-key parity case. Permissions must surface as ["*"] so
// get_server_info can render an unambiguous full-authority warning.
func TestStatusAfterOwnedDiscovery(t *testing.T) {
	_, keyHex := newTestKey(t)
	resolver := &fakeSubaccountResolver{
		resp: &ResolvedSubAccounts{
			Owned: []int64{99},
		},
	}
	broker, err := New(Options{
		DomainProvider:     fakeDomainProvider{},
		SubaccountResolver: resolver,
		PrivateKeyHex:      keyHex,
	})
	if err != nil {
		t.Fatalf("new broker: %v", err)
	}
	if _, err := broker.EnsureSubAccount(context.Background()); err != nil {
		t.Fatalf("ensure subaccount: %v", err)
	}
	status := broker.Status()
	if status.SubaccountSource != SubaccountSourceOwned {
		t.Fatalf("expected owned source, got %q", status.SubaccountSource)
	}
	if len(status.Permissions) != 1 || status.Permissions[0] != "*" {
		t.Fatalf("expected wildcard permissions, got %v", status.Permissions)
	}
	if status.ExpiresAtUnix != 0 {
		t.Fatalf("expected no expiry on owned posture, got %d", status.ExpiresAtUnix)
	}
}

// Keeps math/big in the import set; the chainId gate uses *big.Int.
var _ = big.NewInt(0)
