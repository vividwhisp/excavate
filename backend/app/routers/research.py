import asyncio

from fastapi import APIRouter, Depends, HTTPException, Request
from fastapi.responses import JSONResponse, StreamingResponse

from app.deps import require_auth
from app.sse import Event, format_frame

router = APIRouter(prefix="/api", tags=["research"])


@router.get("/research/stream")
async def stream(request: Request, messageID: str = None, current_user=Depends(require_auth)):  # type: ignore
    if not messageID:
        return JSONResponse({"error": "messageID is required"}, status_code=400)
    # check ownership
    from app.store import messages as mstore, threads as tstore

    async with request.app.state.pool.acquire() as conn:
        msg = await mstore.get(conn, messageID)
        if not msg:
            return JSONResponse({"error": "message_not_found"}, status_code=404)
        th = await tstore.by_id_and_user(conn, str(msg["thread_id"]), current_user["id"])
        if not th:
            return JSONResponse({"error": "message_not_found"}, status_code=404)

    broker = request.app.state.broker

    async def event_generator():
        # preamble
        yield ": connected\n\n"
        from app.store import messages as mstore

        # late join: check status
        async with request.app.state.pool.acquire() as conn:
            msg2 = await mstore.get(conn, messageID)
        if msg2 and msg2["status"] == "complete":
            yield format_frame(Event(type="done", payload={}))
            return
        if msg2 and msg2["status"] == "error":
            err = msg2["error"] or "error"
            yield format_frame(Event(type="error", payload={"message": err}))
            return
        try:
            async for ev in broker.subscribe(messageID):
                yield format_frame(ev)
                if ev.type == "done" or ev.type == "error":
                    break
        except Exception:
            pass

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )
