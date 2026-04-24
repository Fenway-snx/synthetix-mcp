package types

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	snx_lib_api_whitelist "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/whitelist"
	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

// Combined execution/request context in handling "/trade" requests.
type TradeContext struct {
	ContextCommon

	// execution state:

	Authenticator        snx_lib_auth.AccountAuthenticatorInterface
	WhitelistArbitrator  *snx_lib_api_whitelist.WhitelistArbitrator
	WhitelistDiagnostics *snx_lib_api_whitelist.WhitelistDiagnostics

	// request state:

	RequestId         RequestId
	ClientRequestId   ClientRequestId
	WalletAddress     WalletAddress
	SelectedAccountId snx_lib_core.SubAccountId
	actionPayload     any

	// Populated for getRateLimits when the gateway ran order RL; nil otherwise.
	getRateLimitsSubaccountSnapshot *GetRateLimitsSubaccountSnapshot
}

func NewTradeContext(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	nc *nats.Conn,
	js jetstream.JetStream,
	rc *snx_lib_db_redis.SnxClient,
	tradingClient v4grpc.TradingServiceClient,
	subaccountClient v4grpc.SubaccountServiceClient,
	authenticator snx_lib_auth.AccountAuthenticatorInterface,
	whitelistArbitrator *snx_lib_api_whitelist.WhitelistArbitrator,
	whitelistDiagnostics *snx_lib_api_whitelist.WhitelistDiagnostics,
	requestId RequestId,
	clientRequestId ClientRequestId,
	walletAddress WalletAddress,
	selectedAccountId snx_lib_core.SubAccountId,
) TradeContext {
	return TradeContext{
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
		Authenticator:        authenticator,
		WhitelistArbitrator:  whitelistArbitrator,
		WhitelistDiagnostics: whitelistDiagnostics,
		RequestId:            requestId,
		ClientRequestId:      clientRequestId,
		WalletAddress:        walletAddress,
		SelectedAccountId:    selectedAccountId,
	}
}

// WithAction attaches the parsed action payload metadata to the context for downstream handlers.
func (tc TradeContext) WithAction(requestAction RequestAction, payload any) TradeContext {
	tc.actionPayload = payload

	return tc
}

// ActionPayload exposes the typed action payload attached to the context, when available.
func (tc TradeContext) ActionPayload() any {
	return tc.actionPayload
}

// Gateway-passed subaccount bucket state after the debit, or nil when the
// gateway did not run order rate limiting for this request.
func (tc TradeContext) GetRateLimitsSubaccountSnapshot() *GetRateLimitsSubaccountSnapshot {
	return tc.getRateLimitsSubaccountSnapshot
}

// Stores a copy of the post-debit snapshot from a successful CheckOrderLimit
// for getRateLimits. Pass nil to leave the field unset. Use only on that
// success path when order rate limiting is enabled.
func (tc TradeContext) WithGetRateLimitsSubaccountSnapshot(s *GetRateLimitsSubaccountSnapshot) TradeContext {
	if s != nil {
		copied := *s
		tc.getRateLimitsSubaccountSnapshot = &copied
	}

	return tc
}

func (tc TradeContext) BeforeInvokeHandler(requestAction RequestAction) {
	tc.Logger.Debug(fmt.Sprintf("Handling %s request", requestAction))
}
