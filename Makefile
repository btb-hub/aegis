.PHONY: lint type test tidy migrate-up migrate-down migrate-docker \
	setup setup-local install deps seed-dev seed-demo \
	up up-detached down logs ps \
	dev-db dev-db-down dev-api dev-worker dev-web dev-simulator simulate-alert

GO_PKGS := ./pkg/... ./apps/api/... ./apps/worker/...

COMPOSE := docker compose -f deploy/docker-compose.yml
COMPOSE_DEV := docker compose -f deploy/docker-compose.dev.yml

migrate-docker:
	$(COMPOSE) run --rm migrate

ifeq ($(OS),Windows_NT)
LOAD_ENV := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/load-env.ps1
else
LOAD_ENV := bash scripts/load-env.sh
endif

lint:
	cd pkg && go vet ./...
	cd apps/api && go vet ./...
	cd apps/worker && go vet ./...
	cd apps/web && npm run lint
	cd apps/web && npm run build-storybook

type:
	cd pkg && go build ./...
	cd apps/api && go build -o ../../bin/api ./cmd/api
	cd apps/worker && go build -o ../../bin/worker ./cmd/worker
	cd apps/web && npm run typecheck

GO_PKG_PKGS := ./alertparse/... ./alertsim/... ./apperrors/... ./config/... ./i18n/... ./integrations/... ./locale/... ./oncall/... ./rbac/... ./routing/... ./sessiontoken/...
GO_API_PKGS := ./internal/handler/... ./internal/middleware/... ./internal/service/...
GO_WORKER_PKGS := ./internal/processor/...

test:
	cd pkg && go test "-coverprofile=coverage-pkg.out" $(GO_PKG_PKGS)
	cd apps/api && go test "-coverprofile=coverage-api.out" $(GO_API_PKGS)
	cd apps/worker && go test "-coverprofile=coverage-worker.out" $(GO_WORKER_PKGS)
	go run ./scripts/check_coverage.go pkg/coverage-pkg.out
	go run ./scripts/check_coverage.go apps/api/coverage-api.out
	go run ./scripts/check_coverage.go apps/worker/coverage-worker.out
	cd apps/web && npm run check:locales && npm test -- --run --coverage

tidy:
	cd pkg && go mod tidy
	cd apps/api && go mod tidy
	cd apps/worker && go mod tidy

migrate-up:
	migrate -path db/migrations -database "$${DATABASE_URL}" up

migrate-down:
	migrate -path db/migrations -database "$${DATABASE_URL}" down 1

setup:
ifeq ($(OS),Windows_NT)
	@if not exist .env copy deploy\.env.example .env
else
	@test -f .env || cp deploy/.env.example .env
endif
	$(MAKE) deps

setup-local:
ifeq ($(OS),Windows_NT)
	@if not exist .env copy deploy\.env.local.example .env
else
	@test -f .env || cp deploy/.env.local.example .env
endif
	$(MAKE) deps

install: setup

deps:
	cd pkg && go mod download
	cd apps/api && go mod download
	cd apps/worker && go mod download
	cd apps/web && npm install

up: migrate-docker
	$(COMPOSE) up --build

up-detached: migrate-docker
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

dev-db:
	$(COMPOSE_DEV) up -d

dev-db-down:
	$(COMPOSE_DEV) down

dev-api:
	$(LOAD_ENV) go run ./apps/api/cmd/api

dev-worker:
	$(LOAD_ENV) go run ./apps/worker/cmd/worker

dev-web:
	cd apps/web && npm run dev

seed-dev:
	$(LOAD_ENV) go run ./apps/api/cmd/seed-dev

dev-simulator:
	$(LOAD_ENV) go run ./apps/api/cmd/alert-simulator

simulate-alert:
	$(LOAD_ENV) go run ./apps/api/cmd/alert-simulator -once

seed-demo:
	$(LOAD_ENV) go run ./apps/api/cmd/seed-demo
