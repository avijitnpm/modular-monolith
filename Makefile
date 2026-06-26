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
	cd migrations && tern migrate

migration:
	cd migrations && tern new $(name)

setup:
	cp -n .env.example .env 2>/dev/null || true
	cd frontend && pnpm install --frozen-lockfile
