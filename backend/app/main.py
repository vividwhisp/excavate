import asyncio
import logging
import time

import asyncpg
from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.config import get_settings
from app.research.pipeline import Pipeline
from app.research.queue import ResearchQueue
from app.research.worker import run_worker
from app.search.mock import MockSearch
from app.search.tavily import Tavily
from app.extract.mock import MockExtractor
from app.extract.http_extractor import HTTPExtractor
from app.llm.mock import MockLLM
from app.llm.gemini import Gemini
from app.sessions import SessionStore
from app.sse import SSEBroker
from app.store import db as dbmod

logger = logging.getLogger("excavate")


async def create_pool(settings):
    return await asyncpg.create_pool(dsn=settings.database_url, min_size=2, max_size=10)


def build_providers(settings):
    if not settings.real_providers_enabled:
        return MockSearch(), MockExtractor(), MockLLM()
    return Tavily(settings.tavily_api_key), HTTPExtractor(), Gemini(settings.gemini_api_key, settings.gemini_model)


async def lifespan(app: FastAPI):
    settings = get_settings()
    logging.basicConfig(level=logging.DEBUG if settings.debug else logging.INFO)
    pool = await create_pool(settings)
    await dbmod.migrate(settings.database_url)
    app.state.pool = pool
    app.state.settings = settings
    app.state.sessions = SessionStore(settings.session_ttl_hours)
    app.state.broker = SSEBroker()
    app.state.queue = ResearchQueue()
    search, extract, llm = build_providers(settings)
    app.state.pipeline = Pipeline(search, extract, llm, pool, settings, app.state.broker)
    app.state._worker_task = asyncio.create_task(run_worker(app.state.queue, app.state.pipeline))
    logger.info("server listening", extra={"addr": f":{settings.port}", "env": settings.app_env})
    yield
    app.state._worker_task.cancel()
    try:
        await app.state._worker_task
    except Exception:
        pass
    await pool.close()


app = FastAPI(title="Excavate", lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)


@app.middleware("http")
async def recover_and_log(request: Request, call_next):
    start = time.perf_counter()
    try:
        response = await call_next(request)
    except Exception:
        logger.exception("panic", extra={"path": request.url.path})
        response = JSONResponse({"error": "internal_server_error"}, status_code=500)
    duration_ms = int((time.perf_counter() - start) * 1000)
    logger.info(
        "http",
        extra={
            "method": request.method,
            "path": request.url.path,
            "status": response.status_code,
            "duration_ms": duration_ms,
        },
    )
    return response


# CORS - match Go behavior: single origin, allow credentials
settings = get_settings()
app.add_middleware(
    CORSMiddleware,
    allow_origins=[settings.cors_origin] if settings.cors_origin else ["*"],
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
    allow_headers=["Content-Type", "Authorization"],
)

from app.routers import health, auth, threads, messages, research  # noqa

app.include_router(health.router)
app.include_router(auth.router)
app.include_router(threads.router)
app.include_router(messages.router)
app.include_router(research.router)
