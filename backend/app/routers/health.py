import asyncio

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

router = APIRouter(tags=["health"])


@router.get("/api/healthz")
async def healthz() -> JSONResponse:
    return JSONResponse({"status": "ok"})


@router.get("/api/readyz")
async def readyz(request: Request) -> JSONResponse:
    pool = request.app.state.pool
    try:
        async with asyncio.timeout(2):
            await pool.fetchval("SELECT 1")
    except TimeoutError:
        return JSONResponse({"error": "database_unavailable"}, status_code=503)
    except Exception:
        return JSONResponse({"error": "database_unavailable"}, status_code=503)
    return JSONResponse({"status": "ok"})
