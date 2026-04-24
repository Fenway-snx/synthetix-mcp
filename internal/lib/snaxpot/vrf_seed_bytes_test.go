package snaxpot

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AssignmentSeedFromVRFUInt256_NIL(t *testing.T) {
	_, err := AssignmentSeedFromVRFUInt256(nil)

	require.ErrorIs(t, err, errVRFSeedNil)
}

func Test_AssignmentSeedFromVRFUInt256_NEGATIVE(t *testing.T) {
	_, err := AssignmentSeedFromVRFUInt256(big.NewInt(-1))

	require.ErrorIs(t, err, errVRFSeedNegative)
}

func Test_AssignmentSeedFromVRFUInt256_TOO_WIDE(t *testing.T) {
	input := new(big.Int).Lsh(big.NewInt(1), 256)

	_, err := AssignmentSeedFromVRFUInt256(input)

	require.ErrorIs(t, err, errVRFSeedTooWide)
}

func Test_AssignmentSeedFromVRFUInt256_ROUND_TRIP(t *testing.T) {
	input := new(big.Int)
	input.SetString("123456789abcdef123456789abcdef", 16)

	out, err := AssignmentSeedFromVRFUInt256(input)

	require.NoError(
		t,
		err,
		"expected `err` to be `nil`, but it was '%[1]s' (%[1]T)",
		err,
	)
	require.Len(t, out, 32)
	assert.Equal(t, input, new(big.Int).SetBytes(out))
}
