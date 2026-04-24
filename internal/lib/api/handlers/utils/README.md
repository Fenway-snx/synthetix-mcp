# lib/api/handlers/utils <!-- omit in toc -->

This package contains general purpose **API handlers utils** helper functions for use throughout the codebase.

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

* **lib/api/handlers/types**;
* **lib/api/types**;
* **lib/db/repository**;
* **lib/message/grpc**;
* **lib/utils/time**;


Third-party:

* **github.com/shopspring/decimal**;
* **google.golang.org/protobuf/types/known/timestamppb**;


Standard:

* **context**;
* **errors**;
* **fmt**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/handlers/info**;
* **lib/api/handlers/trade**;
