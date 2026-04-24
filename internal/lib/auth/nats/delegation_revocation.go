package authnats

import (
	"encoding/json"

	"github.com/nats-io/nats.go"

	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Subscribes to the delegation-revoked NATS subject and dispatches
// deserialised events to the handler. Core NATS (not JetStream) for
// fan-out to all service instances.
func SubscribeToDelegationRevocations(
	logger snx_lib_logging.Logger,
	conn *nats.Conn,
	handler snx_lib_auth.DelegationRevokedHandler,
) (*nats.Subscription, error) {
	subject := snx_lib_db_nats.SubaccountEventDelegationRevoked.String()

	return conn.Subscribe(subject, func(msg *nats.Msg) {
		var event snx_lib_auth.DelegationRevokedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("Failed to unmarshal delegation revoked event", "error", err)
			return
		}
		handler(event)
	})
}
