# lib/utils/decimal <!-- omit in toc -->

This package contains general purpose **decimal** helper functions for use throughout the codebase.

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

None;


Third-party:

* **github.com/shopspring/decimal**;


Standard:

* **errors**;
* **fmt**;
* **math**;
* **sort**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/db/postgres/types**;
