# Dead Letter Queue (DLQ) <!-- omit in toc -->

The Dead Letter Queue library provides all necessary constructs for implementing the Dead Letter Queue pattern in standalone programs or across our services.

## Table of contents <!-- omit in toc -->

- [Problem Statement](#problem-statement)
	- [Examples](#examples)
		- [Liquidation collateral transfer failure](#liquidation-collateral-transfer-failure)
		- [Deposit collateral lookup failure](#deposit-collateral-lookup-failure)
- [Solution](#solution)
	- [Interfaces](#interfaces)
		- [`DeadLetterQueue` interface](#deadletterqueue-interface)
		- [`DeadLetterDeliverer` interface](#deadletterdeliverer-interface)
		- [`DeadLetterQueueProvider` interface](#deadletterqueueprovider-interface)
	- [Supporting structures and functions](#supporting-structures-and-functions)
		- [`Envelope` structure](#envelope-structure)
		- [`NewDLQHandler()` function](#newdlqhandler-function)
		- [Stock Deliverers](#stock-deliverers)
- [Future Enhancements](#future-enhancements)


## Problem Statement

In any long-lived, distributed system there will always be "call the human" moments, conditions in which a chain of mechanical contingency measures ceases to be possible or practicable. For example, if a system that obtains current weather information and writes summary information to a database cannot connect to the database for how long should it attempt to retry connecting before giving up, and what contingency action (if any) should be taken.

In more sophisticated and important systems, it can be the case that retries are not possible, or an attempt to do so may be deemed too expensive (timewise). Furthermore, it can be that the loss of information could render the organisation liable to losses (legal or otherwise). Hence, a contingency mechanism suitable for such circumstances is warranted. This is called a Dead Letter Queue, which is a subsystem that provides the means to absorb (and transport appropriately) messages describing in detail a failure condition for future consumption by a separate system (or a human).

In the case of a trading exchange, there are many circumstances where a DLQ is warranted. There are many communications channels within and between services that have limits, such as the maximum message size, or the maximum number of in-flight messages, and so forth. It is impractical (if not impossible) to design an efficient system that a-priori guarantees that such limits may never be breached. And to do so may represent an extreme cost for the potential unlikely benefits. Such circumstances are well served by the application of the Dead Letter Queue (hereafter DLQ) pattern.

### Examples

Before we cover the details, consider the following two examples drawn from actual wired DLQ call-sites.

#### Liquidation collateral transfer failure

When the `LiquidationEngine` transfers collateral to the SLP (System Liquidation Pool) actor and the cross-actor send fails, the Matching Engine has already committed the liquidation. The in-memory state is now divergent from what should have happened. A DLQ entry preserves the exact transfer parameters for manual or automated financial reconciliation.

```Go
if err := to.Send(context.Background(), transferMessage); err != nil {
	le.logger.Error("failed to place transfer message in slp inbox",
		"error", err,
		"transfer", transferMessage,
	)

	le.ec.DLQ().Post(transferMessage, snx_lib_dlq.Envelope{
		Error:     err.Error(),
		System:    "liquidation",
		Subsystem: "slp_collateral_transfer",
	})

	return fmt.Errorf("failed to send to slp actor: %w", err)
}
```

This would produce a JSON payload such as:

```JSON
{
	"application": "trading",
	"error": "failed to send to slp actor: inbox full",
	"system": "liquidation",
	"subsystem": "slp_collateral_transfer",
	"post_time": "2026-03-08T14:22:07.331812Z",
	"byte_order": "little-endian",
	"commit": "a1b2c3d",
	"goroutine_count": 847,
	"file_line_function": "/app/services/trading/internal/subaccount/liquidation_engine.go:1155:subaccount.(*LiquidationEngine).transferCollateralToSLP",
	"host_name": "trading-7f8b9c6d4-xk2m1",
	"ip_addresses": ["10.0.42.17"],
	"pid": 1,
	"process_name": "trading",
	"letter": {
		"string_form": "{\"from_sub_account_id\":42,\"to_sub_account_id\":1,\"collateral\":\"USDC\",\"quantity\":\"1500.00000000\",\"transferred_at\":\"2026-03-08T14:22:07.328Z\",\"transfer_id\":\"t-90412\"}",
		"json_conversion_was_incomplete": false
	}
}
```

#### Deposit collateral lookup failure

This is a textbook DLQ scenario in the subaccount service. When a deposit event arrives but the collateral cannot be identified, the on-chain funds have already moved. The deposit cannot be silently dropped. A DLQ entry preserves the full event for investigation and reconciliation.

```Go
colleteralId, err := s.collateralRepo.GetCollateralIdByName(ctx, tx, event.Collateral)
if err != nil {
	s.logger.Error("invalid collateral on deposit event",
		"error", err,
		"event", event,
	)
	s.ec.DLQ().Post(event, snx_lib_dlq.Envelope{
		Error:     err.Error(),
		System:    "subaccount",
		Subsystem: "deposit_collateral_lookup",
	})

	return
}
```


## Solution

The solution is built around the **Envelope-Letter** (aka **Handle-Body**) pattern and the **Strategy** pattern. It comprises the following elements:

* interfaces (all in **lib/dlq**):
  * `DeadLetterQueue`;
  * `DeadLetterDeliverer`;
  * `DeadLetterQueueProvider`;
* supporting structures and functions (also **lib/dlq**)
  * most notably `Envelope`; and
  * the handler, exposed by `NewDLQHandler()`;
* implementations of deliverers (in **lib/runtime/dlq**, **lib/runtime/dlq/nats**, ...):
  * `NATSDeliverer`;
  * `StderrDeliverer`;
  * `AsyncFileSystemDelivererDecorator` (not yet done);
  * `NATSRemoteDeliveryConfirmingDeliverer` (not yet done);
* bootstrapping integration via `BootstrapService()` in **lib/service**;

and resides in the following packages:

* **lib/dlq** - package that prescribes the DLQ (client and server) interfaces, related structures, provides several related utility functions;
* **lib/dlq/doubles** - package that provides **lib/dlq** test doubles, such as `SpyDeadLetterQueue` and `StubDeadLetterQueue`;
* **lib/runtime/dlq** - package that defines specific implementations of DLQ interfaces, such as `StderrDeliverer`;
* **lib/runtime/dlq/nats** - package that defines specific implementations of DLQ interfaces, such as `NATSDeliverer`. This is a separate package to avoid imposing `snx_lib_nats` dependency - which conducts an `init()`-time check for NATS subjects environment variable existence - on the other packages;
* **lib/utils/marshal** - package providing `SafeMarshalJSON()`, which follows the semantics of `json.Marshal()` but skips over unmarshalable elements - such as `func` and `chan` - rather than failing, hence making a best-possible-effort to capture all relevant information in the letter;


### Interfaces

#### `DeadLetterQueue` interface

The DLQ mechanism is exposed to application code (on the execution context) in the guise of the `DeadLetterQueue` interface:

```Go
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
```

#### `DeadLetterDeliverer` interface

The DLQ letters are carried by implementors of the `DeadLetterDeliverer` interface:

```Go
// Defines the responsibilities of a deliverer, which is to take the given
// envelope and make an absolute-best-effort to deliver its
// envelopeGenericForm to the requisite destination.
type DeadLetterDeliverer interface {
	OnPost(envelope Envelope, envelopeJSONString string) error
}
```

#### `DeadLetterQueueProvider` interface

The DLQ mechanism is exposed to application code (on the execution context) via the `DeadLetterQueueProvider` interface:

```Go
// An interface that provides a DeadLetterQueue, usually used as a
// convenience method on an execution context.
type DeadLetterQueueProvider interface {
	// Obtain the provider's DeadLetterQueue references, which will NEVER be
	// nil.
	DLQ() DeadLetterQueue
}
```

### Supporting structures and functions

#### `Envelope` structure

This structure has the public interface as follows:

```Go
type Envelope struct {
	Application string    `json:"application"`     // Optional application name for the program or service that posted the letter
	Error       string    `json:"error,omitempty"` // Optional originating error message from the client code that posted the letter
	System      string    `json:"system"`          // Optional system name for the system that posted the letter
	Subsystem   string    `json:"subsystem"`       // Optional subsystem name for the subsystem that posted the letter
	PostTime    time.Time `json:"post_time"`       // Optional event time for the post, which will otherwise be inferred when posting is attempted
}

func (e *Envelope) LetterString() (letterString string, jsonConversionWasIncomplete bool)

func (e *Envelope) ByteOrder() string
func (e *Envelope) Commit() string
func (e *Envelope) FileLineFunction() string
func (e *Envelope) GoroutineCount() int
func (e *Envelope) HostName() string
func (e *Envelope) IPAddresses() []string
func (e *Envelope) PId() int
func (e *Envelope) ProcessName() string
```

The caller provides contextual fields explicitly -- `Application`, `Error`, `System`, `Subsystem` -- while runtime diagnostics are captured automatically by the implementation via methods such as `Commit()`, `FileLineFunction()`, `GoroutineCount()`, and so forth. When an envelope is passed to `DeadLetterQueue#Post()` it is supplemented at that time with runtime information pertaining to the executing context/process/system, and also with any default `Envelope` information provided at DLQ construction time, so only those fields particular to a calling context need to be specified (as seen in the [Examples](#examples) section).

The `Error` field (added to support Item 14 in the DLQ assessment) captures the originating `error.Error()` string from the client code. It uses `omitempty` so that it is absent from the JSON when not provided.


#### `NewDLQHandler()` function

For standalone uses of the DLQ, the `NewDLQHandler()` function is the way to obtain a `DeadLetterQueue` instance from a deliverer:

```Go
// Creates a new DLQ Handler from the given deliverer and with a default
// envelope.
//
// Parameters:
//   - logger - Logger of last resort. May not be nil;
//   - ctx - Service-lifetime context for shutdown propagation. May not be
//     nil;
//   - deliverer - Deliverer. May not be nil;
//   - defaultEnvelope - The default Envelope that may be used to provide
//     common/default attributes, rather than repeating them at #Post()
//     call sites;
//
// Preconditions (checked at runtime; typed nils are not detected):
//   - logger != nil;
//   - ctx != nil;
//   - deliverer != nil;
func NewDLQHandler(
	logger snx_lib_logging.Logger,
	ctx context.Context,
	deliverer DeadLetterDeliverer,
	defaultEnvelope Envelope,
) (
	dlq DeadLetterQueue,
	err error,
)
```

There are several stock deliverers -- described in the next section -- that may be used directly with this function. An example of such direct use is in the **lib/runtime/dlq/nats/examples/nats_dlq_producer** program, as in:

```Go
const (
	applicationName = "nats_dlq_producer"
)

func main() {
	natsURL := nats.DefaultURL

	. . .

	deliverer, err := snx_lib_runtime_dlq_nats.NewNATSDeliverer(
		logger,
		ctx,
		js,
		replicationFactor,
		fallback,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: create NATS deliverer: %v\n", err)
		os.Exit(1)
	}

	dlq, _ := snx_lib_dlq.NewDLQHandler(
		logger,
		ctx,
		deliverer,
		snx_lib_dlq.Envelope{
			Application: applicationName,
		},
	)

	if err := dlq.Post(letter, snx_lib_dlq.Envelope{}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: post returned: %v\n", err)
	}

	. . .
}
```

Such uses are perfectly proper, but it is envisaged that the system services will be provided their DLQ by a common central set up of DLQ logic in the bootstrapping, so will be of no direct concern to the application programmers. (NOTE: it is intended that `AsyncFileSystemDelivererDecorator` and `NATSRemoteDeliveryConfirmingDeliverer` will be combined such that the following high-resilience functionality will be provided: (1) copy of envelope+letter written synchronously to well-known location with precise identification; (2) NATS form sent on JetStream to DLQ Processor System (not yet specified); (3) asynchronous monitoring for ACK from processor system that the envelope+letter has been received, followed by deletion of requisite local file.)

#### Stock Deliverers

Two stock deliverers are provided:

**`StderrDeliverer`** (in **lib/runtime/dlq**) writes the JSON-encoded envelope to the process's standard error stream. It accepts optional functional options at construction time:

```Go
// Creates a new StderrDeliverer.
func NewStderrDeliverer(opts ...StderrDelivererOption) (DeadLetterDeliverer, error)

// Sets a prefix string prepended to every line written (separated by ": ").
func WithPrefix(prefix string) StderrDelivererOption
```

This is the simplest deliverer and is suitable for local development, standalone programs, and as a fallback for the NATS deliverer.

**`NATSDeliverer`** (in **lib/runtime/dlq/nats**) publishes envelopes to a dedicated NATS JetStream stream (`DLQ`). It accepts an optional fallback `DeadLetterDeliverer` that is invoked when the NATS publish fails. See `lib/runtime/dlq/nats/README.md` for full details on the NATS infrastructure, constructor, fallback behaviour, and error semantics.

A typical production configuration chains both:

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


## Future Enhancements

- [ ] Recognise and act on `DeadLetterPostable` interface (which will guide letter handling, incl. what to do with unexported fields);
- [ ] Have `SafeMarshalJSON()` handle unexported fields (in the case that a given instance is of a type that does not have any exported fields with the `json:"xxx"` struct tag);
- [ ] Review low-level design to see if diffusion of logic between `Envelope`, `handler`, and `defaultEnvelopeDecorator` can be simplified;
- [ ] Fix decorator bypassing cached invariants (repeated syscalls on the default-envelope path);
- [ ] Add DLQ metrics (post counts, delivery failures, latency);
- [ ] Add `FlushableDeadLetterQueue` and `FlushableDeadLetterDeliverer` interfaces for graceful shutdown;
- [ ] Switch `makeDLQ` in `BootstrapService` to use NATS deliverer when environment configuration facilities are available;
- [ ] `AsyncFileSystemDelivererDecorator` for local-file safety net;
- [ ] `NATSRemoteDeliveryConfirmingDeliverer` for end-to-end delivery confirmation;
- [ ] DLQ Processor System, as a separate service;
- [ ] Implement the commit derivation;
- [ ] Determine whether `Post()` calls should come before or after the associated log call(s);
