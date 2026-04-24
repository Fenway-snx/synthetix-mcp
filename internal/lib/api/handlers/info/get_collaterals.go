package info

import (
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

/*
API Docs:
	https://v4-offchain-docs.snxdev.io/developer-resources/api/rest-api/info/getCollaterals
*/

// Handler for "getCollaterals".
//
//dd:span
func Handle_getCollaterals(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	timestamp_us, timestamp_ms := snx_lib_utils_time.NowMicrosAndMillis()

	grpcResp, err := ctx.MarketConfigClient.GetCollateralConfigs(ctx, &v4grpc.GetCollateralConfigsRequest{
		TimestampMs: timestamp_ms,
		TimestampUs: timestamp_us,
	})
	if err != nil {
		return HTTPStatusCode_500_InternalServerError, snx_lib_api_json.NewSystemErrorResponse[any](ctx.ClientRequestId, "Could not retrieve collateral configuration", err)
	}

	collaterals := make([]CollateralConfigResponse, len(grpcResp.Collaterals))
	for i, proto := range grpcResp.Collaterals {
		tiers := make([]CollateralConfigTierResponse, len(proto.Tiers))
		for j, tier := range proto.Tiers {
			tiers[j] = CollateralConfigTierResponse{
				ID:            int64(tier.Id),
				Haircut:       tier.Haircut,
				MaxAmount:     tier.MaxAmount,
				MinAmount:     tier.MinAmount,
				Name:          tier.Name,
				ValueAddition: tier.ValueAddition,
				ValueRatio:    tier.ValueRatio,
			}
		}
		collaterals[i] = CollateralConfigResponse{
			Collateral:  proto.Collateral,
			DepositCap:  proto.DepositCap,
			LLTV:        proto.Lltv,
			LTV:         proto.Ltv,
			Market:      proto.Market,
			Tiers:       tiers,
			WithdrawFee: proto.WithdrawFee,
		}
	}

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, collaterals)
}
