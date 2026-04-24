package dlq

// Exposes dead letter queue functionality
type DeadLetterQueue interface {
	// Receives a letter of arbitrary type, along with details of the sender
	// in the form of a (partial) envelope, for posting by the DLQ underlying
	// mechanism.
	//
	// The letter can be of any type, but is interpreted preferentially
	// according to the following:
	// 1. can be marshaled successfully by the standard `json.Marshal()`
	//    function;
	// 2. can be best-effort converted to JSON by the utility
	//   `SafeMarshalJSON()`. In this case, the marshalled envelope will
	//   contain `"json_conversion_was_incomplete":true`;
	//
	// Returns:
	// `nil` if delivery has been successfully attempted (NOTE: it is not
	//  guaranteed); otherwise, contains information about what may have
	// happened that prevented or limited delivery. It is normal practice to
	// ignore the return value because best-effort attempts will have been
	// made, and DLQ is the "reporter of last resort".
	Post(letter any, envelope Envelope) (err error)
}

// Defines the responsibilities of a deliverer, which is to take the given
// envelope and make an absolute-best-effort to deliver its
// envelopeGenericForm to the requisite destination.
type DeadLetterDeliverer interface {
	OnPost(envelope Envelope, envelopeJSONString string) error
}

// An interface that provides a [DeadLetterQueue], usually used as a
// convenience method on an execution context.
type DeadLetterQueueProvider interface {
	// Obtain the provider's [DeadLetterQueue] references, which will NEVER be
	// `nil`.
	DLQ() DeadLetterQueue
}
