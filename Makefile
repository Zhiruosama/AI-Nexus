SHELL := /bin/bash

.PHONY: bootstrap infra-up infra-down infra-reset infra-status infra-logs run build vet test lint check

bootstrap:
	@test -f .env || (cp .env.example .env && \
		sed -i "s|^JWT_SECRET=.*|JWT_SECRET=$$(openssl rand -hex 32)|" .env && \
		sed -i "s|^CHAT_ENCRYPTION_KEY=.*|CHAT_ENCRYPTION_KEY=$$(openssl rand -hex 32)|" .env)
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
