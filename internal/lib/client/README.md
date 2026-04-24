# lib/client <!-- omit in toc -->

This package contains **client-facing** vocabulary, labels, and small helpers
that REST, WebSocket, and internal publishers share **without** importing the
full HTTP stack (`lib/api` handlers, Echo, or request validation). Domain enums
and rules remain in **lib/core**; the public HTTP/JSON contract remains in
**lib/api**.

## Table of Contents <!-- omit in toc -->

- [Layout](#layout)
- [Rules](#rules)
- [Dependencies](#dependencies)
	- [Synthetix lib Dependencies](#synthetix-lib-dependencies)
		- [Afferent Dependencies (aka "Fan-in")](#afferent-dependencies-aka-fan-in)
		- [Efferent Dependencies (aka "Fan-out")](#efferent-dependencies-aka-fan-out)

## Layout

| Path | Role |
|------|------|
| [`concepts/`](concepts/) | Named ideas (e.g. trade direction) with stable strings or mappings from `core` types. |

Add subpackages under `lib/client` when a concern is clearly **client/shared**
and should not live in `lib/core` (presentation) or `lib/api` (transport).

## Rules

- **`lib/client/...` must not import `github.com/Synthetixio/v4-offchain/lib/api/...`.**
  Otherwise services that avoid `lib/api` would pull it in again indirectly.

- **`lib/core` should not import `lib/client`.**
  Core keeps domain meaning; client packages interpret it for outward-facing
  copy or wire-adjacent strings.

- **`lib/api` may import `lib/client`** when a type or helper should delegate to
  one canonical implementation (optional; avoid thin pass-through wrappers unless
  they aid discovery).

- **Non-gateway services** (everything except `services/api` and
  `services/websocket` per `AGENTS.md`) **must not import `lib/api`**. They may
  import **`lib/client`** when they need the same labels or tokens clients see.

## Dependencies

### Synthetix lib Dependencies

Dependencies within the **lib** superordinate package and its subordinate packages. (NOTE: test dependencies are not recorded.)

#### Afferent Dependencies (aka "Fan-in")

SNX lib:

* **lib/core**;


Third-party:

*None*


Standard:

*None*


#### Efferent Dependencies (aka "Fan-out")

* **services/subaccount/internal/utils**;
* **services/websocket/internal/handlers**;

