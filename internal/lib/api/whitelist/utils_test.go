package whitelist

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseWalletWhitelist(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		input := ""

		m, err := parseWalletWhitelist(input)

		require.NotNil(t, err)

		assert.Nil(t, m)
	})

	t.Run("empty object", func(t *testing.T) {
		input := "{}"

		m, err := parseWalletWhitelist(input)

		require.Nil(t, err)

		assert.NotNil(t, m)
		assert.Equal(t, 0, len(m))
	})

	t.Run("valid object with single entry", func(t *testing.T) {
		input := `{
  "0x79741ea6dde18fefb2805f824b4a3992aafbdff5cbad5077719b10f0f3ac70d4": true
}`

		m, err := parseWalletWhitelist(input)

		require.Nil(t, err)

		assert.NotNil(t, m)
		assert.Equal(t, 1, len(m))

		var isPermitted bool
		var exists bool

		isPermitted, exists = m["abc"]

		assert.False(t, isPermitted)
		assert.False(t, exists)

		isPermitted, exists = m["0x79741ea6dde18fefb2805f824b4a3992aafbdff5cbad5077719b10f0f3ac70d4"]

		assert.True(t, isPermitted)
		assert.True(t, exists)
	})

	t.Run("valid object with two entries", func(t *testing.T) {
		input := `{
  "0x2f328b79c8a94e1eec003d6dffe0c61a46a2f1ec7fa14a299643037e2e9242ca": true,
  "0x79741ea6dde18fefb2805f824b4a3992aafbdff5cbad5077719b10f0f3ac70d4": false
}`

		m, err := parseWalletWhitelist(input)

		require.Nil(t, err)

		assert.NotNil(t, m)
		assert.Equal(t, 2, len(m))

		var isPermitted bool
		var exists bool

		isPermitted, exists = m["abc"]

		assert.False(t, isPermitted)
		assert.False(t, exists)

		isPermitted, exists = m["0x2f328b79c8a94e1eec003d6dffe0c61a46a2f1ec7fa14a299643037e2e9242ca"]

		assert.True(t, isPermitted)
		assert.True(t, exists)

		isPermitted, exists = m["0x79741ea6dde18fefb2805f824b4a3992aafbdff5cbad5077719b10f0f3ac70d4"]

		assert.False(t, isPermitted)
		assert.True(t, exists)
	})

	t.Run("invalid object with two entries", func(t *testing.T) {
		input := `{
  "0x2f328b79c8a94e1eec003d6dffe0c61a46a2f1ec7fa14a299643037e2e9242ca": true,
  "0x79741ea6dde18fefb2805f824b4a3992aafbdff5cbad5077719b10f0f3ac70d4": false,
}`

		m, err := parseWalletWhitelist(input)

		require.NotNil(t, err)

		assert.Nil(t, m)
	})

	t.Run("normalizes keys to lowercase", func(t *testing.T) {
		input := `{
  "0xAA": true,
  "0xBb": false
}`

		m, err := parseWalletWhitelist(input)

		require.Nil(t, err)
		require.NotNil(t, m)

		_, existsMixedUpper := m["0xAA"]
		assert.False(t, existsMixedUpper)

		assert.Equal(t, true, m["0xaa"])
		assert.Equal(t, false, m["0xbb"])
	})
}
