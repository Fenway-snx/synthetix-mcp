package utils

import (
	"errors"
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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

type VenueOrderId uint64
type Timestamp int64

type testWithIgnoredField struct {
	ID      uint64 `json:"id"`
	Secret  string `json:"-"`
	Visible string `json:"visible"`
}

type testWithOmitempty struct {
	ID       uint64 `json:",omitempty"`
	Name     string `json:"name,omitempty"`
	BigValue uint64 `json:"big_value,omitempty"`
}

type testWithNamedTypes struct {
	VenueOrderId VenueOrderId `json:"order_id"`
	Timestamp    Timestamp    `json:"timestamp"`
	SmallID      VenueOrderId `json:"small_id"`
}

type stringmaker uint64

func (s stringmaker) String() string {
	return "foo"
}

func Test_StringifyBigInts(t *testing.T) {
	t.Run("errors work", func(t *testing.T) {
		result := stringifyBigInts(errors.New("some error text"), 0)
		assert.Equal(t, "some error text", result)
	})

	t.Run("raw bytes get cast to strings", func(t *testing.T) {
		result := stringifyBigInts([]byte("some bytes"), 0)
		assert.Equal(t, "some bytes", result)
	})

	t.Run("decimal works", func(t *testing.T) {
		result := stringifyBigInts(shopspring_decimal.NewFromInt(9007199254740992), 0)
		assert.Equal(t, "9007199254740992", result)
	})

	t.Run("custom stringifier check", func(t *testing.T) {
		result := stringifyBigInts(stringmaker(9007199254740992), 0)
		assert.Equal(t, "foo", result)
	})

	t.Run("nil custom stringifier check", func(t *testing.T) {
		var input *stringmaker = nil
		result := stringifyBigInts(input, 0)
		assert.Equal(t, "<nil>", result)
	})

	t.Run("ms timestamp should not be stringified", func(t *testing.T) {
		input := 1765861302785 // js Date.now()
		result := stringifyBigInts(input, 0)
		assert.Equal(t, 1765861302785, result)
	})

	t.Run("big uint64", func(t *testing.T) {
		result := stringifyBigInts(uint64(9007199254740992), 0)
		assert.Equal(t, "9007199254740992", result)
	})

	t.Run("safe uint64 unchanged", func(t *testing.T) {
		result := stringifyBigInts(uint64(12345), 0)
		assert.Equal(t, uint64(12345), result)
	})

	t.Run("big int64", func(t *testing.T) {
		result := stringifyBigInts(int64(9007199254740992), 0)
		assert.Equal(t, "9007199254740992", result)
	})

	t.Run("safe int64 unchanged", func(t *testing.T) {
		result := stringifyBigInts(int64(12345), 0)
		assert.Equal(t, int64(12345), result)
	})

	t.Run("[]uint64 with big ints", func(t *testing.T) {
		result := stringifyBigInts([]uint64{9007199254740992, 123}, 0)
		expected := []any{"9007199254740992", uint64(123)}
		assert.Equal(t, expected, result)
	})

	t.Run("[]int64 with big ints", func(t *testing.T) {
		result := stringifyBigInts([]int64{9007199254740992, 123}, 0)
		expected := []any{"9007199254740992", int64(123)}
		assert.Equal(t, expected, result)
	})

	t.Run("struct with big ints", func(t *testing.T) {
		pos := &testPosition{
			ID:           9007199254740992,
			SubAccountID: 1234567890123456789,
			Symbol:       "BTC-USD",
			OrderIds:     []uint64{9007199254740992, 123},
		}

		result := stringifyBigInts(pos, 0)
		m := result.(map[string]any)

		assert.Equal(t, "9007199254740992", m["id"])
		assert.Equal(t, "1234567890123456789", m["sub_account_id"])
		assert.Equal(t, "BTC-USD", m["symbol"])

		ids := m["order_ids"].([]any)
		assert.Equal(t, "9007199254740992", ids[0])
		assert.Equal(t, uint64(123), ids[1])
	})

	t.Run("3+ levels deep nested structures", func(t *testing.T) {
		account := &testAccount{
			ID:    1234567890123456789,
			Owner: "0x123",
			Positions: []testPosition{
				{
					ID:           9007199254740992,
					SubAccountID: 9007199254740993,
					Symbol:       "BTC-USD",
					OrderIds:     []uint64{9007199254740994, 100},
					Orders: []testOrder{
						{ID: 9007199254740995, Price: "50000", Amount: "1.5"},
						{ID: 200, Price: "51000", Amount: "2.0"},
					},
				},
				{
					ID:           9007199254740996,
					SubAccountID: 300,
					Symbol:       "ETH-USD",
					OrderIds:     []uint64{400, 500},
					Orders:       []testOrder{},
				},
			},
			Metadata: testMetadata{
				CreatedAt: 9007199254740997,
				Tags: []testTag{
					{ID: 9007199254740998, Label: "vip"},
					{ID: 600, Label: "active"},
				},
				Nested: &testNestedConfig{
					Level3ID: 9007199254740999,
					Items: []testLevel4{
						{ID: 9007199254741000, Value: "deep1"},
						{ID: 700, Value: "deep2"},
					},
				},
			},
		}

		result := stringifyBigInts(account, 0)
		m := result.(map[string]any)

		assert.Equal(t, "1234567890123456789", m["id"])
		assert.Equal(t, "0x123", m["owner"])

		positions := m["positions"].([]any)
		assert.Len(t, positions, 2)

		pos1 := positions[0].(map[string]any)
		assert.Equal(t, "9007199254740992", pos1["id"])
		assert.Equal(t, "9007199254740993", pos1["sub_account_id"])

		orderIds := pos1["order_ids"].([]any)
		assert.Equal(t, "9007199254740994", orderIds[0])
		assert.Equal(t, uint64(100), orderIds[1])

		orders := pos1["orders"].([]any)
		assert.Len(t, orders, 2)
		order1 := orders[0].(map[string]any)
		assert.Equal(t, "9007199254740995", order1["id"])
		order2 := orders[1].(map[string]any)
		assert.Equal(t, uint64(200), order2["id"])

		pos2 := positions[1].(map[string]any)
		assert.Equal(t, "9007199254740996", pos2["id"])
		assert.Equal(t, int64(300), pos2["sub_account_id"])

		metadata := m["metadata"].(map[string]any)
		assert.Equal(t, "9007199254740997", metadata["created_at"])

		tags := metadata["tags"].([]any)
		tag1 := tags[0].(map[string]any)
		assert.Equal(t, "9007199254740998", tag1["id"])
		tag2 := tags[1].(map[string]any)
		assert.Equal(t, uint64(600), tag2["id"])

		nested := metadata["nested"].(map[string]any)
		assert.Equal(t, "9007199254740999", nested["level3_id"])

		items := nested["items"].([]any)
		item1 := items[0].(map[string]any)
		assert.Equal(t, "9007199254741000", item1["id"])
		assert.Equal(t, "deep1", item1["value"])
		item2 := items[1].(map[string]any)
		assert.Equal(t, uint64(700), item2["id"])
	})

	t.Run("nil pointer", func(t *testing.T) {
		var pos *testPosition = nil
		result := stringifyBigInts(pos, 0)
		assert.Nil(t, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		result := stringifyBigInts([]uint64{}, 0)
		assert.Equal(t, []any{}, result)
	})

	t.Run("json:- fields are skipped", func(t *testing.T) {
		obj := &testWithIgnoredField{
			ID:      9007199254740992,
			Secret:  "should-not-appear",
			Visible: "should-appear",
		}

		result := stringifyBigInts(obj, 0)
		m := result.(map[string]any)

		assert.Equal(t, "9007199254740992", m["id"])
		assert.Equal(t, "should-appear", m["visible"])
		_, hasSecret := m["Secret"]
		assert.False(t, hasSecret, "json:- field should be skipped")
		_, hasSecretLower := m["secret"]
		assert.False(t, hasSecretLower, "json:- field should be skipped")
	})

	t.Run("json:,omitempty uses field name", func(t *testing.T) {
		obj := &testWithOmitempty{
			ID:       9007199254740992,
			Name:     "test",
			BigValue: 9007199254740993,
		}

		result := stringifyBigInts(obj, 0)
		m := result.(map[string]any)

		assert.Equal(t, "9007199254740992", m["ID"], "field with ,omitempty should use Go field name")
		assert.Equal(t, "test", m["name"])
		assert.Equal(t, "9007199254740993", m["big_value"])
	})

	t.Run("map with big ints", func(t *testing.T) {
		m := map[string]any{
			"balance":    uint64(1234567890123456789),
			"small":      uint64(123),
			"trade_id":   int64(9007199254740992),
			"nested_map": map[string]any{"deep_id": uint64(9007199254740994)},
		}

		result := stringifyBigInts(m, 0)
		rm := result.(map[string]any)

		assert.Equal(t, "1234567890123456789", rm["balance"])
		assert.Equal(t, uint64(123), rm["small"])
		assert.Equal(t, "9007199254740992", rm["trade_id"])

		nestedMap := rm["nested_map"].(map[string]any)
		assert.Equal(t, "9007199254740994", nestedMap["deep_id"])
	})

	t.Run("named integer types", func(t *testing.T) {
		obj := &testWithNamedTypes{
			VenueOrderId: VenueOrderId(1234567890123456789),
			Timestamp:    Timestamp(9007199254740992),
			SmallID:      VenueOrderId(123),
		}

		result := stringifyBigInts(obj, 0)
		m := result.(map[string]any)

		assert.Equal(t, "1234567890123456789", m["order_id"])
		assert.Equal(t, "9007199254740992", m["timestamp"])
		assert.Equal(t, VenueOrderId(123), m["small_id"])
	})

	t.Run("direct named type value", func(t *testing.T) {
		bigOrderID := VenueOrderId(1234567890123456789)
		result := stringifyBigInts(bigOrderID, 0)
		assert.Equal(t, "1234567890123456789", result)

		smallOrderID := VenueOrderId(123)
		result = stringifyBigInts(smallOrderID, 0)
		assert.Equal(t, VenueOrderId(123), result)
	})
}

func Test_StringifyAllBigInts(t *testing.T) {
	t.Run("transforms big ints in values", func(t *testing.T) {
		input := []any{"trade_id", uint64(9007199254740992), "count", uint64(123)}
		result := StringifyAllBigInts(input, 0)

		assert.Equal(t, "trade_id", result[0])
		assert.Equal(t, "9007199254740992", result[1])
		assert.Equal(t, "count", result[2])
		assert.Equal(t, uint64(123), result[3])
	})

	t.Run("does not mutate original slice", func(t *testing.T) {
		original := []any{"trade_id", uint64(9007199254740992), "count", uint64(123)}
		originalCopy := make([]any, len(original))
		copy(originalCopy, original)

		_ = StringifyAllBigInts(original, 0)

		assert.Equal(t, originalCopy[0], original[0])
		assert.Equal(t, originalCopy[1], original[1], "original slice should not be mutated")
		assert.Equal(t, originalCopy[2], original[2])
		assert.Equal(t, originalCopy[3], original[3])
	})
}
