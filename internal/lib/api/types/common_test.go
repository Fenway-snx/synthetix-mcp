package types

import (
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

func Test_stringFromDecimal(t *testing.T) {
	tests := []struct {
		name     string
		v        shopspring_decimal.Decimal
		expected string
	}{
		{
			name:     "zero",
			v:        shopspring_decimal.Zero,
			expected: "0",
		},
		{
			name:     "1",
			v:        shopspring_decimal.New(1, 0),
			expected: "1",
		},
		{
			name:     "1, 1",
			v:        shopspring_decimal.New(1, 1),
			expected: "10",
		},
		{
			name:     "12345, -3",
			v:        shopspring_decimal.New(12345, -3),
			expected: "12.345",
		},
		{
			name:     "-123.45678, -5",
			v:        shopspring_decimal.New(-12345678, -5),
			expected: "-123.45678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual := stringFromDecimalUnvalidated(tt.v)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_stringFromDecimalOrBlankWhenZero(t *testing.T) {
	tests := []struct {
		name     string
		v        shopspring_decimal.Decimal
		expected string
	}{
		{
			name:     "zero",
			v:        shopspring_decimal.Zero,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual := stringFromDecimalOrBlankWhenZero(tt.v)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_DirectionFromCoreDirectionUnvalidated(t *testing.T) {

	tests := []struct {
		name     string
		v        snx_lib_core.Direction
		expected Direction
	}{
		{
			name:     "long",
			v:        snx_lib_core.Direction_Long,
			expected: "long",
		},
		{
			name:     "short",
			v:        snx_lib_core.Direction_Short,
			expected: "short",
		},
		{
			name:     "close-long",
			v:        snx_lib_core.Direction_CloseLong,
			expected: "closeLong",
		},
		{
			name:     "close-short",
			v:        snx_lib_core.Direction_CloseShort,
			expected: "closeShort",
		},
		{
			name:     "0",
			v:        snx_lib_core.Direction(-1),
			expected: "UNKNOWN-Direction<v=-1>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual := DirectionFromCoreDirectionUnvalidated(tt.v)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_MarginSummaryEventType(t *testing.T) {

	assert.Equal(t, "marginUpdate", string(MarginSummaryEventType_marginUpdate))
}

func Test_DepositEventType(t *testing.T) {

	assert.Equal(t, "depositCredited", string(DepositEventType_depositCredited))
	assert.Equal(t, "depositReceived", string(DepositEventType_depositReceived))
}

func Test_WithdrawalEventType(t *testing.T) {

	assert.Equal(t, "withdrawal", string(WithdrawalEventType_withdrawal))
}

func Test_OrderTypeFromCoreOrderTypeUnvalidated(t *testing.T) {

	tests := []struct {
		name     string
		v        snx_lib_core.OrderType
		expected OrderType
	}{
		{
			name:     "limit",
			v:        0,
			expected: OrderType("limit"),
		},
		{
			name:     "market",
			v:        1,
			expected: OrderType("market"),
		},
		{
			name:     "stop-market",
			v:        2,
			expected: OrderType("stopMarket"),
		},
		{
			name:     "take-profit-market",
			v:        3,
			expected: OrderType("takeProfitMarket"),
		},
		{
			name:     "stop-limit",
			v:        4,
			expected: OrderType("stopLimit"),
		},
		{
			name:     "take-profit-limit",
			v:        5,
			expected: OrderType("takeProfitLimit"),
		},
		{
			name:     "invalid value: 6",
			v:        6,
			expected: OrderType("UNKNOWN-OrderType<v=6>"),
		},
		{
			name:     "invalid value: 1234",
			v:        1234,
			expected: OrderType("UNKNOWN-OrderType<v=1234>"),
		},
		{
			name:     "invalid value: -1",
			v:        -1,
			expected: OrderType("UNKNOWN-OrderType<v=-1>"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual := OrderTypeFromCoreOrderTypeUnvalidated(tt.v)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_OrderTypeFromCoreOrderType(t *testing.T) {

	tests := []struct {
		name     string
		v        snx_lib_core.OrderType
		err      error
		expected OrderType
	}{
		{
			name:     "limit",
			v:        0,
			err:      nil,
			expected: OrderType("limit"),
		},
		{
			name:     "market",
			v:        1,
			err:      nil,
			expected: OrderType("market"),
		},
		{
			name:     "stop-market",
			v:        2,
			err:      nil,
			expected: OrderType("stopMarket"),
		},
		{
			name:     "take-profit-market",
			v:        3,
			err:      nil,
			expected: OrderType("takeProfitMarket"),
		},
		{
			name:     "stop-limit",
			v:        4,
			err:      nil,
			expected: OrderType("stopLimit"),
		},
		{
			name:     "take-profit-limit",
			v:        5,
			err:      nil,
			expected: OrderType("takeProfitLimit"),
		},
		{
			name: "invalid value: 6",
			v:    6,
			err:  errOrderTypeUnrecognised,
		},
		{
			name: "invalid value: 1234",
			v:    1234,
			err:  errOrderTypeUnrecognised,
		},
		{
			name: "invalid value: -1",
			v:    -1,
			err:  errOrderTypeUnrecognised,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual, err := OrderTypeFromCoreOrderType(tt.v)

			if tt.err != nil {
				assert.Equal(t, tt.err, err)

			} else {
				assert.Nil(t, err)

				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}

func Test_PositionSideFromCorePositionSideUnvalidated(t *testing.T) {

	tests := []struct {
		name     string
		v        snx_lib_core.PositionSide
		expected PositionSide
	}{
		{
			name:     "short",
			v:        snx_lib_core.PositionSideShort,
			expected: PositionSide_short,
		},
		{
			name:     "long",
			v:        snx_lib_core.PositionSideLong,
			expected: PositionSide_long,
		},
		{
			name:     "invalid value: 2",
			v:        2,
			expected: PositionSide("UNKNOWN-PositionSide<v=2>"),
		},
		{
			name:     "invalid value: 1234",
			v:        1234,
			expected: PositionSide("UNKNOWN-PositionSide<v=1234>"),
		},
		{
			name:     "invalid value: -1",
			v:        -1,
			expected: PositionSide("UNKNOWN-PositionSide<v=-1>"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual := PositionSideFromCorePositionSideUnvalidated(tt.v)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_PositionSideFromCorePositionSide(t *testing.T) {

	tests := []struct {
		name     string
		v        snx_lib_core.PositionSide
		err      error
		expected PositionSide
	}{
		{
			name:     "short",
			v:        snx_lib_core.PositionSideShort,
			err:      nil,
			expected: PositionSide_short,
		},
		{
			name:     "long",
			v:        snx_lib_core.PositionSideLong,
			err:      nil,
			expected: PositionSide_long,
		},
		{
			name: "invalid value: 2",
			v:    2,
			err:  errPositionSideUnrecognised,
		},
		{
			name: "invalid value: 1234",
			v:    1234,
			err:  errPositionSideUnrecognised,
		},
		{
			name: "invalid value: -1",
			v:    -1,
			err:  errPositionSideUnrecognised,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual, err := PositionSideFromCorePositionSide(tt.v)

			if tt.err != nil {
				assert.Equal(t, tt.err, err)

			} else {
				assert.Nil(t, err)

				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}

func Test_SideFromCoreDirectionUnvalidated(t *testing.T) {

	tests := []struct {
		name     string
		v        snx_lib_core.Direction
		expected Side
	}{
		{
			name:     "open-short",
			v:        snx_lib_core.Direction_Short,
			expected: Side_sell,
		},
		{
			name:     "open-long",
			v:        snx_lib_core.Direction_Long,
			expected: Side_buy,
		},
		{
			name:     "close-short",
			v:        snx_lib_core.Direction_CloseShort,
			expected: Side_buy,
		},
		{
			name:     "close-long",
			v:        snx_lib_core.Direction_CloseLong,
			expected: Side_sell,
		},
		{
			name:     "invalid value: 4",
			v:        4,
			expected: Side("UNKNOWN-Side<v=4>"),
		},
		{
			name:     "invalid value: 1234",
			v:        1234,
			expected: Side("UNKNOWN-Side<v=1234>"),
		},
		{
			name:     "invalid value: -1",
			v:        -1,
			expected: Side("UNKNOWN-Side<v=-1>"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual := SideFromCoreDirectionUnvalidated(tt.v)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func Test_SideFromCoreDirection(t *testing.T) {

	tests := []struct {
		name     string
		v        snx_lib_core.Direction
		err      error
		expected Side
	}{
		{
			name:     "open-short",
			v:        snx_lib_core.Direction_Short,
			err:      nil,
			expected: Side_sell,
		},
		{
			name:     "open-long",
			v:        snx_lib_core.Direction_Long,
			err:      nil,
			expected: Side_buy,
		},
		{
			name:     "close-short",
			v:        snx_lib_core.Direction_CloseShort,
			err:      nil,
			expected: Side_buy,
		},
		{
			name:     "close-long",
			v:        snx_lib_core.Direction_CloseLong,
			err:      nil,
			expected: Side_sell,
		},
		{
			name: "invalid value: 4",
			v:    4,
			err:  errSideUnrecognised,
		},
		{
			name: "invalid value: 1234",
			v:    1234,
			err:  errSideUnrecognised,
		},
		{
			name: "invalid value: -1",
			v:    -1,
			err:  errSideUnrecognised,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actual, err := SideFromCoreDirection(tt.v)

			if tt.err != nil {
				assert.Equal(t, tt.err, err)

			} else {
				assert.Nil(t, err)

				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}

func Test_SubAccountIdFromInt(t *testing.T) {

	tests := []struct {
		input               int64
		expectedResult      SubAccountId
		expectedErrContains string
		expectedRawResult   SubAccountId
	}{
		{
			input:               0,
			expectedResult:      SubAccountId_Empty,
			expectedErrContains: "subaccount id cannot be zero",
			expectedRawResult:   SubAccountId("0"),
		},
		{
			input:               -1,
			expectedResult:      SubAccountId_Empty,
			expectedErrContains: "cannot be negative",
			expectedRawResult:   SubAccountId("-1"),
		},
		{
			input:               -123_456_789,
			expectedResult:      SubAccountId_Empty,
			expectedErrContains: "cannot be negative",
			expectedRawResult:   SubAccountId("-123456789"),
		},
		{
			input:               1,
			expectedResult:      SubAccountId("1"),
			expectedErrContains: "",
			expectedRawResult:   SubAccountId("1"),
		},
		{
			input:               123_456_789,
			expectedResult:      SubAccountId("123456789"),
			expectedErrContains: "",
			expectedRawResult:   SubAccountId("123456789"),
		},
		{
			input:               SubAccountMaximumValidValue,
			expectedResult:      SubAccountId("9223372036854775806"),
			expectedErrContains: "",
			expectedRawResult:   SubAccountId("9223372036854775806"),
		},
	}

	for i, tt := range tests {

		// SubAccountIdFromUint

		actual, err := SubAccountIdFromInt(tt.input)
		if tt.expectedErrContains != "" {
			if err != nil {
				assert.Contains(t, err.Error(), tt.expectedErrContains)
			} else {
				assert.Fail(t, "test case %d was expected to fail with err containing '%s', but did not do so", i, tt.expectedErrContains)
			}
		} else {
			// TODO: move this somewhere common
			errStringOrEmpty := func(err error) string {
				if err == nil {
					return ""
				} else {
					return err.Error()
				}
			}
			assert.Nil(t, err, "test case %d was expected not to fail but has done so with err containing '%s'", i, errStringOrEmpty(err))

			assert.Equal(t, tt.expectedResult, actual)
		}

		// SubAccountIdFromIntUnvalidated(

		actual_2 := SubAccountIdFromIntUnvalidated(tt.input)

		assert.Equal(t, tt.expectedRawResult, actual_2)
	}
}

func Test_SubAccountIdToCoreSubaccountId(t *testing.T) {
}

func Test_TriggerPriceType(t *testing.T) {

	t.Run("constants have expected values", func(t *testing.T) {

		assert.Equal(t, "last", string(TriggerPriceType_last))
		assert.Equal(t, "mark", string(TriggerPriceType_mark))
	})

	t.Run("parse via APITriggerPriceTypeFromString", func(t *testing.T) {

		tests := []struct {
			s          string
			recognised bool
			result     TriggerPriceType
			contains   string
		}{
			// valid (exact)
			{
				s:          "last",
				recognised: true,
				result:     TriggerPriceType_last,
			},
			{
				s:          "mark",
				recognised: true,
				result:     TriggerPriceType_mark,
			},
			// empty string defaults to mark
			{
				s:          "",
				recognised: true,
				result:     TriggerPriceType_mark,
			},
			// inexact (case/whitespace) is rejected
			{
				s:          " Last",
				recognised: false,
				contains:   "unrecognised",
			},
			{
				s:          "   mArK ",
				recognised: false,
				contains:   "unrecognised",
			},
			// invalid
			{
				s:          "indexer",
				recognised: false,
				contains:   "unrecognised",
			},
			{
				s:          "  marKET ",
				recognised: false,
				contains:   "unrecognised",
			},
		}

		for _, tt := range tests {

			tpt, err := APITriggerPriceTypeFromString(tt.s)

			if tt.recognised {

				require.Nil(t, err)

				assert.Equal(t, tt.result, tpt)
			} else {

				require.NotNil(t, err, "expected `err` to be not `nil`")
				require.Contains(t, err.Error(), tt.contains)
			}
		}
	})
}

func Benchmark_TriggerPriceType(b *testing.B) {

	type testCase struct {
		name string
		s    string
	}

	groups := []struct {
		groupName string
		tests     []testCase
	}{
		{
			groupName: "pass",
			tests: []testCase{
				{name: "empty_default", s: ""},
				{name: "last", s: "last"},
				{name: "mark", s: "mark"},
			},
		},
		{
			groupName: "fail",
			tests: []testCase{
				{name: "Last_with_space", s: " Last"},
				{name: "mArK_with_spaces", s: "   mArK "},
				{name: "market", s: "marKET"},
				{
					name: "long_invalid",
					s:    "this-is-a-very-long-and-obviously-invalid-trigger-price-type",
				},
			},
		},
	}

	for _, g := range groups {
		b.Run(g.groupName, func(b *testing.B) {

			for _, tt := range g.tests {

				b.Run(tt.name, func(b *testing.B) {

					b.Run("FromString", func(b *testing.B) {
						for b.Loop() {
							_, _ = APITriggerPriceTypeFromString(tt.s)
						}
					})
				})
			}
		})
	}
}
