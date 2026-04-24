# lib/api/rate_limiting <!-- omit in toc -->

This package contains general purpose **API rate limiting** helper functions for use throughout the codebase.

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

* **lib/config**;
* **lib/core**;
* **lib/utils/time**;


Third-party:

* **golang.org/x/time/rate**;
* **github.com/spf13/viper**;


Standard:

* **context**;
* **errors**;
* **fmt**;
* **math**;
* **strconv**;
* **strings**;
* **sync**;
* **sync/atomic**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

* **lib/api/handlers/types**;
