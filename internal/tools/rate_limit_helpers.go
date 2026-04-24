package tools

import (
	snx_lib_api_ratelimiting "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/rate_limiting"
)

func CopyTokenCosts(costs snx_lib_api_ratelimiting.HandlerTokenCosts) map[string]int {
	if len(costs) == 0 {
		return map[string]int{}
	}

	out := make(map[string]int, len(costs))
	for action, cost := range costs {
		out[string(action)] = cost
	}
	return out
}
