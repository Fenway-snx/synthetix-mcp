package doubles

type stubDeliverer struct {
}

var _ DeadLetterDeliverer = (*stubDeliverer)(nil)

func (d *stubDeliverer) OnPost(envelope Envelope, envelopeJSONString string) error {
	return nil
}

// Creates a new instance of a stub implementation of [DeadLetterDeliverer].
func NewStubDeliverer() *stubDeliverer {
	return &stubDeliverer{}
}
