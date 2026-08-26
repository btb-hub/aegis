.PHONY: lint type test tidy migrate-up migrate-down migrate-docker \
	setup setup-local install deps seed-dev \
	up up-detached up-dev up-dev-detached down logs ps \
	dev-db dev-db-down dev-api dev-worker dev-web \
	dev-simulator dev-simulator-bootstrap simulate-alert test-devtools \
	image image-smoke

GO_PKGS := ./pkg/... ./apps/api/... ./apps/worker/...

COMPOSE := docker compose -f deploy/docker-compose.yml
COMPOSE_DEV := docker compose -f deploy/docker-compose.dev.yml
COMPOSE_WITH_SIM := $(COMPOSE) --profile dev
IMAGE_NAME ?= aegis:local

ALERT_SIM := ./devtools/alert-simulator/cmd/alert-simulator

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

GO_PKG_PKGS := ./alertparse/... ./apperrors/... ./config/... ./db/... ./i18n/... ./integrations/... ./locale/... ./oncall/... ./rbac/... ./routing/... ./sessiontoken/...
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
	cd devtools/alert-simulator && go mod tidy

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
	cd devtools/alert-simulator && go mod download
	cd apps/web && npm install

up: migrate-docker
	$(COMPOSE) up --build

up-detached: migrate-docker
	$(COMPOSE) up --build -d

up-dev: migrate-docker
	$(COMPOSE_WITH_SIM) up --build

up-dev-detached: migrate-docker
	$(COMPOSE_WITH_SIM) up --build -d

down:
	$(COMPOSE_WITH_SIM) down

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
	$(LOAD_ENV) go run $(ALERT_SIM)

simulate-alert:
	$(LOAD_ENV) go run $(ALERT_SIM) -once

dev-simulator-bootstrap:
	$(LOAD_ENV) go run $(ALERT_SIM) -bootstrap-only

test-devtools:
	cd devtools/alert-simulator && go test ./...

image:
	docker build -f deploy/Dockerfile -t $(IMAGE_NAME) .

# Requires a running Postgres and env file. Example:
#   make dev-db
#   DATABASE_URL=postgres://aegis:aegis@host.docker.internal:5432/aegis?sslmode=disable \
#   SESSION_SECRET=x WEBHOOK_SECRET=x PUBLIC_URL=http://localhost:3000 \
#   make image-smoke
image-smoke: image
	@test -n "$${DATABASE_URL}" || (echo "DATABASE_URL required" >&2; exit 1)
	@test -n "$${SESSION_SECRET}" || (echo "SESSION_SECRET required" >&2; exit 1)
	@test -n "$${WEBHOOK_SECRET}" || (echo "WEBHOOK_SECRET required" >&2; exit 1)
	@test -n "$${PUBLIC_URL}" || (echo "PUBLIC_URL required" >&2; exit 1)
	@set -euo pipefail; \
	trap 'docker rm -f aegis-smoke >/dev/null 2>&1 || true' EXIT; \
	docker run --rm -d --name aegis-smoke -p 3000:3000 \
	  -e DATABASE_URL \
	  -e SESSION_SECRET \
	  -e WEBHOOK_SECRET \
	  -e PUBLIC_URL \
	  -e HTTP_ADDR=127.0.0.1:8080 \
	  $(IMAGE_NAME); \
	echo "Waiting for /healthz..."; \
	for i in 1 2 3 4 5 6 7 8 9 10; do \
	  if curl -fsS http://127.0.0.1:3000/healthz >/dev/null; then break; fi; \
	  sleep 2; \
	done; \
	curl -fsS http://127.0.0.1:3000/healthz; \
	curl -fsS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3000/ | grep -E '200|304'
