# NATS README <!-- omit in toc -->

Information about the package **lib/db/nats**.


# Table of contents <!-- omit in toc -->

- [Introduction](#introduction)
- [Streams and Subjects](#streams-and-subjects)
	- [Streams](#streams)
	- [Subject naming convention](#subject-naming-convention)

# Introduction

T.B.C.



# Streams and Subjects

## Streams

The former monolithic `ACCOUNTS` stream has been split into four
domain-specific streams.  Each stream owns a wildcard subject of the form
`js.{stream-name}.>` (after the global prefix).

| Stream | Purpose | Retention | Storage |
| --- | --- | --- | --- |
| `EXECUTION_EVENTS` | Orders, trades, positions, auto-exchange, TP/SL, account state | WorkQueue | File |
| `LIQUIDATION_EVENTS` | Insurance protection and liquidation-related audit events | Limits | File |
| `ACCOUNT_LIFECYCLE` | Deposits, withdrawals, status, fee-rate updates, CoW orders, commands | Limits | File |
| `FUNDING_EVENTS` | Funding-rate posts and balance settlement | Limits | File |

Other pre-existing streams (`ORDERS`, `ACCOUNTS_TREASURY`, `RELAYER_TXN_QUEUE`,
etc.) are unchanged.

## Subject naming convention

JetStream subjects follow the pattern:

```
js.{stream}[.{group}...].{designator}
```

- **stream** – kebab-case stream name (e.g. `execution-events`, `account-lifecycle`, `liquidation-events`, `funding-events`)
- **group** – optional logical grouping within the stream; can be multiple levels deep (e.g. `order`, `position`, `withdrawal`)
- **designator** – specific event (e.g. `history`, `atomic-update`, `received`)

Non-JetStream subjects (core NATS pub/sub) keep their original naming and are
not prefixed with `js.`.

See `topics.go` for the authoritative list of subject constants.
