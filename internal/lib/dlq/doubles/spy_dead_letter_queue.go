package doubles

// Entries to be recorded by [SpyDeadLetterQueue].
type SpyDeadLetterQueueEntry struct {
	Envelope Envelope
	Letter   any
}

// Test double that provides Spy on [DeadLetterQueue], in the form of a
// sequence of queue entries. An optional callback can be supplied via
// [SpyDeadLetterQueue.WithOnPost] to control the return value; by
// default, Post returns nil.
type SpyDeadLetterQueue struct {
	onPost  func(letter any, envelope Envelope) error
	Entries []SpyDeadLetterQueueEntry
}

var _ DeadLetterQueue = (*SpyDeadLetterQueue)(nil)

func (dlq *SpyDeadLetterQueue) Post(letter any, envelope Envelope) (err error) {
	dlq.Entries = append(dlq.Entries, SpyDeadLetterQueueEntry{
		Envelope: envelope,
		Letter:   letter,
	})

	if dlq.onPost != nil {
		return dlq.onPost(letter, envelope)
	}

	return
}

// Sets a callback to be invoked on each Post call, allowing tests to
// control the error return. Returns the receiver for chaining.
func (dlq *SpyDeadLetterQueue) WithOnPost(
	fn func(letter any, envelope Envelope) error,
) *SpyDeadLetterQueue {
	dlq.onPost = fn

	return dlq
}

// Creates a new instance of [SpyDeadLetterQueue].
func NewSpyDeadLetterQueue() (dlq *SpyDeadLetterQueue, err error) {

	dlq = &SpyDeadLetterQueue{}

	return
}
