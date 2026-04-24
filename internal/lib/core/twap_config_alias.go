package core

import snx_lib_core_twap "github.com/Fenway-snx/synthetix-mcp/internal/lib/core/twap"

// TWAPConfig is an alias for twap.Config so orders and validation can keep the
// historical name while configuration lives in lib/core/twap.
type TWAPConfig = snx_lib_core_twap.Config
