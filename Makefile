SHELL := /bin/bash

.PHONY: bootstrap infra-up infra-down infra-reset infra-status infra-logs migrate-mail-integration test-mail-integration run build vet test lint check

bootstrap:
	@./scripts/bootstrap-env.sh .env .env.example
	@test -f configs/config.yaml || cp configs/config.example.yaml configs/config.yaml
	@echo "Local configuration is ready. Add MODELSCOPE_API_KEY only when needed."

infra-up: bootstrap
	docker compose up -d

infra-down:
	docker compose down

infra-reset:
	docker compose down -v

infra-status:
	docker compose ps

infra-logs:
	docker compose logs -f

migrate-mail-integration: bootstrap
	@set -a; source .env; set +a; \
		docker compose exec -T mysql mysql -u root -p"$$MYSQL_ROOT_PASSWORD" "$$MYSQL_DATABASE" < migrations/mysql/20260809_email_verification.sql

test-mail-integration: bootstrap migrate-mail-integration
	@set -a; source .env; set +a; \
		AI_NEXUS_TEST_MYSQL_DSN="$$MYSQL_USER:$$MYSQL_PASSWORD@tcp($$MYSQL_HOST:$$MYSQL_PORT)/$$MYSQL_DATABASE?charset=utf8mb4&parseTime=True&loc=Local" \
		go test -tags=integration ./internal/verification -run '^TestStore' -count=1

run: bootstrap
	@set -a; source .env; set +a; go run ./cmd

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

lint:
	golangci-lint run ./...

check: build vet test lint
