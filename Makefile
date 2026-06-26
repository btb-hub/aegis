.PHONY: lint type test migrate-up migrate-down tidy

GO_PKGS := ./pkg/... ./apps/api/... ./apps/worker/...

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

GO_PKG_PKGS := ./alertparse/... ./apperrors/... ./config/... ./i18n/... ./locale/... ./rbac/... ./sessiontoken/...
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
