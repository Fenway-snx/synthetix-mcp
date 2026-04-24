# lib/utils/time <!-- omit in toc -->

This package contains general purpose **time** helper functions for use throughout the codebase.

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

None;


Standard:

* **rand**;
* **slices**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/handlers/info**;
* **lib/api/handlers/trade**;
* **lib/api/handlers/utils**;
* **lib/api/json**;
* **lib/api/rate_limiting**;
* **lib/api/types**;
* **lib/api/validation**;
* **lib/auth**;
* **lib/core**;
* **lib/db/postgres**;
* **lib/diagnostics**;

