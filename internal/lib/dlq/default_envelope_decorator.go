package dlq

// A type that decorates a [DeadLetterQueue] based on a default envelope,
// which can markedly reduce the amount of code required to post letters to
// the DLQ in a given client context.
type defaultEnvelopeDecorator struct {
	dlq             DeadLetterQueue
	defaultEnvelope Envelope
}

var _ DeadLetterQueue = (*defaultEnvelopeDecorator)(nil)

func (ded defaultEnvelopeDecorator) Post(letter any, envelope Envelope) (err error) {
	envelope.affixFileLineFunction(1)

	actualEnvelope := envelope.mergeOver(ded.defaultEnvelope)

	return ded.dlq.Post(letter, actualEnvelope)
}

func NewDefaultEnvelopeDecorator(dlq DeadLetterQueue, defaultEnvelope Envelope) DeadLetterQueue {

	return &defaultEnvelopeDecorator{
		dlq,
		defaultEnvelope,
	}
}
