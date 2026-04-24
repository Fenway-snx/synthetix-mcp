# NATSDeliverer <!-- omit in toc -->

A `DeadLetterDeliverer` implementation that publishes DLQ envelopes to a dedicated NATS JetStream stream.

## Table of contents <!-- omit in toc -->

- [Why a separate package?](#why-a-separate-package)
- [NATS infrastructure](#nats-infrastructure)
- [Constructor](#constructor)
- [Fallback behaviour](#fallback-behaviour)
- [Error semantics](#error-semantics)
- [Usage](#usage)


## Why a separate package?

The `NATSDeliverer` is in a separate subpackage (`lib/runtime/dlq/nats`) rather than alongside `StderrDeliverer` in `lib/runtime/dlq`. This is intentional: importing `lib/runtime/dlq/nats` transitively imports `lib/db/nats`, whose `init()` function requires the `SNX_NATS_SUBJECTS_PREFIX` environment variable (or an explicit opt-in to the default prefix). Placing the NATS deliverer in its own package ensures that consumers who only need `StderrDeliverer` — such as standalone example programs or unit tests — are not forced to satisfy NATS configuration requirements.


## NATS infrastructure

The deliverer uses the following NATS JetStream infrastructure, defined in `lib/db/nats`:

| Concept | Value | Defined in |
|---------|-------|------------|
| Subject | `system.dlq.posted` (prefixed) | `lib/db/nats/topics.go` — `SystemDLQPosted` |
| Stream  | `DLQ` (prefixed) | `lib/db/nats/stream.go` — `StreamName_DLQ` |
| Storage | `FileStorage` | Durable; survives NATS restarts |
| Retention | `LimitsRetention` | Entries remain until age/size limits are reached |
| Max age | 7 days | Substantially longer than operational streams |
| Max messages | 100,000 | `DefaultMaxMsgs` |
| Max bytes | 100 MiB | `DefaultMaxBytes` |

The stream is created or updated idempotently via `CreateOrUpdateStream` in the constructor, which is safe when multiple services start concurrently.


## Constructor

```Go
func NewNATSDeliverer(
    logger            snx_lib_logging.Logger,
    ctx               context.Context,
    js                jetstream.JetStream,
    replicationFactor int,
    fallback          snx_lib_dlq.DeadLetterDeliverer,
) (snx_lib_dlq.DeadLetterDeliverer, error)
```

**Parameters:**

- `logger` — Required (panics if `nil`).
- `ctx` — Service-lifetime context for shutdown propagation. Required (panics if `nil`).
- `js` — A JetStream instance. Required (panics if `nil`).
- `replicationFactor` — Replication factor for the DLQ stream.
- `fallback` — An optional `DeadLetterDeliverer` to invoke when a NATS publish fails. Pass `nil` for no fallback.

**Behaviour:**

1. Validates preconditions (panics on nil `logger`, `ctx`, or `js`; typed nils are not detected).
2. Calls `CreateOrUpdateStream` with the DLQ stream config (10-second timeout).
3. Returns the deliverer on success, or an error if stream creation fails.


## Fallback behaviour

When `OnPost` fails to publish to NATS JetStream, three outcomes are possible depending on the fallback configuration:

| Scenario | Fallback | Error returned |
|----------|----------|----------------|
| NATS publish succeeds | Not invoked | `nil` |
| NATS fails, fallback succeeds | Invoked, succeeds | `"failed to publish dead letter to NATS: ...; published dead letter to fallback"` |
| NATS fails, fallback fails | Invoked, fails | `"failed to publish dead letter to NATS: ...; failed to publish dead letter to fallback: ..."` |
| NATS fails, no fallback | N/A | `"failed to publish dead letter to NATS: ..."` |

In all failure cases, the error is logged before being returned. The fallback is a best-effort safety net; the returned error always reflects the NATS failure regardless of whether the fallback succeeded.

A typical production configuration chains `NATSDeliverer` with a `StderrDeliverer` fallback, ensuring that dead letters reach at least stderr if NATS is unavailable:

```Go
stderrFallback, _ := snx_lib_runtime_dlq.NewStderrDeliverer()

deliverer, err := snx_lib_runtime_dlq_nats.NewNATSDeliverer(
    logger,
    ctx,
    js,
    replicationFactor,
    stderrFallback,
)
```


## Error semantics

`NewNATSDeliverer` **panics** on programming errors (nil `logger`, nil `ctx`, nil `js`) and **returns errors** on infrastructure failures (stream creation). This follows the convention that invalid wiring is a fatal startup bug, while infrastructure unavailability is a recoverable condition.

`OnPost` always returns an error when the NATS publish fails, even if the fallback succeeded. This is because callers (typically `handler.Post`) may wish to log or propagate the failure. In practice, the DLQ return value is usually discarded because the DLQ is the reporter of last resort.


## Usage

Typically wired at service startup alongside the DLQ handler:

```Go
// 1. Create the fallback deliverer
stderrFallback, _ := snx_lib_runtime_dlq.NewStderrDeliverer()

// 2. Create the NATS deliverer with fallback
deliverer, err := snx_lib_runtime_dlq_nats.NewNATSDeliverer(
    logger,
    ctx,
    js,
    replicationFactor,
    stderrFallback,
)
if err != nil {
    // handle error
}

// 3. Create the DLQ handler with service-level defaults
dlq, _ := snx_lib_dlq.NewDLQHandler(
    logger,
    ctx,
    deliverer,
    snx_lib_dlq.Envelope{
        Application: "trading",
        System:      "order-processing",
    },
)

// 4. Use at call sites — only per-call context is needed
dlq.Post(failedRequest, snx_lib_dlq.Envelope{
    Error:     err.Error(),
    Subsystem: "order-fill",
})
```

Each message published to the NATS DLQ stream is a JSON-encoded `Envelope` containing both the caller-supplied context and auto-captured runtime diagnostics. See `lib/dlq/README.md` for the full envelope structure and JSON format.
