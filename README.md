# Excavate

A Perplexity-style research agent. Ask a question, and Excavate searches the
web, grounds the answer in real sources, and streams a citation-backed
markdown response to your browser.

Built with a **Go** backend and a **React** frontend, designed for a **$0
budget** — it runs entirely on free tiers and self-hosted Docker.

> **Status:** milestones 1–5 complete — auth, research pipeline, real providers
> (Tavily + Gemini), and the React UI are all live. M6 (hardening) remains.

---

## Features

- **Research pipeline** — search → extract → ground → stream
- **Citation-backed answers** — every source is captured and rendered as a
  clickable numbered badge, Perplexity-style
- **Live streaming (SSE)** — progress stages and token-by-token deltas stream
  to the browser in real time
- **Session auth** — email/password registration & login with Redis (removed) — simplified to in-process-backed
  sessions (bcrypt password hashing, httpOnly cookies)
- **Threads & history** — conversations persisted in Postgres
- **Modern UI** — cream/sage design system, light & dark themes, streaming
  stage indicator, syntax-highlighted code blocks, mobile-responsive drawer

## Architecture

```
React (Vite+TS) ──SSE stream──▶ FastAPI backend ──▶ Research pipeline
      ▲                              │  │  │
      │         REST / JSON          │  │  └── Redis (removed) — simplified to in-process ─────────────────────┐
      │                              │  │        │
      └──────────────────────────────┤  │        (in-process SSE)           │
                                     │  └── Postgres ──┐        · job queue │
                                     └────── worker (go)  ◀── consume jobs (asyncio.Queue) │
```

### The research pipeline

```
query
  → enqueue job (Redis (removed) — simplified to in-process)
  → search          (Tavily / mock)             → progress: SEARCHING
  → extract         (fetch + HTML→text, rank)  → progress: EXTRACTING  → sources event
  → ground          (query + trimmed pages)    → progress: REASONING
  → stream answer   (Gemini / mock)            → delta events → done
  → persist content + sources to Postgres; cache identical query
```

Everything is behind interfaces (`search.Provider`, `extract.Extractor`,
`llm.Provider`), so real providers can be swapped in without touching the
rest of the code.

## Tech stack

| Layer       | Choice                                   |
|-------------|------------------------------------------|
| Backend     | Python, FastAPI, asyncpg |
| Frontend    | React, Vite, TypeScript, react-markdown  |
| Databases   | PostgreSQL 16                            |
| Auth        | bcrypt + in-memory sessions              |
| Streaming   | Server-Sent Events (in-process broker)   |
| Search      | Tavily API (1,000 free credits/month)    |
| LLM         | Google Gemini Flash (swappable)          |
| Deploy      | Docker Compose (self-hosted)             |

## Architecture (monorepo layout)

```
excavate/
├── backend/
│   ├── app/main.py            # entrypoint: wiring, graceful shutdown
│   └── internal/
│       ├── config/            # env-driven configuration
│       ├── http/              # chi router, middleware, handlers
│       ├── auth/              # register/login/logout, bcrypt, sessions
│       ├── research/          # orchestrator, worker, progress states
│       ├── search/            # Provider interface + implementations
│       ├── extract/           # fetch + HTML→text + ranking
│       ├── llm/               # provider interface + implementations
│       ├── store/             # Postgres repos + migrations
│       └── sse/               # SSE broker, pub/sub, heartbeats
├── frontend/                  # React app
├── scripts/                   # automated test scripts
├── docker-compose.yml
├── Makefile
└── .env.example
```

## Getting started

### Prerequisites

- Python 3.12+
- Node.js 20+ (for the frontend)
- Docker + Docker Desktop running

### 1. Start the data stores

```bash
docker compose up -d postgres
```

### 2. Configure environment

```bash
cp .env.example .env
```

Fill in the provider keys (optional — without them the app runs with mocks):

| Variable          | Purpose                                       | Free tier             |
|-------------------|-----------------------------------------------|-----------------------|
| `TAVILY_API_KEY`  | Tavily search API key                          | 1,000 credits/month   |
| `GEMINI_API_KEY`  | Google AI Studio API key for the LLM           | generous              |
| `GEMINI_MODEL`    | Model id (default `gemini-3.5-flash`)          |                       |
| `RESEARCH_MODE`   | `mock` / `real` / `auto` (default: auto — real when both keys are set) | |

Get a free Tavily key at https://tavily.com and a Gemini key at
https://aistudio.google.com/apikey.

### 3. Run the backend

```bash
cd backend
uvicorn app.main:app --host 0.0.0.0 --port 8080
```

The server applies migrations on startup, then listens on `:8080`.

### 4. Run the frontend

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:5173.

## API reference

| Method | Path                        | Auth | Purpose                                   |
|--------|-----------------------------|------|-------------------------------------------|
| POST   | `/api/auth/register`        |      | Create account, sets session cookie       |
| POST   | `/api/auth/login`           |      | Log in, sets session cookie               |
| POST   | `/api/auth/logout`          | ✓    | Destroy Redis (removed) — simplified to in-process session                     |
| GET    | `/api/me`                   | ✓    | Current user                              |
| POST   | `/api/threads`              | ✓    | Create a conversation thread              |
| GET    | `/api/threads`              | ✓    | List my threads                           |
| GET    | `/api/threads/{id}`         | ✓    | Thread + messages + sources               |
| POST   | `/api/messages`             | ✓    | Ask a question (enqueues a research job)  |
| GET    | `/api/research/stream?messageID=…` | ✓ | Live SSE stream for one message    |
| GET    | `/api/healthz`              |      | Liveness probe                            |
| GET    | `/api/readyz`               |      | Readiness (pings Postgres + Redis (removed) — simplified to in-process)        |

### SSE event schema

```
event: message
data: {"type":"progress","payload":{"stage":"searching"}}
data: {"type":"sources","payload":{"sources":[{"url":"…","title":"…","snippet":"…","position":1}]}}
data: {"type":"delta","payload":{"text":"The answer…"}}
data: {"type":"done","payload":{}}
data: {"type":"error","payload":{"message":"…"}}
```

## Testing

```bash
# Unit tests
make test

# End-to-end auth flow (auto-starts Postgres/Redis (removed) — simplified to in-process + backend, runs 8 assertions)
make test-auth

# End-to-end research pipeline with mock providers (6 assertions, deterministic)
make test-research

# End-to-end research with REAL providers — Tavily + Gemini (uses 1 search credit)
make test-real
```

## Data model

- `users` — email + bcrypt password hash
- `threads` — a conversation belonging to a user
- `messages` — `role` user/assistant, streaming `status`, content
- `message_sources` — citations attached to an assistant message

## Design decisions (why)

- **Interfaces for providers** — swap Google/Gemini/mock without touching the
  pipeline or frontend; also enables easy unit testing.
- **In-process state** — sessions, research job queue, and SSE broker run in-process (single FastAPI worker) for simplicity.
- **Migrations** — schema is versioned with `up`/`down` SQL files, never
  hand-edited.
- **In-process SSE** — events are broadcast in-process; multi-process replicas would need a shared bus.
- **Migrations not auto-created tables** — you get the safety of `up`/`down`.

## Roadmap / milestones

- [x] **M1** Scaffold: server, Docker, config, Postgres schema
- [x] **M2** Auth: register/login/logout, Redis (removed) — simplified to in-process sessions
- [x] **M3** Research pipeline (mock providers) + SSE streaming
- [x] **M4** Real providers: Tavily search + HTML extraction + Gemini
- [x] **M5** React UI (auth, chat, streaming markdown + citations, themes)
- [ ] **M6** Hardening: rate limits, caching, retries, logs, tests

## Production & free-tier constraints

- **$0 budget** — all services run in Docker Compose; provider APIs are on
  free tiers only.
- **Tavily: 1,000 search credits/month** — identical queries are cached in
  Redis (removed) — simplified to in-process to stretch them (each question ≈ 1 credit).
- **Gemini free tier is rate-limited** — retry/backoff is built in.
- **Why not Google CSE?** Programmable Search Engine stopped accepting new
  customers; Tavily was the best $0 alternative.

## License

Private / for demonstration purposes.