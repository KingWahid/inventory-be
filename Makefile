MIGRATIONS_DIR := infra/database/migrations
GO := C:/Program Files/Go/bin/go.exe
GOOSE := "$(GO)" run github.com/pressly/goose/v3/cmd/goose
GOLANGCI_LINT := golangci-lint
PRE_COMMIT := pre-commit
# lint/hooks: put Go's bin dir + GOPATH/bin on PATH so `go` and `golangci-lint` resolve.

.PHONY: help tidy test up down check-dsn lint lint-fix hooks-install hooks-run migration-create migration-status

help:
	@echo "Available targets: tidy, test, up, down, migration-create, migration-status, lint, lint-fix, hooks-install, hooks-run"

tidy:
	"$(GO)" mod tidy

test:
	"$(GO)" test ./...

lint:
	$(GOLANGCI_LINT) run ./...

lint-fix:
	$(GOLANGCI_LINT) run --fix ./...

hooks-install:
	$(PRE_COMMIT) install -c .pre-commit-config.yaml

hooks-run:
	$(PRE_COMMIT) run -c .pre-commit-config.yaml --all-files

up: check-dsn
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" up

down: check-dsn
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" down

migration-status: check-dsn
	$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" status

migration-create:
ifeq ($(strip $(NAME)),)
	$(error NAME is required, example: make migration-create NAME=create_tenants_users)
endif
	$(GOOSE) -dir $(MIGRATIONS_DIR) create "$(NAME)" sql

check-dsn:
ifeq ($(strip $(DB_DSN)),)
	$(error DB_DSN is required)
endif
