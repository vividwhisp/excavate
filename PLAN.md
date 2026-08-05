# Excavate - Perplexity-style Research Agent

## Goal
Perplexity-style research agent: user asks a question, agent searches the web,
grounds an answer in real sources, and streams a citation-backed markdown answer.
Resume project. Budget: $0.

## Stack
- LLM: Google Gemini Flash (free tier) - llm/Provider interface, swappable
- Search: Google Programmable Search Engine (CSE, free 100 q/day) - search/Provider interface
- Grounding: custom extract package (fetch + HTML->text + rank/truncate)
- Auth: email/password + Redis-backed sessions (bcrypt, httpOnly cookie)
- Storage: Postgres (users/threads/messages/sources), Redis (sessions, rate-limit, query cache, SSE pub/sub, job queue)
- Streaming: SSE (progress + token deltas), Redis pub/sub -> API -> React
- Deploy: Docker Compose self-hosted, $0

## Architecture
React (Vite+TS) --SSE--> Go API (chi) --> research pipeline
  ^                          |  |  |
  |        REST/JSON         |  |  `- Redis (sessions, rate-limit, cache, SSE, jobs)
  `--------------------------|  `---- Postgres (users, threads, messages, sources)
                             `------- worker (Redis queue)

## Research pipeline
query -> enqueue job
      -> search (Google CSE metadata)
      -> extract top-K pages (dedupe, strip nav/ads, meaningful text)
      -> rank + budget-aware truncate to context window
      -> Gemini streaming w/ grounding -> markdown + citations
      -> SSE to UI + persist Postgres + cache identical query (Redis)

## Backend layout
backend/
  cmd/server/main.go
  internal/
    config/       env parsing/validation
    http/         chi router + middleware (Recover, Logging, RequestID, Auth, CORS, RateLimit)
    auth/         register/login/logout, bcrypt, session svc
    research/     orchestrator, worker, states PENDING->SEARCHING->EXTRACTING->REASONING->DONE
    search/       Provider iface + googlecse
    extract/      fetch + HTML->text + ranking/truncation
    llm/          Provider iface + gemini
    store/        postgres repos + goose migrations
    cache/        redis: sessions, limiter, query cache, sse pubsub
    sse/          broker, pub/sub, heartbeat

## Schema
- users(id, email, password_hash, created_at)
- threads(id, user_id FK, title, created_at, updated_at)
- messages(id, thread_id FK, role[user|assistant], content, status, created_at)
- message_sources(id, message_id FK, url, title, snippet, position)

## API
- POST /api/auth/register, /login, /logout
- GET/POST /api/threads
- POST /api/messages
- GET /api/research/stream?thread=..&message=..
- GET /healthz, /readyz

## Frontend (React + Vite + TS)
- api/        typed fetch client, SSE useResearchStream hook
- components/ Chat, SearchInput, Sources, CitationBadges, MarkdownViewer, AuthForms
- pages/      Login, Register, Dashboard
- hooks/      useSSE, useAuth, useThreads

## Infra / hardening
- docker-compose.yml (postgres, redis, backend, frontend) + healthchecks + volumes
- Makefile (make up/migrate/test/lint), .env.example
- slog structured logging, graceful shutdown, goose migrations
- CI-ready: go vet, golangci-lint, unit tests (search/extract/llm mocked)

## Budget constraints
- Google CSE 100 queries/day -> Redis answer cache stretches demos
- Gemini free rate-limited -> retry/backoff built in
- Email verification: interface + console-logged token in dev

## Milestones
1. Scaffold: repo, Docker, config, schema/migrations
2. Auth (email/password, Redis sessions)
3. Pipeline w/ mocked search/extract/llm -> SSE works end-to-end
4. Real Google CSE + extract + Gemini impls
5. React UI (auth, chat, streaming markdown + citations)
6. Hardening: caching, rate limits, retries, logging, tests, README
