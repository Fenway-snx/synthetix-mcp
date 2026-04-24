# lib/api/validation <!-- omit in toc -->

This package contains general purpose **API validation** helper functions for use throughout the codebase.

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

* **lib/api/constants**;
* **lib/api/json**;
* **lib/api/types**;
* **lib/api/validation/utils**;
* **lib/core**;
* **lib/message/grpc**;
* **lib/utils/time**;


Third-party:

* **github.com/ethereum/go-ethereum/common**;
* **github.com/go-viper/mapstructure/v2**;
* **github.com/shopspring/decimal**;
* **github.com/synesissoftware/ANGoLS/slices**;


Standard:

* **context**;
* **errors**;
* **fmt**;
* **regexp**;
* **strconv**;
* **strings**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/handlers/trade**;
* **lib/auth**;
