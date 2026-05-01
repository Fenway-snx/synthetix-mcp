package server

// Forked streamable HTTP handler with session-bound and close hooks.
// Revisit once upstream exposes equivalent connection lifecycle callbacks.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

const (
	mcpProtocolVersionHeader = "MCP-Protocol-Version"
	mcpSessionIDHeader       = "Mcp-Session-Id"
	mcpVersion20250326       = "2025-03-26"
	mcpVersion20250618       = "2025-06-18"
	mcpVersion20251125       = "2025-11-25"
)

var supportedMCPProtocolVersions = map[string]struct{}{
	mcpVersion20250326: {},
	mcpVersion20250618: {},
	mcpVersion20251125: {},
}

type streamableSessionHandler struct {
	getServer      func(*http.Request) *mcp.Server
	logger         snx_lib_logging.Logger
	onSessionBound func(string, mcp.Connection)
	opts           mcp.StreamableHTTPOptions

	mu       sync.Mutex
	sessions map[string]*streamableSessionInfo
}

type streamableSessionInfo struct {
	session   *mcp.ServerSession
	transport *boundServerTransport
	userID    string

	timeout time.Duration
	timerMu sync.Mutex
	refs    int
	timer   *time.Timer
}

type boundServerTransport struct {
	base      *mcp.StreamableServerTransport
	onClose   func()
	onConnect func(mcp.Connection)

	mu   sync.Mutex
	conn mcp.Connection
}

type boundConnection struct {
	delegate  mcp.Connection
	onClose   func()
	closeOnce sync.Once
}

func newStreamableSessionHandler(
	logger snx_lib_logging.Logger,
	getServer func(*http.Request) *mcp.Server,
	onSessionBound func(string, mcp.Connection),
	opts *mcp.StreamableHTTPOptions,
) http.Handler {
	handler := &streamableSessionHandler{
		getServer:      getServer,
		logger:         logger,
		onSessionBound: onSessionBound,
		sessions:       make(map[string]*streamableSessionInfo),
	}
	if opts != nil {
		handler.opts = *opts
	}
	if handler.opts.CrossOriginProtection == nil {
		handler.opts.CrossOriginProtection = &http.CrossOriginProtection{}
	}
	return handler
}

func (h *streamableSessionHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !h.opts.DisableLocalhostProtection && localhostRequestWithNonLocalHost(req) {
		http.Error(w, fmt.Sprintf("Forbidden: invalid Host header %q", req.Host), http.StatusForbidden)
		return
	}
	if err := h.opts.CrossOriginProtection.Check(req); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if req.Method == http.MethodPost && req.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be 'application/json'", http.StatusUnsupportedMediaType)
		return
	}
	if err := validateAcceptHeader(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionID := req.Header.Get(mcpSessionIDHeader)
	sessInfo := h.lookupSession(sessionID)
	if sessionID != "" && sessInfo == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if sessInfo != nil && sessInfo.userID != "" {
		tokenInfo := auth.TokenInfoFromContext(req.Context())
		if tokenInfo == nil || tokenInfo.UserID != sessInfo.userID {
			http.Error(w, "session user mismatch", http.StatusForbidden)
			return
		}
	}

	switch req.Method {
	case http.MethodDelete:
		if sessionID == "" {
			http.Error(w, "Bad Request: DELETE requires an Mcp-Session-Id header", http.StatusBadRequest)
			return
		}
		if sessInfo != nil {
			if closeErr := sessInfo.session.Close(); closeErr != nil {
				h.logger.Warn("mcp session close failed", "error", closeErr, "session_id", sessionID)
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodGet:
		if sessionID == "" {
			http.Error(w, "Bad Request: GET requires an Mcp-Session-Id header", http.StatusBadRequest)
			return
		}
	case http.MethodPost:
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	protocolVersion := req.Header.Get(mcpProtocolVersionHeader)
	if protocolVersion == "" {
		protocolVersion = mcpVersion20250326
	}
	if _, ok := supportedMCPProtocolVersions[protocolVersion]; !ok {
		http.Error(w, fmt.Sprintf("Bad Request: Unsupported protocol version %q", protocolVersion), http.StatusBadRequest)
		return
	}

	newSession := false
	if sessInfo == nil {
		if req.Method != http.MethodPost {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		var err error
		sessInfo, sessionID, err = h.createSession(w, req, sessionID)
		if err != nil {
			return
		}
		newSession = true
		session := sessInfo.session
		defer func() {
			if newSession && session.InitializeParams() == nil {
				if closeErr := session.Close(); closeErr != nil {
					h.logger.Warn("mcp session close failed after missing initialize", "error", closeErr, "session_id", sessionID)
				}
			}
		}()
	}

	if req.Method == http.MethodPost {
		sessInfo.startPOST()
		defer sessInfo.endPOST()
	}
	sessInfo.transport.ServeHTTP(w, req)
}

// Allocates and connects a session, writing any HTTP error response inline.
func (h *streamableSessionHandler) createSession(
	w http.ResponseWriter,
	req *http.Request,
	sessionID string,
) (*streamableSessionInfo, string, error) {
	server := h.getServer(req)
	if server == nil {
		http.Error(w, "no server available", http.StatusBadRequest)
		return nil, sessionID, fmt.Errorf("no server available")
	}
	if sessionID == "" {
		newID, err := newSessionID()
		if err != nil {
			http.Error(w, "failed to allocate session", http.StatusInternalServerError)
			return nil, sessionID, err
		}
		sessionID = newID
	}
	// Keep streaming state alive beyond the initialize POST request.
	sessionCtx, cancelSession := context.WithCancel(context.Background())
	transport := &boundServerTransport{
		base: &mcp.StreamableServerTransport{
			SessionID:  sessionID,
			EventStore: h.opts.EventStore,
		},
		onClose: func() {
			cancelSession()
			h.removeSession(sessionID)
		},
		onConnect: func(conn mcp.Connection) {
			if h.onSessionBound != nil {
				h.onSessionBound(sessionID, conn)
			}
		},
	}
	session, err := server.Connect(sessionCtx, transport, nil)
	if err != nil {
		// Connect failed before any connection was established, so the
		// transport's onClose callback will never fire — release the
		// session-scoped context here to avoid leaking it.
		cancelSession()
		http.Error(w, "failed connection", http.StatusInternalServerError)
		return nil, sessionID, err
	}

	var userID string
	if tokenInfo := auth.TokenInfoFromContext(req.Context()); tokenInfo != nil {
		userID = tokenInfo.UserID
	}
	sessInfo := &streamableSessionInfo{
		session:   session,
		transport: transport,
		userID:    userID,
		timeout:   h.opts.SessionTimeout,
	}
	if h.opts.SessionTimeout > 0 {
		sessInfo.timer = time.AfterFunc(h.opts.SessionTimeout, func() {
			if closeErr := session.Close(); closeErr != nil {
				h.logger.Warn("mcp session close failed after idle timeout", "error", closeErr, "session_id", sessionID)
			}
		})
	}
	h.storeSession(sessionID, sessInfo)
	return sessInfo, sessionID, nil
}

func (h *streamableSessionHandler) lookupSession(sessionID string) *streamableSessionInfo {
	if sessionID == "" {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[sessionID]
}

func (h *streamableSessionHandler) storeSession(sessionID string, sessInfo *streamableSessionInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[sessionID] = sessInfo
}

func (h *streamableSessionHandler) removeSession(sessionID string) {
	h.mu.Lock()
	sessInfo := h.sessions[sessionID]
	delete(h.sessions, sessionID)
	h.mu.Unlock()
	if sessInfo != nil {
		sessInfo.stopTimer()
	}
}

func (i *streamableSessionInfo) startPOST() {
	if i.timeout <= 0 {
		return
	}
	i.timerMu.Lock()
	defer i.timerMu.Unlock()
	if i.timer == nil {
		return
	}
	if i.refs == 0 {
		i.timer.Stop()
	}
	i.refs++
}

func (i *streamableSessionInfo) endPOST() {
	if i.timeout <= 0 {
		return
	}
	i.timerMu.Lock()
	defer i.timerMu.Unlock()
	if i.timer == nil {
		return
	}
	if i.refs > 0 {
		i.refs--
	}
	if i.refs == 0 {
		i.timer.Reset(i.timeout)
	}
}

func (i *streamableSessionInfo) stopTimer() {
	i.timerMu.Lock()
	defer i.timerMu.Unlock()
	if i.timer != nil {
		i.timer.Stop()
		i.timer = nil
	}
}

func (t *boundServerTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	wrapped := &boundConnection{
		delegate: conn,
		onClose:  t.onClose,
	}
	t.mu.Lock()
	t.conn = wrapped
	t.mu.Unlock()
	if t.onConnect != nil {
		t.onConnect(wrapped)
	}
	return wrapped, nil
}

func (t *boundServerTransport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t.base.ServeHTTP(w, req)
}

func (t *boundServerTransport) Connection() mcp.Connection {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn
}

func (c *boundConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	return c.delegate.Read(ctx)
}

func (c *boundConnection) Write(ctx context.Context, msg jsonrpc.Message) error {
	return c.delegate.Write(ctx, msg)
}

func (c *boundConnection) Close() error {
	err := c.delegate.Close()
	c.closeOnce.Do(func() {
		if c.onClose != nil {
			c.onClose()
		}
	})
	return err
}

func (c *boundConnection) SessionID() string {
	return c.delegate.SessionID()
}

func validateAcceptHeader(req *http.Request) error {
	acceptValues := strings.Split(strings.Join(req.Header.Values("Accept"), ","), ",")
	var jsonOK bool
	var streamOK bool
	for _, value := range acceptValues {
		switch strings.TrimSpace(value) {
		case "application/json", "application/*":
			jsonOK = true
		case "text/event-stream", "text/*":
			streamOK = true
		case "*/*":
			jsonOK = true
			streamOK = true
		}
	}
	if req.Method == http.MethodGet {
		if !streamOK {
			return fmt.Errorf("accept must contain 'text/event-stream' for GET requests")
		}
		return nil
	}
	if req.Method != http.MethodDelete && (!jsonOK || !streamOK) {
		return fmt.Errorf("accept must contain both 'application/json' and 'text/event-stream'")
	}
	return nil
}

func localhostRequestWithNonLocalHost(req *http.Request) bool {
	localAddr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok || localAddr == nil {
		return false
	}
	host := localAddr.String()
	if tcpAddr, ok := localAddr.(*net.TCPAddr); ok {
		host = tcpAddr.IP.String()
	}
	return isLoopbackHost(host) && !isLoopbackHost(req.Host)
}

func isLoopbackHost(hostport string) bool {
	host := hostport
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]") {
		host = strings.TrimPrefix(strings.SplitN(host, "]", 2)[0], "[")
	} else if parsedHost, _, err := net.SplitHostPort(hostport); err == nil {
		host = parsedHost
	}
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func newSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
