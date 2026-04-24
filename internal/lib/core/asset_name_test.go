package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AssetName_MARSHALING(t *testing.T) {

	t.Run("normative case", func(t *testing.T) {

		asset := AssetName("USDT")

		bytes, err := json.Marshal(asset)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"USDT"`, s)

		var asset2 AssetName
		err = json.Unmarshal(bytes, &asset2)

		require.Nil(t, err)
		assert.Equal(t, asset, asset2)
	})

	t.Run("minimum valid case", func(t *testing.T) {

		asset := AssetName("B")

		bytes, err := json.Marshal(asset)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"B"`, s)

		var asset2 AssetName
		err = json.Unmarshal(bytes, &asset2)

		require.Nil(t, err)
		assert.Equal(t, asset, asset2)
	})

	t.Run("empty asset", func(t *testing.T) {

		var asset AssetName

		bytes, err := json.Marshal(asset)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `""`, s)

		var asset2 AssetName
		err = json.Unmarshal(bytes, &asset2)

		require.NotNil(t, err)
		assert.Equal(t, "asset name empty", err.Error())
	})

	t.Run("invalid asset - contains '-'", func(t *testing.T) {

		asset := AssetName("BT-C")

		bytes, err := json.Marshal(asset)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"BT-C"`, s)

		var asset2 AssetName
		err = json.Unmarshal(bytes, &asset2)

		require.NotNil(t, err)
		assert.Equal(t, "asset name invalid", err.Error())
	})

	t.Run("invalid asset - leading non-word", func(t *testing.T) {

		asset := AssetName("/USDT")

		bytes, err := json.Marshal(asset)

		require.Nil(t, err)

		s := string(bytes)

		assert.Equal(t, `"/USDT"`, s)

		var asset2 AssetName
		err = json.Unmarshal(bytes, &asset2)

		require.NotNil(t, err)
		assert.Equal(t, "asset name invalid", err.Error())
	})
}
