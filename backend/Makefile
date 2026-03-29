COMPOSE=docker compose -f compose.yml

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: fmt build run seed-dev test-integration test-integration-minio db-init db-start db-stop db-status db-migrate minio-start minio-stop minio-status infra-start infra-stop docker-build docker-up docker-down docker-seed docker-bootstrap docker-logs docker-ps

fmt:
	go fmt ./...

build:
	go build ./...

run:
	go run ./cmd/api

seed-dev:
	go run ./cmd/seed

test-integration:
	go test -tags=integration ./tests/integration/...

test-integration-minio:
	go test -tags='integration realminio' ./tests/integration/... -run 'RealMinIO'

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

minio-start:
	./scripts/dev-minio-start.sh

minio-stop:
	./scripts/dev-minio-stop.sh

minio-status:
	./scripts/dev-minio-status.sh

infra-start:
	./scripts/dev-db-start.sh
	./scripts/dev-minio-start.sh

infra-stop:
	./scripts/dev-minio-stop.sh
	./scripts/dev-db-stop.sh

docker-build:
	$(COMPOSE) build api seed

docker-up:
	$(COMPOSE) up --build -d api

docker-down:
	$(COMPOSE) down

docker-seed:
	$(COMPOSE) --profile tools run --rm seed

docker-bootstrap:
	$(COMPOSE) up --build -d api
	$(COMPOSE) --profile tools run --rm seed

docker-logs:
	$(COMPOSE) logs -f api postgres minio

docker-ps:
	$(COMPOSE) ps
