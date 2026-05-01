package json

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
