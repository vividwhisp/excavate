.PHONY: up down ps logs stack run-backend migrate build test test-auth test-research test-real vet fmt clean

up:
	docker compose up -d postgres

down:
	docker compose down

ps:
	docker compose ps

logs:
	docker compose logs -f --tail=200

stack:
	docker compose up --build

run-backend:
	cd backend && PYTHONPATH=. uvicorn app.main:app --reload --host 0.0.0.0 --port 8080

migrate:
	cd backend && PYTHONPATH=. python -c "import asyncio; from app.store import db; asyncio.run(db.migrate('postgres://excavate:excavate@localhost:5432/excavate?sslmode=disable'))"

build:
	cd backend && python -m py_compile app/main.py

test:
	cd backend && PYTHONPATH=. python -c "pass"

test-auth:
	powershell -ExecutionPolicy Bypass -File scripts/test-auth.ps1

test-research:
	powershell -ExecutionPolicy Bypass -File scripts/test-research.ps1

test-real:
	powershell -ExecutionPolicy Bypass -File scripts/test-real.ps1

vet:
	cd backend && python -m py_compile $(find app -name "*.py")

fmt:
	echo "fmt skipped"

clean:
	docker compose down -v
