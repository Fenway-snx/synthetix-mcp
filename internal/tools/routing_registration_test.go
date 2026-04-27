package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
)

type routingDomainProvider struct{}

func (routingDomainProvider) DomainName() string    { return "Synthetix" }
func (routingDomainProvider) DomainVersion() string { return "1" }
func (routingDomainProvider) ChainID() int          { return 1 }

func TestBrokerModeDoesNotRegisterExternalWalletWriteTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	deps := &ToolDeps{
		Cfg:   &config.Config{SessionTTL: 30 * time.Minute},
		Store: &fakeSessionStore{sessions: map[string]*session.State{}},
	}

	RegisterTradingTools(server, deps, nil, nil, nil, nil, false)
	RegisterSignaturePreviewTools(server, deps, routingDomainProvider{}, nil, false)

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

	toolsResult, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}

	names := map[string]bool{}
	for _, tool := range toolsResult.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"preview_order", "preview_auth_message"} {
		if !names[want] {
			t.Fatalf("expected %s to be registered, got %v", want, names)
		}
	}
	for _, forbidden := range []string{
		"preview_trade_signature",
		"signed_place_order",
		"signed_modify_order",
		"signed_cancel_order",
		"signed_cancel_all_orders",
		"signed_close_position",
	} {
		if names[forbidden] {
			t.Fatalf("expected %s to be hidden in broker mode", forbidden)
		}
	}
}
