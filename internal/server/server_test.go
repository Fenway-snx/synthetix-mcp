package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internal_auth "github.com/Fenway-snx/synthetix-mcp/internal/auth"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type stubClients struct {
	readyErr error
}

type stubAuthManager struct{}

type stubStreamingManager struct {
	boundSessionID string
}

func (s *stubClients) Close() error {
	return nil
}

func (s *stubClients) Ready(context.Context) error {
	return s.readyErr
}

func (s *stubAuthManager) Authenticate(_ context.Context, sessionID string, message string, signatureHex string) (*internal_auth.AuthenticateResult, error) {
	return &internal_auth.AuthenticateResult{
		Authenticated:    true,
		SessionExpiresAt: 123,
		SubAccountID:     77,
		WalletAddress:    "0xabc",
	}, nil
}

func (s *stubAuthManager) ValidateTradeAction(string, int64, int64, int64, snx_lib_api_types.RequestAction, any, snx_lib_auth.TradeSignature) error {
	return nil
}

func (s *stubAuthManager) VerifySessionAccess(context.Context, string, int64) error {
	return nil
}

func (s *stubAuthManager) Close() error {
	return nil
}

func (s *stubStreamingManager) BindSession(sessionID string, _ mcp.Connection) {
	s.boundSessionID = sessionID
}

func (s *stubStreamingManager) Close() error {
	return nil
}

func TestHealthRoutes(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		srv := newServer(snx_lib_logging_doubles.NewStubLogger(), testConfig(), &stubClients{}, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"ready"`) {
			t.Fatalf("expected ready response, got %s", rec.Body.String())
		}
	})

	t.Run("not ready", func(t *testing.T) {
		srv := newServer(snx_lib_logging_doubles.NewStubLogger(), testConfig(), &stubClients{
			readyErr: context.DeadlineExceeded,
		}, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"not_ready"`) {
			t.Fatalf("expected not_ready response, got %s", rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "deadline") {
			t.Fatalf("expected readiness response to avoid leaking internal error details, got %s", rec.Body.String())
		}
	})
}

// Pins JSON 404 envelopes for unmatched OAuth and generic probe paths.
func TestUnknownPathsReturnJSONNotFoundForOAuthProbes(t *testing.T) {
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), testConfig(), &stubClients{}, nil, nil, nil)

	probes := []string{
		"/.well-known/oauth-authorization-server",
		"/.well-known/oauth-authorization-server/mcp",
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-protected-resource/mcp",
		"/.well-known/openid-configuration",
		"/register",
		"/authorize",
		"/token",
		"/some-random-path",
		"/",
	}

	for _, path := range probes {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			srv.httpServer.Handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404 for OAuth probe %q, got %d", path, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("expected application/json content type for %q, got %q", path, ct)
			}
			if body := rec.Body.String(); !strings.Contains(body, `"error"`) {
				t.Fatalf("expected JSON error envelope for %q, got %q", path, body)
			}
			if body := rec.Body.String(); strings.Contains(body, "404 page not found") {
				t.Fatalf("response for %q must not leak Go's plain-text 404 (it crashes Claude Code's OAuth JSON parser): %s", path, body)
			}
		})
	}
}

func TestInitializeRoute(t *testing.T) {
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), testConfig(), &stubClients{}, nil, nil, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
	)
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if sessionID := rec.Header().Get("Mcp-Session-Id"); sessionID == "" {
		t.Fatal("expected Mcp-Session-Id header to be set")
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"protocolVersion"`) {
		t.Fatalf("expected initialize response, got %s", body)
	}
	if !strings.Contains(body, `"serverInfo"`) {
		t.Fatalf("expected server info in initialize response, got %s", body)
	}
}

func TestInitializeRouteBindsStreamingSession(t *testing.T) {
	streamingManager := &stubStreamingManager{}
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), testConfig(), &stubClients{}, nil, streamingManager, nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
	)
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	sessionID := rec.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("expected Mcp-Session-Id header to be set")
	}
	if streamingManager.boundSessionID != sessionID {
		t.Fatalf("expected bound session %q, got %q", sessionID, streamingManager.boundSessionID)
	}
}

func TestAuthenticateToolIsRegistered(t *testing.T) {
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), testConfig(), &stubClients{}, nil, nil, nil)
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

	toolsResult, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}

	found := false
	for _, tool := range toolsResult.Tools {
		if tool.Name == "authenticate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected authenticate tool to be registered")
	}
}

func TestServerInstructionsTellAgentsNotToFallbackToBash(t *testing.T) {
	instructions := buildServerInstructions(true)
	for _, want := range []string{
		"quickstart prompt",
		"BTC-USDT",
		"Guardrails",
		"at most once",
		"unknown tool",
		"do not fall back to Bash",
		"restart Claude Code",
		"claude mcp list",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("expected instructions to contain %q, got %s", want, instructions)
		}
	}
}

// Confirms OAuth-discovery probes receive the MCP auth JSON 404.
func TestWellKnownOAuthProbeReturnsJSON404(t *testing.T) {
	srv := newServer(snx_lib_logging_doubles.NewStubLogger(), testConfig(), &stubClients{}, nil, nil, nil)

	probes := []string{
		"/.well-known/oauth-authorization-server",
		"/.well-known/oauth-protected-resource",
		"/.well-known/openid-configuration",
	}
	for _, path := range probes {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("User-Agent", "claude-code/1.2.3")
			rec := httptest.NewRecorder()

			srv.httpServer.Handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("expected application/json content-type, got %q", ct)
			}
			if hint := rec.Header().Get("Mcp-Auth"); hint != "tool:authenticate" {
				t.Fatalf("expected Mcp-Auth advisory header, got %q", hint)
			}
			body := rec.Body.String()
			for _, want := range []string{`"error":"not_found"`, `"auth_method":"mcp_tool"`, `"auth_tool":"authenticate"`} {
				if !strings.Contains(body, want) {
					t.Fatalf("expected body to contain %q, got %s", want, body)
				}
			}
		})
	}
}

func testConfig() *config.Config {
	return &config.Config{
		AuthCacheMaxEntries:        50000,
		APIHTTPTimeout:             time.Second,
		MaxRequestBodyBytes:        1 << 20,
		ServerAddress:              ":8095",
		ServerName:                 "synthetix-mcp",
		ServerVersion:              "0.1.0",
		SessionTTL:                 30 * time.Minute,
		ShutdownTimeout:            5 * time.Second,
		MaxSubscriptionsPerSession: 10,
	}
}
