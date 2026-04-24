package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	"github.com/Fenway-snx/synthetix-mcp/internal/server/backend"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/metrics"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type AuthenticateResult struct {
	Authenticated    bool
	SessionExpiresAt int64
	SubAccountID     int64
	WalletAddress    string
}

type Manager struct {
	accountAuthenticator *snx_lib_auth.AccountAuthenticator
	authCache            *snx_lib_auth.AuthCache
	logger               snx_lib_logging.Logger
	sessionStore         session.Store
	sessionTTL           time.Duration
	ownershipVerifier    OwnershipVerifier

	// Lifecycle for the session-metrics reconciler: cancel stops the
	// loop and Close blocks on done so we never tear down logger or
	// store while a reconcile is mid-flight.
	sessionMetricsCancel context.CancelFunc
	sessionMetricsDone   chan struct{}
}

// Satisfied by *session.MemoryStore.
type sessionCounter interface {
	Count(ctx context.Context) (int, error)
}

// Resync interval for mcp_active_sessions. Small multiple of a
// typical Prometheus scrape so expired sessions surface within a
// scrape or two.
const defaultSessionMetricsReconcileInterval = 30 * time.Second

func NewManager(
	logger snx_lib_logging.Logger,
	cfg *config.Config,
	clients *backend.Clients,
	sessionStore session.Store,
) (*Manager, error) {
	if clients == nil || clients.RESTInfo == nil {
		return nil, errors.New("auth manager requires rest info client")
	}

	authCache := snx_lib_auth.NewAuthCache(cfg.AuthCacheMaxEntries)
	nonceStore := newMemoryNonceStore()

	baseAuthenticator := snx_lib_auth.NewAuthenticator(
		nonceStore,
		nil,
		authCache,
		cfg.EIP712DomainName,
		cfg.EIP712DomainVersion,
		int(cfg.EIP712ChainID),
	)
	lister := restInfoAdapter{
		fetch: func(ctx context.Context, wallet string) ([]string, []string, error) {
			resp, err := clients.RESTInfo.GetSubAccountIdsWithDelegations(ctx, wallet)
			if err != nil {
				return nil, nil, err
			}
			return resp.SubAccountIDs, resp.DelegatedSubAccountIDs, nil
		},
	}
	verifier := newRESTOwnershipVerifier(lister, authCache)
	return newManager(
		logger,
		sessionStore,
		cfg.SessionTTL,
		snx_lib_auth.NewAccountAuthenticator(baseAuthenticator),
		authCache,
		verifier,
	), nil
}

func newManager(
	logger snx_lib_logging.Logger,
	sessionStore session.Store,
	sessionTTLDuration time.Duration,
	accountAuthenticator *snx_lib_auth.AccountAuthenticator,
	authCache *snx_lib_auth.AuthCache,
	ownershipVerifier OwnershipVerifier,
) *Manager {
	m := &Manager{
		accountAuthenticator: accountAuthenticator,
		authCache:            authCache,
		logger:               logger,
		sessionStore:         sessionStore,
		sessionTTL:           sessionTTLDuration,
		ownershipVerifier:    ownershipVerifier,
	}
	m.startSessionMetricsReconciler(defaultSessionMetricsReconcileInterval)
	return m
}

// Periodically resets metrics.ActiveSessions to the live count from
// the session store. Stores that do not implement sessionCounter
// (test fakes) make this a no-op.
func (m *Manager) startSessionMetricsReconciler(interval time.Duration) {
	counter, ok := m.sessionStore.(sessionCounter)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.sessionMetricsCancel = cancel
	m.sessionMetricsDone = make(chan struct{})
	go m.runSessionMetricsReconciler(ctx, counter, interval)
}

func (m *Manager) runSessionMetricsReconciler(
	ctx context.Context,
	counter sessionCounter,
	interval time.Duration,
) {
	defer close(m.sessionMetricsDone)

	m.reconcileSessionMetrics(ctx, counter, interval)

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.reconcileSessionMetrics(ctx, counter, interval)
		}
	}
}

func (m *Manager) reconcileSessionMetrics(
	parent context.Context,
	counter sessionCounter,
	interval time.Duration,
) {
	// Cap each attempt at half the interval so a hung store cannot
	// stall the loop across multiple ticks; the next tick retries
	// fresh.
	ctx, cancel := context.WithTimeout(parent, interval/2)
	defer cancel()
	n, err := counter.Count(ctx)
	if err != nil {
		m.logger.Warn("failed to reconcile mcp_active_sessions gauge from session store", "error", err)
		return
	}
	metrics.ActiveSessions().Set(float64(n))
}

func (m *Manager) Authenticate(
	ctx context.Context,
	sessionID string,
	message string,
	signatureHex string,
) (*AuthenticateResult, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("mcp session ID is required")
	}

	authReq, err := snx_lib_auth.ParseAuthMessage(message, signatureHex)
	if err != nil {
		return nil, fmt.Errorf("parse auth message: %w", err)
	}
	if _, timestamp, err := snx_lib_auth.ExtractAuthData(authReq.TypedData); err != nil {
		return nil, fmt.Errorf("extract auth timestamp: %w", err)
	} else if err := snx_lib_auth.ValidateTimestamp(timestamp); err != nil {
		return nil, fmt.Errorf("timestamp validation failed: %w", err)
	}

	// Pre-recover the wallet and prime the shared AuthCache before
	// ValidateAccountAuth so lib/auth's ownership check inside
	// ValidateAuthentication hits the cache and short-circuits.
	if m.ownershipVerifier != nil {
		subAcctID, _, err := snx_lib_auth.ExtractAuthData(authReq.TypedData)
		if err != nil {
			return nil, fmt.Errorf("extract subaccount id for ownership prime: %w", err)
		}
		recovered, err := snx_lib_auth.VerifyEIP712Signature(authReq.TypedData, authReq.Signature)
		if err != nil {
			return nil, fmt.Errorf("recover wallet for ownership prime: %w", err)
		}
		if err := m.ownershipVerifier.VerifyOwnership(ctx, recovered.Hex(), int64(subAcctID)); err != nil {
			return nil, fmt.Errorf("verify ownership: %w", err)
		}
	}

	authResult, err := m.accountAuthenticator.ValidateAccountAuth(
		authReq.TypedData,
		authReq.Signature,
		mcpAuthExtractor,
		&snx_lib_auth.AuthOptions{
			SupportExpiration: false,
			UseTimestampNonce: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if authResult == nil || !authResult.Valid {
		return nil, fmt.Errorf("invalid session authentication")
	}

	now := snx_lib_utils_time.Now()
	expiresAt := now.Add(m.sessionTTL).UnixMilli()
	state := &session.State{
		AuthMode:       session.AuthModeAuthenticated,
		CreatedAt:      now.UnixMilli(),
		ExpiresAt:      expiresAt,
		LastActivityAt: now.UnixMilli(),
		SubAccountID:   int64(authResult.SubAccountId),
		WalletAddress:  authResult.EthereumAddress,
	}

	// Gating Inc on key-existed-before-save mirrors the reconciler
	// invariant: it counts store keys, not auth transitions.
	keyExistedBeforeSave := false
	switch _, err := m.sessionStore.Get(ctx, sessionID); {
	case err == nil:
		keyExistedBeforeSave = true
	case errors.Is(err, session.ErrSessionNotFound):
	default:
		return nil, fmt.Errorf("load session before authenticate: %w", err)
	}

	if err := m.sessionStore.Save(ctx, sessionID, state, m.sessionTTL); err != nil {
		return nil, fmt.Errorf("save authenticated session: %w", err)
	}
	metrics.SessionEventsTotal("authenticated").Inc()
	if !keyExistedBeforeSave {
		metrics.ActiveSessions().Inc()
	}

	return &AuthenticateResult{
		Authenticated:    true,
		SessionExpiresAt: expiresAt,
		SubAccountID:     int64(authResult.SubAccountId),
		WalletAddress:    authResult.EthereumAddress,
	}, nil
}

func (m *Manager) ValidateTradeAction(
	sessionWalletAddress string,
	sessionSubAccountID int64,
	nonce int64,
	expiresAfter int64,
	action snx_lib_api_types.RequestAction,
	payload any,
	signature snx_lib_auth.TradeSignature,
) error {
	if m == nil || m.accountAuthenticator == nil {
		return fmt.Errorf("trade authenticator unavailable")
	}
	if sessionSubAccountID <= 0 {
		return fmt.Errorf("authenticated subaccount is required")
	}

	if m.ownershipVerifier != nil && sessionWalletAddress != "" {
		if err := m.ownershipVerifier.VerifyOwnership(
			context.Background(),
			sessionWalletAddress,
			sessionSubAccountID,
		); err != nil {
			return fmt.Errorf("verify trade-action ownership: %w", err)
		}
	}

	authResult, err := snx_lib_auth.ValidateTradeActionSignature(
		m.accountAuthenticator,
		snx_lib_auth.AuthConfig{
			DomainName:    m.accountAuthenticator.DomainName(),
			DomainVersion: m.accountAuthenticator.DomainVersion(),
			ChainID:       m.accountAuthenticator.ChainID(),
		},
		snx_lib_auth.SubAccountId(strconv.FormatInt(sessionSubAccountID, 10)),
		snx_lib_auth.Nonce(nonce),
		expiresAfter,
		action,
		payload,
		snx_lib_auth.BuildSignatureHex(signature),
		&snx_lib_auth.AuthOptions{
			RequireOwner:      action.RequiresOwner(),
			SupportExpiration: true,
			UseTimestampNonce: false,
		},
	)
	if err != nil {
		return fmt.Errorf("validate trade action signature: %w", err)
	}
	if authResult == nil || !authResult.Valid {
		return fmt.Errorf("invalid trade action signature")
	}
	if int64(authResult.SubAccountId) != sessionSubAccountID {
		return fmt.Errorf("signed subaccount does not match authenticated session")
	}
	if sessionWalletAddress != "" && authResult.EthereumAddress != sessionWalletAddress {
		return fmt.Errorf("signed wallet does not match authenticated session")
	}

	return nil
}

func (m *Manager) VerifySessionAccess(ctx context.Context, walletAddress string, subAccountID int64) error {
	if m == nil || m.accountAuthenticator == nil {
		return fmt.Errorf("trade authenticator unavailable")
	}
	if walletAddress == "" {
		return fmt.Errorf("authenticated wallet is missing from session state")
	}
	if subAccountID <= 0 {
		return fmt.Errorf("authenticated subaccount is required")
	}

	if m.ownershipVerifier != nil {
		if err := m.ownershipVerifier.VerifyOwnership(ctx, walletAddress, subAccountID); err != nil {
			return fmt.Errorf("authenticated wallet is no longer authorized for this subaccount: %w", err)
		}
		return nil
	}

	owns, err := m.accountAuthenticator.Authenticator.VerifyAccountOwnership(
		snx_lib_api_types.WalletAddress(walletAddress),
		snx_lib_core.SubAccountId(subAccountID),
	)
	if err != nil {
		return fmt.Errorf("authenticated wallet is no longer authorized for this subaccount: %w", err)
	}
	if !owns {
		return fmt.Errorf("authenticated wallet is no longer authorized for this subaccount")
	}

	return nil
}

func (m *Manager) Close() error {
	if m.sessionMetricsCancel != nil {
		m.sessionMetricsCancel()
		<-m.sessionMetricsDone
	}
	return nil
}

// DomainName returns the configured EIP-712 domain name so preview
// tools can build typed-data that matches what this server validates.
func (m *Manager) DomainName() string {
	if m == nil || m.accountAuthenticator == nil {
		return ""
	}
	return m.accountAuthenticator.DomainName()
}

func (m *Manager) DomainVersion() string {
	if m == nil || m.accountAuthenticator == nil {
		return ""
	}
	return m.accountAuthenticator.DomainVersion()
}

func (m *Manager) ChainID() int {
	if m == nil || m.accountAuthenticator == nil {
		return 0
	}
	return m.accountAuthenticator.ChainID()
}

func mcpAuthExtractor(typedData apitypes.TypedData) (
	snx_lib_core.SubAccountId,
	snx_lib_auth.Nonce,
	int64,
	error,
) {
	subAccountID, timestamp, err := snx_lib_auth.ExtractAuthData(typedData)
	if err != nil {
		return 0, 0, 0, err
	}
	return subAccountID, snx_lib_auth.Nonce(timestamp), 0, nil
}
