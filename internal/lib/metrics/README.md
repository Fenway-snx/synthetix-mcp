# lib/metrics <!-- omit in toc -->

This package contains general purpose **metrics** helper functions for use throughout the codebase.

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


Third-party:

* **github.com/prometheus/client_golang/prometheus/promhttp**;
* **github.com/spf13/viper**;


Standard:

* **context**;
* **fmt**;
* **net/http**;
* **net/http/pprof**;
* **runtime**;
* **strings**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

*None*
