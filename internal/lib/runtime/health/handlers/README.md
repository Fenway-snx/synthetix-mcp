# lib/runtime/health/handlers <!-- omit in toc -->

This package contains general purpose **runtime health handlers** helper functions for use throughout the codebase.

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

* **lib/logging**;
* **lib/net/http**;
* **lib/runtime/health/types**;


Third-party:

*None*


Standard:

* **encoding/json**;
* **fmt**;
* **net/http**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/runtime/health**;
