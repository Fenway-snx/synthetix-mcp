# lib/api/handlers/trade <!-- omit in toc -->

This package contains general purpose **API trade handlers** helper functions for use throughout the codebase.

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
* **lib/api/handlers/types**;
* **lib/api/handlers/utils**;
* **lib/api/json**;
* **lib/api/types**;
* **lib/api/validation**;
* **lib/core**;
* **lib/message/grpc**;
* **lib/net/http**;
* **lib/request**;
* **lib/utils/time**;


Third-party:

* **github.com/ethereum/go-ethereum/common**;
* **github.com/go-playground/validator/v10**;
* **github.com/go-viper/mapstructure/v2**;
* **github.com/shopspring/decimal**;
* **google.golang.org/grpc/codes**;
* **google.golang.org/grpc/status**;
* **google.golang.org/protobuf/types/known/timestamppb**;


Standard:

* **errors**;
* **fmt**;
* **strconv**;
* **strings**;
* **time**;


#### Efferent Dependencies (aka "Fan-out")

None (only used by services, not by other lib packages).
