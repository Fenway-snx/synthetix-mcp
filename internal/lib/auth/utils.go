package auth

import snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"

// Returns a shortened form of an Ethereum-like address such as
// "0xABC...DEF" for readability in error messages and logs. It preserves
// the original casing of the input string. If the input is already short,
// it is returned unchanged.
func ShortAddress(walletAddress snx_lib_api_types.WalletAddress) string {
	addr := string(walletAddress)

	if len(addr) == 0 {
		return addr
	}
	if len(addr) <= 10 {
		return addr
	}
	// keep 0x + next 3 and last 3 (e.g. 0xABC...DEF)
	prefixLen := 5
	if len(addr) < prefixLen+3 {
		return addr
	}
	prefix := addr[:prefixLen]
	suffix := addr[len(addr)-3:]
	return prefix + "..." + suffix
}
