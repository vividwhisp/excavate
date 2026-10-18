# Excavate

A Perplexity-style research agent. Ask a question, and Excavate searches the
web, grounds the answer in real sources, and streams a citation-backed
markdown response to your browser.

Built with a **Python/FastAPI** backend and a **React** frontend, designed for
a **$0 budget** — it runs entirely on free tiers and self-hosted Docker.

> **Status:** milestones 1–5 complete — auth, research pipeline, real providers
> (Tavily + Gemini), and the React UI are all live. The backend was rewritten
> from Go to Python/FastAPI with Redis removed (single-process, in-memory
> sessions/queue/SSE). M6 (hardening) remains.

---

## Features

- **Research pipeline** — search → extract → ground → stream
- **Citation-backed answers** — every source is captured and rendered as a
  clickable numbered badge, Perplexity-style
- **Live streaming (SSE)** — progress stages and token-by-token deltas stream
  to the browser in real time
- **Session auth** — email/password registration & login with in-memory
  sessions (bcrypt password hashing, httpOnly cookies)
- **Threads & history** — conversations persisted in Postgres
- **Modern UI** — cream/sage design system, light & dark themes, streaming
  stage indicator, syntax-highlighted code blocks, mobile-responsive drawer

## Architecture

```
React (Vite+TS) ──SSE stream──▶ FastAPI backend ──▶ Research pipeline
      ▲                              │  │
      │         REST / JSON          │  ├── in-memory: sessions · job queue · SSE broker
      │                              │  │
      └──────────────────────────────┴──┴── Postgres (users, threads, messages, sources)
                                       └── asyncio worker consumes research jobs
```

Everything runs in a **single process** — there is no Redis. Sessions, the
research job queue, and the SSE event broker are all in-memory, so state
resets when the server restarts. Durable data lives in Postgres.

### The research pipeline

```
query
  → enqueue job (asyncio.Queue)
  → search          (Tavily / mock)             → progress: SEARCHING
  → extract         (fetch + HTML→text)         → progress: EXTRACTING  → sources event
  → ground          (query + trimmed pages)     → progress: REASONING
  → stream answer   (Gemini / mock)             → delta events → done
  → persist content + sources to Postgres
```

Providers sit behind simple classes (`MockSearch`/`Tavily`,
`MockExtractor`/`HTTPExtractor`, `MockLLM`/`Gemini`), selected at startup by
`RESEARCH_MODE`, so real providers can be swapped without touching the rest
of the code.

## Tech stack

| Layer       | Choice                                   |
|-------------|------------------------------------------|
| Backend     | Python, FastAPI, uvicorn, asyncpg        |
| Frontend    | React, Vite, TypeScript, react-markdown  |
| Database    | PostgreSQL 16                            |
| Auth        | bcrypt + in-memory sessions              |
| Streaming   | Server-Sent Events (in-process broker)   |
| Search      | Tavily API (1,000 free credits/month)    |
| LLM         | Google Gemini Flash (swappable)          |
| Deploy      | Docker Compose (self-hosted)             |

## Repository layout

```
excavate/
├── backend/
│   ├── pyproject.toml         # Python deps (fastapi, asyncpg, httpx, ...)
│   ├── Dockerfile
│   └── app/
│       ├── main.py            # app wiring, lifespan, CORS, middleware
│       ├── config.py          # env-driven configuration (pydantic-settings)
│       ├── auth.py            # bcrypt hash/verify
│       ├── sessions.py        # in-memory session store (sliding TTL)
│       ├── deps.py            # current-user / require-auth dependencies
│       ├── sse.py             # SSE event broker + frame formatting
│       ├── routers/           # auth, threads, messages, research, health
│       ├── research/          # pipeline, job queue, asyncio worker
│       ├── search/            # Tavily + mock
│       ├── extract/           # HTTP fetch + BeautifulSoup + mock
│       ├── llm/               # Gemini streaming + mock
│       └── store/             # asyncpg repos + SQL migrations
├── frontend/                  # React app
├── scripts/                   # automated e2e test scripts
├── docker-compose.yml
├── Makefile
└── .env.example
```

## Getting started

### Prerequisites

- Python 3.12+
- [uv](https://docs.astral.sh/uv/) (or plain pip)
- Node.js 20+ (for the frontend)
- Docker + Docker Desktop running

### 1. Start Postgres

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
uv sync                      # install deps into .venv
uvicorn app.main:app --reload --host 0.0.0.0 --port 8080
```

The server applies migrations on startup, then listens on `:8080`.

### 4. Run the frontend

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:5173 (Vite proxies `/api` to `localhost:8080`).

## API reference

| Method | Path                        | Auth | Purpose                                   |
|--------|-----------------------------|------|-------------------------------------------|
| POST   | `/api/auth/register`        |      | Create account, sets session cookie       |
| POST   | `/api/auth/login`           |      | Log in, sets session cookie               |
| POST   | `/api/auth/logout`          | ✓    | Destroy the in-memory session             |
| GET    | `/api/me`                   | ✓    | Current user                              |
| POST   | `/api/threads`              | ✓    | Create a conversation thread              |
| GET    | `/api/threads`              | ✓    | List my threads                           |
| GET    | `/api/threads/{id}`         | ✓    | Thread + messages + sources               |
| POST   | `/api/messages`             | ✓    | Ask a question (enqueues a research job)  |
| GET    | `/api/research/stream?messageID=…` | ✓ | Live SSE stream for one message    |
| GET    | `/api/healthz`              |      | Liveness probe                            |
| GET    | `/api/readyz`               |      | Readiness (pings Postgres)                |

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
# End-to-end auth flow (auto-starts Postgres + backend, runs 8 assertions)
make test-auth

# End-to-end research pipeline with mock providers (6 assertions, deterministic)
make test-research

# End-to-end research with REAL providers — Tavily + Gemini (uses 1 search credit)
make test-real
```

> Note: these scripts target `http://localhost:8080`; start the backend first
> (`make run-backend`).

## Data model

- `users` — email + bcrypt password hash
- `threads` — a conversation belonging to a user
- `messages` — `role` user/assistant, streaming `status`, content
- `message_sources` — citations attached to an assistant message

## Design decisions (why)

- **Provider classes for search/extract/LLM** — swap Tavily/Gemini/mock
  without touching the pipeline or frontend; also enables offline testing.
- **No Redis, single process** — sessions, the research job queue, and the
  SSE broker are in-memory for simplicity. Trade-off: all state resets on
  server restart, and scaling beyond one process would need a shared bus.
- **Migrations** — schema is versioned with `up`/`down` SQL files applied at
  startup, never hand-edited.
- **In-memory SSE broker with replay** — events are buffered per message, so
  a client that connects after the stream starts still receives everything.

## Roadmap / milestones

- [x] **M1** Scaffold: server, Docker, config, Postgres schema
- [x] **M2** Auth: register/login/logout, sessions
- [x] **M3** Research pipeline (mock providers) + SSE streaming
- [x] **M4** Real providers: Tavily search + HTML extraction + Gemini
- [x] **M5** React UI (auth, chat, streaming markdown + citations, themes)
- [x] **Backend rewrite** Go → Python/FastAPI, Redis removed
- [ ] **M6** Hardening: rate limits, caching, retries, logs, tests

## Production & free-tier constraints

- **$0 budget** — all services run in Docker Compose; provider APIs are on
  free tiers only.
- **Tavily: 1,000 search credits/month** — each question uses ~1 credit
  (a query cache is on the M6 wishlist to stretch them).
- **Gemini free tier is rate-limited** — retries/backoff are on the M6
  wishlist.
- **Why not Google CSE?** Programmable Search Engine stopped accepting new
  customers; Tavily was the best $0 alternative.

## License

Private / for demonstration purposes.
