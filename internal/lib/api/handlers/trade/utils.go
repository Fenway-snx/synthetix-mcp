package trade

import (
	"fmt"
	"strings"

	snx_lib_api_handlers_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/handlers/utils"
)

// Builds the Redis key used to map a delegate wallet address to its owner
// for whitelist resolution (SNX-5190). Normalizes to lowercase to ensure
// consistent read/write regardless of EIP-55 checksumming.
func delegateWhitelistKey(address string) string {
	return fmt.Sprintf("whitelist:wallet:%s", strings.ToLower(address))
}

// Delegates to the shared utils helper so the mapping logic is defined in
// one place across the trade and info handler packages.
func tradeDirectionToSide(direction string) (string, error) {
	return snx_lib_api_handlers_utils.TradeDirectionToSide(direction)
}
