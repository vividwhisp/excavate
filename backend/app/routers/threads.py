from fastapi import APIRouter, Depends, HTTPException, Request
from fastapi.responses import JSONResponse

from app.deps import require_auth
from app.store import threads as tstore

router = APIRouter(prefix="/api", tags=["threads"])


@router.get("/threads")
async def list_threads(current_user=Depends(require_auth), request: Request = None) -> JSONResponse:  # type: ignore
    async with request.app.state.pool.acquire() as conn:
        rows = await tstore.list_by_user(conn, current_user["id"])
    threads = []
    for r in rows:
        threads.append(
            {
                "id": str(r["id"]),
                "userId": str(r["user_id"]),
                "title": r["title"],
                "createdAt": r["created_at"].isoformat().replace("+00:00", "Z") if hasattr(r["created_at"], "isoformat") else str(r["created_at"]),
                "updatedAt": r["updated_at"].isoformat().replace("+00:00", "Z") if hasattr(r["updated_at"], "isoformat") else str(r["updated_at"]),
            }
        )
    return JSONResponse({"threads": threads}, status_code=200)


@router.post("/threads")
async def create_thread(request: Request, current_user=Depends(require_auth)) -> JSONResponse:
    try:
        data = await request.json()
    except Exception:
        return JSONResponse({"error": "invalid_json"}, status_code=400)
    title = str(data.get("title", "")).strip()
    if not title:
        return JSONResponse({"error": "title is required"}, status_code=422)
    async with request.app.state.pool.acquire() as conn:
        r = await tstore.create(conn, current_user["id"], title)
    return JSONResponse(
        {
            "thread": {
                "id": str(r["id"]),
                "userId": str(r["user_id"]),
                "title": r["title"],
                "createdAt": r["created_at"].isoformat().replace("+00:00", "Z") if hasattr(r["created_at"], "isoformat") else str(r["created_at"]),
                "updatedAt": r["updated_at"].isoformat().replace("+00:00", "Z") if hasattr(r["updated_at"], "isoformat") else str(r["updated_at"]),
            }
        },
        status_code=201,
    )


@router.get("/threads/{thread_id}")
async def get_thread(thread_id: str, current_user=Depends(require_auth), request: Request = None) -> JSONResponse:  # type: ignore
    async with request.app.state.pool.acquire() as conn:
        th = await tstore.by_id_and_user(conn, thread_id, current_user["id"])
        if not th:
            return JSONResponse({"error": "thread_not_found"}, status_code=404)
        msgs = await mstore.list_by_thread(conn, thread_id)
    messages = []
    for m in msgs:
        msg = {
            "id": str(m["id"]),
            "threadId": str(m["thread_id"]),
            "role": m["role"],
            "content": m["content"],
            "status": m["status"],
            "error": m["error"] or None,
            "createdAt": m["created_at"].isoformat().replace("+00:00", "Z") if hasattr(m["created_at"], "isoformat") else str(m["created_at"]),
            "sources": [],
        }
        if m["role"] == "assistant":
            async with request.app.state.pool.acquire() as conn2:
                srcs = await mstore.sources_by_message(conn2, str(m["id"]))
            msg["sources"] = [
                {
                    "url": s["url"],
                    "title": s["title"],
                    "snippet": s["snippet"],
                    "position": s["position"],
                }
                for s in srcs
            ]
        messages.append(msg)
    return JSONResponse(
        {
            "thread": {
                "id": str(th["id"]),
                "userId": str(th["user_id"]),
                "title": th["title"],
                "createdAt": th["created_at"].isoformat().replace("+00:00", "Z") if hasattr(th["created_at"], "isoformat") else str(th["created_at"]),
                "updatedAt": th["updated_at"].isoformat().replace("+00:00", "Z") if hasattr(th["updated_at"], "isoformat") else str(th["updated_at"]),
            },
            "messages": messages,
        },
        status_code=200,
    )
