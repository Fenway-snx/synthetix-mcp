package trade

import (
	snx_lib_authtest "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth/authtest"
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	snx_lib_api_handlers_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/types"
	snx_lib_api_validation "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

type mockSnaxpotSubaccountClient struct {
	*snx_lib_authtest.MockSubaccountServiceClient
	clearPreferenceErr   error
	clearPreferenceReq   *v4grpc.ClearSnaxpotPreferenceRequest
	clearPreferenceResp  *v4grpc.ClearSnaxpotPreferenceResponse
	numberPreferenceErr  error
	numberPreferenceReq  *v4grpc.GetSnaxpotNumberPreferenceRequest
	numberPreferenceResp *v4grpc.GetSnaxpotNumberPreferenceResponse
	saveTicketsErr       error
	saveTicketsReq       *v4grpc.SaveSnaxpotTicketsRequest
	saveTicketsResp      *v4grpc.SaveSnaxpotTicketsResponse
	setPreferenceErr     error
	setPreferenceReq     *v4grpc.SetSnaxpotPreferenceRequest
	setPreferenceResp    *v4grpc.SetSnaxpotPreferenceResponse
	ticketStatusErr      error
	ticketStatusReq      *v4grpc.GetSnaxpotTicketStatusRequest
	ticketStatusResp     *v4grpc.GetSnaxpotTicketStatusResponse
}

func newSnaxpotTradeContext(
	subAccountId snx_lib_core.SubAccountId,
	subaccountClient v4grpc.SubaccountServiceClient,
) TradeContext {
	return snx_lib_api_handlers_types.NewTradeContext(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		nil,
		nil,
		nil,
		nil,
		subaccountClient,
		nil,
		nil,
		nil,
		snx_lib_request.NewRequestID(),
		"test-req",
		"0x1234567890123456789012345678901234567890",
		subAccountId,
	)
}

func (m *mockSnaxpotSubaccountClient) GetSnaxpotNumberPreference(
	ctx context.Context,
	req *v4grpc.GetSnaxpotNumberPreferenceRequest,
	opts ...grpc.CallOption,
) (*v4grpc.GetSnaxpotNumberPreferenceResponse, error) {
	m.numberPreferenceReq = req
	if m.numberPreferenceErr != nil {
		return nil, m.numberPreferenceErr
	}

	return m.numberPreferenceResp, nil
}

func (m *mockSnaxpotSubaccountClient) SaveSnaxpotTickets(
	ctx context.Context,
	req *v4grpc.SaveSnaxpotTicketsRequest,
	opts ...grpc.CallOption,
) (*v4grpc.SaveSnaxpotTicketsResponse, error) {
	m.saveTicketsReq = req
	if m.saveTicketsErr != nil {
		return nil, m.saveTicketsErr
	}

	return m.saveTicketsResp, nil
}

func (m *mockSnaxpotSubaccountClient) SetSnaxpotPreference(
	ctx context.Context,
	req *v4grpc.SetSnaxpotPreferenceRequest,
	opts ...grpc.CallOption,
) (*v4grpc.SetSnaxpotPreferenceResponse, error) {
	m.setPreferenceReq = req
	if m.setPreferenceErr != nil {
		return nil, m.setPreferenceErr
	}

	return m.setPreferenceResp, nil
}

func (m *mockSnaxpotSubaccountClient) ClearSnaxpotPreference(
	ctx context.Context,
	req *v4grpc.ClearSnaxpotPreferenceRequest,
	opts ...grpc.CallOption,
) (*v4grpc.ClearSnaxpotPreferenceResponse, error) {
	m.clearPreferenceReq = req
	if m.clearPreferenceErr != nil {
		return nil, m.clearPreferenceErr
	}

	return m.clearPreferenceResp, nil
}

func (m *mockSnaxpotSubaccountClient) GetSnaxpotTicketStatus(
	ctx context.Context,
	req *v4grpc.GetSnaxpotTicketStatusRequest,
	opts ...grpc.CallOption,
) (*v4grpc.GetSnaxpotTicketStatusResponse, error) {
	m.ticketStatusReq = req
	if m.ticketStatusErr != nil {
		return nil, m.ticketStatusErr
	}

	return m.ticketStatusResp, nil
}

func Test_Handle_getSnaxpotNumberPreference(t *testing.T) {
	t.Run("returns mapped number preference", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			numberPreferenceResp: &v4grpc.GetSnaxpotNumberPreferenceResponse{
				CurrentEpochId:       42,
				CurrentEpochSnaxBall: 5,
				EffectiveSnaxBall:    4,
				NumberMode:           "snax_only",
				PersistentSnaxBall:   3,
			},
		}

		statusCode, resp := Handle_getSnaxpotNumberPreference(
			newSnaxpotTradeContext(123, mock),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusOK, int(statusCode))
		assert.Equal(t, "ok", resp.Status)

		response, ok := resp.Response.(SnaxpotNumberPreferenceResponse)
		assert.True(t, ok)
		assert.Equal(t, uint64(42), response.CurrentEpochId)
		assert.Equal(t, int32(5), response.CurrentEpochSnaxBall)
		assert.Equal(t, int32(4), response.EffectiveSnaxBall)
		assert.Equal(t, "snax_only", response.NumberMode)
		assert.Equal(t, int32(3), response.PersistentSnaxBall)
		assert.Equal(t, int64(123), mock.numberPreferenceReq.SubAccountId)
	})

	t.Run("missing subaccount returns 400", func(t *testing.T) {
		statusCode, resp := Handle_getSnaxpotNumberPreference(
			newSnaxpotTradeContext(0, snx_lib_authtest.NewMockSubaccountServiceClient()),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusBadRequest, int(statusCode))
		assert.Equal(t, "error", resp.Status)
		assert.Equal(t, "subAccountId is required", resp.Error.Message)
	})

	t.Run("grpc invalid argument returns 400", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			numberPreferenceErr:         status.Error(codes.InvalidArgument, "invalid subaccount"),
		}

		statusCode, resp := Handle_getSnaxpotNumberPreference(
			newSnaxpotTradeContext(123, mock),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusBadRequest, int(statusCode))
		assert.Equal(t, "error", resp.Status)
		assert.Equal(t, "invalid subaccount", resp.Error.Message)
	})
}

func Test_Handle_getSnaxpotTicketStatus(t *testing.T) {
	t.Run("returns mapped ticket status with purchased usd", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			ticketStatusResp: &v4grpc.GetSnaxpotTicketStatusResponse{
				CurrentEpochId:         7,
				CurrentEpochSnaxBall:   5,
				CumulativeFeesUsd:      "15.25",
				CumulativePurchasedUsd: "12.50",
				EarnedTicketCount:      3,
				EffectiveSnaxBall:      4,
				EpochId:                7,
				IssuedTicketCount:      2,
				NumberMode:             "auto",
				PersistentSnaxBall:     2,
				StakingMultiplier:      "1.50",
				Tickets: []*v4grpc.SnaxpotIssuedTicket{
					{
						TicketSerial: 99,
						Ball_1:       1,
						Ball_2:       2,
						Ball_3:       3,
						Ball_4:       4,
						Ball_5:       5,
						SnaxBall:     1,
					},
				},
			},
		}

		statusCode, resp := Handle_getSnaxpotTicketStatus(
			newSnaxpotTradeContext(456, mock),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusOK, int(statusCode))
		assert.Equal(t, "ok", resp.Status)

		response, ok := resp.Response.(SnaxpotTicketStatusResponse)
		assert.True(t, ok)
		assert.Equal(t, "12.50", response.CumulativePurchasedUsd)
		assert.Equal(t, "15.25", response.CumulativeFeesUsd)
		assert.Equal(t, int32(3), response.EarnedTicketCount)
		assert.Len(t, response.Tickets, 1)
		assert.Equal(t, "99", response.Tickets[0].TicketSerial)
		assert.Equal(t, int64(456), mock.ticketStatusReq.SubAccountId)
	})

	t.Run("grpc not found returns 404", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			ticketStatusErr:             status.Error(codes.NotFound, "subaccount not found"),
		}

		statusCode, resp := Handle_getSnaxpotTicketStatus(
			newSnaxpotTradeContext(456, mock),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusNotFound, int(statusCode))
		assert.Equal(t, "error", resp.Status)
		assert.Equal(t, "subaccount not found", resp.Error.Message)
	})
}

func Test_Handle_setSnaxpotPreference(t *testing.T) {
	t.Run("maps scope to grpc enum and back", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			setPreferenceResp: &v4grpc.SetSnaxpotPreferenceResponse{
				AppliedScope:         v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_PERSISTENT,
				CurrentEpochSnaxBall: 4,
				EffectiveSnaxBall:    3,
				EpochId:              77,
				NumberMode:           "snax_only",
				PersistentSnaxBall:   3,
				UpdatedTicketCount:   2,
			},
		}

		statusCode, resp := Handle_setSnaxpotPreference(
			newSnaxpotTradeContext(456, mock).WithAction(
				"setSnaxpotPreference",
				&ValidatedSetSnaxpotPreferenceAction{
					Payload: &snx_lib_api_validation.SetSnaxpotPreferenceActionPayload{
						Action:   "setSnaxpotPreference",
						Scope:    "persistent",
						SnaxBall: 3,
					},
				},
			),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusOK, int(statusCode))
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, int64(456), mock.setPreferenceReq.SubAccountId)
		assert.Equal(t, int32(3), mock.setPreferenceReq.SnaxBall)
		assert.Equal(
			t,
			v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_PERSISTENT,
			mock.setPreferenceReq.Scope,
		)

		response, ok := resp.Response.(SnaxpotPreferenceMutationResponse)
		assert.True(t, ok)
		assert.Equal(t, "persistent", response.AppliedScope)
		assert.Equal(t, uint64(77), response.EpochId)
		assert.Equal(t, "snax_only", response.NumberMode)
		assert.Equal(t, int32(2), response.UpdatedTicketCount)
	})

	t.Run("invalid payload returns 400", func(t *testing.T) {
		statusCode, resp := Handle_setSnaxpotPreference(
			newSnaxpotTradeContext(
				123,
				snx_lib_authtest.NewMockSubaccountServiceClient(),
			).WithAction("setSnaxpotPreference", "not-valid"),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusBadRequest, int(statusCode))
		assert.Equal(t, "error", resp.Status)
		assert.Equal(t, "Invalid request body", resp.Error.Message)
	})
}

func Test_Handle_clearSnaxpotPreference(t *testing.T) {
	t.Run("maps scope and returns auto when no preference remains", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			clearPreferenceResp: &v4grpc.ClearSnaxpotPreferenceResponse{
				AppliedScope:       v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_PERSISTENT,
				EffectiveSnaxBall:  0,
				EpochId:            88,
				NumberMode:         "auto",
				UpdatedTicketCount: 4,
			},
		}

		statusCode, resp := Handle_clearSnaxpotPreference(
			newSnaxpotTradeContext(789, mock).WithAction(
				"clearSnaxpotPreference",
				&ValidatedClearSnaxpotPreferenceAction{
					Payload: &snx_lib_api_validation.ClearSnaxpotPreferenceActionPayload{
						Action: "clearSnaxpotPreference",
						Scope:  "persistent",
					},
				},
			),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusOK, int(statusCode))
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, int64(789), mock.clearPreferenceReq.SubAccountId)
		assert.Equal(
			t,
			v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_PERSISTENT,
			mock.clearPreferenceReq.Scope,
		)

		response, ok := resp.Response.(SnaxpotPreferenceMutationResponse)
		assert.True(t, ok)
		assert.Equal(t, "persistent", response.AppliedScope)
		assert.Equal(t, uint64(88), response.EpochId)
		assert.Equal(t, "auto", response.NumberMode)
		assert.Equal(t, int32(4), response.UpdatedTicketCount)
	})

	t.Run("invalid payload returns 400", func(t *testing.T) {
		statusCode, resp := Handle_clearSnaxpotPreference(
			newSnaxpotTradeContext(
				123,
				snx_lib_authtest.NewMockSubaccountServiceClient(),
			).WithAction("clearSnaxpotPreference", "not-valid"),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusBadRequest, int(statusCode))
		assert.Equal(t, "error", resp.Status)
		assert.Equal(t, "Invalid request body", resp.Error.Message)
	})
}

func Test_Handle_saveSnaxpotTickets(t *testing.T) {
	t.Run("maps entries and response", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
			saveTicketsResp: &v4grpc.SaveSnaxpotTicketsResponse{
				EpochId: 11,
				Tickets: []*v4grpc.SnaxpotIssuedTicket{
					{
						TicketSerial: 42,
						Ball_1:       1,
						Ball_2:       2,
						Ball_3:       3,
						Ball_4:       4,
						Ball_5:       5,
						SnaxBall:     2,
					},
				},
				UpdatedTicketCount: 1,
			},
		}

		statusCode, resp := Handle_saveSnaxpotTickets(
			newSnaxpotTradeContext(789, mock).WithAction(
				"saveSnaxpotTickets",
				&ValidatedSaveSnaxpotTicketsAction{
					Payload: &snx_lib_api_validation.SaveSnaxpotTicketsActionPayload{
						Action: "saveSnaxpotTickets",
						Entries: []snx_lib_api_validation.SnaxpotTicketMutationEntryPayload{
							{
								Ball1:        1,
								Ball2:        2,
								Ball3:        3,
								Ball4:        4,
								Ball5:        5,
								SnaxBall:     2,
								TicketSerial: "42",
							},
						},
					},
				},
			),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusOK, int(statusCode))
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, int64(789), mock.saveTicketsReq.SubAccountId)
		assert.Len(t, mock.saveTicketsReq.Entries, 1)
		assert.Equal(t, uint64(42), mock.saveTicketsReq.Entries[0].TicketSerial)
		assert.Equal(t, int32(2), mock.saveTicketsReq.Entries[0].SnaxBall)

		response, ok := resp.Response.(SaveSnaxpotTicketsResponse)
		assert.True(t, ok)
		assert.Equal(t, uint64(11), response.EpochId)
		assert.Equal(t, int32(1), response.UpdatedTicketCount)
		assert.Len(t, response.Tickets, 1)
		assert.Equal(t, "42", response.Tickets[0].TicketSerial)
	})

	t.Run("invalid ticket serial returns 400", func(t *testing.T) {
		mock := &mockSnaxpotSubaccountClient{
			MockSubaccountServiceClient: snx_lib_authtest.NewMockSubaccountServiceClient(),
		}

		statusCode, resp := Handle_saveSnaxpotTickets(
			newSnaxpotTradeContext(789, mock).WithAction(
				"saveSnaxpotTickets",
				&ValidatedSaveSnaxpotTicketsAction{
					Payload: &snx_lib_api_validation.SaveSnaxpotTicketsActionPayload{
						Action: "saveSnaxpotTickets",
						Entries: []snx_lib_api_validation.SnaxpotTicketMutationEntryPayload{
							{
								Ball1:        1,
								Ball2:        2,
								Ball3:        3,
								Ball4:        4,
								Ball5:        5,
								SnaxBall:     2,
								TicketSerial: "not-a-number",
							},
						},
					},
				},
			),
			HandlerParams{},
		)

		assert.Equal(t, http.StatusBadRequest, int(statusCode))
		assert.Equal(t, "error", resp.Status)
		assert.Equal(t, "ticketSerial must be a valid number", resp.Error.Message)
		assert.Nil(t, mock.saveTicketsReq)
	})
}
