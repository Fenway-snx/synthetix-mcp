package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	AckWaitDefault     = 30 * time.Second
	DiscardNew         = jetstream.DiscardNew
	FileStorage        = jetstream.FileStorage
	LimitsRetention    = jetstream.LimitsPolicy
	MaxAgeDefault      = 24 * time.Hour
	MemoryStorage      = jetstream.MemoryStorage
	Unlimited          = -1
	WorkQueueRetention = jetstream.WorkQueuePolicy
)
