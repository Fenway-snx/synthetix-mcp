package dlq

import (
	"fmt"
	"os"
)

// A simple deliverer that writes envelopes to the standard error stream. An
// optional prefix can be set via [StderrDelivererOption] to distinguish DLQ
// output from other stderr traffic.
type stderrDeliverer struct {
	prefix string
}

var _ DeadLetterDeliverer = (*stderrDeliverer)(nil)

func (sd stderrDeliverer) OnPost(envelope Envelope, envelopeJSONString string) error {
	if sd.prefix != "" {
		fmt.Fprintf(os.Stderr, "%s: %s\n", sd.prefix, envelopeJSONString)
	} else {
		fmt.Fprintln(os.Stderr, envelopeJSONString)
	}

	return nil
}

// Functional option for [NewStderrDeliverer].
type StderrDelivererOption func(*stderrDeliverer)

// Sets a prefix string that is prepended to every line written to standard
// error stream (separated by `": "`). Useful for distinguishing DLQ output
// from other standard error stream traffic.
func WithPrefix(prefix string) StderrDelivererOption {
	return func(sd *stderrDeliverer) {
		sd.prefix = prefix
	}
}

// Creates a new instance of a type that implements [DeadLetterDeliverer] by
// writing the envelope/letter to the standard error stream.
func NewStderrDeliverer(opts ...StderrDelivererOption) (DeadLetterDeliverer, error) {
	sd := &stderrDeliverer{}

	for _, opt := range opts {
		opt(sd)
	}

	return sd, nil
}
