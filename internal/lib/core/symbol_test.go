package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Symbol_CONVERSION(t *testing.T) {

	t.Run("from AssetName", func(t *testing.T) {

		{
			assetNames := []AssetName{
				AssetName("USDT"),
				AssetName("cbBTC"),
				AssetName("wstETH"),
				AssetName("wETH"),
				AssetName("sUSDe"),
			}

			for _, assetName := range assetNames {

				symbol := assetName.Symbol()

				assert.Equal(t, string(assetName), string(symbol))
			}
		}
	})

	t.Run("from MarketName", func(t *testing.T) {

		{
			marketNames := []MarketName{
				MarketName("BTC-USDT"),
				MarketName("ETH-USDT"),
				MarketName("XRP-USDT"),
			}

			for _, marketName := range marketNames {

				symbol := marketName.Symbol()

				assert.Equal(t, string(marketName), string(symbol))
			}
		}
	})

	t.Run("to AssetName", func(t *testing.T) {

		{
		}
	})
}

func Test_Symbol_MARSHALING(t *testing.T) {

	// T.B.C.
}
