package snaxpot

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateSnaxBallSequence(t *testing.T) {
	require.NoError(t, ValidateSnaxBallSequence([]int{1}))
	require.NoError(t, ValidateSnaxBallSequence([]int{1, 2, 5}))

	err := ValidateSnaxBallSequence(nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errSnaxBallSequenceEmpty)

	err = ValidateSnaxBallSequence([]int{1, 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, errSnaxBallSequenceDuplicate)

	err = ValidateSnaxBallSequence([]int{0})
	require.Error(t, err)
	assert.ErrorIs(t, err, errSnaxBallSequenceInvalid)
}

func Test_AssignSnaxBallSequence(t *testing.T) {
	assignments, err := AssignSnaxBallSequence([]int{1, 2}, 5, 0)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, []int{1, 2, 1, 2, 1}, assignments)

	assignments, err = AssignSnaxBallSequence([]int{1, 2}, 4, 3)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, []int{2, 1, 2, 1}, assignments)

	assignments, err = AssignSnaxBallSequence([]int{1, 2, 3, 4, 5}, 3, math.MaxInt64)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Equal(t, []int{3, 4, 5}, assignments)
}
