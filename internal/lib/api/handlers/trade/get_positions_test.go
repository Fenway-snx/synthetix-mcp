package trade

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func normaliseGetPositionResponseItem(v GetPositionResponseItem) GetPositionResponseItem {
	if len(v.TakeProfitOrderIds) == 0 {
		v.TakeProfitOrderIds = nil
	}
	if len(v.DEPRECATED_TakeProfitOrderIDs) == 0 {
		v.DEPRECATED_TakeProfitOrderIDs = nil
	}
	if len(v.StopLossOrderIds) == 0 {
		v.StopLossOrderIds = nil
	}
	if len(v.DEPRECATED_StopLossOrderIDs) == 0 {
		v.DEPRECATED_StopLossOrderIDs = nil
	}

	return v
}

func Test_GetPositionResponseItem_JSON_MARSHALLING(t *testing.T) {

	t.Run("marshal - empty", func(t *testing.T) {

		v := GetPositionResponseItem{}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"adlBucket":0,"positionId":"","subAccountId":"","symbol":"","side":"","updatedAt":0,"createdAt":0}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)

		// while we can marshal an empty symbol into an empty string, we do
		// not provide symmetrical behaviour - so the unmarshal will fail

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Equal(t, "symbol name empty", err.Error())
		} else {

			assert.Fail(t, "should fail due to being unable to unmarshal empty symbol")
		}
	})

	t.Run("marshal - non-empty with all arrays empty", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:                    "pos_12346",
			SubAccountId:                  "1867542890123456789",
			Symbol:                        "BTC-USDT",
			Side:                          "short",
			EntryPrice:                    Price("42000.00"),
			Quantity:                      Quantity("0.1000"),
			RealizedPnl:                   "25.00",
			UnrealizedPnl:                 "-12.50",
			UsedMargin:                    "840.00",
			MaintenanceMargin:             "420.00",
			LiquidationPrice:              Price("45000.00"),
			Status:                        "open",
			NetFunding:                    "8.50",
			TakeProfitOrderIds:            nil,
			DEPRECATED_TakeProfitOrderIDs: nil,
			StopLossOrderIds:              nil,
			DEPRECATED_StopLossOrderIDs:   nil,
			UpdatedAt:                     1735689600000,
			CreatedAt:                     1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","updatedAt":1735689600000,"createdAt":1735680000000}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("marshal - non-empty with empty slices", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:                    "pos_12346",
			SubAccountId:                  "1867542890123456789",
			Symbol:                        "BTC-USDT",
			Side:                          "short",
			EntryPrice:                    Price("42000.00"),
			Quantity:                      Quantity("0.1000"),
			RealizedPnl:                   "25.00",
			UnrealizedPnl:                 "-12.50",
			UsedMargin:                    "840.00",
			MaintenanceMargin:             "420.00",
			LiquidationPrice:              Price("45000.00"),
			Status:                        "open",
			NetFunding:                    "8.50",
			TakeProfitOrderIds:            []OrderId{},
			DEPRECATED_TakeProfitOrderIDs: []VenueOrderId{},
			StopLossOrderIds:              []OrderId{},
			DEPRECATED_StopLossOrderIDs:   []VenueOrderId{},
			UpdatedAt:                     1735689600000,
			CreatedAt:                     1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		expected := `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","updatedAt":1735689600000,"createdAt":1735680000000}`
		actual := string(bytes)

		assert.Equal(t, expected, actual)

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("marshal - take-profit with one element, stop-loss empty", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:        "pos_12346",
			SubAccountId:      "1867542890123456789",
			Symbol:            "BTC-USDT",
			Side:              "short",
			EntryPrice:        Price("42000.00"),
			Quantity:          Quantity("0.1000"),
			RealizedPnl:       "25.00",
			UnrealizedPnl:     "-12.50",
			UsedMargin:        "840.00",
			MaintenanceMargin: "420.00",
			LiquidationPrice:  Price("45000.00"),
			Status:            "open",
			NetFunding:        "8.50",
			TakeProfitOrderIds: []OrderId{
				{
					VenueId:  "1001",
					ClientId: "cli-tp-1001",
				},
			},
			DEPRECATED_TakeProfitOrderIDs: []VenueOrderId{"1001"},
			StopLossOrderIds:              nil,
			DEPRECATED_StopLossOrderIDs:   nil,
			UpdatedAt:                     1735689600000,
			CreatedAt:                     1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		assert.Equal(t, `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","takeProfitOrders":[{"venueId":"1001","clientId":"cli-tp-1001"}],"takeProfitOrderIds":["1001"],"updatedAt":1735689600000,"createdAt":1735680000000}`, string(bytes))

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("marshal - take-profit empty, stop-loss with one element", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:                    "pos_12346",
			SubAccountId:                  "1867542890123456789",
			Symbol:                        "BTC-USDT",
			Side:                          "short",
			EntryPrice:                    Price("42000.00"),
			Quantity:                      Quantity("0.1000"),
			RealizedPnl:                   "25.00",
			UnrealizedPnl:                 "-12.50",
			UsedMargin:                    "840.00",
			MaintenanceMargin:             "420.00",
			LiquidationPrice:              Price("45000.00"),
			Status:                        "open",
			NetFunding:                    "8.50",
			TakeProfitOrderIds:            nil,
			DEPRECATED_TakeProfitOrderIDs: nil,
			StopLossOrderIds: []OrderId{
				{
					VenueId:  "2002",
					ClientId: "cli-sl-1002",
				},
			},
			DEPRECATED_StopLossOrderIDs: []VenueOrderId{"2002"},
			UpdatedAt:                   1735689600000,
			CreatedAt:                   1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		assert.Equal(t, `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","stopLossOrders":[{"venueId":"2002","clientId":"cli-sl-1002"}],"stopLossOrderIds":["2002"],"updatedAt":1735689600000,"createdAt":1735680000000}`, string(bytes))

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("marshal - both with one element", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:        "pos_12346",
			SubAccountId:      "1867542890123456789",
			Symbol:            "BTC-USDT",
			Side:              "short",
			EntryPrice:        Price("42000.00"),
			Quantity:          Quantity("0.1000"),
			RealizedPnl:       "25.00",
			UnrealizedPnl:     "-12.50",
			UsedMargin:        "840.00",
			MaintenanceMargin: "420.00",
			LiquidationPrice:  Price("45000.00"),
			Status:            "open",
			NetFunding:        "8.50",
			TakeProfitOrderIds: []OrderId{
				{
					VenueId:  "1001",
					ClientId: "cli-tp-1001",
				},
			},
			DEPRECATED_TakeProfitOrderIDs: []VenueOrderId{"1001"},
			StopLossOrderIds: []OrderId{
				{
					VenueId:  "2002",
					ClientId: "cli-sl-1002",
				},
			},
			DEPRECATED_StopLossOrderIDs: []VenueOrderId{"2002"},
			UpdatedAt:                   1735689600000,
			CreatedAt:                   1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		assert.Equal(t, `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","takeProfitOrders":[{"venueId":"1001","clientId":"cli-tp-1001"}],"takeProfitOrderIds":["1001"],"stopLossOrders":[{"venueId":"2002","clientId":"cli-sl-1002"}],"stopLossOrderIds":["2002"],"updatedAt":1735689600000,"createdAt":1735680000000}`, string(bytes))

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("marshal - take-profit with multiple elements, stop-loss empty", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:        "pos_12346",
			SubAccountId:      "1867542890123456789",
			Symbol:            "BTC-USDT",
			Side:              "short",
			EntryPrice:        Price("42000.00"),
			Quantity:          Quantity("0.1000"),
			RealizedPnl:       "25.00",
			UnrealizedPnl:     "-12.50",
			UsedMargin:        "840.00",
			MaintenanceMargin: "420.00",
			LiquidationPrice:  Price("45000.00"),
			Status:            "open",
			NetFunding:        "8.50",
			TakeProfitOrderIds: []OrderId{
				{
					VenueId:  "1001",
					ClientId: "cli-tp-1001",
				},
				{
					VenueId:  "1002",
					ClientId: "cli-tp-1002",
				},
				{
					VenueId:  "1003",
					ClientId: "",
				},
			},
			DEPRECATED_TakeProfitOrderIDs: []VenueOrderId{"1001", "1002", "1003"},
			StopLossOrderIds:              nil,
			DEPRECATED_StopLossOrderIDs:   nil,
			UpdatedAt:                     1735689600000,
			CreatedAt:                     1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		assert.Equal(t, `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","takeProfitOrders":[{"venueId":"1001","clientId":"cli-tp-1001"},{"venueId":"1002","clientId":"cli-tp-1002"},{"venueId":"1003"}],"takeProfitOrderIds":["1001","1002","1003"],"updatedAt":1735689600000,"createdAt":1735680000000}`, string(bytes))

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("marshal - take-profit empty, stop-loss with multiple elements", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:                    "pos_12346",
			SubAccountId:                  "1867542890123456789",
			Symbol:                        "BTC-USDT",
			Side:                          "short",
			EntryPrice:                    Price("42000.00"),
			Quantity:                      Quantity("0.1000"),
			RealizedPnl:                   "25.00",
			UnrealizedPnl:                 "-12.50",
			UsedMargin:                    "840.00",
			MaintenanceMargin:             "420.00",
			LiquidationPrice:              Price("45000.00"),
			Status:                        "open",
			NetFunding:                    "8.50",
			TakeProfitOrderIds:            nil,
			DEPRECATED_TakeProfitOrderIDs: nil,
			StopLossOrderIds: []OrderId{
				{
					VenueId:  "2001",
					ClientId: "cli-sl-1001",
				},
				{
					VenueId:  "2002",
					ClientId: "",
				},
				{
					VenueId:  "2003",
					ClientId: "cli-sl-1003",
				},
			},
			DEPRECATED_StopLossOrderIDs: []VenueOrderId{"2001", "2002", "2003"},
			UpdatedAt:                   1735689600000,
			CreatedAt:                   1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		assert.Equal(t, `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","stopLossOrders":[{"venueId":"2001","clientId":"cli-sl-1001"},{"venueId":"2002"},{"venueId":"2003","clientId":"cli-sl-1003"}],"stopLossOrderIds":["2001","2002","2003"],"updatedAt":1735689600000,"createdAt":1735680000000}`, string(bytes))

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})

	t.Run("marshal - both with multiple elements", func(t *testing.T) {

		v := GetPositionResponseItem{
			PositionId:        "pos_12346",
			SubAccountId:      "1867542890123456789",
			Symbol:            "BTC-USDT",
			Side:              "short",
			EntryPrice:        Price("42000.00"),
			Quantity:          Quantity("0.1000"),
			RealizedPnl:       "25.00",
			UnrealizedPnl:     "-12.50",
			UsedMargin:        "840.00",
			MaintenanceMargin: "420.00",
			LiquidationPrice:  Price("45000.00"),
			Status:            "open",
			NetFunding:        "8.50",
			TakeProfitOrderIds: []OrderId{
				{
					VenueId:  "1001",
					ClientId: "cli-tp-1001",
				},
				{
					VenueId:  "1002",
					ClientId: "cli-tp-1002",
				},
			},
			DEPRECATED_TakeProfitOrderIDs: []VenueOrderId{"1001", "1002"},
			StopLossOrderIds: []OrderId{
				{
					VenueId:  "2001",
					ClientId: "cli-sl-1001",
				},
				{
					VenueId:  "2002",
					ClientId: "",
				},
				{
					VenueId:  "2003",
					ClientId: "cli-sl-1003",
				},
			},
			DEPRECATED_StopLossOrderIDs: []VenueOrderId{"2001", "2002", "2003"},
			UpdatedAt:                   1735689600000,
			CreatedAt:                   1735680000000,
		}

		bytes, err := json.Marshal(v)

		require.Nil(t, err)

		assert.Equal(t, `{"adlBucket":0,"positionId":"pos_12346","subAccountId":"1867542890123456789","symbol":"BTC-USDT","side":"short","entryPrice":"42000.00","quantity":"0.1000","realizedPnl":"25.00","unrealizedPnl":"-12.50","usedMargin":"840.00","maintenanceMargin":"420.00","liquidationPrice":"45000.00","status":"open","netFunding":"8.50","takeProfitOrders":[{"venueId":"1001","clientId":"cli-tp-1001"},{"venueId":"1002","clientId":"cli-tp-1002"}],"takeProfitOrderIds":["1001","1002"],"stopLossOrders":[{"venueId":"2001","clientId":"cli-sl-1001"},{"venueId":"2002"},{"venueId":"2003","clientId":"cli-sl-1003"}],"stopLossOrderIds":["2001","2002","2003"],"updatedAt":1735689600000,"createdAt":1735680000000}`, string(bytes))

		var v2 GetPositionResponseItem
		if err := json.Unmarshal(bytes, &v2); err != nil {

			require.Nil(t, err)
		} else {

			expected := normaliseGetPositionResponseItem(v)
			actual := normaliseGetPositionResponseItem(v2)

			assert.Equal(t, expected, actual)
		}
	})
}
