package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shopspring/decimal"

	"github.com/Fenway-snx/synthetix-mcp/internal/agentbroker"
	internal_auth "github.com/Fenway-snx/synthetix-mcp/internal/auth"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	"github.com/Fenway-snx/synthetix-mcp/internal/notifications/tradeclosed"
	"github.com/Fenway-snx/synthetix-mcp/internal/prompts"
	"github.com/Fenway-snx/synthetix-mcp/internal/resources"
	"github.com/Fenway-snx/synthetix-mcp/internal/risksnapshot"
	"github.com/Fenway-snx/synthetix-mcp/internal/server/backend"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/streaming"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
	"github.com/synthetixio/synthetix-go/restinfo"
	backend_types "github.com/synthetixio/synthetix-go/types"
)

type healthResponse struct {
	Error   string `json:"error,omitempty"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

type Server struct {
	authManager sessionAuthManager
	clients     readyCloser
	cfg         *config.Config
	ErrorChan   chan error
	httpServer  *http.Server
	logger      snx_lib_logging.Logger
	mcpServer   *mcp.Server
	streaming   closeOnly
	tradeClosed *tradeclosed.Service
}

type readyCloser interface {
	Close() error
	Ready(context.Context) error
}

type closeOnly interface {
	Close() error
}

type sessionAuthManager interface {
	closeOnly
	tools.SessionAccessVerifier
	tools.SessionAuthenticator
	ValidateTradeAction(
		sessionWalletAddress string,
		sessionSubAccountID int64,
		nonce int64,
		expiresAfter int64,
		action snx_lib_api_types.RequestAction,
		payload any,
		signature snx_lib_auth.TradeSignature,
	) error
}

func New(
	logger snx_lib_logging.Logger,
	cfg *config.Config,
) (*Server, error) {
	clients, err := backend.NewClients(logger, cfg)
	if err != nil {
		return nil, err
	}

	sessionStore, err := newSessionStore(logger, cfg)
	if err != nil {
		return nil, errors.Join(err, clients.Close())
	}
	authManager, err := internal_auth.NewManager(logger, cfg, clients, sessionStore)
	if err != nil {
		return nil, errors.Join(err, clients.Close())
	}
	// Snapshot manager is needed early for the eventStore callback;
	// the hydration client is injected once tradeReads exists.
	riskSnapshotManager := risksnapshot.NewManager(nil)
	riskSnapshotManager.SetMaxSnapshotAge(cfg.RiskSnapshotMaxAge)
	streamingManager, err := streaming.NewManager(logger, cfg, clients.WSInfo)
	if err != nil {
		return nil, errors.Join(err, authManager.Close(), clients.Close())
	}
	publicSessions := tools.NewPublicSessionTracker()
	eventStore := streaming.NewEventStore(func(sessionID string) {
		streamingManager.SessionClosed(sessionID)
		riskSnapshotManager.SessionClosed(sessionID)
		publicSessions.Forget(sessionID)
		state, err := sessionStore.Get(context.Background(), sessionID)
		if err != nil || state == nil || state.AuthMode != session.AuthModeAuthenticated {
			return
		}
		_, _ = sessionStore.DeleteIfExists(context.Background(), sessionID)
	})
	srv := newServer(logger, cfg, clients, authManager, streamingManager, eventStore)
	toolDeps := &tools.ToolDeps{
		Cfg:            cfg,
		Clients:        clients,
		PublicSessions: publicSessions,
		Store:          sessionStore,
		Verifier:       authManager,
		SnapshotManager: riskSnapshotManager,
	}
	tools.RegisterSessionTools(srv.mcpServer, toolDeps, authManager, streamingManager)
	tools.RegisterSessionStateTools(srv.mcpServer, toolDeps, streamingManager)
	tools.RegisterPublicTools(srv.mcpServer, toolDeps)
	tools.RegisterSystemTools(srv.mcpServer, toolDeps)

	// Broker is built before trade-tool registration so the public tool
	// surface exposes exactly one write path for the active mode.
	var (
		broker     *agentbroker.Broker
		tradeReads *tools.TradeReadClient
	)
	if cfg.AgentBroker.Enabled {
		b, brokerErr := buildAgentBroker(logger, cfg, authManager, clients)
		if brokerErr != nil {
			return nil, errors.Join(fmt.Errorf("init agent broker: %w", brokerErr), streamingManager.Close(), authManager.Close(), clients.Close())
		}
		broker = b
		toolDeps.BrokerStatus = brokerStatusAdapter{broker: broker}
		if clients.RESTTrade != nil {
			tradeReads = tools.NewTradeReadClient(
				clients.RESTTrade,
				brokerReadSignerAdapter{broker: broker},
				brokerTradeSignerAdapter{broker: broker},
				authManager,
				logger,
			)
		}
	}

	riskSnapshotManager.SetHydrationClient(risksnapshotHydrationAdapter{tradeReads: tradeReads})

	// Trade-closed notifications: detect nonzero→zero position
	// transitions in risksnapshot and push a "position.closed" event
	// over the bound MCP session connection. The service owns its
	// own goroutine; Stop is called from Server.Close.
	tradeClosedService := tradeclosed.Wire(riskSnapshotManager, streamingManager, logger)
	srv.tradeClosed = tradeClosedService

	tools.RegisterTradingTools(srv.mcpServer, toolDeps, restMarketReaderAdapter{rest: clients.RESTInfo}, tradeReads, riskSnapshotManager, authManager, broker == nil)
	tools.RegisterSignaturePreviewTools(srv.mcpServer, toolDeps, authManager, tradeReads, broker == nil)
	if broker == nil {
		tools.RegisterLifecycleTools(srv.mcpServer, toolDeps, authManager, tradeReads)
	}
	if broker != nil {
		tools.RegisterBrokerTools(
			srv.mcpServer, toolDeps, broker, authManager, tradeReads,
			riskSnapshotManager, restMarketReaderAdapter{rest: clients.RESTInfo},
		)
		tools.RegisterLifecycleBrokerTools(srv.mcpServer, toolDeps, broker, authManager, tradeReads)
		logger.Info(
			"agent broker tools registered",
			"walletAddress", broker.WalletAddress().Hex(),
			"defaultPreset", broker.GuardrailDefaults().Preset,
		)
	}
	tools.RegisterContextTools(srv.mcpServer, toolDeps, streamingManager, tradeReads)
	tools.RegisterAccountTools(srv.mcpServer, toolDeps, tradeReads)
	tools.RegisterAccountExtraTools(srv.mcpServer, toolDeps, tradeReads)
	tools.RegisterStreamingTools(srv.mcpServer, toolDeps, streamingManager)
	resources.Register(srv.mcpServer, toolDeps, tradeReads)
	prompts.Register(srv.mcpServer, cfg.AgentBroker.Enabled)

	return srv, nil
}

func newSessionStore(logger snx_lib_logging.Logger, cfg *config.Config) (session.Store, error) {
	if cfg.SessionStorePath == "" {
		logger.Info("session store: in-memory (sessions lost on restart)")
		return session.NewMemoryStore(), nil
	}
	store, err := session.NewFileStore(cfg.SessionStorePath)
	if err != nil {
		return nil, fmt.Errorf("init file-backed session store: %w", err)
	}
	logger.Info("session store: file-backed", "path", cfg.SessionStorePath)
	return store, nil
}

func newServer(
	logger snx_lib_logging.Logger,
	cfg *config.Config,
	clients readyCloser,
	authManager sessionAuthManager,
	streamingManager closeOnly,
	eventStore mcp.EventStore,
) *Server {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.ServerName,
		Version: cfg.ServerVersion,
	}, &mcp.ServerOptions{
		Instructions: buildServerInstructions(cfg.AgentBroker.Enabled),
	})
	var bindSession func(string, mcp.Connection)
	if binder, ok := streamingManager.(interface {
		BindSession(string, mcp.Connection)
	}); ok {
		bindSession = binder.BindSession
	}

	mcpHandler := newStreamableSessionHandler(logger, func(*http.Request) *mcp.Server {
		return mcpServer
	}, bindSession, &mcp.StreamableHTTPOptions{
		EventStore:     eventStore,
		SessionTimeout: cfg.SessionTTL,
	})
	s := &Server{
		authManager: authManager,
		clients:     clients,
		cfg:         cfg,
		ErrorChan:   make(chan error, 1),
		logger:      logger,
		mcpServer:   mcpServer,
		streaming:   streamingManager,
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", http.MaxBytesHandler(mcpHandler, cfg.MaxRequestBodyBytes))
	mux.HandleFunc("/health", s.healthCheck)
	mux.HandleFunc("/health/live", s.livenessCheck)
	mux.HandleFunc("/health/ready", s.readinessCheck)
	// Catch-all for unmatched paths. MCP clients (notably Claude Code) probe
	// a wide set of OAuth endpoints during connection setup -- not only
	// /.well-known/* metadata, but also RFC 7591 dynamic client registration
	// (/register) and the legacy RFC 6749 endpoints (/authorize, /token) --
	// and their SDKs JSON.parse() any non-2xx response body. Routing every
	// unknown path through handleWellKnownNotFound guarantees they receive
	// the same JSON 404 envelope (with the Mcp-Auth advisory header and
	// guidance toward `authenticate` / `preview_auth_message` / `quickstart`)
	// instead of Go's default plain-text "404 page not found" body, which
	// crashes their OAuth code path.
	mux.HandleFunc("/", s.handleWellKnownNotFound)

	s.httpServer = &http.Server{
		Addr:              cfg.ServerAddress,
		Handler:           mux,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	return s
}

func (s *Server) Start() error {
	if err := s.refreshReadiness(context.Background()); err != nil {
		s.logger.Warn("MCP server started in unhealthy state", "error", err)
	}

	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.httpServer.Addr, err)
	}

	go func() {
		s.logger.Info("Starting MCP HTTP server", "addr", s.httpServer.Addr)
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.ErrorChan <- err
		}
	}()

	return nil
}

func (s *Server) Errors() <-chan error {
	return s.ErrorChan
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownErr := s.httpServer.Shutdown(ctx)
	authErr := s.closeAuthManager()
	streamErr := s.closeStreaming()
	closeErr := s.closeClients()
	if s.tradeClosed != nil {
		s.tradeClosed.Stop()
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	if authErr != nil {
		return authErr
	}
	if streamErr != nil {
		return streamErr
	}
	return closeErr
}

// MCP clients (Claude Desktop, Claude Code, various SDKs) probe RFC 8414 /
// RFC 9728 / OIDC discovery endpoints under /.well-known/ during connection
// setup. This server implements MCP's tool-level EIP-712 authentication
// instead of OAuth 2.0, so there is no authorization server to advertise.
//
// We return a JSON 404 (not Go's default plain-text 404, which at least one
// popular SDK mis-parses as "Invalid OAuth error response"). The body
// explicitly names our auth model so operators debugging a stuck client can
// tell at a glance what the server expects. Per the MCP spec clients should
// then fall through and use the server's advertised tools (`authenticate` +
// `preview_auth_message`) for authentication.
//
// We also emit a debug log so operators can see which client versions are
// still chasing OAuth discovery (a common cause of "SDK auth failed" errors
// from clients with stale OAuth state).
func (s *Server) handleWellKnownNotFound(w http.ResponseWriter, r *http.Request) {
	if s.logger != nil {
		s.logger.Debug(
			"MCP client probed OAuth discovery endpoint; this server uses tool-level EIP-712 auth",
			"path", r.URL.Path,
			"method", r.Method,
			"user_agent", r.Header.Get("User-Agent"),
		)
	}
	w.Header().Set("Content-Type", "application/json")
	// Advisory header: clients that honor it can short-circuit OAuth probing.
	// No standardized header exists for this yet; we pick a namespaced name
	// ("Mcp-Auth") consistent with the Mcp-Session-Id convention.
	w.Header().Set("Mcp-Auth", "tool:authenticate")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(
		`{"error":"not_found",` +
			`"error_description":"This server does not support OAuth 2.0. Authenticate via the MCP tool 'authenticate' using an EIP-712 signature. See tools 'preview_auth_message' and 'authenticate', or prompt 'quickstart'.",` +
			`"auth_method":"mcp_tool",` +
			`"auth_tool":"authenticate"}`,
	))
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.refreshReadiness(r.Context()); err != nil {
		s.logger.Warn("MCP health check failed", "error", err)
		s.writeJSON(w, http.StatusServiceUnavailable, healthResponse{
			Error:   publicHealthError(err),
			Status:  "unhealthy",
			Version: s.cfg.ServerVersion,
		})
		return
	}

	s.writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: s.cfg.ServerVersion,
	})
}

func (s *Server) livenessCheck(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, healthResponse{
		Status:  "live",
		Version: s.cfg.ServerVersion,
	})
}

func (s *Server) readinessCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.refreshReadiness(r.Context()); err != nil {
		s.logger.Warn("MCP readiness check failed", "error", err)
		s.writeJSON(w, http.StatusServiceUnavailable, healthResponse{
			Error:   publicHealthError(err),
			Status:  "not_ready",
			Version: s.cfg.ServerVersion,
		})
		return
	}

	s.writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ready",
		Version: s.cfg.ServerVersion,
	})
}

func (s *Server) refreshReadiness(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, s.cfg.APIHTTPTimeout)
	defer cancel()

	if err := s.readyClients(checkCtx); err != nil {
		return err
	}

	return nil
}

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, payload healthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Error("Failed to encode HTTP response", "error", err)
	}
}

func publicHealthError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "dependency readiness check timed out"
	}
	return "dependency readiness check failed"
}

// Renders the MCP `Instructions` system prompt. Bakes in two
// non-negotiable rules: never ask a human to paste a signature, and
// prefer canonical broker tools when the broker is enabled (only advertised when
// the tools are actually registered).
func buildServerInstructions(brokerEnabled bool) string {
	const base = "Synthetix MCP service is online. Start with get_context " +
		"or read system://routing-guide and system://agent-guide to orient " +
		"to the current session, markets, and trading workflows. Guardrails " +
		"are optional operator limits; standard guardrails are present when " +
		"configured and otherwise default to unrestricted symbols with normal " +
		"order types."
	const quickstartRule = " When a user asks where to start, suggest running " +
		"the quickstart prompt with a symbol like BTC-USDT. For market-data-only " +
		"setup, call ping, get_server_info, then list_markets before asking " +
		"for any trading/authentication setup."
	const confirmationRule = " Ask for confirmation at most once per trade " +
		"or operation. Combine order details, account context, and guardrails " +
		"into that single confirmation; do not ask separately to approve " +
		"guardrails and then again to approve the order."
	const noPasteRule = " IMPORTANT: never ask a human user to paste an " +
		"EIP-712 signature, hex digest, or private key into the chat. " +
		"Wallet signing is a privileged operation; an LLM agent must " +
		"either sign locally with a key it already holds or use the " +
		"server-side broker tools."
	const brokenConnectionRule = " If a tool call fails with 'unknown tool', " +
		"the MCP connection is broken. Stop, do not fall back to Bash, " +
		"and tell the user to restart Claude Code and re-run claude mcp list."
	if brokerEnabled {
		return base + quickstartRule + confirmationRule + noPasteRule + brokenConnectionRule + " The agent broker is enabled on this " +
			"server: prefer place_order, close_position, " +
			"cancel_order, and cancel_all_orders, which sign " +
			"and submit in one call with no client-side cryptography."
	}
	return base + quickstartRule + confirmationRule + noPasteRule + brokenConnectionRule + " The agent broker is disabled on this " +
		"server. Claude cannot sign EIP-712 payloads by itself; ask the " +
		"operator to run sample/node-scripts/authenticate-external-wallet.mjs " +
		"or another local sidecar signer against this MCP session ID. If no " +
		"sidecar signer is available, refuse the trade and ask the operator " +
		"to enable the broker (SNXMCP_AGENT_BROKER_ENABLED=true plus a private key)."
}

// Bridges *agentbroker.Broker.Status() to tools.BrokerStatusSnapshot.
// Exists solely to keep agentbroker out of the tools package import
// graph; collapse both with a direct embed if those packages merge.
type brokerStatusAdapter struct {
	broker *agentbroker.Broker
}

// Forwards the broker's read-signing surface to the narrow
// interface consumed by tools.
type brokerReadSignerAdapter struct {
	broker *agentbroker.Broker
}

func (a brokerReadSignerAdapter) SignReadAction(subAccountID int64, action snx_lib_api_types.RequestAction) (snx_lib_auth.TradeSignature, int64, error) {
	return a.broker.SignReadAction(subAccountID, action)
}

func (a brokerReadSignerAdapter) WalletAddress() string {
	return a.broker.WalletAddress().Hex()
}

func (a brokerReadSignerAdapter) SubAccountID() int64 {
	return a.broker.SubAccountID()
}

// Write-side counterpart of the read adapter.
type brokerTradeSignerAdapter struct {
	broker *agentbroker.Broker
}

func (a brokerTradeSignerAdapter) SignTradeAction(
	subAccountID int64,
	nonce int64,
	expiresAfter int64,
	action snx_lib_api_types.RequestAction,
	payload any,
) (snx_lib_auth.TradeSignature, error) {
	return a.broker.SignTradeAction(subAccountID, nonce, expiresAfter, action, payload)
}

func (a brokerTradeSignerAdapter) AllocateNonce() (int64, int64) {
	return a.broker.AllocateNonce()
}

func (a brokerTradeSignerAdapter) WalletAddress() string {
	return a.broker.WalletAddress().Hex()
}

func (a brokerTradeSignerAdapter) SubAccountID() int64 {
	return a.broker.SubAccountID()
}

// Bridges the hydration interface to the REST trade-reads shim.
// Reads are broker-signed; the minted ToolContext carries only the
// subaccount.
type risksnapshotHydrationAdapter struct {
	tradeReads *tools.TradeReadClient
}

func (a risksnapshotHydrationAdapter) GetOpenOrders(ctx context.Context, subAccountID int64, _ int, offset int) ([]risksnapshot.HydrationOrder, error) {
	// REST doesn't expose cursor-style pagination; the first page is
	// complete, so return empty on offset > 0 to terminate the loop.
	if a.tradeReads == nil {
		return nil, fmt.Errorf("trade reads unavailable")
	}
	if offset > 0 {
		return nil, nil
	}
	items, err := a.tradeReads.GetOpenOrders(ctx, tools.ToolContext{State: &session.State{SubAccountID: subAccountID}})
	if err != nil {
		return nil, err
	}
	out := make([]risksnapshot.HydrationOrder, 0, len(items))
	for _, item := range items {
		out = append(out, risksnapshot.HydrationOrder{
			ClientOrderID:     item.Order.ClientID,
			OrderType:         item.Type,
			Price:             item.Price,
			Quantity:          item.Quantity,
			ReduceOnly:        item.ReduceOnly,
			RemainingQuantity: remainingQuantityFromFilled(item.Quantity, item.FilledQuantity),
			Side:              item.Side,
			Symbol:            item.Symbol,
			VenueOrderID:      item.Order.VenueID,
		})
	}
	return out, nil
}

func (a risksnapshotHydrationAdapter) GetPositions(ctx context.Context, subAccountID int64, _ int, offset int) ([]risksnapshot.HydrationPosition, error) {
	if a.tradeReads == nil {
		return nil, fmt.Errorf("trade reads unavailable")
	}
	if offset > 0 {
		return nil, nil
	}
	items, err := a.tradeReads.GetPositions(ctx, tools.ToolContext{State: &session.State{SubAccountID: subAccountID}})
	if err != nil {
		return nil, err
	}
	out := make([]risksnapshot.HydrationPosition, 0, len(items))
	for _, item := range items {
		out = append(out, risksnapshot.HydrationPosition{
			Quantity: item.Quantity,
			Side:     item.Side,
			Symbol:   item.Symbol,
		})
	}
	return out, nil
}

// Resolves broker subaccount discovery through /v1/info. Owner,
// permissions, and expiry are not available on the REST path.
type restSubaccountResolverAdapter struct {
	rest *restinfo.Client
}

func (a restSubaccountResolverAdapter) GetSubAccountIdsWithDelegations(ctx context.Context, wallet string) (*agentbroker.ResolvedSubAccounts, error) {
	if a.rest == nil {
		return nil, fmt.Errorf("rest info client unavailable")
	}
	resp, err := a.rest.GetSubAccountIdsWithDelegations(ctx, wallet)
	if err != nil {
		return nil, err
	}
	owned, err := parseInt64Slice(resp.SubAccountIDs)
	if err != nil {
		return nil, fmt.Errorf("parse owned subaccount ids: %w", err)
	}
	delegated, err := parseInt64Slice(resp.DelegatedSubAccountIDs)
	if err != nil {
		return nil, fmt.Errorf("parse delegated subaccount ids: %w", err)
	}
	return &agentbroker.ResolvedSubAccounts{Owned: owned, Delegated: delegated}, nil
}

// Bridges the market-config reader to /v1/info.
type restMarketReaderAdapter struct {
	rest *restinfo.Client
}

func (a restMarketReaderAdapter) GetMarket(ctx context.Context, symbol string) (*backend_types.MarketResponse, error) {
	if a.rest == nil {
		return nil, fmt.Errorf("rest info client unavailable")
	}
	return a.rest.GetMarket(ctx, symbol)
}

func (a restMarketReaderAdapter) GetMarketPrices(ctx context.Context) (map[string]backend_types.MarketPriceResponse, error) {
	if a.rest == nil {
		return nil, fmt.Errorf("rest info client unavailable")
	}
	return a.rest.GetMarketPrices(ctx)
}

func parseInt64Slice(in []string) ([]int64, error) {
	out := make([]int64, 0, len(in))
	for _, raw := range in {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// Returns quantity - filled as a decimal string; falls back to
// quantity when either side fails to parse.
func remainingQuantityFromFilled(quantity, filled string) string {
	if strings.TrimSpace(quantity) == "" {
		return quantity
	}
	if strings.TrimSpace(filled) == "" {
		return quantity
	}
	q, qerr := decimal.NewFromString(quantity)
	f, ferr := decimal.NewFromString(filled)
	if qerr != nil || ferr != nil {
		return quantity
	}
	return q.Sub(f).String()
}

func (a brokerStatusAdapter) Status() tools.BrokerStatusSnapshot {
	s := a.broker.Status()
	return tools.BrokerStatusSnapshot{
		ChainID:          s.ChainID,
		DefaultPreset:    s.DefaultPreset,
		DelegationID:     s.DelegationID,
		DomainName:       s.DomainName,
		DomainVersion:    s.DomainVersion,
		ExpiresAtUnix:    s.ExpiresAtUnix,
		OwnerAddress:     s.OwnerAddress,
		Permissions:      s.Permissions,
		SubAccountID:     s.SubAccountID,
		SubaccountSource: string(s.SubaccountSource),
		WalletAddress:    s.WalletAddress,
	}
}

// buildAgentBroker materialises the in-process EIP-712 signer described
// in services/mcp/internal/agentbroker. The auth manager is wired in as
// the DomainProvider so the broker signs against the same chain ID and
// EIP-712 domain that the manager validates against — divergence here
// would cause every broker-signed write to fail with INVALID_SIGNATURE.
func buildAgentBroker(
	logger snx_lib_logging.Logger,
	cfg *config.Config,
	authManager *internal_auth.Manager,
	clients *backend.Clients,
) (*agentbroker.Broker, error) {
	return agentbroker.New(agentbroker.Options{
		DomainProvider:     authManager,
		Logger:             logger,
		PrivateKeyHex:      cfg.AgentBroker.PrivateKeyHex,
		PrivateKeyFile:     cfg.AgentBroker.PrivateKeyFile,
		SubAccountID:       cfg.AgentBroker.SubAccountID,
		SubaccountResolver: restSubaccountResolverAdapter{rest: clients.RESTInfo},
		GuardrailDefaults: agentbroker.GuardrailDefaults{
			AllowedOrderTypes:   cfg.AgentBroker.DefaultAllowedTypes,
			AllowedSymbols:      cfg.AgentBroker.DefaultAllowedSymbols,
			MaxOrderNotional:    cfg.AgentBroker.DefaultMaxOrderNotional,
			MaxOrderQuantity:    cfg.AgentBroker.DefaultMaxOrderQty,
			MaxPositionNotional: cfg.AgentBroker.DefaultMaxPositionNotional,
			MaxPositionQuantity: cfg.AgentBroker.DefaultMaxPositionQty,
			Preset:              cfg.AgentBroker.DefaultPreset,
		},
	})
}

func (s *Server) closeClients() error {
	if s.clients != nil {
		return s.clients.Close()
	}
	return nil
}

func (s *Server) closeAuthManager() error {
	if s.authManager != nil {
		return s.authManager.Close()
	}
	return nil
}

func (s *Server) closeStreaming() error {
	if s.streaming != nil {
		return s.streaming.Close()
	}
	return nil
}

func (s *Server) readyClients(ctx context.Context) error {
	if s.clients == nil {
		return fmt.Errorf("backend clients not initialised")
	}
	return s.clients.Ready(ctx)
}
