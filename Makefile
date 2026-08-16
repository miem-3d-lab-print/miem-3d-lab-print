SHELL := /bin/sh

.DEFAULT_GOAL := help

.PHONY: help up deploy-server down restart build logs ps migrate test lint lint-backend lint-frontend check

help:
	@printf '%s\n' \
		'make up             Build and start the complete application' \
		'make deploy-server  Configure Docker IPv6 and deploy on Ubuntu' \
		'make down           Stop the application' \
		'make restart        Restart running services' \
		'make build          Build all Docker images' \
		'make logs           Follow service logs' \
		'make ps             Show service status' \
		'make migrate        Apply pending database migrations' \
		'make test           Run backend and frontend unit tests' \
		'make lint           Run backend and frontend linters' \
		'make check          Run all tests, linters, and production builds'

up:
	docker compose up --build -d

deploy-server:
	./scripts/deploy-server.sh

down:
	docker compose down

restart:
	docker compose restart

build:
	docker compose build

logs:
	docker compose logs --since=1m -f

ps:
	docker compose ps

migrate:
	docker compose run --build --rm migrator

test:
	cd backend && go test -race ./...
	cd frontend && npm install --no-audit --no-fund && npm test

lint: lint-backend lint-frontend

lint-backend:
	docker run --rm -v "$(CURDIR):/app:ro" -w /app/backend golangci/golangci-lint:v2.12.2-alpine golangci-lint run --config ../.golangci.yaml

lint-frontend:
	docker run --rm -v "$(CURDIR)/frontend:/app" -v /app/node_modules -w /app node:24-alpine sh -c "npm install --no-audit --no-fund && npm run lint && npm run lint:styles"

check: test lint
	cd backend && go vet ./...
	test -z "$$(gofmt -l backend)"
	cd frontend && npm install --no-audit --no-fund && npm run build
