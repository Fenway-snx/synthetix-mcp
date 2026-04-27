package types

// TODO: rename to asset_name.go when type is renamed to `AssetName`

// =========================================================================
// Constants
// =========================================================================

// =========================================================================
// Types
// =========================================================================

// Represents an asset name.
//
// TODO: SNX-6098: need to rename this to `AssetName`, but at that time build out an API that allows `api.AssetName` <=> `core.AssetName`
type Asset string

const (
	AssetName_None Asset = ""
)

// =========================================================================
// Utility functions
// =========================================================================

// ===========================
// `Asset`
// ===========================

// Converts an asset name from a string obtained from a trusted source,
// without any validation.
func AssetNameFromStringUnvalidated(
	s string,
) Asset {
	return Asset(s)
}

// ===========================
// `Asset`
// ===========================

func AssetNameToString(
	v Asset,
) string {
	return string(v)
}
