package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// fundingPaymentsMock overrides GetFundingPayments on the shared mock client.
type fundingPaymentsMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	resp *v4grpc.GetFundingPaymentsResponse
	err  error
}

func (m *fundingPaymentsMock) GetFundingPayments(
	_ context.Context,
	_ *v4grpc.GetFundingPaymentsRequest,
	_ ...grpc.CallOption,
) (*v4grpc.GetFundingPaymentsResponse, error) {
	return m.resp, m.err
}

func Test_validateGetFundingPaymentsRequest(t *testing.T) {
	t.Run("limit 0 is valid", func(t *testing.T) {
		req := &GetFundingPaymentsRequest{Limit: 0}
		assert.NoError(t, validateGetFundingPaymentsRequest(req))
	})

	t.Run("positive limit within max is valid", func(t *testing.T) {
		req := &GetFundingPaymentsRequest{Limit: 500}
		assert.NoError(t, validateGetFundingPaymentsRequest(req))
	})

	t.Run("limit exactly at max boundary is valid", func(t *testing.T) {
		req := &GetFundingPaymentsRequest{Limit: maxFundingPaymentsLimit}
		assert.NoError(t, validateGetFundingPaymentsRequest(req))
	})

	t.Run("limit one above max returns errLimitTooLarge", func(t *testing.T) {
		req := &GetFundingPaymentsRequest{Limit: maxFundingPaymentsLimit + 1}
		err := validateGetFundingPaymentsRequest(req)
		assert.ErrorIs(t, err, errLimitTooLarge)
	})

	t.Run("very large limit returns errLimitTooLarge", func(t *testing.T) {
		req := &GetFundingPaymentsRequest{Limit: 999_999}
		err := validateGetFundingPaymentsRequest(req)
		assert.ErrorIs(t, err, errLimitTooLarge)
	})

	t.Run("negative limit returns errLimitInvalid", func(t *testing.T) {
		req := &GetFundingPaymentsRequest{Limit: -1}
		err := validateGetFundingPaymentsRequest(req)
		assert.ErrorIs(t, err, errLimitInvalid)
	})

	t.Run("valid symbol is normalised and accepted", func(t *testing.T) {
		req := &GetFundingPaymentsRequest{Symbol: "btc-usd"}
		err := validateGetFundingPaymentsRequest(req)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.NotEmpty(t, req.Symbol)
	})
}

func Test_Handle_getFundingPayments(t *testing.T) {
	emptySuccessResp := &v4grpc.GetFundingPaymentsResponse{
		Summary:        &v4grpc.FundingSummary{},
		FundingHistory: []*v4grpc.FundingPaymentItem{},
	}

	t.Run("error - no subAccountId", func(t *testing.T) {
		ctx := createTestTradeContext("req-fp-1", 0, nil)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("error - negative limit", func(t *testing.T) {
		ctx := createTestTradeContext("req-fp-2", 100, nil)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{
			"limit": -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("error - limit exceeds max", func(t *testing.T) {
		ctx := createTestTradeContext("req-fp-3", 100, nil)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{
			"limit": maxFundingPaymentsLimit + 1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("success - empty funding history", func(t *testing.T) {
		mock := &fundingPaymentsMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp:                        emptySuccessResp,
		}
		ctx := createTestTradeContextWithClient("req-fp-4", 100, nil, mock)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(GetFundingPaymentsResponse)
		require.True(t, ok)
		assert.Empty(t, data.FundingHistory)
	})

	t.Run("success - limit at max boundary is accepted", func(t *testing.T) {
		mock := &fundingPaymentsMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp:                        emptySuccessResp,
		}
		ctx := createTestTradeContextWithClient("req-fp-5", 100, nil, mock)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{
			"limit": maxFundingPaymentsLimit,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("success - maps summary and history fields", func(t *testing.T) {
		ts := timestamppb.Now()
		mock := &fundingPaymentsMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetFundingPaymentsResponse{
				Summary: &v4grpc.FundingSummary{
					TotalFundingReceived: "100.5",
					TotalFundingPaid:     "50.25",
					NetFunding:           "50.25",
					TotalPayments:        "10",
					AveragePaymentSize:   "15.075",
				},
				FundingHistory: []*v4grpc.FundingPaymentItem{
					{
						PaymentId:    "pay-1",
						Symbol:       "BTC-USD",
						PositionSize: "1.5",
						FundingRate:  "0.0001",
						Payment:      "-15.0",
						PaymentTime:  ts,
						FundingTime:  ts,
					},
				},
			},
		}

		ctx := createTestTradeContextWithClient("req-fp-6", 100, nil, mock)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(GetFundingPaymentsResponse)
		require.True(t, ok)

		assert.Equal(t, "100.5", data.Summary.TotalFundingReceived)
		assert.Equal(t, "50.25", data.Summary.TotalFundingPaid)
		assert.Equal(t, "50.25", data.Summary.NetFunding)
		assert.Equal(t, "10", data.Summary.TotalPayments)
		assert.Equal(t, "15.075", data.Summary.AveragePaymentSize)

		require.Len(t, data.FundingHistory, 1)
		payment := data.FundingHistory[0]
		assert.Equal(t, "pay-1", payment.PaymentId)
		assert.Equal(t, Symbol("BTC-USD"), payment.Symbol)
		assert.Equal(t, "1.5", payment.PositionSize)
		assert.Equal(t, "0.0001", payment.FundingRate)
		assert.Equal(t, "-15.0", payment.Payment)

		// Deprecated fields mirror the canonical ones
		assert.Equal(t, payment.PaymentTime, payment.DEPRECATED_Timestamp_NOW_PaymentTime)
		assert.Equal(t, payment.FundingTime, payment.DEPRECATED_FundingTimestamp_NOW_FundingTime)
	})

	t.Run("error - gRPC failure returns 500", func(t *testing.T) {
		mock := &fundingPaymentsMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			err:                         errors.New("connection refused"),
		}

		ctx := createTestTradeContextWithClient("req-fp-7", 100, nil, mock)
		statusCode, resp := Handle_getFundingPayments(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		assert.Equal(t, "error", resp.Status)
	})
}
