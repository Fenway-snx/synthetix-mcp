package json

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

func Benchmark_ValidateCancelOrder(b *testing.B) {
	req := CancelOrderRequest{
		Params: struct {
			Action  string `json:"action"`
			OrderId int64  `json:"orderId"`
		}{
			Action:  "cancelOrder",
			OrderId: 12345,
		},
		Nonce: 1722434008000,
		Signature: EIP712Signature{
			R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			V: 27,
		},
		SubAccountId: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateCancelOrder(&req)
	}
}

func Test_ValidateCancelOrder(t *testing.T) {
	tests := []struct {
		name    string
		req     *CancelOrderRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid cancel order request",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
				SubAccountId: 1,
			},
			wantErr: false,
		},
		{
			name: "valid cancel order with v=28",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 98765,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 28,
				},
				ExpiresAfter: 1722434108000,
			},
			wantErr: false,
		},
		{
			name: "valid cancel order with v=0",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 555,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "valid cancel order with v=1",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 777,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid action type",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "order",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "action type must be 'cancelOrder'",
		},
		{
			name: "missing orderId",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 0,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "orderId is required",
		},
		{
			name: "missing nonce",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 0,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "nonce is required (recommended: current timestamp in milliseconds)",
		},
		{
			name: "missing signature r field",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "signature r and s fields are required",
		},
		{
			name: "missing signature s field",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "signature r and s fields are required",
		},
		{
			name: "invalid signature r format - no 0x prefix",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "signature r field must be a 64-character hex string with 0x prefix",
		},
		{
			name: "invalid signature r format - wrong length",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "signature r field must be a 64-character hex string with 0x prefix",
		},
		{
			name: "invalid signature s format - no 0x prefix",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "signature s field must be a 64-character hex string with 0x prefix",
		},
		{
			name: "invalid signature s format - wrong length",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef00",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "signature s field must be a 64-character hex string with 0x prefix",
		},
		{
			name: "invalid signature v field - 2",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 2,
				},
			},
			wantErr: true,
			errMsg:  "signature v field must be 0, 1, 27, or 28",
		},
		{
			name: "invalid signature v field - 29",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 29,
				},
			},
			wantErr: true,
			errMsg:  "signature v field must be 0, 1, 27, or 28",
		},
		{
			name: "invalid expiresAfter - less than nonce",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce:        1722434008000,
				ExpiresAfter: 1722434007000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "expiresAfter must be greater than nonce",
		},
		{
			name: "invalid expiresAfter - equal to nonce",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce:        1722434008000,
				ExpiresAfter: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
			},
			wantErr: true,
			errMsg:  "expiresAfter must be greater than nonce",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCancelOrder(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			}
		})
	}
}

func Test_CancelOrderRequest_JSON_MARSHALING(t *testing.T) {
	tests := []struct {
		name     string
		req      *CancelOrderRequest
		expected string
	}{
		{
			name: "complete cancel order request",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 12345,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 27,
				},
				VaultAddress: "0x742d35Cc6634C0532925a3b8D4e5C5532925a3b8",
				ExpiresAfter: 1722434108000,
				SubAccountId: 1,
			},
			expected: `{"params":{"action":"cancelOrder","orderId":12345},"nonce":1722434008000,"signature":{"v":27,"r":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef","s":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},"vaultAddress":"0x742d35Cc6634C0532925a3b8D4e5C5532925a3b8","expiresAfter":1722434108000,"subaccountId":1}`,
		},
		{
			name: "minimal cancel order request",
			req: &CancelOrderRequest{
				Params: struct {
					Action  string `json:"action"`
					OrderId int64  `json:"orderId"`
				}{
					Action:  "cancelOrder",
					OrderId: 98765,
				},
				Nonce: 1722434008000,
				Signature: EIP712Signature{
					R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					V: 28,
				},
			},
			expected: `{"params":{"action":"cancelOrder","orderId":98765},"nonce":1722434008000,"signature":{"v":28,"r":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef","s":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonBytes, err := json.Marshal(tt.req)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.JSONEq(t, tt.expected, string(jsonBytes))

			// Test unmarshaling
			var unmarshaled CancelOrderRequest
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
			assert.Equal(t, tt.req.Params.Action, unmarshaled.Params.Action)
			assert.Equal(t, tt.req.Params.OrderId, unmarshaled.Params.OrderId)
			assert.Equal(t, tt.req.Nonce, unmarshaled.Nonce)
			assert.Equal(t, tt.req.Signature, unmarshaled.Signature)
			assert.Equal(t, tt.req.VaultAddress, unmarshaled.VaultAddress)
			assert.Equal(t, tt.req.ExpiresAfter, unmarshaled.ExpiresAfter)
			assert.Equal(t, tt.req.SubAccountId, unmarshaled.SubAccountId)
		})
	}
}

func Test_CancelOrderRequest_ROUND_TRIP_JSON(t *testing.T) {
	originalRequests := []*CancelOrderRequest{
		{
			Params: struct {
				Action  string `json:"action"`
				OrderId int64  `json:"orderId"`
			}{
				Action:  "cancelOrder",
				OrderId: 12345,
			},
			Nonce: 1722434008000,
			Signature: EIP712Signature{
				R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				V: 27,
			},
			VaultAddress: "0x742d35Cc6634C0532925a3b8D4e5C5532925a3b8",
			ExpiresAfter: 1722434108000,
			SubAccountId: 1,
		},
		{
			Params: struct {
				Action  string `json:"action"`
				OrderId int64  `json:"orderId"`
			}{
				Action:  "cancelOrder",
				OrderId: 98765,
			},
			Nonce: 1722434008000,
			Signature: EIP712Signature{
				R: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				S: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				V: 0,
			},
		},
	}

	for i, original := range originalRequests {
		t.Run("round trip test "+string(rune(i+'1')), func(t *testing.T) {
			// Marshal to JSON
			jsonBytes, err := json.Marshal(original)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// Unmarshal back to struct
			var roundTrip CancelOrderRequest
			err = json.Unmarshal(jsonBytes, &roundTrip)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

			// Compare all fields
			assert.Equal(t, original.Params.Action, roundTrip.Params.Action)
			assert.Equal(t, original.Params.OrderId, roundTrip.Params.OrderId)
			assert.Equal(t, original.Nonce, roundTrip.Nonce)
			assert.Equal(t, original.Signature, roundTrip.Signature)
			assert.Equal(t, original.VaultAddress, roundTrip.VaultAddress)
			assert.Equal(t, original.ExpiresAfter, roundTrip.ExpiresAfter)
			assert.Equal(t, original.SubAccountId, roundTrip.SubAccountId)

			// Validate the round-trip result
			err = ValidateCancelOrder(&roundTrip)
			assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		})
	}
}

func Test_OrderStatusResponse_WithError(t *testing.T) {
	t.Run("includes order ID when both venueId and clientId present", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "insufficient margin",
			OrderId: &v4grpc.OrderId{
				VenueId:  456,
				ClientId: "0xabc",
			},
		}

		status := NewOrderStatusResponse().WithError(resp)

		assert.Equal(t, "insufficient margin", status.Error)
		assert.NotNil(t, status.ErrorOrderId)
		assert.NotNil(t, status.ErrorOrderId.VenueId)
		assert.Equal(t, VenueOrderId("456"), *status.ErrorOrderId.VenueId)
		assert.Equal(t, ClientOrderId("0xabc"), status.ErrorOrderId.ClientId)
	})

	t.Run("omits order field when grpc OrderId is nil", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "invalid order format",
		}

		status := NewOrderStatusResponse().WithError(resp)

		assert.Equal(t, "invalid order format", status.Error)
		assert.Nil(t, status.ErrorOrderId)
	})

	t.Run("omits order field when venue ID is zero and no clientId", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "validation error",
			OrderId: &v4grpc.OrderId{
				VenueId: 0,
			},
		}

		status := NewOrderStatusResponse().WithError(resp)

		assert.Equal(t, "validation error", status.Error)
		assert.Nil(t, status.ErrorOrderId)
	})

	t.Run("includes clientId with null venueId for early validation errors", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "order validation failed: price out of bounds",
			OrderId: &v4grpc.OrderId{
				VenueId:  0,
				ClientId: "0xabc",
			},
		}

		status := NewOrderStatusResponse().WithError(resp)

		assert.Equal(t, "order validation failed: price out of bounds", status.Error)
		assert.NotNil(t, status.ErrorOrderId)
		assert.Nil(t, status.ErrorOrderId.VenueId)
		assert.Equal(t, ClientOrderId("0xabc"), status.ErrorOrderId.ClientId)
	})

	t.Run("falls back to status string when message empty", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "",
			Status:    v4grpc.OrderStatus_REJECTED,
			OrderId: &v4grpc.OrderId{
				VenueId: 789,
			},
		}

		status := NewOrderStatusResponse().WithError(resp)

		assert.Equal(t, "Order REJECTED", status.Error)
		assert.NotNil(t, status.ErrorOrderId)
		assert.NotNil(t, status.ErrorOrderId.VenueId)
		assert.Equal(t, VenueOrderId("789"), *status.ErrorOrderId.VenueId)
	})
}

func Test_OrderStatusResponse_WithError_JSON(t *testing.T) {
	t.Run("error with venueId and clientId serializes correctly", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "insufficient margin",
			OrderId: &v4grpc.OrderId{
				VenueId:  456,
				ClientId: "0xabc",
			},
		}

		status := NewOrderStatusResponse().WithError(resp)

		jsonBytes, err := json.Marshal(status)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		expected := `{"error":"insufficient margin","order":{"venueId":"456","clientId":"0xabc"}}`
		assert.JSONEq(t, expected, string(jsonBytes))
	})

	t.Run("error without order ID omits order field", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "invalid order format",
		}

		status := NewOrderStatusResponse().WithError(resp)

		jsonBytes, err := json.Marshal(status)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		expected := `{"error":"invalid order format"}`
		assert.JSONEq(t, expected, string(jsonBytes))
	})

	t.Run("early validation error with clientId only has null venueId", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "order validation failed: price out of bounds",
			OrderId: &v4grpc.OrderId{
				VenueId:  0,
				ClientId: "0xabc",
			},
		}

		status := NewOrderStatusResponse().WithError(resp)

		jsonBytes, err := json.Marshal(status)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		expected := `{"error":"order validation failed: price out of bounds","order":{"venueId":null,"clientId":"0xabc"}}`
		assert.JSONEq(t, expected, string(jsonBytes))
	})

	t.Run("error with venueId only omits clientId", func(t *testing.T) {
		resp := &v4grpc.PlaceOrderResponseItem{
			IsSuccess: false,
			Message:   "some error",
			OrderId: &v4grpc.OrderId{
				VenueId: 789,
			},
		}

		status := NewOrderStatusResponse().WithError(resp)

		jsonBytes, err := json.Marshal(status)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		expected := `{"error":"some error","order":{"venueId":"789"}}`
		assert.JSONEq(t, expected, string(jsonBytes))
	})

	t.Run("batch response with mixed success and error", func(t *testing.T) {
		statuses := []OrderStatusResponse{
			NewOrderStatusResponse().WithResting(&v4grpc.PlaceOrderResponseItem{
				IsSuccess: true,
				OrderId: &v4grpc.OrderId{
					VenueId:  123,
					ClientId: "0xaaa",
				},
			}),
			NewOrderStatusResponse().WithError(&v4grpc.PlaceOrderResponseItem{
				IsSuccess: false,
				Message:   "insufficient margin",
				OrderId: &v4grpc.OrderId{
					VenueId:  456,
					ClientId: "0xbbb",
				},
			}),
			NewOrderStatusResponse().WithResting(&v4grpc.PlaceOrderResponseItem{
				IsSuccess: true,
				OrderId: &v4grpc.OrderId{
					VenueId:  789,
					ClientId: "0xccc",
				},
			}),
		}

		data := OrderDataResponse{Statuses: statuses}

		jsonBytes, err := json.Marshal(data)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		var parsed map[string]interface{}
		err = json.Unmarshal(jsonBytes, &parsed)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		parsedStatuses := parsed["statuses"].([]interface{})
		assert.Len(t, parsedStatuses, 3)

		// First: resting with order
		first := parsedStatuses[0].(map[string]interface{})
		assert.Contains(t, first, "resting")
		assert.NotContains(t, first, "error")

		// Second: error with order
		second := parsedStatuses[1].(map[string]interface{})
		assert.Contains(t, second, "error")
		assert.Contains(t, second, "order")
		assert.Equal(t, "insufficient margin", second["error"])
		orderObj := second["order"].(map[string]interface{})
		assert.Equal(t, "456", orderObj["venueId"])

		// Third: resting with order
		third := parsedStatuses[2].(map[string]interface{})
		assert.Contains(t, third, "resting")
		assert.NotContains(t, third, "error")
	})
}

func Test_ValidateOrderTypeConstraints_TWAPInterval(t *testing.T) {
	t.Run("accepts twap order with zero interval seconds (defaults applied downstream)", func(t *testing.T) {
		err := ValidateOrderTypeConstraints(PlaceOrderRequest{
			OrderType:       "twap",
			DurationSeconds: 600,
		}, 0)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})

	t.Run("rejects negative interval seconds for twap orders", func(t *testing.T) {
		err := ValidateOrderTypeConstraints(PlaceOrderRequest{
			OrderType:       "twap",
			DurationSeconds: 600,
			IntervalSeconds: -1,
		}, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "intervalSeconds must not be negative")
	})

	t.Run("accepts twap order with positive interval seconds", func(t *testing.T) {
		err := ValidateOrderTypeConstraints(PlaceOrderRequest{
			OrderType:       "twap",
			DurationSeconds: 600,
			IntervalSeconds: 120,
		}, 0)
		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	})
}
