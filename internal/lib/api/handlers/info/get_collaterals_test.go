package info

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

func Test_Handle_getCollaterals(t *testing.T) {
	t.Run("returns collaterals with tiers", func(t *testing.T) {
		mc := new(MockMarketConfigClient)
		mc.On("GetCollateralConfigs", mock.Anything, mock.Anything, mock.Anything).
			Return(&v4grpc.GetCollateralConfigsResponse{
				Collaterals: []*v4grpc.CollateralConfig{
					{
						Collateral:  "USDT",
						Market:      "USDTUSD",
						DepositCap:  "10000000.00000000",
						Ltv:         "0.95",
						Lltv:        "0.98",
						WithdrawFee: "5.00000000",
						Tiers: []*v4grpc.CollateralHaircutTierConfig{
							{
								Id:            1,
								Name:          "Tier1",
								MinAmount:     "0.00000000",
								MaxAmount:     "100000.00000000",
								ValueRatio:    "1.00000000",
								Haircut:       "0.00000000",
								ValueAddition: "0.00000000",
							},
							{
								Id:            2,
								Name:          "Tier2",
								MinAmount:     "100000.00000000",
								MaxAmount:     "",
								ValueRatio:    "0.99500000",
								Haircut:       "0.00500000",
								ValueAddition: "0.00000000",
							},
						},
					},
					{
						Collateral:  "ETH",
						Market:      "ETHUSD",
						DepositCap:  "5000.00000000",
						Ltv:         "0.90",
						Lltv:        "0.95",
						WithdrawFee: "5.00000000",
						Tiers: []*v4grpc.CollateralHaircutTierConfig{
							{
								Id:            3,
								Name:          "Tier1",
								MinAmount:     "0.00000000",
								MaxAmount:     "50000.00000000",
								ValueRatio:    "0.98000000",
								Haircut:       "0.02000000",
								ValueAddition: "0.00000000",
							},
						},
					},
				},
			}, nil)

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  snx_lib_logging_doubles.NewStubLogger(),
			},
			MarketConfigClient: mc,
			ClientRequestId:    "test-request",
		}

		status, response := Handle_getCollaterals(ctx, map[string]any{})

		assert.Equal(t, http.StatusOK, int(status))
		assert.Equal(t, "ok", response.Status)

		collaterals, ok := response.Response.([]CollateralConfigResponse)
		assert.True(t, ok, "response should be []CollateralConfigResponse")
		assert.Len(t, collaterals, 2)

		usdt := collaterals[0]
		assert.Equal(t, "USDT", usdt.Collateral)
		assert.Equal(t, "USDTUSD", usdt.Market)
		assert.Equal(t, "10000000.00000000", usdt.DepositCap)
		assert.Equal(t, "0.95", usdt.LTV)
		assert.Equal(t, "0.98", usdt.LLTV)
		assert.Equal(t, "5.00000000", usdt.WithdrawFee)
		assert.Len(t, usdt.Tiers, 2)

		tier1 := usdt.Tiers[0]
		assert.Equal(t, int64(1), tier1.ID)
		assert.Equal(t, "Tier1", tier1.Name)
		assert.Equal(t, "0.00000000", tier1.MinAmount)
		assert.Equal(t, "100000.00000000", tier1.MaxAmount)
		assert.Equal(t, "1.00000000", tier1.ValueRatio)
		assert.Equal(t, "0.00000000", tier1.Haircut)
		assert.Equal(t, "0.00000000", tier1.ValueAddition)

		tier2 := usdt.Tiers[1]
		assert.Equal(t, int64(2), tier2.ID)
		assert.Equal(t, "Tier2", tier2.Name)
		assert.Equal(t, "", tier2.MaxAmount)
		assert.Equal(t, "0.00500000", tier2.Haircut)

		eth := collaterals[1]
		assert.Equal(t, "ETH", eth.Collateral)
		assert.Equal(t, "ETHUSD", eth.Market)
		assert.Len(t, eth.Tiers, 1)

		mc.AssertExpectations(t)
	})

	t.Run("returns empty array when no collaterals configured", func(t *testing.T) {
		mc := new(MockMarketConfigClient)
		mc.On("GetCollateralConfigs", mock.Anything, mock.Anything, mock.Anything).
			Return(&v4grpc.GetCollateralConfigsResponse{}, nil)

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  snx_lib_logging_doubles.NewStubLogger(),
			},
			MarketConfigClient: mc,
			ClientRequestId:    "test-request",
		}

		status, response := Handle_getCollaterals(ctx, map[string]any{})

		assert.Equal(t, http.StatusOK, int(status))
		assert.Equal(t, "ok", response.Status)

		collaterals, ok := response.Response.([]CollateralConfigResponse)
		assert.True(t, ok, "response should be []CollateralConfigResponse")
		assert.Empty(t, collaterals)

		mc.AssertExpectations(t)
	})

	t.Run("returns 500 when gRPC call fails", func(t *testing.T) {
		mc := new(MockMarketConfigClient)
		mc.On("GetCollateralConfigs", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("service unavailable"))

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  snx_lib_logging_doubles.NewStubLogger(),
			},
			MarketConfigClient: mc,
			ClientRequestId:    "test-request",
		}

		status, response := Handle_getCollaterals(ctx, map[string]any{})

		assert.Equal(t, http.StatusInternalServerError, int(status))
		assert.Equal(t, "error", response.Status)
		assert.NotNil(t, response.Error)

		mc.AssertExpectations(t)
	})

	t.Run("collateral with no tiers returns empty tiers array", func(t *testing.T) {
		mc := new(MockMarketConfigClient)
		mc.On("GetCollateralConfigs", mock.Anything, mock.Anything, mock.Anything).
			Return(&v4grpc.GetCollateralConfigsResponse{
				Collaterals: []*v4grpc.CollateralConfig{
					{
						Collateral:  "USDT",
						Market:      "USDTUSD",
						DepositCap:  "10000000.00000000",
						Ltv:         "0.95",
						Lltv:        "0.98",
						WithdrawFee: "5.00000000",
					},
				},
			}, nil)

		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Context: context.Background(),
				Logger:  snx_lib_logging_doubles.NewStubLogger(),
			},
			MarketConfigClient: mc,
			ClientRequestId:    "test-request",
		}

		status, response := Handle_getCollaterals(ctx, map[string]any{})

		assert.Equal(t, http.StatusOK, int(status))
		collaterals, ok := response.Response.([]CollateralConfigResponse)
		assert.True(t, ok)
		assert.Len(t, collaterals, 1)
		assert.Empty(t, collaterals[0].Tiers)

		mc.AssertExpectations(t)
	})
}
