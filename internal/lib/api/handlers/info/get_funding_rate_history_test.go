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

// mockSubaccountClientHistory embeds the interface - only override what we need
type mockSubaccountClientHistory struct {
	v4grpc.SubaccountServiceClient // embedded interface satisfies all methods
	response                       *v4grpc.GetFundingRateHistoryResponse
	err                            error
}

func (m *mockSubaccountClientHistory) GetFundingRateHistory(ctx context.Context, in *v4grpc.GetFundingRateHistoryRequest, opts ...grpc.CallOption) (*v4grpc.GetFundingRateHistoryResponse, error) {
	return m.response, m.err
}

func Test_Handle_getFundingRateHistory(t *testing.T) {
	t.Parallel()

	// Helper timestamps for tests
	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	t.Run("success with multiple records", func(t *testing.T) {
		t.Parallel()

		fundingTime1 := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
		fundingTime2 := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		appliedAt1 := fundingTime1.Add(1 * time.Minute)
		appliedAt2 := fundingTime2.Add(1 * time.Minute)

		mockClient := &mockSubaccountClientHistory{
			response: &v4grpc.GetFundingRateHistoryResponse{
				Symbol: "BTC-USDT",
				FundingRates: []*v4grpc.FundingRateHistoryItem{
					{
						FundingRate: "0.0001",
						FundingTime: timestamppb.New(fundingTime1),
						AppliedAt:   timestamppb.New(appliedAt1),
					},
					{
						FundingRate: "0.00009",
						FundingTime: timestamppb.New(fundingTime2),
						AppliedAt:   timestamppb.New(appliedAt2),
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

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": startTime.UnixMilli(),
			"endTime":   endTime.UnixMilli(),
			"limit":     int32(100),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)

		data := resp.Response.(FundingRateHistoryResponse)
		assert.Equal(t, Symbol("BTC-USDT"), data.Symbol)
		require.Len(t, data.FundingRates, 2)
		assert.Equal(t, "0.0001", data.FundingRates[0].FundingRate)
		assert.Equal(t, "0.00009", data.FundingRates[1].FundingRate)
	})

	t.Run("success with empty funding rates", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClientHistory{
			response: &v4grpc.GetFundingRateHistoryResponse{
				Symbol:       "ETH-USDT",
				FundingRates: []*v4grpc.FundingRateHistoryItem{},
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

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "ETH-USDT",
			"startTime": startTime.UnixMilli(),
			"endTime":   endTime.UnixMilli(),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)

		data := resp.Response.(FundingRateHistoryResponse)
		assert.Equal(t, Symbol("ETH-USDT"), data.Symbol)
		assert.Empty(t, data.FundingRates)
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

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "",
			"startTime": startTime.UnixMilli(),
			"endTime":   endTime.UnixMilli(),
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Contains(t, resp.Error.Message, "symbol is required")
	})

	t.Run("missing startTime", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":  "BTC-USDT",
			"endTime": endTime.UnixMilli(),
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Contains(t, resp.Error.Message, "startTime is required")
	})

	t.Run("missing endTime", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": startTime.UnixMilli(),
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Contains(t, resp.Error.Message, "endTime is required")
	})

	t.Run("startTime equals endTime", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		sameTime := now.UnixMilli()
		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": sameTime,
			"endTime":   sameTime,
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Contains(t, resp.Error.Message, "startTime must be before endTime")
	})

	t.Run("startTime after endTime", func(t *testing.T) {
		t.Parallel()

		ctx := snx_lib_api_handlers_types.NewInfoContext(
			snx_lib_logging_doubles.NewStubLogger(),
			context.Background(),
			nil, nil, nil, nil, nil, nil, nil, nil,
			snx_lib_request.NewRequestID(),
			"req-id",
		)

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": endTime.UnixMilli(),
			"endTime":   startTime.UnixMilli(),
		})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, status)
		assert.Contains(t, resp.Error.Message, "startTime must be before endTime")
	})

	t.Run("grpc error", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClientHistory{
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

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": startTime.UnixMilli(),
			"endTime":   endTime.UnixMilli(),
		})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, status)
		assert.Contains(t, resp.Error.Message, "Failed to get funding rate history")
	})

	t.Run("nil funding time in response", func(t *testing.T) {
		t.Parallel()

		mockClient := &mockSubaccountClientHistory{
			response: &v4grpc.GetFundingRateHistoryResponse{
				Symbol: "BTC-USDT",
				FundingRates: []*v4grpc.FundingRateHistoryItem{
					{
						FundingRate: "0.0001",
						FundingTime: nil,
						AppliedAt:   nil,
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

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": startTime.UnixMilli(),
			"endTime":   endTime.UnixMilli(),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)

		data := resp.Response.(FundingRateHistoryResponse)
		assert.Equal(t, Timestamp(0), data.FundingRates[0].FundingTime)
		assert.Equal(t, Timestamp(0), data.FundingRates[0].AppliedAt)
	})

	t.Run("with limit parameter", func(t *testing.T) {
		t.Parallel()

		fundingTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

		mockClient := &mockSubaccountClientHistory{
			response: &v4grpc.GetFundingRateHistoryResponse{
				Symbol: "BTC-USDT",
				FundingRates: []*v4grpc.FundingRateHistoryItem{
					{
						FundingRate: "0.0001",
						FundingTime: timestamppb.New(fundingTime),
						AppliedAt:   timestamppb.New(fundingTime.Add(1 * time.Minute)),
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

		status, resp := Handle_getFundingRateHistory(ctx, map[string]any{
			"symbol":    "BTC-USDT",
			"startTime": startTime.UnixMilli(),
			"endTime":   endTime.UnixMilli(),
			"limit":     int32(10),
		})

		require.NotNil(t, resp)
		assert.Equal(t, HTTPStatusCode_200_OK, status)
		assert.Equal(t, "ok", resp.Status)
	})
}
