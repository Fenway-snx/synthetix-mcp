package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/prompts"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeRequestRateLimiter struct {
	errByOperation map[string]error
	clientIPs      []string
	operations     []string
}

func (f *fakeRequestRateLimiter) Check(ctx context.Context, operationName string, batchSize int, _ *session.State) error {
	f.clientIPs = append(f.clientIPs, tools.ClientIPFromContext(ctx))
	f.operations = append(f.operations, operationName)
	if f.errByOperation == nil {
		return nil
	}
	return f.errByOperation[operationName]
}

type countingResponseWriter struct {
	header           http.Header
	statusCode       int
	writeHeaderCalls int
}

func (w *countingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *countingResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *countingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.writeHeaderCalls++
}

func TestListPromptsIsRateLimitedAtRequestLayer(t *testing.T) {
	limiter := &fakeRequestRateLimiter{
		errByOperation: map[string]error{
			"list_prompts": tools.NewRateLimitExceededError("ip", "list_prompts", 10),
		},
	}

	srv := newServer(
		snx_lib_logging_doubles.NewStubLogger(),
		testConfig(),
		&stubClients{},
		nil,
		nil,
		nil,
		limiter,
	)
	prompts.Register(srv.mcpServer, false)

	httpServer := httptest.NewServer(srv.httpServer.Handler)
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer session.Close()

	_, err = session.ListPrompts(context.Background(), nil)
	if err == nil {
		t.Fatal("expected list prompts to be rate limited")
	}

	var rpcErr *jsonrpc.Error
	if !jsonrpcErrorAs(err, &rpcErr) {
		t.Fatalf("expected jsonrpc error, got %T: %v", err, err)
	}
	if rpcErr.Code != tools.JSONRPCCodeRateLimitExceeded {
		t.Fatalf("expected rate limit code %d, got %d", tools.JSONRPCCodeRateLimitExceeded, rpcErr.Code)
	}
	if got := lastOperation(limiter.operations); got != "list_prompts" {
		t.Fatalf("expected list_prompts operation, got %q", got)
	}
}

func TestGetPromptIsRateLimitedAtRequestLayer(t *testing.T) {
	limiter := &fakeRequestRateLimiter{
		errByOperation: map[string]error{
			"get_prompt": tools.NewRateLimitExceededError("ip", "get_prompt", 10),
		},
	}

	srv := newServer(
		snx_lib_logging_doubles.NewStubLogger(),
		testConfig(),
		&stubClients{},
		nil,
		nil,
		nil,
		limiter,
	)
	prompts.Register(srv.mcpServer, false)

	httpServer := httptest.NewServer(srv.httpServer.Handler)
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer session.Close()

	_, err = session.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name:      "market-analysis",
		Arguments: map[string]string{"symbol": "BTC-USDT"},
	})
	if err == nil {
		t.Fatal("expected get prompt to be rate limited")
	}

	var rpcErr *jsonrpc.Error
	if !jsonrpcErrorAs(err, &rpcErr) {
		t.Fatalf("expected jsonrpc error, got %T: %v", err, err)
	}
	if rpcErr.Code != tools.JSONRPCCodeRateLimitExceeded {
		t.Fatalf("expected rate limit code %d, got %d", tools.JSONRPCCodeRateLimitExceeded, rpcErr.Code)
	}
	if got := lastOperation(limiter.operations); got != "get_prompt" {
		t.Fatalf("expected get_prompt operation, got %q", got)
	}
}

func TestListToolsIsRateLimitedAtRequestLayer(t *testing.T) {
	limiter := &fakeRequestRateLimiter{
		errByOperation: map[string]error{
			"list_tools": tools.NewRateLimitExceededError("ip", "list_tools", 10),
		},
	}

	srv := newServer(
		snx_lib_logging_doubles.NewStubLogger(),
		testConfig(),
		&stubClients{},
		nil,
		nil,
		nil,
		limiter,
	)
	tools.RegisterSessionTools(srv.mcpServer, &tools.ToolDeps{}, &stubAuthManager{}, nil)

	httpServer := httptest.NewServer(srv.httpServer.Handler)
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer session.Close()

	_, err = session.ListTools(context.Background(), nil)
	if err == nil {
		t.Fatal("expected list tools to be rate limited")
	}

	var rpcErr *jsonrpc.Error
	if !jsonrpcErrorAs(err, &rpcErr) {
		t.Fatalf("expected jsonrpc error, got %T: %v", err, err)
	}
	if rpcErr.Code != tools.JSONRPCCodeRateLimitExceeded {
		t.Fatalf("expected rate limit code %d, got %d", tools.JSONRPCCodeRateLimitExceeded, rpcErr.Code)
	}
	if got := lastOperation(limiter.operations); got != "list_tools" {
		t.Fatalf("expected list_tools operation, got %q", got)
	}
}

func TestProtocolPingIsRateLimitedAtRequestLayer(t *testing.T) {
	limiter := &fakeRequestRateLimiter{
		errByOperation: map[string]error{
			"mcp_ping": tools.NewRateLimitExceededError("ip", "mcp_ping", 10),
		},
	}

	srv := newServer(
		snx_lib_logging_doubles.NewStubLogger(),
		testConfig(),
		&stubClients{},
		nil,
		nil,
		nil,
		limiter,
	)

	httpServer := httptest.NewServer(srv.httpServer.Handler)
	defer httpServer.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer session.Close()

	err = session.Ping(context.Background(), nil)
	if err == nil {
		t.Fatal("expected protocol ping to be rate limited")
	}

	var rpcErr *jsonrpc.Error
	if !jsonrpcErrorAs(err, &rpcErr) {
		t.Fatalf("expected jsonrpc error, got %T: %v", err, err)
	}
	if rpcErr.Code != tools.JSONRPCCodeRateLimitExceeded {
		t.Fatalf("expected rate limit code %d, got %d", tools.JSONRPCCodeRateLimitExceeded, rpcErr.Code)
	}
	if got := lastOperation(limiter.operations); got != "mcp_ping" {
		t.Fatalf("expected mcp_ping operation, got %q", got)
	}
}

func TestWrapMCPHandlerWithRateLimitInjectsClientIPIntoLimiterContext(t *testing.T) {
	limiter := &fakeRequestRateLimiter{
		errByOperation: map[string]error{
			"list_tools": tools.NewRateLimitExceededError("ip", "list_tools", 10),
		},
	}
	handler := wrapMCPHandlerWithRateLimit(snx_lib_logging_doubles.NewStubLogger(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when request rate limit blocks")
	}), limiter, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.RemoteAddr = "198.51.100.10:443"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := lastClientIP(limiter.clientIPs); got != "198.51.100.10" {
		t.Fatalf("expected limiter to see remote client IP, got %q", got)
	}
}

func TestWriteJSONRPCResponsesDoesNotWriteHeadersOnEncodeFailure(t *testing.T) {
	writer := &countingResponseWriter{}

	err := writeJSONRPCResponses(writer, nil, false)
	if !errors.Is(err, errNoJSONRPCResponses) {
		t.Fatalf("expected %v, got %v", errNoJSONRPCResponses, err)
	}
	if writer.writeHeaderCalls != 0 {
		t.Fatalf("expected no headers to be written, got %d calls", writer.writeHeaderCalls)
	}
}

func TestWrapMCPHandlerWithRateLimitOnlyBlocksLimitedBatchEntries(t *testing.T) {
	limiter := &fakeRequestRateLimiter{
		errByOperation: map[string]error{
			"list_tools": tools.NewRateLimitExceededError("ip", "list_tools", 10),
		},
	}
	handler := wrapMCPHandlerWithRateLimit(snx_lib_logging_doubles.NewStubLogger(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"jsonrpc":"2.0","id":2,"result":{"ok":true}}]`))
	}), limiter, nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`[{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}},{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ping","arguments":{}}}]`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var responses []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("unmarshal batch response: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 batch responses, got %d", len(responses))
	}
	if int(responses[0]["id"].(float64)) != 1 || responses[0]["error"] == nil {
		t.Fatalf("expected first response to be rate limit error, got %#v", responses[0])
	}
	if int(responses[1]["id"].(float64)) != 2 || responses[1]["result"] == nil {
		t.Fatalf("expected second response to preserve allowed request, got %#v", responses[1])
	}
}

func TestWrapMCPHandlerWithRateLimitDropsLimitedNotifications(t *testing.T) {
	limiter := &fakeRequestRateLimiter{
		errByOperation: map[string]error{
			"list_tools": tools.NewRateLimitExceededError("ip", "list_tools", 10),
		},
	}
	called := false
	handler := wrapMCPHandlerWithRateLimit(snx_lib_logging_doubles.NewStubLogger(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}), limiter, nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","params":{}}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("expected over-limit notification to be dropped before reaching next handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for dropped notification, got %d", rec.Code)
	}
}

func jsonrpcErrorAs(err error, target **jsonrpc.Error) bool {
	return errors.As(err, target)
}

func lastOperation(operations []string) string {
	if len(operations) == 0 {
		return ""
	}
	return operations[len(operations)-1]
}

func lastClientIP(clientIPs []string) string {
	if len(clientIPs) == 0 {
		return ""
	}
	return clientIPs[len(clientIPs)-1]
}
