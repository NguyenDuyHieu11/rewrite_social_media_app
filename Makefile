.PHONY: help tidy build run-gateway run-dispatcher test lint migrate-up migrate-down migrate-new up down logs

GO            ?= go
MIGRATE       ?= migrate
POSTGRES_DSN  ?= postgres://app:app@localhost:5432/social?sslmode=disable

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

tidy: ## Tidy go.mod
	$(GO) mod tidy

build: ## Build both binaries into ./bin
	$(GO) build -o bin/gateway ./cmd/gateway
	$(GO) build -o bin/dispatcher ./cmd/dispatcher

run-gateway: ## Run the gateway binary
	$(GO) run ./cmd/gateway

run-dispatcher: ## Run the dispatcher binary
	$(GO) run ./cmd/dispatcher

test: ## Run all tests with race detector
	$(GO) test ./... -race -count=1

lint: ## Run go vet
	$(GO) vet ./...

up: ## Start Postgres/Redis/MinIO via docker compose
	docker compose up -d

down: ## Stop docker compose stack
	docker compose down

logs: ## Tail docker compose logs
	docker compose logs -f

migrate-up: ## Apply all pending migrations
	$(MIGRATE) -path ./migrations -database "$(POSTGRES_DSN)" up

migrate-down: ## Roll back the most recent migration
	$(MIGRATE) -path ./migrations -database "$(POSTGRES_DSN)" down 1

migrate-new: ## Create a new migration pair: make migrate-new name=add_something
	$(MIGRATE) create -ext sql -dir ./migrations -seq $(name)
