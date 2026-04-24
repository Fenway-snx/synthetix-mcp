package zerolog

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testOrder struct {
	ID     uint64 `json:"id"`
	Price  string `json:"price"`
	Amount string `json:"amount"`
}

type testPosition struct {
	ID           uint64      `json:"id"`
	SubAccountID int64       `json:"sub_account_id"`
	Symbol       string      `json:"symbol"`
	OrderIds     []uint64    `json:"order_ids"`
	Orders       []testOrder `json:"orders"`
}

type testAccount struct {
	ID        uint64         `json:"id"`
	Owner     string         `json:"owner"`
	Positions []testPosition `json:"positions"`
	Metadata  testMetadata   `json:"metadata"`
}

type testMetadata struct {
	CreatedAt uint64            `json:"created_at"`
	Tags      []testTag         `json:"tags"`
	Settings  map[string]any    `json:"settings"`
	Nested    *testNestedConfig `json:"nested"`
}

type testTag struct {
	ID    uint64 `json:"id"`
	Label string `json:"label"`
}

type testNestedConfig struct {
	Level3ID uint64       `json:"level3_id"`
	Items    []testLevel4 `json:"items"`
}

type testLevel4 struct {
	ID    uint64 `json:"id"`
	Value string `json:"value"`
}

func Test_LoggerKeyValsBigInts(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf)

	logger.Info("test message",
		"trade_id", int64(9007199254740992),
		"sub_account_id", uint64(1234567890123456789),
		"count", uint64(123),
	)

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	assert.Equal(t, "9007199254740992", result["trade_id"])
	assert.Equal(t, "1234567890123456789", result["sub_account_id"])
	assert.Equal(t, float64(123), result["count"])
}

func Test_LoggerWithNestedStruct(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf)

	account := &testAccount{
		ID:    1234567890123456789,
		Owner: "0x123",
		Positions: []testPosition{
			{
				ID:           9007199254740992,
				SubAccountID: 9007199254740993,
				Symbol:       "BTC-USD",
				Orders: []testOrder{
					{ID: 9007199254740995, Price: "50000", Amount: "1.5"},
				},
			},
		},
		Metadata: testMetadata{
			CreatedAt: 9007199254740997,
			Nested: &testNestedConfig{
				Level3ID: 9007199254740999,
				Items: []testLevel4{
					{ID: 9007199254741000, Value: "deep"},
				},
			},
		},
	}

	logger.Info("account update", "account", account)

	var result map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

	acc := result["account"].(map[string]any)
	assert.Equal(t, "1234567890123456789", acc["id"])

	positions := acc["positions"].([]any)
	pos := positions[0].(map[string]any)
	assert.Equal(t, "9007199254740992", pos["id"])

	orders := pos["orders"].([]any)
	order := orders[0].(map[string]any)
	assert.Equal(t, "9007199254740995", order["id"])

	metadata := acc["metadata"].(map[string]any)
	nested := metadata["nested"].(map[string]any)
	assert.Equal(t, "9007199254740999", nested["level3_id"])

	items := nested["items"].([]any)
	item := items[0].(map[string]any)
	assert.Equal(t, "9007199254741000", item["id"])
}
