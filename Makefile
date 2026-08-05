.PHONY: up down ps logs migrate-backend build-backend test vet fmt lint clean

# Spin up Postgres + Redis (data stores for local dev / backend runs via `go run`)
up:
	docker compose up -d postgres redis

down:
	docker compose down

ps:
	docker compose ps

logs:
	docker compose logs -f --tail=200

# Run the whole stack (backend + frontend) as containers
stack:
	docker compose up --build

run-backend:
	cd backend && go run ./cmd/server

migrate:
	cd backend && go run ./cmd/server

build:
	cd backend && go build ./...

test:
	cd backend && go test ./...

vet:
	cd backend && go vet ./...

fmt:
	cd backend && gofmt -w .

clean:
	docker compose down -v