package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// balanceUpdatesMock embeds the real mock but overrides GetBalanceUpdates to return custom data.
type balanceUpdatesMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	resp *v4grpc.GetBalanceUpdatesResponse
}

func (m *balanceUpdatesMock) GetBalanceUpdates(ctx context.Context, req *v4grpc.GetBalanceUpdatesRequest, opts ...grpc.CallOption) (*v4grpc.GetBalanceUpdatesResponse, error) {
	return m.resp, nil
}

// capturingBalanceUpdatesMock records the last GetBalanceUpdates gRPC request.
type capturingBalanceUpdatesMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	captured *v4grpc.GetBalanceUpdatesRequest
}

func (m *capturingBalanceUpdatesMock) GetBalanceUpdates(ctx context.Context, req *v4grpc.GetBalanceUpdatesRequest, opts ...grpc.CallOption) (*v4grpc.GetBalanceUpdatesResponse, error) {
	m.captured = req
	return &v4grpc.GetBalanceUpdatesResponse{
		BalanceUpdates: []*v4grpc.BalanceUpdateItem{},
	}, nil
}

// failingBalanceUpdatesMock returns an error from GetBalanceUpdates.
type failingBalanceUpdatesMock struct {
	*snx_lib_authtest.MockSubaccountServiceClient
}

func (m *failingBalanceUpdatesMock) GetBalanceUpdates(ctx context.Context, req *v4grpc.GetBalanceUpdatesRequest, opts ...grpc.CallOption) (*v4grpc.GetBalanceUpdatesResponse, error) {
	return nil, errors.New("grpc unavailable")
}

func Test_applyBalanceUpdatesDefaultTimeWindow(t *testing.T) {
	const windowMs int64 = 7 * 24 * 60 * 60 * 1000

	t.Run("both unset fills end to now and start seven days earlier", func(t *testing.T) {
		nowMs := int64(1706745600000)
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(nowMs).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(nowMs), req.EndTime)
		assert.Equal(t, Timestamp(nowMs-windowMs), req.StartTime)
	})

	t.Run("both unset uses 1ms start when now minus window is non-positive", func(t *testing.T) {
		nowMs := int64(1000)
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(nowMs).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(nowMs), req.EndTime)
		assert.Equal(t, Timestamp(1), req.StartTime)
	})

	t.Run("no-op when both set", func(t *testing.T) {
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(1_000_000).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{StartTime: 100, EndTime: 200}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(100), req.StartTime)
		assert.Equal(t, Timestamp(200), req.EndTime)
	})

	t.Run("start only fills end seven days later", func(t *testing.T) {
		start := Timestamp(1704067200000)
		nowMs := int64(1705000000000)
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(nowMs).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{StartTime: start}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(1704067200000+windowMs), req.EndTime)
	})

	t.Run("start only caps end to now", func(t *testing.T) {
		start := Timestamp(1700000000000)
		nowMs := int64(1700000001000)
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(nowMs).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{StartTime: start}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(nowMs), req.EndTime)
	})

	t.Run("end only fills start seven days earlier", func(t *testing.T) {
		end := Timestamp(1706745600000)
		nowMs := int64(1708000000000)
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(nowMs).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{EndTime: end}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(1706745600000-windowMs), req.StartTime)
		assert.Equal(t, end, req.EndTime)
	})

	t.Run("end only caps end to now when end is in the future", func(t *testing.T) {
		endMs := int64(1708000000000)
		nowMs := int64(1706745600000)
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(nowMs).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{EndTime: Timestamp(endMs)}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(nowMs), req.EndTime)
		assert.Equal(t, Timestamp(nowMs-windowMs), req.StartTime)
	})

	t.Run("end only uses 1ms when window would be non-positive", func(t *testing.T) {
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(1708000000000).UTC(),
		))
		t.Cleanup(cleanup)

		req := &GetBalanceUpdatesRequest{EndTime: Timestamp(1_000_000)}
		applyBalanceUpdatesDefaultTimeWindow(req)
		assert.Equal(t, Timestamp(1), req.StartTime)
		assert.Equal(t, Timestamp(1_000_000), req.EndTime)
	})

	t.Run("both set with start after end fails ValidateTimestampRange", func(t *testing.T) {
		req := &GetBalanceUpdatesRequest{StartTime: 200, EndTime: 100}
		err := snx_lib_api_validation.ValidateTimestampRange(req.StartTime, req.EndTime, 0, "balanceUpdates")
		require.Error(t, err)
	})
}

func Test_validateGetBalanceUpdatesRequest(t *testing.T) {
	t.Run("limit exceeds max", func(t *testing.T) {
		err := validateGetBalanceUpdatesRequest(&GetBalanceUpdatesRequest{Limit: 1001})
		require.Error(t, err)
	})

	t.Run("offset exceeds max", func(t *testing.T) {
		err := validateGetBalanceUpdatesRequest(&GetBalanceUpdatesRequest{Offset: 10001})
		require.Error(t, err)
	})

	t.Run("negative limit", func(t *testing.T) {
		err := validateGetBalanceUpdatesRequest(&GetBalanceUpdatesRequest{Limit: -1})
		require.Error(t, err)
	})

	t.Run("negative offset", func(t *testing.T) {
		err := validateGetBalanceUpdatesRequest(&GetBalanceUpdatesRequest{Offset: -1})
		require.Error(t, err)
	})

	t.Run("time range exceeds 365 days", func(t *testing.T) {
		start := Timestamp(1_000_000)
		end := start + Timestamp(366*86400*1000)
		err := validateGetBalanceUpdatesRequest(&GetBalanceUpdatesRequest{
			StartTime: start,
			EndTime:   end,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "balanceUpdates")
	})

	t.Run("actionFilter exceeds maximum length", func(t *testing.T) {
		tooLong := strings.Repeat("a", snx_lib_api_validation.MaxEnumFieldLength*4+1)
		err := validateGetBalanceUpdatesRequest(&GetBalanceUpdatesRequest{ActionFilter: tooLong})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actionFilter")
	})

	t.Run("valid at boundaries", func(t *testing.T) {
		start := Timestamp(1_000_000)
		end := start + Timestamp(365*86400*1000)
		err := validateGetBalanceUpdatesRequest(&GetBalanceUpdatesRequest{
			Limit:        1000,
			Offset:       10000,
			StartTime:    start,
			EndTime:      end,
			ActionFilter: "",
		})
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})
}

func Test_Handle_getBalanceUpdates(t *testing.T) {
	t.Run("success - maps FromSubAccountId and ToSubAccountId", func(t *testing.T) {
		fromId := int64(100)
		toId := int64(200)

		mock := &balanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetBalanceUpdatesResponse{
				BalanceUpdates: []*v4grpc.BalanceUpdateItem{
					{
						Id:               1,
						SubAccountId:     100,
						Action:           "TRANSFER",
						Status:           "success",
						Amount:           "500.0",
						Collateral:       "USDT",
						CreatedAt:        timestamppb.Now(),
						FromSubAccountId: &fromId,
						ToSubAccountId:   &toId,
					},
				},
			},
		}

		ctx := createTestTradeContextWithClient("req-1", 100, nil, mock)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "ok", resp.Status)

		data, ok := resp.Response.(GetBalanceUpdatesResponse)
		require.True(t, ok)
		require.Len(t, data.BalanceUpdates, 1)

		update := data.BalanceUpdates[0]
		assert.Equal(t, "1", update.Id)
		assert.Equal(t, "TRANSFER", update.Action)
		assert.Equal(t, "success", update.Status)
		assert.Equal(t, "500.0", update.Amount)
		assert.Equal(t, "USDT", update.Collateral)

		require.NotNil(t, update.FromSubAccountId)
		assert.Equal(t, SubAccountId("100"), *update.FromSubAccountId)
		require.NotNil(t, update.ToSubAccountId)
		assert.Equal(t, SubAccountId("200"), *update.ToSubAccountId)
	})

	t.Run("success - optional fields nil when not set", func(t *testing.T) {
		mock := &balanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetBalanceUpdatesResponse{
				BalanceUpdates: []*v4grpc.BalanceUpdateItem{
					{
						Id:           2,
						SubAccountId: 100,
						Action:       "DEPOSIT",
						Status:       "completed",
						Amount:       "1000.0",
						Collateral:   "USDT",
						CreatedAt:    timestamppb.Now(),
					},
				},
			},
		}

		ctx := createTestTradeContextWithClient("req-2", 100, nil, mock)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		data, ok := resp.Response.(GetBalanceUpdatesResponse)
		require.True(t, ok)
		require.Len(t, data.BalanceUpdates, 1)

		update := data.BalanceUpdates[0]
		assert.Nil(t, update.FromSubAccountId)
		assert.Nil(t, update.ToSubAccountId)
		assert.Nil(t, update.DestinationAddress)
		assert.Nil(t, update.TxHash)
	})

	t.Run("success - maps DestinationAddress and TxHash", func(t *testing.T) {
		dest := "0xabc"
		txHash := "0xdef"

		mock := &balanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetBalanceUpdatesResponse{
				BalanceUpdates: []*v4grpc.BalanceUpdateItem{
					{
						Id:                 3,
						SubAccountId:       100,
						Action:             "WITHDRAWAL",
						Status:             "completed",
						Amount:             "250.0",
						Collateral:         "USDT",
						CreatedAt:          timestamppb.Now(),
						DestinationAddress: &dest,
						TxHash:             &txHash,
					},
				},
			},
		}

		ctx := createTestTradeContextWithClient("req-3", 100, nil, mock)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		data, ok := resp.Response.(GetBalanceUpdatesResponse)
		require.True(t, ok)
		require.Len(t, data.BalanceUpdates, 1)

		update := data.BalanceUpdates[0]
		require.NotNil(t, update.DestinationAddress)
		assert.Equal(t, WalletAddress("0xabc"), *update.DestinationAddress)
		require.NotNil(t, update.TxHash)
		assert.Equal(t, TxHash("0xdef"), *update.TxHash)
	})

	t.Run("error - no subAccountId", func(t *testing.T) {
		ctx := createTestTradeContext("req-4", 0, nil)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("error - invalid action filter", func(t *testing.T) {
		ctx := createTestTradeContext("req-5", 100, nil)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{
			"actionFilter": "INVALID",
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("validation error - negative startTime", func(t *testing.T) {
		ctx := createTestTradeContext("req-neg-st", 100, nil)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{
			"startTime": -1,
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("validation error - startTime after endTime", func(t *testing.T) {
		ctx := createTestTradeContext("req-bad-range", 100, nil)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{
			"startTime": int64(2000),
			"endTime":   int64(1000),
		})

		require.Equal(t, HTTPStatusCode_400_BadRequest, statusCode)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("success - omits start and end on request still sends default window on grpc", func(t *testing.T) {
		mock := &capturingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("req-omit", 100, nil, mock)
		statusCode, _ := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.StartTime)
		require.NotNil(t, mock.captured.EndTime)
		endMs := mock.captured.EndTime.AsTime().UnixMilli()
		startMs := mock.captured.StartTime.AsTime().UnixMilli()
		assert.InDelta(t, 7*24*60*60*1000, endMs-startMs, 2000)
		assert.InDelta(t, time.Now().UnixMilli(), endMs, 3000)
	})

	t.Run("success - forwards startTime and endTime to grpc", func(t *testing.T) {
		mock := &capturingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		startMs := int64(1704067200000)
		endMs := int64(1706745600000)
		ctx := createTestTradeContextWithClient("req-fw", 100, nil, mock)
		statusCode, _ := Handle_getBalanceUpdates(ctx, map[string]any{
			"startTime": startMs,
			"endTime":   endMs,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.StartTime)
		require.NotNil(t, mock.captured.EndTime)
		assert.Equal(t, startMs, mock.captured.StartTime.AsTime().UnixMilli())
		assert.Equal(t, endMs, mock.captured.EndTime.AsTime().UnixMilli())
	})

	t.Run("success - only startTime forwards seven-day window end", func(t *testing.T) {
		mock := &capturingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		startMs := int64(1704067200000)
		endMs := startMs + 7*24*60*60*1000
		ctx := createTestTradeContextWithClient("req-start-only", 100, nil, mock)
		statusCode, _ := Handle_getBalanceUpdates(ctx, map[string]any{
			"startTime": startMs,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.StartTime)
		require.NotNil(t, mock.captured.EndTime)
		assert.Equal(t, startMs, mock.captured.StartTime.AsTime().UnixMilli())
		assert.Equal(t, endMs, mock.captured.EndTime.AsTime().UnixMilli())
	})

	t.Run("success - only endTime forwards seven-day window start", func(t *testing.T) {
		mock := &capturingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		endMs := int64(1706745600000)
		startMs := endMs - 7*24*60*60*1000
		ctx := createTestTradeContextWithClient("req-end-only", 100, nil, mock)
		statusCode, _ := Handle_getBalanceUpdates(ctx, map[string]any{
			"endTime": endMs,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.StartTime)
		require.NotNil(t, mock.captured.EndTime)
		assert.Equal(t, startMs, mock.captured.StartTime.AsTime().UnixMilli())
		assert.Equal(t, endMs, mock.captured.EndTime.AsTime().UnixMilli())
	})

	t.Run("error - grpc failure returns 500", func(t *testing.T) {
		mock := &failingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("req-grpc-err", 100, nil, mock)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_500_InternalServerError, statusCode)
		require.NotNil(t, resp)
		assert.Equal(t, "error", resp.Status)
	})

	t.Run("success - CreatedAt nil yields zero timestamp", func(t *testing.T) {
		mock := &balanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			resp: &v4grpc.GetBalanceUpdatesResponse{
				BalanceUpdates: []*v4grpc.BalanceUpdateItem{
					{
						Id:           99,
						SubAccountId: 100,
						Action:       "DEPOSIT",
						Status:       "completed",
						Amount:       "1",
						Collateral:   "USDT",
						CreatedAt:    nil,
					},
				},
			},
		}

		ctx := createTestTradeContextWithClient("req-no-created", 100, nil, mock)
		statusCode, resp := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		data, ok := resp.Response.(GetBalanceUpdatesResponse)
		require.True(t, ok)
		require.Len(t, data.BalanceUpdates, 1)
		assert.Equal(t, Timestamp(0), data.BalanceUpdates[0].Timestamp)
	})

	t.Run("success - default limit 50 and offset forwarded on grpc", func(t *testing.T) {
		mock := &capturingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("req-limit-off", 100, nil, mock)
		statusCode, _ := Handle_getBalanceUpdates(ctx, map[string]any{
			"offset": int64(42),
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		assert.Equal(t, int64(50), mock.captured.Limit)
		assert.Equal(t, int64(42), mock.captured.Offset)
	})

	t.Run("success - empty actionFilter sends no action_filter on grpc", func(t *testing.T) {
		mock := &capturingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("req-no-filter", 100, nil, mock)
		statusCode, _ := Handle_getBalanceUpdates(ctx, map[string]any{})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		assert.Empty(t, mock.captured.ActionFilter)
	})

	t.Run("success - only endTime in future is capped to now on grpc", func(t *testing.T) {
		nowMs := int64(1706745600000)
		endMs := int64(2000000000000)
		cleanup := snx_lib_utils_time.SetTimeProvider(snx_lib_utils_time.NewFixedTimeProvider(
			time.UnixMilli(nowMs).UTC(),
		))
		t.Cleanup(cleanup)

		mock := &capturingBalanceUpdatesMock{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}
		ctx := createTestTradeContextWithClient("req-end-future", 100, nil, mock)
		statusCode, _ := Handle_getBalanceUpdates(ctx, map[string]any{
			"endTime": endMs,
		})

		require.Equal(t, HTTPStatusCode_200_OK, statusCode)
		require.NotNil(t, mock.captured)
		require.NotNil(t, mock.captured.EndTime)
		assert.Equal(t, nowMs, mock.captured.EndTime.AsTime().UnixMilli())
	})
}

func Test_ValidateActionFilters(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		result, err := validateActionFilters("")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Nil(t, result)
	})

	t.Run("whitespace only returns nil", func(t *testing.T) {
		result, err := validateActionFilters("   ")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Nil(t, result)
	})

	t.Run("DEPOSIT is valid", func(t *testing.T) {
		result, err := validateActionFilters("DEPOSIT")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, []string{"DEPOSIT"}, result)
	})

	t.Run("WITHDRAWAL is valid", func(t *testing.T) {
		result, err := validateActionFilters("WITHDRAWAL")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, []string{"WITHDRAWAL"}, result)
	})

	t.Run("TRANSFER is valid", func(t *testing.T) {
		result, err := validateActionFilters("TRANSFER")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, []string{"TRANSFER"}, result)
	})

	t.Run("all three comma-separated", func(t *testing.T) {
		result, err := validateActionFilters("DEPOSIT,WITHDRAWAL,TRANSFER")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, []string{"DEPOSIT", "WITHDRAWAL", "TRANSFER"}, result)
	})

	t.Run("handles whitespace around filters", func(t *testing.T) {
		result, err := validateActionFilters(" DEPOSIT , TRANSFER ")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, []string{"DEPOSIT", "TRANSFER"}, result)
	})

	t.Run("skips empty segments", func(t *testing.T) {
		result, err := validateActionFilters("DEPOSIT,,TRANSFER")
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, []string{"DEPOSIT", "TRANSFER"}, result)
	})

	t.Run("invalid filter returns error mentioning TRANSFER", func(t *testing.T) {
		_, err := validateActionFilters("INVALID")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TRANSFER")
	})
}
