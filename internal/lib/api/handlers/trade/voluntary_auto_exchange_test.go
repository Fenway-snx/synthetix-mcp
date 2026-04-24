package trade

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// ---------------------------------------------------------------------------
// voluntaryAutoExchange mock
// ---------------------------------------------------------------------------

type mockTradingVoluntaryAutoExchange struct {
	v4grpc.TradingServiceClient
	resp *v4grpc.VoluntaryAutoExchangeResponse
	err  error
}

func (m *mockTradingVoluntaryAutoExchange) VoluntaryAutoExchange(_ context.Context, _ *v4grpc.VoluntaryAutoExchangeRequest, _ ...grpc.CallOption) (*v4grpc.VoluntaryAutoExchangeResponse, error) {
	return m.resp, m.err
}

// ===========================================================================
// Handle_voluntaryAutoExchange
// ===========================================================================

func Test_Handle_voluntaryAutoExchange(t *testing.T) {
	grpcResp := &v4grpc.VoluntaryAutoExchangeResponse{
		TmRespondedAt:     timestamppb.New(snx_lib_utils_time.Now()),
		SubAccountId:      100,
		SourceAsset:       "WETH",
		SourceAmountTaken: "1.5",
		TargetAsset:       "USDT",
		TargetAmount:      "3000.0",
		IndexPrice:        "2000.0",
		EffectiveHaircut:  "0.01",
		Collateral: []*v4grpc.CollateralItem{
			{Symbol: "USDT", Quantity: "5000.0"},
			{Symbol: "WETH", Quantity: "8.5"},
		},
	}

	t.Run("MissingPayload", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{}
		ctx := newActionTradeContext(100, mock, nil)

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("InvalidPayloadType", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{}
		ctx := newActionTradeContext(100, mock, "wrong-type")

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("Success", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{resp: grpcResp}
		validated := &ValidatedVoluntaryAutoExchangeAction{
			SourceAsset:      "WETH",
			TargetUSDTAmount: "3000.0",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_200_OK, code)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(VoluntaryAutoExchangeResponse)
		require.True(t, ok)
		assert.Equal(t, "WETH", data.SourceAsset)
		assert.Equal(t, "1.5", data.SourceAmountTaken)
		assert.Equal(t, "USDT", data.TargetAsset)
		assert.Equal(t, "3000.0", data.TargetAmount)
		assert.Equal(t, Price("2000.0"), data.IndexPrice)
		assert.Equal(t, "0.01", data.EffectiveHaircut)
		require.Len(t, data.Collateral, 2)
		assert.Equal(t, Asset("USDT"), data.Collateral[0].Symbol)
		assert.Equal(t, Quantity("5000.0"), data.Collateral[0].Quantity)
		assert.Equal(t, Asset("WETH"), data.Collateral[1].Symbol)
		assert.Equal(t, Quantity("8.5"), data.Collateral[1].Quantity)
	})

	t.Run("GRPCError/NotFound", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{err: status.Error(codes.NotFound, "sub-account not found")}
		validated := &ValidatedVoluntaryAutoExchangeAction{
			SourceAsset:      "WETH",
			TargetUSDTAmount: "100",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_404_NotFound, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCError/InvalidArgument", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{err: status.Error(codes.InvalidArgument, "bad input")}
		validated := &ValidatedVoluntaryAutoExchangeAction{
			SourceAsset:      "WETH",
			TargetUSDTAmount: "100",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCError/FailedPrecondition", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{err: status.Error(codes.FailedPrecondition, "insufficient balance")}
		validated := &ValidatedVoluntaryAutoExchangeAction{
			SourceAsset:      "WETH",
			TargetUSDTAmount: "100",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_400_BadRequest, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("GRPCError/Internal", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{err: status.Error(codes.Internal, "db failure")}
		validated := &ValidatedVoluntaryAutoExchangeAction{
			SourceAsset:      "WETH",
			TargetUSDTAmount: "100",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("NonGRPCError", func(t *testing.T) {
		mock := &mockTradingVoluntaryAutoExchange{err: errors.New("connection refused")}
		validated := &ValidatedVoluntaryAutoExchangeAction{
			SourceAsset:      "WETH",
			TargetUSDTAmount: "100",
		}
		ctx := newActionTradeContext(100, mock, validated)

		code, resp := Handle_voluntaryAutoExchange(ctx, HandlerParams{})

		assert.Equal(t, HTTPStatusCode_500_InternalServerError, code)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})
}
