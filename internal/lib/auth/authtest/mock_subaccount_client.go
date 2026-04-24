// Package authtest provides grpc-typed test doubles and wiring helpers for
// lib/auth. It lives in a sub-package so that production builds of lib/auth
// stay free of google.golang.org/grpc and the v4 protobuf surface — only
// test binaries that explicitly import authtest pull in those dependencies.
package authtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_authgrpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authgrpc"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

var (
	errServiceUnavailable = errors.New("service unavailable")
)

// Funding interval constant (1 hour in milliseconds)
const FundingInterval = time.Hour

// MockSubaccountServiceClient implements a mock SubaccountService client for tests
type MockSubaccountServiceClient struct {
	accounts               map[snx_lib_api_types.WalletAddress][]snx_lib_core.SubAccountId       // ethereumAddress -> accountIDs owned
	delegations            map[snx_lib_core.SubAccountId][]*v4grpc.DelegationInfo                // subAccountID -> delegations
	delegationsForDelegate map[snx_lib_api_types.WalletAddress][]*v4grpc.DelegationWithOwnerInfo // delegateAddress -> delegations where this address is delegate
	masterAccounts         map[snx_lib_core.SubAccountId][]snx_lib_core.SubAccountId             // masterAccountID -> subAccountIDs (includes master itself)
}

// NewMockSubaccountServiceClient creates a new mock SubaccountService client
func NewMockSubaccountServiceClient() *MockSubaccountServiceClient {
	return &MockSubaccountServiceClient{
		accounts:               make(map[snx_lib_api_types.WalletAddress][]snx_lib_core.SubAccountId),
		delegations:            make(map[snx_lib_core.SubAccountId][]*v4grpc.DelegationInfo),
		delegationsForDelegate: make(map[snx_lib_api_types.WalletAddress][]*v4grpc.DelegationWithOwnerInfo),
		masterAccounts:         make(map[snx_lib_core.SubAccountId][]snx_lib_core.SubAccountId),
	}
}

// AddMockAccount adds a mock account ownership for testing
func (m *MockSubaccountServiceClient) AddMockAccount(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) {
	m.accounts[ethereumAddress] = append(m.accounts[ethereumAddress], accountId)
	// Also add to master accounts map (treat each account as its own master for simplicity)
	m.masterAccounts[accountId] = append(m.masterAccounts[accountId], accountId)
}

// AddMockSubAccount adds a subaccount under a master account for testing
func (m *MockSubaccountServiceClient) AddMockSubAccount(
	masterAccountId snx_lib_core.SubAccountId,
	subAccountId snx_lib_core.SubAccountId,
) {
	m.masterAccounts[masterAccountId] = append(m.masterAccounts[masterAccountId], subAccountId)
}

// ListSubaccounts implements the SubaccountService.ListSubaccounts method for testing
func (m *MockSubaccountServiceClient) ListSubaccounts(ctx context.Context, req *v4grpc.ListSubaccountsRequest, opts ...grpc.CallOption) (*v4grpc.ListSubaccountsResponse, error) {
	var accountIDs []snx_lib_core.SubAccountId

	walletAddress := snx_lib_api_types.WalletAddressFromStringUnvalidated(req.WalletAddress)

	if walletAddress != snx_lib_api_types.WalletAddress_None {
		accountIDs = m.accounts[walletAddress]
	} else {
		subAccountId := snx_lib_core.SubAccountId(req.SubAccountId)

		var exists bool
		accountIDs, exists = m.masterAccounts[subAccountId]
		if !exists {
			accountIDs = []snx_lib_core.SubAccountId{subAccountId}
		}
	}

	var subaccounts []*v4grpc.SubaccountInfo

	for _, accountId := range accountIDs {
		subaccounts = append(subaccounts, &v4grpc.SubaccountInfo{
			Id:   int64(accountId),
			Name: "Test Subaccount",
			MarginSummary: &v4grpc.MarginSummary{
				AccountValue:      "1000.0",
				AvailableMargin:   "800.0",
				UnrealizedPnl:     "0.0",
				MaintenanceMargin: "100.0",
				InitialMargin:     "200.0",
				Withdrawable:      "700.0",
			},
			Collaterals: []*v4grpc.CollateralInfo{},
			Positions:   []*v4grpc.PositionItem{},
		})
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.ListSubaccountsResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Subaccounts: subaccounts,
	}, nil
}

// Implement required SubaccountServiceClient interface methods (stubs for testing)
func (m *MockSubaccountServiceClient) CreateSubaccount(ctx context.Context, req *v4grpc.CreateSubaccountRequest, opts ...grpc.CallOption) (*v4grpc.SubaccountResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.SubaccountResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Id:          1,
		Name:        req.Name,
	}, nil
}

func (m *MockSubaccountServiceClient) UpdateSubaccount(ctx context.Context, req *v4grpc.UpdateSubaccountRequest, opts ...grpc.CallOption) (*v4grpc.SubaccountResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.SubaccountResponse{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		Id:           req.Id,
		SubAccountId: req.Id,
		Name:         req.Name,
	}, nil
}

// UpdateSubaccountStatus implements the SubaccountService.UpdateSubaccountStatus method for testing
func (m *MockSubaccountServiceClient) UpdateSubaccountStatus(ctx context.Context, req *v4grpc.UpdateSubaccountStatusRequest, opts ...grpc.CallOption) (*v4grpc.UpdateSubaccountStatusResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.UpdateSubaccountStatusResponse{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		SubAccountId: req.SubAccountId,
		OldStatus:    v4grpc.SubaccountStatus_ACCOUNT_STATUS_ACTIVE,
		NewStatus:    req.Status,
	}, nil
}

func (m *MockSubaccountServiceClient) AddToWithdrawBlacklist(ctx context.Context, req *v4grpc.AddToWithdrawBlacklistRequest, opts ...grpc.CallOption) (*v4grpc.AddToWithdrawBlacklistResponse, error) {
	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	return &v4grpc.AddToWithdrawBlacklistResponse{TimestampMs: timestampMs, TimestampUs: timestampUs, Added: true}, nil
}

func (m *MockSubaccountServiceClient) AssignWalletTier(ctx context.Context, req *v4grpc.AssignWalletTierRequest, opts ...grpc.CallOption) (*v4grpc.AssignWalletTierResponse, error) {
	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	return &v4grpc.AssignWalletTierResponse{TimestampMs: timestampMs, TimestampUs: timestampUs}, nil
}

func (m *MockSubaccountServiceClient) ListWithdrawBlacklist(ctx context.Context, req *v4grpc.ListWithdrawBlacklistRequest, opts ...grpc.CallOption) (*v4grpc.ListWithdrawBlacklistResponse, error) {
	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	return &v4grpc.ListWithdrawBlacklistResponse{TimestampMs: timestampMs, TimestampUs: timestampUs, WalletAddresses: nil}, nil
}

func (m *MockSubaccountServiceClient) RemoveFromWithdrawBlacklist(ctx context.Context, req *v4grpc.RemoveFromWithdrawBlacklistRequest, opts ...grpc.CallOption) (*v4grpc.RemoveFromWithdrawBlacklistResponse, error) {
	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	return &v4grpc.RemoveFromWithdrawBlacklistResponse{TimestampMs: timestampMs, TimestampUs: timestampUs, Removed: true}, nil
}

func (m *MockSubaccountServiceClient) GetOrderHistory(ctx context.Context, req *v4grpc.GetOrderHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetOrderHistoryResponse, error) {
	return &v4grpc.GetOrderHistoryResponse{}, nil
}

func (m *MockSubaccountServiceClient) GetTradeHistory(ctx context.Context, req *v4grpc.GetTradeHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetTradeHistoryResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetTradeHistoryResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetLastTrades(ctx context.Context, req *v4grpc.GetLastTradesRequest, opts ...grpc.CallOption) (*v4grpc.GetLastTradesResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetLastTradesResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetPositionHistory(ctx context.Context, req *v4grpc.GetPositionHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetPositionHistoryResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetPositionHistoryResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetPositions(ctx context.Context, req *v4grpc.GetPositionsRequest, opts ...grpc.CallOption) (*v4grpc.GetPositionsResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetPositionsResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetPositionsBySymbol(ctx context.Context, req *v4grpc.GetPositionsBySymbolRequest, opts ...grpc.CallOption) (*v4grpc.GetPositionsBySymbolResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetPositionsBySymbolResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetOITotalsBySymbol(ctx context.Context, req *v4grpc.GetOITotalsBySymbolRequest, opts ...grpc.CallOption) (*v4grpc.GetOITotalsBySymbolResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetOITotalsBySymbolResponse{
		TimestampMs:        timestamp_ms,
		TimestampUs:        timestamp_us,
		TotalLongQuantity:  "0",
		TotalShortQuantity: "0",
	}, nil
}

func (m *MockSubaccountServiceClient) GetAllOITotals(ctx context.Context, req *v4grpc.GetAllOITotalsRequest, opts ...grpc.CallOption) (*v4grpc.GetAllOITotalsResponse, error) {
	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetAllOITotalsResponse{
		TimestampMs: timestampMs,
		TimestampUs: timestampUs,
		Items:       []*v4grpc.OITotalsItem{},
	}, nil
}

func (m *MockSubaccountServiceClient) GetOpenOrders(ctx context.Context, req *v4grpc.GetOpenOrdersRequest, opts ...grpc.CallOption) (*v4grpc.GetOpenOrdersResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetOpenOrdersResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetOpenLimitOrders(ctx context.Context, req *v4grpc.GetOpenLimitOrdersRequest, opts ...grpc.CallOption) (*v4grpc.GetOpenLimitOrdersResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetOpenLimitOrdersResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetExpiredOpenOrders(ctx context.Context, req *v4grpc.GetExpiredOpenOrdersRequest, opts ...grpc.CallOption) (*v4grpc.GetExpiredOpenOrdersResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetExpiredOpenOrdersResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetAllSubaccounts(ctx context.Context, req *v4grpc.GetAllSubaccountsRequest, opts ...grpc.CallOption) (*v4grpc.GetAllSubaccountsResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetAllSubaccountsResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) WithdrawCollateral(ctx context.Context, req *v4grpc.WithdrawCollateralRequest, opts ...grpc.CallOption) (*v4grpc.WithdrawCollateralResponse, error) {
	return &v4grpc.WithdrawCollateralResponse{
		TmRespondedAt: timestamppb.New(snx_lib_utils_time.Now()),
		RequestId:     "test-request-id",
		Symbol:        req.Symbol,
		Amount:        req.Amount,
		Destination:   req.Destination,
	}, nil
}

func (m *MockSubaccountServiceClient) GetSLPMappings(ctx context.Context, req *v4grpc.GetSLPMappingsRequest, opts ...grpc.CallOption) (*v4grpc.GetSLPMappingsResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetSLPMappingsResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Mappings:    []*v4grpc.SLPMapping{},
	}, nil
}

// Delegation methods for testing
func (m *MockSubaccountServiceClient) CreateDelegation(ctx context.Context, req *v4grpc.CreateDelegationRequest, opts ...grpc.CallOption) (*v4grpc.CreateDelegationResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.CreateDelegationResponse{
		TimestampMs:     timestamp_ms,
		TimestampUs:     timestamp_us,
		Id:              1,
		SubAccountId:    req.SubAccountId,
		DelegateAddress: req.DelegateAddress,
		Permissions:     req.Permissions,
		ExpiresAt:       req.ExpiresAt,
		CreatedAt:       timestamppb.New(time.Now().UTC()),
	}, nil
}

func (m *MockSubaccountServiceClient) GetLatestFundingRates(ctx context.Context, req *v4grpc.GetLatestFundingRatesRequest, opts ...grpc.CallOption) (*v4grpc.GetLatestFundingRatesResponse, error) {
	now := time.Now().UTC()

	fundingInterval_ms := FundingInterval.Milliseconds()
	lastSettlementTime := now.Add(-FundingInterval)
	nextFundingTime := now.Add(FundingInterval)

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	symbols := req.Symbols
	if len(symbols) == 0 {
		symbols = []string{"BTC-USDT"}
	}

	items := make([]*v4grpc.GetLatestFundingRatesResponseItem, 0, len(symbols))
	for _, symbol := range symbols {
		items = append(items, &v4grpc.GetLatestFundingRatesResponseItem{
			Symbol:               symbol,
			EstimatedFundingRate: "0.00001250",
			LastSettlementRate:   "0.00001200",
			LastSettlementTime:   timestamppb.New(lastSettlementTime),
			FundingIntervalMs:    fundingInterval_ms,
			NextFundingTime:      timestamppb.New(nextFundingTime),
		})
	}

	return &v4grpc.GetLatestFundingRatesResponse{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		FundingRates: items,
	}, nil
}

func (m *MockSubaccountServiceClient) GetFundingRateHistory(ctx context.Context, req *v4grpc.GetFundingRateHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetFundingRateHistoryResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetFundingRateHistoryResponse{
		TimestampMs:  timestamp_ms,
		TimestampUs:  timestamp_us,
		Symbol:       req.Symbol,
		FundingRates: []*v4grpc.FundingRateHistoryItem{},
	}, nil
}

func (m *MockSubaccountServiceClient) RemoveDelegation(ctx context.Context, req *v4grpc.RemoveDelegationRequest, opts ...grpc.CallOption) (*v4grpc.RemoveDelegationResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.RemoveDelegationResponse{
		TimestampMs:     timestamp_ms,
		TimestampUs:     timestamp_us,
		SubAccountId:    req.SubAccountId,
		DelegateAddress: req.DelegateAddress,
		Success:         true,
	}, nil
}

func (m *MockSubaccountServiceClient) RemoveAllDelegations(ctx context.Context, req *v4grpc.RemoveAllDelegationsRequest, opts ...grpc.CallOption) (*v4grpc.RemoveAllDelegationsResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	subAccountId := snx_lib_core.SubAccountId(req.SubAccountId)

	// Get existing delegations to simulate removal
	delegations := m.delegations[subAccountId]
	removedSigners := make([]string, len(delegations))
	for i, d := range delegations {
		removedSigners[i] = d.DelegateAddress
	}

	// Clear delegations for this subaccount
	delete(m.delegations, subAccountId)

	return &v4grpc.RemoveAllDelegationsResponse{
		TimestampMs:    timestamp_ms,
		TimestampUs:    timestamp_us,
		SubAccountId:   req.SubAccountId,
		RemovedSigners: removedSigners,
	}, nil
}

func (m *MockSubaccountServiceClient) GetDelegations(ctx context.Context, req *v4grpc.GetDelegationsRequest, opts ...grpc.CallOption) (*v4grpc.GetDelegationsResponse, error) {
	subAccountId := snx_lib_core.SubAccountId(req.SubAccountId)

	list := m.delegations[subAccountId]

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetDelegationsResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Delegations: list,
	}, nil
}

func (m *MockSubaccountServiceClient) GetDelegationsForDelegate(ctx context.Context, req *v4grpc.GetDelegationsForDelegateRequest, opts ...grpc.CallOption) (*v4grpc.GetDelegationsForDelegateResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	delegateAddress := snx_lib_api_types.WalletAddressFromStringUnvalidated(req.DelegateAddress)

	delegations := m.delegationsForDelegate[delegateAddress]
	if delegations == nil {
		delegations = []*v4grpc.DelegationWithOwnerInfo{}
	}

	return &v4grpc.GetDelegationsForDelegateResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Delegations: delegations,
	}, nil
}

func (m *MockSubaccountServiceClient) GetSubaccount(ctx context.Context, req *v4grpc.GetSubaccountRequest, opts ...grpc.CallOption) (*v4grpc.SubaccountInfo, error) {
	subAccountId := snx_lib_core.SubAccountId(req.SubAccountId)

	for _, accounts := range m.accounts {
		for _, accountId := range accounts {
			if accountId == subAccountId {
				return &v4grpc.SubaccountInfo{
					Name: "Mock Subaccount",
					Id:   int64(accountId),
					MarginSummary: &v4grpc.MarginSummary{
						AccountValue:         "100",
						AvailableMargin:      "80",
						UnrealizedPnl:        "10",
						MaintenanceMargin:    "20",
						InitialMargin:        "20",
						Withdrawable:         "70",
						AdjustedAccountValue: "100",
					},
					Collaterals: []*v4grpc.CollateralInfo{},
					Positions:   []*v4grpc.PositionItem{},
					Leverages:   map[string]uint32{},
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("subaccount %d not found", req.SubAccountId)
}

// VerifySubaccountAuthorization implements the unified authorization check
func (m *MockSubaccountServiceClient) VerifySubaccountAuthorization(ctx context.Context, req *v4grpc.VerifySubaccountAuthorizationRequest, opts ...grpc.CallOption) (*v4grpc.VerifySubaccountAuthorizationResponse, error) {
	subAccountId := snx_lib_core.SubAccountId(req.SubAccountId)
	walletAddress := snx_lib_api_types.WalletAddressFromStringUnvalidated(req.Address)

	// Check if the address owns the subaccount
	accountIDs, exists := m.accounts[walletAddress]
	if exists {
		for _, accountId := range accountIDs {
			if accountId == subAccountId {
				timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

				return &v4grpc.VerifySubaccountAuthorizationResponse{
					TimestampMs:       timestamp_ms,
					TimestampUs:       timestamp_us,
					IsAuthorized:      true,
					AuthorizationType: v4grpc.AuthorizationType_AUTHORIZATION_TYPE_OWNER,
				}, nil
			}
		}
	}

	// Check if the address is a delegate
	list := m.delegations[subAccountId]
	now := time.Now().UTC()
	for _, d := range list {
		if strings.EqualFold(d.DelegateAddress, req.Address) {
			// Check expiration
			if d.ExpiresAt != nil && d.ExpiresAt.AsTime().Before(now) {
				continue
			}
			// Check permissions
			if len(req.Permissions) == 0 {
				timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

				return &v4grpc.VerifySubaccountAuthorizationResponse{
					TimestampMs:       timestamp_ms,
					TimestampUs:       timestamp_us,
					IsAuthorized:      true,
					AuthorizationType: v4grpc.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
				}, nil
			}
			// Check ALL requested permissions exist using hierarchy-aware matching (AND logic)
			allFound := true
			for _, rp := range req.Permissions {
				found := false
				for _, p := range d.Permissions {
					if snx_lib_core.PermissionSatisfiedBy(snx_lib_core.DelegationPermission(rp), snx_lib_core.DelegationPermission(p)) {
						found = true
						break
					}
				}
				if !found {
					allFound = false
					break
				}
			}
			if allFound {
				timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

				return &v4grpc.VerifySubaccountAuthorizationResponse{
					TimestampMs:       timestamp_ms,
					TimestampUs:       timestamp_us,
					IsAuthorized:      true,
					AuthorizationType: v4grpc.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
				}, nil
			}
		}
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	// No authorization found
	return &v4grpc.VerifySubaccountAuthorizationResponse{
		TimestampMs:       timestamp_ms,
		TimestampUs:       timestamp_us,
		IsAuthorized:      false,
		AuthorizationType: v4grpc.AuthorizationType_AUTHORIZATION_TYPE_NONE,
	}, nil
}

func (m *MockSubaccountServiceClient) VoluntaryAutoExchange(ctx context.Context, req *v4grpc.VoluntaryAutoExchangeRequest, opts ...grpc.CallOption) (*v4grpc.VoluntaryAutoExchangeResponse, error) {
	tm_now := snx_lib_utils_time.Now()

	// For testing purposes, return a successful auto-exchange response
	return &v4grpc.VoluntaryAutoExchangeResponse{
		TmRespondedAt:     timestamppb.New(tm_now),
		SubAccountId:      req.SubAccountId,
		SourceAsset:       req.SourceAsset,
		SourceAmountTaken: "100.0", // Mock amount taken
		TargetAsset:       "USDT",
		TargetAmount:      req.TargetUsdtAmount,
		IndexPrice:        "1.0", // Mock index price
		EffectiveHaircut:  "0.0", // No haircut for test
		Collateral: []*v4grpc.CollateralItem{
			{
				Symbol:   req.SourceAsset,
				Quantity: "900.0", // Remaining after exchange
			},
			{
				Symbol:   "USDT",
				Quantity: req.TargetUsdtAmount,
			},
		},
	}, nil
}

// UpdateSubAccountMarketLeverage implements the new leverage update method
func (m *MockSubaccountServiceClient) UpdateSubAccountMarketLeverage(ctx context.Context, req *v4grpc.UpdateSubAccountMarketLeverageRequest, opts ...grpc.CallOption) (*v4grpc.UpdateSubAccountMarketLeverageResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.UpdateSubAccountMarketLeverageResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Message:     "Leverage updated successfully",
	}, nil
}

func (m *MockSubaccountServiceClient) BatchUpdateLeverages(ctx context.Context, req *v4grpc.BatchUpdateLeveragesRequest, opts ...grpc.CallOption) (*v4grpc.BatchUpdateLeveragesResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.BatchUpdateLeveragesResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

// GetFundingPayments implements the new funding payments method
func (m *MockSubaccountServiceClient) GetFundingPayments(ctx context.Context, req *v4grpc.GetFundingPaymentsRequest, opts ...grpc.CallOption) (*v4grpc.GetFundingPaymentsResponse, error) {
	// Return mock funding payments data
	now := time.Now().UTC()
	then := now.Add(-time.Hour)

	fundingHistory := []*v4grpc.FundingPaymentItem{
		{
			PaymentId:    "fp_1",
			Symbol:       "BTC-USDT",
			PositionSize: "2.50000000",
			FundingRate:  "0.00001250",
			Payment:      "-1.12500000",
			PaymentTime:  timestamppb.New(now),
			FundingTime:  timestamppb.New(now),
		},
		{
			PaymentId:    "fp_2",
			Symbol:       "ETH-USDT",
			PositionSize: "-5.00000000",
			FundingRate:  "-0.00002100",
			Payment:      "3.15000000",
			PaymentTime:  timestamppb.New(then),
			FundingTime:  timestamppb.New(then),
		},
	}

	summary := &v4grpc.FundingSummary{
		TotalFundingReceived: "125.75000000",
		TotalFundingPaid:     "89.25000000",
		NetFunding:           "36.50000000",
		TotalPayments:        "247",
		AveragePaymentSize:   "0.87044534",
	}

	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetFundingPaymentsResponse{
		TimestampMs:    timestamp_ms,
		TimestampUs:    timestamp_us,
		Summary:        summary,
		FundingHistory: fundingHistory,
	}, nil
}

// GetPerformanceHistory returns empty histories for testing.
func (m *MockSubaccountServiceClient) GetPerformanceHistory(ctx context.Context, in *v4grpc.GetPerformanceHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetPerformanceHistoryResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	emptyPeriod := &v4grpc.PerformanceHistoryPeriod{
		History: []*v4grpc.PerformanceHistoryPoint{},
		Volume:  "0",
	}

	return &v4grpc.GetPerformanceHistoryResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Period:      in.Period,
		Performance: emptyPeriod,
	}, nil
}

// Insurance methods for testing
func (m *MockSubaccountServiceClient) GetInsuranceSubscription(ctx context.Context, req *v4grpc.GetInsuranceSubscriptionRequest, opts ...grpc.CallOption) (*v4grpc.GetInsuranceSubscriptionResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetInsuranceSubscriptionResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetInsuranceProtection(ctx context.Context, req *v4grpc.GetInsuranceProtectionRequest, opts ...grpc.CallOption) (*v4grpc.GetInsuranceProtectionResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetInsuranceProtectionResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) CreateDefaultInsuranceSubscription(ctx context.Context, req *v4grpc.CreateDefaultInsuranceSubscriptionRequest, opts ...grpc.CallOption) (*v4grpc.CreateDefaultInsuranceSubscriptionResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.CreateDefaultInsuranceSubscriptionResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetBalanceUpdates(ctx context.Context, req *v4grpc.GetBalanceUpdatesRequest, opts ...grpc.CallOption) (*v4grpc.GetBalanceUpdatesResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetBalanceUpdatesResponse{
		TimestampMs:    timestamp_ms,
		TimestampUs:    timestamp_us,
		BalanceUpdates: []*v4grpc.BalanceUpdateItem{},
	}, nil
}

func (m *MockSubaccountServiceClient) UnbrickPendingWithdraw(ctx context.Context, req *v4grpc.UnbrickPendingWithdrawRequest, opts ...grpc.CallOption) (*v4grpc.UnbrickPendingWithdrawResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.UnbrickPendingWithdrawResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		Success:     true,
		Message:     "mock unbrick success",
	}, nil
}

func (m *MockSubaccountServiceClient) TransferBetweenSubAccounts(ctx context.Context, req *v4grpc.TransferBetweenSubAccountsRequest, opts ...grpc.CallOption) (*v4grpc.TransferBetweenSubAccountsResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.TransferBetweenSubAccountsResponse{
		TimestampMs:   timestamp_ms,
		TimestampUs:   timestamp_us,
		Status:        "pending",
		RequestId:     req.RequestId,
		TransferId:    918023717031270,
		FromBalance:   "900",
		ToBalance:     req.Amount,
		TransferredAt: timestamppb.Now(),
	}, nil
}

func (m *MockSubaccountServiceClient) GetTransfers(ctx context.Context, req *v4grpc.GetTransfersRequest, opts ...grpc.CallOption) (*v4grpc.GetTransfersResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetTransfersResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetTradesForPosition(ctx context.Context, req *v4grpc.GetTradesForPositionRequest, opts ...grpc.CallOption) (*v4grpc.GetTradesForPositionResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetTradesForPositionResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) GetSnaxpotNumberPreference(
	ctx context.Context,
	req *v4grpc.GetSnaxpotNumberPreferenceRequest,
	opts ...grpc.CallOption,
) (*v4grpc.GetSnaxpotNumberPreferenceResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetSnaxpotNumberPreferenceResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	}, nil
}

func (m *MockSubaccountServiceClient) SetSnaxpotPreference(
	ctx context.Context,
	req *v4grpc.SetSnaxpotPreferenceRequest,
	opts ...grpc.CallOption,
) (*v4grpc.SetSnaxpotPreferenceResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.SetSnaxpotPreferenceResponse{
		TimestampMs:        timestamp_ms,
		TimestampUs:        timestamp_us,
		AppliedScope:       req.Scope,
		EpochId:            1,
		NumberMode:         "snax_only",
		UpdatedTicketCount: 0,
	}, nil
}

func (m *MockSubaccountServiceClient) ClearSnaxpotPreference(
	ctx context.Context,
	req *v4grpc.ClearSnaxpotPreferenceRequest,
	opts ...grpc.CallOption,
) (*v4grpc.ClearSnaxpotPreferenceResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.ClearSnaxpotPreferenceResponse{
		TimestampMs:        timestamp_ms,
		TimestampUs:        timestamp_us,
		AppliedScope:       req.Scope,
		EpochId:            1,
		NumberMode:         "auto",
		UpdatedTicketCount: 0,
	}, nil
}

func (m *MockSubaccountServiceClient) SaveSnaxpotTickets(
	ctx context.Context,
	req *v4grpc.SaveSnaxpotTicketsRequest,
	opts ...grpc.CallOption,
) (*v4grpc.SaveSnaxpotTicketsResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.SaveSnaxpotTicketsResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		EpochId:     1,
	}, nil
}

func (m *MockSubaccountServiceClient) GetSnaxpotTicketStatus(
	ctx context.Context,
	req *v4grpc.GetSnaxpotTicketStatusRequest,
	opts ...grpc.CallOption,
) (*v4grpc.GetSnaxpotTicketStatusResponse, error) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	return &v4grpc.GetSnaxpotTicketStatusResponse{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
		EpochId:     1,
	}, nil
}

// MockFailingSubaccountServiceClient implements a mock that always fails for testing error scenarios
type MockFailingSubaccountServiceClient struct {
	*MockSubaccountServiceClient
}

// NewMockFailingSubaccountServiceClient creates a mock that fails on ListSubaccounts calls
func NewMockFailingSubaccountServiceClient() *MockFailingSubaccountServiceClient {
	return &MockFailingSubaccountServiceClient{
		MockSubaccountServiceClient: NewMockSubaccountServiceClient(),
	}
}

// ListSubaccounts always returns an error to test service failure scenarios
func (m *MockFailingSubaccountServiceClient) ListSubaccounts(ctx context.Context, req *v4grpc.ListSubaccountsRequest, opts ...grpc.CallOption) (*v4grpc.ListSubaccountsResponse, error) {
	return nil, errServiceUnavailable
}

// GetDelegationsForDelegate always returns an error to test service failure scenarios
func (m *MockFailingSubaccountServiceClient) GetDelegationsForDelegate(ctx context.Context, req *v4grpc.GetDelegationsForDelegateRequest, opts ...grpc.CallOption) (*v4grpc.GetDelegationsForDelegateResponse, error) {
	return nil, errServiceUnavailable
}

// VerifySubaccountAuthorization always returns an error to test service failure scenarios
func (m *MockFailingSubaccountServiceClient) VerifySubaccountAuthorization(ctx context.Context, req *v4grpc.VerifySubaccountAuthorizationRequest, opts ...grpc.CallOption) (*v4grpc.VerifySubaccountAuthorizationResponse, error) {
	return nil, errServiceUnavailable
}

func (m *MockFailingSubaccountServiceClient) GetSubaccount(ctx context.Context, req *v4grpc.GetSubaccountRequest, opts ...grpc.CallOption) (*v4grpc.SubaccountInfo, error) {
	return nil, errServiceUnavailable
}

func (m *MockFailingSubaccountServiceClient) GetPerformanceHistory(ctx context.Context, in *v4grpc.GetPerformanceHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetPerformanceHistoryResponse, error) {
	return nil, errServiceUnavailable
}

// AddMockDelegation adds a mock delegation for a subaccount
func (m *MockSubaccountServiceClient) AddMockDelegation(
	subAccountId snx_lib_core.SubAccountId,
	delegateAddress snx_lib_api_types.WalletAddress,
	permissions []string,
	expiresAt *timestamppb.Timestamp,
) {
	info := &v4grpc.DelegationInfo{
		Id:              uint64(len(m.delegations[subAccountId]) + 1),
		SubAccountId:    int64(subAccountId),
		DelegateAddress: snx_lib_api_types.WalletAddressToString(delegateAddress),
		Permissions:     permissions,
		ExpiresAt:       expiresAt,
		CreatedAt:       timestamppb.New(time.Now().UTC()),
	}
	m.delegations[subAccountId] = append(m.delegations[subAccountId], info)
}

// AddMockDelegationForDelegate adds a mock delegation visible via GetDelegationsForDelegate
func (m *MockSubaccountServiceClient) AddMockDelegationForDelegate(
	delegateAddress snx_lib_api_types.WalletAddress,
	subAccountId snx_lib_core.SubAccountId,
) {
	info := &v4grpc.DelegationWithOwnerInfo{
		Id:              uint64(len(m.delegationsForDelegate[delegateAddress]) + 1),
		SubAccountId:    int64(subAccountId),
		DelegateAddress: snx_lib_api_types.WalletAddressToString(delegateAddress),
		CreatedAt:       timestamppb.New(snx_lib_utils_time.Now()),
	}
	m.delegationsForDelegate[delegateAddress] = append(m.delegationsForDelegate[delegateAddress], info)
}

// CreateTestAuthenticator creates a complete authenticator for tests, backed by
// an in-memory mock subaccount service wired through the authgrpc adapter.
func CreateTestAuthenticator() *snx_lib_auth.Authenticator {
	nonceStore := snx_lib_auth.NewTestNonceStore()
	mockSubaccountClient := NewMockSubaccountServiceClient()
	return snx_lib_auth.NewAuthenticator(
		nonceStore,
		snx_lib_authgrpc.NewVerifier(mockSubaccountClient),
		nil,
		snx_lib_auth.DefaultDomainName,
		"1",
		1,
	)
}

// CreateTestAuthenticatorWithAccount returns CreateTestAuthenticator pre-seeded
// with one (wallet, subaccount) ownership relationship.
func CreateTestAuthenticatorWithAccount(
	ethereumAddress snx_lib_api_types.WalletAddress,
	accountId snx_lib_core.SubAccountId,
) *snx_lib_auth.Authenticator {
	nonceStore := snx_lib_auth.NewTestNonceStore()
	mockSubaccountClient := NewMockSubaccountServiceClient()
	mockSubaccountClient.AddMockAccount(ethereumAddress, accountId)
	return snx_lib_auth.NewAuthenticator(
		nonceStore,
		snx_lib_authgrpc.NewVerifier(mockSubaccountClient),
		nil,
		snx_lib_auth.DefaultDomainName,
		"1",
		1,
	)
}

// TestWallet, NewTestWalletWithSeed and friends now live in lib/auth/test_helpers.go
// (no grpc dependency). Re-export them here so call sites that import authtest
// can keep using authtest.NewTestWalletWithSeed without a second import.
type TestWallet = snx_lib_auth.TestWallet

func NewTestWalletWithSeed(seed byte) *TestWallet {
	return snx_lib_auth.NewTestWalletWithSeed(seed)
}
