package ratelimiting

// Computes how many logical units to bill for rate limiting when scaling token
// costs by batch size. For the placeOrders action, uses the length of the raw
// orders array in params when it is present and non-empty; otherwise returns 1.
// All other actions return 1.
func OrderBatchSize(
	action RequestAction,
	params map[string]any,
) int {
	if action == "placeOrders" {
		orders, ok := params["orders"].([]any)
		if !ok || len(orders) == 0 {
			return 1
		}

		return len(orders)
	}

	return 1
}
