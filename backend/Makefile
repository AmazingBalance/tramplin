APP_NAME=tramplin-api

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: fmt build run test-integration db-init db-start db-stop db-status db-migrate

fmt:
	go fmt ./...

build:
	go build ./...

run:
	go run ./cmd/api

test-integration:
	go test -tags=integration ./tests/integration/...

db-init:
	./scripts/dev-db-init.sh

db-start:
	./scripts/dev-db-start.sh

db-stop:
	./scripts/dev-db-stop.sh

db-status:
	./scripts/dev-db-status.sh

db-migrate:
	./scripts/dev-db-migrate.sh
