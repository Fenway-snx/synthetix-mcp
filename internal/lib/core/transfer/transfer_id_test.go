package transfer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewId(t *testing.T) {
	t.Run("valid positive id", func(t *testing.T) {
		id, err := NewId(1)

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Id(1), id)
	})

	t.Run("valid large positive id", func(t *testing.T) {
		id, err := NewId(9223372036854775807)

		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, Id(9223372036854775807), id)
	})

	t.Run("zero id returns error", func(t *testing.T) {
		id, err := NewId(0)

		require.Error(t, err)
		assert.Equal(t, errInvalidId, err)
		assert.Equal(t, Id(0), id)
	})

	t.Run("negative id returns error", func(t *testing.T) {
		id, err := NewId(-1)

		require.Error(t, err)
		assert.Equal(t, errInvalidId, err)
		assert.Equal(t, Id(0), id)
	})
}

func Test_Id_Zero_Constant(t *testing.T) {
	t.Run("Zero constant is zero value", func(t *testing.T) {
		assert.Equal(t, Id(0), IdZero)
	})
}
