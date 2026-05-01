package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MarketName_MARSHALING(t *testing.T) {

	t.Run("normative case", func(t *testing.T) {

		market := MarketName("BTC-USDT")

		bytes, err := json.Marshal(market)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"BTC-USDT"`, s)

		var market2 MarketName
		err = json.Unmarshal(bytes, &market2)

		require.Nil(t, err)
		assert.Equal(t, market, market2)
	})

	t.Run("minimum valid case", func(t *testing.T) {

		market := MarketName("B-U")

		bytes, err := json.Marshal(market)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"B-U"`, s)

		var market2 MarketName
		err = json.Unmarshal(bytes, &market2)

		require.Nil(t, err)
		assert.Equal(t, market, market2)
	})

	t.Run("empty market", func(t *testing.T) {

		var market MarketName

		bytes, err := json.Marshal(market)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `""`, s)

		var market2 MarketName
		err = json.Unmarshal(bytes, &market2)

		require.NotNil(t, err)
		assert.Equal(t, "market name empty", err.Error())
	})

	t.Run("invalid market - trailing '-'", func(t *testing.T) {

		market := MarketName("BTC-")

		bytes, err := json.Marshal(market)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"BTC-"`, s)

		var market2 MarketName
		err = json.Unmarshal(bytes, &market2)

		require.NotNil(t, err)
		assert.Equal(t, "market name invalid", err.Error())
	})

	t.Run("invalid market - leading '-'", func(t *testing.T) {

		market := MarketName("-USDT")

		bytes, err := json.Marshal(market)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"-USDT"`, s)

		var market2 MarketName
		err = json.Unmarshal(bytes, &market2)

		require.NotNil(t, err)
		assert.Equal(t, "market name invalid", err.Error())
	})
}
