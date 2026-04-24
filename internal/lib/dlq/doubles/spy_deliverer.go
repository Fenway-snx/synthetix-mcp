package doubles

// Entries recorded by [SpyDeliverer].
type SpyDelivererEntry struct {
	Envelope           Envelope
	EnvelopeJSONString string
}

// Test double that provides Spy on [DeadLetterDeliverer], in the form
// of a sequence of deliverer entries. An optional callback can be
// supplied via [SpyDeliverer.WithOnPost] to control the return value;
// by default, OnPost returns nil.
type SpyDeliverer struct {
	onPost  func(envelope Envelope, envelopeJSONString string) error
	Entries []SpyDelivererEntry
}

var _ DeadLetterDeliverer = (*SpyDeliverer)(nil)

func (sd *SpyDeliverer) OnPost(envelope Envelope, envelopeJSONString string) error {
	sd.Entries = append(sd.Entries, SpyDelivererEntry{
		Envelope:           envelope,
		EnvelopeJSONString: envelopeJSONString,
	})

	if sd.onPost != nil {
		return sd.onPost(envelope, envelopeJSONString)
	}

	return nil
}

// Sets a callback to be invoked on each OnPost call, allowing tests
// to control the error return. Returns the receiver for chaining.
func (sd *SpyDeliverer) WithOnPost(
	fn func(envelope Envelope, envelopeJSONString string) error,
) *SpyDeliverer {
	sd.onPost = fn

	return sd
}

// Creates a new instance of [SpyDeliverer].
func NewSpyDeliverer() *SpyDeliverer {
	return &SpyDeliverer{}
}
