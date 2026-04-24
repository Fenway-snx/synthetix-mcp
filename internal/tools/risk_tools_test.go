package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type errorSessionStore struct {
	err error
}

func (s *errorSessionStore) Delete(context.Context, string) error { return nil }
func (s *errorSessionStore) DeleteIfExists(context.Context, string) (bool, error) {
	return true, nil
}
func (s *errorSessionStore) Get(context.Context, string) (*session.State, error) {
	return nil, s.err
}
func (s *errorSessionStore) Save(context.Context, string, *session.State, time.Duration) error {
	return nil
}
func (s *errorSessionStore) Touch(context.Context, string, time.Duration) error { return nil }

func TestGetRateLimitsReturnsPublicModeWhenSessionMissing(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	RegisterRiskTools(server, &ToolDeps{
		Cfg:   testRiskConfig(),
		Store: &errorSessionStore{err: session.ErrSessionNotFound},
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_rate_limits"})
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %T", result.StructuredContent)
	}
	meta, ok := structured["_meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected _meta map, got %T", structured["_meta"])
	}
	if meta["authMode"] != string(session.AuthModePublic) {
		t.Fatalf("expected public auth mode, got %v", meta["authMode"])
	}
}

func TestGetRateLimitsSurfacesStoreErrors(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	RegisterRiskTools(server, &ToolDeps{
		Cfg:   testRiskConfig(),
		Store: &errorSessionStore{err: errors.New("redis unavailable")},
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	defer httpServer.Close()

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	defer cs.Close()

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_rate_limits"})
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error result")
	}
	if len(result.Content) == 0 || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "BACKEND_UNAVAILABLE") {
		t.Fatalf("expected backend unavailable error payload, got %#v", result.Content)
	}
}

func testRiskConfig() *config.Config {
	return &config.Config{
		AuthRPSPerSubAccount:       20,
		Environment:                "development",
		MaxSubscriptionsPerSession: 10,
		PublicRPSPerIP:             10,
		ServerName:                 "synthetix-mcp",
		ServerVersion:              "0.1.0",
		SessionTTL:                 30 * time.Minute,
	}
}
