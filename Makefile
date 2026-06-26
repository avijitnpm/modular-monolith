.PHONY: dev build test fmt lint frontend frontend-build docker-build docker-up docker-down migrate migration setup

dev:
	go run ./cmd/server

build:
	go build -o bin/app ./cmd/server

frontend:
	cd frontend && pnpm run dev

frontend-build:
	cd frontend && pnpm install --frozen-lockfile && pnpm run build

test:
	go test ./... -short

fmt:
	go fmt ./...

lint:
	golangci-lint run

docker-build:
	docker build -t modular-monolith-app:local .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate:
	@echo "Migrations run automatically on app startup. See internal/database/migrate.go"
	@echo "To add a new migration: make migration name=<description>"

migration:
	@touch migrations/$$(printf '%03d' $$(($$(ls migrations/*.sql 2>/dev/null | wc -l) + 1)))_$(name).sql
	@echo "Created: migrations/$$(printf '%03d' $$(($$(ls migrations/*.sql 2>/dev/null | wc -l))))_$(name).sql"

setup:
	cp -n .env.example .env 2>/dev/null || true
	cd frontend && pnpm install --frozen-lockfile
