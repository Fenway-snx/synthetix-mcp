package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateEnum_STRING_CONTEXT(t *testing.T) {
	t.Parallel()

	err := ValidateEnum("maybe", []string{"buy", "sell"}, "side")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "side must be one of")
	assert.Contains(t, err.Error(), "buy")
}

func Test_ValidateEnum_FUNC_CONTEXT(t *testing.T) {
	t.Parallel()

	err := ValidateEnum("maybe", []string{"buy", "sell"}, func() string {
		return "order 2: side"
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "order 2: side must be one of")
}
