MIGRATIONS_DIR := infra/database/migrations
GO := C:/Program Files/Go/bin/go.exe
OAPI_CODEGEN := "$(GO)" run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
GOOSE := "$(GO)" run github.com/pressly/goose/v3/cmd/goose
POWERSHELL := powershell -NoProfile -ExecutionPolicy Bypass -Command
GOLANGCI_LINT := golangci-lint
PRE_COMMIT := pre-commit
-include .env
# lint/hooks: put Go's bin dir + GOPATH/bin on PATH so `go` and `golangci-lint` resolve.

.PHONY: help tidy test test-endpoint generate generate-inventory generate-authentication generate-notification generate-all up down check-dsn lint lint-fix hooks-install hooks-run migration-create migration-status seed seed-mock rollback-mock run-inventory-dev run-authentication-dev run-notification-dev run-common-dev run-worker-dev

help:
	@echo "Available targets: tidy, test, test-endpoint, generate, generate-inventory, generate-authentication, generate-notification, generate-all, up, down, migration-create, migration-status, seed, seed-mock, rollback-mock, run-inventory-dev, run-authentication-dev, run-notification-dev, run-common-dev, run-worker-dev, lint, lint-fix, hooks-install, hooks-run"

# Regenerate all service stubs.
generate: generate-inventory generate-authentication generate-notification

# Regenerate services/inventory/stub/openapi.gen.go
generate-inventory:
	$(OAPI_CODEGEN) -config services/inventory/oapi-codegen.yaml services/inventory/openapi/openapi.yaml

# Regenerate services/authentication/stub/openapi.gen.go
generate-authentication:
	$(OAPI_CODEGEN) -config services/authentication/oapi-codegen.yaml services/authentication/openapi/openapi.yaml

# Regenerate services/notification/stub/openapi.gen.go
generate-notification:
	$(OAPI_CODEGEN) -config services/notification/oapi-codegen.yaml services/notification/openapi/openapi.yaml

# Backward-compatible alias.
generate-all: generate

tidy:
	"$(GO)" mod tidy

test:
	"$(GO)" test ./...

test-endpoint:
ifeq ($(strip $(service)),)
	$(error service is required, example: make test-endpoint service=authentication)
endif
ifeq ($(strip $(service)),authentication)
	"$(GO)" test ./services/authentication/api -run TestAuthenticationEndpointContract -v
else
	$(error unsupported service '$(service)' for test-endpoint)
endif

lint:
	$(GOLANGCI_LINT) run ./...

lint-fix:
	$(GOLANGCI_LINT) run --fix ./...

hooks-install:
	$(PRE_COMMIT) install -c .pre-commit-config.yaml

hooks-run:
	$(PRE_COMMIT) run -c .pre-commit-config.yaml --all-files

up: check-dsn
	$(POWERSHELL) "$$env:GOOSE_DRIVER='postgres'; $$env:GOOSE_DBSTRING='$(DB_DSN)'; & '$(GO)' run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) up"

down: check-dsn
	$(POWERSHELL) "$$env:GOOSE_DRIVER='postgres'; $$env:GOOSE_DBSTRING='$(DB_DSN)'; & '$(GO)' run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) down"

migration-status: check-dsn
	$(POWERSHELL) "$$env:GOOSE_DRIVER='postgres'; $$env:GOOSE_DBSTRING='$(DB_DSN)'; & '$(GO)' run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) status"

seed: check-dsn
	$(POWERSHELL) "$$env:DB_DSN='$(DB_DSN)'; & '$(GO)' run ./infra/database/cmd/seed"

seed-mock: check-dsn
	$(POWERSHELL) "$$env:DB_DSN='$(DB_DSN)'; & '$(GO)' run ./infra/database/cmd/seed --mode seed"

rollback-mock: check-dsn
	$(POWERSHELL) "$$env:DB_DSN='$(DB_DSN)'; & '$(GO)' run ./infra/database/cmd/seed --mode rollback"

run-inventory-dev:
	"$(GO)" run github.com/air-verse/air@latest -c services/inventory/air.toml

run-authentication-dev:
	"$(GO)" run github.com/air-verse/air@latest -c services/authentication/air.toml

run-notification-dev:
	"$(GO)" run github.com/air-verse/air@latest -c services/notification/air.toml

run-common-dev:
	"$(GO)" run github.com/air-verse/air@latest -c services/common/air.toml

run-worker-dev:
	"$(GO)" run github.com/air-verse/air@latest -c workers/air.toml

migration-create:
ifeq ($(strip $(NAME)),)
	$(error NAME is required, example: make migration-create NAME=create_tenants_users)
endif
	$(GOOSE) -dir $(MIGRATIONS_DIR) create "$(NAME)" sql

check-dsn:
ifeq ($(strip $(DB_DSN)),)
	$(error DB_DSN is required)
endif
