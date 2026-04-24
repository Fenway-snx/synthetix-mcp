package status

import (
	"context"
	"errors"
	"time"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_runtime_halt "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/halt"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

const (
	codeServiceDraining = "SERVICE_DRAINING"
	codeStatusDegraded  = "STATUS_DEGRADED"

	exchangeStatusRunning     = "RUNNING"
	exchangeStatusMaintenance = "MAINTENANCE"
)

type ExchangeStatus struct {
	AcceptingOrders bool   `json:"accepting_orders"`
	ExchangeStatus  string `json:"exchange_status"`
	Code            string `json:"code,omitempty"`
	Message         string `json:"message"`
	TimestampMs     int64  `json:"timestamp_ms"`
}

type targetStateReaderFunc func(
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	serviceId string,
	timeout time.Duration,
) (string, bool)

type Inputs struct {
	LocalServiceIsHalting bool
	ReadTimeout           time.Duration
	RedisClient           *snx_lib_db_redis.SnxClient
	ServiceId             string

	// readTargetState overrides the Redis reader for testing. When nil
	// the production implementation is used. Unexported so that only
	// same-package tests can set it; no mutable global required.
	readTargetState targetStateReaderFunc
}

func Build(
	ctx context.Context,
	in Inputs,
) ExchangeStatus {
	now := snx_lib_utils_time.Now()

	if in.LocalServiceIsHalting {
		return ExchangeStatus{
			AcceptingOrders: false,
			ExchangeStatus:  exchangeStatusMaintenance,
			Code:            codeServiceDraining,
			Message:         "Service is draining for deployment",
			TimestampMs:     now.UnixMilli(),
		}
	}

	reader := in.readTargetState
	if reader == nil {
		reader = readRedisTargetStateWithTimeout
	}
	redisTarget, ok := reader(ctx, in.RedisClient, in.ServiceId, in.ReadTimeout)
	if !ok {
		return ExchangeStatus{
			AcceptingOrders: true,
			ExchangeStatus:  exchangeStatusRunning,
			Code:            codeStatusDegraded,
			Message:         "OK (degraded)",
			TimestampMs:     now.UnixMilli(),
		}
	}
	if redisTarget != "" && redisTarget != string(snx_lib_runtime_halt.TargetState_Running) {
		return ExchangeStatus{
			AcceptingOrders: false,
			ExchangeStatus:  exchangeStatusMaintenance,
			Code:            codeServiceDraining,
			Message:         "Service is draining for deployment",
			TimestampMs:     now.UnixMilli(),
		}
	}

	return ExchangeStatus{
		AcceptingOrders: true,
		ExchangeStatus:  exchangeStatusRunning,
		Message:         "OK",
		TimestampMs:     now.UnixMilli(),
	}
}

func readRedisTargetStateWithTimeout(
	ctx context.Context,
	rc *snx_lib_db_redis.SnxClient,
	serviceId string,
	timeout time.Duration,
) (string, bool) {
	if rc == nil || !rc.IsValid() || serviceId == "" {
		return "", false
	}
	if timeout <= 0 {
		return "", false
	}

	readCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	value, err := rc.Get(readCtx, snx_lib_runtime_halt.TargetStateKeyPrefix+serviceId).Result()
	if err != nil {
		if errors.Is(err, snx_lib_db_redis.Nil) {
			return "", true
		}
		return "", false
	}
	return value, true
}

