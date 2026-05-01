package core

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const (
	WalletAddress_PlaceHolder = WalletAddress("0x0000000000000000000000000000000000000001")
	WalletAddress_Zero        = WalletAddress("0x0000000000000000000000000000000000000000")
)

var (
	Err_WalletAddress_Invalid = errors.New("invalid wallet address")
)

// Internal representation of a wallet address identifier.
type WalletAddress string

// Normalizes wallet addresses to EIP-55 checksum form.
func ChecksumWalletAddress(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", Err_WalletAddress_Invalid
	}
	if !common.IsHexAddress(addr) {
		return "", Err_WalletAddress_Invalid
	}
	return common.HexToAddress(addr).Hex(), nil
}

// I know there is more to it, but this should do for now.
// Note:
// Gotta add some validations and checksum
func NewWalletAddress(input string) (WalletAddress, error) {
	if !common.IsHexAddress(input) {
		return WalletAddress_Zero, Err_WalletAddress_Invalid
	}

	return WalletAddress(input), nil
}

func (wa WalletAddress) String() string {
	return string(wa)
}

// MaskAddress redacts a wallet address string for safe log output, retaining
// only the 0x prefix bytes and the last four characters: e.g. 0x1234...abcd.
// Returns "***" when the input is too short to be a valid address.
func MaskAddress(s string) string {
	if len(s) < 10 {
		return "***"
	}
	return s[:6] + "***" + s[len(s)-4:]
}

// Returns true when two hex address strings refer to the same Ethereum
// account. Comparison is performed on the raw 20-byte representation so it
// is case-insensitive and checksum-agnostic.
func AddressesEqual(a, b string) bool {
	return common.HexToAddress(a) == common.HexToAddress(b)
}

// Returns true when two wallet addresses refer to the same Ethereum account.
func (wa WalletAddress) Equal(other WalletAddress) bool {
	return AddressesEqual(string(wa), string(other))
}

// Returns a redacted form of the address for safe logging.
func (wa WalletAddress) Masked() string {
	return MaskAddress(string(wa))
}

// Returns a lowercase hex representation of addr for exact-match queries
// against stored addresses. The input must be a valid hex address.
func LowercaseAddress(addr string) string {
	return strings.ToLower(common.HexToAddress(addr).Hex())
}
