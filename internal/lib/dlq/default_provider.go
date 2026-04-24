package dlq

type deadLetterQueueProvider struct {
	dlq DeadLetterQueue
}

var _ DeadLetterQueueProvider = (*deadLetterQueueProvider)(nil)

func (p *deadLetterQueueProvider) DLQ() DeadLetterQueue {
	return p.dlq
}

// Creates an instance of a type supporting [DeadLetterQueueProvider] from a
// [DeadLetterQueue] instance.
//
// Preconditions (checked at runtime; typed nils are not detected):
//   - `dlq != nil`
func NewDeadLetterQueueProvider(dlq DeadLetterQueue) DeadLetterQueueProvider {
	// precondition enforcement(s)

	if dlq == nil {
		panic("VIOLATION: parameter `dlq` may not be `nil`")
	}

	return &deadLetterQueueProvider{
		dlq: dlq,
	}
}
