import asyncio

from fastapi import APIRouter, Depends, HTTPException, Request
from fastapi.responses import JSONResponse, StreamingResponse

from app.deps import require_auth
from app.research.queue import Job
from app.sse import SSEBroker, Event, format_frame
from app.store import messages as mstore, threads as tstore

router = APIRouter(prefix="/api", tags=["messages"])


@router.post("/messages")
async def post_message(request: Request, current_user=Depends(require_auth)) -> JSONResponse:
    try:
        data = await request.json()
    except Exception:
        return JSONResponse({"error": "invalid_json"}, status_code=400)
    thread_id = str(data.get("threadId") or "")
    content = str(data.get("content") or "")
    if not content:
        return JSONResponse({"error": "content is required"}, status_code=422)
    async with request.app.state.pool.acquire() as conn:
        th = await tstore.by_id_and_user(conn, thread_id, current_user["id"])
        if not th:
            return JSONResponse({"error": "thread_not_found"}, status_code=404)
        user_msg = await mstore.create(conn, thread_id, "user")
    # update content after insert (matches Go: insert role then UPDATE content)
    async with request.app.state.pool.acquire() as conn:
        await conn.execute("UPDATE messages SET content = $2 WHERE id = $1", str(user_msg["id"]), content)
        asst = await mstore.create(conn, thread_id, "assistant")
    job = Job(thread_id=thread_id, message_id=str(asst["id"]), query=content)
    try:
        await request.app.state.queue.enqueue(job)
    except Exception as e:
        async with request.app.state.pool.acquire() as conn:
            await mstore.fail(conn, str(asst["id"]), "failed to enqueue")
        return JSONResponse({"error": "internal_error"}, status_code=500)
    return JSONResponse(
        {
            "message": {
                "id": str(asst["id"]),
                "threadId": thread_id,
                "role": "assistant",
                "content": "",
                "status": "pending",
                "createdAt": asst["created_at"].isoformat().replace("+00:00", "Z") if hasattr(asst["created_at"], "isoformat") else str(asst["created_at"]),
            }
        },
        status_code=201,
    )
