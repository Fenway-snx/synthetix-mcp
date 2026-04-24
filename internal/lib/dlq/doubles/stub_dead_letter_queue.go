package doubles

type stubDeadLetterQueue struct {
}

var _ DeadLetterQueue = (*stubDeadLetterQueue)(nil)

func (dlq *stubDeadLetterQueue) Post(letter any, envelope Envelope) error {
	return nil
}

// Creates a new instance of a stub implementation of [DeadLetterQueue].
func NewStubDeadLetterQueue() *stubDeadLetterQueue {
	return &stubDeadLetterQueue{}
}
