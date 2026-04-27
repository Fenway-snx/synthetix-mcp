package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Symbol(t *testing.T) {

	t.Run("marshaling", func(t *testing.T) {

		// normative case
		{
			symbol := Symbol("BTC-USDT")

			bytes, err := json.Marshal(symbol)

			require.Nil(t, err)

			s := string(bytes)

			assert.Equal(t, `"BTC-USDT"`, s)

			var symbol2 Symbol
			err = json.Unmarshal(bytes, &symbol2)

			require.Nil(t, err)
			assert.Equal(t, symbol, symbol2)
		}

		// minimum valid case
		{
			symbol := Symbol("B-U")

			bytes, err := json.Marshal(symbol)

			require.Nil(t, err)

			s := string(bytes)

			assert.Equal(t, `"B-U"`, s)

			var symbol2 Symbol
			err = json.Unmarshal(bytes, &symbol2)

			require.Nil(t, err)
			assert.Equal(t, symbol, symbol2)
		}

		// empty symbol
		{
			var symbol Symbol

			bytes, err := json.Marshal(symbol)

			require.Nil(t, err)

			s := string(bytes)

			assert.Equal(t, `""`, s)

			var symbol2 Symbol
			err = json.Unmarshal(bytes, &symbol2)

			require.NotNil(t, err)
			assert.Equal(t, "symbol name empty", err.Error())
		}

		// invalid symbol - trailing '-'
		{
			symbol := Symbol("BTC-")

			bytes, err := json.Marshal(symbol)

			require.Nil(t, err)

			s := string(bytes)

			assert.Equal(t, `"BTC-"`, s)

			var symbol2 Symbol
			err = json.Unmarshal(bytes, &symbol2)

			require.NotNil(t, err)
			assert.Equal(t, "symbol name invalid", err.Error())
		}

		// invalid symbol - leading '-'
		{
			symbol := Symbol("-USDT")

			bytes, err := json.Marshal(symbol)

			require.Nil(t, err)

			s := string(bytes)

			assert.Equal(t, `"-USDT"`, s)

			var symbol2 Symbol
			err = json.Unmarshal(bytes, &symbol2)

			require.NotNil(t, err)
			assert.Equal(t, "symbol name invalid", err.Error())
		}
	})
}
