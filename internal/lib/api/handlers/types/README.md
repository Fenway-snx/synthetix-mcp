# lib/api/handlers/types <!-- omit in toc -->

This package contains general purpose **API handlers types** helper functions for use throughout the codebase.

## Table of Contents <!-- omit in toc -->

- [Dependencies](#dependencies)
	- [Synthetix lib Dependencies](#synthetix-lib-dependencies)
		- [Afferent Dependencies (aka "Fan-in")](#afferent-dependencies-aka-fan-in)
		- [Efferent Dependencies (aka "Fan-out")](#efferent-dependencies-aka-fan-out)

## Dependencies

### Synthetix lib Dependencies

Dependencies within the **lib** superordinate package and its subordinate packages. (NOTE: test dependencies are not recorded.)

#### Afferent Dependencies (aka "Fan-in")

SNX lib:

* **lib/api/json**;
* **lib/api/rate_limiting**;
* **lib/api/types**;
* **lib/api/whitelist**;
* **lib/auth**;
* **lib/core**;
* **lib/db/redis**;
* **lib/logging**;
* **lib/message/grpc**;
* **lib/net/http**;


Third-party:

* **github.com/nats-io/nats.go**;
* **github.com/nats-io/nats.go/jetstream**;
* **github.com/shopspring/decimal**;


Standard:

* **context**;
* **fmt**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/handlers/info**;
* **lib/api/handlers/trade**;
* **lib/api/handlers/utils**;
