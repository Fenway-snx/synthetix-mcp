package ratelimiting

import "errors"

var (
	errInvalidKeyValuePair             = errors.New("invalid key/value pair")
	errRateLimitsMayNotBeNagative      = errors.New("rate limits may not be negative")
	errRateLimitDurationMustBePositive = errors.New("duration must be positive")
	errUnrecognisedType                = errors.New("unrecognised type")
)
