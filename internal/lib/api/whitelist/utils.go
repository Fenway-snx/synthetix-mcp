package whitelist

import (
	"encoding/json"
	"strings"
)

func parseWalletWhitelist(
	s string,
) (
	m map[WalletAddress]bool,
	err error,
) {
	err = json.Unmarshal([]byte(s), &m)

	if err != nil {
		return
	}

	m = normalizePermissions(m)

	return
}

func normalizeWalletAddress(addr WalletAddress) WalletAddress {
	return WalletAddress(strings.ToLower(string(addr)))
}

func normalizePermissions(m PermissionsMap) PermissionsMap {
	if m == nil {
		return nil
	}

	normalized := make(PermissionsMap, len(m))

	for addr, allowed := range m {
		normalized[normalizeWalletAddress(addr)] = allowed
	}

	return normalized
}
