package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HandlerParams_MARSHALING(t *testing.T) {

	t.Run("empty", func(t *testing.T) {

		input := []byte(`{}`)

		var hp HandlerParams
		err := json.Unmarshal(input, &hp)

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, map[string]any{}, hp.Map())
	})

	t.Run("single pair", func(t *testing.T) {

		input := []byte(`{"k":true}`)

		var hp HandlerParams
		err := json.Unmarshal(input, &hp)

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, 1, len(hp.Map()))
		assert.Equal(t, map[string]any{"k": true}, hp.Map())
	})

	t.Run("two pairs", func(t *testing.T) {

		input := []byte(`{"k1":"v1", "k2":"v2"}`)

		var hp HandlerParams
		err := json.Unmarshal(input, &hp)

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		assert.Equal(t, 2, len(hp.Map()))
		assert.Equal(t, map[string]any{"k1": "v1", "k2": "v2"}, hp.Map())
	})
}
