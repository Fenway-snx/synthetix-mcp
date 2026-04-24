// Package agentbroker provides an optional, single-tenant, server-side
// EIP-712 signer for the MCP server. When enabled, the broker holds a
// private key in process memory and exposes signing helpers used by the
// `quick_*` MCP tools. Those tools collapse the "discover subaccount →
// preview_auth_message → sign → authenticate → preview_trade_signature
// → sign → place_order" choreography (which an LLM running in a
// restricted client cannot do without writing throwaway TypeScript
// files) into a single tool call.
//
// Security model: the private key never leaves this process. The broker
// is rejected at startup unless the MCP server binds to loopback or the
// operator explicitly opts into a non-loopback bind. There is exactly
// one wallet per server instance — if you need multi-tenant signing,
// keep using the standard preview/authenticate/sign flow against your
// own wallet.
package agentbroker

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	sdkeip712 "github.com/synthetixio/synthetix-go/eip712"
	"github.com/synthetixio/synthetix-go/signer"
)

// Subset of the auth manager's Authenticate interface. Kept as an
// interface so tests can stub the heavy auth manager wiring.
type Authenticator interface {
	Authenticate(ctx context.Context, sessionID string, message string, signatureHex string) (*AuthenticateResult, error)
}

// Mirror of services/mcp/internal/auth.AuthenticateResult to avoid an
// import cycle (the broker is consumed by tools, which are consumed by
// the auth-manager-instantiating server). The fields we need are only
// the wallet/subaccount/expiry triple.
type AuthenticateResult struct {
	SessionExpiresAt int64
	SubAccountID     int64
	WalletAddress    string
}

// Subset of /v1/info used for subaccount discovery. Owner,
// permissions, and expiry are not available on the REST path and
// must be treated as best-effort.
type SubaccountResolver interface {
	GetSubAccountIdsWithDelegations(ctx context.Context, wallet string) (*ResolvedSubAccounts, error)
}

// Transport-neutral shape of the subaccount-discovery response.
type ResolvedSubAccounts struct {
	Owned     []int64
	Delegated []int64
}

// Closed set of EIP-712 primaryType values the broker will sign.
// Anything else (Permit, WithdrawCollateral, AddDelegatedSigner, …)
// is rejected before crypto.Sign sees the digest. To extend: add the
// RequestAction to allowedTradeActions AND the primaryType emitted by
// lib/auth.CreateTradeTypedData here. Both gates must pass.
var allowedPrimaryTypes = map[string]struct{}{
	"AuthMessage":         {},
	"CancelAllOrders":     {},
	"CancelOrders":        {},
	"CancelOrdersByCloid": {},
	"ModifyOrder":         {},
	"ModifyOrderByCloid":  {},
	"PlaceOrders":         {},
	// SubAccountAction is the EIP-712 primary type used by every
	// read action on /v1/trade (getSubAccount, getSubAccounts,
	// getOpenOrders, getPositions, ...). Its typed-data message
	// carries only subAccountId/action/expiresAfter (no nonce) and
	// so cannot move collateral or orders — adding it here lets the
	// broker sign idempotent reads without widening the custodial
	// surface.
	"SubAccountAction": {},
}

// Per-action gate checked before typed-data construction. Withdrawals,
// delegation management, leverage changes, and transfers are
// deliberately absent — the broker is a trading delegate, not a
// custody surface.
var allowedTradeActions = map[snx_lib_api_types.RequestAction]struct{}{
	snx_lib_api_types.RequestAction("placeOrders"):     {},
	snx_lib_api_types.RequestAction("cancelOrders"):    {},
	snx_lib_api_types.RequestAction("cancelAllOrders"): {},
	snx_lib_api_types.RequestAction("modifyOrder"):     {},
}

// Per-action gate checked before signing an idempotent read. Reads
// resolve to the SubAccountAction primaryType and never move funds or
// orders, but we still allowlist them explicitly so a rogue caller
// cannot smuggle an arbitrary get* action through the broker — every
// one here maps to an endpoint the self-hosted MCP service is
// expected to proxy. Keep in sync with the getSubAccount /
// getSubAccounts / getOpenOrders / getPositions handlers in
// lib/api/handlers/trade.
var allowedReadActions = map[snx_lib_api_types.RequestAction]struct{}{
	snx_lib_api_types.RequestAction("getSubAccount"):         {},
	snx_lib_api_types.RequestAction("getSubAccounts"):        {},
	snx_lib_api_types.RequestAction("getOpenOrders"):         {},
	snx_lib_api_types.RequestAction("getPositions"):          {},
	snx_lib_api_types.RequestAction("getOrderHistory"):       {},
	snx_lib_api_types.RequestAction("getTrades"):             {},
	snx_lib_api_types.RequestAction("getFundingPayments"):    {},
	snx_lib_api_types.RequestAction("getPerformanceHistory"): {},
}

// Returned when an EIP-712 payload's primaryType (or domain triple)
// is outside the broker allowlist. Callers must surface verbatim —
// never fall back to a more permissive signer.
var ErrDisallowedPrimaryType = errors.New("agent broker: primaryType not in broker allowlist")

// Returned when a trade action is outside allowedTradeActions, before
// any typed-data construction.
var ErrDisallowedAction = errors.New("agent broker: action not in broker allowlist")

// Domain values come from the live auth manager so the broker signs
// against the same chain/version the server validates against.
type DomainProvider interface {
	DomainName() string
	DomainVersion() string
	ChainID() int
}

// Defaults applied to a freshly auto-authenticated session if the
// agent has not already called set_guardrails. Mirrors the shape of
// guardrails.Config so the wiring layer can hand them straight to the
// session store.
type GuardrailDefaults struct {
	AllowedOrderTypes   []string
	AllowedSymbols      []string
	MaxOrderQuantity    string
	MaxPositionQuantity string
	Preset              string
}

type Options struct {
	DomainProvider     DomainProvider
	GuardrailDefaults  GuardrailDefaults
	Logger             snx_lib_logging.Logger
	PrivateKeyFile     string
	PrivateKeyHex      string
	SubAccountID       int64
	SubaccountResolver SubaccountResolver
}

// Broker holds the in-process private key and caches the resolved
// subaccount. SignAuthMessage / SignTradeAction are the only two
// surface methods callers need; everything else (discovery, lazy auth)
// is hidden so test stubs can swap the heavyweight pieces.
type Broker struct {
	chainID          int
	defaults         GuardrailDefaults
	domainName       string
	domainVersion    string
	logger           snx_lib_logging.Logger
	privateKey       *ecdsa.PrivateKey
	subaccountClient SubaccountResolver
	walletAddress    common.Address

	mu             sync.Mutex
	status         BrokerStatus
	statusLogged   bool
	subAccountID   int64
}

// How the broker resolved its subaccount. Surfaced through Status()
// and get_server_info; values are stable wire identifiers — do not
// rename without coordinating with consumers.
type SubaccountSource string

const (
	// Default before EnsureSubAccount has run.
	SubaccountSourceUnknown SubaccountSource = ""
	// Broker key is the registered owner of the subaccount.
	// Discouraged: grants every primaryType the auth manager accepts
	// (withdrawals, delegation management, …). Prefer delegated.
	SubaccountSourceOwned SubaccountSource = "owned"
	// Broker key is a trading-only delegate of a cold owner wallet.
	// Recommended posture: bounded scope and expiry.
	SubaccountSourceDelegated SubaccountSource = "delegated"
	// Operator pinned SNXMCP_AGENT_BROKER_SUB_ACCOUNT_ID; ownership
	// cannot be inferred without a separate lookup.
	SubaccountSourcePreconfigured SubaccountSource = "preconfigured"
)

// Publishable snapshot of broker posture. Static fields (WalletAddress,
// ChainID, DomainName, DomainVersion, DefaultPreset) populate in New();
// the rest fill in after the first EnsureSubAccount. JSON tags wire
// directly into the public get_server_info payload; nil/empty values
// are omitted so the not-yet-resolved case renders minimally.
type BrokerStatus struct {
	ChainID           int              `json:"chainId,omitempty"`
	DefaultPreset     string           `json:"defaultPreset,omitempty"`
	DelegationID      uint64           `json:"delegationId,omitempty"`
	DomainName        string           `json:"domainName,omitempty"`
	DomainVersion     string           `json:"domainVersion,omitempty"`
	ExpiresAtUnix     int64            `json:"expiresAtUnix,omitempty"`
	OwnerAddress      string           `json:"ownerAddress,omitempty"`
	Permissions       []string         `json:"permissions,omitempty"`
	SubAccountID      int64            `json:"subAccountId,omitempty"`
	SubaccountSource  SubaccountSource `json:"subaccountSource,omitempty"`
	WalletAddress     string           `json:"walletAddress,omitempty"`
}

// New parses and validates the configured private key, derives the
// wallet address, and (if SubAccountID was not pre-pinned) records 0
// so the first authenticate call performs discovery. Returns a
// descriptive error on bad keys / unreadable files so the operator
// sees the failure at server startup, not on the first agent call.
func New(opts Options) (*Broker, error) {
	if opts.DomainProvider == nil {
		return nil, errors.New("agent broker: domain provider is required")
	}
	if opts.SubaccountResolver == nil {
		return nil, errors.New("agent broker: subaccount resolver is required")
	}
	keyHex, err := loadPrivateKeyHex(opts.PrivateKeyHex, opts.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(keyHex), "0x"))
	if err != nil {
		return nil, fmt.Errorf("agent broker: parse private key: %w", err)
	}
	walletAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	defaults := opts.GuardrailDefaults
	if strings.TrimSpace(defaults.Preset) == "" {
		// "standard" is the only writable preset today; defaulting to
		// it (instead of the read_only fallback) is the whole point of
		// the broker — it removes the "Need to set guardrails first"
		// step that bit Claude Code in the original transcript.
		defaults.Preset = "standard"
	}
	if len(defaults.AllowedOrderTypes) == 0 {
		defaults.AllowedOrderTypes = []string{"LIMIT", "MARKET"}
	}

	b := &Broker{
		chainID:          opts.DomainProvider.ChainID(),
		defaults:         defaults,
		domainName:       opts.DomainProvider.DomainName(),
		domainVersion:    opts.DomainProvider.DomainVersion(),
		logger:           opts.Logger,
		privateKey:       privateKey,
		subaccountClient: opts.SubaccountResolver,
		subAccountID:     opts.SubAccountID,
		walletAddress:    walletAddress,
	}
	if b.domainName == "" {
		b.domainName = sdkeip712.DefaultDomainName
	}
	if b.domainVersion == "" {
		b.domainVersion = sdkeip712.DefaultDomainVersion
	}
	if b.chainID == 0 {
		b.chainID = sdkeip712.DefaultChainID
	}
	b.status = BrokerStatus{
		ChainID:       b.chainID,
		DefaultPreset: defaults.Preset,
		DomainName:    b.domainName,
		DomainVersion: b.domainVersion,
		SubAccountID:  opts.SubAccountID,
		WalletAddress: walletAddress.Hex(),
	}
	if opts.SubAccountID > 0 {
		// Pinned but not yet resolved as owned vs delegated; flag it
		// so get_server_info doesn't claim an ownership posture we
		// can't justify.
		b.status.SubaccountSource = SubaccountSourcePreconfigured
	}
	if b.logger != nil {
		b.logger.Info(
			"agent broker initialised",
			"walletAddress", walletAddress.Hex(),
			"subAccountId", opts.SubAccountID,
			"defaultPreset", defaults.Preset,
			"chainId", b.chainID,
			"domainName", b.domainName,
		)
	}
	return b, nil
}

// WalletAddress returns the EIP-55 checksum address derived from the
// broker's private key. Useful for log lines and for the "I'm signed
// in as X" hint surfaced by the quick tools.
func (b *Broker) WalletAddress() common.Address {
	return b.walletAddress
}

// GuardrailDefaults exposes the configured defaults so the session
// auto-bootstrap path can populate the session state with them.
func (b *Broker) GuardrailDefaults() GuardrailDefaults {
	return b.defaults
}

// SubAccountID returns the cached subaccount or 0 if discovery has not
// yet run. Callers that need the resolved value should call
// EnsureSubAccount first.
func (b *Broker) SubAccountID() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.subAccountID
}

// Returns a usable subaccount ID, looking it up on first call. Owned
// subaccounts are preferred; falls back to the first delegated grant.
// Result is cached for the broker's lifetime, and the first
// successful resolution emits a one-shot "agent broker bound to
// subaccount" log line with posture, owner, expiry, and permissions.
func (b *Broker) EnsureSubAccount(ctx context.Context) (int64, error) {
	b.mu.Lock()
	if b.subAccountID > 0 {
		id := b.subAccountID
		b.mu.Unlock()
		return id, nil
	}
	b.mu.Unlock()

	walletHex := b.walletAddress.Hex()
	resp, err := b.subaccountClient.GetSubAccountIdsWithDelegations(ctx, walletHex)
	if err != nil {
		return 0, fmt.Errorf("agent broker: resolve subaccounts for %s: %w", walletHex, err)
	}
	if resp != nil && len(resp.Owned) > 0 {
		id := resp.Owned[0]
		b.cacheSubAccountFromOwned(id)
		return id, nil
	}

	// Fall back to delegated subaccounts so a delegate-only signer
	// onboards without manual subAccountId configuration. Owner,
	// permissions, and expiry aren't exposed by REST.
	if resp != nil && len(resp.Delegated) > 0 {
		id := resp.Delegated[0]
		b.cacheSubAccountFromDelegation(id)
		return id, nil
	}

	return 0, fmt.Errorf(
		"agent broker: wallet %s has no owned or delegated subaccounts; "+
			"run sample/node-scripts/scripts/onboard-agent-key.ts to mint "+
			"a delegate grant from your owner wallet, or set "+
			"SNXMCP_AGENT_BROKER_SUB_ACCOUNT_ID explicitly",
		walletHex,
	)
}

func (b *Broker) cacheSubAccountFromOwned(id int64) {
	b.mu.Lock()
	if b.subAccountID == 0 {
		b.subAccountID = id
	}
	b.status.SubAccountID = b.subAccountID
	b.status.SubaccountSource = SubaccountSourceOwned
	b.status.OwnerAddress = b.walletAddress.Hex()
	// ["*"] (rather than empty) makes it obvious in get_server_info
	// that owned-mode grants every action the auth manager accepts.
	b.status.Permissions = []string{"*"}
	b.status.ExpiresAtUnix = 0
	b.status.DelegationID = 0
	snapshot := b.status
	shouldLog := !b.statusLogged
	b.statusLogged = true
	b.mu.Unlock()
	if shouldLog {
		b.logBoundSummary(snapshot)
	}
}

// Delegate-mode binding. Owner/permissions/expiry are zero-valued
// because REST doesn't expose them.
func (b *Broker) cacheSubAccountFromDelegation(id int64) {
	b.mu.Lock()
	if b.subAccountID == 0 {
		b.subAccountID = id
	}
	b.status.SubAccountID = b.subAccountID
	b.status.SubaccountSource = SubaccountSourceDelegated
	b.status.OwnerAddress = ""
	b.status.Permissions = nil
	b.status.DelegationID = 0
	b.status.ExpiresAtUnix = 0
	snapshot := b.status
	shouldLog := !b.statusLogged
	b.statusLogged = true
	b.mu.Unlock()
	if shouldLog {
		b.logBoundSummary(snapshot)
	}
}

// Returns a detached copy of the current broker state, safe to
// JSON-encode or log without holding the lock. Before the first
// EnsureSubAccount only the static fields are populated.
func (b *Broker) Status() BrokerStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	clone := b.status
	if len(b.status.Permissions) > 0 {
		clone.Permissions = append([]string(nil), b.status.Permissions...)
	}
	return clone
}

// Emits the one-shot "agent broker bound to …" line on first
// resolution. Called by cacheSubAccountFrom* with a snapshot taken
// under the lock.
func (b *Broker) logBoundSummary(snapshot BrokerStatus) {
	if b.logger == nil {
		return
	}
	fields := []any{
		"walletAddress", snapshot.WalletAddress,
		"subAccountId", snapshot.SubAccountID,
		"subaccountSource", string(snapshot.SubaccountSource),
		"defaultPreset", snapshot.DefaultPreset,
	}
	if snapshot.OwnerAddress != "" && snapshot.SubaccountSource == SubaccountSourceDelegated {
		fields = append(fields, "ownerAddress", snapshot.OwnerAddress)
	}
	if len(snapshot.Permissions) > 0 {
		fields = append(fields, "permissions", strings.Join(snapshot.Permissions, ","))
	}
	if snapshot.ExpiresAtUnix > 0 {
		fields = append(fields,
			"delegationExpiresAtUnix", snapshot.ExpiresAtUnix,
			"delegationExpiresInHours", int64(time.Until(time.Unix(snapshot.ExpiresAtUnix, 0)).Hours()),
		)
	}
	b.logger.Info("agent broker bound to subaccount", fields...)
}

// SignAuthMessage produces (typedDataJSON, signatureHex) for the
// session-binding authenticate tool. Mirrors what
// preview_auth_message + an external wallet would emit so the auth
// manager validates the broker output exactly the same as a wallet-
// signed message.
func (b *Broker) SignAuthMessage(subAccountID int64) (string, string, error) {
	if subAccountID <= 0 {
		return "", "", errors.New("agent broker: subAccountId is required")
	}
	typedData := sdkeip712.BuildAuthMessageWithDomain(
		uint64(subAccountID),
		snx_lib_utils_time.Now().Unix(),
		sdkeip712.ActionWebSocketAuth,
		b.domain(),
	)
	serialized, err := sdkeip712.Serialize(typedData)
	if err != nil {
		return "", "", fmt.Errorf("agent broker: serialize auth typed data: %w", err)
	}
	signatureHex, err := b.signTypedData(typedData)
	if err != nil {
		return "", "", err
	}
	return serialized, signatureHex, nil
}

// domain returns the SDK domain separator the broker signs against.
// Centralised so SignAuthMessage / assertTypedDataAllowed share the
// exact same triple — drift here would surface as silent allowlist
// rejections.
func (b *Broker) domain() apitypes.TypedDataDomain {
	return sdkeip712.Domain(b.domainName, b.domainVersion, b.chainID)
}

// Produces TradeSignature{r,s,v} for the validated action payload.
// Caller must reuse the same nonce/expiresAfter when submitting.
// Rejects actions outside allowedTradeActions, then re-verifies the
// resolved primaryType (defence in depth).
func (b *Broker) SignTradeAction(
	subAccountID int64,
	nonce int64,
	expiresAfter int64,
	action snx_lib_api_types.RequestAction,
	payload any,
) (snx_lib_auth.TradeSignature, error) {
	if subAccountID <= 0 {
		return snx_lib_auth.TradeSignature{}, errors.New("agent broker: subAccountId is required")
	}
	if _, ok := allowedTradeActions[action]; !ok {
		return snx_lib_auth.TradeSignature{}, fmt.Errorf("%w: %q", ErrDisallowedAction, string(action))
	}
	typedData, err := snx_lib_auth.CreateTradeTypedData(
		snx_lib_auth.SubAccountId(strconv.FormatInt(subAccountID, 10)),
		snx_lib_auth.Nonce(nonce),
		expiresAfter,
		action,
		payload,
		b.domainName,
		b.domainVersion,
		b.chainID,
	)
	if err != nil {
		return snx_lib_auth.TradeSignature{}, fmt.Errorf("agent broker: build trade typed data: %w", err)
	}
	if err := b.assertTypedDataAllowed(typedData); err != nil {
		return snx_lib_auth.TradeSignature{}, err
	}
	sig, err := signer.SignTypedDataAndSplit(b.privateKey, typedData)
	if err != nil {
		return snx_lib_auth.TradeSignature{}, fmt.Errorf("agent broker: sign trade typed data: %w", err)
	}
	return toTradeSignature(sig), nil
}

// SignReadAction is the read-only sibling of SignTradeAction. It
// signs an idempotent /v1/trade get* action for the broker's own
// subaccount and returns everything the REST caller needs to rebuild
// the signed envelope: the split signature plus the expiresAfter
// timestamp baked into the typed-data message.
//
// Reads have no replay cost (api-service sets SkipNonceCheck for
// get* actions, per lib/auth/trade_auth.go), so we always sign with
// nonce=0, which in turn drops the nonce field from the typed-data
// message per CreateTradeTypedData's backwards-compatibility rule.
// The resolved primaryType is always "SubAccountAction"; the domain
// allowlist already validates that.
//
// Rejects actions outside allowedReadActions to prevent callers from
// turning this into a generic signing oracle.
func (b *Broker) SignReadAction(
	subAccountID int64,
	action snx_lib_api_types.RequestAction,
) (snx_lib_auth.TradeSignature, int64, error) {
	if subAccountID <= 0 {
		return snx_lib_auth.TradeSignature{}, 0, errors.New("agent broker: subAccountId is required")
	}
	if _, ok := allowedReadActions[action]; !ok {
		return snx_lib_auth.TradeSignature{}, 0, fmt.Errorf("%w: %q", ErrDisallowedAction, string(action))
	}
	// 60s is long enough to absorb realistic clock skew between the
	// broker and api-service without keeping a replayable signature
	// around indefinitely. Reads are idempotent, so the lower bound
	// exists purely as hygiene.
	expiresAfter := snx_lib_utils_time.Now().Unix() + 60
	typedData, err := snx_lib_auth.CreateTradeTypedData(
		snx_lib_auth.SubAccountId(strconv.FormatInt(subAccountID, 10)),
		snx_lib_auth.Nonce(0),
		expiresAfter,
		action,
		map[string]any{},
		b.domainName,
		b.domainVersion,
		b.chainID,
	)
	if err != nil {
		return snx_lib_auth.TradeSignature{}, 0, fmt.Errorf("agent broker: build read typed data: %w", err)
	}
	if err := b.assertTypedDataAllowed(typedData); err != nil {
		return snx_lib_auth.TradeSignature{}, 0, err
	}
	sig, err := signer.SignTypedDataAndSplit(b.privateKey, typedData)
	if err != nil {
		return snx_lib_auth.TradeSignature{}, 0, fmt.Errorf("agent broker: sign read typed data: %w", err)
	}
	return toTradeSignature(sig), expiresAfter, nil
}

// AllocateNonce returns a millisecond-resolution monotonic nonce paired
// with a 24h expiry, matching the convention used by
// preview_trade_signature when the agent does not supply its own.
func (b *Broker) AllocateNonce() (int64, int64) {
	nonce := snx_lib_utils_time.Now().UnixMilli()
	return nonce, nonce + int64(24*time.Hour/time.Millisecond)
}

// Clef-style policy gate: every typed-data payload routes through
// here before crypto.Sign. Domain mismatches are treated as
// allowlist rejections because a foreign domain could be replayed
// against a different contract.
func (b *Broker) assertTypedDataAllowed(typedData apitypes.TypedData) error {
	if typedData.Domain.Name != b.domainName {
		return fmt.Errorf(
			"%w: domain.name=%q expected %q",
			ErrDisallowedPrimaryType, typedData.Domain.Name, b.domainName,
		)
	}
	if typedData.Domain.Version != b.domainVersion {
		return fmt.Errorf(
			"%w: domain.version=%q expected %q",
			ErrDisallowedPrimaryType, typedData.Domain.Version, b.domainVersion,
		)
	}
	wantChain := big.NewInt(int64(b.chainID))
	if typedData.Domain.ChainId == nil ||
		(*big.Int)(typedData.Domain.ChainId).Cmp(wantChain) != 0 {
		return fmt.Errorf(
			"%w: domain.chainId=%v expected %d",
			ErrDisallowedPrimaryType, typedData.Domain.ChainId, b.chainID,
		)
	}
	if _, ok := allowedPrimaryTypes[typedData.PrimaryType]; !ok {
		return fmt.Errorf("%w: %q", ErrDisallowedPrimaryType, typedData.PrimaryType)
	}
	return nil
}

// Hashes and signs typedData, returning a 0x-prefixed 65-byte
// signature with v normalised to {27,28}. Routes through
// assertTypedDataAllowed first.
func (b *Broker) signTypedData(typedData apitypes.TypedData) (string, error) {
	if err := b.assertTypedDataAllowed(typedData); err != nil {
		return "", err
	}
	signatureBytes, err := signer.SignTypedData(b.privateKey, typedData)
	if err != nil {
		return "", fmt.Errorf("agent broker: sign typed data: %w", err)
	}
	return "0x" + common.Bytes2Hex(signatureBytes), nil
}

// toTradeSignature adapts the SDK signature shape to the
// lib/auth.TradeSignature triple the auth manager validates against.
// Field names and types are identical; only the package differs.
func toTradeSignature(sig signer.Signature) snx_lib_auth.TradeSignature {
	return snx_lib_auth.TradeSignature{R: sig.R, S: sig.S, V: sig.V}
}

func loadPrivateKeyHex(inlineHex string, filePath string) (string, error) {
	if v := strings.TrimSpace(inlineHex); v != "" {
		return v, nil
	}
	path := strings.TrimSpace(filePath)
	if path == "" {
		return "", errors.New("agent broker: private key not provided")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("agent broker: read private key file %s: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}
