package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SubAccountId_Scan(t *testing.T) {
	t.Run("int64", func(t *testing.T) {
		var s SubAccountId
		require.NoError(t, s.Scan(int64(42)))
		assert.Equal(t, SubAccountId(42), s)
	})

	t.Run("int32", func(t *testing.T) {
		var s SubAccountId
		require.NoError(t, s.Scan(int32(42)))
		assert.Equal(t, SubAccountId(42), s)
	})

	t.Run("int", func(t *testing.T) {
		var s SubAccountId
		require.NoError(t, s.Scan(int(42)))
		assert.Equal(t, SubAccountId(42), s)
	})

	t.Run("nil defaults to zero", func(t *testing.T) {
		s := SubAccountId(99)
		require.NoError(t, s.Scan(nil))
		assert.Equal(t, SubAccountId_Zero, s)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		var s SubAccountId
		err := s.Scan("not a number")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot scan string into SubAccountId")
	})

	t.Run("sentinel values round-trip", func(t *testing.T) {
		for _, sentinel := range []SubAccountId{
			SubAccountId_LCF,
			SubAccountId_SlpPlaceHolder,
			SubAccountId_TF,
			SubAccountId_WF,
		} {
			var s SubAccountId
			require.NoError(t, s.Scan(int64(sentinel)))
			assert.Equal(t, sentinel, s)
		}
	})
}

func Test_SubAccountId_Value(t *testing.T) {
	t.Run("returns int64", func(t *testing.T) {
		v, err := SubAccountId(42).Value()
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, int64(42), v)
	})

	t.Run("zero value", func(t *testing.T) {
		v, err := SubAccountId_Zero.Value()
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, int64(0), v)
	})

	t.Run("sentinel values", func(t *testing.T) {
		v, err := SubAccountId_SlpPlaceHolder.Value()
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
		assert.Equal(t, int64(104), v)
	})
}
