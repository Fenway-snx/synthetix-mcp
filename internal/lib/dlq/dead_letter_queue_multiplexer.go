package dlq

import "sync"

type dlqMultiplexer struct {
	mu           sync.RWMutex
	implementors []DeadLetterQueue
}

var _ DeadLetterQueue = (*dlqMultiplexer)(nil)

func (m *dlqMultiplexer) Post(letter any, envelope Envelope) error {

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, implementor := range m.implementors {

		// TODO: collect all errors if 1+

		_ = implementor.Post(letter, envelope)
	}

	return nil
}

func NewDeadLetterQueueMultiplexer(
	implementors []DeadLetterQueue,
) DeadLetterQueue {
	return &dlqMultiplexer{
		implementors: implementors,
	}
}
