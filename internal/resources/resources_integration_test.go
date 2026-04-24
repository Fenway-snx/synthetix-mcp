package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	"github.com/Fenway-snx/synthetix-mcp/internal/server/backend"
	"github.com/synthetixio/synthetix-go/restinfo"
	"github.com/synthetixio/synthetix-go/resttrade"
	"github.com/synthetixio/synthetix-go/types"
	"github.com/Fenway-snx/synthetix-mcp/internal/config"
	"github.com/Fenway-snx/synthetix-mcp/internal/session"
	"github.com/Fenway-snx/synthetix-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Minimal REST mock that dispatches /v1/info and /v1/trade POSTs by
// action name. Kept local to the resources package — the tools/
// fakeRESTServer lives in a different package and can't be reused.
type resRESTFake struct {
	t      *testing.T
	mu     sync.Mutex
	info   map[string]func(map[string]any) (int, any)
	trade  map[string]func(map[string]any) (int, any)
	server *httptest.Server
}

func newRESTFake(t *testing.T) *resRESTFake {
	t.Helper()
	f := &resRESTFake{
		t:     t,
		info:  map[string]func(map[string]any) (int, any){},
		trade: map[string]func(map[string]any) (int, any){},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/info", f.handle(false))
	mux.HandleFunc("/v1/trade", f.handle(true))
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *resRESTFake) handleInfo(action string, fn func(map[string]any) (int, any)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.info[action] = fn
}

func (f *resRESTFake) handleTrade(action string, fn func(map[string]any) (int, any)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.trade[action] = fn
}

func (f *resRESTFake) handle(isTrade bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var action string
		if isTrade {
			if params, ok := decoded["params"].(map[string]any); ok {
				action, _ = params["action"].(string)
			}
		} else {
			action, _ = decoded["action"].(string)
		}
		f.mu.Lock()
		var fn func(map[string]any) (int, any)
		if isTrade {
			fn = f.trade[action]
		} else {
			fn = f.info[action]
		}
		f.mu.Unlock()
		env := map[string]any{"requestId": "test-req"}
		if fn == nil {
			env["error"] = &types.APIError{Code: "UNHANDLED_ACTION", Message: "no handler for action=" + action}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(env)
			return
		}
		status, resp := fn(decoded)
		if resp != nil {
			raw, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			env["response"] = json.RawMessage(raw)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(env)
	}
}

func (f *resRESTFake) restInfo(t *testing.T) *restinfo.Client {
	t.Helper()
	c, err := restinfo.NewClient(restinfo.Config{BaseURL: f.server.URL, HTTPClient: f.server.Client()})
	if err != nil {
		t.Fatalf("build restinfo: %v", err)
	}
	return c
}

func (f *resRESTFake) restTrade(t *testing.T) *resttrade.Client {
	t.Helper()
	c, err := resttrade.NewClient(resttrade.Config{BaseURL: f.server.URL, HTTPClient: f.server.Client()})
	if err != nil {
		t.Fatalf("build resttrade: %v", err)
	}
	return c
}

// Satisfies tools.BrokerReadSigner with a static binding so the
// TradeReadClient shim can build a signed envelope against the fake.
type resFakeBrokerRead struct {
	wallet string
	sub    int64
}

func (s *resFakeBrokerRead) SignReadAction(int64, snx_lib_api_types.RequestAction) (snx_lib_auth.TradeSignature, int64, error) {
	return snx_lib_auth.TradeSignature{R: "0x1", S: "0x2", V: 27}, 0, nil
}
func (s *resFakeBrokerRead) WalletAddress() string { return s.wallet }
func (s *resFakeBrokerRead) SubAccountID() int64   { return s.sub }

func explicitResourceConfig() *config.Config {
	return &config.Config{
		AuthRPSPerSubAccount:       20,
		Environment:                "test",
		APIHTTPTimeout:             time.Second,
		MaxSubscriptionsPerSession: 10,
		PublicRPSPerIP:             10,
		ServerName:                 "synthetix-mcp",
		ServerVersion:              "0.1.0",
		SessionTTL:                 30 * time.Minute,
	}
}

func explicitResourceClients(t *testing.T, fake *resRESTFake, readyErr error) *backend.Clients {
	t.Helper()
	clients := &backend.Clients{}
	if fake != nil {
		clients.RESTInfo = fake.restInfo(t)
		clients.RESTTrade = fake.restTrade(t)
	}
	clients.SetReadyOverride(func(context.Context) error { return readyErr })
	return clients
}

func connectResourceSession(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}))
	t.Cleanup(httpServer.Close)

	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestStatusResourceReportsRunningWhenReady(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	Register(server, &tools.ToolDeps{Cfg: explicitResourceConfig(), Clients: explicitResourceClients(t, nil, nil), Store: &memorySessionStore{sessions: map[string]*session.State{}}}, nil)

	cs := connectResourceSession(t, server)
	result, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: statusURI})
	if err != nil {
		t.Fatalf("read status resource failed: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected one status resource content entry, got %d", len(result.Contents))
	}
	if result.Contents[0].Text == "" || !containsJSONField(result.Contents[0].Text, `"status": "running"`) {
		t.Fatalf("expected running status payload, got %s", result.Contents[0].Text)
	}
}

func TestFeeScheduleResourceHydratesAuthenticatedSubaccountFields(t *testing.T) {
	fake := newRESTFake(t)
	fake.handleTrade("getSubAccount", func(_ map[string]any) (int, any) {
		return http.StatusOK, types.SubAccountResponse{
			SubAccountID: "77",
			FeeRates: types.FeeRateInfo{
				TierName:     "tier-standard",
				MakerFeeRate: "0.0002",
				TakerFeeRate: "0.0005",
			},
		}
	})

	clients := explicitResourceClients(t, fake, nil)
	tradeReads := tools.NewTradeReadClient(
		clients.RESTTrade,
		&resFakeBrokerRead{wallet: "0xabc", sub: 77},
		nil,
		nil,
		nil,
	)

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	store := &memorySessionStore{sessions: map[string]*session.State{}}
	Register(server, &tools.ToolDeps{Cfg: explicitResourceConfig(), Clients: clients, Store: store}, tradeReads)

	cs := connectResourceSession(t, server)
	store.sessions[cs.ID()] = &session.State{
		AuthMode:      session.AuthModeAuthenticated,
		SubAccountID:  77,
		WalletAddress: "0xabc",
	}

	result, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: feeScheduleURI})
	if err != nil {
		t.Fatalf("read fee schedule failed: %v", err)
	}
	if !containsJSONField(result.Contents[0].Text, `"authenticated": true`) {
		t.Fatalf("expected authenticated fee schedule payload, got %s", result.Contents[0].Text)
	}
	if !containsJSONField(result.Contents[0].Text, `"subAccountId": "77"`) {
		t.Fatalf("expected subaccount 77 in fee schedule payload, got %s", result.Contents[0].Text)
	}
	if !containsJSONField(result.Contents[0].Text, `"tierName": "tier-standard"`) {
		t.Fatalf("expected tier name in fee schedule payload, got %s", result.Contents[0].Text)
	}
	if !containsJSONField(result.Contents[0].Text, `"makerFeeRate": "0.0002"`) {
		t.Fatalf("expected maker fee rate in fee schedule payload, got %s", result.Contents[0].Text)
	}
	if !containsJSONField(result.Contents[0].Text, `"takerFeeRate": "0.0005"`) {
		t.Fatalf("expected taker fee rate in fee schedule payload, got %s", result.Contents[0].Text)
	}
}

func TestMarketSpecsResourceReturnsMarketAndFundingRate(t *testing.T) {
	fake := newRESTFake(t)
	fake.handleInfo("getMarkets", func(_ map[string]any) (int, any) {
		return http.StatusOK, []types.MarketResponse{{
			Symbol:           "BTC-USDT",
			Description:      "Bitcoin perpetual",
			BaseAsset:        "BTC",
			QuoteAsset:       "USDT",
			IsOpen:           true,
			PriceIncrement:   "0.5",
			MinOrderSize:     "0.001",
			ContractSize:     1,
			MinNotionalValue: "10",
		}}
	})
	fake.handleInfo("getFundingRate", func(body map[string]any) (int, any) {
		sym, _ := body["symbol"].(string)
		return http.StatusOK, types.FundingRateResponse{
			Symbol:               sym,
			EstimatedFundingRate: "0.00001250",
			LastSettlementRate:   "0.00000500",
			FundingInterval:      3600000,
		}
	})

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0.1.0"}, nil)
	Register(server, &tools.ToolDeps{Cfg: explicitResourceConfig(), Clients: explicitResourceClients(t, fake, nil), Store: &memorySessionStore{sessions: map[string]*session.State{}}}, nil)

	cs := connectResourceSession(t, server)
	result, err := cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "market://specs/BTC-USDT"})
	if err != nil {
		t.Fatalf("read market specs failed: %v", err)
	}
	if !containsJSONField(result.Contents[0].Text, `"symbol": "BTC-USDT"`) {
		t.Fatalf("expected market symbol in specs payload, got %s", result.Contents[0].Text)
	}
	if !containsJSONField(result.Contents[0].Text, `"estimatedFundingRate": "0.00001250"`) {
		t.Fatalf("expected funding rate in specs payload, got %s", result.Contents[0].Text)
	}
}

func containsJSONField(body string, expected string) bool {
	return len(body) > 0 && strings.Contains(body, expected)
}
