package info

import (
	"context"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// MockMarketConfigClient is a shared mock for the MarketConfigServiceClient interface
// used across multiple test files in this package.
type MockMarketConfigClient struct {
	mock.Mock
}

func (m *MockMarketConfigClient) GetAllMarketPrices(ctx context.Context, in *v4grpc.GetAllMarketPricesRequest, opts ...grpc.CallOption) (*v4grpc.GetAllMarketPricesResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetAllMarketPricesResponse), args.Error(1)
}

func (m *MockMarketConfigClient) GetAllMarkets(ctx context.Context, in *v4grpc.GetAllMarketsRequest, opts ...grpc.CallOption) (*v4grpc.GetAllMarketsResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetAllMarketsResponse), args.Error(1)
}

func (m *MockMarketConfigClient) GetMarket(ctx context.Context, in *v4grpc.GetMarketRequest, opts ...grpc.CallOption) (*v4grpc.GetMarketResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetMarketResponse), args.Error(1)
}

func (m *MockMarketConfigClient) GetFundingSettings(ctx context.Context, in *v4grpc.GetFundingSettingsRequest, opts ...grpc.CallOption) (*v4grpc.GetFundingSettingsResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetFundingSettingsResponse), args.Error(1)
}

func (m *MockMarketConfigClient) UpdateMarketStatus(ctx context.Context, in *v4grpc.UpdateMarketStatusRequest, opts ...grpc.CallOption) (*v4grpc.UpdateMarketStatusResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.UpdateMarketStatusResponse), args.Error(1)
}

func (m *MockMarketConfigClient) GetExchangeConfigs(ctx context.Context, in *v4grpc.GetExchangeConfigsRequest, opts ...grpc.CallOption) (*v4grpc.GetExchangeConfigsResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetExchangeConfigsResponse), args.Error(1)
}

func (m *MockMarketConfigClient) CreateMarket(ctx context.Context, in *v4grpc.CreateMarketRequest, opts ...grpc.CallOption) (*v4grpc.CreateMarketResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.CreateMarketResponse), args.Error(1)
}

func (m *MockMarketConfigClient) GetAllSLPExposureLimits(ctx context.Context, in *v4grpc.GetAllSLPExposureLimitsRequest, opts ...grpc.CallOption) (*v4grpc.GetAllSLPExposureLimitsResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetAllSLPExposureLimitsResponse), args.Error(1)
}

func (m *MockMarketConfigClient) GetCollateralConfigs(ctx context.Context, in *v4grpc.GetCollateralConfigsRequest, opts ...grpc.CallOption) (*v4grpc.GetCollateralConfigsResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetCollateralConfigsResponse), args.Error(1)
}

func (m *MockMarketConfigClient) GetCollateralPricingConfigs(ctx context.Context, in *v4grpc.GetCollateralPricingConfigsRequest, opts ...grpc.CallOption) (*v4grpc.GetCollateralPricingConfigsResponse, error) {
	args := m.Called(ctx, in, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4grpc.GetCollateralPricingConfigsResponse), args.Error(1)
}

// CreateTestMarket creates a test market with the given symbol for use in tests.
func CreateTestMarket(symbol string) *v4grpc.Market {
	return &v4grpc.Market{
		Symbol:                     symbol,
		BaseAsset:                  symbol[:3],
		QuoteAsset:                 "USD",
		IsOpen:                     true,
		PriceExponent:              2,
		QuantityExponent:           8,
		TickSize:                   "0.01",
		MinTradeAmount:             "0.001",
		ContractSize:               "1",
		MaxMarketOrderAmount:       "100",
		MaxLimitOrderAmount:        "1000",
		LimitOrderPriceCapRatio:    "1.1",
		LimitOrderPriceFloorRatio:  "0.9",
		MarketOrderPriceCapRatio:   "1.05",
		MarketOrderPriceFloorRatio: "0.95",
		LiquidationClearanceFee:    "0.001",
		MinNotionalValue:           "10",
		MaintenanceMarginTiers: []*v4grpc.MaintenanceMarginTier{
			{
				MaintenanceMarginRatio: "0.05",
				InitialMarginRatio:     "0.1",
			},
		},
	}
}
