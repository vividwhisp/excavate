.PHONY: up down ps logs migrate-backend build-backend test vet fmt lint clean test-auth test-research test-real

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

# End-to-end auth flow against a running stack (auto-starts deps + backend)
test-auth:
	powershell -ExecutionPolicy Bypass -File scripts/test-auth.ps1

# End-to-end research pipeline flow (auto-starts deps + backend)
test-research:
	powershell -ExecutionPolicy Bypass -File scripts/test-research.ps1

# End-to-end research with REAL providers (Tavily + Gemini; reads backend/.env)
test-real:
	powershell -ExecutionPolicy Bypass -File scripts/test-real.ps1

vet:
	cd backend && go vet ./...

fmt:
	cd backend && gofmt -w .

clean:
	docker compose down -v