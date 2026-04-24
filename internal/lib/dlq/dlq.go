// Defines the Dead Letter Queue (DLQ) API, a system-wide facility that
// provides "call a human" functionality for conditions that exhaust all
// mechanical contingency measures.
//
// The major exported constructs in this package are:
//
//   - [DeadLetterQueue] interface, through which callers post letters for
//     best-effort delivery;
//   - [DeadLetterDeliverer] interface (Strategy pattern), which transports
//     a prepared envelope+letter to its destination;
//   - [DeadLetterQueueProvider] interface, used by execution contexts to
//     expose the process-wide DLQ;
//   - [Envelope], a params-structure whose fields provide context for a
//     particular post -- caller-supplied fields (Application, System,
//     Subsystem) are merged with automatically-captured runtime diagnostics
//     (hostname, IP, PID, caller location, build commit, goroutine count);
//   - [NewHandler], the primary constructor, assembles a [DeadLetterQueue]
//     from a [DeadLetterDeliverer] and an optional default [Envelope];
//   - [NewDefaultEnvelopeDecorator] merges a fixed base envelope into each
//     post;
//   - [NewDeadLetterQueueProvider], which wraps a [DeadLetterQueue] as a
//     [DeadLetterQueueProvider];
//
// Concrete deliverer implementations live in subpackages to avoid imposing
// their dependencies on callers of this package:
//
//   - lib/runtime/dlq -- StderrDeliverer
//   - lib/runtime/dlq/nats -- NATSDeliverer (JetStream, with optional
//     fallback deliverer)
//
// Test doubles for both [DeadLetterQueue] and [DeadLetterDeliverer] (spies
// and stubs) are in lib/dlq/doubles.
package dlq
