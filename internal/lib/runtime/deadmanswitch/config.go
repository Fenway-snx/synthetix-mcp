package deadmanswitch

import (
	"errors"
	"os"
	"strconv"
)

const (
	defaultMaxTimeoutSeconds int64 = 86400
	defaultMinTimeoutSeconds int64 = 5
)

var (
	errDeadManSwitchMaxTimeoutMustBeGreaterThanOrEqualToMinTimeout = errors.New("dead-man-switch max timeout must be greater than or equal to min timeout")
	errDeadManSwitchMaxTimeoutMustBeValidInteger                   = errors.New("dead-man-switch max timeout must be a valid integer")
	errDeadManSwitchMinTimeoutMustBeGreaterThan0                   = errors.New("dead-man-switch min timeout must be greater than 0")
	errDeadManSwitchMinTimeoutMustBeValidInteger                   = errors.New("dead-man-switch min timeout must be a valid integer")
)

// Configured dead-man-switch timeout bounds in seconds.
type TimeoutBounds struct {
	MaxTimeoutSeconds int64
	MinTimeoutSeconds int64
}

// Loads timeout bounds with defaults for local and test workflows.
func LoadTimeoutBounds() (TimeoutBounds, error) {
	bounds := TimeoutBounds{
		MaxTimeoutSeconds: defaultMaxTimeoutSeconds,
		MinTimeoutSeconds: defaultMinTimeoutSeconds,
	}

	if value := os.Getenv("SNX_DEAD_MAN_SWITCH_MAX_TIMEOUT_SECONDS"); value != "" {
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return TimeoutBounds{}, errDeadManSwitchMaxTimeoutMustBeValidInteger
		}
		bounds.MaxTimeoutSeconds = parsedValue
	}

	if value := os.Getenv("SNX_DEAD_MAN_SWITCH_MIN_TIMEOUT_SECONDS"); value != "" {
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return TimeoutBounds{}, errDeadManSwitchMinTimeoutMustBeValidInteger
		}
		bounds.MinTimeoutSeconds = parsedValue
	}

	if bounds.MinTimeoutSeconds <= 0 {
		return TimeoutBounds{}, errDeadManSwitchMinTimeoutMustBeGreaterThan0
	}
	if bounds.MaxTimeoutSeconds < bounds.MinTimeoutSeconds {
		return TimeoutBounds{}, errDeadManSwitchMaxTimeoutMustBeGreaterThanOrEqualToMinTimeout
	}

	return bounds, nil
}
