package info

import (
	"time"

	snx_lib_exchange_status "github.com/Fenway-snx/synthetix-mcp/internal/lib/exchange/status"
	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
)

const exchangeStatusRedisReadTimeout = 200 * time.Millisecond

func Handle_getExchangeStatus(
	ctx InfoContext,
	params HandlerParams,
) (HTTPStatusCode, *snx_lib_api_json.APIResponse[any]) {
	isHalting := false
	if ctx.HaltChecker != nil {
		isHalting = ctx.HaltChecker()
	}

	dto := snx_lib_exchange_status.Build(ctx.Context, snx_lib_exchange_status.Inputs{
		LocalServiceIsHalting: isHalting,
		ReadTimeout:           exchangeStatusRedisReadTimeout,
		RedisClient:           ctx.Rc,
		ServiceId:             ctx.ServiceId,
	})

	return HTTPStatusCode_200_OK, snx_lib_api_json.NewSuccessResponse[any](ctx.ClientRequestId, dto)
}

