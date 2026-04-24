package types

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type ContextCommon struct {
	// diagnostic state
	Logger snx_lib_logging.Logger

	// execution state
	context.Context

	// communication/storage state
	Nc *nats.Conn
	Js jetstream.JetStream
	Rc *snx_lib_db_redis.SnxClient

	// service connectivity state
	TradingClient    v4grpc.TradingServiceClient
	SubaccountClient v4grpc.SubaccountServiceClient
}
