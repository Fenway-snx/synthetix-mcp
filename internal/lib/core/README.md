# lib/core <!-- omit in toc -->

This package contains general purpose **core** helper functions for use throughout the codebase.

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

* **lib/db/postgres/types**;
* **lib/message/grpc**;
* **lib/utils/string**;
* **lib/utils/time**;


Third-party:

* **github.com/ethereum/go-ethereum/common**;
* **github.com/shopspring/decimal**;
* **github.com/synesissoftware/Diagnosticism.Go**;


Standard:

* **encoding/json**;
* **errors**;
* **fmt**;
* **slices**;
* **sort**;
* **strconv**;
* **strings**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/handlers/info**;
* **lib/api/handlers/trade**;
* **lib/api/handlers/types**;
* **lib/api/json**;
* **lib/api/rate_limiting**;
* **lib/api/types**;
* **lib/api/validation**;
* **lib/auth**;
* **lib/db/nats**;
* **lib/db/repository**;
* **lib/nats**;
* **lib/utils**;
