package types

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	snx_lib_api_whitelist "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/whitelist"
	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// Combined execution/request context in handling "/info" requests.
type InfoContext struct {
	ContextCommon

	// execution state:

	MarketDataClient    v4grpc.MarketDataServiceClient
	MarketConfigClient  v4grpc.MarketConfigServiceClient
	HaltChecker         func() bool
	ServiceId           string
	WhitelistArbitrator *snx_lib_api_whitelist.WhitelistArbitrator

	// request state:

	RequestId       RequestId       // Internal server-generated UUID
	ClientRequestId ClientRequestId // Client-provided request ID (echoed in responses)
}

func NewInfoContext(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	nc *nats.Conn,
	js jetstream.JetStream,
	rc *snx_lib_db_redis.SnxClient,
	tradingClient v4grpc.TradingServiceClient,
	subaccountClient v4grpc.SubaccountServiceClient,
	marketDataClient v4grpc.MarketDataServiceClient,
	marketConfigClient v4grpc.MarketConfigServiceClient,
	whitelistArbitrator *snx_lib_api_whitelist.WhitelistArbitrator,
	requestId RequestId,
	clientRequestId ClientRequestId,
) InfoContext {
	return InfoContext{
		ContextCommon: ContextCommon{
			// diagnostic state
			Logger: logger,
			// execution state
			Context: ctx,
			// communication/storage state
			Nc: nc,
			Js: js,
			Rc: rc,
			// service connectivity state
			TradingClient:    tradingClient,
			SubaccountClient: subaccountClient,
		},
		MarketDataClient:    marketDataClient,
		MarketConfigClient:  marketConfigClient,
		WhitelistArbitrator: whitelistArbitrator,
		RequestId:           requestId,
		ClientRequestId:     clientRequestId,
	}
}

func (ic InfoContext) WithServiceState(serviceId string, haltChecker func() bool) InfoContext {
	ic.ServiceId = serviceId
	ic.HaltChecker = haltChecker
	return ic
}

func (ic InfoContext) BeforeInvokeHandler(requestAction RequestAction) {
	ic.Logger.Debug(fmt.Sprintf("Handling %s request", requestAction))
}
