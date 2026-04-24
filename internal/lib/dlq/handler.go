package dlq

import (
	"context"
	"fmt"
	"os"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Process-invariant fields captured once at construction time and
// stamped onto every envelope, avoiding repeated syscalls and
// lookups on the hot path.
type cachedInvariants struct {
	byteOrder   string
	commit      string
	hostName    string
	ipAddresses []string
	pId         int
	processName string
}

// Context for a DLQ instance.
type dlqHandler struct {
	logger    snx_lib_logging.Logger
	ctx       context.Context
	deliverer DeadLetterDeliverer
	cached    cachedInvariants
}

var _ DeadLetterQueue = (*dlqHandler)(nil)

// This is the _actual_ place in the DLQ ecosystem where all the actual
// letter/envelope handling takes place, according to the following
// algorithm:
//   - if the handler's context has been cancelled (e.g. service
//     shutdown), Post returns the context error immediately;
//   - the envelope is prepared, including enhancing (a copy of) it with
//     attributes such as the commit and the file+line+function of the
//     call-site;
//   - the letter is marshalled into the envelope as JSON, first via
//     standard json.Marshal and, if that fails, via best-effort
//     `SafeMarshalJSON()` (which silently omits unrepresentable fields);
//   - the fully-prepared envelope (as a JSON string) is handed to the
//     deliverer for absolute-best-effort delivery to its destination;
func (h *dlqHandler) Post(letter any, envelope Envelope) (err error) {
	// TODO: maybe we should write to stderr if ctx done ??
	if err = h.ctx.Err(); err != nil {
		return
	}

	envelope.affix(1, &h.cached)

	var envelopeJSONString string

	envelopeJSONString, err = envelope.prepare(letter)
	if err != nil {
		h.logger.Error("CRITICAL: DLQ failed to prepare envelope",
			"application", envelope.Application,
			"error", err,
			"letter_type", fmt.Sprintf("%T", letter),
			"note", "letter contents written to host standard error stream to avoid transmitting sensitive information in logs",
			"subsystem", envelope.Subsystem,
		)

		// Write the full payload to stderr for local-only inspection;
		// it is not shipped to external log aggregation.
		fmt.Fprintf(os.Stderr, "DLQ PREPARE FAILURE: letter=%v\n", letter)

		return err
	}

	err = h.deliverer.OnPost(envelope, envelopeJSONString)

	return
}

// Creates a new DLQ Handler from the given deliverer and with a default
// envelope.
//
// Parameters:
//   - logger - Logger of last resort. May not be `nil`;
//   - ctx - Service-lifetime context for shutdown propagation. May not be
//     `nil`;
//   - deliverer - Deliverer. May not be `nil`;
//   - defaultEnvelope - An authoritative [Envelope] whose non-empty fields
//     are stamped onto every posted envelope, overriding any per-call
//     values for those fields. Use this for service-level identity (e.g.
//     Application) that call sites must not vary;
//
// Preconditions (checked at runtime; typed nils are not detected):
//   - `logger != nil`;
//   - `ctx != nil`;
//   - `deliverer != nil`;
func NewDLQHandler(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	deliverer DeadLetterDeliverer,
	defaultEnvelope Envelope,
) (
	dlq DeadLetterQueue,
	err error,
) {
	// precondition enforcement(s)

	if logger == nil {
		panic("VIOLATION: parameter `logger` may not be `nil`")
	}
	if ctx == nil {
		panic("VIOLATION: parameter `ctx` may not be `nil`")
	}
	if deliverer == nil {
		panic("VIOLATION: parameter `deliverer` may not be `nil`")
	}

	dlq = &dlqHandler{
		logger:    logger,
		ctx:       ctx,
		deliverer: deliverer,
		cached:    computeInvariants(),
	}

	if !defaultEnvelope.isDefault() {
		// the default envelope is providing customisation, so we wrap it in a
		// `defaultEnvelopeDecorator` instance

		dlq = NewDefaultEnvelopeDecorator(dlq, defaultEnvelope)
	}

	return
}
