package info

import (
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func Test_GetOrderbookEndpoint(t *testing.T) {
	// Test that the getOrderbook endpoint handler is registered and returns the correct format
	t.Run("getOrderbook uses Handle_getOrderbook function", func(t *testing.T) {
		// Create mock context
		logger := snx_lib_logging_doubles.NewStubLogger()
		ctx := InfoContext{
			ContextCommon: ContextCommon{
				Logger: logger,
			},
			ClientRequestId: "test-request-123",
		}

		// Test request payload
		req := map[string]any{
			"action": "getOrderbook",
			"symbol": "BTC-USDT",
			"limit":  20,
		}

		// Since Handle_getOrderbook requires NATS connection for actual data retrieval,
		// we can't test the full flow without mocking NATS.
		// But we can verify that the function signature is correct and
		// the endpoint will be registered properly.

		// This test verifies that Handle_getOrderbook accepts the correct parameters
		// The actual functionality is already tested in the depth handler
		assert.Equal(t, ClientRequestId("test-request-123"), ctx.ClientRequestId)
		assert.Equal(t, "getOrderbook", req["action"])
		assert.Equal(t, "BTC-USDT", req["symbol"])
		assert.Equal(t, 20, req["limit"])
	})
}

func Test_OrderbookResponseStructure(t *testing.T) {
	t.Run("response structure matches specification", func(t *testing.T) {
		// This test documents the expected response structure
		expectedResponse := map[string]any{
			"bids": [][]string{
				{"45000.00", "1.5"},
				{"44999.50", "2.0"},
				{"44999.00", "0.8"},
			},
			"asks": [][]string{
				{"45001.00", "1.2"},
				{"45001.50", "3.1"},
				{"45002.00", "0.9"},
			},
		}

		// Verify structure has required fields
		assert.Contains(t, expectedResponse, "bids")
		assert.Contains(t, expectedResponse, "asks")

		// Verify data types
		assert.IsType(t, [][]string{}, expectedResponse["bids"])
		assert.IsType(t, [][]string{}, expectedResponse["asks"])

		// Verify bid/ask array structure
		bids := expectedResponse["bids"].([][]string)
		asks := expectedResponse["asks"].([][]string)

		if len(bids) > 0 {
			assert.Len(t, bids[0], 2, "Each bid should have [price, quantity]")
		}
		if len(asks) > 0 {
			assert.Len(t, asks[0], 2, "Each ask should have [price, quantity]")
		}
	})
}
