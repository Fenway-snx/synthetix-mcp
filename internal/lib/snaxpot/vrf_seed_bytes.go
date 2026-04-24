package snaxpot

import (
	"errors"
	"math/big"
)

var (
	errVRFSeedNegative = errors.New("vrf seed must be non-negative")
	errVRFSeedNil      = errors.New("vrf seed is nil")
	errVRFSeedTooWide  = errors.New("vrf seed must fit in 256 bits")
)

// AssignmentSeedFromVRFUInt256 maps the on-chain uint256 VRF seed into the
// canonical 32-byte big-endian encoding used by ticket-number derivation.
func AssignmentSeedFromVRFUInt256(v *big.Int) ([]byte, error) {
	if v == nil {
		return nil, errVRFSeedNil
	}
	if v.Sign() < 0 {
		return nil, errVRFSeedNegative
	}
	if v.BitLen() > 256 {
		return nil, errVRFSeedTooWide
	}

	out := make([]byte, 32)
	v.FillBytes(out)

	return out, nil
}
