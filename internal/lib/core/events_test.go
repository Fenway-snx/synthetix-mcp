package core

import (
	"encoding/json"
	"testing"
	"time"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_OrderHistory_JSON_RejectionReason(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("includes rejection_reason when populated", func(t *testing.T) {
		rejectedAt := now
		history := OrderHistory{
			OrderId:         OrderId{VenueId: 12345, ClientId: "cli-12345"},
			Symbol:          "BTC-USDT",
			Side:            OrderSideLong,
			Type:            OrderTypeLimit,
			Status:          OrderStateRejected,
			Price:           shopspring_decimal.NewFromInt(50000),
			Quantity:        shopspring_decimal.NewFromInt(1),
			SubAccountId:    98_765,
			RejectedAt:      &rejectedAt,
			RejectionReason: "IOC order rejected: no immediate execution available",
			UpdatedAt:       now,
		}

		data, err := json.Marshal(history)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, "IOC order rejected: no immediate execution available", parsed["rejection_reason"])
	})

	t.Run("omits rejection_reason when empty", func(t *testing.T) {
		history := OrderHistory{
			OrderId:      OrderId{VenueId: 12345, ClientId: "cli-12345"},
			Symbol:       "BTC-USDT",
			Side:         OrderSideLong,
			Type:         OrderTypeLimit,
			Status:       OrderStateRejected,
			Price:        shopspring_decimal.NewFromInt(50000),
			Quantity:     shopspring_decimal.NewFromInt(1),
			SubAccountId: 98_765,
			UpdatedAt:    now,
		}

		data, err := json.Marshal(history)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		_, exists := parsed["rejection_reason"]
		assert.False(t, exists, "rejection_reason should be omitted when empty")
	})
}
