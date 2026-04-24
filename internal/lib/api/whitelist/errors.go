package whitelist

import "errors"

var (
	errRedisClientRequired = errors.New("redis client is required")
)
