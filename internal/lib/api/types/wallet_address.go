package types

// =========================================================================
// Constants
// =========================================================================

// =========================================================================
// Types
// =========================================================================

// Represents a wallet address.
type WalletAddress string

const (
	WalletAddress_None WalletAddress = ""
)

// =========================================================================
// Utility functions
// =========================================================================

// ===========================
// `WalletAddress`
// ===========================

// Converts a wallet address from a string obtained from a trusted source,
// without any validation.
func WalletAddressFromStringUnvalidated(
	s string,
) WalletAddress {
	return WalletAddress(s)
}

func WalletAddressPtrFromStringPtrUnvalidated(
	p *string,
) *WalletAddress {
	if p == nil {
		return nil
	} else {
		w := WalletAddress(*p)

		return &w
	}
}

func WalletAddressToString(
	v WalletAddress,
) string {
	return string(v)
}
