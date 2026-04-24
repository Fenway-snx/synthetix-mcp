package types

// Post-debit subaccount token-bucket state for the getRateLimits action,
// produced only after a successful CheckOrderLimit on the authenticated
// subaccount. A nil snapshot on TradeContext means the gateway did not apply
// order rate limiting; the handler should then report zero caps.
type GetRateLimitsSubaccountSnapshot struct {
	// Remaining tokens after the gateway debited this request.
	AvailableTokens int
	// Maximum tokens per rate-limit window for this subaccount (capacity).
	Limit int
}

// Reports consumed and maximum requests for the getRateLimits JSON body
// (requestsUsed and requestsCap). A nil receiver yields zeroes.
func (s *GetRateLimitsSubaccountSnapshot) PublicCounts() (requestsUsed, requestsCap int) {
	if s == nil {
		return 0, 0
	}

	used := s.Limit - s.AvailableTokens
	if used < 0 {
		used = 0
	}

	return used, s.Limit
}
