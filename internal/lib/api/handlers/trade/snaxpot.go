package trade

import (
	"strconv"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type SnaxpotIssuedTicketResponse struct {
	Ball1        int32  `json:"ball1"`
	Ball2        int32  `json:"ball2"`
	Ball3        int32  `json:"ball3"`
	Ball4        int32  `json:"ball4"`
	Ball5        int32  `json:"ball5"`
	SnaxBall     int32  `json:"snaxBall"`
	TicketSerial string `json:"ticketSerial"`
}

type SnaxpotNumberPreferenceResponse struct {
	CurrentEpochId       uint64 `json:"currentEpochId"`
	CurrentEpochSnaxBall int32  `json:"currentEpochSnaxBall"`
	EffectiveSnaxBall    int32  `json:"effectiveSnaxBall"`
	NumberMode           string `json:"numberMode"`
	PersistentSnaxBall   int32  `json:"persistentSnaxBall"`
}

type SnaxpotTicketStatusResponse struct {
	CurrentEpochId       uint64 `json:"currentEpochId"`
	CurrentEpochSnaxBall int32  `json:"currentEpochSnaxBall"`
	CumulativeFeesUsd    string `json:"cumulativeFeesUsd"`
	// Purchase credits come from the dedicated Snaxpot sUSD purchase event
	// pipeline, not from any API-level purchase side effect.
	CumulativePurchasedUsd string                        `json:"cumulativePurchasedUsd"`
	EarnedTicketCount      int32                         `json:"earnedTicketCount"`
	EffectiveSnaxBall      int32                         `json:"effectiveSnaxBall"`
	EpochId                uint64                        `json:"epochId"`
	IssuedTicketCount      int32                         `json:"issuedTicketCount"`
	NumberMode             string                        `json:"numberMode"`
	PersistentSnaxBall     int32                         `json:"persistentSnaxBall"`
	StakingMultiplier      string                        `json:"stakingMultiplier"`
	Tickets                []SnaxpotIssuedTicketResponse `json:"tickets"`
}

// Handler for "getSnaxpotNumberPreference".
//
//dd:span
func Handle_getSnaxpotNumberPreference(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"subAccountId is required",
			nil,
		)
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.GetSnaxpotNumberPreference(
		ctx.Context,
		&v4grpc.GetSnaxpotNumberPreferenceRequest{
			TimestampMs:  timestampMs,
			TimestampUs:  timestampUs,
			SubAccountId: int64(ctx.SelectedAccountId),
		},
	)
	if err != nil {
		ctx.Logger.Error("Failed to get Snaxpot number preference",
			"error", err,
			"sub_account_id", ctx.SelectedAccountId,
		)

		return handleGRPCError(err, ctx.ClientRequestId)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](
		ctx.ClientRequestId,
		SnaxpotNumberPreferenceResponse{
			CurrentEpochId:       grpcResp.CurrentEpochId,
			CurrentEpochSnaxBall: grpcResp.CurrentEpochSnaxBall,
			EffectiveSnaxBall:    grpcResp.EffectiveSnaxBall,
			NumberMode:           grpcResp.NumberMode,
			PersistentSnaxBall:   grpcResp.PersistentSnaxBall,
		},
	)
}

// Handler for "getSnaxpotTicketStatus".
//
// This is a read-only gateway view over Snaxpot state. In particular,
// `cumulativePurchasedUsd` reflects credits that were already ingested from
// the dedicated relayer purchase event flow described in the purchase spec;
// this handler does not initiate or infer purchases itself.
//
//dd:span
func Handle_getSnaxpotTicketStatus(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"subAccountId is required",
			nil,
		)
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.GetSnaxpotTicketStatus(
		ctx.Context,
		&v4grpc.GetSnaxpotTicketStatusRequest{
			TimestampMs:  timestampMs,
			TimestampUs:  timestampUs,
			SubAccountId: int64(ctx.SelectedAccountId),
		},
	)
	if err != nil {
		ctx.Logger.Error("Failed to get Snaxpot ticket status",
			"error", err,
			"sub_account_id", ctx.SelectedAccountId,
		)

		return handleGRPCError(err, ctx.ClientRequestId)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](
		ctx.ClientRequestId,
		SnaxpotTicketStatusResponse{
			CurrentEpochId:         grpcResp.CurrentEpochId,
			CurrentEpochSnaxBall:   grpcResp.CurrentEpochSnaxBall,
			CumulativeFeesUsd:      grpcResp.CumulativeFeesUsd,
			CumulativePurchasedUsd: grpcResp.CumulativePurchasedUsd,
			EarnedTicketCount:      grpcResp.EarnedTicketCount,
			EffectiveSnaxBall:      grpcResp.EffectiveSnaxBall,
			EpochId:                grpcResp.EpochId,
			IssuedTicketCount:      grpcResp.IssuedTicketCount,
			NumberMode:             grpcResp.NumberMode,
			PersistentSnaxBall:     grpcResp.PersistentSnaxBall,
			StakingMultiplier:      grpcResp.StakingMultiplier,
			Tickets:                mapSnaxpotIssuedTickets(grpcResp.Tickets),
		},
	)
}

// Shared response shape for the `setSnaxpotPreference` and
// `clearSnaxpotPreference` mutations. The two endpoints are intentional
// siblings: `set` enters snax_only by saving a Snax-ball, `clear` returns
// to auto by removing one. Both report the resulting state so the
// frontend can refresh without a follow-up read.
type SnaxpotPreferenceMutationResponse struct {
	AppliedScope         string `json:"appliedScope"`
	CurrentEpochSnaxBall int32  `json:"currentEpochSnaxBall"`
	EffectiveSnaxBall    int32  `json:"effectiveSnaxBall"`
	EpochId              uint64 `json:"epochId"`
	NumberMode           string `json:"numberMode"`
	PersistentSnaxBall   int32  `json:"persistentSnaxBall"`
	UpdatedTicketCount   int32  `json:"updatedTicketCount"`
}

type SaveSnaxpotTicketsResponse struct {
	EpochId            uint64                        `json:"epochId"`
	Tickets            []SnaxpotIssuedTicketResponse `json:"tickets"`
	UpdatedTicketCount int32                         `json:"updatedTicketCount"`
}

func snaxBallScopeFromString(scope string) v4grpc.SnaxpotSnaxBallScope {
	switch scope {
	case "currentEpoch":
		return v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_CURRENT_EPOCH
	case "persistent":
		return v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_PERSISTENT
	default:
		return v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_UNSPECIFIED
	}
}

func snaxBallScopeToString(scope v4grpc.SnaxpotSnaxBallScope) string {
	switch scope {
	case v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_CURRENT_EPOCH:
		return "currentEpoch"
	case v4grpc.SnaxpotSnaxBallScope_SNAXPOT_SNAX_BALL_SCOPE_PERSISTENT:
		return "persistent"
	default:
		return "unspecified"
	}
}

// Handler for "setSnaxpotPreference". Saves the user's chosen Snax-ball for
// the given scope and switches the wallet's mode to snax_only. The frontend
// only ever needs this and `clearSnaxpotPreference`; mode is implied by
// whether a preference exists.
//
//dd:span
func Handle_setSnaxpotPreference(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"subAccountId is required",
			nil,
		)
	}

	validated, ok := ctx.ActionPayload().(*ValidatedSetSnaxpotPreferenceAction)
	if !ok || validated == nil || validated.Payload == nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			snx_lib_api_json.ErrorCodeInvalidFormat,
			"Invalid request body",
			nil,
		)
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.SetSnaxpotPreference(
		ctx.Context,
		&v4grpc.SetSnaxpotPreferenceRequest{
			TimestampMs:  timestampMs,
			TimestampUs:  timestampUs,
			SubAccountId: int64(ctx.SelectedAccountId),
			SnaxBall:     validated.Payload.SnaxBall,
			Scope:        snaxBallScopeFromString(validated.Payload.Scope),
		},
	)
	if err != nil {
		ctx.Logger.Error("Failed to set Snaxpot preference",
			"error", err,
			"sub_account_id", ctx.SelectedAccountId,
		)

		return handleGRPCError(err, ctx.ClientRequestId)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](
		ctx.ClientRequestId,
		SnaxpotPreferenceMutationResponse{
			AppliedScope:         snaxBallScopeToString(grpcResp.AppliedScope),
			CurrentEpochSnaxBall: grpcResp.CurrentEpochSnaxBall,
			EffectiveSnaxBall:    grpcResp.EffectiveSnaxBall,
			EpochId:              grpcResp.EpochId,
			NumberMode:           grpcResp.NumberMode,
			PersistentSnaxBall:   grpcResp.PersistentSnaxBall,
			UpdatedTicketCount:   grpcResp.UpdatedTicketCount,
		},
	)
}

// Handler for "clearSnaxpotPreference". Removes the user's Snax-ball for the
// given scope and falls back to auto when no scoped preference remains.
//
//dd:span
func Handle_clearSnaxpotPreference(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"subAccountId is required",
			nil,
		)
	}

	validated, ok := ctx.ActionPayload().(*ValidatedClearSnaxpotPreferenceAction)
	if !ok || validated == nil || validated.Payload == nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			snx_lib_api_json.ErrorCodeInvalidFormat,
			"Invalid request body",
			nil,
		)
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.ClearSnaxpotPreference(
		ctx.Context,
		&v4grpc.ClearSnaxpotPreferenceRequest{
			TimestampMs:  timestampMs,
			TimestampUs:  timestampUs,
			SubAccountId: int64(ctx.SelectedAccountId),
			Scope:        snaxBallScopeFromString(validated.Payload.Scope),
		},
	)
	if err != nil {
		ctx.Logger.Error("Failed to clear Snaxpot preference",
			"error", err,
			"sub_account_id", ctx.SelectedAccountId,
		)

		return handleGRPCError(err, ctx.ClientRequestId)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](
		ctx.ClientRequestId,
		SnaxpotPreferenceMutationResponse{
			AppliedScope:         snaxBallScopeToString(grpcResp.AppliedScope),
			CurrentEpochSnaxBall: grpcResp.CurrentEpochSnaxBall,
			EffectiveSnaxBall:    grpcResp.EffectiveSnaxBall,
			EpochId:              grpcResp.EpochId,
			NumberMode:           grpcResp.NumberMode,
			PersistentSnaxBall:   grpcResp.PersistentSnaxBall,
			UpdatedTicketCount:   grpcResp.UpdatedTicketCount,
		},
	)
}

// Handler for "saveSnaxpotTickets".
//
//dd:span
func Handle_saveSnaxpotTickets(
	ctx TradeContext,
	_ HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	if ctx.SelectedAccountId == 0 {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			ErrorCodeValidationError,
			"subAccountId is required",
			nil,
		)
	}

	validated, ok := ctx.ActionPayload().(*ValidatedSaveSnaxpotTicketsAction)
	if !ok || validated == nil || validated.Payload == nil {
		return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
			ctx.ClientRequestId,
			snx_lib_api_json.ErrorCodeInvalidFormat,
			"Invalid request body",
			nil,
		)
	}

	entries := make([]*v4grpc.SnaxpotTicketMutationEntry, 0, len(validated.Payload.Entries))
	for _, e := range validated.Payload.Entries {
		serial, err := strconv.ParseUint(e.TicketSerial, 10, 64)
		if err != nil {
			return HTTPStatusCode_400_BadRequest, snx_lib_api_json.NewErrorResponse[any](
				ctx.ClientRequestId,
				ErrorCodeValidationError,
				"ticketSerial must be a valid number",
				nil,
			)
		}

		entries = append(entries, &v4grpc.SnaxpotTicketMutationEntry{
			Ball_1:       e.Ball1,
			Ball_2:       e.Ball2,
			Ball_3:       e.Ball3,
			Ball_4:       e.Ball4,
			Ball_5:       e.Ball5,
			SnaxBall:     e.SnaxBall,
			TicketSerial: serial,
		})
	}

	timestampUs, timestampMs := snx_lib_utils_time.NowMicrosAndMillis()
	grpcResp, err := ctx.SubaccountClient.SaveSnaxpotTickets(
		ctx.Context,
		&v4grpc.SaveSnaxpotTicketsRequest{
			TimestampMs:  timestampMs,
			TimestampUs:  timestampUs,
			SubAccountId: int64(ctx.SelectedAccountId),
			Entries:      entries,
		},
	)
	if err != nil {
		ctx.Logger.Error("Failed to save Snaxpot tickets",
			"error", err,
			"sub_account_id", ctx.SelectedAccountId,
		)

		return handleGRPCError(err, ctx.ClientRequestId)
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](
		ctx.ClientRequestId,
		SaveSnaxpotTicketsResponse{
			EpochId:            grpcResp.EpochId,
			Tickets:            mapSnaxpotIssuedTickets(grpcResp.Tickets),
			UpdatedTicketCount: grpcResp.UpdatedTicketCount,
		},
	)
}

func mapSnaxpotIssuedTickets(
	tickets []*v4grpc.SnaxpotIssuedTicket,
) []SnaxpotIssuedTicketResponse {
	out := make([]SnaxpotIssuedTicketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		out = append(out, SnaxpotIssuedTicketResponse{
			Ball1:        ticket.Ball_1,
			Ball2:        ticket.Ball_2,
			Ball3:        ticket.Ball_3,
			Ball4:        ticket.Ball_4,
			Ball5:        ticket.Ball_5,
			SnaxBall:     ticket.SnaxBall,
			TicketSerial: strconv.FormatUint(ticket.TicketSerial, 10),
		})
	}

	return out
}
