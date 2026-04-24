# lib/api/whitelist <!-- omit in toc -->

This package provides **wallet whitelist arbitration** for REST and WebSocket API services, determining whether a given wallet address is permitted to place orders.

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

* **lib/api/types**;
* **lib/db/redis**;
* **lib/logging**;


Third-party:

*None*


Standard:

* **context**;
* **encoding/json**;
* **errors**;
* **strings**;
* **sync/atomic**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/handlers/info**;
* **lib/api/handlers/types**;
