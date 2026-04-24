package utils

import (
	"context"
	"errors"
	"testing"

	shopspring_decimal "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	v4grpc "github.com/Fenway-snx/synthetix-mcp/internal/lib/message/grpc"
)

type mockSubaccountClientGetAllOITotals struct {
	v4grpc.SubaccountServiceClient
	response *v4grpc.GetAllOITotalsResponse
	err      error
}

var _ v4grpc.SubaccountServiceClient = (*mockSubaccountClientGetAllOITotals)(nil)

func (m *mockSubaccountClientGetAllOITotals) GetAllOITotals(ctx context.Context, in *v4grpc.GetAllOITotalsRequest, opts ...grpc.CallOption) (*v4grpc.GetAllOITotalsResponse, error) {
	return m.response, m.err
}

func TestOpenInterestForAllMarkets(t *testing.T) {
	t.Run("returns parsed OI totals for all symbols", func(t *testing.T) {
		client := &mockSubaccountClientGetAllOITotals{
			response: &v4grpc.GetAllOITotalsResponse{
				Items: []*v4grpc.OITotalsItem{
					{
						Symbol:             "BTC-USD",
						TotalLongQuantity:  "1.5",
						TotalShortQuantity: "2.25",
					},
					{
						Symbol:             "ETH-USD",
						TotalLongQuantity:  "3.0",
						TotalShortQuantity: "4.75",
					},
				},
			},
		}

		got, err := OpenInterestForAllMarkets(context.Background(), client)

		assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, OITotals{
			Long:  shopspring_decimal.RequireFromString("1.5"),
			Short: shopspring_decimal.RequireFromString("2.25"),
		}, got[Symbol("BTC-USD")])
		assert.Equal(t, OITotals{
			Long:  shopspring_decimal.RequireFromString("3.0"),
			Short: shopspring_decimal.RequireFromString("4.75"),
		}, got[Symbol("ETH-USD")])
	})

	t.Run("returns wrapped grpc error", func(t *testing.T) {
		client := &mockSubaccountClientGetAllOITotals{
			err: errors.New("service unavailable"),
		}

		got, err := OpenInterestForAllMarkets(context.Background(), client)

		assert.Nil(t, got)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get all OI totals")
	})

	t.Run("returns error for invalid long quantity", func(t *testing.T) {
		client := &mockSubaccountClientGetAllOITotals{
			response: &v4grpc.GetAllOITotalsResponse{
				Items: []*v4grpc.OITotalsItem{
					{
						Symbol:             "BTC-USD",
						TotalLongQuantity:  "not-a-decimal",
						TotalShortQuantity: "2.25",
					},
				},
			},
		}

		got, err := OpenInterestForAllMarkets(context.Background(), client)

		assert.Nil(t, got)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid total_long_quantity for symbol BTC-USD")
	})

	t.Run("returns error for invalid short quantity", func(t *testing.T) {
		client := &mockSubaccountClientGetAllOITotals{
			response: &v4grpc.GetAllOITotalsResponse{
				Items: []*v4grpc.OITotalsItem{
					{
						Symbol:             "ETH-USD",
						TotalLongQuantity:  "1.23",
						TotalShortQuantity: "not-a-decimal",
					},
				},
			},
		}

		got, err := OpenInterestForAllMarkets(context.Background(), client)

		assert.Nil(t, got)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid total_short_quantity for symbol ETH-USD")
	})
}
