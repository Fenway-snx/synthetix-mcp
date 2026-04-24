package constants

import "time"

// Diagnostics constants
const (
	HeartbeatInterval           = 60 * time.Second
	HeartbeatIntervalInitialMax = 10 * time.Second
	HeartbeatIntervalInitialMin = 1 * time.Second
)
