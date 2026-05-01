package auth

import snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"

// Shortens Ethereum-like addresses for readable logs and errors.
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
