# lib/runtime/health <!-- omit in toc -->

This package contains general purpose **runtime health** helper functions for use throughout the codebase.

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
* **lib/runtime/health/handlers**;
* **lib/runtime/health/types**;


Third-party:

* **github.com/spf13/viper**;


Standard:

* **context**;
* **fmt**;
* **net**;
* **net/http**;
* **strings**;


#### Efferent Dependencies (aka "Fan-out")

None (this package is only used by services, not by other `lib` packages);
