# lib/api/types <!-- omit in toc -->

This package contains general purpose **API types** and helper functions for use throughout the codebase.

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

* **lib/api/validation/utils**;
* **lib/core**;
* **lib/message/grpc**;
* **lib/utils/string**;
* **lib/utils/time**;


Third-party:

* **github.com/shopspring/decimal**;
* **github.com/synesissoftware/ANGoLS/strings**;
* **google.golang.org/protobuf/types/known/timestamppb**;


Standard:

* **encoding/json**;
* **errors**;
* **fmt**;
* **math**;
* **strconv**;
* **strings**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/constants**;
* **lib/api/handlers/info**;
* **lib/api/handlers/trade**;
* **lib/api/handlers/types**;
* **lib/api/handlers/utils**;
* **lib/api/json**;
* **lib/api/rate_limiting**;
* **lib/api/validation**;
* **lib/api/whitelist**;
* **lib/auth**;
