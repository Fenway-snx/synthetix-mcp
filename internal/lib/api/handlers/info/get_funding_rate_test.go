package info

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

// mockSubaccountClient embeds the interface - only override what we need
type mockSubaccountClient struct {
	v4grpc.SubaccountServiceClient // embedded interface satisfies all methods
	response                       *v4grpc.GetLatestFundingRatesResponse
	err                            error
}

func (m *mockSubaccountClient) GetLatestFundingRates(ctx context.Context, in *v4grpc.GetLatestFundingRatesRequest, opts ...grpc.CallOption) (*v4grpc.GetLatestFundingRatesResponse, error) {
	return m.response, m.err
}

func Test_Handle_getFundingRate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClient{
			response: &v4grpc.GetLatestFundingRatesResponse{
				FundingRates: []*v4grpc.GetLatestFundingRatesResponseItem{
					{
						Symbol:               "BTC-USDT",
						EstimatedFundingRate: "0.0001",
						LastSettlementRate:   "0.00009",
						NextFundingTime:      timestamppb.New(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)),
						LastSettlementTime:   timestamppb.New(time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)),
						FundingIntervalMs:    3600000,
					},
				},
			},
		}

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRate(ctx, map[string]any{"symbol": "BTC-USDT"})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)

		data := resp.Response.(FundingRateResponse)
		assert.Equal(t, Symbol("BTC-USDT"), data.Symbol)
		assert.Equal(t, "0.0001", data.EstimatedFundingRate)
		assert.Equal(t, "0.00009", data.LastSettlementRate)
		assert.Equal(t, int64(3600000), data.FundingInterval)
	})

	t.Run("nil last settlement time", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClient{
			response: &v4grpc.GetLatestFundingRatesResponse{
				FundingRates: []*v4grpc.GetLatestFundingRatesResponseItem{
					{
						Symbol:             "ETH-USDT",
						NextFundingTime:    timestamppb.New(time.Now()),
						LastSettlementTime: nil,
					},
				},
			},
		}

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRate(ctx, map[string]any{"symbol": "ETH-USDT"})

		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, Timestamp(0), resp.Response.(FundingRateResponse).LastSettlementTime)
	})

	t.Run("empty symbol", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRate(ctx, map[string]any{"symbol": ""})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Contains(t, resp.Error.Message, "symbol is required")
	})

	t.Run("grpc error", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClient{
			err: errors.New("unavailable"),
		}

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRate(ctx, map[string]any{"symbol": "BTC-USDT"})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
		assert.Contains(t, resp.Error.Message, "Failed to get funding rate")
	})

	t.Run("nil grpc response", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClient{
			response: nil,
			err:      nil,
		}

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRate(ctx, map[string]any{"symbol": "BTC-USDT"})

		assert.Equal(t, HTTPStatusCode_404_NotFound, status)
		assert.Contains(t, resp.Error.Message, "Funding rate not found")
	})

	t.Run("nil next funding time", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClient{
			response: &v4grpc.GetLatestFundingRatesResponse{
				FundingRates: []*v4grpc.GetLatestFundingRatesResponseItem{
					{
						Symbol:          "BTC-USDT",
						NextFundingTime: nil,
					},
				},
			},
		}

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil,
			mockClient,
			nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, _ := Handle_getFundingRate(ctx, map[string]any{"symbol": "BTC-USDT"})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
	})
}
