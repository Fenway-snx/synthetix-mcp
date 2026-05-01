.PHONY: run run-readonly setup-broker-key test

run:
	set -a; [ ! -f config.env ] || . ./config.env; set +a; go run ./cmd/server

run-readonly:
	set -a; [ ! -f config.env ] || . ./config.env; set +a; go run ./cmd/server --no-broker

setup-broker-key:
	./scripts/setup-broker-key.sh

test:
	go test ./...
