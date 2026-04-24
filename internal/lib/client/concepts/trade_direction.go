// Package concepts holds cross-cutting client vocabulary derived from core
// domain types. It must not import lib/api.
package concepts

import (
	snx_lib_core "github.com/Fenway-snx/synthetix-mcp/internal/lib/core"
)

// PrimaryTradeDirectionDisplayString returns fixed English phrases for trade
// direction on surfaces such as trade history and WebSocket trade events
// ("Open Long", "Close Short", ...). Unrecognised values yield "Unknown".
func PrimaryTradeDirectionDisplayString(d snx_lib_core.Direction) string {
	switch d {
	case snx_lib_core.Direction_Long:
		return "Open Long"
	case snx_lib_core.Direction_Short:
		return "Open Short"
	case snx_lib_core.Direction_CloseLong:
		return "Close Long"
	case snx_lib_core.Direction_CloseShort:
		return "Close Short"
	default:
		return "Unknown"
	}
}
