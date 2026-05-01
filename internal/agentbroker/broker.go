// Package agentbroker provides an optional single-tenant EIP-712 signer.
// The private key stays in process memory and powers one-call broker tools.
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

// Mirrors the auth result fields needed here without creating an import cycle.
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

// Closed set of EIP-712 primary types the broker may sign.
// New write types must pass both action and primary-type gates.
var allowedPrimaryTypes = map[string]struct{}{
	"AddDelegatedSigner":        {},
	"AuthMessage":               {},
	"CancelAllOrders":           {},
	"CancelOrders":              {},
	"CancelOrdersByCloid":       {},
	"ModifyOrder":               {},
	"ModifyOrderByCloid":        {},
	"PlaceOrders":               {},
	"RemoveAllDelegatedSigners": {},
	"RemoveDelegatedSigner":     {},
	"ScheduleCancel":            {},
	// Read actions carry no nonce and cannot move collateral or orders.
	"SubAccountAction":   {},
	"TransferCollateral": {},
	"UpdateLeverage":     {},
	"WithdrawCollateral": {},
}

// Per-action gate checked before typed-data construction.
var allowedTradeActions = map[snx_lib_api_types.RequestAction]struct{}{
	snx_lib_api_types.RequestAction("addDelegatedSigner"):        {},
	snx_lib_api_types.RequestAction("cancelAllOrders"):           {},
	snx_lib_api_types.RequestAction("cancelOrders"):              {},
	snx_lib_api_types.RequestAction("modifyOrder"):               {},
	snx_lib_api_types.RequestAction("placeOrders"):               {},
	snx_lib_api_types.RequestAction("removeAllDelegatedSigners"): {},
	snx_lib_api_types.RequestAction("removeDelegatedSigner"):     {},
	snx_lib_api_types.RequestAction("scheduleCancel"):            {},
	snx_lib_api_types.RequestAction("transferCollateral"):        {},
	snx_lib_api_types.RequestAction("updateLeverage"):            {},
	snx_lib_api_types.RequestAction("withdrawCollateral"):        {},
}

// Per-action gate checked before signing idempotent reads.
// Each action maps to an expected proxied trade-read endpoint.
var allowedReadActions = map[snx_lib_api_types.RequestAction]struct{}{
	snx_lib_api_types.RequestAction("getSubAccount"):             {},
	snx_lib_api_types.RequestAction("getSubAccounts"):            {},
	snx_lib_api_types.RequestAction("getOpenOrders"):             {},
	snx_lib_api_types.RequestAction("getPositions"):              {},
	snx_lib_api_types.RequestAction("getOrderHistory"):           {},
	snx_lib_api_types.RequestAction("getTrades"):                 {},
	snx_lib_api_types.RequestAction("getFundingPayments"):        {},
	snx_lib_api_types.RequestAction("getPerformanceHistory"):     {},
	snx_lib_api_types.RequestAction("getBalanceUpdates"):         {},
	snx_lib_api_types.RequestAction("getDelegatedSigners"):       {},
	snx_lib_api_types.RequestAction("getDelegationsForDelegate"): {},
	snx_lib_api_types.RequestAction("getFees"):                   {},
	snx_lib_api_types.RequestAction("getPortfolio"):              {},
	snx_lib_api_types.RequestAction("getPositionHistory"):        {},
	snx_lib_api_types.RequestAction("getRateLimits"):             {},
	snx_lib_api_types.RequestAction("getTradesForPosition"):      {},
	snx_lib_api_types.RequestAction("getTransfers"):              {},
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

// Default guardrails for freshly auto-authenticated sessions.
type GuardrailDefaults struct {
	AllowedOrderTypes   []string
	AllowedSymbols      []string
	MaxOrderNotional    string
	MaxOrderQuantity    string
	MaxPositionNotional string
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

// Holds the in-process private key and cached subaccount binding.
type Broker struct {
	chainID          int
	defaults         GuardrailDefaults
	domainName       string
	domainVersion    string
	logger           snx_lib_logging.Logger
	privateKey       *ecdsa.PrivateKey
	subaccountClient SubaccountResolver
	walletAddress    common.Address

	mu           sync.Mutex
	status       BrokerStatus
	statusLogged bool
	subAccountID int64
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

// Publishable snapshot of broker posture.
// Static fields populate at startup; binding fields populate after discovery.
type BrokerStatus struct {
	ChainID          int              `json:"chainId,omitempty"`
	DefaultPreset    string           `json:"defaultPreset,omitempty"`
	DelegationID     uint64           `json:"delegationId,omitempty"`
	DomainName       string           `json:"domainName,omitempty"`
	DomainVersion    string           `json:"domainVersion,omitempty"`
	ExpiresAtUnix    int64            `json:"expiresAtUnix,omitempty"`
	OwnerAddress     string           `json:"ownerAddress,omitempty"`
	Permissions      []string         `json:"permissions,omitempty"`
	SubAccountID     int64            `json:"subAccountId,omitempty"`
	SubaccountSource SubaccountSource `json:"subaccountSource,omitempty"`
	WalletAddress    string           `json:"walletAddress,omitempty"`
}

// Parses signing config, validates the key, and prepares lazy discovery.
// Bad keys fail at startup rather than on the first agent call.
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
		// Default to the writable preset so broker writes work immediately.
		defaults.Preset = "standard"
	}
	if len(defaults.AllowedSymbols) == 0 {
		defaults.AllowedSymbols = []string{"*"}
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

// Returns the EIP-55 checksum address derived from the broker key.
func (b *Broker) WalletAddress() common.Address {
	return b.walletAddress
}

// GuardrailDefaults exposes the configured defaults so the session
// auto-bootstrap path can populate the session state with them.
func (b *Broker) GuardrailDefaults() GuardrailDefaults {
	return b.defaults
}

// Returns the cached subaccount, or zero before discovery.
func (b *Broker) SubAccountID() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.subAccountID
}

// Returns a usable subaccount ID, looking it up on first call.
// Owned subaccounts are preferred, then delegated grants.
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

// Returns a detached copy safe to JSON-encode or log.
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

// Produces typed-data JSON and signature hex for session binding.
// The output matches what an external wallet path would emit.
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

// Returns the SDK domain separator used for broker signatures.
func (b *Broker) domain() apitypes.TypedDataDomain {
	return sdkeip712.Domain(b.domainName, b.domainVersion, b.chainID)
}

// Signs a validated write action and returns split signature fields.
// Callers must reuse the same nonce and expiry when submitting.
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

// Signs an idempotent trade-read action for the broker subaccount.
// Read signatures use nonce 0 and still pass action allowlisting.
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
	// Short expiry absorbs clock skew without keeping signatures around.
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

// Returns a millisecond nonce paired with a 24-hour expiry.
func (b *Broker) AllocateNonce() (int64, int64) {
	nonce := snx_lib_utils_time.Now().UnixMilli()
	return nonce, nonce + int64(24*time.Hour/time.Millisecond)
}

// Policy gate checked before any typed-data signature is produced.
// Domain mismatches are treated as allowlist rejections.
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
