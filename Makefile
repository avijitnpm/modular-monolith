dev:
	go run ./cmd/server

build:
	go build -o bin/app ./cmd/server

frontend:
	cd frontend && pnpm run dev

frontend-build:
	cd frontend && pnpm run build

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

docker-up:
	docker compose up -d

docker-down:
	docker compose down